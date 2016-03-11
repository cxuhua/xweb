package main

import (
	"github.com/cxuhua/xweb"
	"log"
)

//args
type QueryArgs struct {
	xweb.QueryArgs
}

// html/
// json/
// xml/
// text/

type ListModel struct {
	xweb.HtmlModel
	XXX string
}

//1
func (this QueryArgs) Model() xweb.IModel {
	return new(ListModel)
}

//4
func (this *ListModel) OutView() string {
	return this.XXX
}

//2
func (this *ListModel) Run(args QueryArgs) {
	this.XXX = "list"
}

type MainDispatcher struct {
	xweb.HTTPDispatcher
	GET struct {
		IndexHandler QueryArgs `url:"/"` //if IndexHandler func miss,use HTTPDispatcher.HTTPHandler
	} `handler:"LogRequest"`
}

func server() {
	log.SetFlags(log.Llongfile)
	xweb.UseDispatcher(new(MainDispatcher))
	xweb.ListenAndServe(":8010")
	// log.Println(xweb.ListenAndServeTLS(":8010", "rockygame.cn.crt", "rockygame.cn.key"))
}

func main() {
	xweb.Daemon(server, "./", "server.pid", "server.log")
}
