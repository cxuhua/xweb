package xweb

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"github.com/cxuhua/xweb/martini"
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
	FormMaxMemory = int64(1024 * 1024 * 10)
)

const (
	ValidateErrorCode = 10000     //数据校验失败返回
	HandlerSuffix     = "Handler" //处理组件必须的后缀
	ModelSuffix       = "Model"   //创建model方法
	DefaultHandler    = "Default" + HandlerSuffix
)

const (
	NONE_RENDER = iota
	HTML_RENDER
	JSON_RENDER
	XML_RENDER
	TEXT_RENDER
	SCRIPT_RENDER
	DATA_RENDER
	FILE_RENDER
	TEMP_RENDER
	REDIRECT_RENDER
)

func StringToRender(r string) int {
	switch r {
	case "HTML":
		return HTML_RENDER
	case "JSON":
		return JSON_RENDER
	case "XML":
		return XML_RENDER
	case "TEXT":
		return TEXT_RENDER
	case "SCRIPT":
		return SCRIPT_RENDER
	case "DATA":
		return DATA_RENDER
	case "FILE":
		return FILE_RENDER
	case "TEMP":
		return TEMP_RENDER
	case "REDIRECT":
		return REDIRECT_RENDER
	default:
		return 0
	}
}

func RenderToString(r int) string {
	switch r {
	case HTML_RENDER:
		return "HTML"
	case JSON_RENDER:
		return "JSON"
	case XML_RENDER:
		return "XML"
	case TEXT_RENDER:
		return "TEXT"
	case SCRIPT_RENDER:
		return "SCRIPT"
	case DATA_RENDER:
		return "DATA"
	case FILE_RENDER:
		return "FILE"
	case TEMP_RENDER:
		return "TEMP"
	case REDIRECT_RENDER:
		return "REDIRECT"
	default:
		return "NONE"
	}
}

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
	URL() string
}

type HTTPDispatcher struct {
	IDispatcher
}

func (this *HTTPDispatcher) URL() string {
	return ""
}

//获取远程地址
func GetRemoteAddr(req *http.Request) string {
	if x := req.Header.Get("X-Real-IP"); x != "" {
		return strings.Split(x, ",")[0]
	}
	if x := req.Header.Get("X-Forwarded-For"); x != "" {
		return strings.Split(x, ",")[0]
	}
	if x := strings.Split(req.RemoteAddr, ":"); len(x) > 0 {
		return x[0]
	}
	return req.RemoteAddr
}

func (this *HTTPDispatcher) DefaultHandler(c IMVC) {

}

//日志打印调试Handler
func (this *HTTPDispatcher) LoggerHandler(req *http.Request, log *log.Logger) {
	log.Println("----------------------------Logger---------------------------")
	log.Println("Remote:", GetRemoteAddr(req))
	log.Println("Method:", req.Method)
	log.Println("URL:", req.URL.String())
	for k, v := range req.Header {
		log.Println(k, ":", v)
	}
	log.Println("Query:", req.URL.Query())
	log.Println("--------------------------------------------------------------")
}

