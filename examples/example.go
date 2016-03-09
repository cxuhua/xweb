package main

import (
	"github.com/cxuhua/xweb"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	"log"
	"net/http"
)

type FormArg struct {
	xweb.FormArgs
	A string `form:"a"`
}

type JsonArgs struct {
	xweb.JsonArgs `field:"Body"`
	A             string `json:"a"`
	B             int    `json:"b"`
}

type XmlArgs struct {
	xweb.XmlArgs `field:"file"` //from field get data source
	XMLName      struct{}       `xml:"xml"` //xml root element name
	A            string         `xml:"a"`
	B            int            `xml:"b"`
}

type MainDispatcher struct {
	xweb.Dispatcher

	POST struct {
		PostForm FormArg  `url:"/post/form"`
		PostJson JsonArgs `url:"/post/json" `
		PostXml  XmlArgs  `url:"/post/xml" after:"PrintInfo"`
	} `handler:"NeedAuth"`

	GET struct {
		IndexHandler xweb.NullArgs `url:"/" before:"LogRequest"`
	}
}

func (this *MainDispatcher) NeedAuth() {
	log.Println("NeedAuth Handler")
}

func (this *MainDispatcher) PostXml(err binding.Errors, args XmlArgs, render render.Render) {
	log.Println("main handler", args, err)
	// render.Text(http.StatusOK, args.A)
}

func (this *MainDispatcher) PrintInfo(args XmlArgs) {
	log.Println("after handler", args)
}

func (this *MainDispatcher) PostBody(body xweb.BodyArgs, render render.Render) {
	log.Println(body)
	render.Text(http.StatusOK, string(body.Data))
}

func (this *MainDispatcher) PostJson(args JsonArgs, render render.Render) {
	log.Println(args)
	render.Text(http.StatusOK, args.A)
}

func (this *MainDispatcher) PostForm(args FormArg, render render.Render) {
	render.JSON(http.StatusOK, args)
}

func (this *MainDispatcher) IndexHandler(req *http.Request, render render.Render) {
	render.HTML(http.StatusOK, "test", nil)
}

func (this *MainDispatcher) Init(r martini.Router) {

}

func server() {
	log.SetFlags(log.Llongfile)
	m := martini.Classic()

	m.Use(render.Renderer(render.Options{
		IndentJSON: false,
	}))
	xweb.Use(m, new(MainDispatcher))
	m.RunOnAddr(":8010")
}

func main() {
	xweb.Daemon(server, "./", "server.pid", "server.log")
}
