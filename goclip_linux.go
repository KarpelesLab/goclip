package goclip

// #cgo pkg-config: x11
// #include <X11/Xlib.h>
import "C"

import "context"

type linuxStatus struct {
	dpy *C.Display
}

var status linuxStatus

// SetX11Display is to be used if your application is already making use of X11
// and has a connection to the server.
// For example if using glfw, add the following line in a Linux only file:
// goclip.SetX11Display(glfw.GetX11Display())
func SetX11Display(dpy *C.Display) {
	status.dpy = dpy
}

func open(ctx context.Context) error {
	dpy := C.XOpenDisplay(nil)
	_ = dpy

	return nil
}
