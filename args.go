package xweb

import (
	"net/http"
)

//req type
const (
	AT_NONE = iota
	AT_FORM
	AT_JSON
	AT_XML
	AT_URL //body use Query type parse
)

type IArgs interface {
	//是否校验参数
	IsValidate() bool
	//参数解析类型
	ReqType() int
	//返回默认的输出模型
	Model() IModel
	//参数校验失败
	Error(*ValidateModel, IMVC)
	//设置Request
	SetRequest(*http.Request)
	//获得远程Ip地址
	RemoteAddr() string
}

type Args struct {
	IArgs
	*http.Request
}

func (this *Args) RemoteAddr() string {
	return GetRemoteAddr(this.Request)
}

func (this *Args) SetRequest(req *http.Request) {
	this.Request = req
}

type URLArgs struct {
	Args
}

func (this *URLArgs) IsValidate() bool {
	return false
}

func (this *URLArgs) Error(m *ValidateModel, c IMVC) {
	v := &StringModel{Text: m.ToTEXT()}
	c.SetModel(v)
	c.SetRender(TEXT_RENDER)
}

func (this *URLArgs) ReqType() int {
	return AT_URL
}

func (this *URLArgs) Model() IModel {
	return &HtmlModel{}
}

type FORMArgs struct {
	Args
}

func (this *FORMArgs) IsValidate() bool {
	return true
}

func (this *FORMArgs) Error(m *ValidateModel, c IMVC) {
	c.SetModel(m)
	c.SetRender(JSON_RENDER)
}

func (this *FORMArgs) ReqType() int {
	return AT_FORM
}

func (this *FORMArgs) Model() IModel {
	return &HTTPModel{}
}

type JSONArgs struct {
	Args
}

func (this *JSONArgs) IsValidate() bool {
	return true
}

func (this *JSONArgs) Error(m *ValidateModel, c IMVC) {
	c.SetModel(m)
	c.SetRender(JSON_RENDER)
}

func (this *JSONArgs) ReqType() int {
	return AT_JSON
}

func (this *JSONArgs) Model() IModel {
	return &HTTPModel{}
}

type XMLArgs struct {
	Args
}

func (this *XMLArgs) IsValidate() bool {
	return true
}

func (this *XMLArgs) Error(m *ValidateModel, c IMVC) {
	c.SetModel(m)
	c.SetRender(XML_RENDER)
}

func (this *XMLArgs) ReqType() int {
	return AT_XML
}

func (this *XMLArgs) Model() IModel {
	return &HTTPModel{}
}
