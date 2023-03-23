package wasmlogs

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"sync"

	"github.com/rs/zerolog/log"
)

type LogFile struct {
	ctx             context.Context
	filename        string
	file            *os.File
	wg              sync.WaitGroup
	IncomingChannel chan Message
}

func NewLogFile(ctx context.Context, filename string) (*LogFile, error) {
	logfile := &LogFile{
		ctx:             ctx,
		filename:        filename,
		IncomingChannel: make(chan Message, DefaultMessageChannelSize),
	}

	file, err := os.OpenFile(logfile.filename, os.O_RDWR|os.O_CREATE, DefaultFilePerms)
	if err != nil {
		return nil, err
	}
	logfile.file = file

	logfile.wg.Add(1)
	go logfile.run()

	return logfile, nil
}

func (l *LogFile) run() {
	defer l.wg.Done()

	var compactBuffer bytes.Buffer

	for {
		select {
		case <-l.ctx.Done():
			return
		case m := <-l.IncomingChannel:
			compactBuffer.Reset()

			bytes, err := json.Marshal(m)
			if err != nil {
				log.Ctx(l.ctx).Err(err).Msg("failed to unmarshall json into bytes")
				continue
			}

			err = json.Compact(&compactBuffer, bytes)
			if err != nil {
				log.Ctx(l.ctx).Err(err).Msg("failed to compact json bytes")
				continue
			}

			// Trailing newline to the end of the compacted json
			compactBuffer.WriteByte('\n')
			wrote, err := l.file.Write(compactBuffer.Bytes())
			if err != nil || wrote == 0 {
				log.Ctx(l.ctx).Err(err).Msg("failed to write to logfile")
			}
		}
	}
}

func (l *LogFile) Close() {
	close(l.IncomingChannel)

	l.file.Close()
	os.Remove(l.filename)
}
