package xweb

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"
)

type IModel interface {
	Finished()      //处理完成
	Render() string //输出模式
	GetHeader() http.Header
}

type xModel struct {
	IModel
	Header http.Header
}

func (this *xModel) InitHeader() {
	this.Header = http.Header{}
}

func (this *xModel) GetHeader() http.Header {
	return this.Header
}

func (this *xModel) Finished() {

}

func (this *xModel) Render() string {
	return HTML_RENDER
}

type RedirectModel struct {
	xModel
	Url string
}

func (this *RedirectModel) Render() string {
	return REDIRECT_RENDER
}

//html model
type HtmlModel struct {
	xModel
}

func (this *HtmlModel) Finished() {

}

func (this *HtmlModel) Render() string {
	return HTML_RENDER
}

//内存模版输出

type TempModel struct {
	xModel
	Template string
	Model    interface{}
}

func (this *TempModel) Finished() {

}

func (this *TempModel) Render() string {
	return TEMP_RENDER
}

type IHttpFile interface {
	io.Reader
	io.Seeker
	io.Closer
}

//文件输出
type FileModel struct {
	xModel
	Name    string    //名称
	ModTime time.Time //修改时间
	File    IHttpFile //读取接口
}

func (this *FileModel) Render() string {
	return FILE_RENDER
}

func (this *FileModel) Finished() {
	if this.File != nil {
		this.File.Close()
	}
}

func NewFileModel() *FileModel {
	m := &FileModel{ModTime: time.Now()}
	m.Header = http.Header{}
	return m
}

//脚本输出
type ScriptModel struct {
	xModel
	Script string
}

func (this *ScriptModel) Render() string {
	return SCRIPT_RENDER
}

func (this *ScriptModel) Finished() {

}

func NewScriptModel() *ScriptModel {
	m := &ScriptModel{}
	m.Header = http.Header{}
	return m
}

//用于TEXT输出
type StringModel struct {
	xModel
	Text string
}

func (this *StringModel) Finished() {

}

func (this *StringModel) Render() string {
	return TEXT_RENDER
}

func NewStringModel() *StringModel {
	m := &StringModel{}
	m.Header = http.Header{}
	return m
}

//data render model
type BinaryModel struct {
	xModel
	Data []byte
}

func (this *BinaryModel) Finished() {

}

func (this *BinaryModel) Render() string {
	return DATA_RENDER
}

func NewBinaryModel() *BinaryModel {
	m := &BinaryModel{}
	m.Header = http.Header{}
	return m
}

//json render model
type JSONModel struct {
	xModel
}

func (this *JSONModel) Finished() {

}

func (this *JSONModel) Render() string {
	return JSON_RENDER
}

//xml render model
type XMLModel struct {
	xModel
}

func (this *XMLModel) Finished() {

}

func (this *XMLModel) Render() string {
	return XML_RENDER
}

//渲染模型
type HTTPModel struct {
	JSONModel `bson:"-" json:"-" xml:"-"`
	Code      int    `json:"code" xml:"code"`
	Error     string `json:"error,omitempty" xml:"error,omitempty"`
}

func (this *HTTPModel) Finished() {

}

func NewHTTPError(code int, err string) *HTTPModel {
	m := &HTTPModel{Code: code, Error: err}
	m.Header = http.Header{}
	return m
}

func NewHTTPSuccess() *HTTPModel {
	m := &HTTPModel{Code: 0}
	m.Header = http.Header{}
	return m
}

//数据参数校验器是吧输出

type ValidateError struct {
	Field string `xml:"field,attr" json:"field"`
	Error string `xml:",chardata" json:"error"`
}

type ValidateModel struct {
	JSONModel `json:"-" xml:"-" form:"-" url:"-"`
	XMLName   struct{}        `xml:"xml" json:"-" form:"-" url:"-"`
	Code      int             `xml:"code" json:"code"`
	Fileds    []ValidateError `xml:"fileds>item" json:"fileds"`
	Error     string          `xml:"error" json:"error"`
}

func (this *ValidateModel) ToJSON() string {
	d, err := json.Marshal(this)
	if err != nil {
		return ""
	}
	return string(d)
}

func (this *ValidateModel) ToXML() string {
	d, err := xml.Marshal(this)
	if err != nil {
		return ""
	}
	return string(d)
}

func (this *ValidateModel) ToTEXT() string {
	s := []string{}
	for _, i := range this.Fileds {
		s = append(s, i.Field)
	}
	return strings.Join(s, ",")
}

func (this *ValidateModel) Init(e error) {
	this.Error = "args error,look fileds"
	this.Fileds = []ValidateError{}
	this.Code = ValidateErrorCode
	err, ok := e.(ErrorMap)
	if !ok {
		return
	}
	for k, v := range err {
		e := ValidateError{Field: k, Error: v.Error()}
		this.Fileds = append(this.Fileds, e)
	}
}

func NewValidateModel(err error) *ValidateModel {
	m := &ValidateModel{}
	m.Header = http.Header{}
	m.Init(err)
	return m
}

type IMVC interface {
	GetView() string
	SetView(string)

	GetModel() IModel
	SetModel(IModel)

	GetRender() string
	SetRender(string)

	GetStatus() int
	SetStatus(int)

	Redirect(string)
}

type mvc struct {
	IMVC
	status int
	view   string
	render string
	model  IModel
}

func (this *mvc) Redirect(url string) {
	m := &RedirectModel{}
	m.Url = url
	this.SetModel(m)
}

func (this *mvc) String() string {
	return fmt.Sprintf("Status:%d,View:%s,Render:%s,Model:%v", this.status, this.view, this.render, reflect.TypeOf(this.model).Elem())
}

func (this *mvc) GetView() string {
	return this.view
}

func (this *mvc) SetView(v string) {
	this.view = v
}

func (this *mvc) GetModel() IModel {
	return this.model
}

func (this *mvc) SetModel(v IModel) {
	this.model = v
}

func (this *mvc) GetRender() string {
	if this.render == "" {
		this.render = this.model.Render()
	}
	return this.render
}

func (this *mvc) SetRender(v string) {
	this.render = v
}

func (this *mvc) GetStatus() int {
	return this.status
}

func (this *mvc) SetStatus(v int) {
	this.status = v
}
