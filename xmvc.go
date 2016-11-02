package xweb

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/cxuhua/xweb/logging"
	"github.com/cxuhua/xweb/martini"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"
)

type IModel interface {
	Finished()   //处理完成
	Render() int //输出模式
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

func (this *xModel) Render() int {
	return HTML_RENDER
}

type RedirectModel struct {
	xModel
	Url string
}

func (this *RedirectModel) Render() int {
	return REDIRECT_RENDER
}

//html model
type HtmlModel struct {
	xModel
}

func (this *HtmlModel) Finished() {

}

func (this *HtmlModel) Render() int {
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

func (this *TempModel) Render() int {
	return TEMP_RENDER
}

//http文件下载
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

func (this *FileModel) Render() int {
	return FILE_RENDER
}

func (this *FileModel) Finished() {
	if this.File != nil {
		this.File.Close()
	}
}

func NewFileModel() *FileModel {
	m := &FileModel{ModTime: time.Now()}
	m.InitHeader()
	return m
}

//脚本输出
type ScriptModel struct {
	xModel
	Script string
}

func (this *ScriptModel) Render() int {
	return SCRIPT_RENDER
}

func (this *ScriptModel) Finished() {

}

func NewScriptModel() *ScriptModel {
	m := &ScriptModel{}
	m.InitHeader()
	return m
}

//用于TEXT输出
type StringModel struct {
	xModel
	Text string
}

func (this *StringModel) Finished() {

}

func (this *StringModel) Render() int {
	return TEXT_RENDER
}

func NewStringModel() *StringModel {
	m := &StringModel{}
	m.InitHeader()
	return m
}

//data render model
type BinaryModel struct {
	xModel
	Data []byte
}

func (this *BinaryModel) Finished() {

}

func (this *BinaryModel) Render() int {
	return DATA_RENDER
}

func NewBinaryModel() *BinaryModel {
	m := &BinaryModel{}
	m.InitHeader()
	return m
}

//json render model
type JSONModel struct {
	xModel
}

func (this *JSONModel) Finished() {

}

func (this *JSONModel) Render() int {
	return JSON_RENDER
}

//xml render model
type XMLModel struct {
	xModel
}

func (this *XMLModel) Finished() {

}

func (this *XMLModel) Render() int {
	return XML_RENDER
}

//渲染模型
type HTTPModel struct {
	JSONModel `bson:"-" json:"-" xml:"-"`
	Code      int    `bson:"code" json:"code" xml:"code"`
	Error     string `bson:"error,omitempty" json:"error,omitempty" xml:"error,omitempty"`
}

func (this *HTTPModel) Finished() {

}

func NewHTTPError(code int, err string) *HTTPModel {
	m := &HTTPModel{Code: code, Error: err}
	m.InitHeader()
	return m
}

func NewHTTPSuccess() *HTTPModel {
	m := &HTTPModel{Code: 0}
	m.InitHeader()
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
	m.InitHeader()
	m.Init(err)
	return m
}

type IMVC interface {
	SetView(string)
	SetTemplate(string)
	SetViewModel(string, IModel)
	SetModel(IModel)
	SetRender(int)
	SetStatus(int)

	Redirect(string)

	SetCookie(cookie *http.Cookie)

	Map(v interface{})
	MapTo(v interface{}, t interface{})
	Next()
	SkipNext()
	SkipRender(bool)

	RunRender()

	Logger() *logging.Logger
	Request() *http.Request
}

type DefaultMVC struct {
	IMVC
	status   int
	view     string
	render   int
	model    IModel
	cookies  []*http.Cookie
	req      *http.Request
	rev      Render
	ctx      martini.Context
	log      *logging.Logger
	isrender bool
}

func (this *DefaultMVC) Render() Render {
	return this.rev
}

func (this *DefaultMVC) Request() *http.Request {
	return this.req
}

func (this *DefaultMVC) Logger() *logging.Logger {
	return this.log
}

func (this *DefaultMVC) SkipNext() {
	this.ctx.SkipNext()
}

func (this *DefaultMVC) autoView() string {
	path := this.req.URL.Path
	if path == "" {
		return "index"
	}
	l := len(path)
	if path[l-1] == '/' {
		return path[1:] + "index"
	}
	return path[1:]
}

func (this *DefaultMVC) SkipRender(v bool) {
	this.isrender = v
}

func (this *DefaultMVC) RunRender() {
	defer this.model.Finished()
	for ik, iv := range this.model.GetHeader() {
		for _, vv := range iv {
			this.rev.Header().Add(ik, vv)
		}
	}
	for _, cv := range this.cookies {
		this.rev.SetCookie(cv)
	}
	if this.render == NONE_RENDER {
		this.render = this.model.Render()
	}
	switch this.render {
	case HTML_RENDER:
		if this.view == "" {
			this.view = this.autoView()
		}
		this.rev.HTML(this.status, this.view, this.model)
	case JSON_RENDER:
		this.rev.JSON(this.status, this.model)
	case XML_RENDER:
		this.rev.XML(this.status, this.model)
	case SCRIPT_RENDER:
		v, b := this.model.(*ScriptModel)
		if !b {
			panic("RENDER Model error:must set ScriptModel")
		}
		this.rev.Header().Set(ContentType, ContentHTML)
		this.rev.Text(this.status, v.Script)
	case TEXT_RENDER:
		v, b := this.model.(*StringModel)
		if !b {
			panic("RENDER Model error:must set StringModel")
		}
		this.rev.Text(this.status, v.Text)
	case DATA_RENDER:
		v, b := this.model.(*BinaryModel)
		if !b {
			panic("RENDER Model error:must set BinaryModel")
		}
		this.rev.Data(this.status, v.Data)
	case FILE_RENDER:
		v, b := this.model.(*FileModel)
		if !b {
			panic("RENDER Model error:must set FileModel")
		}
		this.rev.File(v.Name, v.ModTime, v.File)
	case TEMP_RENDER:
		v, b := this.model.(*TempModel)
		if !b {
			panic("RENDER Model error:must set TempModel")
		}
		this.rev.TEMP(this.status, v.Template, v.Model)
	case REDIRECT_RENDER:
		v, b := this.model.(*RedirectModel)
		if !b {
			panic("RENDER Model error:must set RedirectModel")
		}
		this.rev.Redirect(v.Url)
	default:
		panic(errors.New(RenderToString(this.render) + " not process"))
	}
}

func (this *DefaultMVC) Map(v interface{}) {
	this.ctx.Map(v)
}
func (this *DefaultMVC) MapTo(v interface{}, t interface{}) {
	this.ctx.MapTo(v, t)
}

func (this *DefaultMVC) Next() {
	this.ctx.Next()
}

func (this *DefaultMVC) SetCookie(cookie *http.Cookie) {
	this.cookies = append(this.cookies, cookie)
}

func (this *DefaultMVC) Redirect(url string) {
	m := &RedirectModel{Url: url}
	m.InitHeader()
	this.SetModel(m)
}

func (this *DefaultMVC) String() string {
	return fmt.Sprintf("Status:%d,View:%s,Render:%s,Model:%v", this.status, this.view, RenderToString(this.render), reflect.TypeOf(this.model).Elem())
}

func (this *DefaultMVC) SetView(v string) {
	this.view = v
}

func (this *DefaultMVC) SetModel(v IModel) {
	this.model = v
}

func (this *DefaultMVC) SetTemplate(v string) {
	this.view = v
	this.render = HTML_RENDER
}

func (this *DefaultMVC) SetViewModel(v string, m IModel) {
	this.view = v
	this.model = m
	this.render = HTML_RENDER
}

func (this *DefaultMVC) SetRender(v int) {
	this.render = v
}

func (this *DefaultMVC) SetStatus(v int) {
	this.status = v
}
