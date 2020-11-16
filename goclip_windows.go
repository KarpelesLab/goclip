//+build windows
package goclip

import (
	"context"
	"syscall"
	"time"
)

// Windows
// https://docs.microsoft.com/en-us/windows/win32/dataxchg/using-the-clipboard

const (
	// https://docs.microsoft.com/en-us/windows/win32/dataxchg/standard-clipboard-formats
	cfBitmap      = 2
	cfTiff        = 6
	cfUnicodeText = 13
	cfHdrop       = 15
)

var (
	// imported APIs
	user32               = syscall.MustLoadDLL("user32")
	openClipboard        = user32.MustFindProc("OpenClipboard")
	closeClipboard       = user32.MustFindProc("CloseClipboard")
	emptyClipboard       = user32.MustFindProc("EmptyClipboard")
	getClipboardData     = user32.MustFindProc("GetClipboardData")
	setClipboardData     = user32.MustFindProc("SetClipboardData")
	enumClipboardFormats = user32.MustFindProc("EnumClipboardFormats")
	shell32              = syscall.NewLazyDLL("shell32")
	dragQueryFile        = shell32.NewProc("DragQueryFileW")

	kernel32     = syscall.NewLazyDLL("kernel32")
	globalAlloc  = kernel32.NewProc("GlobalAlloc")
	globalFree   = kernel32.NewProc("GlobalFree")
	globalLock   = kernel32.NewProc("GlobalLock")
	globalUnlock = kernel32.NewProc("GlobalUnlock")
	lstrcpy      = kernel32.NewProc("lstrcpyW")
)

func open(ctx context.Context) error {
	var r uintptr
	var err error
	var t *time.Ticker

	for {
		r, _, err = openClipboard.Call(0)
		if r != 0 {
			// success
			return nil
		}

		if t == nil {
			t = time.NewTicker(5 * time.Millisecond)
			defer t.Stop()
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
		}
	}
	return err
}

func clear(ctx context.Context) error {
	// perform clipboard clear
	if err := open(ctx); err != nil {
		return err
	}

	defer closeClipboard.Call()

	emptyClipboard.Call()
	return nil
}

func formats() []uint32 {
	// note: requires clipboard to be already open
	var res []uint32
	var fmt uintptr
	var err error

	for {
		fmt, _, err = enumClipboardFormats.Call(fmt)
		if fmt == 0 || err != nil {
			break
		}
		res = append(res, uint32(fmt))
	}
	return res
}
