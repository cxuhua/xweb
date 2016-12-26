package main

import (
	"fmt"
	"github.com/cxuhua/xweb"
	// "github.com/golang/protobuf/proto"
	// "github.com/cxuhua/xweb/martini"
	// "errors"
	// "io/ioutil"
	// "log"
	"os"
)

func (this *MessageReq) Handler(m *xweb.ProtoModel) (*MessageRep, error) {
	return &MessageRep{Id: "111", Count: 33}, nil
}

//返回json数据模型
type FormModel struct {
	xweb.JSONModel `json:"-"`
	A              string `json:"a"`
	B              string `json:"b"`
	C              string `json:"c"`
}

//source:"Body",表示JSON数据来自 form表单的Body字段
type FormArgs struct {
	xweb.FORMArgs `form:"-"`
	A             string        `form:"a" validate:"regexp=^b.*$"` //从表单获取参数并校验
	B             int           `form:"b" validate:"min=1,max=50"`
	URL           string        `url:"c"`            //url bind 参数
	Cookie        string        `cookie:"SessionId"` //从cookie bind参数
	File          xweb.FormFile `form:"file"`        //获取表单文件数据，必须使用multipart/form-data格式
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
	m.A = this.A
	m.B = fmt.Sprintf("%d", this.B)
	m.C = this.URL
}

type IndexArgs struct {
	xweb.URLArgs
	Q string `url:"q"` //?q=121&
	B string `url:"b"` //b=2323
}

//当接收到参数时用此方法处理参数
func (this *IndexArgs) Handler(c xweb.IMVC) {

	c.Logger().Error("Handler")

	// m := &FormModel{}
	// c.SetModel(m)

	// m.A = this.RemoteAddr()
	// m.B = this.Q
	// m.C = this.B
}

type MainDispatcher struct {
	xweb.HTTPDispatcher
	//Group中间件制定使用handler:"Logger"否则使用GroupHandler
	// Group struct {
	// 	// url 指定访问路径
	// 	// method 指定方式
	// 	// FormArgs 指定参数接收类
	// 	PostForm FormArgs `url:"/form" method:"POST"`
	// } `url:"/post" handler:"Logger"`
	// 支持多个中间件嵌套
	// Header2 struct {
	// 	Header1 struct {
	// 		Index IndexArgs `url:"/" view:"index"`
	// 	}
	// }
	//或是这种格式
	Header0 struct {
		Test      MessageReq `url:"/proto"`
		IndexArgs `url:"/" view:"index"`
	} `before:"Header2,Header1"`

	// Header1 IndexArgs `url:"/index" view:"index"`
}

func (this *MainDispatcher) Header1Handler(c xweb.IMVC) {
	c.Logger().Error("header1")
}

func (this *MainDispatcher) Header2Handler(c xweb.IMVC) {
	c.Logger().Error("header2")
}

func (this *MainDispatcher) Header0Handler(c xweb.IMVC) {
	c.Logger().Error("header0")
}

func main() {
	xweb.InitLogger(os.Stdout)
	xweb.UseRender()
	xweb.UseDispatcher(new(MainDispatcher))
	xweb.ListenAndServe(":8010")
	// log.Println(xweb.ListenAndServeTLS(":8010", "rockygame.cn.crt", "rockygame.cn.key"))
}
