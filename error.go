package goclip

import "errors"

var (
	ErrFormatUnavailable = errors.New("goclip: requested format was not available")
	ErrNoSys             = errors.New("goclip: no system is available")
	ErrNoBoard           = errors.New("goclip: requested board is not available")
	ErrDataNotString     = errors.New("goclip: requested data is not a String")
	ErrDataNotImage      = errors.New("goclip: requested data is not an Image")
	ErrDataNotFileList   = errors.New("goclip: requested data is not an FileList")
	ErrTiffImageDecode   = errors.New("goclip: cannot decode TIFF format image")
)
