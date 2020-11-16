package goclip

import "errors"

var (
	ErrFormatUnavailable = errors.New("goclip: requested format was not available")
	ErrNoBoard           = errors.New("goclip: requested board is not available")
)
