package cheshire

import (
    "github.com/trendrr/goshire/dynmap"
    "encoding/json"
    "fmt"
    "bytes"
    "unicode/utf8"
    "sync/atomic"
    // "log"
)

// what Strest protocol version we are using.
const StrestVersion = float32(2)

var strestId int64 = int64(0)



// Special Param Names
const (
    //The partition val (an integer from 0 to TotalPartitions)
    PARAM_SHARD_PARTITION = "_p"

    PARAM_SHARD_KEY = "_k"

    // The version of the router table
    PARAM_SHARD_REVISION = "_v"

    //The query type.
    // This defines how the request can be handled by the router.
    // Possible values:
    // single : return a single result (the first response received)
    // all : (default) return values for all servers, will make an effort to retry on failure, but will generally return error results.
    // all_q : return values for all servers (queue requests if needed, retry until response).  This would typically be for posting
    // none_q : returns success immediately, queues the request and make best effort to ensure it is delivered (TODO)
    PARAM_SHARD_QUERY_TYPE = "_qt"
)

//create a new unique strest txn id
func NewTxnId() string {
    id := atomic.AddInt64(&strestId, int64(1))
    return fmt.Sprintf("%d", id)
}

// Standard STREST request.
// See protocol spec https://github.com/trendrr/strest-server/wiki/STREST-Protocol-Spec
type Request struct {
    version float32
    userAgent string
    txnId string
    txnAccept string
    uri string
    method string
    params *dynmap.DynMap
    contentEncoding string
    content []byte
    Shard *ShardRequest
}

type ShardRequest struct {
    Partition int
    Key string
    Revision int64
}

// Makes it simple to create a new response from
// anything implementing this interface
type RequestTxnId interface {
    TxnId() string
}
//creates a new request from a dynmap
func NewRequestDynMap(mp *dynmap.DynMap) *Request {
    request := &Request{
        // version : mp.MustFloat32("strest.v", mp.StrestVersion), //TODO
        version : StrestVersion,
        uri : mp.MustString("strest.uri", ""),
        method : mp.MustString("strest.method", "GET"),
        txnId : mp.MustString("strest.txn.id", ""),
        txnAccept : mp.MustString("strest.txn.accept", "single"),
        params : mp.MustDynMap("strest.params", dynmap.New()),
    }

    shardMp, ok := mp.GetMap("strest.shard")
    if ok {
        request.Shard = &ShardRequest{
            Partition : request.params.MustInt(PARAM_SHARD_PARTITION, -1),
            Key : request.params.MustString(PARAM_SHARD_KEY, ""),
            Revision : 
        }

    } else {
        request.Shard = &ShardRequest{
            Partition : request.params.MustInt(PARAM_SHARD_PARTITION, -1),
            Key : request.params.MustString(PARAM_SHARD_KEY, ""),
            Revision : request.params.MustInt64(PARAM_SHARD_REVISION, int64(-1)),
        }
    }
    return request
}

// Create a new request object.
// Values are all set to defaults
func NewRequest(uri, method string) *Request {
    request := &Request{
        version : StrestVersion,
        uri : uri,
        method : method,
        txnAccept : "single",
        params : dynmap.New(),
        Shard : &ShardRequest{
            Partition : -1,
        },
    }
    return request
}

// Creates a new dynmap with all the appropriate fields
// This is here for compatibility, but is rather inefficient
func (this *Request) ToDynMap() *dynmap.DynMap {
    request := dynmap.New()
    request.PutWithDot("strest.params", this.params)
    request.PutWithDot("strest.v", this.version)
    request.PutWithDot("strest.uri", this.uri)
    request.PutWithDot("strest.method", this.method)
    request.PutWithDot("strest.txn.accept", this.txnAccept)
    return request
}



func (this *Request) SetContent(contentEncoding string, content []byte) {
    this.contentEncoding = contentEncoding
    this.content = content
}

func (this *Request) ContentIsSet() bool {
    return len(this.contentEncoding) != 0
}

