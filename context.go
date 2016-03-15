package xweb

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"github.com/martini-contrib/sessions"
	"gopkg.in/validator.v2"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"time"
)

type HTTPValues struct {
	url.Values
}

type KeyValue struct {
	Key   string
	Value interface{}
}
type KV []KeyValue

func (this HTTPValues) MD5Sign(key string) string {
	s := this.RawEncode()
	if len(s) > 0 {
		s += "&"
	}
	s += "key=" + key
	return MD5String(s)
}

func (this HTTPValues) RawEncode() string {
	if this.Values == nil {
		return ""
	}
	var buf bytes.Buffer
	keys := make([]string, 0, len(this.Values))
	for k := range this.Values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		vs := this.Values[k]
		if len(vs) == 0 {
			continue
		}
		s := k + "="
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(s)
			buf.WriteString(v)
		}
	}
	return buf.String()
}

func (this HTTPValues) Add(key string, value interface{}) {
	vv := fmt.Sprintf("%v", value)
	this.Values.Add(key, vv)
}

func (this HTTPValues) Set(key string, value interface{}) {
	vv := fmt.Sprintf("%v", value)
	this.Values.Set(key, vv)
}

func (this HTTPValues) IsEmpty() bool {
	return len(this.Values) == 0
}

func NewHTTPValues() HTTPValues {
	ret := HTTPValues{}
	ret.Values = url.Values{}
	return ret
}

type HTTPClient struct {
	http.Client
	IsSecure bool
}

var (
	NoDataError = errors.New("http not response data")
)

func (this HTTPClient) readResponseData(res *http.Response) ([]byte, error) {
	if res.Body == nil {
		return nil, NoDataError
	}
	defer res.Body.Close()
	return ioutil.ReadAll(res.Body)
}

func (this HTTPClient) Get(url string, q HTTPValues) ([]byte, error) {
	if !q.IsEmpty() {
		url = url + "?" + q.Encode()
	}
	if res, err := this.Client.Get(url); err != nil {
		return nil, err
	} else {
		return this.readResponseData(res)
	}
}

func (this HTTPClient) PostForm(url string, v HTTPValues) ([]byte, error) {
	if res, err := this.Client.PostForm(url, v.Values); err != nil {
		return nil, err
	} else {
		return this.readResponseData(res)
	}
}

//http.MethodPost http.MethodGet http.MethodHead...
func (this HTTPClient) NewRequest(method, url string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, url, body)
}

func (this HTTPClient) Do(req *http.Request) (*http.Response, error) {
	return this.Client.Do(req)
}

func NewHTTPClient(secure bool) HTTPClient {
	ret := HTTPClient{}
	tr := &http.Transport{}
	if secure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	ret.Client = http.Client{Transport: tr}
	ret.IsSecure = secure
	return ret
}

//default context

var (
	main = NewContext()
)

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

type Context struct {
	martini.ClassicMartini
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
