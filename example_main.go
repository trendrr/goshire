package main

import (
    "log"
    "github.com/trendrr/cheshire-golang/cheshire"
)

func main() {

    bootstrap := cheshire.NewBootstrapFile("example_config.yaml")
    log.Println("HERE:1")
    //a ping controller api controller.  
    pinger := func(request *cheshire.Request,conn cheshire.Connection) {
        // log.Printf("PING! %s", request.Strest.Params)
        response := request.NewResponse()
        response.Put("data", "PONG")
        // log.Printf("Sending REsponse: %s", response.TxnId())
        conn.Write(response)
    }
    //now register the api call
    cheshire.RegisterApi("/ping", "GET", pinger)
    log.Println("HERE:1")
    
    //an example html page
    four04 := func(request *cheshire.Request, conn *cheshire.HtmlConnection) {
        context := make(map[string]interface{})
        context["message"] = "this is a 404 page"
        conn.Render("/404.html", context)
    }
    cheshire.RegisterHtml("/404", "GET", four04)

    log.Println("Starting")
    //starts listening on all configured interfaces
    bootstrap.Start()
}
