package cheshire

import (
	"encoding/json"
	"fmt"
	"github.com/trendrr/goshire/dynmap"
	"log"
	"net/http"
	// "net/url"
	"sync"
	// "bytes"
)

type HttpWriter struct {
	Writer        http.ResponseWriter
	HttpRequest   *http.Request
	Request       *Request
	ServerConfig  *ServerConfig
	headerWritten sync.Once
}

func (this *HttpWriter) Type() string {
	return "http"
}

func (conn *HttpWriter) Write(response *Response) (int, error) {
	bytes := 0
	json, err := json.Marshal(response)
	if err != nil {
		//TODO: uhh, do something..
		log.Print(err)
	}
	conn.headerWritten.Do(func() {
		conn.Writer.Header().Set("Content-Type", "application/json")
		conn.Writer.WriteHeader(response.StatusCode())
	})
	b, err := conn.Writer.Write(json)
	if err != nil {
		return bytes, err
	}
	bytes += b
	b, err = conn.Writer.Write([]byte("\n"))
	if err != nil {
		return bytes, err
	}
	bytes += b

	flusher, ok := conn.Writer.(http.Flusher)
	if !ok {
		return bytes, fmt.Errorf("Wrong type in http writer!")
	}
	flusher.Flush()
	return bytes, err
}

type httpHandler struct {
	serverConfig *ServerConfig
}

// Implement this interface for a controller to skip the normal cheshire life cycle
// This should be only used in special cases (static file serving, websockets, ect)
// controllers that implement this interface will skip the HandleRequest function alltogether
type HttpHijacker interface {
	HttpHijack(writer http.ResponseWriter, req *http.Request, serverConfig *ServerConfig)
}

func (this *httpHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	controller := this.serverConfig.Router.Match(req.Method, req.URL.Path)

	//check if controller is the special HttpHijacker.
	h, hijack := controller.(HttpHijacker)
	if hijack {
		h.HttpHijack(writer, req, this.serverConfig)
		return
	}

	//we are already in a go routine, so no need to start another one.
	request := ToStrestRequest(req)

	conn := &HttpWriter{
		Writer:       writer,
		HttpRequest:  req,
		Request:      request,
		ServerConfig: this.serverConfig,
	}
	HandleRequest(request, conn, controller, this.serverConfig)
}

func ToStrestRequest(req *http.Request) *Request {
	
	//print out the http request.
	// log.Println("*******************")
	// var doc bytes.Buffer
	// req.Write(&doc)
	// log.Println(doc.String())
	// log.Println("*******************")

	var request = NewRequest(req.URL.Path, req.Method)
	request.SetUri(req.URL.Path)
	request.SetMethod(req.Method)
	request.SetTxnId(req.Header.Get("Strest-Txn-Id"))
	request.SetTxnAccept(req.Header.Get("Strest-Txn-Accept"))
	if len(request.TxnAccept()) == 0 {
		request.SetTxnAccept("single")
	}

	err := req.ParseForm()
	if err != nil {
		log.Printf("Error parsing form values: %s", err)
	}
	params := dynmap.New()
	err = params.UnmarshalURLValues(req.Form)
	if err != nil {
		log.Printf("Error parsing form values: %s", err)	
	}
	request.SetParams(params)
	return request
}


func HttpListen(port int, serverConfig *ServerConfig) error {
	handler := &httpHandler{serverConfig}

	log.Println("HTTP Listener on port: ", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), handler)
}
