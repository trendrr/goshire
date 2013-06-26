package client

import(
    "time"
    "sync/atomic"
    "fmt"
    "log"
)

type ClientCreator interface {
    
    //Should create and connect to a new client
    Create() (*cheshireConn, error)
    //Should clean up the connection resources
    //implementation should deal with Cleanup possibly being called multiple times
    Cleanup(*cheshireConn)
}

// A simple channel based pool.
// originally from http://www.ryanday.net/2012/09/12/golang-using-channels-for-a-connection-pool/
type Pool struct {
    size int
    clients chan *cheshireConn
    creator ClientCreator
    shutdown int32
}

// Creates a new pool, and initializes the clients.
// will fail with an error if any of the clients fail
func NewPool(size int, creator ClientCreator) (*Pool,error) {
    log.Println("CREATIGN POOL")
    p := &Pool{
        size : size,
        creator : creator,
        clients : make(chan *cheshireConn, size),
        shutdown : 0,
    }
    for i := 0; i < size; i++ {
        conn, err := creator.Create()
        if err != nil {
            p.Close()
            return nil, err
        }

        p.clients <- conn
    }
    return p, nil
}

//Close this pool and clean up all connections
func (this *Pool) Close() {
    atomic.StoreInt32(&this.shutdown, 1)

    for {
        select {
        case c := <-this.clients:
            this.creator.Cleanup(c)
        default:
            //done
            return
        }
    }
}

//checkout a connection
//returns an error if this pool is 
//closed, or timeout
func (this *Pool) Borrow(timeout time.Duration) (*cheshireConn, error) {
    if this.shutdown != 0 {
        return nil, fmt.Errorf("Pool is closed. Try the beach")
    }

    select {
    case c := <- this.clients:
        return c, nil
    case <- time.After(timeout):
        return nil, fmt.Errorf("Timeout trying to checkout from pool")
    }
}

// return to the pool
func (this *Pool) Return(conn *cheshireConn) {
    if this.shutdown != 0 {
        this.creator.Cleanup(conn)
        return
    }
    this.clients <- conn
} 

// Returns a broken item.
// this item will be closed and a fresh one will be added to the pool
// If a new item is not able to be added to the pool (creation fails), then
// the broken one will be added back to the pool.  This is to ensure that
// we always have the correct # of items in the pool
func (this *Pool) ReturnBroken(conn *cheshireConn) {
    if this.shutdown != 0 {
        this.creator.Cleanup(conn)
        return
    }

    c, err := this.creator.Create()
    if err != nil {
        log.Printf("ERROR in Pool %s", err)
        this.Return(conn)
        return
    }
    //add new item to the pool
    this.clients <- c

    //destroy the old one
    this.creator.Cleanup(conn)
}