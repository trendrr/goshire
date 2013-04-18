package main

import (
    "log"
    "github.com/trendrr/cheshire-golang/cheshire"
)

type DummyFilter struct {
    filterName string
}
func (this *DummyFilter) Before(*cheshire.Txn) bool {
    log.Printf("BEFORE! (%s) \n", this.filterName)
    return true
}
func (this *DummyFilter) After(*cheshire.Response, *cheshire.Txn) {
    log.Printf("AFTER! (%s) \n", this.filterName)   
}

func main() {

    //create a couple dummy filters

    //this one will get executed on every request.
    globalFilter := &DummyFilter{"global"}
    //this one will only get executed on ping requests.
    pingFilter := &DummyFilter{"ping"}

    bootstrap := cheshire.NewBootstrapFile("example_config.yaml")

    bootstrap.AddFilters(globalFilter)


    log.Println("HERE:1")
    //a ping controller api controller.  
    pinger := func(txn *cheshire.Txn) {
        // log.Printf("PING! %s", request.Strest.Params)
        response := cheshire.NewResponse(txn)
        response.Put("data", "PONG")
        // log.Printf("Sending REsponse: %s", response.TxnId())
        txn.Write(response)
    }
    //now register the api call
    cheshire.RegisterApi("/ping", "GET", pinger, pingFilter)
    log.Println("HERE:1")
    
    //an example html page
    four04 := func(txn *cheshire.Txn) {
        context := make(map[string]interface{})
        context["message"] = "this is a 404 page"
        cheshire.Render(txn, "/404.html", context)
    }
    cheshire.RegisterHtml("/404", "GET", four04)

    //an example redirect page
    redirect := func(txn *cheshire.Txn) {
        cheshire.Redirect(txn, "/ping")
    }
    cheshire.RegisterHtml("/redirect", "GET", redirect)


    log.Println("Starting")
    //starts listening on all configured interfaces
    bootstrap.Start()
}
