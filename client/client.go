package client

import (
	"bytes"
	"fmt"
	"github.com/trendrr/goshire/cheshire"
	"github.com/trendrr/goshire/dynmap"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)



type Client interface {
	// Does a synchronous api call.  times out after the requested timeout.
	// This will automatically set the txn accept to single
	ApiCallSync(req *cheshire.Request, timeout time.Duration) (*cheshire.Response, error)
	// Does an api call.
	ApiCall(req *cheshire.Request, responseChan chan *cheshire.Response, errorChan chan error) error

	//Closes this client
	Close()
}

type HttpClient struct {
	Address string
}

// does an asynchrounous api call to the requested address.
func HttpApiCall(address string, req *cheshire.Request, responseChan chan *cheshire.Response, errorChan chan error) {
	cl := NewHttp(address)
	cl.ApiCall(req, responseChan, errorChan)
}

// does a synchronous api call to the requested address.
func HttpApiCallSync(address string, req *cheshire.Request, timeout time.Duration) (*cheshire.Response, error) {
	cl := NewHttp(address)
	res, err := cl.ApiCallSync(req, timeout)
	return res, err
}

func NewHttp(address string) *HttpClient {
	addr := strings.TrimLeft(address, "http://")

	return &HttpClient{
		Address: addr,
	}
}

func (this *HttpClient) Close() {
	//do nothing..
}

// Make an async api call
func (this *HttpClient) ApiCall(req *cheshire.Request, responseChan chan *cheshire.Response, errorChan chan error) error {
	go func() {
		//TODO we could do something that allows streaming http
		res, err := this.ApiCallSync(req, 4*60*time.Second)
		if err != nil {
			errorChan <- err
		} else {
			responseChan <- res
		}
	}()
	return nil
}

