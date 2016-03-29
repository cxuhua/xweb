package main

import (
	"fmt"
	"github.com/cxuhua/xweb"
	// "log"
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

type MainDispatcher struct {
	xweb.HTTPDispatcher
	Group struct {
		PostForm FormArgs `url:"/form" method:"POST"`
	} `url:"/post" handler:"Logger"`
	Logger struct {
		Index xweb.URLArgs `url:"/" view:"test"`
	}
}

func main() {
	xweb.UseDispatcher(new(MainDispatcher))
	xweb.ListenAndServe(":8010")
	// log.Println(xweb.ListenAndServeTLS(":8010", "rockygame.cn.crt", "rockygame.cn.key"))
}
