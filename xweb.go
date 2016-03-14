package xweb

/*
Deps:
	go get github.com/sevlyar/go-daemon
	go get github.com/go-martini/martini
	go get github.com/martini-contrib/binding
*/

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	"gopkg.in/validator.v2"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"
)

const (
	ValidateErrorCode = 10000
	ValidateSuffix    = "Validate" //validate组件必须的后缀
	HandlerSuffix     = "Handler"  //处理组件必须的后缀
)

func FormFileBytes(fh *multipart.FileHeader) ([]byte, error) {
	if fh == nil {
		return nil, errors.New("args null")
	}
	f, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ioutil.ReadAll(f)
}

type IDispatcher interface {
	//保存上下文
	SetContext(*Context)
	//获得上下文
	GetContext() *Context
	//获得URL前缀
	GetPrefix() string
}

type ValidateError struct {
	Field string `xml:"field,attr" json:"field"`
	Error string `xml:",chardata" json:"error"`
}

type ValidateModel struct {
	XMLName struct{}        `xml:"xml" json:"-"`
	Code    int             `xml:"code" json:"code"`
	Errors  []ValidateError `xml:"errors>item" json:"errors"`
}

func (this *ValidateModel) Init(e error) {
	this.Errors = []ValidateError{}
	this.Code = ValidateErrorCode
	err, ok := e.(validator.ErrorMap)
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

type HTTPDispatcher struct {
	IDispatcher
	ctx *Context
}

func (this *HTTPDispatcher) GetPrefix() string {
	return ""
}

func (this *HTTPDispatcher) SetContext(ctx *Context) {
	this.ctx = ctx
}

func (this *HTTPDispatcher) GetContext() *Context {
	return this.ctx
}

//校验结果传递到下个组件
func (this *HTTPDispatcher) ToNEXTValidate(c martini.Context, args IArgs) {
	var m *ValidateModel = nil
	if err := this.ctx.Validate(args); err != nil {
		m = NewValidateModel(err)
	}
	c.Map(m)
}

//校验失败输出json
func (this *HTTPDispatcher) ToJSONValidate(args IArgs, render render.Render) {
	if err := this.ctx.Validate(args); err != nil {
		m := NewValidateModel(err)
		render.JSON(http.StatusOK, m)
	}
}

//校验失败输出xml
func (this *HTTPDispatcher) ToXMLValidate(args IArgs, render render.Render) {
	if err := this.ctx.Validate(args); err != nil {
		m := NewValidateModel(err)
		render.XML(http.StatusOK, m)
	}
}

//日志打印调试Handler
func (this *HTTPDispatcher) LoggerHandler(req *http.Request, log *log.Logger) {
	log.Println("----------------------------Logger---------------------------")
	log.Println("Remote:", req.RemoteAddr)
	log.Println("Method:", req.Method)
	log.Println("URL:", req.URL.String())
	for k, v := range req.Header {
		log.Println(k, ":", v)
	}
	log.Println("Query:", req.URL.Query())
	log.Println("--------------------------------------------------------------")
}

//req type
const (
	AT_NONE = iota
	AT_URL
	AT_FORM
	AT_JSON
	AT_XML
)

type HTTPModel struct {
	XMLName struct{} `xml:"xml" json:"-"`
	Code    int      `json:"code" xml:"code"`
	Error   string   `json:"error" xml:"error"`
}

type IArgs interface {
	//request parse data type
	//AT_*
	ReqType() int
}

type URLArgs struct {
	IArgs
}

func (this URLArgs) ReqType() int {
	return AT_URL
}

type FORMArgs struct {
	URLArgs
}

func (this FORMArgs) ReqType() int {
	return AT_FORM
}

type JSONArgs struct {
	URLArgs
}

func (this JSONArgs) ReqType() int {
	return AT_JSON
}

type XMLArgs struct {
	URLArgs
}

func (this XMLArgs) ReqType() int {
	return AT_XML
}

//execute tempate render html
func (this *Context) Execute(render render.Render, view string, m interface{}) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)
	err := render.Template().ExecuteTemplate(buf, view, m)
	return buf, err
}

