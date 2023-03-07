package streams

import "io"

type Tag byte

const (
	StdOutTag Tag = iota
	StdErrTag
)

type TaggedReader struct {
	reader   io.Reader
	tag      Tag
	complete bool
}

func NewTaggedReader(tag Tag, reader io.Reader) TaggedReader {
	return TaggedReader{reader: reader, tag: tag, complete: false}
}

func (t *TaggedReader) Read(b []byte) (int, error) {
	return t.reader.Read(b)
}
