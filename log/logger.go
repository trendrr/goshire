package cheshire

import(
    golog "log"
    "sync"
)




// This is a special logger.  It has the same methods as the standard golang Logger class,
// but allows you to register for updates.
// This allows us to expose the logging events easily via an api call.
type Logger struct {
    golog.Logger
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

var log = NewLogger()

func Fatal(v ...interface{}) {
    log.Fatal(v)
}
func Fatalf(format string, v ...interface{}) {
    log.Fatal(format, v)
}
func Fatalln(v ...interface{}) {
    log.Fatalln(v)
}
func Flags() int {
    return log.Flags()
}
func Panic(v ...interface{}) {
    log.Panic(v)
}
func Panicf(format string, v ...interface{}) {
    log.Panicf(format, v)
}
func Panicln(v ...interface{}) {
    log.Panicln(v)
}
func Prefix() string {
    return log.Prefix()
}
func Print(v ...interface{}) {
    log.Print(v)
}
func Printf(format string, v ...interface{}) {
    log.Printf(format, v)
}
func Println(v ...interface{}) {
    log.Println(v)
}
func SetFlags(flag int) {
    log.SetFlags(flag)
}
func SetPrefix(prefix string) {
    log.SetPrefix(prefix)
}

func Listen(eventchan chan LoggerEvent) {
    log.Listen(eventchan)
}

func Unlisten(eventchan chan LoggerEvent) {
    log.Unlisten(eventchan)
}
func Emit(eventType, eventMessage string) {
    log.Emit(eventType, eventMessage)
}

// Creates a new Logger object
func NewLogger() *Logger{
    logger := &Logger{
        inchan : make(chan LoggerEvent, 10),
        outchans : make([]chan LoggerEvent, 0),
        Type : "log",
    }

    logger.Logger = *golog.New(logger, "", 0)

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
    golog.Println(string(p))
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
            golog.Println("Removing channel from cheshire.Logger...")
        }
    }
    this.outchans = ch
}