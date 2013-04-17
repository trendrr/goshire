package cheshire

import (
	"fmt"
	"github.com/hoisie/mustache"
	"log"
	"net/http"
)

type HtmlWriter struct {
	*HttpWriter
}

func (this *HtmlWriter) Type() string {
	return "html"
}

// Renders with a layout template.  
// 
// Layout should have {{content}} variable
func (this *HtmlWriter) RenderInLayout(path, layoutPath string, context map[string]interface{}) {
	viewsPath := this.ServerConfig.MustString("http.html.view_directory", "")
	layPath := fmt.Sprintf("%s%s", viewsPath, layoutPath)
	templatePath := fmt.Sprintf("%s%s", viewsPath, path)
	this.WriteResponse("text/html", mustache.RenderFileInLayout(templatePath, layPath, this.context(context)))
}

func (this *HtmlWriter) Render(path string, context map[string]interface{}) {
	viewsPath := this.ServerConfig.MustString("http.html.view_directory", "")
	templatePath := fmt.Sprintf("%s%s", viewsPath, path)
	this.WriteResponse("text/html", mustache.RenderFile(templatePath, this.context(context)))
}

//Adds the special variables to the context.
func (this *HtmlConnection) context(context map[string]interface{}) (map[string]interface{}) {
	context["request"] = this.Request
	context["params"] = this.Request.Params().Map
	return context
}

func (this *HtmlWriter) WriteResponse(contentType string, value interface{}) {

	this.Writer.Header().Set("Content-Type", contentType)
	this.Writer.WriteHeader(200)
	this.WriteContent(value)
}

//Issues a redirect (301) to the url
func (this *HtmlWriter) Redirect(url string) {
	this.Writer.Header().Set("Location", url)
	this.Writer.WriteHeader(301)
	this.WriteContent("<html><head><title>Moved</title></head><body><h1>Moved</h1><p>This page has moved to <a href=\"%s\">%s</a>.</p></body></html>")
}

//write out an object 
//this assumes the header has been written already
func (this *HtmlWriter) WriteContent(value interface{}) {
	switch v := value.(type) {
	case string:
		this.Writer.Write([]byte(v))
	case []byte:
		this.Writer.Write(v)
	default:
		log.Println("Dont know how to write :", value)
		//TODO: response object, dynmap, map ect.
	}
}

type HtmlController struct {
	Handlers map[string]func(*Request, *HtmlWriter)
	Conf     *ControllerConfig
}

func NewHtmlController(route string, methods []string, handler func(*Request, *HtmlWriter)) *HtmlController {
	controller := &HtmlController{
		Handlers: make(map[string]func(*Request, *HtmlWriter)), 
		Conf: NewControllerConfig(route),
	}

	for _, m := range methods {
		controller.Handlers[m] = handler
	}
	return controller
}

func (this *HtmlController) Config() *ControllerConfig {
	return this.Conf
}

func (this *HtmlController) HandleRequest(request *Request, conn Writer) {
	handler := this.Handlers[request.Method()]
	if handler == nil {
		handler = this.Handlers["ALL"]
	}
	if handler == nil {
		log.Println("Error, not found ", request.Uri())
		//not found!
		//TODO: send 404 page.
		return
	}

	connection, ok := conn.(*HttpWriter)
	if !ok {
		log.Println("not an http connection")
		//not an http connect
		//TODO: send error
		return
	}
	htmlconn := &HtmlWriter{connection}
	handler(request, htmlconn)
}

type StaticFileController struct {
	Route   string
	Path    string
	Conf    *ControllerConfig
	Handler http.Handler
}

// initial the handler via http.StripPrefix("/tmpfiles/", http.FileServer(http.Dir("/tmp")))
func NewStaticFileController(route string, path string) *StaticFileController {
	handler := http.StripPrefix(route, http.FileServer(http.Dir(path)))
	def := &StaticFileController{Handler: handler, Path: path, Route: route, Conf: NewControllerConfig(route)}
	return def
}

func (this *StaticFileController) Config() *ControllerConfig {
	return this.Conf
}

func (this StaticFileController) HandleRequest(*Request, Writer) {
	//Empty method, this is never called because we have the HttpHijack method in place
}

func (this StaticFileController) HttpHijack(writer http.ResponseWriter, req *http.Request) {
	this.Handler.ServeHTTP(writer, req)
}
