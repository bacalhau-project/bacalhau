package handler

import (
	"bufio"
	"bytes"
	"time"

	"github.com/spf13/cobra"
)

func ReadFromStdinIfAvailable(cmd *cobra.Command) ([]byte, error) {
	r := bufio.NewReader(cmd.InOrStdin())
	reader := bufio.NewReader(r)

	// buffered channel of dataStream
	dataStream := make(chan []byte, 1)

	// Run scanner.Bytes() function in it's own goroutine and pass back it's
	// response into dataStream channel.
	go func() {
		for {
			res, err := reader.ReadBytes("\n"[0])
			if err != nil {
				break
			}
			dataStream <- res
		}
		close(dataStream)
	}()

	// Listen on dataStream channel AND a timeout channel - which ever happens first.
	var err error
	var bytesResult bytes.Buffer
	timedOut := false
	select {
	case res := <-dataStream:
		_, err = bytesResult.Write(res)
		if err != nil {
			return nil, err
		}
	case <-time.After(time.Duration(10) * time.Millisecond): //nolint:gomnd // 10ms timeout
		timedOut = true
	}

	if timedOut {
		cmd.Println("No input provided, waiting ... (Ctrl+D to complete)")
	}

	for read := range dataStream {
		_, err = bytesResult.Write(read)
	}

	return bytesResult.Bytes(), err
}
