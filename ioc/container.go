package ioc

const (
	ConfigNamespace     = "config"
	DataBaseNamespace   = "database"
	HandlerNamespace    = "handler"
	ControllerNamespace = "controller"
)

var Container = new(Store)

func init() {
	Container = NewStore(WithNamespaces(map[string]*namespace{
		ConfigNamespace:     {name: ConfigNamespace, objects: make(map[string]*object), priority: -9},
		DataBaseNamespace:   {name: DataBaseNamespace, objects: make(map[string]*object), priority: -8},
		HandlerNamespace:    {name: HandlerNamespace, objects: make(map[string]*object), priority: -7},
		ControllerNamespace: {name: ControllerNamespace, objects: make(map[string]*object), priority: -6},
	}))
}
