// Package entry owns capture invocation and shortcut lifecycle independently of
// Wails. Platform callbacks request focus; they never create or submit a draft.
package entry

import (
	"errors"
	"strings"
	"sync"
)

type Shortcut struct {
	Enabled bool   `json:"enabled"`
	Key     string `json:"key"`
	Command bool   `json:"command"`
	Control bool   `json:"control"`
	Option  bool   `json:"option"`
	Shift   bool   `json:"shift"`
}

func Default() Shortcut { return Shortcut{Key: "I", Command: true} }

// Carbon virtual key codes name physical ANSI positions, not translated text.
var keyCodes = map[string]uint32{"A": 0, "S": 1, "D": 2, "F": 3, "H": 4, "G": 5, "Z": 6, "X": 7, "C": 8, "V": 9, "B": 11, "Q": 12, "W": 13, "E": 14, "R": 15, "Y": 16, "T": 17, "1": 18, "2": 19, "3": 20, "4": 21, "6": 22, "5": 23, "9": 25, "7": 26, "8": 28, "0": 29, "O": 31, "U": 32, "I": 34, "P": 35, "L": 37, "J": 38, "K": 40, "N": 45, "M": 46, "Space": 49}

func Normalize(s Shortcut) (Shortcut, error) {
	if strings.EqualFold(s.Key, "space") {
		s.Key = "Space"
	} else {
		s.Key = strings.ToUpper(strings.TrimSpace(s.Key))
	}
	if _, ok := keyCodes[s.Key]; !ok || (!s.Command && !s.Control) {
		return s, errors.New("choose an ANSI letter, digit or Space with Command or Control")
	}
	return s, nil
}
func (s Shortcut) Carbon() (uint32, uint32) {
	var mask uint32
	if s.Command {
		mask |= 256
	}
	if s.Shift {
		mask |= 512
	}
	if s.Option {
		mask |= 2048
	}
	if s.Control {
		mask |= 4096
	}
	return keyCodes[s.Key], mask
}

type Native interface {
	Supported() bool
	Start(func()) error
	Configure(Shortcut) error // Failure must retain the previous registration.
	Close()
}
type Status struct {
	Shortcut   Shortcut `json:"shortcut"`
	Available  bool     `json:"available"`
	Registered bool     `json:"registered"`
	Error      string   `json:"error"`
}
type Controller struct {
	mu                              sync.Mutex
	native                          Native
	save                            func(Shortcut) error
	invoke                          func()
	status                          Status
	started, closed, ready, pending bool
}

func New(native Native, save func(Shortcut) error, invoke func()) *Controller {
	return &Controller{native: native, save: save, invoke: invoke, status: Status{Shortcut: Default()}}
}
func (c *Controller) Start(s Shortcut) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.started {
		return
	}
	c.started = true
	defer func() {
		if c.pending && c.ready {
			c.pending = false
			c.invoke()
		}
	}()
	normalized, err := Normalize(s)
	if err != nil {
		c.status.Error = err.Error()
		normalized = Default()
	}
	c.status.Shortcut = normalized
	c.status.Available = c.native.Supported()
	if !c.status.Available {
		return
	}
	if err = c.native.Start(c.Request); err != nil {
		c.status.Available = false
		c.status.Error = err.Error()
		return
	}
	if normalized.Enabled {
		if err = c.native.Configure(normalized); err != nil {
			c.status.Error = err.Error()
		} else {
			c.status.Registered = true
		}
	}
}
func (c *Controller) Status() Status { c.mu.Lock(); defer c.mu.Unlock(); return c.status }
func (c *Controller) Apply(s Shortcut) (Status, error) {
	s, err := Normalize(s)
	if err != nil {
		return c.Status(), err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || !c.started {
		return c.status, errors.New("capture entry is not ready")
	}
	if s.Enabled && !c.status.Available {
		return c.status, errors.New("global capture shortcut requires macOS native support")
	}
	old := c.status
	if c.status.Available {
		if err = c.native.Configure(s); err != nil {
			c.status.Error = err.Error()
			return c.status, err
		}
	}
	if err = c.save(s); err != nil {
		if c.status.Available {
			if rollbackErr := c.native.Configure(old.Shortcut); rollbackErr != nil {
				c.native.Close()
				c.status.Available = false
				c.status.Registered = false
				c.status.Error = "shortcut settings could not be saved; native entry stopped after rollback failure"
				return c.status, errors.New(c.status.Error)
			}
		}
		c.status.Error = "shortcut settings could not be saved; previous shortcut retained"
		return c.status, errors.New(c.status.Error)
	}
	c.status.Shortcut = s
	c.status.Registered = s.Enabled && c.status.Available
	c.status.Error = ""
	return c.status, nil
}
func (c *Controller) Request() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	if !c.ready || !c.started {
		c.pending = true
		return
	}
	c.invoke()
}
func (c *Controller) Ready() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.ready = true
	if c.pending && c.started {
		c.pending = false
		c.invoke()
	}
}
func (c *Controller) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	c.pending = false
	c.native.Close()
	c.status.Registered = false
}
