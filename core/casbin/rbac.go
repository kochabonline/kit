package casbin

type Policy struct {
	Role   string `json:"role" validate:"required"`
	Path   string `json:"path" validate:"required"`
	Method string `json:"method" validate:"required"`
}

type ApiPolicies struct {
	Policies []Policy `json:"policies" validate:"required,dive"`
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

	policies := make([]Policy, 0, len(filter))
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

	policies := make([]Policy, 0, len(filter))
	for _, policy := range filter {
		policies = append(policies, Policy{
			Role:   policy[0],
			Path:   policy[1],
			Method: policy[2],
		})
	}

	return policies, nil
}

func (c *Casbin) AddPolicies(policies []Policy) (bool, error) {
	r := convert(policies)
	return c.SyncedCachedEnforcer.AddPolicies(r)
}

func (c *Casbin) RemovePolicies(policies []Policy) (bool, error) {
	r := convert(policies)
	return c.SyncedCachedEnforcer.RemovePolicies(r)
}

func (c *Casbin) UpdatePolicies(oldPolicies []Policy, newPolicies []Policy) (bool, error) {
	old := convert(oldPolicies)
	new := convert(newPolicies)
	return c.SyncedCachedEnforcer.UpdatePolicies(old, new)
}

func (c *Casbin) Enforce(role string, path, method string) (bool, error) {
	return c.SyncedCachedEnforcer.Enforce(role, path, method)
}
