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
	"fmt"
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
	//IArgs validate error return code
	ValidateErrorCode = 10000

	//model miss return code
	ModelMissErrorCode = 10001
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

	//当数据校验失败时候返回输出Model
	ValidateError(error) IModel
}

type HTTPValidate struct {
	Field string `xml:"field,attr" json:"field"`
	Error string `xml:",chardata" json:"error"`
}

type HTTPValidateModel struct {
	IModel  `json:"-"`
	XMLName struct{}       `xml:"xml" json:"-"`
	Code    int            `xml:"code" json:"code"`
	Errors  []HTTPValidate `xml:"errors>item" json:"errors"`
}

func (this *HTTPValidateModel) Init(err validator.ErrorMap) {
	this.Errors = []HTTPValidate{}
	for k, v := range err {
		e := HTTPValidate{Field: k, Error: v.Error()}
		this.Errors = append(this.Errors, e)
	}
	this.Code = ValidateErrorCode
}

func (this *HTTPValidateModel) ANY(args IArgs, render render.Render) {
	switch args.ErrorType() {
	case OT_JSON:
		render.JSON(http.StatusOK, this)
	case OT_XML:
		render.XML(http.StatusOK, this)
	case OT_TEXT:
		render.Text(http.StatusOK, fmt.Sprintf("%v", this))
	default:
		render.HTML(http.StatusOK, args.ErrorView(), this)
	}
}

const (
	//默认处理组件
	DEFAULT_HANDLER = "HTTPHandler"
)

type HTTPDispatcher struct {
	IDispatcher
	ctx *Context
}

func (this *HTTPDispatcher) SetContext(ctx *Context) {
	this.ctx = ctx
}

func (this *HTTPDispatcher) GetContext() *Context {
	return this.ctx
}

func (this *HTTPDispatcher) ValidateError(err error) IModel {
	v, ok := err.(validator.ErrorMap)
	if !ok {
		return nil
	}
	m := &HTTPValidateModel{}
	m.Init(v)
	return m
}

func (this *HTTPDispatcher) HTTPHandler(c martini.Context, args IArgs, render render.Render, log *log.Logger) {
	var m IModel = nil
	if err := this.ctx.Validate(args); err != nil {
		m = this.ValidateError(err)
	} else {
		m = args.Model()
	}
	if m == nil {
		panic(errors.New(reflect.TypeOf(args).Name() + " Model nil"))
	}
	if reflect.TypeOf(m).Kind() != reflect.Ptr {
		panic(errors.New(reflect.TypeOf(m).Name() + " Model must is Ptr type"))
	}
	v := reflect.ValueOf(m)
	if mf := v.MethodByName("HTML"); mf.IsValid() {
		if _, err := c.Invoke(mf.Interface()); err != nil {
			panic(err)
		}
		view := m.View()
		if len(view) == 0 {
			panic(errors.New(reflect.TypeOf(m).Elem().Name() + " View nil"))
		}
		render.HTML(http.StatusOK, view, m)
		return
	}
	if mf := v.MethodByName("JSON"); mf.IsValid() {
		if _, err := c.Invoke(mf.Interface()); err != nil {
			panic(err)
		}
		render.JSON(http.StatusOK, m)
		return
	}
	if mf := v.MethodByName("XML"); mf.IsValid() {
		if _, err := c.Invoke(mf.Interface()); err != nil {
			panic(err)
		}
		render.XML(http.StatusOK, m)
		return
	}
	if mf := v.MethodByName("ANY"); mf.IsValid() {
		if _, err := c.Invoke(mf.Interface()); err != nil {
			panic(err)
		}
		return
	}
}

func (this *HTTPDispatcher) LogRequest(req *http.Request, log *log.Logger) {
	log.Println("----------------------------LogRequest------------------------")
	log.Println("Method:", req.Method)
	log.Println("URL:", req.URL.String())
	for k, v := range req.Header {
		log.Println(k, ":", v)
	}
	log.Println("Query:", req.URL.Query())
	log.Println("--------------------------------------------------------------")
}

//args type
const (
	AT_QUERY = iota
	AT_FORM
	AT_JSON
	AT_XML
)

//output type
const (
	OT_TEXT = iota
	OT_JSON
	OT_XML
)

type IModel interface {
	View() string
}

type HTTPModel struct {
	IModel  `json:"-"`
	XMLName struct{} `xml:"xml" json:"-"`
	Code    int      `json:"code" xml:"code"`
	Error   string   `json:"error" xml:"error"`
}

func (this *HTTPModel) ANY(args IArgs, render render.Render) {
	if args.ReqType() == AT_JSON {
		render.JSON(http.StatusOK, this)
		return
	}
	if args.ReqType() == AT_XML {
		render.XML(http.StatusOK, this)
		return
	}
}

type IArgs interface {
	//request parse data type
	ReqType() int //AT_*

	//process Model
	Model() IModel //IModel

	//error output type,model not set HTML JSON XML ANY func
	ErrorType() int    //OT_*
	ErrorView() string //error Output html template view
}

