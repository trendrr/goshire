Goshire
===============

### Overview 

Goshire is a GO framework that simplifies fast and scalable apis, services, and websites. 

Is it just another webframe?

* Yes, and no.  It certainly qualifies as a web framework, but that is not what makes it interesting.  It is a "services" framework.  Programmers can write controllers that are similar to controllers in other frameworks.  These controllers respond to both standard http requests AND strest requests.  


What is strest?

* STREST is a protocol that allows asyncronious communication between client and server.  A client will make a request packet which is sent to the server.  Each request has a client generated txn_id.  The server will respond to the request and either close the txn or say that more responses are on the way.  A more detailed description of the protocol is available here https://github.com/trendrr/strest-server/wiki/STREST-Protocol-Spec

### Api Example

Here is a simple example that shows the power of Goshire.  Streams of realtime data (i.e. firehoses) can be difficult to implement in other frameworks.  With Goshire it is simple as can be.


First we create a controller.  Controllers are simply a function that takes a Txn as input. 

```    
func Firehose(txn *cheshire.Txn) {
   for i := 0; true; i++ {
  		response := cheshire.NewResponse(txn)
  		response.Put("iteration", i)
  		response.Put("data", "This is a firehose, I never stop")
  		response.SetTxnStatus("continue") //set the status to continue so clients know to expect more responses
  		txn.Write(response)
  		time.Sleep(200 * time.Millisecond)
  	}
}
```

Then register the route

```
cheshire.RegisterApi("/firehose", "GET", Firehose)
```

Thats it!  Visiting the /firehose endpoint in your browser will print one new line of JSON every 200 milliseconds.  You can also connect to the endpoint via any of the STREST client libs available below.


## Web Example

Goshire is also a powerful web framework.  It includes template rendering via mustache. 
It includes middlewarehooks, flash messages, and sessions as well. 

The web controllers look similar to the api ones, but are registered as Html instead of api.

```
//an example html page
func Index(txn *cheshire.Txn) {
 //create a context map to be passed to the template
	context := make(map[string]interface{})
	context["content"] = "Welcome to the wild(ing)!"

 //set a flash message
	cheshire.Flash(txn, "success", "this is a flash message!")

 //Render index template in layout
	cheshire.RenderInLayout(txn, "/public/index.html", "/layouts/base.html", context)
}

```

Then register it


```
cheshire.RegisterHtml("/", "GET", Index)
```



###Cheshire includes:

* Mustache templates
* STREST JSON Protocol compliant https://github.com/trendrr/strest-server/wiki/STREST-Protocol-Spec
* Websockets


## Install

```
go get github.com/kylelemons/go-gypsy/yaml
go get github.com/hoisie/mustache
go get code.google.com/p/go.net/websocket
go get github.com/trendrr/goshire
```

##Quickstart

For a quick start boilerplate project, check out Wildling. http://github.com/mdennebaum/wildling

##GoDocs

Here are the go docs. http://godoc.org/github.com/trendrr/goshire/cheshire

##Credits

Cheshire builds on top of a bunch of smart peoples code. Below is credit where credit is due. 

* Inspired By - https://github.com/dustismo/cheshire
* Yaml  Support - http://github.com/kylelemons/go-gypsy/yaml
* Mustache Templates - http://github.com/hoisie/mustache
