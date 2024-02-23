// This file includes unmodified functions from the HashiCorp Nomad project.
// The original file can be found at:
// https://github.com/hashicorp/nomad/blob/fc9076731c7c920ab0373c224ba8e9fd5544d386/api/api.go
//
// This entire file is licensed under the Mozilla Public License 2.0
// Original Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/gorilla/websocket"
)

// encodeBody prepares the reader to serve as the request body.
//
// Returns the `obj` input if it is a raw io.Reader object; otherwise
// returns a reader of the json format of the passed argument.
func encodeBody(obj interface{}) (io.Reader, error) {
	if reader, ok := obj.(io.Reader); ok {
		return reader, nil
	}

	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)
	if err := enc.Encode(obj); err != nil {
		return nil, err
	}
	return buf, nil
}

// decodeBody is used to JSON decode a body
func decodeBody(resp *http.Response, out interface{}) error {
	switch resp.ContentLength {
	case 0:
		if out == nil {
			return nil
		}
		return errors.New("got 0 byte response with non-nil decode object")
	default:
		dec := json.NewDecoder(resp.Body)
		return dec.Decode(out)
	}
}

// multiCloser is to wrap a ReadCloser such that when close is called, multiple
// Closes occur.
type multiCloser struct {
	reader       io.Reader
	inorderClose []io.Closer
}

func (m *multiCloser) Close() error {
	for _, c := range m.inorderClose {
		if err := c.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (m *multiCloser) Read(p []byte) (int, error) {
	return m.reader.Read(p)
}

// autoUnzip modifies resp in-place, wrapping the response body with a gzip
// reader if the Content-Encoding of the response is "gzip".
func autoUnzip(resp *http.Response) error {
	if resp == nil || resp.Header == nil {
		return nil
	}

	if resp.Header.Get("Content-Encoding") == "gzip" {
		zReader, err := gzip.NewReader(resp.Body)
		if err == io.EOF {
			// zero length response, do not wrap
			return nil
		} else if err != nil {
			// some other error (e.g. corrupt)
			return err
		}

		// The gzip reader does not close an underlying reader, so use a
		// multiCloser to make sure response body does get closed.
		resp.Body = &multiCloser{
			reader:       zReader,
			inorderClose: []io.Closer{zReader, resp.Body},
		}
	}

	return nil
}

func webSocketDialer[T any](ctx context.Context, c *httpClient, endpoint string, in apimodels.GetRequest) (
	<-chan *concurrency.AsyncResult[T], error) {
	r := in.ToHTTPRequest()
	httpR, err := c.toHTTP(ctx, http.MethodGet, endpoint, r)
	if err != nil {
		return nil, err
	}
	httpR.URL.Scheme = "ws"

	// Connect to the server
	conn, resp, err := websocket.DefaultDialer.Dial(httpR.URL.String(), httpR.Header)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read messages from the server, and send them to the conn is closed or the context is cancelled
	ch := make(chan *concurrency.AsyncResult[T], c.config.WebsocketChannelBuffer)
	go func() {
		defer func() {
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			conn.Close()
			close(ch)
		}()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				result := new(concurrency.AsyncResult[T])
				err := conn.ReadJSON(result)
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
						result.Err = err
						ch <- result
					}
					return
				}
				ch <- result
			}
		}
	}()

	return ch, nil
}
