package casbin

type Menu struct {
	Role   string `json:"role"`
	Path   string `json:"path"`
	Method string `json:"method"`
}

func (c *Casbin) AddPolicy(menus []Menu) error {
	for _, menu := range menus {
		if _, err := c.E.AddPolicy(menu.Role, menu.Path, menu.Method); err != nil {
			return err
		}
	}
	return nil
}

func (c *Casbin) RemovePolicy(menus []Menu) error {
	for _, menu := range menus {
		if _, err := c.E.RemovePolicy(menu.Role, menu.Path, menu.Method); err != nil {
			return err
		}
	}
	return nil
}

func (c *Casbin) UpdatePolicy(menus []Menu) error {
	if err := c.RemovePolicy(menus); err != nil {
		return err
	}
	return c.AddPolicy(menus)
}

func (c *Casbin) RemoveRole(role string) error {
	if _, err := c.E.DeleteRole(role); err != nil {
		return err
	}
	return nil
}

func (c *Casbin) Enforce(role int, path, method string) (bool, error) {
	return c.E.Enforce(role, path, method)
}
