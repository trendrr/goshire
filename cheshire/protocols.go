package cheshire

import (
    "encoding/json"
    "io"
            "github.com/trendrr/goshire/dynmap"
)


type Protocol interface {
    NewDecoder(io.Reader) Decoder
    WriteResponse(*Response, io.Writer) (int, error)
    WriteRequest(*Response, io.Writer) (int, error)
}

type Decoder interface {
    //Decode the next response from the reader
    DecodeResponse() (*Response, error)

    //decode the next request from the reader
    DecodeRequest() (*Request, error)
}


type JSONProtocol struct {   

}

var JSON = &JSONProtocol{}


func (this *JSONProtocol) NewDecoder(reader io.Reader) Decoder {
    dec := &JSONDecoder{
        dec : json.NewDecoder(reader),
    } 
    return dec
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
