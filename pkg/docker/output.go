package docker

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/rs/zerolog/log"
)

const (
	// // SystemInfo represents non-error messages originating from the system that make it into the multiplexed stream.
	// SystemInfo      stdcopy.StdType = stdcopy.Systemerr + 1
	// SystemStreamEnd represents the end of the multiplexed stream
	SystemStreamEnd stdcopy.StdType = stdcopy.Systemerr + 1

	stdWriterPrefixLen = 8
	stdWriterFdIndex   = 0
	stdWriterSizeIndex = 4
	timestampLen       = 30 // 0-padded RFC3339Nano in UTC

	startingBufLen               = 32*1024 + stdWriterPrefixLen + 1
	endFrameWait   time.Duration = 100 * time.Millisecond
)

// A modified version of Docker stdcopy.StdCopy that processes multiplexed container output streams with timestamps.
// It converts these timestamped frames (created when using the Timestamps LogOption) into a non-timestamped
// format compatible with the original StdCopy.
//
// Optionally, it filters out frames older than the specified 'since' parameter.
//
// The Docker multiplexed stream format follows this structure
//
//	[8]byte{STREAM_TYPE, 0, 0, 0, [4]byte{FRAME_SIZE}}[]byte{OUTPUT}
//
// When the Timestamps option is enabled, the OUTPUT is prefixed with timestamp and space:
//
//	[8]byte{STREAM_TYPE, 0, 0, 0, [4]byte{FRAME_SIZE}}[]byte{[30]byte{TIMESTAMP}, [1]byte{SPACE}, []byte{OUTPUT}}
//
// The timestamp is formatted in RFC3339Nano in UTC timezone with nanoseconds padded
// to 9 decimal places (e.g., "2023-04-01T12:34:56.123400000Z") to ensure alignment.
//
// For more details on Docker's multiplexed stream format, see:
// - github.com/docker/docker/pkg/stdcopy
// - github.com/docker/go-docker/blob/master/container_logs.go
//
// See also discussions on timestamp timezone github.com/moby/moby/pull/35051#issuecomment-334117784
func TimestampedStdCopy(dstout io.Writer, src io.Reader, since time.Time, stopOnEndFrame bool) (written int64, err error) {
	var (
		buf                 = make([]byte, startingBufLen)
		bufLen              = len(buf)
		numRead, numWritten int
		frameSize           int
	)

	for {
		// Make sure we have at least a full header
		for numRead < stdWriterPrefixLen {
			var numRead2 int
			numRead2, err = src.Read(buf[numRead:])
			numRead += numRead2
			if err == io.EOF {
				if !stopOnEndFrame && numRead < stdWriterPrefixLen {
					return written, nil
				}
				time.Sleep(endFrameWait)
				continue
				// break
			}
			if err != nil {
				return 0, err
			}
		}

		// Determine which stream the frame belongs to (stdout/stderr)
		streamType := stdcopy.StdType(buf[stdWriterFdIndex])

		// If encountered a stream end frame, return immediately
		if streamType == SystemStreamEnd {
			return written, nil
		}

		// Retrieve the size of the frame (with the timestamp)
		frameSize = int(binary.BigEndian.Uint32(buf[stdWriterSizeIndex : stdWriterSizeIndex+4]))

		// Check that the frame is at least as long a timestamp and a space
		if frameSize < timestampLen+1 {
			return written, fmt.Errorf("log frame is too short")
		}

		// Check if the buffer is big enough to read the frame.
		// Extend it if necessary.
		if frameSize+stdWriterPrefixLen > bufLen {
			buf = append(buf, make([]byte, frameSize+stdWriterPrefixLen-bufLen+1)...)
			bufLen = len(buf)
		}

		// While the amount of bytes read is less than the size of the frame + header, we keep reading
		for numRead < frameSize+stdWriterPrefixLen {
			var numRead2 int
			numRead2, err = src.Read(buf[numRead:])
			numRead += numRead2
			if err == io.EOF {
				if numRead < frameSize+stdWriterPrefixLen {
					return written, nil
				}
				break
			}
			if err != nil {
				return 0, err
			}
		}

		// if there was an error while writing into the original multiplexed stream (Docker daemon), return it
		if streamType == stdcopy.Systemerr {
			return written, fmt.Errorf("error from daemon in stream: %s", string(buf[stdWriterPrefixLen:frameSize+stdWriterPrefixLen]))
		}

		// if we encountered the stream end, return immediately
		if streamType == SystemStreamEnd {
			return written, nil
		}

		// retrieve the timestamp
		timestamp, err := time.Parse(time.RFC3339Nano, string(buf[stdWriterPrefixLen:stdWriterPrefixLen+timestampLen]))
		if err != nil {
			return written, err
		}

		if !timestamp.Before(since) {
			// write modified frame into the output

			newPrefixBuf := make([]byte, stdWriterPrefixLen)

			// stream type with padding from the original header
			copy(newPrefixBuf, buf[:stdWriterSizeIndex])

			// modified frame size (subtract the length of the timestamp and the space after it)
			newFrameSize := uint32(frameSize - (timestampLen + 1))
			binary.BigEndian.PutUint32(newPrefixBuf[stdWriterSizeIndex:], newFrameSize)

			// write new frame prefix to the destination
			numWritten, err = dstout.Write(newPrefixBuf)

			if err != nil {
				return 0, err
			}
			written += int64(numWritten)

			// write the rest of the frame without the timestamp
			numWritten, err = dstout.Write(buf[stdWriterPrefixLen+timestampLen+1 : frameSize+stdWriterPrefixLen])
			if err != nil {
				return 0, err
			}

			// if the frame has not been fully written: error
			if numWritten != frameSize-(timestampLen+1) {
				return 0, io.ErrShortWrite
			}
			written += int64(numWritten)
		}

		// move the rest of the buffer to the beginning
		copy(buf, buf[frameSize+stdWriterPrefixLen:])
		// move the index
		numRead -= frameSize + stdWriterPrefixLen
	}
}

func ContainerOutputStream(source io.ReadCloser, since time.Time, follow bool) (io.ReadCloser, error) {
	dstReader, dstWriter := io.Pipe()
	go func() {
		defer closer.CloseWithLogOnError("multiplexedWriter", dstWriter)
		defer closer.CloseWithLogOnError("multiplexedReader", source)

		_, err := TimestampedStdCopy(dstWriter, source, since, follow)
		if err != nil {
			log.Error().Err(err).Msg("error reading container output")
		}
	}()
	return dstReader, nil
}

func WriteMultiplexedOutput(dstWriter io.Writer, outputReader io.Reader) error {
	buf := make([]byte, startingBufLen)

	for {
		numRead, err := outputReader.Read(buf)

		// Any data that has been read gets written to the destination
		if numRead > 0 {
			if _, writeErr := dstWriter.Write(buf[:numRead]); writeErr != nil {
				return writeErr
			}
		}

		if err == io.EOF {
			// Write a stream end frame
			if writeErr := writeEndFrame(dstWriter); writeErr != nil {
				return writeErr
			}
			return nil
		} else if err != nil {
			// Return any other error
			return err
		}
	}
}

// Writes an end frame to the given writer indicating that no more data is expected to be received.
// An end frame can be used to follow container logs when it is still running.
func writeEndFrame(dstWriter io.Writer) error {
	buf := make([]byte, stdWriterPrefixLen)

	// Set the stream type to be SystemStreamEnd
	buf[0] = byte(SystemStreamEnd)
	// Rest of the prefix is zero-ed out and there is no data following the prefix in the frame

	// Write the prefix to the destination
	_, err := dstWriter.Write(buf)
	return err
}