//returns the current content encoding, 
func (this *Request) ContentEncoding() (string, bool) {
    if !this.ContentIsSet() {
        return this.contentEncoding, false
    }
    return this.contentEncoding, true
}

func (this *Request) Content() ([]byte, bool) {
    if !this.ContentIsSet() {
        return this.content, false
    }
    return this.content, true
}

func (this *Request) Method() string {
    return this.method
}

func (this *Request) SetMethod(method string) {
    this.method = method
}

func (this *Request) Uri() string {
    return this.uri
}

func (this *Request) SetUri(uri string) {
    this.uri = uri
}

func (this *Request) Params() *dynmap.DynMap {
    return this.params
}

func (this *Request) SetParams(params *dynmap.DynMap) {
    this.params = params
}

//return the txnid.
func (this *Request) TxnId() string {
    return this.txnId
}

func (this *Request) SetTxnId(id string) {
    this.txnId = id
}

func (this *Request) TxnAccept() string {
    return this.txnAccept
}

//Set to either "single" or "multi"
func (this *Request) SetTxnAccept(accept string) {
    this.txnAccept = accept
}

//This request will accept multiple responses
func (this *Request) SetTxnAcceptMulti() {
    this.SetTxnAccept("multi")
}

//This request will only accept a single response
func (this *Request) SetTxnAcceptSingle() {
    this.SetTxnAccept("single")
}
func (this *Request) StrestVersion() float32 {
    return this.version
}
func (this *Request) UserAgent() string {
    return this.userAgent
}

func (this *Request) MarshalJSON() ([]byte, error) {
    //handle the sharding shit
    if this.Shard != nil {
        if this.Shard.Partition >=0 {
            this.Params().PutIfAbsent(PARAM_SHARD_PARTITION, this.Shard.Partition)
        }
        if len(this.Shard.Key) > 0 {
            this.Params().PutIfAbsent(PARAM_SHARD_KEY, this.Shard.Key)
        }
        this.Params().PutIfAbsent(PARAM_SHARD_REVISION, this.Shard.Revision)
    }

    bytes, err := this.Params().MarshalJSON()
    if err != nil {
        return bytes, err
    }
    
    //need to encode since this might contain " or \
    ua, err := JSONEncodeString(this.UserAgent())
    if err != nil {
        return nil, err
    }

    json := fmt.Sprintf(
        "{ \"strest\" : {\"v\" : %f, \"user-agent\":%s, \"txn\":{\"id\":\"%s\",\"accept\":\"%s\"},\"uri\":\"%s\",\"method\" : \"%s\", \"params\" : %s}}",
        this.StrestVersion(),
        ua,
        this.TxnId(),
        this.TxnAccept(),
        this.Uri(),
        this.Method(),
        string(bytes),
    )
    return []byte(json), err
}


// Standard STREST response
// See protocol spec https://github.com/trendrr/strest-server/wiki/STREST-Protocol-Spec
type Response struct {
    dynmap.DynMap
    txnId string
    txnStatus string
    statusCode int
    statusMessage string
    contentEncoding string
    content []byte
}


//creates a new response from a dynmap
func NewResponseDynMap(mp *dynmap.DynMap) *Response {
    strest := mp.MustDynMap("strest", dynmap.New())
    status := mp.MustDynMap("status", dynmap.New())
    mp.Remove("strest")
    mp.Remove("status")

    response := &Response{
        DynMap : *mp,
        txnId : strest.MustString("txn.id", ""),
        txnStatus : strest.MustString("txn.status", "completed"),
        statusCode : status.MustInt("code", 200),
        statusMessage : status.MustString("message", "OK"),
    }
    return response
}

func (this *Response) SetContent(contentEncoding string, content []byte) {
    this.contentEncoding = contentEncoding
    this.content = content
}

func (this *Response) ContentIsSet() bool {
    return len(this.contentEncoding) != 0
}

//returns the current content encoding, 
func (this *Response) ContentEncoding() (string, bool) {
    if !this.ContentIsSet() {
        return this.contentEncoding, false
    }
    return this.contentEncoding, true
}

func (this *Response) Content() ([]byte, bool) {
    if !this.ContentIsSet() {
        return this.content, false
    }
    return this.content, true
}

