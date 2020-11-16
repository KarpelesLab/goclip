package goclip

import "image"

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
