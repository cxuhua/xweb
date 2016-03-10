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

type AuthArg struct {
	xweb.FormArgs
	A string `form:"a"`
}

type JsonArgs struct {
	xweb.JsonArgs `field:"Body"`
	A             string `json:"a" validate:"min=1,max=2"`
	B             int    `json:"b"`
}

type XmlArgs struct {
	xweb.XmlArgs `field:"file"` //from field get data source
	XMLName      struct{}       `xml:"xml"` //xml root element name
	A            string         `xml:"a"`
	B            int            `xml:"b"`
}

/////////////////////////////////////////////////////////////////////////////////////////////////////

//args
type QueryArgs struct {
	xweb.QueryArgs
}

//bind model
func (this QueryArgs) Model() xweb.IModel {
	m := IdxModel{}
	m.A = 100
	return &m
}

//model
type IdxModel struct {
	xweb.Model
	A int
}

func (this *IdxModel) Invoke(req *http.Request) {
	log.Println("IdxModel Invoke", req.URL)
}

//bind view
func (this *IdxModel) View() string {
	return "test"
}

////////////////////////////////////////////////////////////////////////////////////////////////////

type SubDispatcher struct {
	GET struct {
		PostTest xweb.QueryArgs `url:"/test"`
	}
}

func (this *SubDispatcher) PostTest() {
	x := xweb.GetDispatcher((*SubDispatcher)(nil))
	log.Println(x)
}

type MainDispatcher struct {
	xweb.HttpDispatcher
	SubDispatcher //子分发器不能取名字

	POST struct {
		PostForm FormArg `url:"/form" before:"LogRequest"`
	} `url:"/post" handler:"NeedAuth"`

	POST2 struct {
		PostJson JsonArgs `url:"/json"`
		PostXml  XmlArgs  `url:"/xml" after:"PrintInfo"`
	} `url:"/post" handler:"LogRequest,NeedAuth" method:"POST"`

	GET struct {
		IndexHandler QueryArgs `url:"/"`
	} `handler:"LogRequest"`
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
	m := args.Model()
	render.JSON(http.StatusOK, m)
}

func (this *MainDispatcher) PostForm(args FormArg, render render.Render) {
	render.JSON(http.StatusOK, args)
}

func (this *MainDispatcher) IndexHandler(c martini.Context, args QueryArgs, render render.Render) {
	m := args.Model()
	render.HTML(http.StatusOK, m.View(), m)
}

func server() {
	log.SetFlags(log.Llongfile)
	xweb.Dispatcher(new(MainDispatcher))
	xweb.RunOnAddr(":8010")
}

func main() {
	xweb.Daemon(server, "./", "server.pid", "server.log")
}