func setKindValue(vk reflect.Kind, val string, sf reflect.Value) {
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
			val = "true"
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

func MapFormType(v interface{}, form url.Values, files map[string][]*multipart.FileHeader, urls url.Values) {
	MapFormValue(reflect.ValueOf(v), form, files, urls)
}

func MapFormValue(value reflect.Value, form url.Values, files map[string][]*multipart.FileHeader, urls url.Values) {
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	vtyp := value.Type()
	for i := 0; i < vtyp.NumField(); i++ {
		tf := vtyp.Field(i)
		sf := value.Field(i)
		if !sf.CanSet() {
			continue
		}
		if tf.Type.Kind() == reflect.Ptr {
			sf.Set(reflect.New(tf.Type.Elem()))
			MapFormValue(sf.Elem(), form, files, urls)
		} else if tf.Type.Kind() == reflect.Struct && tf.Type != FormFileType {
			MapFormValue(sf, form, files, urls)
		} else if name := tf.Tag.Get("form"); name != "-" && name != "" {
			if input, ok := form[name]; ok {
				num := len(input)
				if num == 0 {
					continue
				}
				if sf.Kind() == reflect.Slice {
					skind := sf.Type().Elem().Kind()
					slice := reflect.MakeSlice(sf.Type(), num, num)
					for j := 0; j < num; j++ {
						setKindValue(skind, input[j], slice.Index(j))
					}
					sf.Set(slice)
				} else {
					setKindValue(tf.Type.Kind(), input[0], sf)
				}
			}
			if input, ok := files[name]; ok {
				num := len(input)
				if num == 0 {
					continue
				}
				if sf.Kind() == reflect.Slice && sf.Type().Elem() == FormFileType {
					slice := reflect.MakeSlice(sf.Type(), num, num)
					for j := 0; j < num; j++ {
						item := reflect.ValueOf(FormFile{FileHeader: input[j]})
						slice.Index(j).Set(item)
					}
					sf.Set(slice)
				} else if sf.Type() == FormFileType {
					item := reflect.ValueOf(FormFile{FileHeader: input[0]})
					sf.Set(item)
				}
			}
		} else if name := tf.Tag.Get("url"); name != "-" && name != "" {
			if input, ok := urls[name]; ok {
				num := len(input)
				if num == 0 {
					continue
				}
				if sf.Kind() == reflect.Slice {
					skind := sf.Type().Elem().Kind()
					slice := reflect.MakeSlice(sf.Type(), num, num)
					for j := 0; j < num; j++ {
						setKindValue(skind, input[j], slice.Index(j))
					}
					sf.Set(slice)
				} else {
					setKindValue(tf.Type.Kind(), input[0], sf)
				}
			}
		}
	}
}

//获得http post数据
func (this *HttpContext) GetBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, errors.New("body data miss")
	}
	return ioutil.ReadAll(req.Body)
}

func (this *HttpContext) newURLArgs(iv IArgs, req *http.Request, param martini.Params, log *log.Logger) IArgs {
	t := reflect.TypeOf(iv).Elem()
	v := reflect.New(t)
	args, ok := v.Interface().(IArgs)
	if !ok {
		panic(errors.New(t.Name() + "not imp URLArgs"))
	}
	UnmarshalURL(args, param, req)
	return args
}

func UnmarshalForm(iv IArgs, param martini.Params, req *http.Request, log *log.Logger) {
	v := reflect.ValueOf(iv)
	ct := strings.ToLower(req.Header.Get(ContentType))
	uv := req.URL.Query()
	for k, v := range param {
		uv.Add(k, v)
	}
	if strings.Contains(ct, MultipartFormData) {
		if err := req.ParseMultipartForm(FormMaxMemory); err == nil {
			MapFormValue(v, req.MultipartForm.Value, req.MultipartForm.File, uv)
		} else {
			log.Println("parse multipart form error", err)
		}
	} else {
		if err := req.ParseForm(); err == nil {
			MapFormValue(v, req.Form, nil, uv)
		} else {
			log.Println("parse form error", err)
		}
	}
}

func (this *HttpContext) newFormArgs(iv IArgs, req *http.Request, param martini.Params, log *log.Logger) IArgs {
	t := reflect.TypeOf(iv).Elem()
	v := reflect.New(t)
	args, ok := v.Interface().(IArgs)
	if !ok {
		panic(errors.New(t.Name() + "not imp FORMArgs"))
	}
	UnmarshalForm(args, param, req, log)
	return args
}

func UnmarshalURL(iv IArgs, param martini.Params, req *http.Request) {
	v := reflect.ValueOf(iv)
	uv := req.URL.Query()
	for k, v := range param {
		uv.Add(k, v)
	}
	MapFormValue(v, nil, nil, uv)
}

func (this *HttpContext) newJSONArgs(iv IArgs, req *http.Request, param martini.Params, log *log.Logger) IArgs {
	t := reflect.TypeOf(iv).Elem()
	v := reflect.New(t)
	args, ok := v.Interface().(IArgs)
	if !ok {
		panic(errors.New(t.Name() + "not imp JSONArgs"))
	}
	data, err := this.GetBody(req)
	if err != nil {
		log.Println(err)
	}
	if err := json.Unmarshal(data, args); err != nil {
		log.Println(err)
	}
	UnmarshalURL(args, param, req)
	return args
}

