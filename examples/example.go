package main

import (
	"github.com/cxuhua/xweb"
	"github.com/martini-contrib/render"
	"log"
	"mime/multipart"
	"net/http"
)

//form:"Body",表示JSON数据来自 form=Body字段
type JsonArgs struct {
	xweb.JSONArgs `form:"Body" json:"-"`
	A             string `json:"a" validate:"regexp=^b.*$"`
	B             int    `json:"b" validate:"min=1,max=50"`
}

//是否校验参数 返回校验错误输出类型
func (this JsonArgs) ValType() int {
	return xweb.AT_NONE
}

type XX struct {
	Body string `form:"Body"`
}

type FormArgs struct {
	xweb.FORMArgs
	XX
	File *multipart.FileHeader `form:"file"`
}

type SubDispatcher struct {
	xweb.HTTPDispatcher
	GET struct {
		Index xweb.IArgs `url:"/"`
	}
}

func (this *SubDispatcher) IndexHandler(render render.Render) {
	render.Text(http.StatusOK, "SubDispatcher.IndexHandler")
}

type MainDispatcher struct {
	xweb.HTTPDispatcher
	// SubDispatcher `url:"/sub" handler:"Logger"`
	// Group         struct {
	// 	PostJson JsonArgs `url:"/json" method:"POST"` //LoggerHandler,PostJsonHandler
	// 	PostForm FormArgs `url:"/form" method:"POST"` //LoggerHandler,PostFormHandler
	// } `url:"/post" handler:"Logger"`
	Logger struct {
		Index xweb.IArgs `url:"/"` //LoggerHandler,IndexHandler
	}
	// Test FormArgs   `url:"/test" method:"POST"` //->TestHandler
	// List xweb.IArgs `url:"/list"` //->ListHandler
}

func (this *MainDispatcher) PostJsonHandler(args JsonArgs, render render.Render) {
	log.Println(args)
	render.JSON(http.StatusOK, args)
}

func (this *MainDispatcher) PostFormHandler(args FormArgs, render render.Render) {
	log.Println(args)
	log.Println(xweb.FormFileBytes(args.File))
	render.JSON(http.StatusOK, args)
}

func (this *MainDispatcher) IndexHandler(render render.Render) {
	log.Println("MainDispatcher.IndexHandler")
	render.HTML(http.StatusOK, "test", nil)
}

func (this *MainDispatcher) TestHandler(render render.Render) {
	render.HTML(http.StatusOK, "test", nil)
}

func (this *MainDispatcher) ListHandler(render render.Render) {
	render.HTML(http.StatusOK, "test", nil)
}

func main() {
	xweb.UseDispatcher(new(MainDispatcher))
	xweb.ListenAndServe(":8010")
	// log.Println(xweb.ListenAndServeTLS(":8010", "rockygame.cn.crt", "rockygame.cn.key"))
}
