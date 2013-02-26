package cheshire

import (
	"log"
		"github.com/trendrr/cheshire-golang/dynmap"
)

// what Strest protocol version we are using.
const StrestVersion = float32(2)

// Standard STREST request.
// See protocol spec https://github.com/trendrr/strest-server/wiki/STREST-Protocol-Spec
type Request struct {
	//params parsed into a dynmap
	Params  dynmap.DynMap `json:"-"`

	Strest struct {
		Version float32                `json:"v"`
		Method  string                 `json:"method"`
		Uri     string                 `json:"uri"`
		Params map[string]interface{} `json:"params"`
		
		Txn     struct {
			Id     string `json:"id"`
			Accept string `json:"accept"`
		} `json:"txn"`
	} `json:"strest"`
}

// Standard STREST response
// See protocol spec https://github.com/trendrr/strest-server/wiki/STREST-Protocol-Spec
type Response struct {
	dynmap.DynMap
}

func (this *Response) TxnId() string {
	return this.GetStringOrDefault("strest.txn.id", "")
}

func (this *Response) SetTxnId(id string) {
	this.PutWithDot("strest.txn.id", id)
}

func (this *Response) TxnStatus() string {
	return this.GetStringOrDefault("strest.txn.status", "")
}

func (this *Response) SetTxnStatus(status string) {
	this.PutWithDot("strest.txn.status", status)
}

func (this *Response) SetStatus(code int, message string) {
	this.SetStatusCode(code)
	this.SetStatusMessage(message)
}

func (this *Response) StatusCode() int {
	return 200 //this.getIntOrDefault("status.code", 200)
}

func (this *Response) SetStatusCode(code int) {
	this.PutWithDot("status.code", code)
}

func (this *Response) StatusMessage() string {
	return this.GetStringOrDefault("status.message", "")
}

func (this *Response) SetStatusMessage(message string) {
	this.PutWithDot("status.message", message)
}

// Create a new response object.
// Values are all set to defaults
func NewResponse(request *Request) *Response {
	response := &Response{*dynmap.NewDynMap()}
	response.SetStatusMessage("OK")
	response.SetStatusCode(200)
	response.SetTxnStatus("completed")
	response.SetTxnId(request.Strest.Txn.Id)
	response.PutWithDot("strest.v", StrestVersion)
	return response
}


func NewErrorResponse(request *Request, code int, message string) *Response{
	response := NewResponse(request)
	response.SetStatus(code, message)
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
}

// Creates a new server config with a default routematcher
func NewServerConfig() *ServerConfig {
	return &ServerConfig{dynmap.NewDynMap(), NewDefaultRouter()}
}

// Registers a controller with the RouteMatcher.  
// shortcut to conf.Router.Register(controller)
func (this *ServerConfig) Register(methods []string, controller Controller) {
	log.Println("Registering: ", methods, " ", controller.Config().Route, " ", controller)
	this.Router.Register(methods, controller)
}

// Configuration for a specific controller.
type ControllerConfig struct {
	Route string
}

func NewControllerConfig(route string) *ControllerConfig {
	return &ControllerConfig{Route: route}
}

// a Controller object
type Controller interface {
	Config() *ControllerConfig
	HandleRequest(*Request, Connection)
}

type DefaultController struct {
	Handlers map[string]func(*Request, Connection)
	Conf     *ControllerConfig
}

func (this *DefaultController) Config() *ControllerConfig {
	return this.Conf
}
func (this *DefaultController) HandleRequest(request *Request, conn Connection) {
	handler := this.Handlers[request.Strest.Method]
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

	def := &DefaultController{Handlers: make(map[string]func(*Request, Connection)), Conf: NewControllerConfig(route)}
	for _, m := range methods {
		def.Handlers[m] = handler
	}
	return def
}

// creates a new controller that will process all method types
func NewControllerAll(route string, handler func(*Request, Connection)) *DefaultController {
	return NewController(route, []string{"ALL"}, handler)
}
