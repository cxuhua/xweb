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
	Init(*Context)
}

const (
	DEFAULT_HANDLER = "HTTPHandler"
)

type HTTPDispatcher struct {
	IDispatcher
}

func (this *HTTPDispatcher) HTTPHandler(c martini.Context, args IArgs, render render.Render, log *log.Logger) {
	m := args.Model()
	if v := reflect.ValueOf(m); !v.IsValid() {
		log.Println("model", reflect.TypeOf(m), "value not valid")
	} else if mf := v.MethodByName("HTML"); mf.IsValid() {
		if _, err := c.Invoke(mf.Interface()); err != nil {
			panic(err)
		}
		render.HTML(http.StatusOK, m.View(), m)
	} else if mf := v.MethodByName("JSON"); mf.IsValid() {
		if _, err := c.Invoke(mf.Interface()); err != nil {
			panic(err)
		}
		render.JSON(http.StatusOK, m)
	} else if mf := v.MethodByName("XML"); mf.IsValid() {
		if _, err := c.Invoke(mf.Interface()); err != nil {
			panic(err)
		}
		render.XML(http.StatusOK, m)
	} else if mf := v.MethodByName("ANY"); mf.IsValid() {
		if _, err := c.Invoke(mf.Interface()); err != nil {
			panic(err)
		}
	} else {
		log.Println("model", reflect.TypeOf(m), "miss (HTML|JSON|XML|ANY) method")
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

func (this *HTTPDispatcher) Init(m *Context) {

}

const (
	AT_QUERY = iota
	AT_FORM
	AT_JSON
	AT_XML
	AT_BODY
)

type IModel interface {
	View() string
}

type Model struct {
	IModel `json:"-"`
	Code   int   `json:"code"`
	Error  error `json:"error,omitempty"`
}

func (this *Model) View() string {
	return ""
}

type IArgs interface {
	ReqType() int  //AT_*
	Model() IModel //
}

type QueryArgs struct {
	IArgs
}

func (this QueryArgs) ReqType() int {
	return AT_QUERY
}

func (this QueryArgs) Model() IModel {
	return &Model{}
}

type FormArgs struct {
	IArgs
}

func (this FormArgs) ReqType() int {
	return AT_FORM
}

func (this FormArgs) Model() IModel {
	return &Model{}
}

type BodyArgs struct {
	IArgs
	Data []byte
}

func (this BodyArgs) ReqType() int {
	return AT_BODY
}

func (this BodyArgs) Model() IModel {
	return &Model{}
}

type JsonArgs struct {
	IArgs
}

func (this JsonArgs) ReqType() int {
	return AT_JSON
}

func (this JsonArgs) Model() IModel {
	return &Model{}
}

type XmlArgs struct {
	IArgs
}

func (this XmlArgs) ReqType() int {
	return AT_XML
}

func (this XmlArgs) Model() IModel {
	return &Model{}
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

func (this *Context) SetDispatcher(c IDispatcher) {
	c.Init(this)
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
			in := []martini.Handler{}
			//group handler
			if mv := svv.MethodByName(handler); mv.IsValid() {
				in = append(in, mv.Interface().(martini.Handler))
			}
			//args handler
			field := queryFieldName(iv)
			switch iv.ReqType() {
			case AT_QUERY:
				in = append(in, QueryHandler(iv))
			case AT_FORM:
				in = append(in, binding.Bind(iv))
			case AT_JSON:
				in = append(in, JsonHandler(iv, field))
			case AT_BODY:
				in = append(in, BodyHandler(field))
			case AT_XML:
				in = append(in, XmlHandler(iv, field))
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
	this.Map(c)
}
