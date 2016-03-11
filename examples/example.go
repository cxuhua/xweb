package main

import (
	"github.com/cxuhua/xweb"
	"github.com/martini-contrib/render"
	"log"
	"net/http"
)

//args
type QueryArgs struct {
	xweb.URLArgs
}

type JsonArgs struct {
	xweb.JSONArgs `form:"Body" json:"-"`
	A             string `json:"a" validate:"regexp=^a$"`
	B             int    `json:"b" validate:"min=1,max=50"`
}

type MainDispatcher struct {
	xweb.HTTPDispatcher
	POST struct {
		PostJson JsonArgs `url:"/json" validate:"ToJSON"`
	} `url:"/post" handler:"Logger"`
	GET struct {
		Index QueryArgs `url:"/"` //if IndexHandler func miss,use HTTPDispatcher.HTTPHandler
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
	xweb.ListenAndServe(":8010")
	// log.Println(xweb.ListenAndServeTLS(":8010", "rockygame.cn.crt", "rockygame.cn.key"))
}

func main() {
	xweb.Daemon(server, "./", "server.pid", "server.log")
}