func (this *Context) QueryHttpRequestData(name string, req *http.Request) ([]byte, error) {
	if req.Method != http.MethodPost {
		return nil, errors.New("http method error")
	}
	if len(name) > 0 {
		contentType := req.Header.Get("Content-Type")
		if strings.Contains(contentType, "multipart/form-data") {
			if err := req.ParseMultipartForm(binding.MaxMemory); err != nil {
				return nil, err
			}
		} else {
			if err := req.ParseForm(); err != nil {
				return nil, err
			}
		}
		data := req.FormValue(name)
		if len(data) > 0 {
			return []byte(data), nil
		}
		_, file, err := req.FormFile(name)
		if err == nil {
			return FormFileBytes(file)
		}
		q := req.URL.Query().Get(name)
		if len(q) > 0 {
			return []byte(q), nil
		}
	} else if req.Body != nil {
		defer req.Body.Close()
		return ioutil.ReadAll(req.Body)
	}
	return nil, errors.New("form data miss")
}

func (this *Context) JsonHandler(v interface{}, name string) martini.Handler {
	if reflect.TypeOf(v).Kind() == reflect.Ptr {
		panic("Pointers are not accepted as binding json models")
	}
	return func(c martini.Context, req *http.Request) {
		errors := binding.Errors{}
		v := reflect.New(reflect.TypeOf(v))
		data, err := this.QueryHttpRequestData(name, req)
		if err != nil {
			errors.Add([]string{}, binding.RequiredError, "request method must POST")
		}
		if err := json.Unmarshal(data, v.Interface()); err != nil {
			errors.Add([]string{}, binding.DeserializationError, err.Error())
		}
		c.Map(errors)
		c.Map(v.Elem().Interface())
	}
}

func (this *Context) FormHandler(obj interface{}, ifv ...interface{}) martini.Handler {
	return binding.Bind(obj, ifv...)
}

func (this *Context) URLHandler(v interface{}) martini.Handler {
	return func(c martini.Context, req *http.Request) {
		t := reflect.TypeOf(v)
		v := reflect.New(t)
		if args, ok := v.Interface().(IArgs); ok {
			c.Map(args)
		}
	}
}

func (this *Context) XmlHandler(v interface{}, name string) martini.Handler {
	if reflect.TypeOf(v).Kind() == reflect.Ptr {
		panic("Pointers are not accepted as binding xml models")
	}
	return func(c martini.Context, req *http.Request) {
		errors := binding.Errors{}
		v := reflect.New(reflect.TypeOf(v))
		data, err := this.QueryHttpRequestData(name, req)
		if err != nil {
			errors.Add([]string{}, binding.RequiredError, "request method must POST")
		}
		if err := xml.Unmarshal(data, v.Interface()); err != nil {
			errors.Add([]string{}, binding.DeserializationError, err.Error())
		}
		c.Map(errors)
		c.Map(v.Elem().Interface())
	}
}

//from name get data source,use AT_JSON AT_XML
func (this *Context) queryFieldName(v interface{}) string {
	t := reflect.TypeOf(v)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		name := f.Tag.Get("form")
		if len(name) > 0 {
			return name
		}
	}
	return ""
}

func (this *Context) doMethod(m string) bool {
	switch m {
	case http.MethodHead:
		return true
	case http.MethodOptions:
		return true
	case http.MethodPatch:
		return true
	case http.MethodDelete:
		return true
	case http.MethodPut:
		return true
	case http.MethodGet:
		return true
	case http.MethodPost:
		return true
	default:
		return false
	}
}

