package goclip

import "fmt"

type Board uint8

const (
	InvalidBoard       Board = iota
	Default                  // the default clipboard
	PrimarySelection         // the primary selection (X11 only)
	SecondarySelection       // the secondary selection (X11 only)
)

func (b Board) String() string {
	switch b {
	case InvalidBoard:
		return "Invalid"
	case Default:
		return "Default"
	case PrimarySelection:
		return "Primary Selection"
	case SecondarySelection:
		return "Secondary Selection"
	default:
		return fmt.Sprintf("Invalid #%d", b)
	}
}
