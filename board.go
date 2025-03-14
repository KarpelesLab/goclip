package goclip

import "fmt"

// Board represents a clipboard selection board
// Each platform has at least one board (Default), while X11-based systems
// like Linux have additional selections (Primary and Secondary)
type Board uint8

const (
	// InvalidBoard represents an invalid clipboard board
	InvalidBoard Board = iota
	// Default is the standard clipboard used across all platforms
	Default // the default clipboard
	// PrimarySelection is the X11 primary selection (X11/Linux only)
	PrimarySelection // the primary selection (X11 only)
	// SecondarySelection is the X11 secondary selection (X11/Linux only)
	SecondarySelection // the secondary selection (X11 only)
)

// String returns a human-readable name for the clipboard board
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
