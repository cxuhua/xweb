package xweb

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"github.com/cxuhua/xweb/logging"
	"github.com/cxuhua/xweb/martini"
	"io/ioutil"
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
	ValidateErrorCode = 10000     //数据校验失败返回code
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

var (
	smap = map[string]int{
		"HTML":     HTML_RENDER,
		"JSON":     JSON_RENDER,
		"XML":      XML_RENDER,
		"TEXT":     TEXT_RENDER,
		"SCRIPT":   SCRIPT_RENDER,
		"DATA":     DATA_RENDER,
		"FILE":     FILE_RENDER,
		"TEMP":     TEMP_RENDER,
		"REDIRECT": REDIRECT_RENDER,
	}
	rmap = map[int]string{
		HTML_RENDER:     "HTML",
		JSON_RENDER:     "JSON",
		XML_RENDER:      "XML",
		TEXT_RENDER:     "TEXT",
		SCRIPT_RENDER:   "SCRIPT",
		DATA_RENDER:     "DATA",
		FILE_RENDER:     "FILE",
		TEMP_RENDER:     "TEMP",
		REDIRECT_RENDER: "REDIRECT",
	}
)

func StringToRender(r string) int {
	if v, ok := smap[r]; ok {
		return v
	} else {
		return 0
	}
}

func RenderToString(r int) string {
	if v, ok := rmap[r]; ok {
		return v
	} else {
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

//默认处理方法
func (this *HTTPDispatcher) DefaultHandler(log *logging.Logger, c IMVC) {
	log.Info("invoke default handler")
}

//日志打印调试Handler
func (this *HTTPDispatcher) LoggerHandler(req *http.Request, log *logging.Logger, c IMVC) {
	log.Info("----------------------------Logger---------------------------")
	log.Info("Remote:", GetRemoteAddr(req))
	log.Info("Method:", req.Method)
	log.Info("URL:", req.URL.String())
	for k, v := range req.Header {
		log.Info(k, ":", v)
	}
	log.Info("Query:", req.URL.Query())
	log.Info("--------------------------------------------------------------")
}

func setKindValue(vk reflect.Kind, val string, sf reflect.Value) error {
	switch vk {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val == "" {
			val = "0"
		}
		intVal, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return err
		}
		sf.SetInt(intVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if val == "" {
			val = "0"
		}
		uintVal, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return err
		}
		sf.SetUint(uintVal)
	case reflect.Bool:
		if val == "" {
			val = "false"
		}
		boolVal, err := strconv.ParseBool(val)
		if err != nil {
			return err
		}
		sf.SetBool(boolVal)
	case reflect.Float32:
		if val == "" {
			val = "0.0"
		}
		floatVal, err := strconv.ParseFloat(val, 32)
		if err != nil {
			return err
		}
		sf.SetFloat(floatVal)
	case reflect.Float64:
		if val == "" {
			val = "0.0"
		}
		floatVal, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		}
		sf.SetFloat(floatVal)
	case reflect.String:
		sf.SetString(val)
	}
	return nil
}

func MapFormBindType(v interface{}, form url.Values, files map[string][]*multipart.FileHeader, urls url.Values, cookies url.Values) {
	MapFormBindValue(reflect.ValueOf(v), form, files, urls, cookies)
}

