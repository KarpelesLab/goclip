package goclip

// https://developer.apple.com/documentation/appkit/nspasteboard

/*
#include <goclip_darwin.h>

#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
*/
import "C"
import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"sync"
	"unsafe"

	"golang.org/x/image/tiff"
)

type internal struct {
	sub      *C.ClipboardInternal
	mon      []*Monitor
	startMon sync.Once
	pollch   chan struct{}
}

type macOSClipboard struct {
	i        *internal
	dataType Type
	data     []byte
}

func (cb macOSClipboard) Board() Board {
	// the only possible board
	return Default
}

func (cb macOSClipboard) ToText(ctx context.Context) (string, error) {
	if cb.Type() == Invalid {
		// perform read
		err := cb.performRead(Text)
		if err != nil {
			return "", err
		}
	}
	if cb.Type() != Text {
		return "", ErrDataNotString
	}
	return string(cb.data), nil
}

func (cb macOSClipboard) ToImage(ctx context.Context) (image.Image, error) {
	if cb.Type() == Invalid {
		// perform read
		err := cb.performRead(Image)
		if err != nil {
			return nil, err
		}
	}
	if cb.Type() != Image {
		return nil, ErrDataNotImage
	}
	img, err := png.Decode(bytes.NewReader(cb.data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode PNG image: %w", err)
	}
	return img, nil
}

func (cb macOSClipboard) HasFormat(fmt string) bool {
	// Basic implementation that checks if we have data in the format corresponding to fmt
	switch fmt {
	case "text/plain":
		return cb.Type() == Text
	case "image/png":
		return cb.Type() == Image
	case "text/uri-list":
		return cb.Type() == FileList
	default:
		return false
	}
}

func (cb macOSClipboard) GetFormat(ctx context.Context, fmt string) ([]byte, error) {
	// If we don't have data yet, try to read it first
	if cb.Type() == Invalid {
		if fmt == "text/plain" {
			err := cb.performRead(Text)
			if err != nil {
				return nil, err
			}
		} else if fmt == "image/png" {
			err := cb.performRead(Image)
			if err != nil {
				return nil, err
			}
		} else if fmt == "text/uri-list" {
			err := cb.performRead(FileList)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, os.ErrNotExist
		}
	}

	// Check if the format matches what we have
	switch fmt {
	case "text/plain":
		if cb.Type() == Text {
			return cb.data, nil
		}
	case "image/png":
		if cb.Type() == Image {
			return cb.data, nil
		}
	case "text/uri-list":
		if cb.Type() == FileList {
			return cb.data, nil
		}
	}

	return nil, os.ErrNotExist
}

func (cb macOSClipboard) GetAllFormats() ([]DataOption, error) {
	// If we don't have data yet, try to read it
	if cb.Type() == Invalid {
		// Try to read all types
		err := cb.performRead(Text, Image, FileList)
		if err != nil {
			return nil, err
		}
	}

	// Return the available format as a DataOption
	var options []DataOption

	switch cb.Type() {
	case Text:
		options = append(options, &StaticDataOption{
			StaticType: "text/plain",
			StaticData: cb.data,
		})
	case Image:
		options = append(options, &StaticDataOption{
			StaticType: "image/png",
			StaticData: cb.data,
		})
	case FileList:
		options = append(options, &StaticDataOption{
			StaticType: "text/uri-list",
			StaticData: cb.data,
		})
	}

	return options, nil
}

func (cb *macOSClipboard) Type() Type {
	return cb.dataType
}

func (cb *macOSClipboard) FileList() ([]string, error) {
	if cb.Type() != FileList {
		return nil, ErrDataNotFileList
	}
	return nil, nil
}

func (cb *macOSClipboard) performRead(types ...Type) error {
	filter := &C.ClipboardTypeFilter{}
	for _, e := range types {
		switch e {
		case Text:
			filter.text = true
		case Image:
			filter.image = true
		case FileList:
			filter.files = true
		}
	}

	C.readClipboard(cb.i.sub, filter)

	return cb.processRead()
}

// processRead will handle data that was freshly read from the clipboard
func (cb *macOSClipboard) processRead() error {
	dataType := Type(cb.i.sub.cbi.typeInt)
	switch dataType {
	case Text, Image, FileList:
		// valid type
	default:
		return fmt.Errorf("goclip: could not find clipboard Type for %d", dataType)
	}

	dataLength := C.int(cb.i.sub.cb.dataLength)
	dataBytes := C.GoBytes(unsafe.Pointer(cb.i.sub.cb.data), dataLength)

	if dataType == Image {
		if cb.i.sub.cbi.formatTypeInt == C.CLIPBOARD_FORMAT_IMAGE_TIFF {
			img, err := tiff.Decode(bytes.NewReader(dataBytes))
			if err != nil {
				return ErrTiffImageDecode
			}
			buf := new(bytes.Buffer)
			png.Encode(buf, img)
			cb.i.sub.cbi.formatTypeInt = C.CLIPBOARD_FORMAT_IMAGE_PNG
			dataBytes = buf.Bytes()
		}
	}

	cb.dataType = dataType
	cb.data = dataBytes
	return nil
}

func doInit() *internal {
	log.Printf("goclip: [darwin] opening general pasteboard")
	sub := C.cocoaPbFactory()
	return &internal{sub: sub, pollch: make(chan struct{})}
}

func (i *internal) copy(ctx context.Context, board Board, values ...interface{}) error {
	if board != Default {
		// only default board on macos
		return ErrNoBoard
	}
	for _, v := range values {
		if s, ok := v.(string); ok {
			// ok that's text
			log.Printf("goclip: set text to %s", s)
			C.pasteWriteAddText(C.CString(s), C.int(len(s)))
			C.pasteWrite(i.sub)
			return nil
		}
	}
	return ErrFormatUnavailable
}

func (i *internal) info(ctx context.Context, board Board) (Data, error) {
	if board != Default {
		return nil, ErrNoBoard
	}

	C.readInformation(i.sub)

	return i.spawnData(), nil
}

func (i *internal) paste(ctx context.Context, board Board, types ...Type) (Data, error) {
	if board != Default {
		return nil, ErrNoBoard
	}

	res := i.spawnData()
	return res, res.performRead(types...)
}

func (i *internal) runMonitor() {
	go func() {
		var pos int
		for {
			select {
			case <-i.pollch:
			}
			if i.sub == nil {
				return
			}
			v := int(C.cocoaPbChangeCount(i.sub))
			if v != pos {
				pos = v
				i.triggerData(i.spawnData())
				continue
			}
			//time.Sleep(5 * time.Second)
		}
	}()
}

func (i *internal) monitor(mon *Monitor) error {
	i.startMon.Do(i.runMonitor)
	i.mon = append(i.mon, mon)
	return nil
}

func (i *internal) unmonitor(mon *Monitor) error {
	// locate & remove from i.mon
	for n, v := range i.mon {
		if v == mon {
			i.mon = append(i.mon[:n], i.mon[n+1:]...)
			return nil
		}
	}
	return os.ErrNotExist
}

func (i *internal) triggerData(data Data) {
	for _, m := range i.mon {
		m.fire(data)
	}
}

func (i *internal) spawnData() *macOSClipboard {
	return &macOSClipboard{i: i}
}

func (i *internal) poll(m *Monitor) error {
	select {
	case i.pollch <- struct{}{}:
	default:
	}
	return nil
}
