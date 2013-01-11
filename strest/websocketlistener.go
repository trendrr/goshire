package strest

import (
    "code.google.com/p/go.net/websocket"
    "log"
    "io"
    "bufio"
    "encoding/json"
    "net/http"
)

type WebsocketConnection struct {
    conn *websocket.Conn  
    writer *bufio.Writer  
}

func (conn WebsocketConnection) Write(response *Response) (int, error) {
    json, err := json.Marshal(response)
    if err != nil {
        //TODO: uhh, do something..
        log.Print(err)
    }
    // log.Println("writing ", string(json))
    bytes, err := conn.writer.Write(json)
    conn.writer.Flush()
    return bytes, err
}

type WebsocketController struct {
    Conf *ControllerConfig
    Handler websocket.Handler
    serverConfig *ServerConfig
}

func (wc WebsocketController) Config() (*ControllerConfig) {
    return wc.Conf
}
func (wc WebsocketController) HandleRequest(*Request, Connection) {
    //do nothing, this should never be called. 
}

func NewWebsocketController(route string, config *ServerConfig) WebsocketController {
    var wc = new(WebsocketController)
    wc.Conf = NewControllerConfig(route)
    wc.serverConfig = config
    wc.Handler = websocket.Handler(func(ws *websocket.Conn) { wc.HandleWCConnection(ws) }) //use anon function because a method is impossible
    return *wc
}

// implements the HttpHijacker interface so we can handle the request directly.
func (this WebsocketController) HttpHijack(writer http.ResponseWriter, req *http.Request) {
    this.Handler.ServeHTTP(writer, req)
}

func (wc WebsocketController) HandleWCConnection(ws *websocket.Conn) {
    // Uhh, guessing we are already in a go routine..
    log.Print("CONNECT!")
    
    defer ws.Close()
    // log.Print("CONNECT!")
    // conn.writer = bufio.NewWriter(conn.conn)

    dec := json.NewDecoder(bufio.NewReader(ws))
    conn := WebsocketConnection{conn: ws, writer : bufio.NewWriter(ws)}
    for {
        var req Request
        err := dec.Decode(&req)

        if err == io.EOF {
            log.Print(err)
            break
        } else if err != nil {
            log.Print(err)
            break
        }

        log.Print(req)
        log.Print(req.Strest.Uri)
        controller := wc.serverConfig.Router.Match(req.Strest.Uri)

        log.Print("GOT CONTROLLER ")
        log.Print(controller)


        go controller.HandleRequest(&req, conn)
    }
    log.Print("DISCONNECT!")



    // io.Copy(ws, ws)

}