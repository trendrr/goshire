package main

import (
    "log"
        "github.com/trendrr/goshire/cheshire"
    "time"
    c "github.com/trendrr/goshire/client"
    // "os"
    // "runtime/pprof"
    _ "net/http/pprof"
    "net/http"
    "runtime"
)

func asyncTest(client c.Client, total int) {

   

    resChan := make(chan *cheshire.Response, 20)
    errorChan := make(chan error, 200)

    start := time.Now().Unix()

    sent := total
    go func() {

        for i :=0; i < total; i++ {
            if i % 1000 == 0 {
                log.Printf("Sending %d", i)    
            }
            err := client.ApiCall(cheshire.NewRequest("/ping", "GET"), resChan, errorChan)        
            if err != nil {
                log.Printf("apicall error %s", err)
                sent--
            } 
            // if i % 2 == 0 {
            //     time.Sleep(1 * time.Millisecond)
            // }
        }
        }()
    count := 0

    log.Println("Starting select!")
    for {
        select {
        case res := <-resChan:
            count++
            if count % 1000 == 0 {
                log.Printf("Recieved 1000 more, total %d, total time: %d", count, (time.Now().Unix() - start))
                log.Printf("RESULT %s", res)
            }

        case err :=<-errorChan:
            count++
            log.Printf("ERROR FROM CHAN %s", err)
        }

        if count == sent {
            log.Println("FINISHED!")
            break
        }
    }

    log.Printf("Pinged %d in %d", total, (time.Now().Unix() - start))

}

func syncTest(client c.Client, total int) {

    start := time.Now().Unix()



    for i :=0; i < total; i++ {
        if i % 1000 == 0 {
            log.Printf("Sending %d", i)    
        }
        _, err := client.ApiCallSync(cheshire.NewRequest("/ping", "GET"), 2 * time.Second)        
        if err != nil {
            log.Printf("apicall error %s", err)
        } 
    }

    log.Printf("Pinged %d in %d", total, (time.Now().Unix() - start))

}

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU())
//assumes a running server on port 8009
    log.Println("HERE")

 // start the http server for profiling
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()

    // client := NewHttp("localhost:8010")
    client := c.NewBin("localhost", 8011)
    // client := c.NewJson("localhost", 8009)
   
    // with Poolsize = 10 and maxInflight = 500
    // 2013/05/16 12:56:28 Pinged 100000 in 38 seconds

    client.PoolSize = 10
    client.MaxInFlight = 500
    err := client.Connect()
    if err != nil {
        log.Println(err)
    }
    defer client.Close()
    //warm it up
    res, err := client.ApiCallSync(cheshire.NewRequest("/ping", "GET"), 10*time.Second)
    if err != nil {
        log.Printf("error %s")
    }
    log.Println(res)
    
    asyncTest(client, 1000000)

}