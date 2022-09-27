package runtime

import (
	"io"
	"time"

	"github.com/avast/retry-go"
	_ "github.com/influxdata/influxdb1-client"
	client "github.com/influxdata/influxdb1-client/v2"

	"github.com/testground/sdk-go"
)

type Batcher interface {
	io.Closer

	WritePoint(p *client.Point)
}

type batcher struct {
	re        *RunEnv
	client    client.Client
	length    int
	interval  time.Duration
	retryOpts []retry.Option

	writeCh chan *client.Point
	flushCh chan struct{}
	doneCh  chan struct{}

	pending []*client.Point
	sending []*client.Point
	sendRes chan error
	doneErr chan error
}

func newBatcher(re *RunEnv, cli client.Client, length int, interval time.Duration, retry ...retry.Option) *batcher {
	b := &batcher{
		re:        re,
		client:    cli,
		length:    length,
		interval:  interval,
		retryOpts: retry,

		writeCh: make(chan *client.Point),
		flushCh: make(chan struct{}, 1),
		sendRes: make(chan error, 1),
		doneCh:  make(chan struct{}),
		doneErr: make(chan error),

		pending: nil,
		sending: nil,
	}

	go b.background()

	return b
}

func (b *batcher) background() {
	tick := time.NewTicker(b.interval)
	defer tick.Stop()

	attemptFlush := func() {
		if b.sending != nil {
			// there's already a flush taking place.
			return
		}
		select {
		case b.flushCh <- struct{}{}:
		default:
			// there's a flush queued to be accepted.
		}
	}

	for {
		select {
		case p := <-b.writeCh:
			b.pending = append(b.pending, p)
			if len(b.pending) >= b.length {
				attemptFlush()
			}

		case err := <-b.sendRes:
			if err == nil {
				b.pending = b.pending[len(b.sending):]
				if sdk.Verbose {
					b.re.RecordMessage("influxdb: uploaded %d points", len(b.sending))
				}
			} else {
				b.re.RecordMessage("influxdb: failed to upload %d points; err: %s", len(b.sending), err)
			}
			b.sending = nil
			if len(b.pending) >= b.length {
				attemptFlush()
			}

		case <-tick.C:
			attemptFlush()

		case <-b.flushCh:
			if b.sending != nil {
				continue
			}
			l := len(b.pending)
			if l == 0 {
				continue
			}
			if l > b.length {
				l = b.length
			}
			b.sending = b.pending[:l]
			go b.send()

		case <-b.doneCh:
			if b.sending != nil {
				// we are currently sending, wait for the send to finish first.
				if err := <-b.sendRes; err == nil {
					b.pending = b.pending[len(b.sending):]
					if sdk.Verbose {
						b.re.RecordMessage("influxdb: uploaded %d points", len(b.sending))
					}
				} else {
					b.re.RecordMessage("influxdb: failed to upload %d points; err: %s", len(b.sending), err)
				}
			}

			var err error
			if len(b.pending) > 0 {
				// send all remaining data at once.
				b.sending = b.pending
				go b.send()
				err = <-b.sendRes
				if err == nil {
					if sdk.Verbose {
						b.re.RecordMessage("influxdb: uploaded %d points", len(b.sending))
					}
				} else {
					b.re.RecordMessage("influxdb: failed to upload %d points; err: %s", len(b.sending), err)
				}
				b.sending = nil
			}
			b.doneErr <- err
			return
		}
	}
}

func (b *batcher) WritePoint(p *client.Point) {
	b.writeCh <- p
}

// Close flushes any remaining points and returns any errors from the final flush.
func (b *batcher) Close() error {
	select {
	case _, ok := <-b.doneCh:
		if !ok {
			return nil
		}
	default:
	}
	close(b.doneCh)
	return <-b.doneErr
}

func (b *batcher) send() {
	points, err := client.NewBatchPoints(client.BatchPointsConfig{Database: "testground"})
	if err != nil {
		b.sendRes <- err
		return
	}

	for _, p := range b.sending {
		points.AddPoint(p)
	}

	err = retry.Do(func() error { return b.client.Write(points) }, b.retryOpts...)
	b.sendRes <- err
}

type nilBatcher struct {
	client.Client
}

func (n *nilBatcher) WritePoint(p *client.Point) {
	bp, _ := client.NewBatchPoints(client.BatchPointsConfig{Database: "testground"})
	bp.AddPoint(p)
	_ = n.Write(bp)
}

func (n *nilBatcher) Close() error {
	return nil
}

var _ Batcher = (*nilBatcher)(nil)
