package partition

import (
    "github.com/trendrr/cheshire-golang/dynmap"
    "github.com/trendrr/cheshire-golang/cheshire"
    "time"
    "fmt"
)


type Partitioner interface {
    //Return the current routertable.
    RouterTable() (*RouterTable, error)

    //Set the router table.  through an error if
    //attempting to set an older partition table
    SetRouterTable(routerTable *RouterTable) error
    
    //Lock the data.  this is for rebalance operations
    //returns an error if lock failed.
    //This should be atomic, i.e. it either locks
    // all the requested partitions or fails
    // it should skip if a partition is already locked (not fail)
    Lock(partition []int) (error)
    // Unlock all requested partitions or return error
    // it should skip if the partition is already unlocked (not fail)
    Unlock(partition []int) (error)

    //Gets all the data for a specific partition
    //should send total # of items on the finished chanel when complete
    Data(partition int, deleteData bool, dataChan chan *dynmap.DynMap, finished chan int, errorChan chan error)
    SetData(partition int, data *dynmap.DynMap)
}

var partitioner Partitioner 

// Sets the partitioner and registers the necessary 
// controllers
func setupPartitionControllers(par Partitioner) {
    partitioner = par

    //register the controllers.
    cheshire.RegisterApi("/chs/rt/get", "GET", GetRouterTable)
    cheshire.RegisterApi("/chs/rt/set", "POST", SetRouterTable)
    cheshire.RegisterApi("/chs/lock", "POST", Lock)
    cheshire.RegisterApi("/chs/unlock", "POST", Unlock)
    cheshire.RegisterApi("/chs/checkin", "GET", Checkin)
}

func Checkin(request *cheshire.Request, conn cheshire.Connection) {
    table, err := partitioner.RouterTable()
    revision := int64(0)
    if err == nil {
        revision = table.Revision
    }
    response := request.NewResponse()
    response.Put("router_table_revision", revision)
    response.Put("ts", time.Now())
    conn.Write(response)
}

func GetRouterTable(request *cheshire.Request, conn cheshire.Connection) {
    tble, err := partitioner.RouterTable()
    if err != nil {
        conn.Write(request.NewError(506, fmt.Sprintf("Error: %s",err)))
        return
    }
    response := request.NewResponse()
    response.Put("router_table", tble.ToDynMap())
    conn.Write(response)
}

func SetRouterTable(request *cheshire.Request, conn cheshire.Connection) {
    rtmap, ok := request.Params().GetDynMap("router_table")
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

    partitions, ok := request.Params().GetIntSliceSplit("partitions", ",")
    if !ok {
        conn.Write(request.NewError(406, fmt.Sprintf("partitions param missing")))
        return
    }

    err := partitioner.Lock(partitions)
    if err != nil {
        //now send back an error
        conn.Write(request.NewError(406, fmt.Sprintf("Unable to lock partitions (%s)", err)))
        return
    }
    conn.Write(request.NewResponse())
}

func Unlock(request *cheshire.Request, conn cheshire.Connection) {
    partitions, ok := request.Params().GetIntSliceSplit("partitions", ",")
    if !ok {
        conn.Write(request.NewError(406, fmt.Sprintf("partitions param missing")))
        return
    }
    err := partitioner.Unlock(partitions)
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