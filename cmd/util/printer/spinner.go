package printer

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/rs/zerolog/log"
	"github.com/theckman/yacspin"

	"github.com/bacalhau-project/bacalhau/cmd/util"
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

const (
	SpacerText  = "  ... "
	ReverseText = " ...  "
)

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
	complete      bool
	handleSigint  bool
}

// NewSpinner creates a new `Spinner` using the provided io.Writer
// and expecting a message of no more than `maxWidth` characters.
// The `maxWidth` is required to ensure that following steps in
// the lifetime of the spinner line up.
func NewSpinner(ctx context.Context, w io.Writer, maxWidth int, handleSigint bool) (*Spinner, error) {
	ctx, cancel := context.WithCancel(ctx)

	s := &Spinner{
		maxWidth:     maxWidth,
		doneChannel:  make(chan bool),
		ctx:          ctx,
		cancel:       cancel,
		handleSigint: handleSigint,
		msg:          NewLineMessage("", maxWidth),
	}

	spacer := 6

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

	if s.handleSigint {
		s.signalChannel = make(chan os.Signal, 2)
		signal.Notify(s.signalChannel, util.ShutdownSignals...)
	}

	return s, nil
}

type SpinnerStopReason int

const (
	StopSuccess SpinnerStopReason = iota
	StopFailed
	StopCancel
)

// Done stops the spinner, ignoring any errors as there
// is no further use for the spinner.
func (s *Spinner) Done(reason SpinnerStopReason) {
	s.complete = true

	stop := s.spin.Stop
	switch reason {
	case StopSuccess:
		s.spin.StopMessage(s.msg.PrintOnDone())
	case StopFailed:
		stop = s.spin.StopFail
	}

	_ = stop()
	s.cancel()
}

// NextStep completes the current line (if any) and
// progresses to the next line, starting a new timer.
func (s *Spinner) NextStep(line string) {
	if s.complete {
		return
	}

	// Stop the spinner and wait until it is stopped
	if s.spin.Status() == yacspin.SpinnerRunning {
		s.spin.StopMessage(s.msg.PrintOnDone())
		s.actionChannel <- SpinActionStop
		<-s.doneChannel
	}
	s.msgMutex.Lock()

	s.msg = NewLineMessage(line, s.maxWidth)
	s.spin.Prefix(s.msg.SpinnerPrefix() + SpacerText)
	s.spin.Suffix(ReverseText)
	s.msgMutex.Unlock()

	s.updateText(time.Duration(0) * time.Millisecond)

	// Start the spinner and wait until it is started if we have
	// not just displayed an error state.
	s.actionChannel <- SpinActionStart
	<-s.doneChannel
}

func (s *Spinner) updateText(duration time.Duration) {
	s.msg.TimerString = spinnerFmtDuration(duration)
	s.spin.Message(s.msg.SpinnerMessage())
	s.spin.StopMessage(s.msg.PrintOnCancel())
	s.spin.StopFailMessage(s.msg.PrintOnFail())
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
						log.Ctx(s.ctx).Err(err).Msg("failed to start spinner")
					}

					s.doneChannel <- true
				case SpinActionStop:
					s.ticker.Stop()

					err := s.spin.Stop()
					if err != nil {
						log.Ctx(s.ctx).Err(err).Msg("failed to stop spinner")
					}
					s.doneChannel <- true
				}
			case signal := <-s.signalChannel:
				log.Ctx(s.ctx).Debug().Msgf("Captured %v. Exiting...", s)
				if signal == os.Interrupt {
					s.ticker.Stop()
					s.Done(StopCancel)
					os.Exit(0)
				}
			}
		}
	}()
}

const (
	StatusNone = "       "
	StatusDone = "done ✅"
	StatusWait = "wait ⏳"
	StatusErr  = "err  ❌"
)

const (
	WidthDots   = 16
	WidthStatus = 6
	// Don't left pad the timer column because we want it to be left aligned.
	WidthTimer = 0
)

const (
	maxUint8Value = 255
	minUint8Value = 0
)

type LineMessage struct {
	Message      string
	Detail       string
	TimerString  string
	Waiting      bool
	Failure      bool
	ColumnWidths []uint8
}

func NewLineMessage(msg string, maxWidth int) LineMessage {
	safeWidth := maxWidth
	if maxWidth > maxUint8Value {
		safeWidth = maxUint8Value // Cap at maximum uint8 value
	} else if maxWidth < minUint8Value {
		safeWidth = minUint8Value // Handle negative values
	}

	//nolint:gosec    // Safe uint8 conversion - value is bounded by maxUint8Value check above
	return LineMessage{
		Message:      msg,
		TimerString:  spinnerFmtDuration(SpinnerFormatDurationDefault),
		ColumnWidths: []uint8{uint8(safeWidth), WidthDots, WidthStatus, WidthTimer},
	}
}

func (f *LineMessage) BlankSpinner(col int) string {
	return strings.Repeat(".", int(f.ColumnWidths[col]))
}

func (f *LineMessage) PrintOnDone() string {
	return "\t" + formatLineMessage(
		f.ColumnWidths[0:4],
		f.Message,
		f.BlankSpinner(1),
		StatusDone,
		f.TimerString,
	)
}

func (f *LineMessage) PrintOnFail() string {
	return "\t" + formatLineMessage(
		f.ColumnWidths[0:4],
		f.Message,
		f.BlankSpinner(1),
		StatusErr,
		f.TimerString,
	)
}

func (f *LineMessage) PrintOnCancel() string {
	status := StatusNone
	detail := f.TimerString
	if f.Waiting {
		status = StatusWait
		detail = f.Detail
	}

	return "\t" + formatLineMessage(
		f.ColumnWidths[0:4],
		f.Message,
		f.BlankSpinner(1),
		status,
		detail,
	)
}

func (f *LineMessage) SpinnerPrefix() string {
	return "\t" + formatLineMessage(
		f.ColumnWidths[0:1],
		f.Message,
	)
}

func (f *LineMessage) SpinnerMessage() string {
	status := StatusNone
	detail := f.TimerString
	if f.Waiting {
		status = StatusWait
		detail = f.Detail
	}

	return formatLineMessage(
		f.ColumnWidths[2:4],
		status,
		detail,
	)
}

func formatLineMessage(widths []uint8, parts ...string) string {
	paddedParts := make([]string, 0, len(parts))
	for i, part := range parts {
		amountToPad := 0
		if i < len(widths) { // Don't pad last column
			amountToPad = math.Max(int(widths[i])-len(part), 0)
		}
		paddedParts = append(paddedParts, strings.Repeat(" ", amountToPad)+part)
	}
	return strings.Join(paddedParts, "  ")
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
