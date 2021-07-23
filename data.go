package goclip

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"os"
	"strings"
)

type Data interface {
	Type() Type
	Board() Board
	ToText(ctx context.Context) (string, error)
	ToImage(ctx context.Context) (image.Image, error)
	FileList() ([]string, error)

	// direct format accessors using MIME formats
	HasFormat(fmt string) bool
	GetFormat(ctx context.Context, fmt string) ([]byte, error)
	GetAllFormats() ([]DataOption, error)
}

type StaticDataOption struct {
	StaticType string // such as "image/png" or "text/plain", or "text/plain;charset=utf-8"
	StaticData []byte // actual data
}

func (s *StaticDataOption) Type() Type {
	return simpleTypeFromMime(s.StaticType)
}

func simpleTypeFromMime(mime string) Type {
	// very simplistic but should work in most cases
	ppos := strings.IndexByte(mime, '/')
	if ppos == -1 {
		return Invalid
	}
	switch strings.ToLower(mime[:ppos]) {
	case "image":
		return Image
	case "text":
		return Text
	default:
		return Invalid
	}
}

func (s *StaticDataOption) Mime() string {
	return s.StaticType
}

func (s *StaticDataOption) Data(ctx context.Context) ([]byte, error) {
	return s.StaticData, nil
}

// DataOption is a simple option from within a list of options
type DataOption interface {
	Type() Type
	Mime() string
	Data(ctx context.Context) ([]byte, error)
}

// StaticData is a type of data used to represent a whole clipboard, including
// multiple formats as made available by the system. Options can either contain
// instances of StaticDataOption, or objects following the DataOption interface
type StaticData struct {
	TargetBoard Board
	Options     []DataOption
}

func (s *StaticData) Type() Type {
	if len(s.Options) == 0 {
		return Invalid
	}
	return s.Options[0].Type()
}

func (s *StaticData) Board() Board {
	return s.TargetBoard
}

func (s *StaticData) String() string {
	var t []string
	for _, o := range s.Options {
		t = append(t, o.Mime())
	}
	return fmt.Sprintf("goclip: %s [%s]", s.TargetBoard.String(), strings.Join(t, ", "))
}

func (s *StaticData) ToText(ctx context.Context) (string, error) {
	for _, data := range s.Options {
		if data.Type() == Text {
			res, err := data.Data(ctx)
			return string(res), err
		}
	}
	return "", os.ErrNotExist
}

func (s *StaticData) ToImage(ctx context.Context) (image.Image, error) {
	var buf []byte
	var err error
	var img image.Image

	for _, opt := range s.Options {
		if opt.Type() != Image {
			continue
		}
		// fmt is a kind of image, attempt to fetch & decode it
		buf, err = opt.Data(ctx)
		if err != nil {
			continue
		}

		// attempt to decode
		// Note: golang has no method to get a list of loaded formats. Having a method such as image.ListFormats() would
		// help a lot in selecting the best format from what is available in the clipboard. Instead we just try each one
		// until we have a hit.
		// See: https://github.com/golang/go/pull/46455 http://golang.org/cl/323669
		img, _, err = image.Decode(bytes.NewReader(buf))
		if err == nil {
			return img, nil
		}
	}

	if err != nil {
		return nil, err
	}
	return nil, os.ErrNotExist
}

func (s *StaticData) FileList() ([]string, error) {
	return nil, errors.New("TODO")
}

func (s *StaticData) HasFormat(fmt string) bool {
	for _, data := range s.Options {
		if data.Mime() == fmt {
			return true
		}
	}
	return false
}

func (s *StaticData) GetFormat(ctx context.Context, fmt string) ([]byte, error) {
	for _, data := range s.Options {
		if data.Mime() == fmt {
			return data.Data(ctx)
		}
	}
	// fallback to partial match for mime (ie if asking for text/plain, return text/plain;charset=utf-8)
	for _, data := range s.Options {
		m := data.Mime()
		if ppos := strings.IndexByte(m, ';'); ppos != -1 {
			if fmt == m[:ppos] {
				return data.Data(ctx)
			}
		}
	}
	return nil, os.ErrNotExist
}

func (s *StaticData) GetAllFormats() ([]DataOption, error) {
	return s.Options, nil
}
