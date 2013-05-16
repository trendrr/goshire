package client

import (
	"log"
	"testing"
	"github.com/trendrr/cheshire-golang/cheshire"
	"time"
)

func TestHttpClient(t *testing.T) {
	client := NewHttp("localhost:8010")
	res, err := client.ApiCallSync(cheshire.NewRequest("/ping", "GET"), 10*time.Second)
	log.Println(res)
	if err != nil {
		t.Errorf("error %s", err)
	}
}

func TestJsonClient(t *testing.T) {
	client := NewJson("localhost", 8009)
	client.Connect()
	defer client.Close()
	res, err := client.ApiCallSync(cheshire.NewRequest("/ping", "GET"), 10*time.Second)
	log.Println(res)
	if err != nil {
		t.Errorf("error %s", err)
	}
}

//go test -v github.com/trendrr/cheshire-golang/cheshire
func TestClient(t *testing.T) {
	//NOT actually a test ;)

	//assumes a running server on port 8009
	log.Println("HERE")
	client := NewHttp("localhost:8010")
	// client := NewJson("localhost", 8009)
	// client.Connect()
	defer client.Close()
	//warm it up
	res, err := client.ApiCallSync(cheshire.NewRequest("/ping", "GET"), 10*time.Second)
	if err != nil {
		t.Errorf("error %s")
	}
	log.Println(res)

	resChan := make(chan *cheshire.Response, 2)
	errorChan := make(chan error)
	total := 10000
	start := time.Now().Unix()
	go func() {

	    for i :=0; i < total; i++ {
	        client.ApiCall(cheshire.NewRequest("/ping", "GET"), resChan, errorChan)        
	    }
	    }()
	count := 0

	log.Println("Starting select!")
	for {
		select {
		case <-resChan:
			count++
			if count%500 == 0 {
				log.Printf("Pinged 500 more, total time: %d", (time.Now().Unix() - start))
			}

		case <-errorChan:

			log.Println(err)
		}

		if count == total {
			log.Println("FINISHED!")
			break
		}
	}

	log.Printf("Pinged %d in %d", total, (time.Now().Unix() - start))
}
