package goclip

import "errors"

// Monitor returns a new clipboard monitor that can capture events from the
// clipboard based on various rules.
type Monitor struct {
	C <-chan struct{}
}

func NewMonitor() (*Monitor, error) {
	return nil, errors.New("TODO")
}

// Poll should be called when the app regains focus for example, and will check
// if any change happened to the clipboard.
func (m *Monitor) Poll() {
	// TODO
}
