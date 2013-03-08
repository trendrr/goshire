package dynmap

import (
	"strings"
	// "log"
	"encoding/json"
	"errors"
	"time"
	"fmt"
	"net/url"
)

//Dont make this a map type, since we want the option of 
//extending this and adding members.
type DynMap struct {
	Map map[string]interface{}
}

type DynMaper interface {
	ToDynMap() *DynMap
}

func NewDynMap() *DynMap {
	return &DynMap{make(map[string]interface{})}
}

//encodes this map into a url encoded string.
//maps are encoded in the rails style (key[key2][key2]=value)
func (this *DynMap) URLEncode() (string, error) {
	vals := &url.Values{}	
	for key,value := range(this.Map) {
		err := this.urlEncode(vals, key, value)
		if err != nil {
			return "", err
		}
	}
	return vals.Encode(), nil
}

//adds the requested value to the Values
func (this *DynMap) urlEncode(vals *url.Values, key string, value interface{}) error{
	
	if DynMapConvertable(value) {
		mp, ok := ToDynMap(value)
		if !ok {
			return fmt.Errorf("Unable to convert %s", mp)
		}	
		for k,v := range(mp.Map) {
			//encode in rails style key[key2]=value
			this.urlEncode(vals, fmt.Sprintf("%s[%s]",key,k), v)
		}
		return nil
	}
	switch v := value.(type) {
		case []interface{} :
			for _,tmp := range(v) {
				this.urlEncode(vals, key, tmp)
			}
			return nil
	}
	vals.Add(key, ToString(value))
	return nil
}

func (this *DynMap) MarshalJSON() ([]byte, error) {
	bytes, err := json.Marshal(this.Map)
	return bytes, err
}

func (this *DynMap) UnmarshalJSON(bytes []byte) error {
	return json.Unmarshal(bytes, &this.Map)
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

func (this *DynMap) MustInt(key string, def int) int {
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
func (this *DynMap) GetString(key string) (string, bool) {
	tmp, ok := this.Get(key)
	if !ok {
		return ToString(tmp), ok
	}
	return ToString(tmp), true
}

// gets a string. if string is not available in the map, then the default
//is returned
func (this *DynMap) GetStringOrDefault(key string, def string) string {
	tmp, ok := this.GetString(key)
	if !ok {
		return def
	}
	return tmp
}

func (this *DynMap) GetTime(key string) (time.Time, bool) {
	tmp, ok := this.Get(key)
	if !ok {
		return time.Now(), false
	}
	t, err := ToTime(tmp)
	if err != nil {
		return time.Now(), false
	}
	return t, true
}

func (this *DynMap) GetTimeOrDefault(key string, def time.Time) (time.Time) {
	tmp, ok := this.GetTime(key)
	if !ok {
		return def
	}
	return tmp
}

//Gets a dynmap from the requested.
// This will update the value in the map if the 
// value was not already a dynmap.
func (this *DynMap) DynMap(key string) (*DynMap, bool) {
	tmp, ok := this.Get(key)
	if !ok {
		return nil, ok
	}
	mp, ok := ToDynMap(tmp)
	return mp, ok
}

func (this *DynMap) MustDynMap(key string, def *DynMap) *DynMap {
	tmp, ok := this.DynMap(key)
	if !ok {
		return def
	}
	return tmp
}

// gets a slice of dynmaps
func (this *DynMap) GetDynMapSlice(key string) ([]*DynMap, bool) {
	lst, ok := this.Get(key)
	if !ok {
		return nil, false
	}
	switch v := lst.(type) {
	case []*DynMap :
		return v, true
	case []interface{} :
		retlist := make([]*DynMap, 0)
		for _,tmp := range(v) {
			in, ok := ToDynMap(tmp)
			if !ok {
				return nil, false
			}
			retlist = append(retlist, in)
		}
		return retlist, true
	}
	return nil, false
}

//Returns a slice of ints
func (this *DynMap) GetIntSlice(key string) ([]int, bool) {
	lst, ok := this.Get(key)
	if !ok {
		return nil, false
	}
	switch v := lst.(type) {
	case []int :
		return v, true
	case []interface{} :
		retlist := make([]int, 0)
		for _,tmp := range(v) {
			in, err := ToInt(tmp)
			if err != nil {
				return nil, false
			}
			retlist = append(retlist, in)
		}
		return retlist, true
	}
	return nil, false
}

// Adds the item to a slice
func (this *DynMap) AddToSlice(key string, mp interface{}) error{
	this.PutIfAbsent(key, make([]interface{}, 0))
	lst, _ := this.Get(key)
	switch v := lst.(type) {
	case []interface{} :
		v = append(v, mp)
		this.Put(key, v)
	}
	return nil
}

// puts all the values from the passed in map into this dynmap
// the passed in map must be convertable to a DynMap via ToDynMap.
// returns false if the passed value is not convertable to dynmap
func (this *DynMap) PutAll(mp interface{}) bool {
	dynmap, ok := ToDynMap(mp)
	if !ok {
		return false
	}
	for k, v := range dynmap.Map {
		this.Put(k, v)
	}
	return true
}

// 
// Puts the value into the map if and only if no value exists at the 
// specified key.
// This does not honor the dot operator on insert.
func (this *DynMap) PutIfAbsent(key string, value interface{}) (interface{}, bool) {
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
func (this *DynMap) PutIfAbsentWithDot(key string, value interface{}) (interface{}, bool) {
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
func (this *DynMap) PutWithDot(key string, value interface{}) error {
	splitStr := strings.Split(key, ".")
	if len(splitStr) == 1 {
		this.Put(key, value)
		return nil
	}
	mapKeys := splitStr[:(len(splitStr) - 1)]
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
				return val, ok
			}
		}
	}
	return val, ok
}