type QueryArgs struct {
	IArgs
}

func (this QueryArgs) ErrorView() string {
	return "error"
}

func (this QueryArgs) ErrorType() int {
	return -1
}

func (this QueryArgs) ReqType() int {
	return AT_QUERY
}

func (this QueryArgs) Model() IModel {
	m := new(HTTPModel)
	m.Code = ModelMissErrorCode
	m.Error = reflect.TypeOf(this).Name() + " Model miss"
	return m
}

type FormArgs struct {
	QueryArgs
}

func (this FormArgs) ReqType() int {
	return AT_FORM
}

type JsonArgs struct {
	QueryArgs
}

func (this JsonArgs) ErrorType() int {
	return OT_JSON
}

func (this JsonArgs) ReqType() int {
	return AT_JSON
}

type XmlArgs struct {
	QueryArgs
}

func (this XmlArgs) ErrorType() int {
	return OT_XML
}

func (this XmlArgs) ReqType() int {
	return AT_XML
}

//execute tempate render html
func Execute(render render.Render, m IModel) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)
	v := m.View()
	if len(v) == 0 {
		return nil, errors.New("model view miss")
	}
	err := render.Template().ExecuteTemplate(buf, v, m)
	return buf, err
}

func QueryHttpRequestData(name string, req *http.Request) ([]byte, error) {
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

func JsonHandler(v interface{}, name string) martini.Handler {
	if reflect.TypeOf(v).Kind() == reflect.Ptr {
		panic("Pointers are not accepted as binding json models")
	}
	return func(c martini.Context, req *http.Request) {
		errors := binding.Errors{}
		v := reflect.New(reflect.TypeOf(v))
		data, err := QueryHttpRequestData(name, req)
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

func QueryHandler(v interface{}) martini.Handler {
	return func(c martini.Context, req *http.Request) {
		t := reflect.TypeOf(v)
		if v := reflect.New(t); v.IsValid() {
			c.Map(v.Elem().Interface())
		}
	}
}

func XmlHandler(v interface{}, name string) martini.Handler {
	if reflect.TypeOf(v).Kind() == reflect.Ptr {
		panic("Pointers are not accepted as binding xml models")
	}
	return func(c martini.Context, req *http.Request) {
		errors := binding.Errors{}
		v := reflect.New(reflect.TypeOf(v))
		data, err := QueryHttpRequestData(name, req)
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
func queryFieldName(v interface{}) string {
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

func doMethod(m string) bool {
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
	}
	return false
}

func doFields(tv reflect.Type, nv reflect.Value, pv func(string, *reflect.StructField, *reflect.Value)) {
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
		if doMethod(method) {
			pv(method, &f, &v)
		}
		doFields(f.Type, v, pv)
	}
}

func (this *Context) UseDispatcher(c IDispatcher, v ...interface{}) {
	c.SetContext(this)
	log := this.Logger()
	stv := reflect.TypeOf(c).Elem()
	svv := reflect.ValueOf(c)
	snv := svv.Elem()
	doFields(stv, snv, func(method string, fs *reflect.StructField, nv *reflect.Value) {
		tv := fs.Type
		handler := fs.Tag.Get("handler")
		gurl := fs.Tag.Get("url")
		for i := 0; i < tv.NumField(); i++ {
			f := tv.Field(i)
			//get http url
			url := f.Tag.Get("url")
			if len(url) == 0 {
				log.Println("must set url path")
				continue
			}
			//get field value
			v := nv.FieldByName(f.Name)
			if !v.IsValid() {
				continue
			}
			iv, ok := v.Interface().(IArgs)
			if !ok {
				continue
			}
			//append group url
			url = gurl + url
			in := []martini.Handler{}
			//group handler
			if mv := svv.MethodByName(handler); mv.IsValid() {
				in = append(in, mv.Interface().(martini.Handler))
			}
			//args handler
			switch iv.ReqType() {
			case AT_QUERY:
				in = append(in, QueryHandler(iv))
			case AT_FORM:
				in = append(in, binding.Bind(iv))
			case AT_JSON:
				name := queryFieldName(iv)
				in = append(in, JsonHandler(iv, name))
			case AT_XML:
				name := queryFieldName(iv)
				in = append(in, XmlHandler(iv, name))
			}
			//before handler
			if mv := svv.MethodByName(f.Tag.Get("before")); mv.IsValid() {
				in = append(in, mv.Interface().(martini.Handler))
			}
			//main handler
			if mv := svv.MethodByName(f.Name); mv.IsValid() {
				in = append(in, mv.Interface().(martini.Handler))
			} else if mv := svv.MethodByName(DEFAULT_HANDLER); mv.IsValid() {
				in = append(in, mv.Interface().(martini.Handler))
			}
			//after handler
			if mv := svv.MethodByName(f.Tag.Get("after")); mv.IsValid() {
				in = append(in, mv.Interface().(martini.Handler))
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
		}
	})
	//map dispatcher
	if len(v) == 0 {
		this.Map(c)
	} else {
		this.MapTo(c, v[0])
	}
}
