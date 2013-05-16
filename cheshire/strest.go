package cheshire

import (
	"github.com/trendrr/cheshire-golang/dynmap"
)

// what Strest protocol version we are using.
const StrestVersion = float32(2)

// Standard STREST request.
// See protocol spec https://github.com/trendrr/strest-server/wiki/STREST-Protocol-Spec
type Request struct {
	dynmap.DynMap
}

// Makes it simple to create a new response from
// anything implementing this interface
type RequestTxnId interface {
	TxnId() string
}

// Create a new request object.
// Values are all set to defaults
func NewRequest(uri, method string) *Request {
	request := &Request{*dynmap.NewDynMap()}
	request.PutWithDot("strest.params", dynmap.NewDynMap())
	request.PutWithDot("strest.v", StrestVersion)
	request.PutWithDot("strest.uri", uri)
	request.PutWithDot("strest.method", method)
	request.PutWithDot("strest.txn.accept", "single")
	return request
}

func (this *Request) ToDynMap() *dynmap.DynMap {
	return &this.DynMap
}

func (this *Request) Method() string {
	return this.MustString("strest.method", "")
}

func (this *Request) SetMethod(method string) {
	this.PutWithDot("strest.method", method)
}

func (this *Request) Uri() string {
	return this.MustString("strest.uri", "")
}

func (this *Request) SetUri(uri string) {
	this.PutWithDot("strest.uri", uri)
}

func (this *Request) Params() *dynmap.DynMap {
	m, ok := this.GetDynMap("strest.params")
	if !ok {
		this.PutIfAbsentWithDot("strest.params", dynmap.NewDynMap())
		m, ok = this.GetDynMap("strest.params")
	}
	return m
}

func (this *Request) SetParams(params *dynmap.DynMap) {
	this.PutWithDot("strest.params", params)
}

//return the txnid.
func (this *Request) TxnId() string {
	return this.MustString("strest.txn.id", "")
}

func (this *Request) SetTxnId(id string) {
	this.PutWithDot("strest.txn.id", id)
}

func (this *Request) TxnAccept() string {
	return this.MustString("strest.txn.accept", "single")
}

//Set to either "single" or "multi"
func (this *Request) SetTxnAccept(accept string) {
	this.PutWithDot("strest.txn.accept", accept)
}

//This request will accept multiple responses
func (this *Request) SetTxnAcceptMulti() {
	this.SetTxnAccept("multi")
}

//This request will only accept a single response
func (this *Request) SetTxnAcceptSingle() {
	this.SetTxnAccept("single")
}

// Standard STREST response
// See protocol spec https://github.com/trendrr/strest-server/wiki/STREST-Protocol-Spec
type Response struct {
	dynmap.DynMap
}

func (this *Response) TxnId() string {
	return this.MustString("strest.txn.id", "")
}

func (this *Response) SetTxnId(id string) {
	this.PutWithDot("strest.txn.id", id)
}

func (this *Response) TxnStatus() string {
	return this.MustString("strest.txn.status", "")
}

// complete or continue
func (this *Response) SetTxnStatus(status string) {
	this.PutWithDot("strest.txn.status", status)
}

func (this *Response) SetStatus(code int, message string) {
	this.SetStatusCode(code)
	this.SetStatusMessage(message)
}

func (this *Response) StatusCode() int {
	return this.MustInt("status.code", 200)
}

func (this *Response) SetStatusCode(code int) {
	this.PutWithDot("status.code", code)
}

func (this *Response) StatusMessage() string {
	return this.MustString("status.message", "")
}

func (this *Response) SetStatusMessage(message string) {
	this.PutWithDot("status.message", message)
}

func (this *Response) ToDynMap() *dynmap.DynMap {
	return &this.DynMap
}

// Create a new response object.
// Values are all set to defaults

// We keep this private scope, so external controllers never use it directly
// they should all use request.NewResponse
func newResponse() *Response {
	response := &Response{*dynmap.NewDynMap()}
	response.SetStatusMessage("OK")
	response.SetStatusCode(200)
	response.SetTxnStatus("completed")
	response.PutWithDot("strest.v", StrestVersion)
	return response
}

// A generic cache.
type Cache interface {
	Set(key string, value []byte, expireSeconds int)

	// Sets the value if and only if there is no value associated with this key
	SetIfAbsent(key string, value []byte, expireSeconds int) bool

	// Deletes the value at the requested key
	Delete(key string)

	// Gets the value at the requested key
	Get(key string) ([]byte, bool)

	// Increment the key by val (val is allowed to be negative)
	// in most implementation expireSeconds will be from the first increment, but users should not count on that.
	// if no value is a present it should be added.  
	// If a value is present which is not a number an error should be returned.
	Inc(key string, val int64, expireSeconds int) (int64, error)
}
