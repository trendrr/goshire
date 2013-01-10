package strest

import (
    "net/http"
    "encoding/json"
    "log"
    "fmt"
    "sync"
)

type HttpConnection struct {
    Writer http.ResponseWriter
    request *http.Request
    ServerConfig *ServerConfig
    headerWritten sync.Once
}

func (conn *HttpConnection) Write(response *Response) (int, error) {

    json, err := json.Marshal(response)
    if err != nil {
        //TODO: uhh, do something..
        log.Print(err)
    }
    conn.headerWritten.Do(func() {
        conn.Writer.Header().Set("Content-Type", "application/json")
        conn.Writer.WriteHeader(response.StatusCode())
    })
    conn.Writer.Write(json)
    conn.Writer.Write([]byte("\n"))
    v,ok := conn.Writer.(http.Flusher)
    if ok{
        v.Flush()    
    }

    return 200, err
}

type httpHandler struct {
    serverConfig *ServerConfig
    
}

func (this *httpHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
    controller := this.serverConfig.Router.Match(req.URL.Path)
    
    //check if controller is the special websocket controller.
    wc, isWebsocket := controller.(WebsocketController)
    if isWebsocket {
        wc.Handler.ServeHTTP(writer, req)
        log.Print("WEBSOCKET!")
        return;
    }


    //we are already in a go routine, so no need to start another one.
    request := ToStrestRequest(req)

    //TODO: filters here..
    conn := HttpConnection{Writer:writer, request:req, ServerConfig:this.serverConfig}
    
    controller.HandleRequest(request, &conn)
} 

func ToStrestRequest(req *http.Request) (*Request) {
    var request = new(Request)
    request.Strest.Uri = req.URL.Path
    request.Strest.Method = req.Method
    request.Strest.Txn.Id = req.Header.Get("Strest-Txn-Id")
    request.Strest.Txn.Accept = req.Header.Get("Strest-Txn-Accept")
    if len(request.Strest.Txn.Accept) == 0 {
        request.Strest.Txn.Accept = "single"
    }

    //parse the query params
    values := req.URL.Query()
    params := map[string]interface{}{}

    for k := range values { 
        var v = values[k]
        if len(v) == 1 {
            params[k] = v[0]
        } else {
            params[k] = v
        }
    }
    
    return request
}

func HttpListen(port int, serverConfig *ServerConfig) error {
    handler := &httpHandler{serverConfig}

    log.Println("HTTP Listener on port: ", port)
    http.ListenAndServe(fmt.Sprintf(":%d", port), handler)
    return nil
}

