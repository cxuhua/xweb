package main

import (
	"github.com/cxuhua/xweb"
	"log"
	"mime/multipart"
	"net/http"
)

//source:"Body",表示JSON数据来自 form表单的Body字段
type JsonArgs struct {
	xweb.JSONArgs `source:"Body" json:"-"`
	A             string `json:"a" validate:"regexp=^b.*$"`
	B             int    `json:"b" validate:"min=1,max=50"`
}

func (this JsonArgs) Modal() xweb.IModel {
	return &xweb.HTTPModel{}
}

func (this JsonArgs) Run(m *xweb.HTTPModel) {
	m.Error = "Run in test"
	m.Code = 100
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

type XModel struct {
	xweb.HTTPModel
	Title string
}

func (this *XModel) GetString() string {
	return this.Title
}

type SubDispatcher struct {
	xweb.HTTPDispatcher
	GET struct {
		Index xweb.IArgs `url:"/"`
	}
}

func (this *SubDispatcher) IndexHandler(render xweb.Render) {
	render.Text(http.StatusOK, "SubDispatcher.IndexHandler")
}

type MainDispatcher struct {
	xweb.HTTPDispatcher
	// SubDispatcher `url:"/sub" handler:"Logger"`
	Group struct {
		PostJson JsonArgs `url:"/json" method:"POST" render:"JSON"` //LoggerHandler,PostJsonHandler
		// PostForm FormArgs `url:"/form" method:"POST" render:"JSON"` //LoggerHandler,PostFormHandler
	} `url:"/post" handler:"Logger"`
	// Logger struct {
	// 	Index xweb.IArgs `url:"/" view:"test" render:"HTML"` //LoggerHandler,IndexHandler
	// }
	// Test FormArgs   `url:"/test" method:"POST"` //->TestHandler
	// List xweb.IArgs `url:"/list"`               //->ListHandler
}

func (this *MainDispatcher) PostJsonHandler(args JsonArgs, c xweb.IMVC) {
	c.SetModel(xweb.NewHTTPSuccess())
}

func (this *MainDispatcher) PostFormHandler(args FormArgs, render xweb.Render) {
	log.Println(args)
	log.Println(xweb.FormFileBytes(args.File))
	render.JSON(http.StatusOK, args)
}

func (this *MainDispatcher) IndexHandler(c xweb.IMVC) {
	m := &XModel{}
	m.Title = "这是个测试"
	// mvc.SetView("list")
	// mvc.SetRender()
	// mvc.SetStatus()
	c.SetModel(m)
}

func (this *MainDispatcher) TestHandler(render xweb.Render) {
	render.HTML(http.StatusOK, "test", nil)
}

func (this *MainDispatcher) ListHandler(render xweb.Render) {
	render.HTML(http.StatusOK, "test", nil)
}

func main() {
	xweb.UseDispatcher(new(MainDispatcher))
	xweb.ListenAndServe(":8010")
	// log.Println(xweb.ListenAndServeTLS(":8010", "rockygame.cn.crt", "rockygame.cn.key"))
}
