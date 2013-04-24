package partition

import (
    // "time"
    "github.com/trendrr/cheshire-golang/dynmap"
    "github.com/trendrr/cheshire-golang/cheshire"
    "fmt"
    "sync"
    "io/ioutil"
    "os"
    "log"
    "time"
)

type Partitioner interface {
    
    //Gets all the data for a specific partition
    //should send total # of items on the finished chanel when complete
    Data(partition int, dataChan chan *dynmap.DynMap, finished chan int, errorChan chan error)

    //Imports a data item
    SetData(partition int, data *dynmap.DynMap)

    //Deletes the requested partition
    DeleteData(partition int)
}

type DummyPartitioner struct {

}

func (this *DummyPartitioner) Data(partition int, deleteData bool, dataChan chan *dynmap.DynMap, finished chan int, errorChan chan error) {
    log.Printf("Requesting Data from dummy partitioner, ignoring.. (partition: %d),(deleteData: %s)", partition, deleteData)
}

func (this *DummyPartitioner) SetData(partition int, data *dynmap.DynMap) {
    log.Printf("Requesting SetData from dummy partitioner, ignoring.. (partition: %d),(data: %s)", partition, data)
}


type EventType string


type Event struct {
    EventType string


}


// Manages the router table and connections and things
type Manager struct {
    table *RouterTable
    lock sync.RWMutex
    connections map[string]cheshire.Client
    ServiceName string
    DataDir string
    //my entry id.  TODO: need a good way to autodiscover this..
    MyEntryId string
    partitioner Partitioner
    lockedPartitions map[int]bool
}

// Creates a new manager.  Uses the one or more seed urls to download the 
// routing table.
func NewManagerSeed(partitioner Partitioner, serviceName, dataDir, myEntryId string, seedHttpUrls []string) (*Manager, error) {
    //TODO: can we get the servicename from the routing table?

    manager := NewManager(partitioner, serviceName, dataDir, myEntryId)
    var err error
    for _,url := range(seedHttpUrls) {

        client := cheshire.NewHttpClient(url)
        tble, err := manager.RequestRouterTable(client)
        client.Close()
        if err != nil {
            //found a table..  woot
            manager.SetRouterTable(tble)
            return manager, nil
        }
    }

    if manager.table != nil {
        //uhh, I guess we sucessfully loaded it elsewhere
        return manager, nil
    }
    //we still return the manager since it is usable just doesnt have a routing table.
    return manager, err
}

//Creates a new manager.  will load the routing table from disk if
//it exists
func NewManager(partitioner Partitioner, serviceName, dataDir, myEntryId string) *Manager {
    manager := &Manager{
        table : nil,
        connections : make(map[string]cheshire.Client),
        DataDir : dataDir,
        ServiceName : serviceName,
        MyEntryId : myEntryId,
    }
    //attempt to load from disk
    err := manager.load()
    if err != nil {
        log.Println(err)
    }
    return manager
}

// Registers all the necessary controllers for partitioning.
func (this *Manager) RegisterControllers() error {
    setupPartitionControllers(this)
    return nil
}

// Puts a lock on the specified partition (locally only)
func (this *Manager) LockPartition(partition int) error {
    this.lock.Lock()
    defer this.lock.Unlock()
    this.lockedPartitions[partition] = true
    return nil
}

// // Locks all remote.
// func (this *Manager) LockPartitionRemote(partition int) error {
//     clients, err := this.Clients(partition)
//     if err != nil {
//         return err
//     }
//     for _,c := range(clients) {
//         req := cheshire.NewRequest("/chs/lock", "POST")
//         req.Params().Put("partitions", partition)


//         res, err := c.ApiCallSync(req, time.Second*10)
    
//         if err != nil {
//             //retry once
//             res, err = c.ApiCallSync(req, time.Second*10)
//             if err != nil {
//                 log.Printf("Error locking remote: %s", err)    
//             } else if res.StatusCode() != 200 {
//                 log.Printf("Error lock %s", res)
//             }
//         } else if res.StatusCode() != 200 {
//             log.Printf("Error lock %s", res)
//         }
//     }
//     //TODO: fail if all locks fail
//     return nil
// }

func (this *Manager) UnlockPartition(partition int) error {
    this.lock.Lock()
    defer this.lock.Unlock()
    delete(this.lockedPartitions, partition)
    return nil 
}


