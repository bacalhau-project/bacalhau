// DataFrame wraps the byte framing used by docker to multiplex output
// from a docker container when there is no TTY in use. Although docker
// provides some functions to work with this stream, we want to keep
// our data in this form as long as possible, as well as making it
// implementable by other executors.
//
// Docker documents this as:
//
//	[8]byte{STREAM_TYPE, 0, 0, 0, SIZE1, SIZE2, SIZE3, SIZE4}[]byte{OUTPUT}
//
// STREAM_TYPE can be 1 for stdout and 2 for stderr
//
// SIZE1, SIZE2, SIZE3, and SIZE4 are four bytes of uint32 encoded as big endian.
// This is the size of OUTPUT.
package logger

import (
	"encoding/binary"
	"fmt"
	"io"
)

type StreamTag byte

const (
	UnknownStreamTag StreamTag = iota
	StdoutStreamTag
	StderrStreamTag
)

const headerLength = 8

type DataFrame struct {
	Tag  StreamTag
	Size int
	Data []byte
}

var EmptyDataFrame DataFrame

func NewDataFrameFromReader(reader io.Reader) (DataFrame, error) {
	header := make([]byte, headerLength)

	n, err := reader.Read(header)
	if err != nil {
		return DataFrame{}, err
	}
	if n != headerLength {
		return DataFrame{}, fmt.Errorf("unable to read dataframe header")
	}

	df := DataFrame{}
	df.Tag = StreamTag(binary.LittleEndian.Uint32(header))
	df.Size = int(binary.BigEndian.Uint32(header[4:]))
	df.Data = make([]byte, df.Size)

	n, err = reader.Read(df.Data)
	if err != nil {
		return DataFrame{}, err
	}
	if n != df.Size {
		return DataFrame{}, fmt.Errorf("unable to read dataframe data, read %d wanted %d", n, df.Size)
	}

	return df, nil
}

// NewDataFrameFromBytes will compose a new data frame
// wrapping the provided tag and data with the necessary
// structure.
func NewDataFrameFromData(tag StreamTag, data []byte) DataFrame {
	return DataFrame{
		Tag:  tag,
		Size: len(data),
		Data: append([]byte(nil), data...),
	}
}

// ToBytes converts the data frame into a format suitable for
// transmission across a Writer.
func (df DataFrame) ToBytes() []byte {
	output := make([]byte, headerLength+df.Size)
	binary.LittleEndian.PutUint32(output, uint32(df.Tag))
	binary.BigEndian.PutUint32(output[4:], uint32(df.Size))
	copy(output[8:], df.Data)
	return output
}
