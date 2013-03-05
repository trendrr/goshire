package cheshire

import (
    "bufio"
    "net"
    "time"
    "encoding/json"
    "log"
    "io"
        "sync/atomic"
    "fmt"
        "github.com/trendrr/cheshire-golang/dynmap"
)

var strestId int64 = int64(0)
//create a new unique strest txn id
func NewTxnId() string {
    id := atomic.AddInt64(&strestId, int64(1))
    return fmt.Sprintf("go%d", id)
}


type Client struct {
    Host string
    Port int
    PingUri string
    conn *cheshireConn
    //channel is alerted when the connection is disconnected.
    disconnectChan chan *cheshireConn 
    exitChan chan int
}

//Creates a connects 
func NewClient(host string, port int) (*Client, error) {
    client := &Client{
        Host: host,
        Port: port,
        disconnectChan: make(chan *cheshireConn),
        PingUri: "/ping",
    }
    conn, err := client.createConn()
    if err != nil {
        return nil, err
    }
    client.conn = conn
    go client.eventLoop()

    return client, nil
}

func (this *Client) createConn() (*cheshireConn, error) {
    c, err := newCheshireConn(fmt.Sprintf("%s:%d",this.Host, this.Port), this.disconnectChan, 20 * time.Second)
    if err != nil {
        return nil, err
    }

    go c.eventLoop()
    return c, nil
}

//returns the connection.  
// use this rather then access directly from the struct, will
// make it easier to pool connections if we need.
func (this *Client) connection() (*cheshireConn, error) {
    return this.conn, nil
}

//Attempt to close this connection and make a new connection.
func (this *Client) reconnect(oldconn *cheshireConn) (*cheshireConn, error) {
    if oldconn.connectedAt.After(time.Now().Add(5*time.Second)) {
       //only allow one reconnect attempt per 5 second interval
       //returning the old connection, because this was likely a concurrent reconnect 
       // attempt, and perhaps the previous one was successfull
       return oldconn, nil 
    }

    oldconn.Close()
    con, err := this.createConn()
    return con,err
}

func (this *Client) eventLoop() {
    //client event loop pings, and listens for client disconnects.
    c := time.Tick(30 * time.Second)
    select {
    case <- this.exitChan :
        //close all connections
        this.conn.Close()
        break
    case <- c :
        //ping all the connections.
        _, err := this.ApiCallSync(NewRequest(this.PingUri, "GET"), 10*time.Second)
        if err != nil {
            // uhh should we reconnect?
        }
    case conn := <- this.disconnectChan:
        //reconnect
        this.reconnect(conn)
    }
}



// Does a synchronous api call.  times out after the requested timeout.
func (this *Client) ApiCallSync(req *Request, timeout time.Duration) (*Response, error) {
    responseChan := make(chan *Response)
    errorChan := make(chan error)
    this.doApiCall(req, responseChan, errorChan)
    select {
    case response := <- responseChan :
        return response, nil
    case err :=<-errorChan :
        return nil, err
    case <- time.After(timeout) :
        return nil, fmt.Errorf("Request timeout")
    }
    return nil, fmt.Errorf("Impossible error happened, alert NASA")
}

// Does an api call.
func (this *Client) ApiCall(req *Request, responseChan chan *Response, errorChan chan error) {
    this.doApiCall(req, responseChan, errorChan)
}

//does the actual call, returning the connection and the internal request
func (this *Client) doApiCall(req *Request, responseChan chan *Response, errorChan chan error)(*cheshireConn, *cheshireRequest) {
    if req.TxnId() == "" {
        req.SetTxnId(NewTxnId())
    }
    r := newRequest(req, responseChan, errorChan)

    conn, err := this.connection()
    if err != nil {
        errorChan <- err
    } else {
        conn.outgoingChan <- r    
    }
    return conn, r
}

// Connection to a cheshire server. 
type cheshireConn struct {
    net.Conn
    addr    string
    readTimeout      time.Duration
    writeTimeout     time.Duration
    incomingChan chan *Response
    outgoingChan chan *cheshireRequest
    exitChan chan int
    disconnectChan chan *cheshireConn
    //map of txnId to request
    requests map[string] *cheshireRequest
    connectedAt time.Time
}

//wrap a request so we dont lose track of the result channels
type cheshireRequest struct {
    req *Request
    resultChan chan *Response
    errorChan chan error
}

func newRequest(req *Request, resultChan chan *Response, errorChan chan error) *cheshireRequest {
    return &cheshireRequest{
        req: req,
        resultChan: resultChan,
        errorChan: errorChan,
    }
}

func newCheshireConn(addr string, disconnect chan *cheshireConn, writeTimeout time.Duration) (*cheshireConn, error) {
    conn, err := net.DialTimeout("tcp", addr, time.Second)
    if err != nil {
        return nil, err
    }

    nc := &cheshireConn{
        Conn:             conn,
        addr:             addr,
        writeTimeout:     writeTimeout,
        exitChan:         make(chan int),
        incomingChan:   make(chan *Response, 25),
        outgoingChan:   make(chan *cheshireRequest, 25),
        disconnectChan: disconnect,
        requests: make(map[string] *cheshireRequest),
        connectedAt: time.Now(),
    }
    return nc, nil
}

func (this *cheshireConn) Close() {
    this.exitChan <- 1
}

func (this *cheshireConn) String() string {
    return this.addr
}



// loop that listens for incoming messages.
func (this *cheshireConn) listener() {
    decoder := json.NewDecoder(bufio.NewReader(this.Conn))
    log.Printf("Starting Cheshire Client %s", this.addr)
    defer func() {this.exitChan <- 1}()
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
        log.Printf("DECODED: %s", res.TxnId())
        this.incomingChan <- res
    }
}

func (this *cheshireConn) cleanup() {
    this.Conn.Close()
    log.Println("Closing Cheshire Client: %s", this.addr)

    err := fmt.Errorf("Connection is closed %s", this.addr)
    //now error out all waiting
    for len(this.outgoingChan) > 0 {
        req := <- this.outgoingChan
        //send an error to the error chan
        req.errorChan <- err
    }

    for k,v := range(this.requests) {
        v.errorChan <- err
        delete(this.requests, k)
    }

}

func (this *cheshireConn) eventLoop() {
    go this.listener()

    writer := bufio.NewWriter(this.Conn)
    
    defer this.cleanup()
    for {
        select {
            case request := <- this.outgoingChan:
                //add to the request map

                this.requests[request.req.TxnId()] = request

                //send the request
                this.SetWriteDeadline(time.Now().Add(this.writeTimeout))
                
                // log.Printf("Sending: %s", request.req.TxnId())
                json, err := json.Marshal(request.req)
                if err != nil {
                    //TODO: uhh, do something..
                    log.Print(err)
                    continue;
                } 
                _, err = writer.Write(json)
                writer.Flush()
                if err != nil {
                    //TODO: uhh, do something..
                    log.Print(err)
                    continue;
                }
            case response := <- this.incomingChan:
                log.Printf("RECEIVED: %s", response.TxnId())
                req, ok := this.requests[response.TxnId()]
                if !ok {
                    log.Printf("Uhh, received response, but had no request %s", response)
                    // for k,_ := range(this.requests) {
                    //     log.Println(k)
                    // }
                    continue; //break?
                }
                req.resultChan <- response
                //remove if txn is finished..
                if response.TxnStatus() == "completed" {
                    delete(this.requests, response.TxnId())
                }
            case <- this.exitChan:
                break
        }
    }
}
