package xweb

import (
	"errors"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"reflect"
)

var (
	FormFileType = reflect.TypeOf(FormFile{})
)

type FormFile struct {
	*multipart.FileHeader
}

func (this FormFile) Write(data []byte, pfunc func(string) string) (string, error) {
	md5 := MD5Bytes(data)
	path := pfunc(md5)
	//exists file check
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		return md5, nil
	}
	return md5, ioutil.WriteFile(path, data, 0666)
}

//read file data
func (this FormFile) ReadAll() ([]byte, error) {
	if this.FileHeader == nil {
		return nil, errors.New("file header nil")
	}
	f, err := this.FileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ioutil.ReadAll(f)
}

//req type
const (
	AT_NONE = iota
	AT_FORM //表单数据解析  	use:form tag
	AT_JSON //json数据解析	use:json tag
	AT_XML  //xml数据解析	use:xml tag
	AT_URL  //url可以和以上结构体混用 use:url tag
)

type IArgs interface {
	//初始化
	Init(*http.Request)
	//是否校验参数
	IsValidate() bool
	//参数校验失败
	Validate(*ValidateModel, IMVC)
	//参数解析类型
	ReqType() int
	//返回默认的输出模型
	Model() IModel
	//获得远程Ip地址
	RemoteAddr() string
}

type xArgs struct {
	IArgs
	*http.Request
}

func (this *xArgs) Init(req *http.Request) {
	this.Request = req
}

func (this *xArgs) Model() IModel {
	return &HtmlModel{}
}

func (this *xArgs) ReqType() int {
	return AT_NONE
}

func (this *xArgs) RemoteAddr() string {
	return GetRemoteAddr(this.Request)
}

func (this *xArgs) IsValidate() bool {
	return true
}

func (this *xArgs) Validate(m *ValidateModel, c IMVC) {
	v := &StringModel{Text: m.ToTEXT()}
	c.SetModel(v)
	c.SetRender(TEXT_RENDER)
}

func (this *xArgs) HttpBody() ([]byte, error) {
	if this.Request == nil {
		panic(errors.New("request nil"))
	}
	return GetBody(this.Request)
}

type URLArgs struct {
	xArgs
}

func (this *URLArgs) ReqType() int {
	return AT_URL
}

type FORMArgs struct {
	xArgs
}

func (this *FORMArgs) Validate(m *ValidateModel, c IMVC) {
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
	xArgs
}

func (this *JSONArgs) Validate(m *ValidateModel, c IMVC) {
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
	xArgs
}

func (this *XMLArgs) Validate(m *ValidateModel, c IMVC) {
	c.SetModel(m)
	c.SetRender(XML_RENDER)
}

func (this *XMLArgs) ReqType() int {
	return AT_XML
}

func (this *XMLArgs) Model() IModel {
	return &HTTPModel{}
}
