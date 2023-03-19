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
	"errors"
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
	image, err := png.Decode(bytes.NewReader(cb.data))
	if err != nil {
		panic(err) // debug for now
	}
	return image, nil
}

func (cb macOSClipboard) HasFormat(fmt string) bool {
	//TODO implement me
	panic("implement me")
}

func (cb macOSClipboard) GetFormat(ctx context.Context, fmt string) ([]byte, error) {
	//TODO implement me
	return nil, errors.New("unsupported method (TODO)")
}

func (cb macOSClipboard) GetAllFormats() ([]DataOption, error) {
	//TODO implement me
	return nil, errors.New("unsupported method (TODO)")
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

	C.readClipboard(i.sub, filter)

	return cb.processRead()
}

// processRead will handle data that was freshly read from the clipboard
func (cb *macOSClipboard) processRead() error {
	dataType := Type(cb.i.sub.cbi.typeInt)
	if ok := isValidType(dataType); !ok {
		err := fmt.Errorf("goclip: could not find clipboard Type for %d", dataType)
		return err
	}

	dataLength := C.int(i.sub.cb.dataLength)
	dataBytes := C.GoBytes(unsafe.Pointer(i.sub.cb.data), dataLength)

	if dataType == Image {
		if i.sub.cbi.formatTypeInt == C.CLIPBOARD_FORMAT_IMAGE_TIFF {
			image, err := tiff.Decode(bytes.NewReader(dataBytes))
			if err != nil {
				return ErrTiffImageDecode
			}
			buf := new(bytes.Buffer)
			png.Encode(buf, image)
			i.sub.cbi.formatTypeInt = C.CLIPBOARD_FORMAT_IMAGE_PNG
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