// func (this *Manager) UnlockPartitionRemote(partition int) error {
//     clients, err := this.Clients(partition)
//     if err != nil {
//         return err
//     }
//     for _,c := range(clients) {
//         req := cheshire.NewRequest("/chs/unlock", "POST")
//         req.Params().Put("partitions", partition)
//         res, err := c.ApiCallSync(req, time.Second*10)
//         if err != nil {
//             //retry once
//             res, err = c.ApiCallSync(req, time.Second*10)
//             if err != nil {
//                 log.Printf("Error locking remote: %s", err)    
//             } else if res.StatusCode() != 200 {
//                 log.Printf("Error lock %s", res)
//             }
//         } else if res.StatusCode() != 200 {
//             log.Printf("Error lock %s", res)
//         }
//     }
//     //TODO: fail if all locks fail
//     return nil
// } 

// Returns the list of partitions I am responsible for 
// returns an empty list if I am not responsible for any
func (this *Manager) MyPartitions() map[int]bool {
    this.lock.RLock()
    defer this.lock.RUnlock()
    if this.table == nil {
        return make(map[int]bool, 0)
    }
    me := this.table.MyEntry
    if me == nil {
        return make(map[int]bool, 0)   
    }
    return me.PartitionsMap
}

// Checks if this partition is my responsibility.
// This is also how we test for locked partitions.
//
// returns responsibility, locked
// 
func (this *Manager) MyResponsibility(partition int) (bool, bool) {
    this.lock.RLock()
    defer this.lock.RUnlock()

    par := this.MyPartitions()
    _, isMine := par[partition]
    locked, ok :=this.lockedPartitions[partition]
    if !ok {
        locked = false
    }
    return isMine, locked
}

//Sets the partitioner for this manager
//this should only be called once at initialization.  it is not threadsafe
func (this *Manager) SetPartitioner(par Partitioner) {
    this.partitioner = par
}

// Does a checkin with the requested client.  returns the 
// router table revision of the connection.  
func (this *Manager) Checkin(client cheshire.Client) (int64, error){
    response, err := client.ApiCallSync(cheshire.NewRequest("/chs/checkin", "GET"), 10*time.Second)
    if err != nil {
        return int64(0), err
    }
    revision := response.MustInt64("router_table_revision", int64(0))
    return revision, nil
}

//Request a new router table for the given client
//Does not call SetRouterTable
func (this *Manager) RequestRouterTable(client cheshire.Client) (*RouterTable, error) {
    response, err := client.ApiCallSync(cheshire.NewRequest("/chs/rt/get", "GET"), 10*time.Second)
    if err != nil {
        return nil, err
    }
    if response.StatusCode() != 200 {
        return nil, fmt.Errorf("Error from server %d %s", response.StatusCode(), response.StatusMessage())
    }

    mp, ok := response.GetDynMap("router_table")
    if !ok {
        return nil, fmt.Errorf("No router_table in response : %s", response)   
    }

    table,err := ToRouterTable(mp)
    if err != nil {
        return nil, err
    }
    return table, nil
}

//loads the stored version
func (this *Manager) load() error{
    bytes, err := ioutil.ReadFile(this.filename())
    if err != nil {
        return err
    }
    mp := dynmap.NewDynMap()
    err = mp.UnmarshalJSON(bytes)
    if err != nil {
        return err
    }
    table,err := ToRouterTable(mp)
    if err != nil {
        return err
    }
    this.SetRouterTable(table)    
    return nil
}

func (this *Manager) save() error {
    mp := this.table.ToDynMap()
    bytes,err := mp.MarshalJSON()
    if err != nil {
        return err
    }
    err = ioutil.WriteFile(this.filename(), bytes, 0644)
    return err
}

func (this *Manager) filename() string {
    if this.DataDir== "" {
        return fmt.Sprintf("%s.routertable", this.ServiceName)
    }
    return fmt.Sprintf("%s%s%s.routertable", this.DataDir, os.PathSeparator, this.ServiceName)
}

func (this *Manager) Clients(partition int) ([]cheshire.Client, error) {
    this.lock.RLock()
    defer this.lock.RUnlock()

    c, err := this.tableClients(this.table, partition)
    return c, err
}

