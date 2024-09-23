package printer

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// ANSI escape codes for cursor control
const (
	hideCursor = "\033[?25l"
	showCursor = "\033[?25h"
)

// FishSpinner represents a simple fish emoji spinner
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
		width:       23, // Width of the animation area
		stopChan:    make(chan struct{}),
		writer:      writer,
		spinnerText: "  Waiting for events  ",
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

// Pause temporarily stops the spinner animation to allow for printing
func (s *FishSpinner) Pause() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.paused = true
}

// Resume restarts the spinner animation
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
	ticker := time.NewTicker(200 * time.Millisecond)
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

	fmt.Fprintf(s.writer, "\r%s %s", s.spinnerText, animation)
}

// Clear removes the spinner from the console
func (s *FishSpinner) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	fmt.Fprint(s.writer, "\r"+strings.Repeat(" ", len(s.spinnerText)+s.width+2)+"\r")
}

// hideCursor hides the cursor
func (s *FishSpinner) hideCursor() {
	fmt.Fprint(s.writer, hideCursor)
}

// showCursor shows the cursor
func (s *FishSpinner) showCursor() {
	fmt.Fprint(s.writer, showCursor)
}
