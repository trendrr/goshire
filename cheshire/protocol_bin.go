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
    "json", //0
    "msgpack", //1
}

var CONTENT_ENCODING = []string{
    "string", //0
    "bytes", //1
    "json", //2
    "msqpack", //3
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

    //The fields decoded from the hello
    Hello *dynmap.DynMap
}

func (this *BinDecoder) DecodeHello() (*dynmap.DynMap, error) {
    log.Println("DECODE HELLO")
    //read the hello
    helloEncoding := int8(0)
    err := binary.Read(this.reader, binary.BigEndian, &helloEncoding)
    if err != nil {
        return nil, err
    }

    hello, err := ReadByteArray(this.reader)
    if err != nil {
        log.Print(err)
        //TODO: Send bad hello.
        return nil, err
    }
    log.Printf(" HELLO %s", hello)
    this.Hello = dynmap.New()
    err = this.Hello.UnmarshalJSON(hello)
    if err != nil {
        log.Print(err)
        //TODO: Send bad hello.
        return nil, err
    }
    return this.Hello, nil
}    

    //Decode the next response from the reader
func (this *BinDecoder) DecodeResponse() (*Response, error) {
    
    txnId, err := ReadString(this.reader)
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
    statusMessage, err := ReadString(this.reader)
    if err != nil {
        return nil, err
    }
    

    //params
    paramEncoding := int8(0)
    err = binary.Read(this.reader, binary.BigEndian, &paramEncoding)
    if err != nil {
        return nil, err
    }
    if int(paramEncoding) >= len(PARAM_ENCODING) {
        return nil, fmt.Errorf("paramEncoding too large %d", paramEncoding)
    }

    paramsLength := int32(0)
    err = binary.Read(this.reader, binary.BigEndian, &paramsLength)
    if err != nil {
        return nil, err
    }
    // log.Printf("contentLentgh %d", contentLength)

    paramsArray := make([]byte, paramsLength)
    
    if paramsLength > 0 {
        _, err = io.ReadAtLeast(this.reader, paramsArray, int(paramsLength))
        if err != nil {
            return nil, err
        }
    }

    params, err := ParseParams(paramEncoding, paramsArray)
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
        DynMap : *params,
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

// read a shard request from the socket.
func (this *BinDecoder) DecodeShardRequest() (*ShardRequest, error) {
    //sharding..
    partition := int16(0)
    err := binary.Read(this.reader, binary.BigEndian, &partition)
    if err != nil {
        return nil, err
    }

    shardkey, err := ReadString(this.reader)
    if err != nil {
        return nil, err
    }

    revision := int64(0)
    err = binary.Read(this.reader, binary.BigEndian, &revision)
    if err != nil {
        return nil, err
    }

    s := &ShardRequest{
        Partition : int(partition),
        Key : shardkey,
        Revision : revision,
    }

    return s, nil
}

//decode the next request from the reader
func (this *BinDecoder) DecodeRequest() (*Request, error) {

    //shard header
    shard, err := this.DecodeShardRequest()
    if err != nil {
        return nil, err
    }

    //txn id
    txnId, err := ReadString(this.reader)
    if err != nil {
        return nil, err
    }

    //txn accept
    txnAccept := int8(0)
    err = binary.Read(this.reader, binary.BigEndian, &txnAccept)
    if err != nil {
        return nil, err
    }
    if int(txnAccept) >= len(TXN_ACCEPT) {
        return nil, fmt.Errorf("TxnAccept too large %d", txnAccept)
    }

    //method
    method := int8(0)
    err = binary.Read(this.reader, binary.BigEndian, &method)
    if err != nil {
        return nil, err
    }
    if int(method) >= len(METHOD) {
        return nil, fmt.Errorf("Method too large %d", method)
    }

    //uri
    uri, err := ReadString(this.reader)
    if err != nil {
        return nil, err
    }

    //params
    paramEncoding := int8(0)
    err = binary.Read(this.reader, binary.BigEndian, &paramEncoding)
    if err != nil {
        return nil, err
    }
    if int(paramEncoding) >= len(PARAM_ENCODING) {
        return nil, fmt.Errorf("paramEncoding too large %d", paramEncoding)
    }

    paramsArray, err := ReadByteArray32(this.reader)
    if err != nil {
        return nil, err
    }
    params, err := ParseParams(paramEncoding, paramsArray)
    if err != nil {
        return nil, err
    }
    
    //content
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
        Shard : shard,
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
func (this *BinProtocol) WriteHello(writer io.Writer, hello *dynmap.DynMap) error {
    err := binary.Write(writer, binary.BigEndian, BINCONST.ParamEncoding["json"])
    if err != nil {
        return err
    }

    hello.PutIfAbsent("v", StrestVersion)
    hello.PutIfAbsent("useragent", "golang")
    b, err := hello.MarshalJSON()
    if err != nil {
        return err
    }
    _, err = WriteString(writer, string(b))
    return err
}

func (this *BinProtocol) NewDecoder(reader io.Reader) Decoder {
    dec := &BinDecoder{
        reader : reader,
    } 
    return dec
}


//write out the shard request.
func (this *BinProtocol) WriteShardRequest(s *ShardRequest, writer io.Writer) error {
    if s == nil {
        return nil
    }

    err := binary.Write(writer, binary.BigEndian, int16(s.Partition))
    if err != nil {
        return err
    }
    _, err = WriteString(writer, s.Key)
    if err != nil {
        return err
    }
    
    err = binary.Write(writer, binary.BigEndian, s.Revision)
    if err != nil {
        return err
    }

    return nil
}

func (this *BinProtocol) WriteResponse(response *Response, writer io.Writer) (int, error) {
    //txn id
    _, err := WriteString(writer, response.TxnId())
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

    _, err = WriteString(writer, response.StatusMessage())
    if err != nil {
        return 0, err
    }

    //params
    paramEncoding := int8(0); //json
    params, err := response.DynMap.MarshalJSON()
    if err != nil {
        return 0, err
    }
    err = binary.Write(writer, binary.BigEndian, paramEncoding)
    if err != nil {
        return 0, err
    }

    paramsLength := int32(len(params))
    err = binary.Write(writer, binary.BigEndian, paramsLength)
    if err != nil {
        return 0, err
    }
    _, err = writer.Write(params)
    if err != nil {
        return 0, err
    }

    //content
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
    } 
    binary.Write(writer, binary.BigEndian, contentEncoding)
    contentLength = int32(len(content))
    binary.Write(writer, binary.BigEndian, contentLength)
    if contentLength > 0{
        writer.Write(content)
    }

    return int(contentLength), nil
}


func (this *BinProtocol) WriteRequest(request *Request, writer io.Writer) (int, error) {
    
    err := this.WriteShardRequest(request.Shard, writer)
    if err != nil {
        return 0, err
    }

    //txn id
    _, err = WriteString(writer, request.TxnId())
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

    _, err = WriteString(writer, request.Uri())
    if err != nil {
        return 0, err
    }

    //params
    paramEncoding := int8(0); //json
    params, err := request.Params().MarshalJSON()
    if err != nil {
        return 0, err
    }
    err = binary.Write(writer, binary.BigEndian, paramEncoding)
    if err != nil {
        return 0, err
    }
    _, err = WriteByteArray32(writer, params)
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
        _, err = WriteByteArray32(writer, content)
        if err != nil {
            return 0, err
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
    return int(contentLength), nil
}   


// Reads a length prefixed byte array
// it assumes the first 
func ReadByteArray(reader io.Reader) ([]byte, error) {
    length := int16(0)
    err := binary.Read(reader, binary.BigEndian, &length)
    if err != nil {
        return nil, err
    }

    if length < 0 {
        log.Printf("Error length is %d", length)
        return nil, fmt.Errorf("Length is negative!")
    }

    bytes := make([]byte, length)
    _, err = io.ReadAtLeast(reader, bytes, int(length))
    return bytes, err
}

// Reads a length prefixed byte array
// it assumes the first 
func ReadByteArray32(reader io.Reader) ([]byte, error) {
    length := int32(0)
    err := binary.Read(reader, binary.BigEndian, &length)
    if err != nil {
        return nil, err
    }

    if length < 0 {
        log.Printf("Error length is %d", length)
        return nil, fmt.Errorf("Length is negative!")
    }

    bytes := make([]byte, length)
    _, err = io.ReadAtLeast(reader, bytes, int(length))
    return bytes, err
}
// copies a length prefixed byte array from the src to the dest.
func CopyByteArray(dest io.Writer, src io.Reader) error {
    length := int16(0)
    err := binary.Read(src, binary.BigEndian, &length)
    if err != nil {
        return err
    }
    err = binary.Write(dest, binary.BigEndian, length)
    if err != nil {
        return err
    }
    err = CopyN(dest, src, int64(length))
    return err
}

// copies a byte array with a int32 length 
func CopyByteArray32(dest io.Writer, src io.Reader) error {
    length := int32(0)
    err := binary.Read(src, binary.BigEndian, &length)
    if err != nil {
        return err
    }
    err = binary.Write(dest, binary.BigEndian, length)
    if err != nil {
        return err
    }
    err = CopyN(dest, src, int64(length))
    return err
}

// Copies N bytes to the writer from the reader,
// hopefully faster then the fucked up way in the stdlib
func CopyN(dst io.Writer, src io.Reader, bytes int64) (err error) {

    // remaining := bytes
    // bufsize := 1024
    
    // buf := make([]byte, 1024)
    // for remaining > 0 {
    //     if 

    //     nr, er := src.Read(buf)
    //     if nr > 0 {
    //         nw, ew := dst.Write(buf[0:nr])
    //         if nw > 0 {
    //             written += int64(nw)
    //         }
    //         if ew != nil {
    //             err = ew
    //             break
    //         }
    //         if nr != nw {
    //             err = fmt.Errorf("Short write error")
    //             break
    //         }
    //     }
    //     if er == io.EOF {
    //         break
    //     }
    //     if er != nil {
    //         err = er
    //         break
    //     }
    // }
    // return err

    _, err = io.CopyN(dst, src, bytes)
    return err
}

//Reads a length prefixed utf8 string 
func ReadString(reader io.Reader) (string, error) {
    b, err := ReadByteArray(reader)
    if err != nil {
        return "", err
    }
    return string(b), nil
}

//writes a length prefixed byte array
func WriteByteArray(writer io.Writer, bytes []byte) (int, error) {
    length := int16(len(bytes))
    err := binary.Write(writer, binary.BigEndian, length)
    if err != nil {
        return 0, err
    }
    l, err := writer.Write(bytes)
    return l+2, err
}

//writes a length int32 prefixed byte array
func WriteByteArray32(writer io.Writer, bytes []byte) (int, error) {
    length := int32(len(bytes))
    err := binary.Write(writer, binary.BigEndian, length)
    if err != nil {
        return 0, err
    }
    l, err := writer.Write(bytes)
    return l+4, err
}

//writes a length prefixed utf8 string 
func WriteString(writer io.Writer, str string) (int, error) {
    l,err := WriteByteArray(writer, []byte(str))
    return l,err
}

type Flusher interface {
    Flush() error
}
