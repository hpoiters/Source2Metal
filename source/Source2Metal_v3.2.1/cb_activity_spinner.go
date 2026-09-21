package main

import (
	"strings"
	"sync"
	"time"
)

// cbActivitySpinner owns only the ChessBase adapter's display. It runs on a
// clock, independently of the decoder's progress callbacks, so a slow game
// still has a visibly rotating indicator. The decoder and its output are
// untouched. Stop always waits for the final console write before the caller
// clears or finishes the shared progress line.
type cbActivitySpinner struct {
	renderer *progressRenderer
	mu       sync.Mutex
	label    string
	done     int64
	total    int64
	start    time.Time
	frame    int
	stop     chan struct{}
	stopped  chan struct{}
	once     sync.Once
}

func newCBActivitySpinner(renderer *progressRenderer, label string, start time.Time) *cbActivitySpinner {
	s := &cbActivitySpinner{
		renderer: renderer,
		label:    label,
		start:    start,
		stop:     make(chan struct{}),
		stopped:  make(chan struct{}),
	}
	s.render() // Activity is visible before the first record has completed.
	go func() {
		defer close(s.stopped)
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-s.stop:
				return
			case <-ticker.C:
				s.render()
			}
		}
	}()
	return s
}

func (s *cbActivitySpinner) Update(label string, done, total int64, start time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.label, s.done, s.total, s.start = label, done, total, start
}

func (s *cbActivitySpinner) render() {
	s.mu.Lock()
	label, done, total, start, frame := s.label, s.done, s.total, s.start, s.frame
	s.frame = (s.frame + 1) % 4
	s.mu.Unlock()
	s.renderer.Render(func(width int) string {
		return cbSpinnerProgressLine(label, done, total, start, width, "|/-\\"[frame])
	})
}

func (s *cbActivitySpinner) Stop() {
	s.once.Do(func() { close(s.stop) })
	<-s.stopped
}

// Reserve two extra columns for the rotating character without changing the
// established progress bar, counts or ETA formatting. The normal completion
// state uses cbAdapterProgressLine directly and has no spinner.
func cbSpinnerProgressLine(label string, done, total int64, start time.Time, width int, frame byte) string {
	if width < 4 {
		return cbAdapterProgressLine(label, done, total, start, width)
	}
	line := cbAdapterProgressLine(label, done, total, start, width-2)
	return "  " + string(frame) + " " + strings.TrimPrefix(line, "  ")
}
