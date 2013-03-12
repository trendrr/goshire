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
)


// A router table.
// The table is considered generally immutable.  If any changes occur a new table should
// be generated and propagated.
type RouterTable struct {
    //The name of the service. should be unique.  (Example: "trendrrdb")
    Service string

    //The revision # of the router table
    //this should be always increasing so greater revision means more upto date router table
    //this should typically be a timestamp 
    Revision int64

    //total # of partitions
    TotalPartitions int

    //Replication Factory
    ReplicationFactor int

    //entries organized by partition
    //index in the array is the partition
    EntriesPartition []*RouterEntry

    //The unique entries
    Entries []*RouterEntry

    //serialized dynmap
    DynMap *dynmap.DynMap
}

func NewRouterTable(service string) *RouterTable {
    return &RouterTable {
        Service : service,
        DynMap : dynmap.NewDynMap(),
    }
}

// Creates a new router table from the dynmap passed in
func ToRouterTable(mp *dynmap.DynMap) (*RouterTable, error) {
    t := &RouterTable{}

    var ok bool
    t.Service, ok = mp.GetString("service")
    if !ok {
        return nil, fmt.Errorf("No Service in the table %s", mp)
    }

    t.Revision, ok = mp.GetInt64("revision")
    if !ok {
        return nil, fmt.Errorf("No Revision in the table %s", mp)   
    }

    t.TotalPartitions, ok = mp.GetInt("total_partitions")
    if !ok {
        return nil, fmt.Errorf("No total_partitions in the table %s", mp)
    }

    t.ReplicationFactor, ok = mp.GetInt("replication_factor")
    if !ok {
        return nil, fmt.Errorf("No replication_factor in the table %s", mp)
    }

    //fill the entries
    t.Entries = make([]*RouterEntry, 0)
    entryMaps, ok := mp.GetDynMapSlice("entries")
    if !ok {
        return nil, fmt.Errorf("Bad entries in the table %s", mp)      
    }
    for _, em := range(entryMaps) {
        routerEntry, err := ToRouterEntry(em)
        if err != nil {
            return nil, err
        }
        t.Entries = append(t.Entries, routerEntry)
    }


    // set up the partition to entry mapping
    partitionCount := 0
    t.EntriesPartition = make([]*RouterEntry, t.TotalPartitions)
    for _, e := range(t.Entries) {
        for _,p := range(e.Partitions) {
            if p >= t.TotalPartitions {
                return nil, fmt.Errorf("Bad Partition entry (greater then total partitions): %s", e)
            }
            t.EntriesPartition[p] = e
            partitionCount++
        }
    }

    if partitionCount != t.TotalPartitions {
        return nil, fmt.Errorf("Bad table, some partitions un accounted for")
    }


    t.DynMap = t.ToDynMap()
    return t, nil
}


// Translate to a DynMap of the form:
// {
//     "service" : "trendrrdb",
//     "revision" : 898775762309309,
//     "total_partitions" : 256,
//     "entries" : [
//         {/*router entry 1*/},
//         {/*router entry 2*/}
//     ]
// }
func (this *RouterTable) ToDynMap() *dynmap.DynMap {
    if this.DynMap != nil && len(this.DynMap.Map) > 0 {
        return this.DynMap
    }
    mp := dynmap.NewDynMap()
    mp.Put("service", this.Service)
    mp.Put("revision", this.Revision)
    mp.Put("total_partitions", this.TotalPartitions)
    mp.Put("replication_factor", this.ReplicationFactor)
    entries := make([]*dynmap.DynMap, 0)
    for _,e := range(this.Entries) {
        entries = append(entries, e.ToDynMap())
    }
    mp.Put("entries", entries)
    this.DynMap = mp
    return mp
}

func (this *RouterTable) Entry(partition int) (*RouterEntry, error) {
    if partition >= this.TotalPartitions {
        return nil, fmt.Errorf("Requested partition %d is out of bounds (%d) ", partition, this.TotalPartitions )
    }
    return this.EntriesPartition[partition], nil
}


type RouterEntry struct {
    //The address of this entry
    Address string
    JsonPort int
    HttpPort int

    //Is this entry me?
    Self bool
    
    //list of partitions this entry is responsible for
    Partitions []int

    //this entry serialized as a DynMap
    DynMap *dynmap.DynMap
}

