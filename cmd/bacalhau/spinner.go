package bacalhau

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/theckman/yacspin"
)

const (
	TickerUpdateFrequency        = 50 * time.Millisecond
	SpinnerFormatDurationDefault = 30 * time.Millisecond
	TextLineupSpacing            = 10
)

type SpinActionType int

const (
	SpinActionStart SpinActionType = iota
	SpinActionStop
)

var SpinnerEmoji = [...]string{"🐟", "🐠", "🐡"}

const SpacerText = "  ..."

type Spinner struct {
	cfg           yacspin.Config
	spin          yacspin.Spinner
	msg           LineMessage
	maxWidth      int
	ticker        *time.Ticker
	msgMutex      sync.Mutex
	ctx           context.Context
	cancel        context.CancelFunc
	actionChannel chan SpinActionType
	doneChannel   chan bool
	signalChannel chan os.Signal
}

// NewSpinner creates a new `Spinner` using the provided io.Writer
// and expecting a message of no more than `maxWidth` characters.
// The `maxWidth` is required to ensure that following steps in
// the lifetime of the spinner line up.
func NewSpinner(ctx context.Context, w io.Writer, maxWidth int) (*Spinner, error) {
	ctx, cancel := context.WithCancel(ctx)

	s := &Spinner{
		maxWidth:    maxWidth,
		doneChannel: make(chan bool),
		ctx:         ctx,
		cancel:      cancel,
	}

	spacer := 8

	var spinnerCharSet []string
	for _, emoji := range SpinnerEmoji {
		for i := 0; i < spacer; i++ {
			spinnerCharSet = append(spinnerCharSet, fmt.Sprintf("%s%s%s",
				strings.Repeat(" ", spacer-i),
				emoji,
				strings.Repeat(" ", i)))
		}
	}

	s.cfg = yacspin.Config{
		Frequency: 100 * time.Millisecond,
		CharSet:   spinnerCharSet,
		Writer:    w,
	}

	ys, err := yacspin.New(s.cfg)
	if err != nil {
		log.Ctx(s.ctx).Err(err)
		return nil, fmt.Errorf("failed to generate spinner from methods: %v", err)
	}
	s.spin = *ys

	if err := s.spin.CharSet(spinnerCharSet); err != nil {
		log.Ctx(s.ctx).Err(err)
		return nil, fmt.Errorf("failed to set charset: %v", err)
	}

	s.signalChannel = make(chan os.Signal, 2)
	signal.Notify(s.signalChannel, ShutdownSignals...)

	return s, nil
}

// Done stops the spinner, ignoring any errors as there
// is no further use for the spinner.
func (s *Spinner) Done(success bool) {
	if !success {
		s.msg.Failure = true
		s.updateText(time.Duration(0) * time.Millisecond)
	}

	_ = s.spin.Stop()
	s.cancel()

	_, _ = s.cfg.Writer.Write([]byte("\n"))
}

// NextStep completes the current line (if any) and
// progresses to the next line, starting a new timer.
// If failure is passed, then the Done text/image is
// displayed as err, X - and it is expected that the
// process has completed and Done will be called
// immediately.
func (s *Spinner) NextStep(line string) {
	// Stop the spinner and wait until it is stopped
	if s.spin.Status() == yacspin.SpinnerRunning {
		s.actionChannel <- SpinActionStop
		<-s.doneChannel
	}
	s.msgMutex.Lock()

	s.msg = NewLineMessage(line, s.maxWidth)
	s.cfg.Prefix = s.msg.Message

	s.spin.Prefix(fmt.Sprintf("%s%s", s.msg.Message, SpacerText))
	s.msgMutex.Unlock()

	s.updateText(time.Duration(0) * time.Millisecond)

	// Start the spinner and wait until it is started if we have
	// not just displayed an error state.
	s.actionChannel <- SpinActionStart
	<-s.doneChannel
}

func (s *Spinner) updateText(duration time.Duration) {
	s.msg.TimerString = spinnerFmtDuration(duration)
	s.spin.Message(fmt.Sprintf("%s %s", SpacerText, s.msg.TimerString))
	s.spin.StopMessage(s.msg.PrintDone())
}

