package cheshire

import (
	"github.com/kylelemons/go-gypsy/yaml"
	"strest"
	"log"
	"fmt"
)


// Parses a server config from a YAML file
func NewServerConfigFile(path string) *strest.ServerConfig {
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
	instance := strest.NewServerConfig()
	instance.PutAll(dynmap)
	return instance
}


func fromNode(node yaml.Node) interface{} {
	switch valType := node.(type) {	
		case yaml.Map :
			return toDynMap(valType)
		case yaml.List :
			sl := make([]interface{}, len(valType))
			for i,v := range valType {
				sl[i] = fromNode(v)
			}
			return sl
		case yaml.Scalar :
			return valType.String()
		
			
	}
	return nil //should never be possible
}

// fills the passed in dynmap with the values from the yaml map
func toDynMap(mp yaml.Map) *strest.DynMap {
	dynmap := strest.NewDynMap()
	for k,v := range mp {
		dynmap.Put(k, fromNode(v))
	}
	return dynmap
}
