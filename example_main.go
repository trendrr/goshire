package main

import (
    "log"
    "cheshire"
    "strest"
)

func main() {

    bootstrap := cheshire.NewBootstrapFile("config.yaml")

    //a ping controller api controller.  
    pinger := func(request *strest.Request,conn strest.Connection) {
        response := strest.NewResponse(request)
        response.Put("data", "PONG")
        conn.Write(response)
    }
    //now register the api call
    bootstrap.RegisterApi("/ping", []string{"GET"}, pinger)

    //an example html page
    four04 := func(request *strest.Request, conn *cheshire.HtmlConnection) {
        context := make(map[string]interface{})
        context["message"] = "this is a 404 page"
        conn.Render("/404.html", context)
    }
    bootstrap.RegisterHtml("/404", []string{"GET"}, four04)

    log.Println("Starting")
    //starts listening on all configured interfaces
    bootstrap.Start()
}
