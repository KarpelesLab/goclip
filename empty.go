package goclip

import (
	"context"
	"image"
	"os"
)

var Empty Data = emptyData{}

type emptyData struct{}

func (e emptyData) Board() Board {
	return Default
}

func (e emptyData) Type() Type {
	return Invalid
}

func (e emptyData) String() string {
	return "goclip: empty data"
}

func (e emptyData) ToText(ctx context.Context) (string, error) {
	return "", nil
}

func (e emptyData) ToImage(ctx context.Context) (image.Image, error) {
	return nil, os.ErrNotExist
}

func (e emptyData) FileList() ([]string, error) {
	return nil, nil
}

func (e emptyData) GetFormat(ctx context.Context, f string) ([]byte, error) {
	return nil, os.ErrNotExist
}

func (e emptyData) HasFormat(f string) bool {
	return false
}

func (e emptyData) GetAllFormats() ([]DataOption, error) {
	return nil, nil
}
