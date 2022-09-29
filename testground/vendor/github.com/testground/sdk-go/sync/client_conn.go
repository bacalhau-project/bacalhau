package sync

import (
	"context"
	"errors"
	"strconv"
	"time"

	sync "github.com/testground/sync-service"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func (c *DefaultClient) nextID() (id string) {
	c.nextMu.Lock()
	id = strconv.Itoa(c.next)
	c.next++
	c.nextMu.Unlock()
	return id
}

func (c *DefaultClient) responsesWorker() {
	for {
		res, err := c.readSocket()
		if err != nil {
			if errors.Is(err, context.Canceled) ||
				errors.Is(c.ctx.Err(), context.Canceled) ||
				websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				break
			}

			c.log.Fatalw("error while reading socket", "error", err)
		}

		var ch chan *sync.Response
		c.handlersMu.Lock()
		ch = c.handlers[res.ID]
		c.handlersMu.Unlock()

		if ch == nil {
			c.log.Warnf("no handler available for response: %s", res.ID)
		} else {
			ch <- res
		}
	}

	c.wg.Done()
}

func (c *DefaultClient) makeRequest(ctx context.Context, req *sync.Request) (chan *sync.Response, error) {
	if c.ctx.Err() != nil {
		return nil, errors.New("tried to make request after context being cancelled")
	}

	if req.ID == "" {
		req.ID = c.nextID()
	}

	ch := make(chan *sync.Response)

	c.handlersMu.Lock()
	c.handlers[req.ID] = ch
	c.handlersMu.Unlock()

	err := c.writeSocket(req)
	if err != nil {
		return nil, err
	}

	c.wg.Add(1)

	go func() {
		// Wait for either of the contexts to fire.
		select {
		case <-c.ctx.Done():
		case <-ctx.Done():
		}

		c.handlersMu.Lock()
		close(c.handlers[req.ID])
		delete(c.handlers, req.ID)
		c.handlersMu.Unlock()
		c.wg.Done()
	}()

	return ch, nil
}

func (c *DefaultClient) readSocket() (*sync.Response, error) {
	// After one hour without receiving information from the sync service,
	// the test will inevitably fail. Note(hacdias): consider changing
	// the timeout to a larger value in case slower tests fail. The same
	// value must be changed on the sync service side too.
	ctx, cancel := context.WithTimeout(c.ctx, time.Hour)
	defer cancel()

	var req *sync.Response
	err := wsjson.Read(ctx, c.socket, &req)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New("received nil from socket")
	}
	return req, err
}

func (c *DefaultClient) writeSocket(req *sync.Request) error {
	ctx, cancel := context.WithTimeout(c.ctx, time.Second)
	defer cancel()
	return wsjson.Write(ctx, c.socket, req)
}
