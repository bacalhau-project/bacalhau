package client

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"net/http"
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