func (this *HttpContext) newXMLArgs(iv IArgs, req *http.Request, param martini.Params, log *log.Logger) IArgs {
	t := reflect.TypeOf(iv).Elem()
	v := reflect.New(t)
	args, ok := v.Interface().(IArgs)
	if !ok {
		panic(errors.New(t.Name() + "not imp XMLArgs"))
	}
	data, err := this.GetBody(req)
	if err != nil {
		log.Println(err)
	}
	if err := xml.Unmarshal(data, args); err != nil {
		log.Println(err)
	}
	UnmarshalURL(args, param, req)
	return args
}

func (this *HttpContext) IsIArgs(v reflect.Value) (IArgs, bool) {
	if !v.IsValid() {
		return nil, false
	}
	if !v.CanAddr() {
		return nil, false
	}
	if a, ok := v.Addr().Interface().(IArgs); !ok {
		return nil, false
	} else {
		return a, true
	}
}

func (this *HttpContext) IsIDispatcher(v reflect.Value) (IDispatcher, bool) {
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

func (this *HttpContext) useHandler(method string, r martini.Router, url, view, render string, args IArgs, in ...martini.Handler) {
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
	if render == "" {
		render = RenderToString(args.Model().Render())
	}
	//保存记录打印
	urls := URLS{}
	urls.Method = rv.Method()
	urls.Pattern = rv.Pattern()
	urls.View = view
	urls.Render = render
	urls.Args = args
	this.URLS = append(this.URLS, urls)
}

func (this *HttpContext) GetArgsHandler(args IArgs) interface{} {
	v := reflect.ValueOf(args)
	if hv := v.MethodByName(HandlerSuffix); hv.IsValid() {
		return hv.Interface()
	} else {
		return nil
	}
}

func (this *HttpContext) GetArgsModel(args IArgs) interface{} {
	v := reflect.ValueOf(args)
	if hv := v.MethodByName(ModelSuffix); hv.IsValid() {
		return hv.Interface()
	} else {
		return nil
	}
}

//输出html结束
func (this *HttpContext) mvcRender(mvc IMVC, render Render) {
	m := mvc.GetModel()
	defer m.Finished()
	for ik, iv := range m.GetHeader() {
		for _, vv := range iv {
			render.Header().Add(ik, vv)
		}
	}
	for _, cv := range mvc.GetCookie() {
		render.SetCookie(cv)
	}
	s := mvc.GetStatus()
	v := mvc.GetView()
	switch mvc.GetRender() {
	case HTML_RENDER:
		if v == "" {
			panic("RENDER Model error:Template miss")
		}
		render.HTML(s, v, m)
	case JSON_RENDER:
		render.JSON(s, m)
	case XML_RENDER:
		render.XML(s, m)
	case SCRIPT_RENDER:
		v, b := m.(*ScriptModel)
		if !b {
			panic("RENDER Model error:must set ScriptModel")
		}
		render.Header().Set(ContentType, ContentHTML)
		render.Text(s, v.Script)
	case TEXT_RENDER:
		v, b := m.(*StringModel)
		if !b {
			panic("RENDER Model error:must set StringModel")
		}
		render.Text(s, v.Text)
	case DATA_RENDER:
		v, b := m.(*BinaryModel)
		if !b {
			panic("RENDER Model error:must set BinaryModel")
		}
		render.Data(s, v.Data)
	case FILE_RENDER:
		v, b := m.(*FileModel)
		if !b {
			panic("RENDER Model error:must set FileModel")
		}
		render.File(v.Name, v.ModTime, v.File)
	case TEMP_RENDER:
		v, b := m.(*TempModel)
		if !b {
			panic("RENDER Model error:must set TempModel")
		}
		render.TEMP(s, v.Template, v.Model)
	case REDIRECT_RENDER:
		v, b := m.(*RedirectModel)
		if !b {
			panic("RENDER Model error:must set RedirectModel")
		}
		render.Redirect(v.Url)
	default:
		panic(errors.New(RenderToString(mvc.GetRender()) + " not process"))
	}
}

func (this *HttpContext) newArgs(iv IArgs, req *http.Request, param martini.Params, log *log.Logger) IArgs {
	var args IArgs = nil
	switch iv.ReqType() {
	case AT_URL:
		args = this.newURLArgs(iv, req, param, log)
	case AT_FORM:
		args = this.newFormArgs(iv, req, param, log)
	case AT_JSON:
		args = this.newJSONArgs(iv, req, param, log)
	case AT_XML:
		args = this.newXMLArgs(iv, req, param, log)
	default:
		panic(errors.New("args reqtype error"))
	}
	args.Init(req)
	return args
}

var (
	ErrorArgs  = errors.New("args nil")
	ErrorModel = errors.New("model nil")
)

//mvc模式预处理
func (this *HttpContext) mvcHandler(iv IArgs, hv reflect.Value, dv reflect.Value, view string, render string) martini.Handler {
	if !dv.IsValid() {
		panic(errors.New("DefaultHandler miss"))
	}
	return func(c martini.Context, rv Render, param martini.Params, req *http.Request, log *log.Logger) {
		mvc := &mvc{}
		mvc.SetStatus(http.StatusOK)
		mvc.SetView(view)
		mvc.SetRender(StringToRender(render))
		//map mvc
		c.MapTo(mvc, (*IMVC)(nil))
		args := this.newArgs(iv, req, param, log)
		if args == nil {
			panic(ErrorArgs)
		}
		//map args
		c.Map(args)
		model := args.Model()
		if model == nil {
			panic(ErrorModel)
		}
		//map model
		c.Map(model)
		mvc.SetModel(model)
		if err := this.Validate(args); err != nil {
			args.Validate(NewValidateModel(err), mvc)
		} else if fm := this.GetArgsHandler(args); fm != nil {
			//args Handler
			c.Invoke(fm)
		} else if hv.IsValid() {
			//dispatch Handler
			c.Invoke(hv.Interface())
		} else {
			//default Handler
			c.Invoke(dv.Interface())
		}
		this.mvcRender(mvc, rv)
	}
}

func (this *HttpContext) useValue(pmethod string, r martini.Router, c IDispatcher, vv reflect.Value) {
	vt := vv.Type()
	sv := reflect.ValueOf(c)
	for i := 0; i < vt.NumField(); i++ {
		f := vt.Field(i)
		v := vv.Field(i)
		url := f.Tag.Get("url")
		view := f.Tag.Get("view")
		render := f.Tag.Get("render")
		handler := f.Tag.Get("handler")
		if handler == "" {
			handler = f.Name
		}
		method := f.Tag.Get("method")
		//使用父结构方法
		if method == "" {
			method = pmethod
		}
		method = strings.ToUpper(method)
		in := []martini.Handler{}
		//设置前置组件
		hv := sv.MethodByName(handler + HandlerSuffix)
		//默认组建
		dv := sv.MethodByName(DefaultHandler)
		//检测参数
		iv, ab := this.IsIArgs(v)
		if ab && url != "" {
			render = strings.ToUpper(render)
			in = append(in, this.mvcHandler(iv, hv, dv, view, render))
		}
		if d, b := this.IsIDispatcher(v); b {
			if hv.IsValid() {
				in = append(in, hv.Interface())
			}
			this.Group(d.URL()+url, func(r martini.Router) {
				this.useRouter(r, d)
			}, in...)
		} else if ab {
			this.useHandler(method, r, url, view, render, iv, in...)
		} else if v.Kind() == reflect.Struct {
			if hv.IsValid() {
				in = append(in, hv.Interface())
			}
			this.Group(url, func(r martini.Router) {
				this.useValue(method, r, c, v)
			}, in...)
		} else {
			this.useHandler(method, r, url, view, render, iv, in...)
		}
	}
}

func (this *HttpContext) useRouter(r martini.Router, c IDispatcher) {
	v := reflect.ValueOf(c)
	this.useValue(http.MethodGet, r, c, v.Elem())
}

func (this *HttpContext) UseDispatcher(c IDispatcher, in ...martini.Handler) {
	this.Group(c.URL(), func(r martini.Router) {
		this.useRouter(r, c)
	}, in...)
}
