package goclip

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"os"
	"strings"
)

// Data is the interface for clipboard data access
// It provides methods to retrieve clipboard data in various formats
// and access to platform-specific clipboard formats
type Data interface {
	// Type returns the primary type of the clipboard data (Text, Image, FileList)
	Type() Type
	// Board returns the clipboard board this data is associated with
	Board() Board
	// ToText converts the clipboard data to a string representation
	ToText(ctx context.Context) (string, error)
	// ToImage converts the clipboard data to an image representation
	ToImage(ctx context.Context) (image.Image, error)
	// FileList returns a list of files if the clipboard contains file references
	FileList() ([]string, error)

	// direct format accessors using MIME formats
	// HasFormat checks if data in a specific MIME format exists
	HasFormat(fmt string) bool
	// GetFormat retrieves data in a specific MIME format
	GetFormat(ctx context.Context, fmt string) ([]byte, error)
	// GetAllFormats returns all available data formats
	GetAllFormats() ([]DataOption, error)
}

// StaticDataOption represents a single clipboard data format with MIME type
// and associated binary data
type StaticDataOption struct {
	// StaticType is the MIME type such as "image/png" or "text/plain;charset=utf-8"
	StaticType string
	// StaticData contains the actual binary data
	StaticData []byte
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
// instances of StaticDataOption, or objects following the DataOption interface.
// This is the primary implementation of the Data interface.
type StaticData struct {
	// TargetBoard is the clipboard board this data belongs to
	TargetBoard Board
	// Options is a list of available clipboard data formats
	Options []DataOption
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
	// Look for file list format in options
	for _, opt := range s.Options {
		if mime := opt.Mime(); strings.HasPrefix(mime, "text/uri-list") {
			data, err := opt.Data(context.Background())
			if err != nil {
				return nil, err
			}

			// Parse URI list - one URI per line
			files := []string{}
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue // Skip empty lines and comments
				}
				// Convert URI to file path - basic implementation
				if strings.HasPrefix(line, "file://") {
					path := line[7:]
					files = append(files, path)
				}
			}

			if len(files) > 0 {
				return files, nil
			}
		}
	}

	return nil, os.ErrNotExist
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
