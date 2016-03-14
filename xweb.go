package xweb

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"gopkg.in/validator.v2"
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
	AT_FORM
	AT_JSON
	AT_XML
)

type HTTPModel struct {
	Code  int    `json:"code" xml:"code"`
	Error string `json:"error" xml:"error"`
}

func NewHTTPError(code int, err string) *HTTPModel {
	return &HTTPModel{Code: code, Error: err}
}

func NewHTTPSuccess() *HTTPModel {
	return &HTTPModel{Code: 0, Error: ""}
}

type IArgs interface {
	ValType() int //validate failed out
	ReqType() int
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
		if err != nil {
			sf.SetFloat(floatVal)
		}
	case reflect.Float64:
		if val == "" {
			val = "0.0"
		}
		floatVal, err := strconv.ParseFloat(val, 64)
		if err != nil {
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
		} else if inputFieldName := tf.Tag.Get("form"); inputFieldName != "" {
			if !sf.CanSet() {
				continue
			}
			inputValue, exists := form[inputFieldName]
			if exists {
				numElems := len(inputValue)
				if sf.Kind() == reflect.Slice && numElems > 0 {
					sliceOf := sf.Type().Elem().Kind()
					slice := reflect.MakeSlice(sf.Type(), numElems, numElems)
					for i := 0; i < numElems; i++ {
						this.setWithProperType(sliceOf, inputValue[i], slice.Index(i))
					}
					value.Field(i).Set(slice)
				} else {
					this.setWithProperType(tf.Type.Kind(), inputValue[0], sf)
				}
				continue
			}
			inputFile, exists := files[inputFieldName]
			if !exists {
				continue
			}
			fileType := reflect.TypeOf((*multipart.FileHeader)(nil))
			numElems := len(inputFile)
			if sf.Kind() == reflect.Slice && numElems > 0 && sf.Type().Elem() == fileType {
				slice := reflect.MakeSlice(sf.Type(), numElems, numElems)
				for i := 0; i < numElems; i++ {
					slice.Index(i).Set(reflect.ValueOf(inputFile[i]))
				}
				sf.Set(slice)
			} else if sf.Type() == fileType {
				sf.Set(reflect.ValueOf(inputFile[0]))
			}
		}
	}
}

func (this *Context) validateMapData(c martini.Context, v IArgs, render render.Render) {
	if v.ValType() == AT_NONE {
		return
	}
	var m *ValidateModel = nil
	if err := this.Validate(v); err != nil {
		m = NewValidateModel(err)
	}
	if m == nil {
		c.Map(m)
		return
	}
	switch v.ValType() {
	case AT_JSON:
		render.JSON(http.StatusOK, m)
	case AT_XML:
		render.XML(http.StatusOK, m)
	case AT_FORM:
		c.Map(m)
	}
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
		if len(ct) == 0 {
			c.Map(v.Elem().Interface())
			return
		}
		ct = strings.ToLower(ct)
		if strings.Contains(ct, "multipart/form-data") {
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
			//拼接url
			url = c.GetPrefix() + g.Tag.Get("url") + url
			in := []martini.Handler{}
			//args handler
			if iv, ok := v.Interface().(IArgs); ok {
				switch iv.ReqType() {
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
			}
			//group handler
			if mv := svv.MethodByName(g.Tag.Get("handler") + HandlerSuffix); mv.IsValid() {
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
				panic(errors.New(mhs + " MISS"))
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
