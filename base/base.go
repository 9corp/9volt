package base

import (
	"context"
)

type IComponent interface {
	Start() error
	Stop() error
	Identify() string
}

type Component struct {
	Identifier string

	Ctx    context.Context
	Cancel context.CancelFunc
}

func (c *Component) Identify() string {
	return c.Identifier
}
