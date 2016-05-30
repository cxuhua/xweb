package xweb

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"github.com/go-martini/martini"
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
	DefaultHandler    = "Default" + HandlerSuffix
)

const (
	HTML_RENDER     = "HTML"
	JSON_RENDER     = "JSON"
	XML_RENDER      = "XML"
	TEXT_RENDER     = "TEXT"
	SCRIPT_RENDER   = "SCRIPT"
	DATA_RENDER     = "DATA"
	FILE_RENDER     = "FILE"
	TEMP_RENDER     = "TEMP"
	REDIRECT_RENDER = "REDIRECT"
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
	//url 前缀
	URL() string
}

type HTTPDispatcher struct {
	IDispatcher
}

func (this *HTTPDispatcher) Render() string {
	return ""
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

func formSetValue(vk reflect.Kind, val string, sf reflect.Value) {
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

func MapFormValue(value reflect.Value, form url.Values, files map[string][]*multipart.FileHeader, urls url.Values) {
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	typ := value.Type()
	for i := 0; i < typ.NumField(); i++ {
		tf := typ.Field(i)
		sf := value.Field(i)
		if tf.Type.Kind() == reflect.Ptr && tf.Anonymous {
			sf.Set(reflect.New(tf.Type.Elem()))
			MapFormValue(sf.Elem(), form, files, urls)
			if reflect.DeepEqual(sf.Elem().Interface(), reflect.Zero(sf.Elem().Type()).Interface()) {
				sf.Set(reflect.Zero(sf.Type()))
			}
		} else if tf.Type.Kind() == reflect.Struct {
			MapFormValue(sf, form, files, urls)
		} else if name := tf.Tag.Get("form"); name != "-" && name != "" && sf.CanSet() {
			if input, ok := form[name]; ok {
				num := len(input)
				if num == 0 {
					continue
				}
				if sf.Kind() == reflect.Slice {
					skind := sf.Type().Elem().Kind()
					slice := reflect.MakeSlice(sf.Type(), num, num)
					for i := 0; i < num; i++ {
						formSetValue(skind, input[i], slice.Index(i))
					}
					value.Field(i).Set(slice)
				} else {
					formSetValue(tf.Type.Kind(), input[0], sf)
				}
			} else if input, ok := files[name]; ok {
				fileType := reflect.TypeOf((*multipart.FileHeader)(nil))
				num := len(input)
				if num == 0 {
					continue
				}
				if sf.Kind() == reflect.Slice && sf.Type().Elem() == fileType {
					slice := reflect.MakeSlice(sf.Type(), num, num)
					for i := 0; i < num; i++ {
						slice.Index(i).Set(reflect.ValueOf(input[i]))
					}
					sf.Set(slice)
				} else if sf.Type() == fileType {
					sf.Set(reflect.ValueOf(input[0]))
				}
			}
		} else if name := tf.Tag.Get("url"); name != "-" && name != "" && sf.CanSet() {
			input, ok := urls[name]
			if !ok {
				continue
			}
			num := len(input)
			if num == 0 {
				continue
			}
			if sf.Kind() == reflect.Slice {
				skind := sf.Type().Elem().Kind()
				slice := reflect.MakeSlice(sf.Type(), num, num)
				for i := 0; i < num; i++ {
					formSetValue(skind, input[i], slice.Index(i))
				}
				value.Field(i).Set(slice)
			} else {
				formSetValue(tf.Type.Kind(), input[0], sf)
			}
		}
	}
}

//获得http数据
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

func UnmarshalForm(iv IArgs, param martini.Params, req *http.Request) {
	v := reflect.ValueOf(iv)
	ct := req.Header.Get("Content-Type")
	uv := req.URL.Query()
	for k, v := range param {
		uv.Add(k, v)
	}
	if strings.Contains(strings.ToLower(ct), "multipart/form-data") {
		if err := req.ParseMultipartForm(MaxMemory); err == nil {
			MapFormValue(v, req.MultipartForm.Value, req.MultipartForm.File, uv)
		}
	} else {
		if err := req.ParseForm(); err == nil {
			MapFormValue(v, req.Form, nil, uv)
		}
	}
	iv.SetRequest(req)
}

func (this *HttpContext) newFormArgs(iv IArgs, req *http.Request, param martini.Params, log *log.Logger) IArgs {
	t := reflect.TypeOf(iv).Elem()
	v := reflect.New(t)
	args, ok := v.Interface().(IArgs)
	if !ok {
		panic(errors.New(t.Name() + "not imp FORMArgs"))
	}
	UnmarshalForm(args, param, req)
	return args
}

func UnmarshalURL(iv IArgs, param martini.Params, req *http.Request) {
	v := reflect.ValueOf(iv)
	uv := req.URL.Query()
	for k, v := range param {
		uv.Add(k, v)
	}
	MapFormValue(v, nil, nil, uv)
	iv.SetRequest(req)
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
		render = args.Model().Render()
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

//输出html结束
func (this *HttpContext) mvcRender(mvc IMVC, render Render, rw http.ResponseWriter, req *http.Request) {
	m := mvc.GetModel()
	defer m.Finished()
	for ik, iv := range m.GetHeader() {
		for _, vv := range iv {
			rw.Header().Add(ik, vv)
		}
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
		rw.WriteHeader(s)
		rw.Header().Set(ContentType, ContentJSON)
		rw.Write([]byte(v.Script))
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
		http.ServeContent(rw, req, v.Name, v.ModTime, v.File)
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
		http.Redirect(rw, req, v.Url, http.StatusFound)
	default:
		panic(errors.New(mvc.GetRender() + " not process"))
	}
}

func (this *HttpContext) newArgs(iv IArgs, req *http.Request, param martini.Params, log *log.Logger) IArgs {
	switch iv.ReqType() {
	case AT_URL:
		return this.newURLArgs(iv, req, param, log)
	case AT_FORM:
		return this.newFormArgs(iv, req, param, log)
	case AT_JSON:
		return this.newJSONArgs(iv, req, param, log)
	case AT_XML:
		return this.newXMLArgs(iv, req, param, log)
	default:
		panic(errors.New("args reqtype error"))
	}
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
	return func(c martini.Context, rv Render, rw http.ResponseWriter, param martini.Params, req *http.Request, log *log.Logger) {
		mvc := &mvc{}
		mvc.SetStatus(http.StatusOK)
		mvc.SetView(view)
		mvc.SetRender(render)
		c.MapTo(mvc, (*IMVC)(nil))
		args := this.newArgs(iv, req, param, log)
		if args == nil {
			panic(ErrorArgs)
		}
		c.Map(args)
		model := args.Model()
		if model == nil {
			panic(ErrorModel)
		}
		c.Map(model)
		mvc.SetModel(model)
		if err := this.Validate(args); err != nil {
			args.ValidateError(NewValidateModel(err), mvc)
		} else {
			fm := this.GetArgsHandler(args)
			var err error = nil
			var out []reflect.Value = nil
			if fm != nil {
				out, err = c.Invoke(fm)
			} else if hv.IsValid() {
				out, err = c.Invoke(hv.Interface())
			} else {
				out, err = c.Invoke(dv.Interface())
			}
			if err != nil {
				panic(err)
			}
			if len(out) > 0 {
				log.Println(out)
			}
		}
		this.mvcRender(mvc, rv, rw, req)
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
		//使用上父结构方法
		if method == "" {
			method = pmethod
		}
		in := []martini.Handler{}
		//设置前置组件
		hv := sv.MethodByName(handler + HandlerSuffix)
		//默认组建
		dv := sv.MethodByName(DefaultHandler)
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
	this.useValue(http.MethodGet, r, c, reflect.ValueOf(c).Elem())
}

func (this *HttpContext) UseDispatcher(c IDispatcher, in ...martini.Handler) {
	this.Group(c.URL(), func(r martini.Router) {
		this.useRouter(r, c)
	}, in...)
}
