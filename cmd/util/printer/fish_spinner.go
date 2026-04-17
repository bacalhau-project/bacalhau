package printer

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/bacalhau-project/bacalhau/cmd/util/output"
)

// ANSI escape codes for cursor control
const (
	hideCursor     = "\033[?25l"
	showCursor     = "\033[?25h"
	tickerInterval = 200 * time.Millisecond
)

// FishSpinner represents a simple fish emoji spinner.
// It provides a visual indicator of ongoing processing while allowing
// concurrent event printing.
//
// The FishSpinner operates in its own goroutine, continuously updating
// and printing the spinner animation. However, it's designed to work
// in conjunction with an event printer, which may need to write to the
// console periodically.
//
// To prevent the spinner and event printer from writing to the console
// simultaneously and causing overlapping output, the FishSpinner implements
// a synchronization mechanism:
//
//  1. The event printer should call Pause() before printing a new event row.
//     This pauses the spinner animation
//  2. The event printer then calls clear() to remove the spinner from the console
//  2. The event printer then prints its event row.
//  3. After printing, the event printer should call Resume() to restart
//     the spinner animation from where it left before pausing.
//
// This synchronization ensures that the spinner and event printer's outputs
// don't interfere with each other, maintaining clean and readable console output.
type FishSpinner struct {
	frames      []string
	index       int
	position    int
	width       int
	stopChan    chan struct{}
	writer      io.Writer
	spinnerText string
	mu          sync.Mutex
	paused      bool
}

// NewFishSpinner creates a new FishSpinner
func NewFishSpinner(writer io.Writer) *FishSpinner {
	return &FishSpinner{
		frames:      []string{"🐟", "🐠", "🐡"},
		index:       0,
		position:    0,
		width:       21, // Width of the animation area
		stopChan:    make(chan struct{}),
		writer:      writer,
		spinnerText: output.ItalicStr(" Processing   "),
	}
}

// Start begins the spinner animation
func (s *FishSpinner) Start() {
	s.hideCursor()
	go s.run()
}

// Stop stops the spinner animation
func (s *FishSpinner) Stop() {
	close(s.stopChan)
	s.showCursor()
}

// Pause temporarily stops the spinner animation and clears its output
// This function is called before the event printer wants to print a new row
func (s *FishSpinner) Pause() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.paused = true
}

// Resume restarts the spinner animation after it has been paused
// This function is called after the event printer has finished printing
func (s *FishSpinner) Resume() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.paused {
		s.paused = false
		s.print() // Print immediately when resuming
	}
}

// run continuously updates the spinner animation
func (s *FishSpinner) run() {
	ticker := time.NewTicker(tickerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			if !s.paused {
				s.update()
				s.print()
			}
			s.mu.Unlock()
		case <-s.stopChan:
			return
		}
	}
}

// update advances the spinner's state
func (s *FishSpinner) update() {
	s.position++
	if s.position >= s.width {
		s.position = 0
		s.index = (s.index + 1) % len(s.frames)
	}
}

// print displays the current frame of the spinner
func (s *FishSpinner) print() {
	frame := s.frames[s.index]
	dots := strings.Repeat(".", s.width)
	animation := dots[:s.position] + frame + dots[s.position+1:]

	_, _ = fmt.Fprintf(s.writer, "\r%s %s", s.spinnerText, animation)
}

// Clear removes the spinner from the console
func (s *FishSpinner) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = fmt.Fprint(s.writer, "\r"+strings.Repeat(" ", len(s.spinnerText)+s.width+2)+"\r")
}

// hideCursor hides the cursor
func (s *FishSpinner) hideCursor() {
	_, _ = fmt.Fprint(s.writer, hideCursor)
}

// showCursor shows the cursor
func (s *FishSpinner) showCursor() {
	_, _ = fmt.Fprint(s.writer, showCursor)
}
