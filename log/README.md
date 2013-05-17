cheshire-log
===============

##Overview


A simple log replacement.

This logger allows you to easily subscribe to log events. 

To use simply change the log import path from to

```
    import ("github.com/trendrr/goshire/log")

    //then use as you would the regular log
    log.Println("Heres a log message")
```

To subscribe to log events:

```
    log.Listen(eventChannel)
```

To unsubscribe to log events:

```
    log.Unlisten(eventChannel)
```

