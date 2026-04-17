package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nuid"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"

	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
)

// RequestChanLen Default request channel length for buffering asynchronous results.
const RequestChanLen = 16

// inboxPrefix is the prefix for all streaming inbox subjects.
// similar to https://github.com/nats-io/nats.go/blob/main/nats.go#L4015
const (
	inboxPrefix     = "_SINBOX."
	heartBeatPrefix = "_HEARTBEAT"
	inboxPrefixLen  = len(inboxPrefix)
	replySuffixLen  = 8 // Gives us 62^8
	rDigits         = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	base            = 62
	nuidSize        = 22
)

// streamingBucket is a structure to hold the response channel and context
type streamingBucket struct {
	// ctx is the context for the channel consumer that requested and waiting for messages
	ctx        context.Context
	token      string
	createdAt  time.Time
	requestSub string
	ch         chan *concurrency.AsyncResult[[]byte]
	cancel     context.CancelFunc
	closeOnce  sync.Once
}

// newStreamingBucket creates a new streamingBucket.
func newStreamingBucket(ctx context.Context, token string, requestSub string) *streamingBucket {
	ctx, cancel := context.WithCancel(ctx)
	return &streamingBucket{
		ctx:        ctx,
		cancel:     cancel,
		createdAt:  time.Now(),
		requestSub: requestSub,
		token:      token,
		ch:         make(chan *concurrency.AsyncResult[[]byte], RequestChanLen),
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
	respSub       string                        // The wildcard subject
	respSubPrefix string                        // the wildcard prefix including trailing .
	respSubLen    int                           // the length of the wildcard prefix excluding trailing .
	respScanf     string                        // The scanf template to extract mux token
	respMux       *nats.Subscription            // A single response subscription
	respMap       map[string]*streamingBucket   // Request map for the response msg channels
	reqSubMap     map[string][]*streamingBucket // Request Subject map which hold a request subject where request was sent for streams
	respRand      *rand.Rand                    // Used for generating suffix

	heartBeatRequestSub string // A heart beat subject where the producer sends heart beat request to convey existing stream ids

	config StreamConsumerClientConfig
}

// NewConsumerClient creates a new NATS client.
func NewConsumerClient(params ConsumerClientParams) (*ConsumerClient, error) {
	nc := &ConsumerClient{
		Conn:      params.Conn,
		respMap:   make(map[string]*streamingBucket),
		reqSubMap: make(map[string][]*streamingBucket),
		respRand:  rand.New(rand.NewSource(time.Now().UnixNano())), //nolint:gosec // using same inbox naming as nats
		config:    params.Config,
	}

	// Setup response subscription.
	newInbox := nc.newInbox()
	nc.respSubPrefix = fmt.Sprintf("%s.", newInbox)
	nc.respSubLen = len(nc.respSubPrefix)
	nc.respSub = fmt.Sprintf("%s*", nc.respSubPrefix)
	nc.heartBeatRequestSub = fmt.Sprintf("%s.%s", heartBeatPrefix, newInbox)

	// Create the response subscription we will use for all streaming responses.
	// This will be on an _SINBOX with an additional terminal token. The subscription
	// will be on a wildcard.
	sub, err := nc.Conn.Subscribe(nc.respSub, nc.respHandler)
	if err != nil {
		return nil, err
	}
	nc.respScanf = strings.ReplaceAll(nc.respSub, "*", "%s")
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
		sb.WriteByte(rDigits[rn%base])
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

	bucket, err := nc.createNewRequestAndSend(ctx, subj, data)
	if err != nil {
		return nil, err
	}
	return bucket.ch, nil
}

func (nc *ConsumerClient) heartBeatRespHandler(msg *nats.Msg) {
	request := new(HeartBeatRequest)
	err := json.Unmarshal(msg.Data, request)
	if err != nil {
		log.Err(err).Msg("Failed to parse heart beat request for NATs based consumer client")
		return
	}

	nonActiveStreamIds := nc.getNotActiveStreamIds(request.ActiveStreamIds)

	data, err := json.Marshal(ConsumerHeartBeatResponse{NonActiveStreamIds: nonActiveStreamIds})
	if err != nil {
		log.Err(err).Msg("failed to marshal ConsumerHeartBeatResponse")
		return
	}

	err = nc.Conn.Publish(msg.Reply, data)
	if err != nil {
		log.Err(err).Msg("failed to publish heart beat response")
		return
	}
}

// createNewRequestAndSend sets up and sends a new request, returning the response bucket.
func (nc *ConsumerClient) createNewRequestAndSend(
	ctx context.Context,
	subj string,
	data []byte) (*streamingBucket, error) {
	nc.mu.Lock()

	// Create new literal Inbox and map to a bucket.
	respInbox := nc.newRespInbox()
	token := respInbox[nc.respSubLen:]
	bucket := newStreamingBucket(ctx, token, subj)
	nc.respMap[token] = bucket
	nc.reqSubMap[subj] = append(nc.reqSubMap[subj], bucket)

	nc.mu.Unlock()

	streamRequest := Request{
		ConsumerID:          nc.respSubPrefix,
		StreamID:            token,
		HeartBeatRequestSub: nc.heartBeatRequestSub,
		Data:                data,
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

func (nc *ConsumerClient) getNotActiveStreamIds(activeStreamIDsAtProducer map[string][]string) map[string][]string {
	nonActiveStreamIds := make(map[string][]string)
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	// Loop through all active stream ids at producer
	for subject, producerStreamIds := range activeStreamIDsAtProducer {
		consumerBuckets, consumerHasSubject := nc.reqSubMap[subject]

		// Check if request subject does not exist in consumer
		if !consumerHasSubject {
			nonActiveStreamIds[subject] = producerStreamIds
			continue
		}

		nonRecentBuckets := lo.Filter(consumerBuckets, func(bucket *streamingBucket, _ int) bool {
			return time.Since(bucket.createdAt) < nc.config.StreamCancellationBufferDuration
		})

		// If no non-recent buckets, means all are active streams
		if len(nonRecentBuckets) == 0 {
			continue
		}

		nonRecentStreamIds := lo.Map(nonRecentBuckets, func(bucket *streamingBucket, _ int) string {
			return bucket.token
		})

		_, nonActiveStreamIds[subject] = lo.Difference(nonRecentStreamIds, producerStreamIds)
	}
	return nonActiveStreamIds
}

// NewWriter creates a new streaming writer.
func (nc *ConsumerClient) NewWriter(subject string) *Writer {
	return &Writer{
		conn:    nc.Conn,
		subject: subject,
	}
}
