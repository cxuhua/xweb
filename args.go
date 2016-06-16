package xweb

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"os"
)

//req type
const (
	AT_NONE = iota
	AT_FORM //表单数据解析
	AT_JSON //json数据解析
	AT_XML  //xml数据解析
	AT_URL  //body use Query type parse
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

type URLArgs struct {
	xArgs
}

func (this *URLArgs) ReqType() int {
	return AT_URL
}

type FORMArgs struct {
	xArgs
}

//写文件返回md5
func (this *FORMArgs) WriteFile(data []byte, pfunc func(string) string) (string, error) {
	md5 := MD5Bytes(data)
	path := pfunc(md5)
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		return md5, nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if n, err := f.Write(data); err != nil || n != len(data) {
		return "", err
	}
	return md5, nil
}

//读取上传多文件
func (this *FORMArgs) ReadFile(file *multipart.FileHeader) ([]byte, error) {
	if file == nil {
		return nil, errors.New("file args null")
	}
	f, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var fb bytes.Buffer
	if _, err := fb.ReadFrom(f); err != nil {
		return nil, err
	}
	return fb.Bytes(), nil
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
