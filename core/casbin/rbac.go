package casbin

type Api struct {
	Role   string `json:"role"`
	Path   string `json:"path"`
	Method string `json:"method"`
}

func (c *Casbin) GetApiGroupingPolicies(g string) []Api {
	filter := c.SyncedCachedEnforcer.GetFilteredGroupingPolicy(0, g)
	var apis []Api
	for _, v := range filter {
		p := c.GetApiPolicies(v[1])
		apis = append(apis, p...)
	}
	return apis
}

func (c *Casbin) GetApiPolicies(role string) []Api {
	policies := c.SyncedCachedEnforcer.GetFilteredPolicy(0, role)
	apis := make([]Api, len(policies))
	for i, policy := range policies {
		apis[i] = Api{
			Role:   policy[0],
			Path:   policy[1],
			Method: policy[2],
		}
	}
	return apis
}

func (c *Casbin) AddApiPolicies(apis []Api) (bool, error) {
	policies := convertApisToPolicies(apis)
	return c.SyncedCachedEnforcer.AddPolicies(policies)
}

func (c *Casbin) RemoveApiPolicies(apis []Api) (bool, error) {
	policies := convertApisToPolicies(apis)
	return c.SyncedCachedEnforcer.RemovePolicies(policies)
}

func (c *Casbin) UpdateApiPolicies(apis []Api) (bool, error) {
	oldApiPolicies := c.GetApiPolicies(apis[0].Role)
	oldPolicies := convertApisToPolicies(oldApiPolicies)
	newPolicies := convertApisToPolicies(apis)
	return c.SyncedCachedEnforcer.UpdatePolicies(oldPolicies, newPolicies)
}

func (c *Casbin) RemoveRole(role string) (bool, error) {
	return c.SyncedCachedEnforcer.DeleteRole(role)
}

func (c *Casbin) Enforce(role string, path, method string) (bool, error) {
	return c.SyncedCachedEnforcer.Enforce(role, path, method)
}

func convertApisToPolicies(apis []Api) [][]string {
	policies := make([][]string, len(apis))
	for i, api := range apis {
		policies[i] = []string{api.Role, api.Path, api.Method}
	}
	return policies
}
