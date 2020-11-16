package goclip

type Board uint8

const (
	InvaliBoard        Board = iota
	Default                  // the default clipboard
	PrimarySelection         // the primary selection (X11 only)
	SecondarySelection       // the secondary selection (X11 only)
)
