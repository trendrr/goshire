package strest

import (
    "fmt"
    "strconv"
    "log"
)

func ToInt64(value interface{}) (i int64, err error) {
    switch v := value.(type) {
    case string:
        i, err := strconv.ParseInt(v, 0, 64)
        return i, err
    case int64:
        return v, nil
    case int32:
        return int64(v), nil
    case int16:
        return int64(v), nil
    case int8:
        return int64(v), nil
    case uint64:
        return int64(v), nil
    case uint32:
        return int64(v), nil
    case uint16:
        return int64(v), nil
    case uint8:
        return int64(v), nil
    case float32:
        return int64(v), nil
    case float64:
        return int64(v), nil
    default:
        log.Println("uhh ", v )
    
    }
    //TODO: baaaad
    return -1, nil
}

func ToString(value interface{}) (string) {
    return fmt.Sprint(value)
}

func ToDynMap(value interface{}) (*DynMap, bool) {
    switch v := value.(type) {
    case map[string]interface{}:
        dynmap := NewDynMap()
        dynmap.Map = v
        return dynmap, true
    case *map[string]interface{}:
        dynmap := NewDynMap()
        dynmap.Map = *v
        return dynmap, true
    case DynMap:
        return &v, true
    case *DynMap:
        return v, true
    }
    return nil, false
}

//
// attempts to convert the given value to a map.
// returns 
func ToMap(value interface{}) (map[string]interface{}, bool) {
    switch v := value.(type) {
    case map[string]interface{}:
        return v, true
    case *map[string]interface{}:
        return *v, true
    default:
        dynmap, ok := ToDynMap(value)
        if ok {
            return dynmap.Map, true
        }
    }
    return nil, false
}
