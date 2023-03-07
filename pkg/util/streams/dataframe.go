package streams

import (
	"encoding/binary"
	"fmt"
)

const (
	BufferSize  = 65535
	TagIndex    = 2
	FrameLength = 3
)

type DataFrame struct {
	length uint16
	tag    Tag
	data   []byte
	err    error
}

func NewDataFrame() *DataFrame {
	return &DataFrame{}
}

// read is used to create a DataFrame using bytes most
// likely read from the network
func (d *DataFrame) read(b []byte) *DataFrame {
	if len(b) < FrameLength {
		d.err = fmt.Errorf("malformed header, not enough bytes")
		return d
	}

	d.length = binary.BigEndian.Uint16(b[0:2])
	d.tag = Tag(b[TagIndex])
	d.data = make([]byte, d.length)
	copy(d.data, b[FrameLength:FrameLength+d.length])

	return d
}

// fromData constructs a new data frame using the data provided.
func (d *DataFrame) loadData(tag Tag, data []byte) {
	d.length = uint16(len(data))
	d.tag = tag
	d.data = make([]byte, d.length)
	copy(d.data, data[:d.length])
}

// toBytes converts a DataFrame into a []byte ready for writing
// to the network
func (d *DataFrame) toBytes() []byte {
	buffer := make([]byte, FrameLength+len(d.data))

	binary.BigEndian.PutUint16(buffer, d.length)
	buffer[TagIndex] = byte(d.tag)
	copy(buffer[3:], d.data)

	return buffer
}
