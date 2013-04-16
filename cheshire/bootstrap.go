package cheshire

import (
	"log"
	"reflect"
	"runtime"
	"strings"
)

type Bootstrap struct {
	Conf *ServerConfig
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
	mp, ok := this.Conf.GetInt64("maxprocs")
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
	if this.Conf.Exists("http.static_files.route") {
		route, ok := this.Conf.GetString("http.static_files.route")
		if !ok {
			log.Println("Error initing static files: http.static_files.route")
			return
		}

		path, ok := this.Conf.GetString("http.static_files.directory")
		if !ok {
			log.Println("Error initing static files: http.static_files.directory")
			return
		}
		this.Conf.Register([]string{"GET"}, NewStaticFileController(route, path))
	}
}

func (this *Bootstrap) InitWebSockets() {
	if this.Conf.Exists("listeners.http.websockets.route") {
		route, ok := this.Conf.GetString("listeners.http.websockets.route")
		if ok {
			this.Conf.Register([]string{"GET", "POST", "PUT", "DELETE"}, NewWebsocketController(route, this.Conf))
		}
	}

}

func (this *Bootstrap) InitControllers() {
	for _, contr := range registerQueue {
		this.Conf.Register(contr.Methods, contr.C)
	}
}

func (this *Bootstrap) AddFilters(filters ...ControllerFilter) {
	for _,f := range(filters) {
		this.Conf.Filters = append(this.Conf.Filters, f)
	}
}

//
// a queue of controllers so we can register controllers 
// before the bootstrap is initialized
var registerQueue []controllerWrapper

type controllerWrapper struct {
	C       Controller
	Methods []string
}

// Registers a controller funtion for api calls 
func RegisterApi(route string, method string, handler func(*Request, Writer), filters ...ControllerFilter) {
	Register([]string{method}, NewController(route, []string{method}, handler))
}

// Registers a controller function for html pages  
func RegisterHtml(route string, method string, handler func(*Request, *HtmlWriter), filters ...ControllerFilter) {
	Register([]string{method}, NewHtmlController(route, []string{method}, handler))
}

// Registers a new controller
func Register(methods []string, controller Controller) {
	registerQueue = append(registerQueue, controllerWrapper{controller, methods})
}



func NewBootstrapFile(configPath string) *Bootstrap {
	conf := NewServerConfigFile(configPath)
	return NewBootstrap(conf)
}

func NewBootstrap(config *ServerConfig) *Bootstrap {
	//create an instance of our application bootstrap
	bs := &Bootstrap{Conf: config}

	//return a pointer to our application
	return bs
}

func NewExtendedBootstrap(configPath string, extentions []func(conf *ServerConfig)) *Bootstrap {
	//create and run the default bootstrap
	bs := NewBootstrapFile(configPath)

	//loop over the bootstrap extentions
	for i := 0; i < len(extentions); i++ {
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
	log.Println("********** Starting Cheshire **************")
	//now start listening.
	if this.Conf.Exists("http.port") {
		port, ok := this.Conf.GetInt("http.port")
		if !ok {
			log.Println("ERROR: Couldn't start http listener ", port)
		} else {
			go HttpListen(port, this.Conf)
		}
	}

	if this.Conf.Exists("json.port") {
		port, ok := this.Conf.GetInt("json.port")
		if !ok {
			log.Println("ERROR: Couldn't start json listener")
		} else {
			go JsonListen(port, this.Conf)
		}
	}

	//this just makes the current thread sleep.  kinda stupid currently.
	//but we should update to get messages from the listeners, like when a listener quites
	channel := make(chan string)
	val := <-channel
	log.Println(val)
}
