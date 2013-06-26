package client

import (
    "net"
    "log"
    "fmt"
    "time"
    "github.com/trendrr/goshire/cheshire"
    "github.com/trendrr/goshire/dynmap"
    "io"
    "bufio"
    "encoding/json"
    "sync"

)

// A single connection to a cheshire server. 
// The connection is considered fail fast
// should be disconnected and reaped
type cheshireConn struct {
    net.Conn
    addr           string
    connected      bool  //use accessors only for threadsafety
    readTimeout    time.Duration
    writeTimeout   time.Duration
    incomingChan   chan *cheshire.Response
    outgoingChan   chan *cheshireRequest
    exitChan       chan int
    disconnectChan chan *cheshireConn
    //if max inflight is reached, we wait on this chan
    inwaitChan chan bool
    
    //map of txnId to request
    requests    map[string]*cheshireRequest
    connectedAt time.Time
    maxInFlight int
    //the index in the owning pool
    poolIndex int
    lock sync.RWMutex
}

//wrap a request so we dont lose track of the result channels
type cheshireRequest struct {
    bytes       []byte
    req        *cheshire.Request
    resultChan chan *cheshire.Response
    errorChan  chan error
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
        incomingChan:   make(chan *cheshire.Response, 25),
        outgoingChan:   make(chan *cheshireRequest, 25),
        //if max inflight is reached, we wait on this chan
        inwaitChan: make(chan bool),
        disconnectChan: disconnect,
        requests:       make(map[string]*cheshireRequest),
        connectedAt:    time.Now(),
    }
    return nc, nil
}

func (this *cheshireConn) setConnected(v bool) {
    this.lock.Lock()
    defer this.lock.Unlock()
    this.connected = v
}

func (this *cheshireConn) Connected() bool {
    this.lock.RLock()
    defer this.lock.RUnlock()
    return this.connected
}

//returns the current # of requests in flight
//unsafe
func (this *cheshireConn) inflight() int {
    this.lock.RLock()
    defer this.lock.RUnlock()
    return len(this.requests)
}

// Sends a new request.
// this will check the max inflight, and will block for max 20 seconds waiting for the # inflilght to go down.
// if inflight does not go down it will close the connection and return error.
// errors are returned, not sent to the errorchan
func (this *cheshireConn) sendRequest(request *cheshire.Request, resultChan chan *cheshire.Response, errorChan chan error) (*cheshireRequest, error) {

    if this.inflight() > this.maxInFlight {
        
        log.Printf("Max inflight reached (%d) of (%d), waiting pool index: %d", this.inflight(), this.maxInFlight, this.poolIndex)
        //TODO: timeout channel
        select {
        case <- this.inwaitChan :
            //yay
        case <-time.After(20 * time.Second): 
            //should close this connection..
            this.Close()
            return nil, fmt.Errorf("Max inflight sustained for more then 20 seconds, fail")
        }
    }

    if !this.Connected() {
        return nil, fmt.Errorf("Not connected")
    }

    json, err := json.Marshal(request)
    if err != nil {
        return nil, err
    }


    req := &cheshireRequest{
        req:        request,
        bytes:      json,
        resultChan: resultChan,
        errorChan:  errorChan,
    }
    this.outgoingChan <- req
    return req, nil
}

func (this *cheshireConn) Close() {
    if !this.Connected() {
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
        res := &cheshire.Response{*dynmap.New()}
        err := decoder.Decode(res)
        if err == io.EOF {
            log.Print(err)
            break
        } else if err != nil {
            log.Print(err)
            break
        }
        this.incomingChan <- res

        //alert the inwaitchan, non-blocking
        select {
        case this.inwaitChan <- true :
        default:
        }
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
    this.lock.Lock()
    for k, v := range this.requests {
        v.errorChan <- err
        delete(this.requests, k)
    }
    this.lock.Unlock()

    log.Println("Ended request clear")
    this.disconnectChan <- this
}

func (this *cheshireConn) eventLoop() {
    go this.listener()

    writer := bufio.NewWriter(this.Conn)

    defer this.cleanup()
    for this.Connected() {
        select {
        case response := <-this.incomingChan:
            this.lock.RLock()
            req, ok := this.requests[response.TxnId()]
            this.lock.RUnlock()
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
                this.lock.Lock()
                delete(this.requests, response.TxnId())
                this.lock.Unlock()
            }
        case request := <-this.outgoingChan:
            //add to the request map
            this.lock.Lock()
            this.requests[request.req.TxnId()] = request
            this.lock.Unlock()

            //send the request
            this.SetWriteDeadline(time.Now().Add(this.writeTimeout))
            _, err := writer.Write(request.bytes)
            if err != nil {
                //TODO: uhh, do something..
                log.Print(err)
                continue
            }
            writer.Flush()


        case <-this.exitChan:
            this.setConnected(false)
        }
    }
}
