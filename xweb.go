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
	"net/url"
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

const (
	HTML_RENDER = "HTML"
	JSON_RENDER = "JSON"
	XML_RENDER  = "XML"
	TEXT_RENDER = "TEXT"
	DATA_RENDER = "DATA"
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

//获得可用的render
func getRender(render string) string {
	switch render {
	case HTML_RENDER:
		return render
	case JSON_RENDER:
		return render
	case XML_RENDER:
		return render
	case TEXT_RENDER:
		return render
	case DATA_RENDER:
		return render
	default:
		return HTML_RENDER
	}
}

type IDispatcher interface {
	//url 前缀
	URL() string
	//mvc创建方法
	MVCMake() IMVC
	//mvc渲染方法
	MVCRender(IMVC, render.Render)
}

type HTTPDispatcher struct {
	IDispatcher
}

func (this *HTTPDispatcher) URL() string {
	return ""
}

func (this *HTTPDispatcher) MVCMake() IMVC {
	return &MVC{}
}

//输出html结束
func (this *HTTPDispatcher) MVCRender(mvc IMVC, render render.Render) {
	m := mvc.GetModel()
	s := mvc.GetStatus()
	v := mvc.GetView()
	switch mvc.GetRender() {
	case HTML_RENDER:
		render.HTML(s, v, m)
	case JSON_RENDER:
		render.JSON(s, m)
	case XML_RENDER:
		render.XML(s, m)
	case TEXT_RENDER:
		render.Text(s, m.GetString())
	case DATA_RENDER:
		render.Data(s, m.GetData())
	}
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
	AT_QUERY //body use Query type parse
)

//数据模型
type DataModel struct {
	IModel `bson:"-" json:"-" xml:"-"`
}

func (this *DataModel) GetString() string {
	return reflect.TypeOf(this).Elem().Name()
}

func (this *DataModel) GetData() []byte {
	return []byte(this.GetString())
}

//渲染模型
type HTTPModel struct {
	IModel `bson:"-" json:"-" xml:"-"`
	Code   int    `json:"code" xml:"code"`
	Error  string `json:"error,omitempty" xml:"error,omitempty"`
}

func (this *HTTPModel) GetString() string {
	return fmt.Sprintf("%v", this)
}

func (this *HTTPModel) GetData() []byte {
	return []byte(this.GetString())
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
	Method() string
}

type QUERYArgs struct {
	IArgs
}

func (this QUERYArgs) ValType() int {
	return AT_JSON
}

func (this QUERYArgs) ReqType() int {
	return AT_QUERY
}

func (this QUERYArgs) Method() string {
	return http.MethodPost
}

type FORMArgs struct {
	IArgs
}

func (this FORMArgs) ValType() int {
	return AT_JSON
}

func (this FORMArgs) ReqType() int {
	return AT_FORM
}

func (this FORMArgs) Method() string {
	return http.MethodPost
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

func (this JSONArgs) Method() string {
	return http.MethodPost
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

func (this XMLArgs) Method() string {
	return http.MethodPost
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

func (this *Context) jsonHandler(v interface{}, name string) martini.Handler {
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

func (this *Context) setValue(vk reflect.Kind, val string, sf reflect.Value) {
	switch vk {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val == "" {
			val = "0"
		}
		if intVal, err := strconv.ParseInt(val, 10, 64); err == nil {
			sf.SetInt(intVal)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if val == "" {
			val = "0"
		}
		if uintVal, err := strconv.ParseUint(val, 10, 64); err == nil {
			sf.SetUint(uintVal)
		}
	case reflect.Bool:
		if val == "" {
			val = "false"
		}
		if boolVal, err := strconv.ParseBool(val); err == nil {
			sf.SetBool(boolVal)
		}
	case reflect.Float32:
		if val == "" {
			val = "0.0"
		}
		if floatVal, err := strconv.ParseFloat(val, 32); err == nil {
			sf.SetFloat(floatVal)
		}
	case reflect.Float64:
		if val == "" {
			val = "0.0"
		}
		if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
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
					this.setValue(skind, input[i], slice.Index(i))
				}
				value.Field(i).Set(slice)
			} else {
				this.setValue(tf.Type.Kind(), input[0], sf)
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

func (this *Context) queryHandler(v interface{}, ifv ...interface{}) martini.Handler {
	if reflect.TypeOf(v).Kind() == reflect.Ptr {
		panic("Pointers are not accepted as binding QUERY Args")
	}
	return func(c martini.Context, req *http.Request, log *log.Logger, render render.Render) {
		t := reflect.TypeOf(v)
		v := reflect.New(t)
		args, ok := v.Interface().(IArgs)
		if !ok {
			panic(errors.New(t.Name() + "not imp IArgs"))
		}
		av := url.Values{}
		for ik, iv := range req.URL.Query() {
			for _, vv := range iv {
				av.Add(ik, vv)
			}
		}
		bc, _ := ioutil.ReadAll(req.Body)
		form, _ := url.ParseQuery(string(bc))
		for ik, iv := range form {
			for _, vv := range iv {
				av.Add(ik, vv)
			}
		}
		req.PostForm = av
		this.mapForm(v, av, nil)
		c.Map(v.Elem().Interface())
		this.validateMapData(c, args, render)
	}
}

func (this *Context) formHandler(v interface{}, ifv ...interface{}) martini.Handler {
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
			} else {
				log.Println(err)
			}
		} else {
			if err := req.ParseForm(); err == nil {
				this.mapForm(v, req.Form, nil)
			} else {
				log.Println(err)
			}
		}
		c.Map(v.Elem().Interface())
		this.validateMapData(c, args, render)
	}
}

func (this *Context) xmlHandler(v interface{}, name string) martini.Handler {
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
func (this *Context) QueryFieldName(v interface{}, n string) string {
	t := reflect.TypeOf(v)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		name := f.Tag.Get(n)
		if len(name) > 0 {
			return name
		}
	}
	return ""
}

func (this *Context) IsIArgs(v reflect.Value) (IArgs, bool) {
	if !v.IsValid() {
		return nil, false
	}
	if a, ok := v.Interface().(IArgs); !ok {
		return nil, false
	} else {
		return a, true
	}
}

func (this *Context) IsIDispatcher(v reflect.Value) (IDispatcher, bool) {
	if !v.IsValid() {
		return nil, false
	}
	if !v.CanAddr() {
		return nil, false
	}
	v = v.Addr()
	if !v.IsValid() {
		return nil, false
	}
	if a, ok := v.Interface().(IDispatcher); !ok {
		return nil, false
	} else {
		return a, true
	}
}

func (this *Context) useHandler(r martini.Router, url, method, nv, view, render string, in ...martini.Handler) {
	if len(in) == 0 || url == "" {
		return
	}
	var rv martini.Route = nil
	switch method {
	case http.MethodHead:
		rv = r.Head(url, in...)
	case http.MethodOptions:
		rv = r.Options(url, in...)
	case http.MethodPatch:
		rv = r.Patch(url, in...)
	case http.MethodDelete:
		rv = r.Delete(url, in...)
	case http.MethodPut:
		rv = r.Put(url, in...)
	case http.MethodGet:
		rv = r.Get(url, in...)
	case http.MethodPost:
		rv = r.Post(url, in...)
	default:
		panic(errors.New(method + " do not support"))
	}
	//保存记录打印
	urls := URLS{}
	urls.Method = rv.Method()
	urls.Pattern = rv.Pattern()
	urls.View = view
	urls.Render = render
	urls.Handler = nv
	this.urls = append(this.urls, urls)
}

//mvc模式预处理
func (this *Context) mvcPrepare(view string, render string, d IDispatcher) martini.Handler {
	return func(c martini.Context) {
		mvc := d.MVCMake()
		mvc.SetView(view)
		mvc.SetRender(render)
		mvc.SetStatus(http.StatusOK)
		c.MapTo(mvc, (*IMVC)(nil))
	}
}

func (this *Context) useValue(r martini.Router, c IDispatcher, vv reflect.Value) {
	vt := vv.Type()
	sv := reflect.ValueOf(c)
	for i := 0; i < vt.NumField(); i++ {
		f := vt.Field(i)
		v := vv.Field(i)
		//获取TAG参数
		nv := f.Tag.Get("handler")
		url := f.Tag.Get("url")
		view := f.Tag.Get("view")
		render := strings.ToUpper(f.Tag.Get("render"))
		method := strings.ToUpper(f.Tag.Get("method"))
		in := []martini.Handler{}
		hv := reflect.Value{}
		if nv == "" {
			nv = f.Name + HandlerSuffix
			hv = sv.MethodByName(f.Name + HandlerSuffix)
		} else {
			nv = nv + HandlerSuffix
			hv = sv.MethodByName(nv + HandlerSuffix)
		}
		nv = sv.Type().Elem().PkgPath() + "/" + sv.Type().Elem().Name() + "." + nv
		//启用MVC参数
		if view != "" || render != "" {
			render = getRender(render)
			in = append(in, this.mvcPrepare(view, render, c))
		}
		//自动分析参数
		if iv, b := this.IsIArgs(v); b && hv.IsValid() {
			switch iv.ReqType() {
			case AT_FORM:
				in = append(in, this.formHandler(iv))
			case AT_JSON:
				in = append(in, this.jsonHandler(iv, this.QueryFieldName(iv, "source")))
			case AT_XML:
				in = append(in, this.xmlHandler(iv, this.QueryFieldName(iv, "source")))
			case AT_QUERY:
				in = append(in, this.queryHandler(iv))
			}
			if method == "" {
				method = strings.ToUpper(iv.Method())
			}
		}
		//控制函数
		if hv.IsValid() {
			in = append(in, hv.Interface())
		}
		//启用MVC输出
		if view != "" || render != "" {
			in = append(in, c.MVCRender)
		}
		//设置默认GET
		if method == "" {
			method = http.MethodGet
		}
		//挂接请求类型
		if d, b := this.IsIDispatcher(v); b {
			this.Group(url, func(r martini.Router) {
				this.useRouter(r, d)
			}, in...)
		} else if _, b := this.IsIArgs(v); b {
			this.useHandler(r, url, method, nv, view, render, in...)
		} else if v.Kind() == reflect.Struct {
			this.Group(url, func(r martini.Router) {
				this.useValue(r, c, v)
			}, in...)
		} else {
			this.useHandler(r, url, method, nv, view, render, in...)
		}
	}
}

func (this *Context) useRouter(r martini.Router, c IDispatcher) {
	this.useValue(r, c, reflect.ValueOf(c).Elem())
}

func (this *Context) UseDispatcher(c IDispatcher, in ...martini.Handler) {
	this.Group(c.URL(), func(r martini.Router) {
		this.useRouter(r, c)
	}, in...)
}
