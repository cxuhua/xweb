package xweb

import (
	"fmt"
	"github.com/cxuhua/xweb/logging"
	"github.com/cxuhua/xweb/martini"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"sort"
)

//default context

var (
	m            = NewHttpContext()
	LoggerFormat = logging.MustStringFormatter(`%{color}%{time:15:04:05.000} %{shortfile} %{shortfunc} ▶ %{level:.5s} %{id:d}%{color:reset} %{message}`)
	LoggerPrefix = ""
)

func WritePID() {
	pid := fmt.Sprintf("%v", os.Getpid())
	ioutil.WriteFile("pid", []byte(pid), 0666)
}

func WritePIDFile(file string) {
	pid := fmt.Sprintf("%v", os.Getpid())
	ioutil.WriteFile(file, []byte(pid), 0666)
}

func AddExtType(ext string, typ string) {
	mime.AddExtensionType(ext, typ)
}

func InitLogger(w io.Writer) {
	m.InitDefaultLogger(w)
}

func ServeHTTP(res http.ResponseWriter, req *http.Request) {
	m.ServeHTTP(res, req)
}

func SetValidationFunc(name string, vf ValidationFunc) error {
	return m.Validator.SetValidationFunc(name, vf)
}

func Validate(v IArgs) error {
	return m.Validator.Validate(v)
}

func SetEnv(env string) {
	martini.Env = env
}

func Map(v interface{}) {
	m.Map(v)
}

func GetBody(req *http.Request) ([]byte, error) {
	return m.GetBody(req)
}

func MapTo(v interface{}, t interface{}) {
	m.MapTo(v, t)
}

func UseDispatcher(c IDispatcher) {
	m.UseDispatcher(c)
}

func Use(h martini.Handler) {
	m.Use(h)
}

func UseRender(opts ...RenderOptions) {
	m.UseRender(opts...)
}

func Serve(addr string) error {
	return m.ListenAndServe(addr)
}

func ListenAndServe(addr string) error {
	return m.ListenAndServe(addr)
}

func ServeTLS(addr string, cert, key string) error {
	return m.ListenAndServeTLS(addr, cert, key)
}

func ListenAndServeTLS(addr string, cert, key string) error {
	return m.ListenAndServeTLS(addr, cert, key)
}

func Logger() *logging.Logger {
	return m.GetLogger()
}

type URLS struct {
	Method  string
	Pattern string
	View    string
	Render  string
	Args    IArgs
}

type URLSlice []URLS

func (p URLSlice) Len() int {
	return len(p)
}
func (p URLSlice) Less(i, j int) bool {
	return p[i].Pattern < p[j].Pattern
}
func (p URLSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
func (p URLSlice) Sort() {
	sort.Sort(p)
}

type HttpContext struct {
	martini.ClassicMartini
	Validator *Validator
	URLS      URLSlice
}

func (this *HttpContext) InitDefaultLogger(w io.Writer) {
	lw := logging.NewLogBackend(w, LoggerPrefix, 0)
	backend := logging.NewBackendFormatter(lw, LoggerFormat)
	logging.SetBackend(backend)
}

func (this *HttpContext) UseRender(opts ...RenderOptions) {
	this.Use(Renderer(opts...))
}

func (this *HttpContext) SetValidationFunc(name string, vf ValidationFunc) error {
	return this.Validator.SetValidationFunc(name, vf)
}

func (this *HttpContext) Validate(v IArgs) error {
	if !v.IsValidate() {
		return nil
	}
	return this.Validator.Validate(v)
}

func (this *HttpContext) Logger() *logging.Logger {
	return this.GetLogger()
}

func (this *HttpContext) ListenAndServe(addr string) error {
	this.PrintURLS()
	this.Logger().Infof("http listening on %s (%s)\n", addr, martini.Env)
	return http.ListenAndServe(addr, this)
}

func (this *HttpContext) ListenAndServeTLS(addr string, cert, key string) error {
	this.PrintURLS()
	this.Logger().Infof("https listening on %s (%s)\n", addr, martini.Env)
	return http.ListenAndServeTLS(addr, cert, key, this)
}

func (this *HttpContext) PrintURLS() {
	log := this.GetLogger()
	this.URLS.Sort()
	mc, pc, vc, rc := 0, 0, 0, 0
	for _, u := range this.URLS {
		if len(u.Method) > mc {
			mc = len(u.Method)
		}
		if len(u.Pattern) > pc {
			pc = len(u.Pattern)
		}
		if len(u.View) > vc {
			vc = len(u.View)
		}
		if u.Render == "" {
			u.Render = RenderToString(u.Args.Model().Render())
		}
		if len(u.Render) > rc {
			rc = len(u.Render)
		}
	}
	fs := fmt.Sprintf("+ %%-%ds %%-%ds %%-%ds %%-%ds\n", mc, pc, vc, rc)
	for _, u := range this.URLS {
		log.Infof(fs, u.Method, u.Pattern, u.View, u.Render)
	}
}

func NewHttpContext() *HttpContext {
	h := &HttpContext{}
	r := martini.NewRouter()
	m := martini.New()
	m.Use(martini.Logger())
	m.Use(martini.Recovery())
	m.Use(martini.Static("public"))
	m.MapTo(r, (*martini.Routes)(nil))
	m.Action(r.Handle)
	h.Validator = NewValidator()
	h.URLS = []URLS{}
	h.Martini = m
	h.Router = r
	return h
}
