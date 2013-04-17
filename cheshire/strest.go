package cheshire

import (
	"github.com/trendrr/cheshire-golang/dynmap"
	"log"
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
    Delete(key string) bool

    // Gets the value at the requested key
    Get(key string) ([]byte, bool)

    // Increment the key by val (val is allowed to be negative)
    // in most implementation expireSeconds will be from the first increment, but users should not count on that.
    // if no value is a present it should be added.  
    // If a value is present which is not a number an error should be returned.
    Inc(key string, val int64, expireSeconds int) (int64, error)
}

type Writer interface {
	//writes the response to the underlying channel 
	// i.e. either to an http response writer or json socket.
	// implementers should make sure that this method is threadsafe as the 
	// Writer may be shared across go routines.
	Write(*Response) (int, error)

	//What type of writer is this?
	//Examples: http,json,websocket
	Type() string
}

// Represents a single transaction.  This wraps the underlying Writer, and
// allows saving of session state ect.
type Txn struct {
	Request *Request

	//writer should be threadsafe
	Writer Writer

	//Session is not currently threadsafe
	Session *dynmap.DynMap

	//The filters that will be run on this txn
	Filters []ControllerFilter

	//the immutable server config
	ServerConfig *ServerConfig
}

func (this *Txn) Params() *dynmap.DynMap {
	return this.Request.Params()
}

func (this *Txn) TxnId() string {
	return this.Request.TxnId()
}

// Writes a response to the underlying writer.
func (this *Txn) Write(response *Response) (int, error) {
	c, err := this.Writer.Write(response)
	//Call the after filters.
	for _, f := range this.Filters {
		f.After(response, this)
	}
	return c, err
}

//Returns the connection type.
//currently will be one of http,html,json,websocket
func (this *Txn) Type() string {
	return this.Writer.Type()
}

func NewTxn(request *Request, writer Writer, filters []ControllerFilter, serverConfig *ServerConfig) *Txn {
	return &Txn{
		Request:      request,
		Writer:       writer,
		Session:      dynmap.NewDynMap(),
		Filters:      filters,
		ServerConfig: serverConfig,
	}
}

type RouteMatcher interface {
	// A controller matches the given method, path
	Match(string, string) Controller
	// Registers a controller for the specified methods 
	Register([]string, Controller)
}
type ServerConfig struct {
	*dynmap.DynMap
	Router  RouteMatcher
	Filters []ControllerFilter
}

// Creates a new server config with a default routematcher
func NewServerConfig() *ServerConfig {
	return &ServerConfig{
		dynmap.NewDynMap(),
		NewDefaultRouter(),
		make([]ControllerFilter, 0),
	}
}

// Registers a controller with the RouteMatcher.  
// shortcut to conf.Router.Register(controller)
func (this *ServerConfig) Register(methods []string, controller Controller) {
	log.Println("Registering: ", methods, " ", controller.Config().Route, " ", controller)
	this.Router.Register(methods, controller)
}

type ControllerFilter interface {
	//This is called before the Controller is called. 
	//returning false will stop the execution
	Before(*Txn) bool

	//This is called after the controller is called.
	After(*Response, *Txn)
}

// Configuration for a specific controller.
type ControllerConfig struct {
	Route   string
	Filters []ControllerFilter
}

func NewControllerConfig(route string) *ControllerConfig {
	return &ControllerConfig{
		Route:   route,
		Filters: make([]ControllerFilter, 0),
	}
}

// a Controller object
type Controller interface {
	Config() *ControllerConfig
	HandleRequest(*Txn)
}

// Implements the handle request, does the full filter stack.
func HandleRequest(request *Request, conn Writer, controller Controller, serverConfig *ServerConfig) {

	//slice of all the filters
	filters := append(make([]ControllerFilter, 0), serverConfig.Filters...)
	if controller.Config() != nil {
		filters = append(filters, controller.Config().Filters...)
	}

	//wrap the writer in a Txn
	txn := NewTxn(request, conn, filters, serverConfig)

	//controller Before filters
	for _, f := range filters {
		ok := f.Before(txn)
		if !ok {
			return
		}
	}
	controller.HandleRequest(txn)
}

type DefaultController struct {
	Handlers map[string]func(*Txn)
	Conf     *ControllerConfig
}

func (this *DefaultController) Config() *ControllerConfig {
	return this.Conf
}
func (this *DefaultController) HandleRequest(txn *Txn) {
	handler := this.Handlers[txn.Request.Method()]
	if handler == nil {
		handler = this.Handlers["ALL"]
	}
	if handler == nil {
		//not found!
		//TODO: method not allowed 
		return
	}
	handler(txn)
}

// creates a new controller for the specified route for a specific method types (GET, POST, PUT, ect)
func NewController(route string, methods []string, handler func(*Txn)) *DefaultController {
	// def := new(DefaultController)
	// def.Conf = NewConfig(route)

	def := &DefaultController{
		Handlers: make(map[string]func(*Txn)),
		Conf:     NewControllerConfig(route),
	}
	for _, m := range methods {
		def.Handlers[m] = handler
	}
	return def
}

// creates a new controller that will process all method types
func NewControllerAll(route string, handler func(*Txn)) *DefaultController {
	return NewController(route, []string{"ALL"}, handler)
}
