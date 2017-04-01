package base

type IComponent interface {
	Start() error
	Stop() error
	Identify() string
}

type Component struct {
	Identifier string
}

func (c *Component) Identify() string {
	return c.Identifier
}
