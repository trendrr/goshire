package cheshire

import (
    "encoding/binary"
    "io"
    "fmt"
    "log"
    "github.com/trendrr/goshire/dynmap"
)

var TXN_ACCEPT = []string{
    "single", //0
    "multi", //1
}

var TXN_STATUS = []string{
    "completed", //0
    "continue", //1
}


var METHOD = []string{
    "GET", //0
    "POST", //1
    "PUT", //2
    "DELETE", //3
}

var PARAM_ENCODING = []string{
    "json",
    "msgpack",
}

var CONTENT_ENCODING = []string{
    "json",
    "msgpack",
    "json-gzip",
    "gzip",
    "bytes",
}

type BinConstants struct {
    TxnAccept map[string]int8
    TxnStatus map[string]int8
    Method map[string]int8
    ParamEncoding map[string]int8
    ContentEncoding map[string]int8
}

var BINCONST = contstantsinit()

func contstantsinit() *BinConstants {
    c := &BinConstants{
        TxnAccept : make(map[string]int8, len(TXN_ACCEPT)),
        TxnStatus : make(map[string]int8, len(TXN_STATUS)),
        Method : make(map[string]int8, len(METHOD)),
        ParamEncoding : make(map[string]int8, len(PARAM_ENCODING)),
        ContentEncoding : make(map[string]int8, len(CONTENT_ENCODING)),
    }
    for i,s := range(TXN_ACCEPT) {
        c.TxnAccept[s] = int8(i)

    }
    for i,s := range(TXN_STATUS) {
        c.TxnStatus[s] = int8(i)
    }
    for i,s := range(METHOD) {
        c.Method[s] = int8(i)
    }
    for i,s := range(PARAM_ENCODING) {
        c.ParamEncoding[s] = int8(i)
    }
    for i,s := range(CONTENT_ENCODING) {
        c.ContentEncoding[s] = int8(i)
    }
    return c
}

type BinDecoder struct {
    reader io.Reader
}

func (this *BinDecoder) DecodeHello() error {
    //read the hello
    hello, err := readString(this.reader)
    if err != nil {
        log.Print(err)
        //TODO: Send bad hello.
        return err
    }
    log.Println(hello)
    return nil
}    

    //Decode the next response from the reader
func (this *BinDecoder) DecodeResponse() (*Response, error) {
    headerLength := int32(0) 
    err := binary.Read(this.reader, binary.BigEndian, &headerLength)
    if err != nil {
        return nil, err
    }

    txnId, err := readString(this.reader)
    if err != nil {
        return nil, err
    }

    // log.Printf("txn %s", txnId)
    txnStatus := int8(0)
    err = binary.Read(this.reader, binary.BigEndian, &txnStatus)
    if err != nil {
        return nil, err
    }
    if int(txnStatus) >= len(TXN_STATUS) {
        return nil, fmt.Errorf("TxnStatus too large %d", txnStatus)
    }

    statusCode := int16(0)
    err = binary.Read(this.reader, binary.BigEndian, &statusCode)
    if err != nil {
        return nil, err
    }
    // log.Printf("Status %d", statusCode)
    statusMessage, err := readString(this.reader)
    if err != nil {
        return nil, err
    }
    
    contentEncoding := int8(0)
    err = binary.Read(this.reader, binary.BigEndian, &contentEncoding)
    if err != nil {
        return nil, err
    }
    if int(contentEncoding) >= len(CONTENT_ENCODING) {
        return nil, fmt.Errorf("contentEncoding too large %d", contentEncoding)
    }
    
    // log.Println(contentEncoding)

    contentLength := int32(0)
    err = binary.Read(this.reader, binary.BigEndian, &contentLength)
    if err != nil {
        return nil, err
    }
    // log.Printf("contentLentgh %d", contentLength)

    content := make([]byte, contentLength)
    
    if contentLength > 0 {
        _, err = io.ReadAtLeast(this.reader, content, int(contentLength))
        if err != nil {
            return nil, err
        }
    }
    //create the response

    response := &Response{
        DynMap : *dynmap.New(),
        txnId : txnId,
        txnStatus : TXN_STATUS[int(txnStatus)],
        statusCode : int(statusCode),
        statusMessage : statusMessage,
        contentEncoding : CONTENT_ENCODING[int(contentEncoding)],
        content : content,
    }
    // log.Printf("READ RESPONSE: %s", response)
    return response, nil
}

    //decode the next request from the reader
