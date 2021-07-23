package goclip

import (
	"context"
)

type Type int

var i = doInit()

const (
	Invalid Type = iota
	Text
	Image
	FileList
)

func isValidType(value Type) bool {
	switch value {
	case Text, Image, FileList:
		return true
	default:
		return false
	}
}

func Copy(ctx context.Context, values ...interface{}) error {
	value, err := spawnValue(values...)
	if err != nil {
		return err
	}
	return i.copy(ctx, Default, value)
}

func CopyTo(ctx context.Context, board Board, values ...interface{}) error {
	value, err := spawnValue(values...)
	if err != nil {
		return err
	}
	return i.copy(ctx, board, value)
}

func Paste(ctx context.Context) (Data, error) {
	return PasteFrom(ctx, Default)
}

func PasteFrom(ctx context.Context, from Board) (Data, error) {
	return i.paste(ctx, from)
}
