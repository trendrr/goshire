cheshire-log
===============

##Overview


A simple log replacement.

This logger allows you to easily subscribe to log events.  This makes it simple to publish log events via a cheshire endpoint.

To use simply change the log import path from to

```
    import ("github.com/trendrr/cheshire-golang/dynmap")

    //then use as you would the regular log
    log.Println("Heres a log message")
```

To subscribe to log events:

```
    log.Listen(eventChannel)
    //to unlisten
    log.Unlisten(eventChannel)
```