func (this *BinDecoder) DecodeRequest() (*Request, error) {
    headerLength := int32(0) 
    err := binary.Read(this.reader, binary.BigEndian, &headerLength)
    if err != nil {
        return nil, err
    }

    txnId, err := readString(this.reader)
    if err != nil {
        return nil, err
    }

    txnAccept := int8(0)
    err = binary.Read(this.reader, binary.BigEndian, &txnAccept)
    if err != nil {
        return nil, err
    }
    if int(txnAccept) >= len(TXN_ACCEPT) {
        return nil, fmt.Errorf("TxnAccept too large %d", txnAccept)
    }

    method := int8(0)
    err = binary.Read(this.reader, binary.BigEndian, &method)
    if err != nil {
        return nil, err
    }
    if int(method) >= len(METHOD) {
        return nil, fmt.Errorf("Method too large %d", method)
    }

    uri, err := readString(this.reader)
    if err != nil {
        return nil, err
    }

    paramEncoding := int8(0)
    err = binary.Read(this.reader, binary.BigEndian, &paramEncoding)
    if err != nil {
        return nil, err
    }
    if int(paramEncoding) >= len(PARAM_ENCODING) {
        return nil, fmt.Errorf("paramEncoding too large %d", paramEncoding)
    }

    paramsArray, err := readByteArray(this.reader)
    if err != nil {
        return nil, err
    }
    params, err := ParseParams(paramEncoding, paramsArray)
    if err != nil {
        return nil, err
    }
    // log.Println(params)
    contentEncoding := int8(0)
    err = binary.Read(this.reader, binary.BigEndian, &contentEncoding)
    if err != nil {
        return nil, err
    }
    if int(contentEncoding) >= len(CONTENT_ENCODING) {
        return nil, fmt.Errorf("contentEncoding too large %d", contentEncoding)
    }
    // log.Printf("Content encoding %d", contentEncoding)
    
    contentLength := int32(0)
    err = binary.Read(this.reader, binary.BigEndian, &contentLength)
    if err != nil {
        return nil, err
    }
    // log.Printf("Content len %d", contentLength)

    content := make([]byte, contentLength)
    _, err = io.ReadAtLeast(this.reader, content, int(contentLength))
    if err != nil {
        return nil, err
    }

    //create the request
    request := &Request{
        version : StrestVersion, //TODO: get this from hello
        userAgent : "binary",
        txnId : txnId,
        txnAccept : TXN_ACCEPT[int(txnAccept)],
        uri : uri,
        method : METHOD[int(method)],
        params : params,
        contentEncoding : CONTENT_ENCODING[int(contentEncoding)],
        content : content,
    }

    return request, nil


}

