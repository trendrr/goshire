package cheshire


import (
    "reflect"
    "runtime"
    "strings"
    "github.com/trendrr/cheshire-golang/strest"
    "log"
    // "os"
)

type Bootstrap struct {
    Conf       *strest.ServerConfig
}

// Runs All methods that have prefix of Init
func (this *Bootstrap) RunInitMethods(target interface{}) {
    t := reflect.TypeOf(target)
    for i := 0; i < t.NumMethod(); i++ {
        method := t.Method(i)
        if strings.HasPrefix(method.Name, "Init") {
            reflect.ValueOf(target).Method(i).Call([]reflect.Value{})
        }
    }
}

// func (this *Bootstrap) InitMemcached() {
//     //init our memcached client
//     Memcached().Connect(this.Conf.Get("memcached.servers"))
// }

func (this *Bootstrap) InitProcs() {
    //lets tell our app how to best use the processors
    mp,ok := this.Conf.GetInt64("maxprocs")
    if ok {
        //set the max procs to our config setting
        runtime.GOMAXPROCS(int(mp))
    } else {
        //set the app to utilize all available cpus
        runtime.GOMAXPROCS(runtime.NumCPU())
    }
}

//this needs to be setup correctly to key off of the config yaml
func (this *Bootstrap) InitStaticFiles() {
    if this.Conf.Exists("listeners.http.static_files.route") {

        // http.Handle(
        //         this.Conf.Get("app.static_route"),
        //         http.StripPrefix(this.Conf.Get("app.static_route"),
        //             http.FileServer(http.Dir(this.Conf.Get("app.static_path")))))

    }
}


func (this *Bootstrap) InitWebSockets() {
    if this.Conf.Exists("listeners.http.websockets.route") {
        route, ok := this.Conf.GetString("listeners.http.websockets.route")
        if ok {
            this.Conf.Register(strest.NewWebsocketController(route, this.Conf))    
        }
    }

}

// Registers a controller funtion for api calls 
func (this *Bootstrap) RegisterApi(route string, methods []string, handler func(*strest.Request,strest.Connection)) {
    this.Conf.Register(strest.NewController(route, methods, handler))
}

// Registers a controller function for html pages  
func (this *Bootstrap) RegisterHtml(route string, methods []string, handler func(*strest.Request, *HtmlConnection)) {
    this.Conf.Register(NewHtmlController(route, methods, handler))
}

// Registers a new controller
func (this *Bootstrap) Register(controller strest.Controller) {
    this.Conf.Register(controller)
}

func NewBootstrapFile(configPath string) *Bootstrap {
    conf := NewServerConfigFile(configPath)
    return NewBootstrap(conf)
}

func NewBootstrap(config *strest.ServerConfig) *Bootstrap {
    //create an instance of our application bootstrap
    bs := &Bootstrap{Conf: config}

    //return a pointer to our application
    return bs
}

func NewExtendedBootstrap(configPath string,extentions []func(conf *strest.ServerConfig)) *Bootstrap {
    //create and run the default bootstrap
    bs := NewBootstrapFile(configPath)

    //loop over the bootstrap extentions
    for i := 0; i < len(extentions) ; i++ {
        //execute each extenion method
        extentions[i](bs.Conf)
    }

    //return a pointer to our application
    return bs
}


//starts listening in all the configured listeners
//this method does not return until all listeners exit (i.e. never).
func (this *Bootstrap) Start() {
    this.RunInitMethods(this)
    log.Println("**********")
    log.Println(this.Conf.Map)
    //now start listening.
    if this.Conf.Exists("http.port") {
        port, ok := this.Conf.GetInt("http.port")
        if !ok {
            log.Println("ERROR: Couldn't start http listener ", port)
        } else {
            go strest.HttpListen(port, this.Conf)    
        }
    }

    if this.Conf.Exists("json.port") {
        port, ok := this.Conf.GetInt("json.port")
        if !ok {
            log.Println("ERROR: Couldn't start json listener")
        } else {
            go strest.JsonListen(port, this.Conf)    
        }
    }

    //this just makes the current thread sleep.  kinda stupid currently.
    //but we should update to get messages from the listeners, like when a listener quites
    channel := make(chan string)
    val := <-channel 
    log.Println(val)
}