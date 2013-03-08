package dynmap

import(
    "testing"
    "log"
)

//go test -v github.com/trendrr/cheshire-golang/cheshire
func TestJsonMarshal(t *testing.T) {
    mp := NewDynMap()
    mp.PutWithDot("this.that.test", 80)
    mp.PutWithDot("this.eight", 8)
    bytes, _ := mp.MarshalJSON()
    log.Printf("Got JSON %s", bytes)

    un := NewDynMap()
    un.UnmarshalJSON(bytes)

    unbytes, _ := mp.MarshalJSON()
    if string(unbytes) != string(bytes) {
        log.Println("ERROR")
    }
}

func TestURLEncode(t *testing.T) {
    mp := NewDynMap()
    mp.PutWithDot("this.that.test", 80)
    mp.PutWithDot("this.eight", 8)
    url, err := mp.URLEncode()
    if err != nil {
        log.Println(err)
    }
    log.Printf("Got URL : %s", url)
}

