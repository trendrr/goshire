package partition

import (
    "github.com/trendrr/cheshire-golang/dynmap"
)


type Partitioner interface {
    //Return the current routertable.
    RouterTable() (*RouterTable, error)

    //Set the router table.  through an error if
    //attempting to set an older partition table
    SetRouterTable(routerTable *RouterTable) error
    
    //Lock the data.  this is for rebalance operations
    Lock() error
    Unlock() error

    //Gets all the data for a specific partition
    //should send total # of items on the finished chanel when complete
    Data(partition int, deleteData bool, chan *dynmap.DynMap, finished chan int, errorChan chan error)
}

var partitioner Partitioner 

// Sets the partitioner and registers the necessary 
// controllers
func SetPartitioner(par Partitioner) {
    partitioner = par

    //register the controllers.
    cheshire.RegisterApi("/chs/rt/get", "GET", GetRouterTable)
    cheshire.RegisterApi("/chs/rt/set", "POST", SetRouterTable)
    cheshire.RegisterApi("/chs/lock", "POST", Lock)
    cheshire.RegisterApi("/chs/unlock", "POST", Unlock)

}


func GetRouterTable(request *cheshire.Request, conn cheshire.Connection) {
    tble, err := partitioner.RouterTable()
    if err != nil {
        conn.Write(request.NewError(506, string(err)))
        return
    }
    response := request.NewResponse()

    if request.Params().MustBool("rv_only", false) {
        response.PutWithDot("router_table.revision" tble.Revision)
    } else {
        response.Put("router_table", tble.ToDynMap())
    }
    conn.Write(response)
}

func SetRouterTable(request *cheshire.Request, conn cheshire.Connection) {
    rtmap, ok := request.Params().DynMap("router_table")
    if !ok {
        conn.Write(request.NewError(406, "No router_table"))
        return   
    }

    rt, err := ToRouterTable(rtmap)
    if err != nil {
        conn.Write(request.NewError(406, fmt.Sprintf("Unparsable router table (%s)", err)))
        return
    }

    err = partitioner.SetRouterTable(rt)
    if err != nil {
        conn.Write(request.NewError(406, fmt.Sprintf("Unable to set router table (%s)", err)))
        return
    }
    conn.Write(request.NewResponse())
}

func Lock(request *cheshire.Request, conn cheshire.Connection) {
    err := partitioner.Lock()
    if err != nil {
        conn.Write(request.NewError(406, fmt.Sprintf("Unable to lock (%s)", err)))
        return
    }
    conn.Write(request.NewResponse())
}

func Unlock(request *cheshire.Request, conn cheshire.Connection) {
    err := partitioner.Unlock()
    if err != nil {
        conn.Write(request.NewError(406, fmt.Sprintf("Unable to unlock (%s)", err)))
        return
    }
    conn.Write(request.NewResponse())
}


func Data(request *cheshire.Request, conn cheshire.Connection) {  
    part, ok := request.Params().GetInt("partition")
    if !ok {
        conn.Write(request.NewError(406, fmt.Sprintf("partition param is manditory")))
        return   
    }

    remove := request.Params().MustBool("remove", false)
    dataChan := make(chan *dynmap.DynMap, 10)
    finishedChan := make(chan int)
    errorChan := make(chan error)
    go func() {
        for {
            select {
                case data := <- dataChan :
                    //send a data packet
                    res := request.NewResponse()
                    res.SetTxnStatus("continue")
                    res.Put("data", data)
                    conn.Write(res)
                case <- finishedChan :
                    res := request.NewResponse()
                    res.SetTxnStatus("complete")
                    conn.Write(res)
                    return
                case err := <- errorChan :        
                    conn.Write(request.NewError(406, fmt.Sprintf("Unable to unlock (%s)", err)))
                    return
            }
        }
    }()
    partitioner.Data(part, remove, dataChan, finishedChan, errorChan)
}