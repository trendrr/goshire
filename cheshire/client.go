package cheshire

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/trendrr/cheshire-golang/dynmap"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var strestId int64 = int64(0)

//create a new unique strest txn id
func NewTxnId() string {
	id := atomic.AddInt64(&strestId, int64(1))
	return fmt.Sprintf("go%d", id)
}

type Client interface {
	// Does a synchronous api call.  times out after the requested timeout.
	// This will automatically set the txn accept to single
	ApiCallSync(req *Request, timeout time.Duration) (*Response, error)
	// Does an api call.
	ApiCall(req *Request, responseChan chan *Response, errorChan chan error)

	//Closes this client
	Close()
}

type HttpClient struct {
	Address string
}

func NewHttpClient(address string) *HttpClient {
	return &HttpClient{
		Address: address,
	}
}

func (this *HttpClient) Close() {
	//do nothing..
}

func (this *HttpClient) ApiCall(req *Request, responseChan chan *Response, errorChan chan error) {
	go func() {
		//TODO we could do something that allows streaming http
		res, err := this.ApiCallSync(req, 4*60*time.Second)
		if err != nil {
			errorChan <- err
		} else {
			responseChan <- res
		}
	}()
}

func (this *HttpClient) ApiCallSync(req *Request, timeout time.Duration) (*Response, error) {
	uri := req.Uri()
	pms, err := req.Params().MarshalURL()
	if err != nil {
		return nil, err
	}
	reqBody := strings.NewReader("")

	if req.Method() == "GET" {
		joiner := "&"
		//add params to the uri
		if !strings.Contains(uri, "?") {
			joiner = "?"
		}
		uri = fmt.Sprintf("%s%s%s", uri, joiner, pms)
	} else {
		reqBody = strings.NewReader(pms)
	}
	url := fmt.Sprintf("http://%s%s", this.Address, uri)
	//convert to an http.Request
	request, err := http.NewRequest(req.Method(), url, reqBody)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(request)

	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	//convert to a strest response2
	var response = &Response{*dynmap.NewDynMap()}
	err = response.UnmarshalJSON(body)
	if err != nil {
		return nil, err
	}
	return response, nil
}

type JsonClient struct {
	Host     string
	Port     int
	PingUri  string
	isClosed bool
	conn     *cheshireConn
	//channel is alerted when the connection is disconnected.
	disconnectChan chan *cheshireConn
	exitChan       chan int
	connectLock    sync.RWMutex
}

//Creates a connects 
func NewJsonClient(host string, port int) (*JsonClient, error) {
	client := &JsonClient{
		Host:           host,
		Port:           port,
		isClosed:       false,
		disconnectChan: make(chan *cheshireConn),
		exitChan:       make(chan int),
		PingUri:        "/ping",
	}
	conn, err := client.createConn()
	if err != nil {
		return nil, err
	}
	client.conn = conn
	go client.eventLoop()

	return client, nil
}

//Close this client.
func (this *JsonClient) Close() {
	this.exitChan <- 1
	log.Println("Send exit message")
}

func (this *JsonClient) createConn() (*cheshireConn, error) {
	defer this.connectLock.Unlock()
	this.connectLock.Lock()
	c, err := newCheshireConn(fmt.Sprintf("%s:%d", this.Host, this.Port), this.disconnectChan, 20*time.Second)
	if err != nil {
		return nil, err
	}

	go c.eventLoop()
	return c, nil
}

//returns the connection.  
// use this rather then access directly from the struct, will
// make it easier to pool connections if we need.
func (this *JsonClient) connection() (*cheshireConn, error) {
	defer this.connectLock.RUnlock()
	this.connectLock.RLock()

	if !this.conn.connected {
		return this.conn, fmt.Errorf("Not connected")
	}
	return this.conn, nil
}

