package xweb

import (
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"gopkg.in/validator.v2"
	"log"
	"reflect"
)

//default context

var (
	main = NewContext()
)

func Map(v interface{}) {
	main.Map(v)
}

func MapTo(v interface{}, t interface{}) {
	main.MapTo(v, t)
}

func Dispatcher(c IDispatcher) {
	main.Dispatcher(c)
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

func RunOnAddr(addr string) {
	main.RunOnAddr(addr)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

type Context struct {
	martini.ClassicMartini
}

func (this *Context) SetValidationFunc(name string, vf validator.ValidationFunc) error {
	return validator.SetValidationFunc(name, vf)
}

func (this *Context) Validate(v interface{}) error {
	return validator.Validate(v)
}

func (this *Context) Logger() *log.Logger {
	return this.Injector.Get(reflect.TypeOf((*log.Logger)(nil))).Interface().(*log.Logger)
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
	h.Use(render.Renderer(render.Options{
		IndentJSON: false,
		Directory:  "templates",
		Extensions: []string{".tmpl"},
	}))
	return h
}
