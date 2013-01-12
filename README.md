cheshire-golang
===============

##Overview

Cheshire is a GO web framework that simplifies creating fast & scalable websites and apis. 
It's inspired by the cheshire java framework https://github.com/dustismo/cheshire. 

###Cheshire includes:

* Mustache templates
* STREST JSON Protocol compliant https://github.com/trendrr/strest-server/wiki/STREST-Protocol-Spec
* Websockets

###Near Term Roadmap Features:
* GZip Response Compression
* Extensible middleware
  * Pluggable authentication
  * Memcached caching


## Install

```
go get github.com/kylelemons/go-gypsy/yaml
go get github.com/hoisie/mustache
go get code.google.com/p/go.net/websocket
go get github.com/trendrr/cheshire-golang
```

##Quickstart

For a quick start boilerplate project, check out Wildling. http://github.com/mdennebaum/wildling

##Docs

Here are the go docs. http://godoc.org/github.com/trendrr/cheshire-golang/cheshire

##Credits

Cheshire builds on top of a bunch of smart peoples code. Below is credit where credit is due. 

* Inspired By - https://github.com/dustismo/cheshire
* Yaml  Support - http://github.com/kylelemons/go-gypsy/yaml
* Mustache Templates - http://github.com/hoisie/mustache