func (this *Response) TxnId() string {
    return this.txnId
}

func (this *Response) SetTxnId(id string) {
    this.txnId = id
}

func (this *Response) TxnStatus() string {
    return this.txnStatus
}

// completed or continue
func (this *Response) SetTxnStatus(status string) {
    this.txnStatus = status
}

func (this *Response) SetTxnComplete() {
    this.txnStatus = "completed"
}

func (this *Response) TxnComplete() bool {
    return this.txnStatus == "completed"
}

func (this *Response) SetTxnContinue() {
    this.txnStatus = "continue"
}

func (this *Response) TxnContinue() bool {
    return this.txnStatus == "continue"
}

func (this *Response) SetStatus(code int, message string) {
    this.SetStatusCode(code)
    this.SetStatusMessage(message)
}

func (this *Response) StatusCode() int {
    return this.statusCode
}

func (this *Response) SetStatusCode(code int) {
    this.statusCode = code
}

func (this *Response) StatusMessage() string {
    return this.statusMessage 
}

func (this *Response) SetStatusMessage(message string) {
    this.statusMessage = message
}

func (this *Response) StrestVersion() float32 {
    return StrestVersion
}

func (this *Response) ToDynMap() *dynmap.DynMap {
    //TODO?

    return &this.DynMap
}

func (this *Response) MarshalJSON() ([]byte, error) {
    bytes, err := json.Marshal(this.Map)
    if err != nil {
        return bytes, err
    }
    comma := ","
    if len(this.Map) == 0 {
        comma = ""
    }

    msg, err := JSONEncodeString(this.StatusMessage())
    if err != nil {
        return nil, err 
    }
    json := fmt.Sprintf(
        "{ \"status\": {\"code\": %d, \"message\" : %s }, \"strest\" : {\"txn\":{\"id\":\"%s\",\"status\":\"%s\"},\"v\":%f}%s %s",
        this.StatusCode(),
        msg,
        this.TxnId(),
        this.TxnStatus(),
        this.StrestVersion(),
        comma,
        string(bytes[1:]), //skip the first byte as it is the '{'
    )

    return []byte(json), err
}

// Create a new response object.
// Values are all set to defaults

// We keep this private scope, so external controllers never use it directly
// they should all use request.NewResponse
func newResponse() *Response {
    response := &Response{
        DynMap : *dynmap.NewDynMap(),
        statusMessage : "OK",
        statusCode : 200,
        txnStatus : "completed",
    }
    return response
}


var hex = "0123456789abcdef"

// Json quotes and escapes a string.
// Modified and taken from the json encoding standard lib
func JSONEncodeString(s string) (string, error){
    buf := new(bytes.Buffer)
    buf.WriteByte('"')
    start := 0
    for i := 0; i < len(s); {
        if b := s[i]; b < utf8.RuneSelf {
            if 0x20 <= b && b != '\\' && b != '"' && b != '<' && b != '>' {
                i++
                continue
            }
            if start < i {
                buf.WriteString(s[start:i])
            }
            switch b {
            case '\\', '"':
                buf.WriteByte('\\')
                buf.WriteByte(b)
            case '\n':
                buf.WriteByte('\\')
                buf.WriteByte('n')
            case '\r':
                buf.WriteByte('\\')
                buf.WriteByte('r')
            default:
                // This encodes bytes < 0x20 except for \n and \r,
                // as well as < and >. The latter are escaped because they
                // can lead to security holes when user-controlled strings
                // are rendered into JSON and served to some browsers.
                buf.WriteString(`\u00`)
                buf.WriteByte(hex[b>>4])
                buf.WriteByte(hex[b&0xF])
            }
            i++
            start = i
            continue
        }
        c, size := utf8.DecodeRuneInString(s[i:])
        if c == utf8.RuneError && size == 1 {
            return s, &json.InvalidUTF8Error{s}
        }
        i += size
    }
    if start < len(s) {
        buf.WriteString(s[start:])
    }
    buf.WriteByte('"')
    return string(buf.Bytes()), nil
}