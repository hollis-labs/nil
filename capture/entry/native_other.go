//go:build !darwin || !cgo

package entry

import "errors"

type platformNative struct{}

func NewNative() Native                 { return &platformNative{} }
func (*platformNative) Supported() bool { return false }
func (*platformNative) Start(func()) error {
	return errors.New("macOS native capture entry unavailable")
}
func (*platformNative) Configure(Shortcut) error {
	return errors.New("macOS native capture entry unavailable")
}
func (*platformNative) Close() {}
