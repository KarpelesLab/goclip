//go:build !linux && !windows && !darwin
// +build !linux,!windows,!darwin

package goclip

// fallback when no support is available
type internal struct{}
