package ioc

import (
	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/log/zerolog"
)

const (
	ConfigNamespace     = "config"
	DataBaseNamespace   = "database"
	HandlerNamespace    = "handler"
	ControllerNamespace = "controller"
)

var Container = &Store{
	namespaces: map[string]namespace{
		ConfigNamespace:     {name: ConfigNamespace, object: map[string]object{}, priority: -9},
		DataBaseNamespace:   {name: DataBaseNamespace, object: map[string]object{}, priority: -8},
		HandlerNamespace:    {name: HandlerNamespace, object: map[string]object{}, priority: -7},
		ControllerNamespace: {name: ControllerNamespace, object: map[string]object{}, priority: -6},
	},
}

func init() {
	Container.SetLogger(log.NewHelper(zerolog.New()))
}
