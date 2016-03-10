package xweb

/*
Deps:
	go get github.com/sevlyar/go-daemon
	go get github.com/go-martini/martini
	go get github.com/martini-contrib/binding
*/

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"
)

//获得上传文件数据
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
	Init(martini.Router) error
}

type Dispatcher struct {
	IDispatcher
}

//print request info handler
func (this *Dispatcher) LogRequest(req *http.Request, log *log.Logger) {
	log.Println("----------------------------LogRequest------------------------")
	log.Println("Method:", req.Method)
	log.Println("URL:", req.URL.String())
	for k, v := range req.Header {
		log.Println(k, ":", v)
	}
	log.Println("Query:", req.URL.Query())
	log.Println("--------------------------------------------------------------")
}

func (this *Dispatcher) Init(r martini.Router) error {
	return nil
}

//参数类型
const (
	AT_NULL = iota
	AT_FORM
	AT_JSON
	AT_XML
	AT_BODY
)

type IArgs interface {
	ReqType() int //AT_*
}

//null args
type NullArgs struct {
	IArgs
}

func (this NullArgs) ReqType() int {
	return AT_NULL
}

//form表单 用于:POST 需要:enctype=application/x-www-form-urlencoded
type FormArgs struct {
	IArgs
}

func (this FormArgs) ReqType() int {
	return AT_FORM
}

//数据流，将获得 []byte参数类型
type BodyArgs struct {
	IArgs
	Data []byte
}

func (this BodyArgs) ReqType() int {
	return AT_BODY
}

//post json实体数据 用于:POST 需要enctype=application/json
type JsonArgs struct {
	IArgs
}

func (this JsonArgs) ReqType() int {
	return AT_JSON
}

//post xml数据 用于:POST 需要enctype=application/xml
type XmlArgs struct {
	IArgs
}

func (this XmlArgs) ReqType() int {
	return AT_XML
}

//获取指定的form数据
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

//解析json数据
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

//解析xml数据
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

//map []byte
func BodyHandler(name string) martini.Handler {
	return func(c martini.Context, req *http.Request) {
		data := []byte{}
		errors := binding.Errors{}
		data, err := QueryHttpRequestData(name, req)
		if err != nil {
			errors.Add([]string{}, binding.RequiredError, err.Error())
		}
		c.Map(errors)
		c.Map(BodyArgs{Data: data})
	}
}

//查询数据源字段名称
func queryFieldName(v interface{}) string {
	t := reflect.TypeOf(v)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		field := f.Tag.Get("field")
		if len(field) > 0 {
			return field
		}
	}
	return ""
}

//support method
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

//
func doHandlers(f *reflect.StructField, tag string) []string {
	ret := []string{}
	for _, s := range strings.Split(f.Tag.Get(tag), ",") {
		if len(s) == 0 {
			continue
		}
		ret = append(ret, s)
	}
	return ret
}

//搜索可用挂接字段
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

//注册控制器
func Use(r martini.Router, c IDispatcher) error {
	if err := c.Init(r); err != nil {
		return err
	}
	stv := reflect.TypeOf(c).Elem()
	svv := reflect.ValueOf(c)
	snv := svv.Elem()
	doFields(stv, snv, func(method string, fs *reflect.StructField, nv *reflect.Value) {
		tv := fs.Type
		handler := fs.Tag.Get("handler")
		gurl := fs.Tag.Get("url")
		for i := 0; i < tv.NumField(); i++ {
			f := tv.Field(i)
			//get field value
			v := nv.FieldByName(f.Name)
			if !v.IsValid() {
				continue
			}
			iv, ok := v.Interface().(IArgs)
			if !ok {
				continue
			}
			//get http url
			url := f.Tag.Get("url")
			if len(url) == 0 {
				log.Println("must set url path")
				continue
			}
			//append group url
			url = gurl + url
			//query handler
			in := []martini.Handler{}
			ns := []string{}
			//group handler
			for _, htv := range strings.Split(handler, ",") {
				if len(htv) == 0 {
					continue
				}
				if mv := svv.MethodByName(htv); mv.IsValid() {
					in = append(in, mv.Interface().(martini.Handler))
					ns = append(ns, htv)
				} else {
					log.Println("group handler", htv, "miss")
				}
			}
			//bind args handler
			if method == http.MethodPost {
				field := queryFieldName(iv)
				it := reflect.TypeOf(iv)
				switch iv.ReqType() {
				case AT_FORM:
					in = append(in, binding.Bind(iv))
					ns = append(ns, "AT_FORM{"+it.Name()+"}")
				case AT_JSON:
					in = append(in, JsonHandler(iv, field))
					ns = append(ns, "AT_JSON{"+it.Name()+"}")
				case AT_BODY:
					in = append(in, BodyHandler(field))
					ns = append(ns, "AT_BODY{"+it.Name()+"}")
				case AT_XML:
					in = append(in, XmlHandler(iv, field))
					ns = append(ns, "AT_XML{"+it.Name()+"}")
				}
			}
			//before handler
			for _, htv := range doHandlers(&f, "before") {
				if mv := svv.MethodByName(htv); mv.IsValid() {
					in = append(in, mv.Interface().(martini.Handler))
					ns = append(ns, htv)
				} else {
					log.Println("before handler", htv, "miss")
				}
			}
			//main handler
			for _, htv := range strings.Split(f.Name, "_") {
				if len(htv) == 0 {
					continue
				}
				if mv := svv.MethodByName(htv); mv.IsValid() {
					in = append(in, mv.Interface().(martini.Handler))
					ns = append(ns, htv)
				} else {
					log.Println("main handler", htv, "miss")
				}
			}
			//before handler
			for _, htv := range doHandlers(&f, "after") {
				if mv := svv.MethodByName(htv); mv.IsValid() {
					in = append(in, mv.Interface().(martini.Handler))
					ns = append(ns, htv)
				} else {
					log.Println("after handler", htv, "miss")
				}
			}
			//set method handler
			switch method {
			case http.MethodHead:
				r.Head(url, in...)
			case http.MethodOptions:
				r.Options(url, in...)
			case http.MethodPatch:
				r.Patch(url, in...)
			case http.MethodDelete:
				r.Delete(url, in...)
			case http.MethodPut:
				r.Put(url, in...)
			case http.MethodGet:
				r.Get(url, in...)
			case http.MethodPost:
				r.Post(url, in...)
			}
		}
	})
	return nil
}
