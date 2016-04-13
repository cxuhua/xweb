package xweb

import (
	"fmt"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/sessions"
	"log"
	"net/http"
	"reflect"
	"sort"
)

//default context

var (
	m = NewContext()
)

func SetValidationFunc(name string, vf ValidationFunc) error {
	return m.Validator.SetValidationFunc(name, vf)
}

func Validate(v IArgs) error {
	return m.Validator.Validate(v)
}

func UseCookie(key string, name string, opts sessions.Options) {
	m.UseCookie(key, name, opts)
}

func SetEnv(env string) {
	martini.Env = env
}

func Map(v interface{}) {
	m.Map(v)
}

func MapTo(v interface{}, t interface{}) {
	m.MapTo(v, t)
}

func UseDispatcher(c IDispatcher, in ...martini.Handler) {
	m.UseDispatcher(c, in...)
}

func Use(h martini.Handler) {
	m.Use(h)
}

func GetDispatcher(t interface{}) IDispatcher {
	v := m.Injector.Get(reflect.TypeOf(t))
	if !v.IsValid() {
		return nil
	}
	if i, ok := v.Interface().(IDispatcher); ok {
		return i
	}
	return nil
}

func ListenAndServe(addr string, opts ...RenderOptions) error {
	m.UseRender(opts...)
	return m.ListenAndServe(addr)
}

func ListenAndServeTLS(addr string, cert, key string, opts ...RenderOptions) error {
	m.UseRender(opts...)
	return m.ListenAndServeTLS(addr, cert, key)
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

type Context struct {
	martini.ClassicMartini
	Validator *Validator
	URLS      URLSlice
}

func (this *Context) UseCookie(key string, name string, opts sessions.Options) {
	store := sessions.NewCookieStore([]byte(key))
	store.Options(opts)
	this.Use(sessions.Sessions(name, store))
}

func (this *Context) UseRender(opts ...RenderOptions) {
	this.Use(Renderer(opts...))
}

func (this *Context) SetValidationFunc(name string, vf ValidationFunc) error {
	return this.Validator.SetValidationFunc(name, vf)
}

func (this *Context) Validate(v IArgs) error {
	if !v.IsValidate() {
		return nil
	}
	return this.Validator.Validate(v)
}

func (this *Context) Logger() *log.Logger {
	t := reflect.TypeOf((*log.Logger)(nil))
	if v := this.Injector.Get(t); !v.IsValid() {
		return nil
	} else if r, ok := v.Interface().(*log.Logger); ok {
		return r
	} else {
		return nil
	}
}

func (this *Context) ListenAndServe(addr string) error {
	if log := this.Logger(); log != nil {
		this.printURLS(log)
		log.Printf("http listening on %s (%s)\n", addr, martini.Env)
	}
	return http.ListenAndServe(addr, this)
}

func (this *Context) ListenAndServeTLS(addr string, cert, key string) error {
	if log := this.Logger(); log != nil {
		this.printURLS(log)
		log.Printf("https listening on %s (%s)\n", addr, martini.Env)
	}
	return http.ListenAndServeTLS(addr, cert, key, this)
}

//分析参数文档
func (this *Context) DumpDoc(v interface{}) {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		log.Println(t.Field(i))
	}
}

func (this *Context) printURLS(log *log.Logger) {
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
			u.Render = u.Args.Model().Render()
		}
		if len(u.Render) > rc {
			rc = len(u.Render)
		}
	}
	fs := fmt.Sprintf("+ %%-%ds %%-%ds %%-%ds %%-%ds\n", mc, pc, vc, rc)
	for _, u := range this.URLS {
		log.Printf(fs, u.Method, u.Pattern, u.View, u.Render)
	}
}

func NewContext() *Context {
	h := new(Context)
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