// Creates a new router entry from the dynmap passed in
func ToRouterEntry(mp *dynmap.DynMap) (*RouterEntry, error) {
    e := &RouterEntry{
        Self : false,
    }
    var ok bool
    e.Address, ok = mp.GetString("address")
    if !ok {
        return nil, fmt.Errorf("No Address in Entry: %s", mp)
    }

    e.JsonPort = mp.MustInt("ports.json", 0)
    e.HttpPort = mp.MustInt("ports.http", 0)

    e.Partitions, ok = mp.GetIntSlice("partitions")
    if !ok {
        return nil, fmt.Errorf("No Partitions in Entry: %s", mp)
    }
    e.DynMap = e.ToDynMap()
    return e, nil
}


// Translate to a DynMap of the form:
// {
//     "address" : "localhost",
//     "ports" : {
//         "json" : 8009,
//         "http" : 8010
//     }
//     "partitions" : [1,2,3,4,5,6,7,8,9]
// }
func (this *RouterEntry) ToDynMap() *dynmap.DynMap {
    if this.DynMap != nil && len(this.DynMap.Map) > 0 {
        return this.DynMap
    }

    mp := dynmap.NewDynMap()
    mp.Put("address", this.Address)
    if this.JsonPort > 0 {
        mp.PutWithDot("ports.json", this.JsonPort)
    }

    if this.HttpPort > 0 {
        mp.PutWithDot("ports.http", this.HttpPort)
    }
    
    mp.Put("partitions", this.Partitions)
    this.DynMap = mp
    return mp
}


// Manages the router table and connections and things
type Routing struct {
    table *RouterTable
    lock sync.RWMutex
    connections map[string]cheshire.Client
    ServiceName string
    DataDir string
}


func NewRouting(serviceName string, dataDir string) *Routing {
    routing := &Routing{
        table : nil,
        connections : make(map[string]cheshire.Client),
        DataDir : dataDir,
        ServiceName : serviceName,
    }
    //attempt to load from disk
    err := routing.load()
    if err != nil {
        log.Println(err)
    }
    return routing
}

//loads the stored version
func (this *Routing) load() error{
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

func (this *Routing) save() error {
    mp := this.table.ToDynMap()
    bytes,err := mp.MarshalJSON()
    if err != nil {
        return err
    }
    err = ioutil.WriteFile(this.filename(), bytes, 0644)
    return err
}

func (this *Routing) filename() string {
    if this.DataDir== "" {
        return fmt.Sprintf("%s.routertable", this.ServiceName)
    }
    return fmt.Sprintf("%s%s%s.routertable", this.DataDir, os.PathSeparator, this.ServiceName)
}

//gets the client and entry for a specific partition
func (this *Routing) Entry(partition int) (cheshire.Client, *RouterEntry, error) {
    this.lock.RLock()
    defer this.lock.RUnlock()

    entry, err := this.table.Entry(partition)
    if err != nil {
        return nil, nil, err
    }
    host,port := this.getHostPort(entry)
    key := fmt.Sprintf("%s:%d", host,port)
    conn, ok := this.connections[key]
    if !ok {
      return nil, nil, fmt.Errorf("no connection found")
    } 
    return conn, entry, nil
}

//sets a new router table
func (this *Routing) SetRouterTable(table *RouterTable) {
    this.lock.Lock()
    defer this.lock.Unlock()

    //create a new map for connections
    c := make(map[string]cheshire.Client)
    for _,e := range(table.Entries) {
        key := this.getConnectionKey(e)
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

    this.connections = c
    this.table = table
    this.save()
}

func (this *Routing) createConnection(entry *RouterEntry) (cheshire.Client) {
    host,port := this.getHostPort(entry)
    c, _ := cheshire.NewJsonClient(host, port)
    return c
}

func (this *Routing) getConnectionKey(entry *RouterEntry) string {
    host,port := this.getHostPort(entry)
    key := fmt.Sprintf("%s:%d", host,port)
    return key
}

//gets the host and port from an entry
func (this *Routing) getHostPort(entry *RouterEntry) (string, int) {
    //TODO: we want to allow http or other transports as a configuration option
    return entry.Address, entry.JsonPort
}