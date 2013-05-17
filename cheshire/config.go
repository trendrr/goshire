package cheshire

import (
	"fmt"
	"github.com/kylelemons/go-gypsy/yaml"
	"github.com/trendrr/goshire/dynmap"
	"log"
)

type ServerConfig struct {
	*dynmap.DynMap
	Router  RouteMatcher
	Filters []ControllerFilter
}

// Creates a new server config with a default routematcher
func NewServerConfig() *ServerConfig {
	return &ServerConfig{
		dynmap.NewDynMap(),
		NewDefaultRouter(),
		make([]ControllerFilter, 0),
	}
}

// Registers a controller with the RouteMatcher.  
// shortcut to conf.Router.Register(controller)
func (this *ServerConfig) Register(methods []string, controller Controller) {
	log.Println("Registering: ", methods, " ", controller.Config().Route, " ", controller)
	this.Router.Register(methods, controller)
}


// Parses a server config from a YAML file
func NewServerConfigFile(path string) *ServerConfig {
	conf, err := yaml.ReadFile(path)
	if err != nil {
		//do something
		log.Println(err)
		return nil
	}

	m, e := conf.Root.(yaml.Map)
	if !e {
		//not a proper config. fail!
		panic(fmt.Sprintf("Config.init error(%q): %s", path, err))
	}

	dynmap := toDynMap(m)
	instance := NewServerConfig()
	instance.PutAll(dynmap)
	return instance
}

func fromNode(node yaml.Node) interface{} {
	switch valType := node.(type) {
	case yaml.Map:
		return toDynMap(valType)
	case yaml.List:
		sl := make([]interface{}, len(valType))
		for i, v := range valType {
			sl[i] = fromNode(v)
		}
		return sl
	case yaml.Scalar:
		return valType.String()

	}
	return nil //should never be possible
}

// fills the passed in dynmap with the values from the yaml map
func toDynMap(mp yaml.Map) *dynmap.DynMap {
	d := dynmap.NewDynMap()
	for k, v := range mp {
		d.Put(k, fromNode(v))
	}
	return d
}
