package cheshire

import (
    "encoding/binary"
    "io"
    "fmt"
    "encoding/json"
)

var TXN_ACCEPT = []string{
    "single", //0
    "multi", //1
}

var METHOD = []string{
    "GET", //0
    "POST", //1
    "PUT", //2
    "DELETE" //3
}

var PARAM_ENCODING = []string{
    
}


// Implementation of the binary protocol
type BinProtocol struct {

}

var BIN = &BinProtocol{

}

func (this *BinProtocol) Type() string {
    return "bin"
}

// Say hello
func (this *BinProtocol) Hello(writer io.Writer) error {
    str := fmt.Sprintf("%f %s", StrestVersion, "golang")
    return writeString(str, writer)
}

func (this *BinProtocol) NewDecoder(reader io.Reader) Decoder {
    dec := &JSONDecoder{
        dec : json.NewDecoder(reader),
    } 
    return dec
}
func (this *BinProtocol) WriteResponse(response *Response, writer io.Writer) (int, error) {
    




    json, err := response.MarshalJSON()
    if err != nil {
        return 0, err
    }
    bytes, err := writer.Write(json)
    return bytes, err
}
func (this *BinProtocol) WriteRequest(request *Request, writer io.Writer) (int, error) {
    json, err := request.MarshalJSON()
    if err != nil {
        return 0, err
    }
    bytes, err := writer.Write(json)
    return bytes, err   
}




// Reads a length prefixed byte array
// it assumes the first 
func readByteArray(reader io.Reader) ([]byte, error) {
    length := int16(0)
    err := binary.Read(reader, binary.BigEndian, &length)
    if err != nil {
        return nil, err
    }

    bytes := make([]byte, length)
    _, err = io.ReadAtLeast(reader, bytes, int(length))
    return bytes, err
}

//Reads a length prefixed utf8 string 
func readString(reader io.Reader) (string, error) {
    b, err := readByteArray(reader)
    if err != nil {
        return "", err
    }
    return string(b), nil
}

//writes a length prefixed byte array
func writeByteArray(bytes []byte, writer io.Writer) error {
    length := int16(len(bytes))
    err := binary.Write(writer, binary.BigEndian, length)
    if err != nil {
        return err
    }
    _, err = writer.Write(bytes)
    return err
}

//writes a length prefixed utf8 string 
func writeString(str string, writer io.Writer) error {
    return writeByteArray([]byte(str), writer)
}