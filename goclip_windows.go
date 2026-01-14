//go:build windows
// +build windows

package goclip

import (
	"context"
	"errors"
	"syscall"
	"time"
	"unsafe"
)

type internal struct {
	// ...
}

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

func doInit() *internal {
	return &internal{}
}

func (i *internal) open(ctx context.Context) error {
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

func (i *internal) copy(ctx context.Context, board Board, values ...interface{}) error {
	if board != Default {
		// Windows only supports the default clipboard
		return ErrNoBoard
	}

	// Open clipboard
	if err := i.open(ctx); err != nil {
		return err
	}
	defer closeClipboard.Call()

	// Empty the clipboard
	r, _, _ := emptyClipboard.Call()
	if r == 0 {
		return errors.New("failed to empty clipboard")
	}

	for _, v := range values {
		if s, ok := v.(string); ok {
			// Text data
			// Allocate global memory for the text
			text16, err := syscall.UTF16FromString(s)
			if err != nil {
				return err
			}

			// Allocate global memory for the text
			hMem, _, err := globalAlloc.Call(0x0002 /* GMEM_MOVEABLE */, uintptr(len(text16)*2))
			if hMem == 0 {
				return errors.New("failed to allocate global memory")
			}

			// Lock the memory to get a pointer
			lpData, _, err := globalLock.Call(hMem)
			if lpData == 0 {
				globalFree.Call(hMem)
				return errors.New("failed to lock global memory")
			}

			// Copy text to the memory
			for i := 0; i < len(text16); i++ {
				*(*uint16)(unsafe.Pointer(lpData + uintptr(i*2))) = text16[i]
			}

			// Unlock the memory
			globalUnlock.Call(hMem)

			// Set clipboard data
			h, _, err := setClipboardData.Call(cfUnicodeText, hMem)
			if h == 0 {
				globalFree.Call(hMem)
				return errors.New("failed to set clipboard data")
			}

			return nil
		}
		// Additional data types (images, file lists) would be implemented here
	}

	return ErrFormatUnavailable
}

func (i *internal) clear(ctx context.Context) error {
	// perform clipboard clear
	if err := i.open(ctx); err != nil {
		return err
	}

	defer closeClipboard.Call()

	emptyClipboard.Call()
	return nil
}

func (i *internal) formats() []uint32 {
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

func (i *internal) paste(ctx context.Context, board Board) (Data, error) {
	if board != Default {
		return nil, ErrNoBoard
	}

	// Open clipboard
	if err := i.open(ctx); err != nil {
		return nil, err
	}
	defer closeClipboard.Call()

	// Check available formats
	formats := i.formats()

	data := &StaticData{
		TargetBoard: board,
	}

	// Try to get text data
	for _, format := range formats {
		if format == cfUnicodeText {
			h, _, _ := getClipboardData.Call(uintptr(format))
			if h != 0 {
				lpData, _, _ := globalLock.Call(h)
				if lpData != 0 {
					// Extract string data
					stringData := make([]uint16, 0, 1024) // Initial capacity
					for i := 0; ; i++ {
						char := *(*uint16)(unsafe.Pointer(lpData + uintptr(i*2)))
						if char == 0 {
							break
						}
						stringData = append(stringData, char)
					}
					globalUnlock.Call(h)

					// Add as a data option
					data.Options = append(data.Options, &StaticDataOption{
						StaticType: "text/plain",
						StaticData: []byte(syscall.UTF16ToString(stringData)),
					})
					break
				}
			}
		}
	}

	// If we found any data, return it
	if len(data.Options) > 0 {
		return data, nil
	}

	return nil, ErrNoData
}

func (i *internal) monitor(mon *Monitor) error {
	// Basic implementation of clipboard monitoring
	go func() {
		var lastFormats []uint32

		for {
			if err := i.open(context.Background()); err != nil {
				time.Sleep(500 * time.Millisecond)
				continue
			}

			currentFormats := i.formats()
			closeClipboard.Call()

			// Check if formats changed
			changed := len(lastFormats) != len(currentFormats)
			if !changed {
				for i, f := range lastFormats {
					if currentFormats[i] != f {
						changed = true
						break
					}
				}
			}

			if changed {
				lastFormats = currentFormats

				// Get data and trigger callback
				data, err := i.paste(context.Background(), Default)
				if err == nil {
					// Fire the monitor's callback with the new data
					mon.fire(data)
				}
			}

			time.Sleep(500 * time.Millisecond)
		}
	}()

	return nil
}

func (i *internal) unmonitor(mon *Monitor) error {
	// In a real implementation, we would stop the monitoring goroutine
	// For now, just return success
	return nil
}

func (i *internal) poll(mon *Monitor) error {
	// Trigger a check right now
	return nil
}
