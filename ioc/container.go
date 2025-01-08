package ioc

const (
	ConfigNamespace     = "config"
	DataBaseNamespace   = "database"
	HandlerNamespace    = "handler"
	ControllerNamespace = "controller"
)

var Container = new(Store)

func init() {
	Container = NewStore(WithNamespaces(map[string]namespace{
		ConfigNamespace:     {name: ConfigNamespace, object: map[string]object{}, priority: -9},
		DataBaseNamespace:   {name: DataBaseNamespace, object: map[string]object{}, priority: -8},
		HandlerNamespace:    {name: HandlerNamespace, object: map[string]object{}, priority: -7},
		ControllerNamespace: {name: ControllerNamespace, object: map[string]object{}, priority: -6},
	}))
}
