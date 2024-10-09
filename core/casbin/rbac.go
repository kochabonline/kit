package casbin

type Policy struct {
	Role   string `json:"role"`
	Path   string `json:"path"`
	Method string `json:"method"`
}

func convert(policies []Policy) [][]string {
	p := make([][]string, len(policies))
	for i, policy := range policies {
		p[i] = []string{policy.Role, policy.Path, policy.Method}
	}
	return p
}

func (c *Casbin) GetGroupingPolicies(g string) ([]Policy, error) {
	filter, err := c.SyncedCachedEnforcer.GetFilteredGroupingPolicy(0, g)
	if err != nil {
		return nil, err
	}

	var policies []Policy
	for _, v := range filter {
		p, err := c.GetPolicies(v[1])
		if err != nil {
			return nil, err
		}
		policies = append(policies, p...)
	}

	return policies, nil
}

func (c *Casbin) GetPolicies(role string) ([]Policy, error) {
	filter, err := c.SyncedCachedEnforcer.GetFilteredPolicy(0, role)
	if err != nil {
		return nil, err
	}

	policies := make([]Policy, len(filter))
	for i, policy := range filter {
		policies[i] = Policy{
			Role:   policy[0],
			Path:   policy[1],
			Method: policy[2],
		}
	}

	return policies, nil
}

func (c *Casbin) AddPolicies(policies []Policy) (bool, error) {
	p := convert(policies)
	return c.SyncedCachedEnforcer.AddPolicies(p)
}

func (c *Casbin) RemovePolicies(policies []Policy) (bool, error) {
	p := convert(policies)
	return c.SyncedCachedEnforcer.RemovePolicies(p)
}

func (c *Casbin) UpdatePolicies(policies []Policy) (bool, error) {
	old, err := c.GetPolicies(policies[0].Role)
	if err != nil {
		return false, err
	}
	oldPolicies := convert(old)
	newPolicies := convert(policies)
	return c.SyncedCachedEnforcer.UpdatePolicies(oldPolicies, newPolicies)
}

func (c *Casbin) RemoveRole(role string) (bool, error) {
	return c.SyncedCachedEnforcer.DeleteRole(role)
}

func (c *Casbin) Enforce(role string, path, method string) (bool, error) {
	return c.SyncedCachedEnforcer.Enforce(role, path, method)
}
