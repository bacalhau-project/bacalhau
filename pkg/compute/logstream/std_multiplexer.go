package logstream

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/pkg/stdcopy"
	"github.com/rs/zerolog/log"
)

const (
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
// If follow is true, it will keep trying to read from the source even if it reaches EOF until a cancellation is received
// or an end frame is encountered.
//
// Optionally, it filters out frames older than the specified 'since' parameter. If nil is passed no filtering is performed
//
// A cancellation can be sent through the cancelCh channel. It will be processed when the next frame is read.
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
//
//nolint:funlen,gocyclo // Complex stream parsing logic, difficult to simplify without losing clarity
func TimestampedStdCopy(
	dstout io.Writer,
	src io.Reader,
	since *time.Time,
	follow bool,
	cancelCh chan struct{},
) (written int64, err error) {
	var (
		buf                 = make([]byte, startingBufLen)
		bufLen              = len(buf)
		numRead, numWritten int
		frameSize           uint32
	)

	followTicker := time.NewTicker(endFrameWait)
	defer followTicker.Stop()
	for {
		// Make sure we have at least a full header
		for numRead < stdWriterPrefixLen {
			var numRead2 int
			numRead2, err = src.Read(buf[numRead:])
			numRead += numRead2
			if err == io.EOF {
				if !follow && numRead < stdWriterPrefixLen {
					// EOF and not following:
					// Stop reading more frames
					log.Debug().Int("buffered_bytes", numRead).Msg("partial header read while not following")
					return written, nil
				}
				// EOF and following:
				// Periodically check if there are updates in the source stream
				// and if a cancellation has been received
				followTicker.Reset(endFrameWait)
				select {
				case <-cancelCh:
					log.Debug().Int("buffered_bytes", numRead).Msg("cancelling read while waiting for header")
					return written, nil
				case <-followTicker.C:
					continue
				}
			}
			if err != nil {
				return 0, err
			}
		}

		// Determine which stream the frame belongs to
		streamType := stdcopy.StdType(buf[stdWriterFdIndex])

		// Retrieve the size of the frame
		frameSize = binary.BigEndian.Uint32(buf[stdWriterSizeIndex : stdWriterSizeIndex+4])

		// If encountered a stream end frame, return immediately even if follow is set
		// because there are no more frames expected after the stream end frame
		if streamType == SystemStreamEnd {
			return written, nil
		}

		// If there was an error while writing into the original multiplexed stream (Docker daemon), return it
		if streamType == stdcopy.Systemerr {
			errorMsg := string(buf[stdWriterPrefixLen : frameSize+stdWriterPrefixLen])
			return written, fmt.Errorf("error from daemon in stream: %s", errorMsg)
		}

		// Check that the frame is at least as long a timestamp and a space
		if frameSize < timestampLen+1 {
			return written, fmt.Errorf("log frame is too short")
		}

		// Check if the buffer is big enough to read the frame.
		// Extend it if necessary.
		if int(frameSize+stdWriterPrefixLen) > bufLen {
			buf = append(buf, make([]byte, int(frameSize+stdWriterPrefixLen)-bufLen+1)...)
			bufLen = len(buf)
		}

		// While the amount of bytes read is less than the size of the frame + header, we keep reading
		for numRead < int(frameSize+stdWriterPrefixLen) {
			var numRead2 int
			numRead2, err = src.Read(buf[numRead:])
			numRead += numRead2
			if err == io.EOF {
				if !follow && numRead < int(frameSize+stdWriterPrefixLen) {
					return written, nil
				}
				followTicker.Reset(endFrameWait)
				select {
				case <-cancelCh:
					log.Debug().Int("buffered_size", numRead).Msg("cancelling read while waiting for frame")
					return written, nil
				case <-followTicker.C:
					continue
				}
			}
			if err != nil {
				return 0, err
			}
		}

		// retrieve the timestamp
		timestamp, err := time.Parse(time.RFC3339Nano, string(buf[stdWriterPrefixLen:stdWriterPrefixLen+timestampLen]))
		if err != nil {
			return written, err
		}

		// Check if cancelled.
		// Note that this is a best effort attempt to cancel the read.
		// If a cancellation arrives white data is being written to the destination,
		// it will only be handled when processing the next frame.
		select {
		case <-cancelCh:
			log.Debug().Int("buffered_size", numRead).Msg("dropping frame due to cancellation")
			return written, nil
		default:
			// continue
		}

		if since == nil || !timestamp.Before(*since) {
			// write modified frame into the output

			newPrefixBuf := make([]byte, stdWriterPrefixLen)

			// stream type with padding from the original header
			copy(newPrefixBuf, buf[:stdWriterSizeIndex])

			// modified frame size (subtract the length of the timestamp and the space after it)
			newFrameSize := frameSize - (timestampLen + 1)
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
			if numWritten != int(frameSize)-(timestampLen+1) {
				return 0, io.ErrShortWrite
			}
			written += int64(numWritten)
		}

		// move the rest of the buffer to the beginning
		copy(buf, buf[frameSize+stdWriterPrefixLen:])
		// move the index
		numRead -= int(frameSize + stdWriterPrefixLen)
	}
}

// StdCopyWithEndFrame is a modified version of Docker stdcopy.StdCopy that writes an end frame
// to the destination when EOF is reached.
func StdCopyWithEndFrame(dstout io.Writer, src io.Reader) (written int64, err error) {
	buf := make([]byte, startingBufLen)
	for {
		numRead, err := src.Read(buf)

		// Any data that has been read gets written to the destination
		if numRead > 0 {
			if numWritten, writeErr := dstout.Write(buf[:numRead]); writeErr != nil {
				written += int64(numWritten)
				return written, writeErr
			}
		}

		if err == io.EOF {
			// Write a stream end frame
			numWritten, writeErr := writeEndFrame(dstout)
			log.Debug().Msg("writing end frame")
			written += int64(numWritten)
			return written, writeErr
		} else if err != nil {
			// Return any other error
			return written, err
		}
	}
}

func writeEndFrame(dstWriter io.Writer) (int, error) {
	buf := make([]byte, stdWriterPrefixLen)

	// Set the stream type to be SystemStreamEnd
	buf[0] = byte(SystemStreamEnd)
	// Rest of the prefix is zero-ed out and there is no data following the prefix in the frame

	// Write the prefix to the destination
	return dstWriter.Write(buf)
}
