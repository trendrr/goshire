package cheshire

import (
	"fmt"
	"github.com/hoisie/mustache"
	"github.com/trendrr/goshire/dynmap"
	"log"
	"net/http"
)

type HtmlWriter struct {
	*HttpWriter
}

func (this *HtmlWriter) Type() string {
	return "html"
}

//Special filter for html lifecycle
type HtmlFilter interface {
	ControllerFilter

	//Allows you to hook in before anything is writen.  
	//makes it possible to
	//set headers cookies, ect.
	BeforeHtmlWrite(txn *Txn, writer http.ResponseWriter) bool
}

// Renders with a layout template.  
// 
// Layout should have {{content}} variable
func RenderInLayout(txn *Txn, path, layoutPath string, context map[string]interface{}) {
	viewsPath := txn.ServerConfig.MustString("http.html.view_directory", "")
	layPath := fmt.Sprintf("%s%s", viewsPath, layoutPath)
	templatePath := fmt.Sprintf("%s%s", viewsPath, path)
	writeResponse(txn, "text/html", mustache.RenderFileInLayout(templatePath, layPath, contxt(txn, context)))
}

func Render(txn *Txn, path string, context map[string]interface{}) {
	viewsPath := txn.ServerConfig.MustString("http.html.view_directory", "")
	templatePath := fmt.Sprintf("%s%s", viewsPath, path)
	writeResponse(txn, "text/html", mustache.RenderFile(templatePath, contxt(txn, context)))
}

func Flash(txn *Txn, severity, message string) {
	d := dynmap.NewDynMap()
	d.Put("severity", severity)
	d.Put("message", message)
	txn.Session.AddToSlice("_flash", d)
}

//Adds the special variables to the context.
func contxt(txn *Txn, context map[string]interface{}) map[string]interface{} {
	if context == nil {
		context = make(map[string]interface{})
	}

	context["request"] = txn.Request
	context["params"] = txn.Request.Params().Map

	flash, ok := txn.Session.GetDynMapSlice("_flash")
	if ok {
		//convert to map slice
		fl := make([]map[string]interface{}, 0)
		for _, f := range(flash) {
			fl = append(fl, f.Map)
		}
		context["flash"] = fl
	}
	txn.Session.Remove("_flash")
	return context
}

func writeResponse(txn *Txn, contentType string, value interface{}) {
	writer, err := ToHttpWriter(txn)
	if err != nil {
		SendError(txn, 400, fmt.Sprintf("Error: %s", err))
	}
	if !beforeWrite(txn, writer) {
		return
	}
	writer.Writer.Header().Set("Content-Type", contentType)
	writer.Writer.WriteHeader(200)
	writeContent(writer, value)
}

// call the html hooks.
// Always remember to call this!
func beforeWrite(txn *Txn, writer *HttpWriter) bool {
	//Call the filters.
	for _, filter := range txn.Filters {
		f, ok := filter.(HtmlFilter)
		if ok {
			if !f.BeforeHtmlWrite(txn, writer.Writer) {
				return false
			}
		}
	}
	return true
}

func ToHttpWriter(txn *Txn) (*HttpWriter, error) {
	writer, ok := txn.Writer.(*HttpWriter)
	if !ok {
		wr, ok := txn.Writer.(*HtmlWriter)
		if !ok {
			return writer, fmt.Errorf("Could not convert to httpwriter %s", txn.Writer)
		}
		writer = wr.HttpWriter
	}
	return writer, nil
}

//Issues a redirect (301) to the url
func Redirect(txn *Txn, url string) {
	writer, err := ToHttpWriter(txn)
	if err != nil {
		SendError(txn, 400, fmt.Sprintf("Error: %s", err))
	}
	if !beforeWrite(txn, writer) {
		return
	}
	writer.Writer.Header().Set("Location", url)
	writer.Writer.WriteHeader(301)
	writeContent(writer, "<html><head><title>Moved</title></head><body><h1>Moved</h1><p>This page has moved to <a href=\"%s\">%s</a>.</p></body></html>")
}

//write out an object 
//this assumes the header has been written already
func writeContent(writer *HttpWriter, value interface{}) {
	switch v := value.(type) {
	case string:
		writer.Writer.Write([]byte(v))
	case []byte:
		writer.Writer.Write(v)
	default:
		log.Println("Dont know how to write :", value)
		//TODO: response object, dynmap, map ect.
	}
}

type HtmlController struct {
	Handlers map[string]func(*Txn)
	Conf     *ControllerConfig
}

func NewHtmlController(route string, methods []string, handler func(*Txn)) *HtmlController {
	controller := &HtmlController{
		Handlers: make(map[string]func(*Txn)),
		Conf:     NewControllerConfig(route),
	}

	for _, m := range methods {
		controller.Handlers[m] = handler
	}
	return controller
}

func (this *HtmlController) Config() *ControllerConfig {
	return this.Conf
}

// We hijack the request so we can use the html writer instead of the regular http writer.
// mostly this is so the filters know this is of type="html" 
func (this *HtmlController) HttpHijack(writer http.ResponseWriter, req *http.Request, serverConfig *ServerConfig) {
	request := ToStrestRequest(req)
	conn := &HtmlWriter{
		&HttpWriter{
			Writer:       writer,
			HttpRequest:  req,
			Request:      request,
			ServerConfig: serverConfig,
		},
	}
	HandleRequest(request, conn, this, serverConfig)
}

func (this *HtmlController) HandleRequest(txn *Txn) {
	handler := this.Handlers[txn.Request.Method()]
	if handler == nil {
		handler = this.Handlers["ALL"]
	}
	if handler == nil {
		log.Println("Error, not found ", txn.Request.Uri())
		//not found!
		SendError(txn, 404, "Not found")
		return
	}
	if txn.Type() != "html" {
		SendError(txn, 400, "not an html connection")
		return
	}
	handler(txn)
}

// Allows us to use the fast static file handler built into golang standard lib
// Note that this skips the cheshire lifecycle so no middleware filters will be
// executed.
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

func (this *StaticFileController) HandleRequest(txn *Txn) {
	//Empty method, this is never called because we have the HttpHijack method in place
}

func (this *StaticFileController) HttpHijack(writer http.ResponseWriter, req *http.Request, serverConfig *ServerConfig) {
	this.Handler.ServeHTTP(writer, req)
}
