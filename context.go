package xweb

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"github.com/martini-contrib/sessions"
	"log"
	"net/http"
	"reflect"
	"sort"
	"time"
)

//default context

var (
	main = NewContext()
)

func SetValidationFunc(name string, vf ValidationFunc) error {
	return main.Validator.SetValidationFunc(name, vf)
}

func Validate(v interface{}) error {
	return main.Validator.Validate(v)
}

func UseCookie(key string, name string, opts sessions.Options) {
	main.UseCookie(key, name, opts)
}

func UseRedis(addr string) {
	main.UseRedis(addr)
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

func UseDispatcher(c IDispatcher, in ...martini.Handler) {
	main.UseDispatcher(c, in...)
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

func ListenAndServe(addr string, opts ...render.Options) error {
	main.UseRender(opts...)
	return main.ListenAndServe(addr)
}

func ListenAndServeTLS(addr string, cert, key string, opts ...render.Options) error {
	main.UseRender(opts...)
	return main.ListenAndServeTLS(addr, cert, key)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func newRedisPool(server string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp4", server)
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

//初始化Redis
func InitRedis(addr string) martini.Handler {
	pool := newRedisPool(addr)
	return func(c martini.Context) {
		conn := pool.Get()
		defer conn.Close()
		c.Map(conn)
		c.Next()
	}
}

type URLS struct {
	Method  string
	Pattern string
	View    string
	Render  string
	Handler string
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
	urls      URLSlice
}

func (this *Context) UseCookie(key string, name string, opts sessions.Options) {
	store := sessions.NewCookieStore([]byte(key))
	store.Options(opts)
	this.Use(sessions.Sessions(name, store))
}

func (this *Context) UseRedis(addr string) {
	this.Use(InitRedis(addr))
}

func (this *Context) UseRender(opts ...render.Options) {
	this.Use(render.Renderer(opts...))
}

func (this *Context) SetValidationFunc(name string, vf ValidationFunc) error {
	return this.Validator.SetValidationFunc(name, vf)
}

func (this *Context) Validate(v interface{}) error {
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

func (this *Context) printURLS(log *log.Logger) {
	this.urls.Sort()
	mc, pc, hc, vc, rc := 0, 0, 0, 0, 0
	for _, u := range this.urls {
		if len(u.Method) > mc {
			mc = len(u.Method)
		}
		if len(u.Pattern) > pc {
			pc = len(u.Pattern)
		}
		if len(u.Handler) > hc {
			hc = len(u.Handler)
		}
		if len(u.View) > vc {
			vc = len(u.View)
		}
		if len(u.Render) > rc {
			rc = len(u.Render)
		}
	}
	fs := fmt.Sprintf("+ %%-%ds %%-%ds %%-%ds %%-%ds %%-%ds\n", mc, pc, hc, vc, rc)
	for _, u := range this.urls {
		log.Printf(fs, u.Method, u.Pattern, u.Handler, u.View, u.Render)
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
	h.urls = []URLS{}
	h.Martini = m
	h.Router = r
	return h
}
