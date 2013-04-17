package main

import (
    "log"
    "github.com/trendrr/cheshire-golang/cheshire"
)

func main() {

    bootstrap := cheshire.NewBootstrapFile("example_config.yaml")
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
    cheshire.RegisterApi("/ping", "GET", pinger)
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