//Returns the clients associated with this partition.
func (this *Manager) tableClients(table *RouterTable, partition int) ([]cheshire.Client, error) {
    

    clients := make([]cheshire.Client, 0)

    entries, err := table.PartitionEntries(partition)
    if err != nil {
        return clients, err
    }

    for _,entry := range(entries) {
        conn, ok := this.connections[entry.Id()]
        if !ok {
          log.Printf("no connection found for %s", entry)
        } 
        clients = append(clients, conn)
    }
    return clients, nil
}

//returns the current router table
func (this *Manager) RouterTable() (*RouterTable, error) {
    log.Println(this)
    log.Println(this.lock)

    this.lock.RLock()
    defer this.lock.RUnlock()
    if this.table == nil {
        return nil, fmt.Errorf("No Router Table available")
    }

    return this.table, nil
}


// //Rebalance 
// //Sync any data from remote, remove any partitions that
// //we are no longer in charge of.
// func (this *Manager) Rebalance(newtable, oldtable *RouterTable) {
   
//     newPartitions := make([]int, 0)

//     //yikes, this is an ugly way to do this..
//     for _,n := range(newtable.MyEntry.Partitions) {
//         contains := false
//         for _, o := range(oldtable.MyEntry.Partitions) {
//             if n == o {
//                 contains = true
//             }
//         }
//         if !contains {
//             newPartitions = append(newPartitions, n)
//         }
//     }


//     for _, n := range(newPartitions) {
//         //lock then move the data.
//         this.DataPull(n)

//     }

//     //now look for items removed in new table and delete those partitions.

// }



// func (this *Manager) DataPull(partition int) error {
//     this.LockPartition(partition)
//     this.LockPartitionRemote(partition)

//     defer this.UnlockPartition(partition)
//     defer this.UnlockPartitionRemote(partition)

//     clients, err := this.Clients(partition)
//     if err != nil {
//         log.Println("ERROR %s", err)
//         //TODO: umm, wtf?
//     }

//     var client cheshire.Client
//     //now choose a good client
//     for _, c := range(clients) {
//         //need to make sure this client is not me.
//         //TODO: we also need to make sure this partition existed on this node
//         //in the old table. (Or we need this to be a precondition!)
//         client = c
//     }

//     request := cheshire.NewRequest("/chs/data/pull", "GET")
//     request.SetTxnAcceptMulti()
//     responseChan := make(chan *cheshire.Response, 100)
//     errorChan := make(chan error)

//     client.ApiCall(request, responseChan, errorChan)
//     for {
//         select {
//             case response := <- responseChan :
//                 log.Printf("Data pull response: %s", response)
//                 if response.StatusCode() != 200 {
//                     //umm, what to do ?
//                     log.Printf("BAD Response: %s", response)
//                 }
//                 mp := response.ToDynMap()
//                 data, ok := mp.GetDynMap("data")
//                 if ok {
//                     this.partitioner.SetData(partition, data)
//                 }
//                 if response.TxnStatus() == "complete" {
//                     //yay!
//                     log.Println("FINISHED DATA PULL for partition %d", partition)
//                     return nil
//                 }
//             case err := <- errorChan :
//                 //TODO: try a different client?
//                 return err
//         }
//     }
//     return nil
// }

//sets a new router table
// returns the old router table, or error
func (this *Manager) SetRouterTable(table *RouterTable) (*RouterTable, error){
    this.lock.Lock()
    defer this.lock.Unlock()
    if this.table != nil {
        if this.table.Revision >= table.Revision {
            return nil, fmt.Errorf("Trying to set an older revision %d vs %d", this.table.Revision, table.Revision)
        }
    }

    //create a new map for connections
    c := make(map[string]cheshire.Client)
    for _,e := range(table.Entries) {
        key := e.Id()
        if key == this.MyEntryId {
            e.Self = true
            table.MyEntry = e
            continue
        }

        conn, ok := this.connections[key]
        if !ok {
            conn = this.createConnection(e)
        } 
        delete(this.connections, key)
        c[key] = conn
    }
    //now close any Clients for removed entries
    for _, client := range(this.connections) {
        client.Close()
    }
    oldTable := this.table
    this.connections = c
    this.table = table
    this.save()

    return oldTable, nil
}

func (this *Manager) createConnection(entry *RouterEntry) (cheshire.Client) {
    c, _ := cheshire.NewJsonClient(entry.Address, entry.JsonPort)
    return c
}