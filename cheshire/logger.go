package cheshire

import(
    "log"
    "sync"
)

// This is a special logger.  It has the same methods as the standard golang Logger class,
// but allows you to register for updates.
// This allows us to expose the logging events easily via an api call.
type Logger struct {
    log.Logger
    inchan chan LoggerEvent
    outchans []chan LoggerEvent
    lock sync.Mutex
    //The event type that will show up when using the log standard calls (default to "log")
    Type string
}

type LoggerEvent struct {
    Type string
    Message string
}

// Creates a new Logger object
func NewLogger() *Logger{
    logger := &Logger{
        inchan : make(chan LoggerEvent, 10),
        outchans : make([]chan LoggerEvent, 0),
        Type : "log",
    }

    logger.Logger = *log.New(logger, "", 0)

    go func(logger *Logger) {
        for {
            e := <- logger.inchan
            logger.lock.Lock()
            for _, out := range(logger.outchans) {
                go func(event LoggerEvent) {
                    select {
                        case out <- event:
                        default: //the channel is unavail. 
                            // we assume the channel owner will clean up..
                    }
                }(e)
            }
            logger.lock.Unlock()
        }
    }(logger)   
    return logger
}

//Conform to Writer interface.  will write events as "log", string
// This allows us to use this in Logger object.
//Will Effectively never throw an error
func (this *Logger) Write(p []byte) (n int, err error) {
    this.Emit("log", string(p))
    //also log to stdout
    log.Println(string(p))
    return len(p), nil
}


// Emits this message to all the listening channels
func (this *Logger) Emit(eventType, eventMessage string) {
    this.inchan <- LoggerEvent{Type: eventType, Message:eventMessage}
}

func (this *Logger) Listen(eventchan chan LoggerEvent) {
    this.lock.Lock()
    defer this.lock.Unlock()
    this.outchans = append(this.outchans, eventchan)
}

func (this *Logger) Unlisten(eventchan chan LoggerEvent) {
    this.lock.Lock()
    defer this.lock.Unlock()
    this.remove(eventchan)
}

func (this *Logger) remove(eventchan chan LoggerEvent) {
    ch := make([]chan LoggerEvent, 0)
    for _, c := range(this.outchans) {
        if c != eventchan {
            ch = append(ch, c)
        } else {
            log.Println("Removing channel from cheshire.Logger...")
        }
    }
    this.outchans = ch
}