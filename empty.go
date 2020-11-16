package goclip

import "image"

type emptyData struct{}

func (e emptyData) Type() Type {
	return Invalid
}

func (e emptyData) String() string {
	return ""
}

func (e emptyData) Image() image.Image {
	return nil
}

func (e emptyData) FileList() []string {
	return nil
}
