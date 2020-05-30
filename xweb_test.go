package xweb

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cxuhua/xweb/martini"
)

var (
	cks = map[string][]byte{}
)

type cacheimp struct {
}

//设置值
func (c *cacheimp) Set(k string, v interface{}, exp ...time.Duration) error {
	cks[k] = v.([]byte)
	return nil
}

//获取值
func (c *cacheimp) Get(k string, v interface{}) error {
	vp, ok := cks[k]
	if !ok {
		return fmt.Errorf("key %s miss", k)
	}
	rp, ok := v.(*[]byte)
	if !ok {
		return fmt.Errorf("v type error")
	}
	*rp = vp
	return nil
}

//删除值
func (c *cacheimp) Del(k ...string) (int64, error) {
	return 0, nil
}

type TestModel struct {
	JSONModel `json:"-"`
	A         int `json:"a"`
}

type TestArgs struct {
	JSONArgs
	Info string `json:"info"`
}

func (a *TestArgs) Model() IModel {
	return &TestModel{A: 1000}
}

//如果有此方法需要处理缓存,返回缓存实现，key,超时时间
//key为空将不进行缓存处理
func (a *TestArgs) CacheParams(imp ICache, mvc IMVC) *CacheParams {
	return &CacheParams{
		Imp:  imp,
		Key:  "111",
		Time: time.Second,
	}
}

func (a *TestArgs) Handler(m *TestModel, mvc IMVC) {
	m.A = 171718
}

func CacheNew() martini.Handler {
	imp := &cacheimp{}
	return func(m martini.Context) {
		m.Map(imp)
	}
}

func TestCache(t *testing.T) {
	response := httptest.NewRecorder()
	response.Body = new(bytes.Buffer)

	type D struct {
		HTTPDispatcher
		Test TestArgs `url:"/test" method:"POST"`
	}

	Use(CacheNew())
	UseRender()
	UseDispatcher(&D{})

	req, err := http.NewRequest("POST", "http://localhost:3000/test", strings.NewReader(`{"info":"trestinfo"}`))
	if err != nil {
		t.Error(err)
	}

	ServeHTTP(response, req)

	for k, v := range response.Header() {
		log.Println(k, v)
	}

	dat, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Error(err)
	}
	log.Println(string(dat))

	req, err = http.NewRequest("POST", "http://localhost:3000/test", strings.NewReader(`{"info":"trestinfo"}`))
	if err != nil {
		t.Error(err)
	}

	ServeHTTP(response, req)

	for k, v := range response.Header() {
		log.Println(k, v)
	}

	dat, err = ioutil.ReadAll(response.Body)
	if err != nil {
		t.Error(err)
	}
	log.Println(string(dat))
}