func MapFormBindValue(value reflect.Value, form url.Values, files map[string][]*multipart.FileHeader, urls url.Values, cookies url.Values) {
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
			MapFormBindValue(sf.Elem(), form, files, urls, cookies)
		} else if tf.Type.Kind() == reflect.Struct && tf.Type != FormFileType {
			MapFormBindValue(sf, form, files, urls, cookies)
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
		} else if name := tf.Tag.Get("cookie"); name != "-" && name != "" {
			if input, ok := cookies[name]; ok {
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

func (this *HttpContext) newURLArgs(iv IArgs, req *http.Request, param martini.Params, log *logging.Logger) IArgs {
	t := reflect.TypeOf(iv).Elem()
	v := reflect.New(t)
	args, ok := v.Interface().(IArgs)
	if !ok {
		panic(errors.New(t.Name() + "not imp URLArgs"))
	}
	UnmarshalURLCookie(args, param, req)
	return args
}

func UnmarshalForm(iv IArgs, param martini.Params, req *http.Request, log *logging.Logger) {
	v := reflect.ValueOf(iv)
	ct := strings.ToLower(req.Header.Get(ContentType))
	//
	uv := req.URL.Query()
	for k, v := range param {
		uv.Add(k, v)
	}
	//
	cv := url.Values{}
	for _, v := range req.Cookies() {
		cv.Add(v.Name, v.Value)
	}
	//
	if strings.Contains(ct, MultipartFormData) {
		if err := req.ParseMultipartForm(FormMaxMemory); err == nil {
			MapFormBindValue(v, req.MultipartForm.Value, req.MultipartForm.File, uv, cv)
		} else {
			log.Error("parse multipart form error", err)
		}
		return
	}
	if err := req.ParseForm(); err == nil {
		MapFormBindValue(v, req.Form, nil, uv, cv)
	} else {
		log.Error("parse form error", err)
	}
}

func (this *HttpContext) newFormArgs(iv IArgs, req *http.Request, param martini.Params, log *logging.Logger) IArgs {
	t := reflect.TypeOf(iv).Elem()
	v := reflect.New(t)
	args, ok := v.Interface().(IArgs)
	if !ok {
		panic(errors.New(t.Name() + "not imp FORMArgs"))
	}
	UnmarshalForm(args, param, req, log)
	return args
}

func UnmarshalURLCookie(iv IArgs, param martini.Params, req *http.Request) {
	v := reflect.ValueOf(iv)
	uv := req.URL.Query()
	for k, v := range param {
		uv.Add(k, v)
	}
	cv := url.Values{}
	for _, v := range req.Cookies() {
		cv.Add(v.Name, v.Value)
	}
	MapFormBindValue(v, nil, nil, uv, cv)
}

func (this *HttpContext) newJSONArgs(iv IArgs, req *http.Request, param martini.Params, log *logging.Logger) IArgs {
	t := reflect.TypeOf(iv).Elem()
	v := reflect.New(t)
	args, ok := v.Interface().(IArgs)
	if !ok {
		panic(errors.New(t.Name() + "not imp JSONArgs"))
	}
	data, err := this.GetBody(req)
	if err != nil {
		log.Error(err)
	}
	if err := json.Unmarshal(data, args); err != nil {
		log.Error(err)
	}
	UnmarshalURLCookie(args, param, req)
	return args
}

func (this *HttpContext) newXMLArgs(iv IArgs, req *http.Request, param martini.Params, log *logging.Logger) IArgs {
	t := reflect.TypeOf(iv).Elem()
	v := reflect.New(t)
	args, ok := v.Interface().(IArgs)
	if !ok {
		panic(errors.New(t.Name() + "not imp XMLArgs"))
	}
	data, err := this.GetBody(req)
	if err != nil {
		log.Error(err)
	}
	if err := xml.Unmarshal(data, args); err != nil {
		log.Error(err)
	}
	UnmarshalURLCookie(args, param, req)
	return args
}

func (this *HttpContext) IsIArgs(v reflect.Value) (a IArgs, ok bool) {
	if !v.IsValid() {
		return nil, false
	}
	if !v.CanAddr() {
		return nil, false
	}
	addr := v.Addr()
	if !addr.IsValid() || !addr.CanInterface() {
		return nil, false
	}
	a, ok = addr.Interface().(IArgs)
	return
}

func (this *HttpContext) IsIDispatcher(v reflect.Value) (av IDispatcher, ok bool) {
	if !v.IsValid() {
		return
	}
	if !v.CanAddr() {
		return
	}
	v = v.Addr()
	if !v.IsValid() || !v.CanInterface() {
		return
	}
	av, ok = v.Interface().(IDispatcher)
	return
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

func (this *HttpContext) newArgs(iv IArgs, req *http.Request, param martini.Params, log *logging.Logger) IArgs {
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
	return args
}

var (
	ErrorArgs  = errors.New("args nil")
	ErrorModel = errors.New("model nil")
)

//-> / -> index
//-> /list -> list
//-> /goods/list/info -> goods/list/info
//-> /goods/ -> goods/index
func (this *HttpContext) autoView(req *http.Request) string {
	path := req.URL.Path
	if path == "" {
		return "index"
	}
	l := len(path)
	if path[l-1] == '/' {
		return path[1:] + "index"
	}
	return path[1:]
}

//mvc模式预处理
func (this *HttpContext) handlerWithArgs(iv IArgs, hv reflect.Value, dv reflect.Value, view string, render string) martini.Handler {
	if !dv.IsValid() {
		panic(errors.New("DefaultHandler miss"))
	}
	return func(c martini.Context, mvc IMVC, rv Render, param martini.Params, req *http.Request, log *logging.Logger) {
		var err error
		mvc.SetView(view)
		mvc.SetRender(StringToRender(render))
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
		if err = this.Validate(args); err != nil {
			err = args.Validate(NewValidateModel(err), mvc)
		} else if fm := this.GetArgsHandler(args); fm != nil {
			_, err = c.Invoke(fm)
		} else if hv.IsValid() {
			_, err = c.Invoke(hv.Interface())
		} else {
			_, err = c.Invoke(dv.Interface())
		}
		if err != nil {
			panic(err)
		}
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
		render := strings.ToUpper(f.Tag.Get("render"))
		handler := f.Tag.Get("handler")
		if handler == "" {
			handler = f.Name
		}
		method := f.Tag.Get("method")
		if method == "" {
			method = pmethod
		}
		method = strings.ToUpper(method)
		in := []martini.Handler{}
		hv := sv.MethodByName(handler + HandlerSuffix)
		dv := sv.MethodByName(DefaultHandler)
		iv, ab := this.IsIArgs(v)
		if ab && url != "" {
			in = append(in, this.handlerWithArgs(iv, hv, dv, view, render))
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

func (this *HttpContext) NewMVCHandler() martini.Handler {
	return func(ctx martini.Context, rv Render, rw http.ResponseWriter, param martini.Params, req *http.Request, log *logging.Logger) {
		mrw, ok := rw.(martini.ResponseWriter)
		if !ok {
			panic(errors.New("ResponseWriter not martini.ResponseWriter"))
		}
		mvc := &DefaultMVC{
			ctx:      ctx,
			model:    &xModel{},
			status:   http.StatusOK,
			req:      req,
			log:      log,
			render:   NONE_RENDER,
			isrender: true,
			rw:       mrw,
			rev:      rv}
		mvc.MapTo(mvc, (*IMVC)(nil))
		mvc.Next()
		if !mvc.isrender {
			return
		}
		mvc.RunRender()
	}
}

func (this *HttpContext) UseDispatcher(c IDispatcher) {
	in := []martini.Handler{this.NewMVCHandler()}
	this.Group(c.URL(), func(r martini.Router) {
		this.useRouter(r, c)
	}, in...)
}
