package client

import (
	"fmt"
	"github.com/trendrr/goshire/dynmap"
	"github.com/trendrr/goshire/cheshire"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"bytes"
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
	var response = &cheshire.Response{*dynmap.NewDynMap()}
	err = response.UnmarshalJSON(body)
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
	isClosed bool
	//The connection
	//TODO: this should be a pool
	conn     []*cheshireConn
	
	//this channel is alerted when the connection is disconnected.
	disconnectChan chan *cheshireConn
	exitChan       chan int
	connectLock    sync.RWMutex

	//All reconnections happen in this channel
	reconnectChan chan *cheshireConn

	//The max number of requests that can be waiting for a response.
	//When max inflight is reached, the client will start
	//blocking and waiting for room.
	//connection will eventually close if it waits too long.
	MaxInFlight int

	// The connection pool size
	PoolSize int

	// Number of times we should retry to send a request if there is a connection problem
	// default is 1
	Retries int 

	//The amount of time to pause between retries
	// default is 500 millis
	RetryPause time.Duration

	count uint64
	maxInFlightPer int
}

//Creates a new Json client 
// Remember to call client.Connect
func NewJson(host string, port int) (*JsonClient) {
	client := &JsonClient{
		Host:           host,
		Port:           port,
		isClosed:       false,
		disconnectChan: make(chan *cheshireConn),
		exitChan:       make(chan int),
		reconnectChan:  make(chan *cheshireConn, 50),
		PingUri:        "/ping",
		PoolSize:		5,
		MaxInFlight: 200,
		conn:		make([]*cheshireConn, 0),
		Retries: 1,
		RetryPause: time.Duration(500)*time.Millisecond,
	}
	return client
}

// returns the total # of requests that are currently inflight (i.e. txn in progress)
// func (this *JsonClient) CurrentInFlight() int {


// 	return len(this.conn.requests)
// }

// Starts the json event loop and initializes one or
// more connections
// if a connection already exists it will be closed
func (this *JsonClient) Connect() error {
	this.connectLock.Lock()
	defer this.connectLock.Unlock()

	if len(this.conn) > 0 {
		//close all the connections.
		this.Close()
	}

	this.isClosed = false

	this.maxInFlightPer = int(this.MaxInFlight / this.PoolSize)
	if this.maxInFlightPer < 10 {
		log.Printf("Max Inflight is less then 10 per connection, suggest raising it (%d)", this.maxInFlightPer)
	}
	if this.maxInFlightPer < 2 {
		log.Printf("Setting max inflight to 2")
		this.maxInFlightPer = 2
	}

	this.conn = make([]*cheshireConn, this.PoolSize)
	for i:=0; i < this.PoolSize;i++ {
		conn, err := this.createConn()
		if err != nil {
			return err
		}
		conn.poolIndex = i
		this.conn[i] = conn
	}

	go this.eventLoop()
	return nil
}

//Close this client.
func (this *JsonClient) Close() {
	this.exitChan <- 1
	log.Println("Send exit message")
}

//Creates a new cheshire connection and returns it
func (this *JsonClient) createConn() (*cheshireConn, error) {
	c, err := newCheshireConn(fmt.Sprintf("%s:%d", this.Host, this.Port), this.disconnectChan, 20*time.Second)
	if err != nil {
		return nil, err
	}
	log.Printf("Max Inflight %d, Per %d, Poolsize %d", this.MaxInFlight, this.maxInFlightPer, this.PoolSize)
	c.maxInFlight = this.maxInFlightPer
	go c.eventLoop()
	return c, nil
}

//returns the connection.  
// use this rather then access directly from the struct, will
// make it easier to pool connections if we need.
// This will attempt to return the next operating connection
func (this *JsonClient) connection() (*cheshireConn, error) {
	var err error

	for i:=0; i < this.PoolSize; i++ {
		conn, err := this.nextConnection()
		if err != nil {
			continue
		}
		if conn.inflight() >= this.maxInFlightPer {
			continue
		}
		return conn, nil
	}
	if err == nil {
		err = fmt.Errorf("Unable to get connection, likely all are busy")
	}
	return nil, err
	
}

//returns the next connection
func (this *JsonClient) nextConnection() (*cheshireConn, error) {
	index := int(atomic.AddUint64(&this.count, uint64(1)) % uint64(this.PoolSize))
	this.connectLock.RLock()
	conn := this.conn[index]
	this.connectLock.RUnlock()
	if !conn.Connected() {
		return conn, fmt.Errorf("Not connected")
	}
	return conn, nil
}

//Attempt to close this connection and make a new connection.
func (this *JsonClient) reconnect(oldconn *cheshireConn) {
	this.reconnectChan <- oldconn
}

