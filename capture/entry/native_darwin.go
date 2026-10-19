//go:build darwin && cgo

package entry

/*
#cgo CFLAGS: -x objective-c -fblocks
#cgo LDFLAGS: -framework Cocoa -framework Carbon
#include "native_darwin.h"
*/
import "C"

import (
	"fmt"
	"runtime/cgo"
	"unsafe"
)

type platformNative struct {
	ptr      unsafe.Pointer
	callback cgo.Handle
}

func NewNative() Native                 { return &platformNative{} }
func (*platformNative) Supported() bool { return true }
func (n *platformNative) Start(callback func()) error {
	if n.ptr != nil {
		return nil
	}
	n.callback = cgo.NewHandle(callback)
	var status C.int32_t
	n.ptr = C.nil_entry_start(C.uintptr_t(n.callback), &status)
	if n.ptr == nil {
		n.callback.Delete()
		n.callback = 0
		return fmt.Errorf("macOS capture menu unavailable (status %d)", int32(status))
	}
	return nil
}
func (n *platformNative) Configure(s Shortcut) error {
	if n.ptr == nil {
		return fmt.Errorf("macOS capture entry is not started")
	}
	key, modifiers := s.Carbon()
	enabled := 0
	if s.Enabled {
		enabled = 1
	}
	status := C.nil_entry_configure(n.ptr, C.int(enabled), C.uint32_t(key), C.uint32_t(modifiers))
	if status != 0 {
		return fmt.Errorf("macOS could not register shortcut (status %d); it may belong to another app; previous shortcut retained", int32(status))
	}
	return nil
}
func (n *platformNative) Close() {
	if n.ptr == nil {
		return
	}
	C.nil_entry_close(n.ptr)
	n.ptr = nil
	n.callback.Delete()
	n.callback = 0
}

//export nilEntryRequested
func nilEntryRequested(handle C.uintptr_t) {
	// Never block AppKit's main thread on Go/Wails or controller locks. The native
	// event handler is removed synchronously before its cgo handle is deleted.
	callback := cgo.Handle(handle).Value().(func())
	go callback()
}
