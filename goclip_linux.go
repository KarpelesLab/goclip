package goclip

// #cgo pkg-config: x11
// #include <X11/Xlib.h>
import "C"

import "context"

func open(ctx context.Context) error {
	dpy := C.XOpenDisplay(nil)
	_ = dpy

	return nil
}