func ParseParams(paramEncoding int8, params []byte) (*dynmap.DynMap, error) {
    if len(params) == 0 {
        return dynmap.New(), nil
    }

    if paramEncoding == 0 {
        //json
        mp := dynmap.New()
        err := mp.UnmarshalJSON(params)
        return mp, err
    }
    return nil, fmt.Errorf("Unsupported param encoding %d", paramEncoding)
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
func (this *BinProtocol) WriteHello(writer io.Writer) error {
    str := fmt.Sprintf("%f %s", StrestVersion, "golang")
    _, err := writeString(writer, str)
    return err
}

func (this *BinProtocol) NewDecoder(reader io.Reader) Decoder {
    dec := &BinDecoder{
        reader : reader,
    } 
    return dec
}
func (this *BinProtocol) WriteResponse(response *Response, writer io.Writer) (int, error) {
    headerLength :=
        2 + //txnId length
        len(response.TxnId()) +
        1 + //txn status
        2 + //status
        2 + //statusmessage length
        len(response.StatusMessage()) +
        2 + //content encoding
        4 //content length

    //write the header length
    err := binary.Write(writer, binary.BigEndian, int32(headerLength))

    //txn id
    _, err = writeString(writer, response.TxnId())
    if err != nil {
        return 0, err
    }

    //txn status
    txnStatus, ok := BINCONST.TxnStatus[response.TxnStatus()]
    if !ok {
        return 0, fmt.Errorf("Bad TXN Status %s", response.TxnStatus())
    }
    err = binary.Write(writer, binary.BigEndian, txnStatus)
    if err != nil {
        return 0, err
    }

    err = binary.Write(writer, binary.BigEndian, int16(response.StatusCode()))
    if err != nil {
        return 0, err
    }

    _, err = writeString(writer, response.StatusMessage())
    if err != nil {
        return 0, err
    }

    contentLength := int32(0)
    contentEncoding, ok := BINCONST.ContentEncoding["bytes"]
    contentEncodingStr, contentSet := response.ContentEncoding()
    content := []byte{}

    if contentSet {
        contentEncoding, ok = BINCONST.ContentEncoding[contentEncodingStr]
        if !ok {
            log.Println("Bad content encoding %s", contentEncodingStr)
            contentEncoding, ok = BINCONST.ContentEncoding["bytes"]
        }        
        content, ok = response.Content()
    } else if len(response.DynMap.Map) > 0 {
        content, err = response.DynMap.MarshalJSON()
        if err != nil {
            return 0, err
        }
    }

    binary.Write(writer, binary.BigEndian, contentEncoding)
    contentLength = int32(len(content))
    binary.Write(writer, binary.BigEndian, contentLength)
    if contentLength > 0{
        writer.Write(content)
    }

    return 4+headerLength + int(contentLength), nil
}


func (this *BinProtocol) WriteRequest(request *Request, writer io.Writer) (int, error) {
    
    paramEncoding := int8(0); //json
    params, err := request.Params().MarshalJSON()
    if err != nil {
        return 0, err
    }

    headerLength := 
        2 + //txnId length
        len(request.TxnId()) +
        1 + //txn accept
        1 + //method
        2 + //uri length
        len(request.Uri()) +
        1 + //param encoding
        len(params) +
        2 + //content encoding
        4 //content lenght

    //write the header length
    err = binary.Write(writer, binary.BigEndian, int32(headerLength))

    //txn id
    _, err = writeString(writer, request.TxnId())
    if err != nil {
        return 0, err
    }

    //txn status
    txnAccept, ok := BINCONST.TxnAccept[request.TxnAccept()]
    if !ok {
        return 0, fmt.Errorf("Bad TXN Accept %s", request.TxnAccept())
    }
    err = binary.Write(writer, binary.BigEndian, txnAccept)
    if err != nil {
        return 0, err
    }

    method, ok := BINCONST.Method[request.Method()]
    if !ok {
        return 0, fmt.Errorf("Bad Method %s", request.Method())
    }
    err = binary.Write(writer, binary.BigEndian, method)
    if err != nil {
        return 0, err
    }

    _, err = writeString(writer, request.Uri())
    if err != nil {
        return 0, err
    }

    err = binary.Write(writer, binary.BigEndian, paramEncoding)
    if err != nil {
        return 0, err
    }
    _, err = writeByteArray(writer, params)
    if err != nil {
        return 0, err
    }

    contentLength := int32(0)
    contentEncodingStr, contentSet := request.ContentEncoding()

    if contentSet {
        contentEncoding, ok := BINCONST.ContentEncoding[contentEncodingStr]
        if !ok {
            log.Printf("Bad content encoding %s", contentEncodingStr)
            contentEncoding, ok = BINCONST.ContentEncoding["bytes"]
        }        
        binary.Write(writer, binary.BigEndian, contentEncoding)
        content, ok := request.Content()
        contentLength = int32(len(content))
        binary.Write(writer, binary.BigEndian, contentLength)
        if contentLength > 0{
            writer.Write(content)
        }
    } else {
        contentEncoding, _ := BINCONST.ContentEncoding["bytes"]
        binary.Write(writer, binary.BigEndian, contentEncoding)
        content, _ := request.Content()
        contentLength = int32(len(content))
        binary.Write(writer, binary.BigEndian, contentLength)
        if contentLength > 0{
            writer.Write(content)
        }
    }
    return 4+headerLength + int(contentLength), nil
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
func writeByteArray(writer io.Writer, bytes []byte) (int, error) {
    length := int16(len(bytes))
    err := binary.Write(writer, binary.BigEndian, length)
    if err != nil {
        return 0, err
    }
    l, err := writer.Write(bytes)
    return l+2, err
}

//writes a length prefixed utf8 string 
func writeString(writer io.Writer, str string) (int, error) {
    l,err := writeByteArray(writer, []byte(str))
    return l,err
}