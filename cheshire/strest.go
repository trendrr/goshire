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

	// Strest struct {
	// 	Version float32        `json:"v"`
	// 	Method  string         `json:"method"`
	// 	Uri     string         `json:"uri"`
	// 	Params  *dynmap.DynMap `json:"params"`

	// 	Txn struct {
	// 		Id     string `json:"id"`
	// 		Accept string `json:"accept"`
	// 	} `json:"txn"`
	// } `json:"strest"`
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
	this.SetTxnAccept("multi");
}

//This request will only accept a single response
func (this *Request) SetTxnAcceptSingle() {
	this.SetTxnAccept("single");
}

// Creates a new response based on this request.
// auto fills the txn id
func (this *Request) NewResponse() *Response {
	response := newResponse()
	response.SetTxnId(this.TxnId())
	return response
}


func (this *Request) NewError(code int, message string) *Response {
	response := this.NewResponse()
	response.SetStatus(code, message)
	return response
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


type Connection interface {
	//writes the response to the underlying channel 
	// i.e. either to an http response writer or json socket.
	Write(*Response) (int, error)
}

type RouteMatcher interface {
	// A controller matches the given method, path
	Match(string, string) Controller
	// Registers a controller for the specified methods 
	Register([]string, Controller)
}
type ServerConfig struct {
	*dynmap.DynMap
	Router RouteMatcher
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
	Before(*Request, Connection) bool

	//This is called after the controller is called.
	After(*Request, *Response, Connection)
}

// Configuration for a specific controller.
type ControllerConfig struct {
	Route string
	Filters []ControllerFilter
}

func NewControllerConfig(route string) *ControllerConfig {
	return &ControllerConfig{Route: route}
}

// a Controller object
type Controller interface {
	Config() *ControllerConfig
	HandleRequest(*Request, Connection)
}

// Implements the handle request, does the full filter stack.
func HandleRequest(request *Request, conn Connection, controller Controller, serverConfig ServerConfig) {

	//Handle Global Before filters
	for _,f := range(serverConfig.Filters) {
		ok := f.Before(request, conn)
		if !ok {
			return
		}
	}
	//controller local Before filters
	for _,f := range(controller.Config().Filters) {
		ok := f.Before(request, conn)
		if !ok {
			return
		}	
	}
	controller.HandleRequest(request, conn)
	//TODO: need to get ahold of the response object, if available..


}

type DefaultController struct {
	Handlers map[string]func(*Request, Connection)
	Conf     *ControllerConfig
}

func (this *DefaultController) Config() *ControllerConfig {
	return this.Conf
}
func (this *DefaultController) HandleRequest(request *Request, conn Connection) {
	handler := this.Handlers[request.Method()]
	if handler == nil {
		handler = this.Handlers["ALL"]
	}
	if handler == nil {
		//not found!
		//TODO: method not allowed 
		return
	}
	handler(request, conn)
}

// creates a new controller for the specified route for a specific method types (GET, POST, PUT, ect)
func NewController(route string, methods []string, handler func(*Request, Connection)) *DefaultController {
	// def := new(DefaultController)
	// def.Conf = NewConfig(route)

	def := &DefaultController{
		Handlers: make(map[string]func(*Request, Connection)), 
		Conf: NewControllerConfig(route),
	}
	for _, m := range methods {
		def.Handlers[m] = handler
	}
	return def
}

// creates a new controller that will process all method types
func NewControllerAll(route string, handler func(*Request, Connection)) *DefaultController {
	return NewController(route, []string{"ALL"}, handler)
}