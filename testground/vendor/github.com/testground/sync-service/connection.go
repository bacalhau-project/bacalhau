package sync

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/testground/testground/pkg/logging"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type connection struct {
	*websocket.Conn
	service   Service
	ctx       context.Context
	responses chan *Response

	// cancelFuncs contains cancel functions for requests that can
	// be canceled, such as subscribes.
	cancelFuncs   map[string]context.CancelFunc
	cancelFuncsMu sync.RWMutex
}

func (c *connection) consumeRequests() error {
	for {
		req, err := c.readTimeout(time.Hour)
		if err != nil {
			return err
		}

		if req.IsCancel {
			var cancel context.CancelFunc
			c.cancelFuncsMu.Lock()
			cancel = c.cancelFuncs[req.ID]
			delete(c.cancelFuncs, req.ID)
			c.cancelFuncsMu.Unlock()

			if cancel == nil {
				logging.S().Warnw("attempt to cancel not cancellable request", "id", req.ID)
				continue
			}
		}

		switch {
		case req.PublishRequest != nil:
			go c.publishHandler(req.ID, req.PublishRequest)
		case req.SubscribeRequest != nil:
			go c.subscribeHandler(req.ID, req.SubscribeRequest)
		case req.BarrierRequest != nil:
			go c.barrierHandler(req.ID, req.BarrierRequest)
		case req.SignalEntryRequest != nil:
			go c.signalEntryHandler(req.ID, req.SignalEntryRequest)
		}
	}
}

func (c *connection) publishHandler(id string, req *PublishRequest) {
	ctx, cancel := context.WithTimeout(c.ctx, time.Second*10)
	defer cancel()

	resp := &Response{ID: id}
	seq, err := c.service.Publish(ctx, req.Topic, req.Payload)
	if err != nil {
		resp.Error = err.Error()
	} else {
		resp.PublishResponse = &PublishResponse{
			Seq: seq,
		}
	}
	c.responses <- resp
}

func (c *connection) subscribeHandler(id string, req *SubscribeRequest) {
	ctx, cancel := context.WithCancel(c.ctx)
	c.cancelFuncsMu.Lock()
	c.cancelFuncs[id] = cancel
	c.cancelFuncsMu.Unlock()

	sub, err := c.service.Subscribe(ctx, req.Topic)
	if err != nil {
		c.responses <- &Response{ID: id, Error: err.Error()}
		return
	}

	for {
		select {
		case data := <-sub.outCh:
			c.responses <- &Response{ID: id, SubscribeResponse: data}
		case err = <-sub.doneCh:
			if err == nil || errors.Is(err, context.Canceled) {
				// Cancelled by the user.
				return
			}

			c.responses <- &Response{ID: id, Error: err.Error()}
			return
		case <-c.ctx.Done():
			// Cancelled by the user.
			return
		}
	}
}

func (c *connection) barrierHandler(id string, req *BarrierRequest) {
	resp := &Response{ID: id}
	err := c.service.Barrier(c.ctx, req.State, req.Target)
	if err != nil {
		resp.Error = err.Error()
	}
	c.responses <- resp
}

func (c *connection) signalEntryHandler(id string, req *SignalEntryRequest) {
	ctx, cancel := context.WithTimeout(c.ctx, time.Second*10)
	defer cancel()

	resp := &Response{ID: id}
	seq, err := c.service.SignalEntry(ctx, req.State)
	if err != nil {
		resp.Error = err.Error()
	} else {
		resp.SignalEntryResponse = &SignalEntryResponse{
			Seq: seq,
		}
	}
	c.responses <- resp
}

func (c *connection) consumeResponses() error {
	for {
		select {
		case resp := <-c.responses:
			err := c.writeTimeout(time.Second*10, resp)
			if err != nil {
				return err
			}
		case <-c.ctx.Done():
			return c.ctx.Err()
		}
	}
}

func (c *connection) readTimeout(timeout time.Duration) (*Request, error) {
	ctx, cancel := context.WithTimeout(c.ctx, timeout)
	defer cancel()

	var req *Request
	err := wsjson.Read(ctx, c.Conn, &req)
	return req, err
}

func (c *connection) writeTimeout(timeout time.Duration, resp *Response) error {
	ctx, cancel := context.WithTimeout(c.ctx, timeout)
	defer cancel()
	return wsjson.Write(ctx, c.Conn, resp)
}
