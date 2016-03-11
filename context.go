package xweb

import (
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"gopkg.in/validator.v2"
	"log"
	"net/http"
	"reflect"
)

//default context

var (
	main = NewContext()
)

func UseRender(opts ...render.Options) {
	main.UseRender(opts...)
}

func SetEnv(env string) {
	martini.Env = env
}

func Map(v interface{}) {
	main.Map(v)
}

func MapTo(v interface{}, t interface{}) {
	main.MapTo(v, t)
}

func UseDispatcher(c IDispatcher) {
	main.UseDispatcher(c)
}

func Group(url string, rf func(martini.Router), hs ...martini.Handler) {
	main.Group(url, rf, hs...)
}

func Use(h martini.Handler) {
	main.Use(h)
}

func GetDispatcher(t interface{}) IDispatcher {
	v := main.Injector.Get(reflect.TypeOf(t))
	if !v.IsValid() {
		return nil
	}
	if i, ok := v.Interface().(IDispatcher); ok {
		return i
	}
	return nil
}

func SetValidationFunc(name string, vf validator.ValidationFunc) error {
	return main.SetValidationFunc(name, vf)
}

func Validate(v interface{}) error {
	return main.Validate(v)
}

func ListenAndServe(addr string) error {
	main.UseRender()
	return main.ListenAndServe(addr)
}

func ListenAndServeTLS(addr string, cert, key string) error {
	main.UseRender()
	return main.ListenAndServeTLS(addr, cert, key)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

type Context struct {
	martini.ClassicMartini
}

func (this *Context) UseRender(opts ...render.Options) {
	this.Use(render.Renderer(opts...))
}

func (this *Context) SetValidationFunc(name string, vf validator.ValidationFunc) error {
	return validator.SetValidationFunc(name, vf)
}

func (this *Context) Validate(v interface{}) error {
	return validator.Validate(v)
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
		log.Printf("http listening on %s (%s)\n", addr, martini.Env)
	}
	return http.ListenAndServe(addr, this)
}

func (this *Context) ListenAndServeTLS(addr string, cert, key string) error {
	if log := this.Logger(); log != nil {
		log.Printf("https listening on %s (%s)\n", addr, martini.Env)
	}
	return http.ListenAndServeTLS(addr, cert, key, this)
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

	h.Martini = m
	h.Router = r
	return h
}
