package cheshire

import (
    "bufio"
    "net"
    "time"
    "encoding/json"
    "log"
    "io"
)

type Client struct {
    host string
    port int


}

// Connection to a cheshire server. 
type cheshireConn struct {
    net.Conn
    addr    string
    readTimeout      time.Duration
    writeTimeout     time.Duration
    incomingChan chan *Response
    outgoingChan chan *Request
    exitChan chan int
    disconnectChan chan *cheshireConn
    pingUri string
    requests map[string] *cheshireRequest
}

type cheshireRequest struct {
    req *Request
    resultChan chan *Result
    errorChan chan error
}

func newCheshireConn(addr string, disconnect chan *cheshireConn, writeTimeout time.Duration, pingUri string) (*cheshireConn, error) {
    conn, err := net.DialTimeout("tcp", addr, time.Second)
    if err != nil {
        return nil, err
    }

    nc := &cheshireConn{
        Conn:             conn,
        addr:             addr,
        writeTimeout:     writeTimeout,
        exitChan:         make(chan int),
        incomingChan:   make(chan *Response),
        outgoingChan:   make(chan *Request, 25),
        disconnectChan: disconnect,
        pingUri: pingUri,
        request
    }
    return nc, nil
}

func (this *cheshireConn) close() {
    this.exitChan <- 1
}

func (this *cheshireConn) String() string {
    return this.addr
}

// loop that listens for incoming messages.
func (this *cheshireConn) listener() {

    decoder := json.NewDecoder(bufio.NewReader(this.Conn))

    for {
        var res Response
        err := decoder.Decode(&res)

        if err == io.EOF {
            log.Print(err)
            break
        } else if err != nil {
            log.Print(err)
            break
        }
        
    }

    //TODO alert a disconnect channel?
    this.exitChan <- 1
}

func (this *cheshireConn) eventLoop() {
    go this.listener()

    writer := bufio.NewWriter(this.Conn)
    ping := time.Tick(30 * time.Second)

    defer this.Conn.Close()
    for {
        select {
            case request := <- this.outgoingChan:
                //send the request
                this.SetWriteDeadline(time.Now().Add(this.writeTimeout))
                json, err := json.Marshal(request)
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
            case <- this.exitChan:
                break
            case <- ping:
                log.Println("PING!")
                //TODO: implement ping.
        }
    }
}