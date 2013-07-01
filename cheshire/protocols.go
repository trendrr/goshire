package cheshire

import (
    "encoding/json"
    "io"
            "github.com/trendrr/goshire/dynmap"
    // "log"
)


type Protocol interface {
    NewDecoder(io.Reader) Decoder
    //Say hello on first connection (bin only)
    WriteHello(io.Writer) error
    WriteResponse(*Response, io.Writer) (int, error)
    WriteRequest(*Request, io.Writer) (int, error)
    Type() string

}

type Decoder interface {
    //decode the initial hello (bin only)
    DecodeHello() error

    //Decode the next response from the reader
    DecodeResponse() (*Response, error)

    //decode the next request from the reader
    DecodeRequest() (*Request, error)
}



///////////////////////
// The JSON protocol implementation
///////////////////////
type JSONProtocol struct {   

}

var JSON = &JSONProtocol{}

func (this *JSONProtocol) Type() string {
    return "json"
}

func (this *JSONProtocol) NewDecoder(reader io.Reader) Decoder {
    dec := &JSONDecoder{
        dec : json.NewDecoder(reader),
    } 
    return dec
}

func (this *JSONProtocol) WriteHello(writer io.Writer) error {
    return nil
}

func (this *JSONProtocol) WriteResponse(response *Response, writer io.Writer) (int, error) {
    json, err := response.MarshalJSON()
    if err != nil {
        return 0, err
    }
    bytes, err := writer.Write(json)
    return bytes, err
}
func (this *JSONProtocol) WriteRequest(request *Request, writer io.Writer) (int, error) {
    json, err := request.MarshalJSON()
    if err != nil {
        return 0, err
    }
    bytes, err := writer.Write(json)
    return bytes, err   
}

type JSONDecoder struct {
    dec *json.Decoder
}

func (this *JSONDecoder) DecodeHello() error {
    return nil
}

func (this *JSONDecoder) DecodeResponse() (*Response, error) {
    mp := dynmap.New()
    err := this.dec.Decode(mp)
    if err != nil {
        return nil, err
    }
    req := NewResponseDynMap(mp)
    return req, nil
}

func (this *JSONDecoder) DecodeRequest() (*Request, error) {
    mp := dynmap.New()
    err := this.dec.Decode(mp)
    if err != nil {
        return nil, err
    }
    req := NewRequestDynMap(mp)
    return req, nil
}