func (this *HttpClient) ApiCallSync(req *cheshire.Request, timeout time.Duration) (*cheshire.Response, error) {
	uri := req.Uri()

	var reqBody io.Reader

	if req.Method() == "GET" {
		joiner := "&"
		//add params to the uri
		if !strings.Contains(uri, "?") {
			joiner = "?"
		}

		pms, err := req.Params().MarshalURL()
		if err != nil {
			return nil, err
		}
		uri = fmt.Sprintf("%s%s%s", uri, joiner, pms)
	} else {
		// set the request body as json
		json, err := req.Params().MarshalJSON()
		if err != nil {
			return nil, err
		}

		// log.Printf("JSON %s", string(json))
		reqBody = bytes.NewReader(json)
	}

	url := fmt.Sprintf("http://%s%s", this.Address, uri)
	//convert to an http.Request
	request, err := http.NewRequest(req.Method(), url, reqBody)

	if req.Method() != "GET" {
		//set the content type
		request.Header.Set("Content-Type", "application/json")
	}

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
	mp := dynmap.New()
	// var response = &cheshire.Response{*dynmap.NewDynMap()}
	err = mp.UnmarshalJSON(body)
	response := cheshire.NewResponseDynMap(mp)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// Client that utilizes the json protocol and
// connections are static.
// maintains an internal connection pool.
type JsonClient struct {
	Host     string
	Port     int
	PingUri  string
	shutdown int32
	pool     *Pool
	PoolSize int
	exitChan chan int

	//The max number of requests that can be waiting for a response.
	//When max inflight is reached, the client will start
	//blocking and waiting for room.
	//connection will eventually close if it waits too long.
	MaxInFlight int

	// Number of times we should retry to send a request if there is a connection problem
	// default is 1
	Retries int

	//The amount of time to pause between retries
	// default is 500 millis
	RetryPause time.Duration

	count          uint64
	maxInFlightPer int
	protocol cheshire.Protocol
}

//Creates a new Json client
// Remember to call client.Connect
func NewJson(host string, port int) *JsonClient {
	client := &JsonClient{
		Host:        host,
		Port:        port,
		shutdown:    1,
		exitChan:    make(chan int),
		PingUri:     "/ping",
		PoolSize:    5,
		MaxInFlight: 200,
		Retries:     1,
		RetryPause:  time.Duration(500) * time.Millisecond,
		protocol: cheshire.JSON,
	}
	return client
}

func (this *JsonClient) setClosed(v bool) {
	if v {
		atomic.StoreInt32(&this.shutdown, 1)
	} else {
		atomic.StoreInt32(&this.shutdown, 0)
	}
}

func (this *JsonClient) Closed() bool {
	return this.shutdown != 0
}

// Starts the json event loop and initializes one or
// more connections
// if a connection already exists it will be closed
func (this *JsonClient) Connect() error {
	if !this.Closed() {
		return fmt.Errorf("Connect called on connected client")
	}
	this.setClosed(false)

	this.maxInFlightPer = int(this.MaxInFlight / this.PoolSize)
	if this.maxInFlightPer < 10 {
		log.Printf("Max Inflight is less then 10 per connection, suggest raising it (%d)", this.maxInFlightPer)
	}
	if this.maxInFlightPer < 2 {
		log.Printf("Setting max inflight to 2")
		this.maxInFlightPer = 2
	}

	clientPoolCreator := &clientPoolCreator{
		client: this,
	}

	pool, err := NewPool(this.PoolSize, clientPoolCreator)
	if err != nil {
		return err
	}
	this.pool = pool

	go this.pingLoop()
	return nil
}

//Close this client.
func (this *JsonClient) Close() {
	this.exitChan <- 1
	log.Println("Send exit message")
}

//returns the connection.
// use this rather then access directly from the struct, will
// make it easier to pool connections if we need.
// This will attempt to return the next operating connection
func (this *JsonClient) connection() (*cheshireConn, error) {
	var err error
	for x := 0; x < this.Retries; x++ {
		for i := 0; i < this.PoolSize; i++ {
			c, err := this.pool.Borrow(1 * time.Second)
			if err != nil {
				log.Printf("Error getting connection from pool : %s", err)
				continue
			}
			return c, nil
		}
		if x < this.Retries {
			time.Sleep(this.RetryPause)	
		}
	}
	if err == nil {
		err = fmt.Errorf("Unable to get connection, likely all are busy")
	}
	return nil, err

}

func (this *JsonClient) pingLoop() {
	pingTimer := time.Tick(25 * time.Second)
	for !this.Closed() {
		<-pingTimer
		for i := 0; i < this.PoolSize; i++ {
			//Do the ping
			_, err := this.doApiCallSync(cheshire.NewRequest(this.PingUri, "GET"), 10*time.Second)
			if err != nil {
				log.Printf("Error in ping %s", err)
			}
		}
	}
}

// Does a synchronous api call.  times out after the requested timeout.
// This will automatically set the txn accept to single
func (this *JsonClient) ApiCallSync(req *cheshire.Request, timeout time.Duration) (*cheshire.Response, error) {
	req.SetTxnAccept("single")
	response, err := this.doApiCallSync(req, timeout)
	return response, err
}

// Does an api call.
// Transport errors and others will arrive on the channel.
// will return an error immediately if no connection is available.

func (this *JsonClient) ApiCall(req *cheshire.Request, responseChan chan *cheshire.Response, errorChan chan error) error {
	_, err := this.doApiCall(req, responseChan, errorChan)
	return err
}

func (this *JsonClient) CurrentInFlight() int {
	log.Println("CurrentInflight is not implemented!")
	return 0
}

func (this *JsonClient) doApiCallSync(req *cheshire.Request, timeout time.Duration) (*cheshire.Response, error) {
	responseChan := make(chan *cheshire.Response)
	errorChan := make(chan error, 5)
	_, err := this.doApiCall(req, responseChan, errorChan)
	if err != nil {
		return nil, err
	}
	select {
	case response := <-responseChan:
		return response, nil
	case err := <-errorChan:
		log.Printf("GOT ERROR! %s", err)
		return nil, err
	case <-time.After(timeout):
		return nil, fmt.Errorf("Request timeout")
	}
	return nil, fmt.Errorf("Impossible error happened, alert NASA")
}

//does the actual call, returning the connection and the internal request
func (this *JsonClient) doApiCall(req *cheshire.Request, responseChan chan *cheshire.Response, errorChan chan error) (*cheshireRequest, error) {
	conn, err := this.connection()
	if err != nil {
		return nil, err
	}
	if req.TxnId() == "" {
		req.SetTxnId(cheshire.NewTxnId())
	}
	r, err := conn.sendRequest(req, responseChan, errorChan)
	if err == nil {
		this.pool.Return(conn)
	} else {
		this.pool.ReturnBroken(conn)
	}
	return r, err
}

//handles the creation for the pool
type clientPoolCreator struct {
	client *JsonClient
}

//Should create and connect to a new client
func (this *clientPoolCreator) Create() (*cheshireConn, error) {
	c, err := newCheshireConn(this.client.protocol, fmt.Sprintf("%s:%d", this.client.Host, this.client.Port), 20*time.Second)
	if err != nil {
		return nil, err
	}
	log.Printf("Max Inflight %d, Per %d, Poolsize %d", this.client.MaxInFlight, this.client.maxInFlightPer, this.client.PoolSize)
	c.maxInFlight = this.client.maxInFlightPer
	go c.eventLoop()
	return c, nil
}

//Should clean up the connection resources
//implementation should deal with Cleanup possibly being called multiple times
func (this *clientPoolCreator) Cleanup(conn *cheshireConn) {

}
