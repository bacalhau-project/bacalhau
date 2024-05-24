package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nuid"
	"github.com/rs/zerolog/log"
)

// RequestChanLen Default request channel length for buffering asynchronous results.
const RequestChanLen = 16

// inboxPrefix is the prefix for all streaming inbox subjects.
// similar to https://github.com/nats-io/nats.go/blob/main/nats.go#L4015
const (
	inboxPrefix    = "_SINBOX."
	inboxPrefixLen = len(inboxPrefix)
	replySuffixLen = 8 // Gives us 62^8
	rdigits        = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	base           = 62
	nuidSize       = 22
)

// streamingBucket is a structure to hold the response channel and context
type streamingBucket struct {
	// ctx is the context for the channel consumer that requested and waiting for messages
	ctx            context.Context
	token          string
	createdAt      time.Time
	producerConnID string
	ch             chan *concurrency.AsyncResult[[]byte]
	cancel         context.CancelFunc
	closeOnce      sync.Once
}

// newStreamingBucket creates a new streamingBucket.
func newStreamingBucket(ctx context.Context, token string, producerConnID string) *streamingBucket {
	ctx, cancel := context.WithCancel(ctx)
	return &streamingBucket{
		ctx:            ctx,
		cancel:         cancel,
		createdAt:      time.Now(),
		producerConnID: producerConnID,
		token:          token,
		ch:             make(chan *concurrency.AsyncResult[[]byte], RequestChanLen),
	}
}

// close will close the channel and cancel the context.
func (sb *streamingBucket) close() {
	sb.closeOnce.Do(func() {
		sb.cancel()
		close(sb.ch)
	})
}

type ConsumerClientParams struct {
	Conn   *nats.Conn
	Config StreamConsumerClientConfig
}

// ConsumerClient represents a NATS streaming client.
type ConsumerClient struct {
	Conn *nats.Conn
	mu   sync.RWMutex // Protects access to the response map.

	// response handler
	respSub       string                      // The wildcard subject
	respSubPrefix string                      // the wildcard prefix including trailing .
	respSubLen    int                         // the length of the wildcard prefix excluding trailing .
	respScanf     string                      // The scanf template to extract mux token
	respMux       *nats.Subscription          // A single response subscription
	respMap       map[string]*streamingBucket // Request map for the response msg channels
	respRand      *rand.Rand                  // Used for generating suffix

	heartBeatRequestSub string // A heart beat subject where the producer sends heart beat request to convey existing stream ids

	config StreamConsumerClientConfig
}

// NewConsumerClient creates a new NATS client.
func NewConsumerClient(params ConsumerClientParams) (*ConsumerClient, error) {
	nc := &ConsumerClient{
		Conn:     params.Conn,
		respMap:  make(map[string]*streamingBucket),
		respRand: rand.New(rand.NewSource(time.Now().UnixNano())), //nolint:gosec // using same inbox naming as nats
		config:   params.Config,
	}

	// Setup response subscription.
	newInbox := nc.newInbox()
	nc.respSubPrefix = fmt.Sprintf("%s.", newInbox)
	nc.respSubLen = len(nc.respSubPrefix)
	nc.respSub = fmt.Sprintf("%s*", nc.respSubPrefix)
	nc.heartBeatRequestSub = fmt.Sprintf("%s.%s", "OrchestratorHeartBeatRequestSub", newInbox)

	// Create the response subscription we will use for all streaming responses.
	// This will be on an _SINBOX with an additional terminal token. The subscription
	// will be on a wildcard.
	sub, err := nc.Conn.Subscribe(nc.respSub, nc.respHandler)
	if err != nil {
		return nil, err
	}
	nc.respScanf = strings.Replace(nc.respSub, "*", "%s", -1)
	nc.respMux = sub

	_, err = nc.Conn.Subscribe(nc.heartBeatRequestSub, nc.heartBeatRespHandler)
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("Streaming client created with inbox %s", sub.Subject)
	return nc, nil
}

// newInbox will return a new inbox string for this client.
func (nc *ConsumerClient) newInbox() string {
	var b [inboxPrefixLen + nuidSize]byte
	pres := b[:inboxPrefixLen]
	copy(pres, inboxPrefix)
	ns := b[inboxPrefixLen:]
	copy(ns, nuid.Next())
	return string(b[:])
}

// respHandler is the global response handler. It will look up
// the appropriate channel based on the last token and place
// the message on the channel if possible.
func (nc *ConsumerClient) respHandler(m *nats.Msg) {
	// Just return if closed.
	if nc.Conn.IsClosed() {
		return
	}

	nc.mu.Lock()
	rt := nc.respToken(m.Subject)
	bucket, ok := nc.respMap[rt]
	nc.mu.Unlock()
	if !ok {
		log.Debug().Str("subject", m.Subject).Msg("No response handler for subject")
		return
	}

	closeErr := new(CloseError)
	var asyncResult *concurrency.AsyncResult[[]byte]

	sMsg := new(StreamingMsg)
	err := json.Unmarshal(m.Data, sMsg)
	if err != nil {
		closeErr = &CloseError{Code: CloseUnsupportedData, Text: err.Error()}
		asyncResult = concurrency.NewAsyncError[[]byte](closeErr)
	} else {
		switch sMsg.Type {
		case streamingMsgTypeClose:
			closeErr = sMsg.CloseError
			asyncResult = concurrency.NewAsyncError[[]byte](closeErr)
		case streamingMsgTypeData:
			asyncResult = concurrency.NewAsyncValue(sMsg.Data)
		default:
			log.Warn().Msgf("Unknown streaming message type: %d", sMsg.Type)
			return
		}
	}

	// if normal closure, then we close the channel without adding any error message
	if closeErr != nil && closeErr.Code == CloseNormalClosure {
		nc.cleanupBucket(rt)
		return
	}

	// Explicitly check if the context is done before attempting to send a message.
	if err = bucket.ctx.Err(); err != nil {
		// The context is already done. Handle cleanup and exit.
		nc.cleanupBucket(rt)
		return
	}

	// while the channel is buffered, this will block if processing messages is slow.
	// TODO: Consider a non-blocking send here.
	select {
	case bucket.ch <- asyncResult:
		// if there was an error, we close the channel after notifying the channel consumer
		if asyncResult.Err != nil {
			nc.cleanupBucket(rt)
		}
	case <-bucket.ctx.Done():
		// remove the bucket from the map and close the channel
		nc.cleanupBucket(rt)
	}
}

