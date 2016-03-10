package main

import (
	"github.com/cxuhua/xweb"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	"gopkg.in/validator.v2"
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

type SubDispatcher struct {
	xweb.Dispatcher
	POST struct {
		PostTest xweb.NullArgs `url:"/post/test"`
	}
}

func (this *BD) PostTest() {

}

type MainDispatcher struct {
	xweb.Dispatcher
	SubDispatcher //子分发器不能取名字

	POST struct {
		PostForm FormArg `url:"/form" before:"LogRequest"`
	} `url:"/post" handler:"NeedAuth"`

	POST2 struct {
		PostJson JsonArgs `url:"/json"`
		PostXml  XmlArgs  `url:"/xml" after:"PrintInfo"`
	} `url:"/post" handler:"LogRequest,NeedAuth" method:"POST"`

	GET struct {
		IndexHandler xweb.NullArgs `url:"/"`
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
	if err := validator.Validate(args); err != nil {
		render.JSON(http.StatusOK, err)
		return
	}
	log.Println(args)
	render.Text(http.StatusOK, args.A)
}

func (this *MainDispatcher) PostForm(args FormArg, render render.Render) {
	render.JSON(http.StatusOK, args)
}

func (this *MainDispatcher) IndexHandler(req *http.Request, render render.Render) {
	render.HTML(http.StatusOK, "test", nil)
}

func (this *MainDispatcher) Init(r martini.Router) error {
	return nil
}

func server() {
	log.SetFlags(log.Llongfile)
	m := martini.Classic()

	m.Use(render.Renderer(render.Options{
		IndentJSON: false,
	}))
	c := new(MainDispatcher)
	if err := xweb.Use(m, c); err != nil {
		panic(err)
	}
	m.RunOnAddr(":8010")
}

func main() {
	xweb.Daemon(server, "./", "server.pid", "server.log")
}
