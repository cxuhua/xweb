package xweb

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/cxuhua/xweb/martini"
	"io/ioutil"
	"mime/multipart"
	"os"
	"reflect"
	"runtime"
)

var (
	FormFileType = reflect.TypeOf(FormFile{})
)

type protoError struct {
	Code    int
	Message string
	File    string
}

func (this protoError) GetCode() string {
	return fmt.Sprintf("%d", this.Code)
}

func (this protoError) Error() string {
	if this.File != "" {
		return this.Message + "\n" + this.File
	}
	return this.Message
}

func ProtoError(code int, format string, args ...interface{}) *protoError {
	if code == 0 {
		panic(errors.New("code != 0,0 is success"))
	}
	// 返回错误信息
	r := &protoError{}
	r.Code = code
	r.Message = fmt.Sprintf(format, args...)
	// 开发模式返回出错位置
	if martini.Env == martini.Dev {
		_, file, line, _ := runtime.Caller(1)
		r.File = fmt.Sprintf("%s:%d", file, line)
	}
	return r
}

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

func (this FormFile) ToJson(v interface{}) error {
	data, err := this.ReadAll()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func (this FormFile) ToXml(v interface{}) error {
	data, err := this.ReadAll()
	if err != nil {
		return err
	}
	return xml.Unmarshal(data, v)
}

func (this FormFile) ToString() (string, error) {
	data, err := this.ReadAll()
	return string(data), err
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
	AT_NONE  = iota
	AT_FORM  //表单数据解析  	use:form tag
	AT_JSON  //json数据解析	use:json tag
	AT_XML   //xml数据解析	use:xml tag
	AT_PROTO // proto协议	user:protobuf
	AT_URL   //url可以和以上结构体混用 use:url tag
)

type IArgs interface {
	//是否校验参数
	IsValidate() bool
	//参数校验失败
	Validate(*ValidateModel, IMVC) error
	//参数解析类型
	ReqType() int
	//返回默认的输出模型
	Model() IModel
}

type xArgs struct {
	IArgs
}

func (this *xArgs) Model() IModel {
	return &HtmlModel{}
}

func (this *xArgs) ReqType() int {
	return AT_NONE
}

func (this *xArgs) IsValidate() bool {
	return true
}

func (this *xArgs) Validate(m *ValidateModel, c IMVC) error {
	v := &StringModel{Text: m.ToTEXT()}
	c.SetModel(v)
	c.SetRender(TEXT_RENDER)
	return nil
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

func (this *FORMArgs) Validate(m *ValidateModel, c IMVC) error {
	c.SetModel(m)
	c.SetRender(JSON_RENDER)
	return nil
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

func (this *JSONArgs) Validate(m *ValidateModel, c IMVC) error {
	c.SetModel(m)
	c.SetRender(JSON_RENDER)
	return nil
}

func (this *JSONArgs) ReqType() int {
	return AT_JSON
}

func (this *JSONArgs) Model() IModel {
	return NewHTTPSuccess()
}

type PROTOArgs struct {
	xArgs
}

func (this *PROTOArgs) Validate(m *ValidateModel, c IMVC) error {
	return nil
}

func (this *PROTOArgs) IsValidate() bool {
	return false
}

func (this *PROTOArgs) ReqType() int {
	return AT_PROTO
}

func (this *PROTOArgs) Model() IModel {
	return nil
}

type XMLArgs struct {
	xArgs
}

func (this *XMLArgs) Validate(m *ValidateModel, c IMVC) error {
	c.SetModel(m)
	c.SetRender(XML_RENDER)
	return nil
}

func (this *XMLArgs) ReqType() int {
	return AT_XML
}

func (this *XMLArgs) Model() IModel {
	return NewHTTPSuccess()
}
