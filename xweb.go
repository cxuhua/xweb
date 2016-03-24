package xweb

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

var (
	MaxMemory = int64(1024 * 1024 * 10)
)

const (
	ValidateErrorCode = 10000     //数据校验失败返回
	HandlerSuffix     = "Handler" //处理组件必须的后缀
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

type ValidateError struct {
	Field string `xml:"field,attr" json:"field"`
	Error string `xml:",chardata" json:"error"`
}

type ValidateModel struct {
	XMLName struct{}        `xml:"xml" json:"-"`
	Code    int             `xml:"code" json:"code"`
	Errors  []ValidateError `xml:"errors>item,omitempty" json:"errors,omitempty"`
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

type IDispatcher interface {
	SetContext(*Context)
	GetContext() *Context
	GetPrefix() string
	SetPrefix(string)
}

type HTTPDispatcher struct {
	IDispatcher
	prefix string
	ctx    *Context
}

func (this *HTTPDispatcher) SetPrefix(url string) {
	if url == "" {
		return
	}
	this.prefix = url
}

func (this *HTTPDispatcher) GetPrefix() string {
	return this.prefix
}

func (this *HTTPDispatcher) SetContext(ctx *Context) {
	if ctx == nil {
		return
	}
	this.ctx = ctx
}

func (this *HTTPDispatcher) GetContext() *Context {
	return this.ctx
}

//获取远程地址
func (this *HTTPDispatcher) GetRemoteAddr(req *http.Request) string {
	if x1, ok := req.Header["X-Forwarded-For"]; ok && len(x1) > 0 {
		return x1[len(x1)-1]
	}
	if x2, ok := req.Header["X-Real-IP"]; ok && len(x2) > 0 {
		return x2[len(x2)-1]
	}
	if x3 := strings.Split(req.RemoteAddr, ":"); len(x3) > 0 {
		return x3[0]
	}
	return req.RemoteAddr
}

//日志打印调试Handler
func (this *HTTPDispatcher) LoggerHandler(req *http.Request, log *log.Logger) {
	log.Println("----------------------------Logger---------------------------")
	log.Println("Remote:", this.GetRemoteAddr(req))
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
	AT_FORM
	AT_JSON
	AT_XML
)

type HTTPModel struct {
	Code  int    `json:"code" xml:"code"`
	Error string `json:"error,omitempty" xml:"error,omitempty"`
}

func NewHTTPError(code int, err string) *HTTPModel {
	return &HTTPModel{Code: code, Error: err}
}

func NewHTTPSuccess() *HTTPModel {
	return &HTTPModel{Code: 0, Error: ""}
}

type IArgs interface {
	ValType() int //validate failed out
	ReqType() int //request type in
}

type FORMArgs struct {
	IArgs
}

func (this FORMArgs) ValType() int {
	return AT_FORM
}

func (this FORMArgs) ReqType() int {
	return AT_FORM
}

type JSONArgs struct {
	IArgs
}

func (this JSONArgs) ValType() int {
	return AT_JSON
}

func (this JSONArgs) ReqType() int {
	return AT_JSON
}

type XMLArgs struct {
	IArgs
}

func (this XMLArgs) ValType() int {
	return AT_XML
}

func (this XMLArgs) ReqType() int {
	return AT_XML
}

func (this *Context) queryHttpRequestData(name string, req *http.Request) ([]byte, error) {
	if req.Method != http.MethodPost {
		return nil, errors.New("http method error")
	}
	if len(name) > 0 {
		ct := strings.ToLower(req.Header.Get("Content-Type"))
		if strings.Contains(ct, "multipart/form-data") {
			if err := req.ParseMultipartForm(MaxMemory); err != nil {
				return nil, err
			}
			if data, ok := req.MultipartForm.Value[name]; ok && len(data) > 0 {
				return []byte(data[0]), nil
			}
			file, _, err := req.FormFile(name)
			if err == nil {
				defer file.Close()
				return ioutil.ReadAll(file)
			}
		} else {
			if err := req.ParseForm(); err != nil {
				return nil, err
			}
			data := req.FormValue(name)
			if len(data) > 0 {
				return []byte(data), nil
			}
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
		panic("Pointers are not accepted as binding JSON Args")
	}
	return func(c martini.Context, req *http.Request, log *log.Logger, render render.Render) {
		t := reflect.TypeOf(v)
		v := reflect.New(t)
		args, ok := v.Interface().(IArgs)
		if !ok {
			panic(errors.New(t.Name() + "not imp IArgs"))
		}
		data, err := this.queryHttpRequestData(name, req)
		if err != nil {
			log.Println(err)
		}
		if err := json.Unmarshal(data, v.Interface()); err != nil {
			log.Println(err)
		}
		c.Map(v.Elem().Interface())
		this.validateMapData(c, args, render)
	}
}

func (this *Context) setWithProperType(vk reflect.Kind, val string, sf reflect.Value) {
	switch vk {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val == "" {
			val = "0"
		}
		intVal, err := strconv.ParseInt(val, 10, 64)
		if err == nil {
			sf.SetInt(intVal)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if val == "" {
			val = "0"
		}
		uintVal, err := strconv.ParseUint(val, 10, 64)
		if err == nil {
			sf.SetUint(uintVal)
		}
	case reflect.Bool:
		if val == "" {
			val = "false"
		}
		boolVal, err := strconv.ParseBool(val)
		if err == nil {
			sf.SetBool(boolVal)
		}
	case reflect.Float32:
		if val == "" {
			val = "0.0"
		}
		floatVal, err := strconv.ParseFloat(val, 32)
		if err == nil {
			sf.SetFloat(floatVal)
		}
	case reflect.Float64:
		if val == "" {
			val = "0.0"
		}
		floatVal, err := strconv.ParseFloat(val, 64)
		if err == nil {
			sf.SetFloat(floatVal)
		}
	case reflect.String:
		sf.SetString(val)
	}
}

func (this *Context) mapForm(value reflect.Value, form map[string][]string, files map[string][]*multipart.FileHeader) {
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	typ := value.Type()
	for i := 0; i < typ.NumField(); i++ {
		tf := typ.Field(i)
		sf := value.Field(i)
		if tf.Type.Kind() == reflect.Ptr && tf.Anonymous {
			sf.Set(reflect.New(tf.Type.Elem()))
			this.mapForm(sf.Elem(), form, files)
			if reflect.DeepEqual(sf.Elem().Interface(), reflect.Zero(sf.Elem().Type()).Interface()) {
				sf.Set(reflect.Zero(sf.Type()))
			}
		} else if tf.Type.Kind() == reflect.Struct {
			this.mapForm(sf, form, files)
		} else if name := tf.Tag.Get("form"); name == "" || !sf.CanSet() {
			continue
		} else if input, ok := form[name]; ok {
			num := len(input)
			if sf.Kind() == reflect.Slice && num > 0 {
				skind := sf.Type().Elem().Kind()
				slice := reflect.MakeSlice(sf.Type(), num, num)
				for i := 0; i < num; i++ {
					this.setWithProperType(skind, input[i], slice.Index(i))
				}
				value.Field(i).Set(slice)
			} else {
				this.setWithProperType(tf.Type.Kind(), input[0], sf)
			}
		} else if input, ok := files[name]; ok {
			fileType := reflect.TypeOf((*multipart.FileHeader)(nil))
			num := len(input)
			if sf.Kind() == reflect.Slice && num > 0 && sf.Type().Elem() == fileType {
				slice := reflect.MakeSlice(sf.Type(), num, num)
				for i := 0; i < num; i++ {
					slice.Index(i).Set(reflect.ValueOf(input[i]))
				}
				sf.Set(slice)
			} else if sf.Type() == fileType {
				sf.Set(reflect.ValueOf(input[0]))
			}
		}
	}
}

func (this *Context) validateMapData(c martini.Context, v IArgs, render render.Render) bool {
	if v.ValType() == AT_NONE {
		return false
	}
	var m *ValidateModel = nil
	if err := this.Validate(v); err != nil {
		m = NewValidateModel(err)
	}
	if m == nil {
		c.Map(m)
		return false
	}
	switch v.ValType() {
	case AT_JSON:
		render.JSON(http.StatusOK, m)
		return true
	case AT_XML:
		render.XML(http.StatusOK, m)
		return true
	case AT_FORM:
		c.Map(m)
	default:
		panic(errors.New("IArgs ValType error"))
	}
	return false
}

func (this *Context) FormHandler(v interface{}, ifv ...interface{}) martini.Handler {
	if reflect.TypeOf(v).Kind() == reflect.Ptr {
		panic("Pointers are not accepted as binding FORM Args")
	}
	return func(c martini.Context, req *http.Request, log *log.Logger, render render.Render) {
		t := reflect.TypeOf(v)
		v := reflect.New(t)
		args, ok := v.Interface().(IArgs)
		if !ok {
			panic(errors.New(t.Name() + "not imp IArgs"))
		}
		if req.Method != http.MethodPost {
			c.Map(v.Elem().Interface())
			return
		}
		ct := req.Header.Get("Content-Type")
		if strings.Contains(strings.ToLower(ct), "multipart/form-data") {
			if err := req.ParseMultipartForm(MaxMemory); err == nil {
				this.mapForm(v, req.MultipartForm.Value, req.MultipartForm.File)
			}
		} else {
			if err := req.ParseForm(); err == nil {
				this.mapForm(v, req.Form, nil)
			}
		}
		c.Map(v.Elem().Interface())
		this.validateMapData(c, args, render)
	}
}

func (this *Context) XmlHandler(v interface{}, name string) martini.Handler {
	if reflect.TypeOf(v).Kind() == reflect.Ptr {
		panic("Pointers are not accepted as binding XML Args")
	}
	return func(c martini.Context, req *http.Request, log *log.Logger, render render.Render) {
		t := reflect.TypeOf(v)
		v := reflect.New(t)
		args, ok := v.Interface().(IArgs)
		if !ok {
			panic(errors.New(t.Name() + "not imp IArgs"))
		}
		data, err := this.queryHttpRequestData(name, req)
		if err != nil {
			log.Println(err)
		}
		if err := xml.Unmarshal(data, v.Interface()); err != nil {
			log.Println(err)
		}
		c.Map(v.Elem().Interface())
		this.validateMapData(c, args, render)
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

//
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

//
func (this *Context) doSubs(parent IDispatcher, f *reflect.StructField, v *reflect.Value) bool {
	var dv IDispatcher = nil
	if !v.CanAddr() {
		return false
	} else if av := v.Addr(); !av.IsValid() {
		return false
	} else if sv, ok := av.Interface().(IDispatcher); !ok {
		return false
	} else {
		dv = sv
	}
	//prefix url
	if url := f.Tag.Get("url"); url != "" {
		dv.SetPrefix(parent.GetPrefix() + dv.GetPrefix() + url)
	}
	//prefix handler
	in := []martini.Handler{}
	if hv := f.Tag.Get("handler"); hv != "" {
		pv := reflect.ValueOf(parent)
		if mv := pv.MethodByName(hv + HandlerSuffix); mv.IsValid() {
			in = append(in, mv.Interface())
		}
	}
	//use sub dispatcher
	this.UseDispatcher(dv, in...)
	return true
}

//
func (this *Context) doFields(parent IDispatcher, tv reflect.Type, nv reflect.Value, list func(string, *reflect.StructField, *reflect.Value)) {
	for i := 0; i < tv.NumField(); i++ {
		f := tv.Field(i)
		v := nv.Field(i)
		if !v.IsValid() || f.Type.Kind() != reflect.Struct {
			continue
		}
		if this.doSubs(parent, &f, &v) {
			continue
		}
		method := f.Tag.Get("method")
		if len(method) == 0 {
			method = f.Name
		}
		if len(method) > 0 {
			method = strings.ToUpper(method)
		}
		if !this.doMethod(method) {
			continue
		}
		list(method, &f, &v)
	}
}

//
func (this *Context) UseDispatcher(c IDispatcher, hs ...martini.Handler) {
	c.SetContext(this)
	log := this.Logger()
	stv := reflect.TypeOf(c).Elem()
	svv := reflect.ValueOf(c)
	snv := svv.Elem()
	this.doFields(c, stv, snv, func(method string, fv *reflect.StructField, nv *reflect.Value) {
		for i := 0; i < fv.Type.NumField(); i++ {
			f := fv.Type.Field(i)
			v := nv.Field(i)
			url := f.Tag.Get("url")
			if !v.IsValid() || url == "" {
				continue
			}
			//merge url
			url = c.GetPrefix() + fv.Tag.Get("url") + url
			in := []martini.Handler{}
			//prefix handler
			for _, mv := range hs {
				if mv != nil {
					in = append(in, mv)
				}
			}
			//args handler
			if iv, ok := v.Interface().(IArgs); ok {
				switch iv.ReqType() {
				case AT_FORM:
					in = append(in, this.FormHandler(iv))
				case AT_JSON:
					in = append(in, this.JsonHandler(iv, this.queryFieldName(iv)))
				case AT_XML:
					in = append(in, this.XmlHandler(iv, this.queryFieldName(iv)))
				}
			}
			//group handler
			if hv := fv.Tag.Get("handler"); hv != "" {
				if mv := svv.MethodByName(hv + HandlerSuffix); mv.IsValid() {
					in = append(in, mv.Interface())
				}
			}
			//before handler
			if hv := f.Tag.Get("before"); hv != "" {
				if mv := svv.MethodByName(hv + HandlerSuffix); mv.IsValid() {
					in = append(in, mv.Interface())
				}
			}
			//main handler
			mhs := ""
			if hv := f.Tag.Get("handler"); hv != "" {
				if mv := svv.MethodByName(hv + HandlerSuffix); mv.IsValid() {
					mhs = hv + HandlerSuffix
					in = append(in, mv.Interface())
				}
			} else {
				if mv := svv.MethodByName(f.Name + HandlerSuffix); mv.IsValid() {
					mhs = f.Name + HandlerSuffix
					in = append(in, mv.Interface())
				}
			}
			//after handler
			if hv := f.Tag.Get("after"); hv != "" {
				if mv := svv.MethodByName(hv + HandlerSuffix); mv.IsValid() {
					in = append(in, mv.Interface())
				}
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
			mhs = fmt.Sprintf("%v.%s", reflect.TypeOf(c).Elem(), mhs)
			log.Println("+", method, url, "->", mhs)
		}
	})
}
