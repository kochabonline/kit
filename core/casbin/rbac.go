package casbin

type Rule struct {
	Role   string `json:"role" validate:"required"`
	Path   string `json:"path" validate:"required"`
	Method string `json:"method" validate:"required"`
}

type Policies struct {
	Rules []Rule `json:"rules" validate:"required,dive"`
}

func convert(rules []Rule) [][]string {
	r := make([][]string, len(rules))
	for i, rule := range rules {
		r[i] = []string{rule.Role, rule.Path, rule.Method}
	}
	return r
}

func (c *Casbin) GetGroupingPolicies(g string) ([]Rule, error) {
	filter, err := c.SyncedCachedEnforcer.GetFilteredGroupingPolicy(0, g)
	if err != nil {
		return nil, err
	}

	var rules []Rule
	for _, v := range filter {
		policies, err := c.GetPolicies(v[1])
		if err != nil {
			return nil, err
		}
		rules = append(rules, policies...)
	}

	return rules, nil
}

func (c *Casbin) GetPolicies(role string) ([]Rule, error) {
	filter, err := c.SyncedCachedEnforcer.GetFilteredPolicy(0, role)
	if err != nil {
		return nil, err
	}

	rules := make([]Rule, 0, len(filter))
	for _, rule := range filter {
		rules = append(rules, Rule{
			Role:   rule[0],
			Path:   rule[1],
			Method: rule[2],
		})
	}

	return rules, nil
}

func (c *Casbin) AddPolicies(rules []Rule) (bool, error) {
	r := convert(rules)
	return c.SyncedCachedEnforcer.AddPolicies(r)
}

func (c *Casbin) RemovePolicies(rules []Rule) (bool, error) {
	r := convert(rules)
	return c.SyncedCachedEnforcer.RemovePolicies(r)
}

func (c *Casbin) UpdatePolicies(oldRules []Rule, newRules []Rule) (bool, error) {
	oldPolicies := convert(oldRules)
	newPolicies := convert(newRules)
	return c.SyncedCachedEnforcer.UpdatePolicies(oldPolicies, newPolicies)
}

func (c *Casbin) Enforce(role string, path, method string) (bool, error) {
	return c.SyncedCachedEnforcer.Enforce(role, path, method)
}
