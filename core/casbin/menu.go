package casbin

type Menu struct {
	Role   string `json:"role"`
	Path   string `json:"path"`
	Method string `json:"method"`
}

func (c *Casbin) GetMenuGroupingPolicies(g string) ([]Menu, error) {
	filter := c.SyncedCachedEnforcer.GetFilteredGroupingPolicy(0, g)
	var menus []Menu
	for _, v := range filter {
		p, err := c.GetMenuPolicies(v[1])
		if err != nil {
			return nil, err
		}

		menus = append(menus, p...)
	}
	return menus, nil
}

func (c *Casbin) GetMenuPolicies(role string) ([]Menu, error) {
	policies := c.SyncedCachedEnforcer.GetFilteredPolicy(0, role)
	menus := make([]Menu, len(policies))
	for i, policy := range policies {
		menus[i] = Menu{
			Role:   policy[0],
			Path:   policy[1],
			Method: policy[2],
		}
	}
	return menus, nil
}

func (c *Casbin) AddMenuPolicies(menus []Menu) (bool, error) {
	policies := make([][]string, len(menus))
	for i, menu := range menus {
		policies[i] = []string{menu.Role, menu.Path, menu.Method}
	}
	return c.SyncedCachedEnforcer.AddPolicies(policies)
}

func (c *Casbin) RemoveMenuPolicies(menus []Menu) (bool, error) {
	policies := make([][]string, len(menus))
	for i, menu := range menus {
		policies[i] = []string{menu.Role, menu.Path, menu.Method}
	}
	return c.SyncedCachedEnforcer.RemovePolicies(policies)
}

func (c *Casbin) UpdateMenuPolicies(menus []Menu) (bool, error) {
	oldMenuPolicies, err := c.GetMenuPolicies(menus[0].Role)
	if err != nil {
		return false, err
	}

	oldPolicies := make([][]string, len(oldMenuPolicies))
	for i, menu := range oldMenuPolicies {
		oldPolicies[i] = []string{menu.Role, menu.Path, menu.Method}
	}

	newPolicies := make([][]string, len(menus))
	for i, menu := range menus {
		newPolicies[i] = []string{menu.Role, menu.Path, menu.Method}
	}
	return c.SyncedCachedEnforcer.UpdatePolicies(oldPolicies, newPolicies)
}

func (c *Casbin) RemoveRole(role string) (bool, error) {
	return c.SyncedCachedEnforcer.DeleteRole(role)
}

func (c *Casbin) Enforce(role string, path, method string) (bool, error) {
	return c.SyncedCachedEnforcer.Enforce(role, path, method)
}