// Run starts the spinner running and accepting messages from other
// functions belonging to the spinner using the `actionChannel` to
// trigger various states for the spinner.
func (s *Spinner) Run() {
	s.actionChannel = make(chan SpinActionType, 1)
	s.ticker = time.NewTicker(TickerUpdateFrequency)

	// Make sure we set the values up front as if we had received a timer tick
	// in case the next message is set in less than `TickerUpdateFrequency`
	s.msgMutex.Lock()
	s.updateText(time.Duration(0) * time.Millisecond)
	s.msgMutex.Unlock()

	go func() {
		var msg SpinActionType
		jobStartedAt := time.Now()

		for {
			select {
			case t := <-s.ticker.C:
				s.msgMutex.Lock()
				s.updateText(t.Sub(jobStartedAt))
				s.msgMutex.Unlock()
			case msg = <-s.actionChannel:
				switch msg {
				case SpinActionStart:
					// Reset the start time and the ticker
					// used to update the line text.
					jobStartedAt = time.Now()
					s.ticker.Reset(TickerUpdateFrequency)

					err := s.spin.Start()
					if err != nil {
						log.Ctx(s.ctx).Err(err)
					}

					s.doneChannel <- true
				case SpinActionStop:
					s.ticker.Stop()

					err := s.spin.Stop()
					if err != nil {
						log.Ctx(s.ctx).Err(err)
					}
					s.doneChannel <- true
				}
			case signal := <-s.signalChannel:
				log.Ctx(s.ctx).Debug().Msgf("Captured %v. Exiting...", s)
				if signal == os.Interrupt {
					s.ticker.Stop()

					s.Done(false)
					_, _ = os.Stderr.WriteString("\n\rPrintout canceled.")

					os.Exit(0)
				}
			}
		}
	}()
}

type LineMessage struct {
	Message     string
	TimerString string
	StopString  string
	Width       int
	Failure     bool
}

func NewLineMessage(msg string, maxWidth int) LineMessage {
	return LineMessage{
		Message:     formatLineMessage(msg, maxWidth),
		TimerString: spinnerFmtDuration(SpinnerFormatDurationDefault),
		StopString:  "",
		Width:       6,
		Failure:     false,
	}
}

func (f *LineMessage) String() string {
	return fmt.Sprintf("%s %s ",
		f.Message,
		f.StopString)
}

func (f *LineMessage) PrintDone() string {
	if f.Failure {
		return f.PrintError()
	}

	return fmt.Sprintf("%s%s%s %s",
		f.String(),
		strings.Repeat(".", f.Width+TextLineupSpacing), // extra spacing
		" done ✅ ",
		f.TimerString)
}

func (f *LineMessage) PrintError() string {
	return fmt.Sprintf("%s%s%s %s",
		f.String(),
		strings.Repeat(".", f.Width+TextLineupSpacing), // extra spacing
		" err  ❌ ",
		f.TimerString)
}

func formatLineMessage(msg string, maxLength int) string {
	return fmt.Sprintf("\t%s%s",
		strings.Repeat(" ", maxLength-len(msg)+2), msg)
}

func spinnerFmtDuration(d time.Duration) string {
	d = d.Round(time.Millisecond)

	min := (d % time.Hour) / time.Minute
	sec := (d % time.Minute) / time.Second
	ms := (d % time.Second) / time.Millisecond / 100

	minString, secString, msString := "", "", ""
	if min > 0 {
		minString = fmt.Sprintf("%02dm", min)
		secString = fmt.Sprintf("%02d", sec)
		msString = fmt.Sprintf(".%01ds", ms)
	} else if sec > 0 {
		secString = fmt.Sprintf("%01d", sec)
		msString = fmt.Sprintf(".%01ds", ms)
	} else {
		msString = fmt.Sprintf("0.%01ds", ms)
	}
	// If hour string exists, set it
	return fmt.Sprintf("%s%s%s", minString, secString, msString)
}
