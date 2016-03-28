package xweb

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/sessions"
	"log"
	"net/http"
	"reflect"
	"sort"
	"time"
)

//default context

var (
	Main = NewContext()
)

func SetValidationFunc(name string, vf ValidationFunc) error {
	return Main.Validator.SetValidationFunc(name, vf)
}

func Validate(v interface{}) error {
	return Main.Validator.Validate(v)
}

func UseCookie(key string, name string, opts sessions.Options) {
	Main.UseCookie(key, name, opts)
}

func UseRedis(addr string) {
	Main.UseRedis(addr)
}

func SetEnv(env string) {
	martini.Env = env
}

func Map(v interface{}) {
	Main.Map(v)
}

func MapTo(v interface{}, t interface{}) {
	Main.MapTo(v, t)
}

func UseDispatcher(c IDispatcher, in ...martini.Handler) {
	Main.UseDispatcher(c, in...)
}

func Use(h martini.Handler) {
	Main.Use(h)
}

func GetDispatcher(t interface{}) IDispatcher {
	v := Main.Injector.Get(reflect.TypeOf(t))
	if !v.IsValid() {
		return nil
	}
	if i, ok := v.Interface().(IDispatcher); ok {
		return i
	}
	return nil
}

func ListenAndServe(addr string, opts ...RenderOptions) error {
	Main.UseRender(opts...)
	return Main.ListenAndServe(addr)
}

func ListenAndServeTLS(addr string, cert, key string, opts ...RenderOptions) error {
	Main.UseRender(opts...)
	return Main.ListenAndServeTLS(addr, cert, key)
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
	Args    string
	Modal   string
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

func (this *Context) UseRedis(addr string) {
	this.Use(InitRedis(addr))
}

func (this *Context) UseRender(opts ...RenderOptions) {
	this.Use(Renderer(opts...))
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
	this.URLS.Sort()
	mc, pc, hc, vc, rc, ac := 0, 0, 0, 0, 0, 0
	for _, u := range this.URLS {
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
		if len(u.Args) > ac {
			ac = len(u.Args)
		}
	}
	fs := fmt.Sprintf("+ %%-%ds %%-%ds %%-%ds %%-%ds %%-%ds %%-%ds\n", mc, pc, ac, hc, vc, rc)
	for _, u := range this.URLS {
		log.Printf(fs, u.Method, u.Pattern, u.Args, u.Handler, u.View, u.Render)
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