func (this *Context) doFields(tv reflect.Type, nv reflect.Value, pv func(string, *reflect.StructField, *reflect.Value)) {
	if tv.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < tv.NumField(); i++ {
		f := tv.Field(i)
		v := nv.FieldByName(f.Name)
		k := f.Type.Kind()
		if !v.IsValid() {
			continue
		}
		if k != reflect.Struct {
			continue
		}
		method := f.Tag.Get("method")
		if len(method) == 0 {
			method = f.Name
		}
		if len(method) > 0 {
			method = strings.ToUpper(method)
		}
		if this.doMethod(method) {
			pv(method, &f, &v)
		}
		this.doFields(f.Type, v, pv)
	}
}

func (this *Context) UseDispatcher(c IDispatcher) {
	c.SetContext(this)
	log := this.Logger()
	stv := reflect.TypeOf(c).Elem()
	svv := reflect.ValueOf(c)
	snv := svv.Elem()
	this.doFields(stv, snv, func(method string, g *reflect.StructField, nv *reflect.Value) {
		for i := 0; i < g.Type.NumField(); i++ {
			f := g.Type.Field(i)
			url := f.Tag.Get("url")
			if len(url) == 0 {
				log.Println(f.Name, "must set url path")
				continue
			}
			//get field value
			v := nv.FieldByName(f.Name)
			if !v.IsValid() {
				log.Println(f.Name, "value not vaild")
				continue
			}
			iv, ok := v.Interface().(IArgs)
			if !ok {
				log.Println(f.Name, "error,only support IArgs type")
				continue
			}
			//拼接url
			url = c.GetPrefix() + g.Tag.Get("url") + url
			in := []martini.Handler{}
			//group handler
			if mv := svv.MethodByName(g.Tag.Get("handler") + HandlerSuffix); mv.IsValid() {
				in = append(in, mv.Interface())
			}
			//args handler
			switch iv.ReqType() {
			case AT_URL:
				in = append(in, this.URLHandler(iv))
			case AT_FORM:
				in = append(in, this.FormHandler(iv))
			case AT_JSON:
				name := this.queryFieldName(iv)
				in = append(in, this.JsonHandler(iv, name))
			case AT_XML:
				name := this.queryFieldName(iv)
				in = append(in, this.XmlHandler(iv, name))
			default:
				panic(errors.New(url + " field reqType not supprt"))
			}
			//global validate handler
			if mv := svv.MethodByName(g.Tag.Get("validate") + ValidateSuffix); mv.IsValid() {
				in = append(in, mv.Interface())
			}
			//single validate handler
			if mv := svv.MethodByName(f.Tag.Get("validate") + ValidateSuffix); mv.IsValid() {
				in = append(in, mv.Interface())
			}
			//before handler
			if mv := svv.MethodByName(f.Tag.Get("before") + HandlerSuffix); mv.IsValid() {
				in = append(in, mv.Interface())
			}
			//main handler
			mhs := stv.PkgPath() + "/" + stv.Name()
			if mv := svv.MethodByName(f.Tag.Get("handler") + HandlerSuffix); mv.IsValid() {
				mhs += "." + f.Tag.Get("handler") + HandlerSuffix
				in = append(in, mv.Interface())
			} else if mv := svv.MethodByName(f.Name + HandlerSuffix); mv.IsValid() {
				mhs += "." + f.Name + HandlerSuffix
				in = append(in, mv.Interface())
			} else {
				mhs += ".(" + f.Name + "|" + f.Tag.Get("handler") + ")" + HandlerSuffix
				panic(errors.New(mhs + " miss"))
			}
			//after handler
			if mv := svv.MethodByName(f.Tag.Get("after") + HandlerSuffix); mv.IsValid() {
				in = append(in, mv.Interface())
			}
			//set method handler
			switch method {
			case http.MethodHead:
				this.Head(url, in...)
			case http.MethodOptions:
				this.Options(url, in...)
			case http.MethodPatch:
				this.Patch(url, in...)
			case http.MethodDelete:
				this.Delete(url, in...)
			case http.MethodPut:
				this.Put(url, in...)
			case http.MethodGet:
				this.Get(url, in...)
			case http.MethodPost:
				this.Post(url, in...)
			}
			log.Println("+", method, url, mhs)
		}
	})
	this.Map(c)
}
