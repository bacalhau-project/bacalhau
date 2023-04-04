package wasmlogs

import (
	"context"
	"sync"
)

const (
	StdoutTag = "stdout"
	StderrTag = "stderr"

	DefaultMessageChannelSize = 32
	DefaultFilePerms          = 0755
)

type LogManager struct {
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	logfile *LogFile
	//	mbuffer        *generic.MessageBuffer[*Message]
	inChannel      chan Message
	stdoutLive     chan Message
	stderrLive     chan Message
	stdoutWantLive chan bool
	stderrWantLive chan bool
}

func NewLogManager(c context.Context, filename string) (*LogManager, error) {
	ctx, cancel := context.WithCancel(c)
	mgr := &LogManager{
		ctx:            ctx,
		cancel:         cancel,
		inChannel:      make(chan Message, DefaultMessageChannelSize),
		stdoutLive:     make(chan Message, DefaultMessageChannelSize),
		stderrLive:     make(chan Message, DefaultMessageChannelSize),
		stdoutWantLive: make(chan bool, 1),
		stderrWantLive: make(chan bool, 1),
	}

	l, err := NewLogFile(ctx, filename)
	if err != nil {
		return nil, err
	}
	mgr.logfile = l

	return mgr, nil
}

// GetWriters returns two io.Writer objects, a stdout and a stderr which
// whilst looking to the caller like a writer for a file, are in fact
// being used to ship data over channels.
func (mgr *LogManager) GetWriters() (*LogWriter, *LogWriter) {
	stdout := NewLogWriter(mgr.ctx, "stdout", mgr.inChannel)
	stderr := NewLogWriter(mgr.ctx, "stderr", mgr.inChannel)

	mgr.wg.Add(1)
	go mgr.readWriters()

	return stdout, stderr
}

// readWriters only responsibility is to read from the two channels
// and write to a third, which is the one owned by the LogFile.  This
// channels are being written to by the LogWriters created in
// `GetWriters` and so the writers should not be blocking at all.
func (mgr *LogManager) readWriters() {
	defer mgr.wg.Done()

	stdoutReady := false
	stderrReady := false

	// TODO: Properly cache recent messages and track the last
	// timestamp we sent so that when a reader wants a livestream
	// we can make sure they don't miss out.
	lastStdOut := Message{}
	lastStdErr := Message{}

	for {
		select {
		case <-mgr.stdoutWantLive:
			stdoutReady = true
			if lastStdOut.Stream != "" {
				mgr.stdoutLive <- lastStdOut
			}
		case <-mgr.stderrWantLive:
			stderrReady = true
			if lastStdErr.Stream != "" {
				mgr.stderrLive <- lastStdErr
			}
		case m, more := <-mgr.inChannel:
			if !more {
				return
			}
			mgr.logfile.IncomingChannel <- m
			if m.Stream == StderrTag {
				lastStdErr = m
			} else if m.Stream == StdoutTag {
				lastStdOut = m
			}
			if stderrReady && m.Stream == StderrTag {
				mgr.stderrLive <- m
			}
			if stdoutReady && m.Stream == StdoutTag {
				mgr.stdoutLive <- m
			}
		case <-mgr.ctx.Done():
			return
		}
	}
}

// func (mgr *LogManager) GetMuxedReader(follow bool) (*LogReader, error) {
// 	r1, r2, err := GetReaders(follow)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// }

// GetReaders returns two LogReader structs which will read all of the
// data from the LogFile, and then once it is up to date, will 'read'
// messages that are received in `readWriters`
func (mgr *LogManager) GetReaders(follow bool) (*LogReader, *LogReader, error) {
	stdout, err := NewLogReader(LogReaderOptions{
		ctx:            mgr.ctx,
		filename:       mgr.logfile.filename,
		streamName:     "stdout",
		wantLiveStream: mgr.stdoutWantLive,
		liveStream:     mgr.stdoutLive,
		follow:         follow,
	})
	if err != nil {
		return nil, nil, err
	}

	stderr, err := NewLogReader(LogReaderOptions{
		ctx:            mgr.ctx,
		filename:       mgr.logfile.filename,
		streamName:     "stderr",
		wantLiveStream: mgr.stderrWantLive,
		liveStream:     mgr.stderrLive,
		follow:         follow,
	})
	if err != nil {
		return nil, nil, err
	}

	return stdout, stderr, nil
}

func (mgr *LogManager) Close() {
	mgr.cancel()
	mgr.wg.Wait()

	close(mgr.inChannel)
	close(mgr.stdoutLive)
	close(mgr.stderrLive)
	close(mgr.stdoutWantLive)
	close(mgr.stderrWantLive)

	mgr.logfile.Close()
}
