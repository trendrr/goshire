package cheshire

import(
    "testing"
    "log"
    "time"
)

//go test -v github.com/trendrr/cheshire-golang/cheshire
func TestClient(t *testing.T) {
    //assumes a running server on port 8009
    log.Println("HERE")
    client, err := NewClient("localhost", 8009)
    if err != nil {
        log.Println(err)
        return
    }
    
    res, err := client.ApiCallSync(NewRequest("/ping", "GET"), 10*time.Second)
    log.Println(res)

    

    resChan := make(chan *Response, 2)
    errorChan := make(chan error)

    go func() {
        
        for i :=0; i < 100000; i++ {
            client.ApiCall(NewRequest("/ping", "GET"), resChan, errorChan)        
        }
        }()

    count := 0
        start := time.Now().Unix()
        log.Println("Starting select!")
        for {
            select {
            case <- resChan:
                count++ 
                log.Println(count)
                if count % 1000 ==0{
                    log.Println("Pinged 1k in %d", (time.Now().Unix()-start))
                }
                if count == 100000 {
                    return
                }
            case err :=<- errorChan:
                log.Println(err)
            }
        }


}
