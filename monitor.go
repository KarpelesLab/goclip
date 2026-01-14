package goclip

// MonitorCallback is a function triggered by the monitor in case of event
// happening. If there are multiple callbacks but one returns an error, the
// following callbacks won't be called and the error may be returned or
// ignored.
type MonitorCallback func(Data) error

// Monitor returns a new clipboard monitor that can capture events from the
// clipboard based on various rules.
type Monitor struct {
	cb []MonitorCallback
}

func NewMonitor() (*Monitor, error) {
	mon := &Monitor{}
	err := i.monitor(mon)
	if err != nil {
		return nil, err
	}
	return mon, nil
}

func (m *Monitor) Subscribe(cb MonitorCallback) {
	m.cb = append(m.cb, cb)
}

func (m *Monitor) fire(ev Data) error {
	// call all callbacks
	for _, cb := range m.cb {
		err := cb(ev)
		if err != nil {
			return err
		}
	}
	return nil
}

// Poll should be called when the app regains focus for example, and will check
// if any change happened to the clipboard.
func (m *Monitor) Poll() error {
	return i.poll(m)
}

func (m *Monitor) Close() error {
	return i.unmonitor(m)
}
