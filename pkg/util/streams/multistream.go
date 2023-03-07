package streams

import (
	"context"
	"io"
)

// MultiSourceStream reads from a provided set of readers (each with a unique
// tag) and after wrapping them in a frame, writes them to the provided
// io.Writer.
type MultiSourceStream struct {
	writer  io.Writer
	readers []TaggedReader
}

func NewMultiSourceStream(writer io.Writer, readers ...TaggedReader) *MultiSourceStream {
	return &MultiSourceStream{
		writer:  writer,
		readers: readers,
	}
}

// Start will begin reading from the tagged readers and writing
// to the writer.
func (m *MultiSourceStream) Copy(ctx context.Context) (int, error) {
	buffer := make([]byte, BufferSize)

	completeCount := 0
	totalBytes := 0

	for {
		// Once all readers have been read from, we can stop looping as
		// there is nothing else to do.
		if completeCount == len(m.readers) {
			break
		}

		select {
		case <-ctx.Done():
			return totalBytes, ctx.Err()
		default:
			for _, reader := range m.readers {
				if reader.complete {
					continue
				}

				// Read from the current TaggedReader, but start from 3 bytes
				// in to leave space for the framing of the data. We will write
				// a BE uint16 into the first 2 bytes of the buffer, followed
				// by the one byte tag, and then the data being read. This
				// means that the Read can only read len(buffer)-3 bytes maximum.
				read, err := reader.Read(buffer)
				if read > 0 {
					frame := NewDataFrame()
					frame.loadData(reader.tag, buffer[:read])
					_, writeErr := m.writer.Write(frame.toBytes())
					if writeErr != nil {
						return totalBytes, writeErr
					}
				}

				totalBytes += read

				if err != nil {
					reader.complete = true
					completeCount += 1
				}

				if ctx.Err() != nil {
					return totalBytes, ctx.Err()
				}
			}
		}
	}

	return totalBytes, nil
}

// StreamReader encapsulates a reader which is receiving data from a
// MultiSourceStream, and knows how to separate the dataframes
// being written to it.
type StreamReader struct {
	reader io.Reader
}

func NewStreamReader(r io.Reader) StreamReader {
	return StreamReader{reader: r}
}

type DataHandler func(Tag, []byte) bool

// HandleData will continuously read from it's encapsulated
// read, decoding dataframes as it goes. Each frame is
// provided to the consumer, who can decide whether they
// wish to continue processing or stop immediately.
//
// This function blocks until either all of the data is
// consumed (an EOF is read), an error occurs reading, or
// the consumer returns False.
func (s *StreamReader) HandleData(handler DataHandler) {
	buffer := make([]byte, BufferSize)

	for {
		read, err := s.reader.Read(buffer)
		if read > 0 {
			d := NewDataFrame().read(buffer[:read])

			shouldContinue := handler(d.tag, d.data)
			if !shouldContinue {
				break
			}
		}

		if err != nil {
			break
		}
	}
}