//Attempt to close this connection and make a new connection.
func (this *JsonClient) reconnect(oldconn *cheshireConn) (*cheshireConn, error) {
	if this.conn != oldconn {
		log.Println("Error oldconn is not contained in client for reconnect (%s)", oldconn)
	}
	if oldconn.connectedAt.After(time.Now().Add(5 * time.Second)) {
		//only allow one reconnect attempt per 5 second interval
		//returning the old connection, because this was likely a concurrent reconnect 
		// attempt, and perhaps the previous one was successfull
		log.Println("Skipping reconnect too early")
		return oldconn, nil
	}
	log.Println("Closing old")
	oldconn.Close()

	log.Println("Creating new!")
	con, err := this.createConn()
	if err != nil {
		log.Println("COUldn't create new %s", err)

		return oldconn, err
	}
	this.connectLock.Lock()
	this.conn = con
	this.connectLock.Unlock()

	log.Println("DONE RECONNECT %s", con)
	return con, err
}

func (this *JsonClient) eventLoop() {
	//client event loop pings, and listens for client disconnects.
	c := time.Tick(5 * time.Second)
	defer log.Println("CLOSED!!!!!!!!")
	for !this.isClosed {
		select {
		case <-this.exitChan:
			log.Println("Exiting Client")
			//close all connections
			this.conn.Close()
			this.isClosed = true
			break
		case <-c:
			//ping all the connections.
			log.Println("PING!!!!!!!!!!!!!!")
			_, conn, err := this.doApiCallSync(NewRequest(this.PingUri, "GET"), 10*time.Second)
			if err != nil {
				log.Println("COULDNT PING")
				// uhh should we reconnect?
				this.reconnect(conn)
				log.Println("DONE PINGIGN")
			}
		case conn := <-this.disconnectChan:
			log.Printf("DISCONNECTED %s:%p, attempting reconnect", this.Host, this.Port)
			//reconnect

			//attempt a reconnect immediately.  
			// else will attempt to reconnect at next ping
			// 
			this.reconnect(conn)
		}
	}

}

// Does a synchronous api call.  times out after the requested timeout.
// This will automatically set the txn accept to single
func (this *JsonClient) ApiCallSync(req *Request, timeout time.Duration) (*Response, error) {
	req.SetTxnAccept("single")
	response, _, err := this.doApiCallSync(req, timeout)
	return response, err
}

// Does an api call.
func (this *JsonClient) ApiCall(req *Request, responseChan chan *Response, errorChan chan error) {
	this.doApiCall(req, responseChan, errorChan)
}

func (this *JsonClient) doApiCallSync(req *Request, timeout time.Duration) (*Response, *cheshireConn, error) {

	log.Println("Do api sync")
	defer log.Println("DONE api sync")

	responseChan := make(chan *Response)
	errorChan := make(chan error, 5)
	conn, _ := this.doApiCall(req, responseChan, errorChan)
	select {
	case response := <-responseChan:
		return response, conn, nil
	case err := <-errorChan:
		log.Println("GOT ERROR!")
		return nil, conn, err
	case <-time.After(timeout):
		return nil, conn, fmt.Errorf("Request timeout")
	}
	return nil, conn, fmt.Errorf("Impossible error happened, alert NASA")
}

//does the actual call, returning the connection and the internal request
func (this *JsonClient) doApiCall(req *Request, responseChan chan *Response, errorChan chan error) (*cheshireConn, *cheshireRequest) {
	if req.TxnId() == "" {
		req.SetTxnId(NewTxnId())
	}
	r := newRequest(req, responseChan, errorChan)

	conn, err := this.connection()

	if err != nil {
		log.Println("Sending error %s", err)
		errorChan <- err
	} else if !conn.connected {
		errorChan <- fmt.Errorf("Not connected")
	} else {
		conn.outgoingChan <- r
	}

	return conn, r
}

// Connection to a cheshire server. 
type cheshireConn struct {
	net.Conn
	addr           string
	connected      bool
	readTimeout    time.Duration
	writeTimeout   time.Duration
	incomingChan   chan *Response
	outgoingChan   chan *cheshireRequest
	exitChan       chan int
	disconnectChan chan *cheshireConn
	//map of txnId to request
	requests    map[string]*cheshireRequest
	connectedAt time.Time
}

