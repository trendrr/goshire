package strest

import (
    "strings"
    // "log"
    "errors"
    "encoding/json"
    // "fmt"
)

//Dont make this a map type, since we want the option of 
//extending this and adding members.
type DynMap struct {
   Map map[string]interface{} 
}

func NewDynMap() *DynMap {
    return &DynMap{make(map[string]interface{})}
}

func (this *DynMap) MarshalJSON() ([]byte, error) {
    bytes,err := json.Marshal(this.Map)
    return bytes,err
}

func (this *DynMap) UnmarshalJSON(bytes []byte) error {
    return json.Unmarshal(bytes, this.Map)
}
// Gets the value at the specified key as an int64.  returns -1,false if value not available or is not convertable
func (this *DynMap) GetInt64(key string) (int64, bool) {
    tmp, ok := this.Get(key)
    if !ok {
        return -1, ok
    }
    val, err := ToInt64(tmp)
    if err == nil {
        return val, true
    }
    return -1, false
}

func (this *DynMap) GetIntOrDefault(key string, def int) (int) {
    v, ok := this.GetInt(key)
    if ok {
        return v
    }
    return def
} 

func (this *DynMap) GetInt(key string) (int, bool) {
    v, ok := this.GetInt64(key)
    if !ok {
        return -1, ok
    }
    return int(v), true
}

// 
// Gets a string representation of the value at key
// 
func (this *DynMap) GetString(key string) (string,bool) {
    tmp, ok := this.Get(key)
    if !ok {
        return ToString(tmp), ok
    }
    return ToString(tmp), true
}

// gets a string. if string is not available in the map, then the default
//is returned
func (this *DynMap) GetStringOrDefault(key string, def string) (string) {
    tmp, ok := this.GetString(key)
    if !ok {
        return def
    }
    return tmp
}

// puts all the values from the passed in map into this dynmap
// the passed in map must be convertable to a DynMap via ToDynMap.
// returns false if the passed value is not convertable to dynmap
func (this *DynMap) PutAll(mp interface{}) bool {
    dynmap, ok := ToDynMap(mp)
    if !ok {
        return false
    }
    for k,v := range dynmap.Map {
        this.Put(k, v)
    }
    return true;
}

// 
// Puts the value into the map if and only if no value exists at the 
// specified key.
// This does not honor the dot operator on insert.
func (this *DynMap) PutIfAbsent(key string, value interface{}) (interface{}, bool){
    v, ok := this.Get(key)
    if ok {
        return v, false
    }
    this.Put(key, value)
    return value, true
}

// 
// Same as PutIfAbsent but honors the dot operator
//
func (this *DynMap) PutIfAbsentWithDot(key string, value interface{}) (interface{}, bool){
    v, ok := this.Get(key)
    if ok {
        return v, false
    }
    this.PutWithDot(key, value)
    return value, true
}


//
// Put's a value into the map
//
func (this *DynMap) Put(key string, value interface{}) {
    this.Map[key] = value
}

//
// puts the value into the map, honoring the dot operator.
// so PutWithDot("map1.map2.value", 100)
// would result in:
// {
//   map1 : { map2 : { value: 100 }}
//
// }
func (this *DynMap) PutWithDot(key string, value interface{}) (error) {
    splitStr := strings.Split(key, ".")
    if len(splitStr) == 1 {
        this.Put(key, value)
        return nil
    }
    mapKeys := splitStr[:(len(splitStr)-1)]
    var mp = this.Map
    for _, k := range mapKeys {
        tmp, o := mp[k]
        if !o {
            //create a new map and insert
            newmap := make(map[string]interface{})
            mp[k] = newmap
            mp = newmap
        } else {
            mp, o = ToMap(tmp)
            if !o {
                //error
                return errors.New("Error, value at key was not a map")
            }
        }
    }
    mp[splitStr[len(splitStr)-1]] = value
    return nil 
}

func (this *DynMap) Exists(key string) bool {
    _, ok := this.Get(key)
    return ok
}

//
// Get's the value.  will honor the dot operator if needed.
// key = 'map.map2'
// will first attempt to matche the literal key 'map.map2'
// if no value is present it will look for a sub map at key 'map' 
//
func (this *DynMap) Get(key string) (interface{}, bool) {
    val, ok := this.Map[key]
    if ok {
        return val, true
    }
    //look for dot operator.
    splitStr := strings.Split(key, ".")
    if len(splitStr) == 1 {
        return val, false
    }

    var mp = this.Map
    for index, k := range splitStr {
        tmp, o := mp[k]
        if !o {
            return val, ok
        }

        if index == (len(splitStr) - 1) {
            return tmp, o
        } else {
            mp, o = ToMap(tmp)
            if !o {
                return val,ok
            }
        }
    }
    return val, ok
}

