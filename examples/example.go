package main

import (
	"fmt"
	"github.com/cxuhua/xweb"
	"log"
)

type FormModel struct {
	xweb.HTTPModel
	A string
	B string
}

//source:"Body",表示JSON数据来自 form表单的Body字段
type FormArgs struct {
	xweb.FORMArgs `form:"-"`
	A             string `form:"a" validate:"regexp=^b.*$"`
	B             int    `form:"b" validate:"min=1,max=50"`
}

//创建model时
func (this *FormArgs) Model() xweb.IModel {
	return &FormModel{}
}

//当校验失败时触发
// func (this *FormArgs) Error(m *xweb.ValidateModel, c xweb.IMVC) {
// 	log.Println(m)
// }

//当未定义处理函数时时触发用来处理函数
func (this *FormArgs) Handler(m *FormModel) {
	m.Error = "Run in test"
	m.Code = 1001
	m.A = this.A
	m.B = fmt.Sprintf("%d", this.B)
}

type IndexArgs struct {
	xweb.URLArgs
}

func (this *IndexArgs) Handler(c xweb.IMVC) {
	m := &FormModel{}
	m.A = this.RemoteAddr()
	c.SetModel(m)
}

type MainDispatcher struct {
	xweb.HTTPDispatcher
	Group struct {
		PostForm  FormArgs     `url:"/form" method:"POST"`
		PostForm2 xweb.URLArgs `url:"/get" method:"GET"`
	} `url:"/post" handler:"Logger"`
	Logger struct {
		Index IndexArgs    `url:"/" view:"test"`
		WX    xweb.URLArgs `url:"/list"`
	}
}

func (this *MainDispatcher) PostForm2Handler(args *xweb.URLArgs, c xweb.IMVC) {
	log.Println(args, "a")
}

func (this *MainDispatcher) WXHandler(args *xweb.URLArgs, c xweb.IMVC) {
	log.Println("WXHandler")
	m := &xweb.StringModel{}
	c.SetModel(m)
	m.Text = "a"
}

func main() {
	xweb.UseDispatcher(new(MainDispatcher))
	xweb.ListenAndServe(":8010")
	// log.Println(xweb.ListenAndServeTLS(":8010", "rockygame.cn.crt", "rockygame.cn.key"))
}
