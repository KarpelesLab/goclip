package goclip

import (
	"context"
)

func (a atom) Type() Type {
	if t, ok := fmtTypes[a.name]; ok {
		return t
	}
	return simpleTypeFromMime(a.name)
}

func (a atom) Mime() string {
	// note: might not always be mime
	return a.name
}

func (a atom) Data(ctx context.Context) ([]byte, error) {
	return i.fetch(ctx, a.board, a.value)
}
