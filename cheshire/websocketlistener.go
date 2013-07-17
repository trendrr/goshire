package cheshire

import (
	"bufio"
	"code.google.com/p/go.net/websocket"
	"io"
	"log"
	"net/http"
	"sync"
)

type WebsocketWriter struct {
	conn       *websocket.Conn
	writerLock sync.Mutex
}

func (this *WebsocketWriter) Write(response *Response) (int, error) {
	defer this.writerLock.Unlock()
	this.writerLock.Lock()

	bytes, err := JSON.WriteResponse(response, this.conn)
	return bytes, err
}

func (this *WebsocketWriter) Type() string {
	return "websocket"
}

type WebsocketController struct {
	Conf         *ControllerConfig
	Handler      websocket.Handler
	serverConfig *ServerConfig
}

func (wc WebsocketController) Config() *ControllerConfig {
	return wc.Conf
}
func (wc WebsocketController) HandleRequest(txn *Txn) {
	//do nothing, this should never be called. 
	log.Println("ERROR! Websocket Controller HandleRequest should never execute")
}

func NewWebsocketController(route string, config *ServerConfig) *WebsocketController {
	ws := &WebsocketController{
		Conf : NewControllerConfig(route),
		serverConfig : config,
	}
	ws.Handler = websocket.Handler(func(con *websocket.Conn) { ws.HandleWCConnection(con) })
	return ws
}

// implements the HttpHijacker interface so we can handle the request directly.
func (this *WebsocketController) HttpHijack(writer http.ResponseWriter, req *http.Request, serverConfig *ServerConfig) {
	this.Handler.ServeHTTP(writer, req)
}

func (this *WebsocketController) HandleWCConnection(ws *websocket.Conn) {
	// Uhh, guessing we are already in a go routine..
	log.Print("CONNECT!")

	defer ws.Close()
	// log.Print("CONNECT!")
	// conn.writer = bufio.NewWriter(conn.conn)


	// dec := json.NewDecoder(bufio.NewReader(conn.conn))
	dec := JSON.NewDecoder(bufio.NewReader(ws))
	writer := &WebsocketWriter{conn: ws}
	for {
		req, err := dec.DecodeRequest()

		if err == io.EOF {
			log.Print(err)
			break
		} else if err != nil {
			log.Print(err)
			break
		}
		
		controller := this.serverConfig.Router.Match(req.Method(), req.Uri())
		go HandleRequest(req, writer, controller, this.serverConfig)
	}
	log.Print("DISCONNECT!")
}
