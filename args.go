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
	AT_FORM
	AT_JSON
	AT_XML
	AT_URL //body use Query type parse
)

type IArgs interface {
	//是否校验参数
	IsValidate() bool
	//参数校验失败
	ValidateError(*ValidateModel, IMVC)
	//参数解析类型
	ReqType() int
	//返回默认的输出模型
	Model() IModel
	//设置Request
	SetRequest(*http.Request)
	//获得远程Ip地址
	RemoteAddr() string
}

type XArgs struct {
	IArgs
	*http.Request
}

func (this *XArgs) Model() IModel {
	return &HtmlModel{}
}

func (this *XArgs) ReqType() int {
	return AT_NONE
}

func (this *XArgs) RemoteAddr() string {
	return GetRemoteAddr(this.Request)
}

func (this *XArgs) IsValidate() bool {
	return true
}

func (this *XArgs) SetRequest(req *http.Request) {
	this.Request = req
}

func (this *XArgs) ValidateError(m *ValidateModel, c IMVC) {
	v := &StringModel{Text: m.ToTEXT()}
	c.SetModel(v)
	c.SetRender(TEXT_RENDER)
}

type URLArgs struct {
	XArgs
}

func (this *URLArgs) ReqType() int {
	return AT_URL
}

type FORMArgs struct {
	XArgs
}

//写文件返回md5
func (this *FORMArgs) WriteFile(file *multipart.FileHeader, pfunc func(string) string) (string, error) {
	data, err := this.ReadFile(file)
	if err != nil {
		return "", err
	}
	md5 := MD5Bytes(data)
	path := pfunc(md5)
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		return md5, nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0666)
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

func (this *FORMArgs) ValidateError(m *ValidateModel, c IMVC) {
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
	XArgs
}

func (this *JSONArgs) ValidateError(m *ValidateModel, c IMVC) {
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
	XArgs
}

func (this *XMLArgs) ValidateError(m *ValidateModel, c IMVC) {
	c.SetModel(m)
	c.SetRender(XML_RENDER)
}

func (this *XMLArgs) ReqType() int {
	return AT_XML
}

func (this *XMLArgs) Model() IModel {
	return &HTTPModel{}
}
