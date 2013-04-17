package cheshire

import (
	"log"
	"testing"
	"time"
)

func TestHttpClient(t *testing.T) {
	client := NewHttpClient("localhost:8010")
	res, err := client.ApiCallSync(NewRequest("/ping", "GET"), 10*time.Second)
	log.Println(res)
	if err != nil {
		t.Errorf("error %s", err)
	}
}

func TestJsonClient(t *testing.T) {
	client, err := NewJsonClient("localhost", 8009)
	if err != nil {
		log.Println(err)
		return
	}
	defer client.Close()

	res, err := client.ApiCallSync(NewRequest("/ping", "GET"), 10*time.Second)
	log.Println(res)
	if err != nil {
		t.Errorf("error %s", err)
	}
}

//go test -v github.com/trendrr/cheshire-golang/cheshire
// func TestClient(t *testing.T) {
// 	//NOT actually a test ;)

// 	//assumes a running server on port 8009
// 	log.Println("HERE")
// 	client, err := NewClient("localhost", 8009)
// 	if err != nil {
// 		log.Println(err)
// 		return
// 	}
// 	//warm it up
// 	res, err := client.ApiCallSync(NewRequest("/ping", "GET"), 10*time.Second)
// 	log.Println(res)

// 	resChan := make(chan *Response, 2)
// 	errorChan := make(chan error)
// 	total := 10000000
// 	start := time.Now().Unix()
// 	go func() {

// 	    for i :=0; i < total; i++ {
// 	        client.ApiCall(NewRequest("/ping", "GET"), resChan, errorChan)        
// 	    }
// 	    }()
// 	count := 0

// 	log.Println("Starting select!")
// 	for {
// 		select {
// 		case <-resChan:
// 			count++
// 			if count%5000 == 0 {
// 				log.Printf("Pinged 5k more, total time: %d", (time.Now().Unix() - start))
// 			}

// 		case <-errorChan:

// 			// log.Println(err)
// 		}

// 		if count == total {
// 			log.Println("FINISHED!")
// 			break
// 		}
// 	}

// 	log.Printf("Pinged %d in %d", total, (time.Now().Unix() - start))
// }