func (this *JsonClient) reconnectLoop(closeChan chan bool) {
	//TODO is the closechan necessary?  will it get gc'ed if
	// the go routine is abandoned?

	for {
		select {
		case <- closeChan :
			log.Println("Closing reconnect loop")
			return
		case conn := <-this.reconnectChan :
			err := this.reconn(conn)
			if err != nil {
				log.Printf("Error during reconnect -- %s", err)
			}
		}
	}
}

//The actual reconnect logic.  do not call this directly, it is handled in a 
// special go routine
func (this *JsonClient) reconn(oldconn *cheshireConn) (error) {
	if oldconn.connectedAt.After(time.Now().Add(-5 * time.Second)) {

		log.Printf("Last connection attempted %s NOW is %s", oldconn.connectedAt.Local(), time.Now().Local())
		//only allow one reconnect attempt per 5 second interval
		//returning the old connection, because this was likely a concurrent reconnect 
		// attempt, and perhaps the previous one was successfull
		log.Println("Skipping reconnect too early")
		return nil
	}
	//update the connect attempt time.
	oldconn.connectedAt = time.Now()

	log.Println("Closing old")
	oldconn.Close()

	log.Println("Creating new!")
	con, err := this.createConn()
	if err != nil {
		log.Printf("COUldn't create new %s", err)

		return err
	}
	this.connectLock.Lock()
	defer this.connectLock.Unlock()

	//double check the connection pool.  make certain no leaks
	if this.conn[oldconn.poolIndex] != oldconn {
		log.Println("Ack! old connection is not in the pool anymore!")
		this.conn[oldconn.poolIndex].Close()
	}
	this.conn[oldconn.poolIndex] = con
	

	log.Println("DONE RECONNECT %s", con)
	return err
}

func (this *JsonClient) eventLoop() {
	//client event loop pings, and listens for client disconnects.
	pingTimer := time.Tick(5 * time.Second)

	reconnectExit := make(chan bool)
	go this.reconnectLoop(reconnectExit)

	defer log.Println("CLOSED!!!!!!!!")
	defer func (){reconnectExit <- true}()

	for !this.isClosed {
		select {
		case <-this.exitChan:
			log.Println("Exiting Client")
			//close all connections
			for _,conn := range(this.conn) {
				conn.Close()
			}
			this.isClosed = true
			break
		case <-pingTimer:
			//ping all the connections.
			
			for _, conn := range(this.conn) {
				
				_, err := this.doApiCallSync(conn, cheshire.NewRequest(this.PingUri, "GET"), 10*time.Second)
				if err != nil {
					log.Printf("COULDNT PING (%s) Attempting to reconnect", err)
					this.reconnect(conn)
				}
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
func (this *JsonClient) ApiCallSync(req *cheshire.Request, timeout time.Duration) (*cheshire.Response, error) {
	conn, err := this.connection()
	//retry if necessary
	for i := 0; i < this.Retries && err != nil; i++ {
		time.Sleep(this.RetryPause)
		conn, err = this.connection()
	}
	if err != nil {
		return nil, err
	}

	req.SetTxnAccept("single")
	response, err := this.doApiCallSync(conn, req, timeout)
	return response, err
}

// Does an api call.
// Transport errors and others will arrive on the channel.  
// will return an error immediately if no connection is available.

func (this *JsonClient) ApiCall(req *cheshire.Request, responseChan chan *cheshire.Response, errorChan chan error) error {
	conn, err := this.connection()
	//retry if necessary
	for i := 0; i < this.Retries && err != nil; i++ {
		time.Sleep(this.RetryPause)
		conn, err = this.connection()
	}

	if err != nil {
		return err
	}
	_, err = this.doApiCall(conn, req, responseChan, errorChan)
	return err
}

func (this *JsonClient) CurrentInFlight() int {
	log.Println("CurrentInflight is not implemented!")
	return 0
}

func (this *JsonClient) doApiCallSync(conn *cheshireConn, req *cheshire.Request, timeout time.Duration) (*cheshire.Response, error) {
	responseChan := make(chan *cheshire.Response)
	errorChan := make(chan error, 5)
	_, err := this.doApiCall(conn, req, responseChan, errorChan)
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
func (this *JsonClient) doApiCall(conn *cheshireConn, req *cheshire.Request, responseChan chan *cheshire.Response, errorChan chan error) (*cheshireRequest, error) {
	if conn == nil {
		return nil, fmt.Errorf("Cannot do api call, conn is nil")
	}
	if req.TxnId() == "" {
		req.SetTxnId(NewTxnId())
	}
	r, err := conn.sendRequest(req, responseChan, errorChan)
	return r, err
}
