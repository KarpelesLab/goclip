// Package goclip provides cross-platform clipboard functionality for Go
package goclip

import (
	"context"
)

// Type represents the type of data stored in the clipboard
type Type int

var i = doInit()

const (
	// Invalid represents an invalid or unsupported clipboard data type
	Invalid Type = iota
	// Text represents text data in the clipboard
	Text
	// Image represents image data in the clipboard
	Image
	// FileList represents a list of files in the clipboard
	FileList
)

// Copy copies the given values to the default clipboard
func Copy(ctx context.Context, values ...interface{}) error {
	value, err := spawnValue(values...)
	if err != nil {
		return err
	}
	return i.copy(ctx, Default, value)
}

// CopyTo copies the given values to the specified clipboard board
func CopyTo(ctx context.Context, board Board, values ...interface{}) error {
	value, err := spawnValue(values...)
	if err != nil {
		return err
	}
	return i.copy(ctx, board, value)
}

// Paste retrieves data from the default clipboard
func Paste(ctx context.Context) (Data, error) {
	return PasteFrom(ctx, Default)
}

// PasteFrom retrieves data from the specified clipboard board
func PasteFrom(ctx context.Context, from Board) (Data, error) {
	return i.paste(ctx, from)
}
