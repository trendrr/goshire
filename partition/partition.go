package partition

import (
    "github.com/trendrr/cheshire-golang/dynmap"
    "github.com/trendrr/cheshire-golang/cheshire"
    "time"
    "fmt"
    "log"
)

var manager *Manager 

// Sets the partitioner and registers the necessary 
// controllers
func setupPartitionControllers(man *Manager) {
    manager = man

    //register the controllers.
    cheshire.RegisterApi("/chs/rt/get", "GET", GetRouterTable)
    cheshire.RegisterApi("/chs/rt/set", "POST", SetRouterTable)
    cheshire.RegisterApi("/chs/lock", "POST", Lock)
    cheshire.RegisterApi("/chs/unlock", "POST", Unlock)
    cheshire.RegisterApi("/chs/checkin", "GET", Checkin)
    // cheshire.RegisterApi("/chs/data/pull", "GET", DataPull)
}

func Checkin(txn *cheshire.Txn) {
    table, err := manager.RouterTable()
    revision := int64(0)
    if err == nil {
        revision = table.Revision
    }
    response := cheshire.NewResponse(txn)
    response.Put("router_table_revision", revision)
    response.Put("ts", time.Now())
    txn.Write(response)
}

func GetRouterTable(txn *cheshire.Txn) {
    log.Println("GetRouterTable")
    tble, err := manager.RouterTable()
    if err != nil {
        cheshire.SendError(txn, 506, fmt.Sprintf("Error: %s",err))
        return
    }
    response := cheshire.NewResponse(txn)
    response.Put("router_table", tble.ToDynMap())
    txn.Write(response)
}

func SetRouterTable(txn *cheshire.Txn) {
    rtmap, ok := txn.Params().GetDynMap("router_table")
    if !ok {
        cheshire.SendError(txn, 406, "No router_table")
        return   
    }

    rt, err := ToRouterTable(rtmap)
    if err != nil {
        cheshire.SendError(txn, 406, fmt.Sprintf("Unparsable router table (%s)", err))
        return
    }

    _, err = manager.SetRouterTable(rt)
    if err != nil {
        cheshire.SendError(txn, 406, fmt.Sprintf("Unable to set router table (%s)", err))
        return
    }
    response := cheshire.NewResponse(txn)
    txn.Write(response)
}

func Lock(txn *cheshire.Txn) {

    partition, ok := txn.Params().GetInt("partition")
    if !ok {
        cheshire.SendError(txn, 406, fmt.Sprintf("partition param missing"))
        return
    }

    err := manager.LockPartition(partition)
    if err != nil {
        //now send back an error
        cheshire.SendError(txn, 406, fmt.Sprintf("Unable to lock partitions (%s)", err))
        return
    }
    response := cheshire.NewResponse(txn)
    txn.Write(response)
}

func Unlock(txn *cheshire.Txn) {
    partition, ok := txn.Params().GetInt("partition")
    if !ok {
        cheshire.SendError(txn, 406, fmt.Sprintf("partition param missing"))
        return
    }

    err := manager.UnlockPartition(partition)
    if err != nil {
        //now send back an error
        cheshire.SendError(txn, 406, fmt.Sprintf("Unable to lock partitions (%s)", err))
        return
    }
    response := cheshire.NewResponse(txn)
    txn.Write(response)
}


func Data(txn *cheshire.Txn) {  
    part, ok := txn.Params().GetInt("partition")
    if !ok {
        cheshire.SendError(txn, 406, fmt.Sprintf("partition param is manditory"))
        return   
    }

    remove := txn.Params().MustBool("remove", false)
    dataChan := make(chan *dynmap.DynMap, 10)
    finishedChan := make(chan int)
    errorChan := make(chan error)
    go func() {
        for {
            select {
                case data := <- dataChan :
                    //send a data packet
                    res := cheshire.NewResponse(txn)
                    res.SetTxnStatus("continue")
                    res.Put("data", data)
                    txn.Write(res)
                case <- finishedChan :
                    res := cheshire.NewResponse(txn)
                    res.SetTxnStatus("complete")
                    txn.Write(res)
                    return
                case err := <- errorChan :        
                    cheshire.SendError(txn, 406, fmt.Sprintf("Unable to unlock (%s)", err))
                    return
            }
        }
    }()
    manager.partitioner.Data(part, remove, dataChan, finishedChan, errorChan)
}