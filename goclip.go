package goclip

import (
	"context"
	"errors"
	"image"
)

type Type int

const (
	Invalid Type = iota
	Text
	Image
	FileList
)

type Data interface {
	Type() Type
	String() string
	Image() image.Image
	FileList() []string
}

func Copy(ctx context.Context, values ...interface{}) error {
	return errors.New("TODO")
}

func Paste(ctx context.Context, types ...Type) (Data, error) {
	return PasteFrom(ctx, Default, types...)
}

func PasteFrom(ctx context.Context, from Board, types ...Type) (Data, error) {
	return emptyData{}, errors.New("TODO")
}