//wrap a request so we dont lose track of the result channels
type cheshireRequest struct {
	req        *Request
	resultChan chan *Response
	errorChan  chan error
}

func newRequest(req *Request, resultChan chan *Response, errorChan chan error) *cheshireRequest {
	return &cheshireRequest{
		req:        req,
		resultChan: resultChan,
		errorChan:  errorChan,
	}
}

func newCheshireConn(addr string, disconnect chan *cheshireConn, writeTimeout time.Duration) (*cheshireConn, error) {
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		return nil, err
	}

	//this doesnt work
	// if c, ok := conn.(net.TCPConn); ok { 
	//     err = c.SetKeepAlive(true)
	//     if err != nil {
	//         return nil, err
	//     }
	// }

	nc := &cheshireConn{
		Conn:           conn,
		connected:      true,
		addr:           addr,
		writeTimeout:   writeTimeout,
		exitChan:       make(chan int),
		incomingChan:   make(chan *Response, 25),
		outgoingChan:   make(chan *cheshireRequest, 25),
		disconnectChan: disconnect,
		requests:       make(map[string]*cheshireRequest),
		connectedAt:    time.Now(),
	}
	return nc, nil
}

func (this *cheshireConn) Close() {
	if !this.connected {
		return //do nothing.
	}
	this.exitChan <- 1
}

func (this *cheshireConn) String() string {
	return this.addr
}

// loop that listens for incoming messages.
func (this *cheshireConn) listener() {
	decoder := json.NewDecoder(bufio.NewReader(this.Conn))
	log.Printf("Starting Cheshire Connection %s", this.addr)
	defer func() { this.exitChan <- 1 }()
	for {
		res := &Response{*dynmap.NewDynMap()}
		err := decoder.Decode(res)
		if err == io.EOF {
			log.Print(err)
			break
		} else if err != nil {
			log.Print(err)
			break
		}
		this.incomingChan <- res
	}
}

func (this *cheshireConn) cleanup() {
	this.Conn.Close()
	log.Printf("Closing Cheshire Connection: %s", this.addr)

	err := fmt.Errorf("Connection is closed %s", this.addr)
	//now error out all waiting
	for len(this.outgoingChan) > 0 {
		req := <-this.outgoingChan
		//send an error to the error chan
		req.errorChan <- err
	}
	log.Println("ended outchan")
	for k, v := range this.requests {
		v.errorChan <- err
		delete(this.requests, k)
	}
	log.Println("Ended request clear")
	this.disconnectChan <- this
}

func (this *cheshireConn) eventLoop() {
	go this.listener()

	writer := bufio.NewWriter(this.Conn)

	defer this.cleanup()
	for this.connected {
		select {
		case request := <-this.outgoingChan:
			//add to the request map

			this.requests[request.req.TxnId()] = request

			//send the request
			this.SetWriteDeadline(time.Now().Add(this.writeTimeout))

			// log.Printf("Sending: %s", request.req.TxnId())
			json, err := json.Marshal(request.req)
			if err != nil {
				//TODO: uhh, do something..
				log.Print(err)
				continue
			}
			_, err = writer.Write(json)
			writer.Flush()
			if err != nil {
				//TODO: uhh, do something..
				log.Print(err)
				continue
			}
		case response := <-this.incomingChan:
			req, ok := this.requests[response.TxnId()]
			if !ok {
				log.Printf("Uhh, received response, but had no request %s", response)
				// for k,_ := range(this.requests) {
				//     log.Println(k)
				// }
				continue //break?
			}
			req.resultChan <- response
			//remove if txn is finished..
			if response.TxnStatus() == "completed" {
				delete(this.requests, response.TxnId())
			}
		case <-this.exitChan:
			this.connected = false
		}
	}
}
