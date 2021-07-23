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
	"time"
	"unsafe"

	"golang.org/x/image/tiff"
)

type internal struct {
	pb  *C.NSPasteboard
	cb  *C.ClipboardData
	cbi *C.ClipboardInformation
}

type macOSClipboard struct {
	dataType Type
	data     []byte
}

func (cb macOSClipboard) Type() Type {
	return cb.dataType
}

func (cb macOSClipboard) String() (string, error) {
	if cb.Type() != Text {
		return "", ErrDataNotString
	}
	return string(cb.data), nil
}

func (cb macOSClipboard) Image() (image.Image, error) {
	if cb.Type() != Image {
		return nil, ErrDataNotImage
	}
	image, err := png.Decode(bytes.NewReader(cb.data))
	if err != nil {
		panic(err) // debug for now
	}
	return image, nil
}

func (cb macOSClipboard) FileList() ([]string, error) {
	if cb.Type() != FileList {
		return nil, ErrDataNotFileList
	}
	return nil, nil
}

// debug for now
// NSString -> C string
func cstring(s *C.NSString) *C.char { return C.nsstring2cstring(s) }

// NSString -> Go string
func gostring(s *C.NSString) string { return C.GoString(cstring(s)) }

func doInit() *internal {
	log.Printf("goclip: opening general pasteboard")
	pb := C.cocoaPbFactory()
	fmt.Printf("%+v\n", pb)
	return &internal{pb: pb, cb: &C.ClipboardData{}, cbi: &C.ClipboardInformation{}}
}

func (i *internal) open(ctx context.Context) error {
	return nil
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
			C.pasteWrite(i.pb)
			return nil
		}
	}
	return ErrFormatUnavailable
}

func (i *internal) info(ctx context.Context, board Board) (ClipboardInformation, error) {
	if board != Default {
		return ClipboardInformation{}, ErrNoBoard
	}

	C.readInformation(i.pb, i.cbi)

	count := int(i.cbi.count)
	typeInt := uint32(i.cbi.typeInt)
	formatTypeInt := uint32(i.cbi.formatTypeInt)

	return ClipboardInformation{
		count:         count,
		clipboardType: Type(typeInt),
		formatType:    FormatType(formatTypeInt),
	}, nil
}

func (i *internal) paste(ctx context.Context, board Board, types ...Type) (macOSClipboard, error) {
	if board != Default {
		return macOSClipboard{}, ErrNoBoard
	}

	filter := &C.ClipboardTypeFilter{text: false, image: false, files: false}
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

	C.readClipboard(i.pb, i.cb, i.cbi, filter)

	typeInt := uint32(i.cbi.typeInt)
	dataType := Type(typeInt)
	if ok := isValidType(dataType); !ok {
		err := fmt.Errorf("goclip: could not find clipboard Type for %d", typeInt)
		return macOSClipboard{}, err
	}

	dataLength := C.int(i.cb.dataLength)
	dataBytes := C.GoBytes(unsafe.Pointer(i.cb.data), dataLength)

	if dataType == Image {
		if i.cbi.formatTypeInt == C.CLIPBOARD_FORMAT_IMAGE_TIFF {
			image, err := tiff.Decode(bytes.NewReader(dataBytes))
			if err != nil {
				return macOSClipboard{}, ErrTiffImageDecode
			}
			buf := new(bytes.Buffer)
			png.Encode(buf, image)
			i.cbi.formatTypeInt = C.CLIPBOARD_FORMAT_IMAGE_PNG
			return macOSClipboard{
				dataType: dataType,
				data:     buf.Bytes(),
			}, nil
		}
	}

	return macOSClipboard{
		dataType: dataType,
		data:     dataBytes,
	}, nil
}

func (i *internal) monitor(c chan struct{}) {
	go func() {
		var pos int
		for {
			if i.pb == nil {
				return
			}
			v := int(C.cocoaPbChangeCount(i.pb))
			if v != pos {
				pos = v
				c <- struct{}{}
				continue
			}
			time.Sleep(5 * time.Second)
		}
	}()
}