func (nc *ConsumerClient) cleanupBucket(token string) {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	if bucket, ok := nc.respMap[token]; ok {
		delete(nc.respMap, token)
		bucket.close()
	}
}

// newRespInbox creates a new literal response subject
// that will trigger the mux subscription handler.
// Lock should be held.
func (nc *ConsumerClient) newRespInbox() string {
	var sb strings.Builder
	sb.WriteString(nc.respSubPrefix)

	rn := nc.respRand.Int63()
	for i := 0; i < replySuffixLen; i++ {
		sb.WriteByte(rdigits[rn%base])
		rn /= base
	}

	return sb.String()
}

// respToken will return the last token of a literal response inbox
// which we use for the message channel lookup.
// Lock should be held.
func (nc *ConsumerClient) respToken(respInbox string) string {
	var token string
	n, err := fmt.Sscanf(respInbox, nc.respScanf, &token)
	if err != nil || n != 1 {
		return ""
	}
	return token
}

// OpenStream takes a context, a subject and payload
// in bytes and expects a channel with multiple responses.
func (nc *ConsumerClient) OpenStream(
	ctx context.Context, subj string,
	producerConnID string,
	data []byte) (<-chan *concurrency.AsyncResult[[]byte], error) {
	if ctx == nil {
		return nil, nats.ErrInvalidContext
	}
	if nc == nil {
		return nil, nats.ErrInvalidConnection
	}
	// Check whether the context is done already before making
	// the request.
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	bucket, err := nc.createNewRequestAndSend(ctx, subj, producerConnID, data)
	if err != nil {
		return nil, err
	}
	return bucket.ch, nil
}

func (nc *ConsumerClient) heartBeatRespHandler(msg *nats.Msg) {
	request := new(HeartBeatRequest)
	err := json.Unmarshal(msg.Data, request)
	if err != nil {
		log.Err(err)
		return
	}

	var nonRecentStreamIds []string
	for k, v := range nc.respMap {
		if v.producerConnID == request.ProducerConnID &&
			time.Since(v.createdAt) > nc.config.StreamCancellationBufferDuration {
			nonRecentStreamIds = append(nonRecentStreamIds, k)
		}
	}

	data, err := json.Marshal(ConsumerHeartBeatResponse{NonActiveStreamIds: Difference(nonRecentStreamIds, request.ActiveStreamIds)})
	if err != nil {
		log.Err(err)
		return
	}

	err = nc.Conn.Publish(msg.Reply, data)
	if err != nil {
		log.Err(err)
		return
	}
}

// createNewRequestAndSend sets up and sends a new request, returning the response bucket.
func (nc *ConsumerClient) createNewRequestAndSend(
	ctx context.Context,
	subj string,
	producerConnID string,
	data []byte) (*streamingBucket, error) {
	nc.mu.Lock()

	// Create new literal Inbox and map to a bucket.
	respInbox := nc.newRespInbox()
	token := respInbox[nc.respSubLen:]
	bucket := newStreamingBucket(ctx, token, producerConnID)

	nc.respMap[token] = bucket
	nc.mu.Unlock()

	streamRequest := Request{
		ConnectionDetails: ConnectionDetails{
			ConnID:              nc.respSubPrefix,
			StreamID:            token,
			HeartBeatRequestSub: nc.heartBeatRequestSub,
		},
		Data: data,
	}

	request, err := json.Marshal(streamRequest)
	if err != nil {
		return nil, err
	}

	msg := &nats.Msg{
		Subject: subj,
		Reply:   respInbox,
		Data:    request,
	}

	if err := nc.Conn.PublishMsg(msg); err != nil {
		return nil, err
	}

	return bucket, nil
}

// NewWriter creates a new streaming writer.
func (nc *ConsumerClient) NewWriter(subject string) *Writer {
	return &Writer{
		conn:    nc.Conn,
		subject: subject,
	}
}

func Difference(a, b []string) []string {
	i, j := 0, 0
	var diff []string

	sort.Strings(a)
	sort.Strings(b)

	for i < len(a) && j < len(b) {
		if a[i] < b[j] {
			diff = append(diff, a[i])
			i++
		} else if b[j] < a[i] {
			j++
		} else {
			i++
			j++
		}
	}
	for ; i < len(a); i++ {
		diff = append(diff, a[i])
	}

	return diff
}
