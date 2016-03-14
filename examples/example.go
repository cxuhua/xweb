package main

import (
	"github.com/cxuhua/xweb"
	"github.com/martini-contrib/render"
	"log"
	"net/http"
)

//form:"Body",表示JSON数据来自 form=Body字段
type JsonArgs struct {
	xweb.JSONArgs `form:"Body" json:"-"`
	A             string `json:"a" validate:"regexp=^a.*$"`
	B             int    `json:"b" validate:"min=1,max=50"`
}

type MainDispatcher struct {
	xweb.HTTPDispatcher
	POST struct {
		PostJson JsonArgs `url:"/json" validate:"ToJSON"` //ToJSON ToXML ToNEXT
	} `url:"/post" handler:"Logger"`
	GET struct {
		Index xweb.IArgs `url:"/"` //use IArgs not bind args handler
	} `handler:"Logger"`
}

func (this *MainDispatcher) PostJsonHandler(args JsonArgs, render render.Render) {
	log.Println(args)
	render.JSON(http.StatusOK, args)
}

func (this *MainDispatcher) IndexHandler(render render.Render) {
	render.HTML(http.StatusOK, "test", nil)
}

func server() {
	xweb.UseDispatcher(new(MainDispatcher))
	xweb.UseRender()
	xweb.ListenAndServe(":8010")
	// log.Println(xweb.ListenAndServeTLS(":8010", "rockygame.cn.crt", "rockygame.cn.key"))
}

func main() {
	xweb.Daemon(server, "./", "server.pid", "server.log")
}
