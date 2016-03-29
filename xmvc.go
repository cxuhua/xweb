package xweb

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"strings"
	"time"
)

type IModel interface {
	Render() string //输出模式
}

//html model
type HtmlModel struct {
	IModel
}

func (this *HtmlModel) Render() string {
	return HTML_RENDER
}

//内存模版输出

type TempModel struct {
	IModel
	Template string
	Model    interface{}
}

func (this *TempModel) Render() string {
	return TEMP_RENDER
}

//文件输出
type FileModel struct {
	IModel
	http.Header
	Name    string        //名称
	ModTime time.Time     //修改时间
	Reader  io.ReadSeeker //读取接口
}

func (this *FileModel) Render() string {
	return FILE_RENDER
}

func NewFileModel() *FileModel {
	return &FileModel{Header: http.Header{}, ModTime: time.Now()}
}

//用于TEXT输出
type StringModel struct {
	IModel
	Text string
}

func (this *StringModel) Render() string {
	return TEXT_RENDER
}

//data render model
type BinaryModel struct {
	IModel
	Data []byte
}

func (this *BinaryModel) Render() string {
	return DATA_RENDER
}

//json render model
type JSONModel struct {
	IModel `bson:"-" json:"-" xml:"-"`
}

func (this *JSONModel) Render() string {
	return JSON_RENDER
}

//xml render model
type XMLModel struct {
	IModel `bson:"-" json:"-" xml:"-"`
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

func NewHTTPError(code int, err string) *HTTPModel {
	return &HTTPModel{Code: code, Error: err}
}

func NewHTTPSuccess() *HTTPModel {
	return &HTTPModel{Code: 0, Error: ""}
}

//数据参数校验器是吧输出

type ValidateError struct {
	Field string `xml:"field,attr" json:"field"`
	Error string `xml:",chardata" json:"error"`
}

type ValidateModel struct {
	JSONModel
	XMLName struct{}        `xml:"xml" json:"-"`
	Code    int             `xml:"code" json:"code"`
	Errors  []ValidateError `xml:"errors>item,omitempty" json:"errors,omitempty"`
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
	for _, i := range this.Errors {
		s = append(s, i.Field)
	}
	return strings.Join(s, ",")
}

func (this *ValidateModel) Init(e error) {
	this.Errors = []ValidateError{}
	this.Code = ValidateErrorCode
	err, ok := e.(ErrorMap)
	if !ok {
		return
	}
	for k, v := range err {
		e := ValidateError{Field: k, Error: v.Error()}
		this.Errors = append(this.Errors, e)
	}
}

func NewValidateModel(err error) *ValidateModel {
	m := &ValidateModel{}
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
}

type mvc struct {
	IMVC
	status int
	view   string
	render string
	model  IModel
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
