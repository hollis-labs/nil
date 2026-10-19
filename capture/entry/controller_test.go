package entry

import (
	"errors"
	"sync"
	"testing"
)

type fakeNative struct {
	calls    []Shortcut
	callback func()
	fail     bool
	closed   int
}

func (*fakeNative) Supported() bool               { return true }
func (f *fakeNative) Start(callback func()) error { f.callback = callback; return nil }
func (f *fakeNative) Configure(s Shortcut) error {
	if f.fail {
		return errors.New("shortcut already registered")
	}
	f.calls = append(f.calls, s)
	return nil
}
func (f *fakeNative) Close() { f.closed++ }
func TestDefaultDoesNotRegisterAndPendingRequestsCoalesce(t *testing.T) {
	native := &fakeNative{}
	invoked := 0
	c := New(native, func(Shortcut) error { return nil }, func() { invoked++ })
	c.Start(Default())
	if len(native.calls) != 0 || c.Status().Registered {
		t.Fatal("default stole existing shortcut")
	}
	native.callback()
	native.callback()
	if invoked != 0 {
		t.Fatal("invoked before renderer listener ready")
	}
	c.Ready()
	if invoked != 1 {
		t.Fatal("pending requests lost or duplicated")
	}
	c.Request()
	if invoked != 2 {
		t.Fatal("ready invocation lost")
	}
	c.Close()
	native.callback()
	if invoked != 2 || native.closed != 1 {
		t.Fatal("late callback after shutdown")
	}
}
func TestShortcutConflictAndPersistenceRollback(t *testing.T) {
	native := &fakeNative{}
	persistFail := false
	saved := Default()
	c := New(native, func(s Shortcut) error {
		if persistFail {
			return errors.New("disk")
		}
		saved = s
		return nil
	}, func() {})
	c.Start(Default())
	s := Default()
	s.Enabled = true
	status, err := c.Apply(s)
	if err != nil || !status.Registered || !saved.Enabled {
		t.Fatal(status, err)
	}
	native.fail = true
	other := s
	other.Key = "K"
	status, err = c.Apply(other)
	if err == nil || status.Shortcut.Key != "I" || saved.Key != "I" || !status.Registered {
		t.Fatal("conflict displaced previous registration")
	}
	native.fail = false
	persistFail = true
	status, err = c.Apply(other)
	if err == nil || status.Shortcut.Key != "I" || native.calls[len(native.calls)-1].Key != "I" {
		t.Fatal("failed persistence lost original shortcut")
	}
	persistFail = false
	s.Enabled = false
	status, err = c.Apply(s)
	if err != nil || status.Registered {
		t.Fatal("explicit disable failed")
	}
	c.Close()
	c.Close()
	if native.closed != 1 {
		t.Fatal("shutdown not idempotent")
	}
}
func TestCarbonDefaultAndValidation(t *testing.T) {
	key, mask := Default().Carbon()
	if key != 34 || mask != 256 {
		t.Fatal("default differs from Command-I")
	}
	s := Default()
	s.Key = "k"
	s.Shift = true
	normalized, err := Normalize(s)
	if err != nil || normalized.Key != "K" {
		t.Fatal(err)
	}
	key, mask = normalized.Carbon()
	if key != 40 || mask != 768 {
		t.Fatal(key, mask)
	}
	s.Command = false
	if _, err = Normalize(s); err == nil {
		t.Fatal("unmodified capture shortcut accepted")
	}
}

func TestReadinessBeforeStartAndConcurrentShutdown(t *testing.T) {
	native := &fakeNative{}
	count := 0
	c := New(native, func(Shortcut) error { return nil }, func() { count++ })
	c.Ready()
	c.Request()
	c.Request()
	if count != 0 {
		t.Fatal("invoked before native startup")
	}
	c.Start(Default())
	if count != 1 {
		t.Fatal("startup dropped pending invocation")
	}
	var workers sync.WaitGroup
	for range 50 {
		workers.Add(1)
		go func() { defer workers.Done(); c.Request() }()
	}
	workers.Add(1)
	go func() { defer workers.Done(); c.Close() }()
	workers.Wait()
	final := count
	c.Request()
	if count != final || native.closed != 1 {
		t.Fatal("request escaped shutdown fencing")
	}
}
