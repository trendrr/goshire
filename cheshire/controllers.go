package cheshire

import (
    "github.com/trendrr/goshire/dynmap"
)


// a Controller object
type Controller interface {
    Config() *ControllerConfig
    HandleRequest(*Txn)
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

    //Call the filters.
    for _, filter := range this.Filters {
        f, ok := filter.(FilterAdvanced)
        if ok {
            f.BeforeWrite(response, this)
        }
    }

    c, err := this.Writer.Write(response)
    //Call the filters.
    for _, filter := range this.Filters {
        f, ok := filter.(FilterAdvanced)
        if ok {
            f.AfterWrite(response, this)
        }
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

// A default ping controller
func PingController(txn *Txn) {
    // log.Printf("PING! %s", request.Strest.Params)
    response := NewResponse(txn)
    response.Put("data", "PONG")
    // log.Printf("Sending REsponse: %s", response.TxnId())
    txn.Write(response)
}
