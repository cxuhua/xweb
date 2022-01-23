package xweb

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cxuhua/xweb/martini"
	"github.com/stretchr/testify/require"
)

type Info struct {
	A    int     `form:"a"`
	B    string  `url:"b"`
	C    float32 `cookie:"c"`
	D    bool    `header:"d"`
	Info *Info
}

func TestMapFormBindValue(t *testing.T) {
	i := &Info{}
	form := url.Values{}
	form.Set("a", "1")
	urls := url.Values{}
	urls.Set("b", "b")
	cookies := url.Values{}
	cookies.Set("c", "12.3")
	header := url.Values{}
	header.Set("d", "true")
	MapFormBindValue(reflect.ValueOf(i), form, nil, urls, cookies, header)
	if i.A != 1 {
		t.Fatal("A test error")
	}
	if i.B != "b" {
		t.Fatal("B test error")
	}
	if i.C != 12.3 {
		t.Fatal("C test error")
	}
	if i.D != true {
		t.Fatal("D test error")
	}
}

func TestBytesAes(t *testing.T) {
	a := []byte{1, 2, 3}
	b, err := BytesEncrypt(a)
	if err != nil {
		t.Fatal(err)
	}
	c, err := BytesDecrypt(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, c) {
		t.Error("test failed")
	}
}

func TestStringAes(t *testing.T) {
	a := "12121212"
	b, err := TokenEncrypt(a)
	if err != nil {
		t.Fatal(err)
	}
	c, err := TokenDecrypt(b)
	if err != nil {
		t.Fatal(err)
	}
	if a != c {
		t.Error("test failed")
	}
}

func TestShutdown(t *testing.T) {
	go func() {
		err := ListenAndServe(":9100")
		log.Println("closed", err)
	}()
	time.Sleep(time.Second)
	m.Shutdown()
	time.Sleep(time.Second * 30)
}

func TestGenId(t *testing.T) {
	smap := map[string]bool{}
	for i := 0; i < 100000; i++ {
		id := GenId()
		if _, has := smap[id]; has {
			t.Errorf("repeat id %d - %s", i, id)
			break
		}
		smap[id] = true
		log.Println(id, os.Getpid(), len(id))
	}
}

func TestHttpGet(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()
	c := NewHTTPClientWithContext(ctx, "https://baidu.com")
	req, err := c.NewGet("/")
	if err != nil {
		t.Fatal(err)
	}
	res, err := c.Do(req)
	log.Println(res.ToString())
}

type cachenode struct {
	b   []byte
	exp time.Time
}

var (
	lck sync.RWMutex
	cks = map[string]cachenode{}
)

type cacheimp struct {
}

func (c *cacheimp) TTL(k string) (time.Duration, error) {
	lck.RLock()
	defer lck.RUnlock()
	node, ok := cks[k]
	if !ok {
		return 0, fmt.Errorf("not found")
	}
	tp := node.exp.Sub(time.Now())
	if tp < 0 {
		tp = 0
	}
	return tp, nil
}

//设置值
func (c *cacheimp) Set(k string, v interface{}, exp ...time.Duration) error {
	lck.Lock()
	defer lck.Unlock()
	now := time.Now()
	if len(exp) > 0 {
		now = now.Add(exp[0])
	}
	cks[k] = cachenode{b: v.([]byte), exp: now}
	return nil
}

//获取值
func (c *cacheimp) Get(k string, v interface{}) error {
	lck.RLock()
	defer lck.RUnlock()
	vp, ok := cks[k]
	if !ok {
		return fmt.Errorf("key %s miss", k)
	}
	if vp.exp.Sub(time.Now()) < 0 {
		return fmt.Errorf("key value expire")
	}
	rp, ok := v.(*[]byte)
	if !ok {
		return fmt.Errorf("v type error")
	}
	*rp = vp.b
	return nil
}

//删除值
func (c *cacheimp) Del(k ...string) (int64, error) {
	lck.Lock()
	defer lck.Unlock()
	delete(cks, k[0])
	return 1, nil
}

type locker struct {
	key string
	c   *cacheimp
}

//Release 释放锁
func (l *locker) Release() {
	l.c.Del(l.key)
}

func (l *locker) Refresh(ttl time.Duration) error {
	lck.Lock()
	defer lck.Unlock()
	node, ok := cks[l.key]
	if !ok {
		return fmt.Errorf("not found")
	}
	node.exp = time.Now().Add(ttl)
	return nil
}

//TTL 锁超时时间 返回0表示锁已经释放
func (l *locker) TTL() (time.Duration, error) {
	lck.Lock()
	defer lck.Unlock()
	node, ok := cks[l.key]
	if !ok {
		return 0, fmt.Errorf("not found")
	}
	tp := node.exp.Sub(time.Now())
	if tp <= 0 {
		delete(cks, l.key)
		return 0, nil
	}
	return tp, nil
}

//获取锁
func (c *cacheimp) Locker(key string, ttl time.Duration, meta ...string) (ILocker, error) {
	lck.Lock()
	defer lck.Unlock()
	l := &locker{key: "lck_" + key, c: c}
	lp, ok := cks[l.key]
	if ok && lp.exp.Sub(time.Now()) > 0 {
		return nil, fmt.Errorf("locker exist")
	}
	cks[l.key] = cachenode{b: []byte{1, 2, 3}, exp: time.Now().Add(ttl)}
	return l, nil
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
	return NewCacheParams(imp, time.Second, time.Hour, "111")
}

func (a *TestArgs) Handler(m *TestModel, mvc IMVC) error {
	m.A = 171718
	m.Set("abc", "123")
	return nil
}

func CacheNew() martini.Handler {
	imp := &cacheimp{}
	return func(m martini.Context) {
		m.Map(imp)
	}
}

func TestLocker(t *testing.T) {
	c := &cacheimp{}
	l, err := c.Locker("a", time.Second)
	require.NoError(t, err)
	ttl, err := l.TTL()
	require.NoError(t, err)
	require.Equal(t, time.Second, ttl)
	_, err = c.Locker("a", time.Second)
	require.Error(t, err)
	time.Sleep(time.Second * 2)
	l, err = c.Locker("a", time.Second*2)
	require.NoError(t, err)
	ttl, err = l.TTL()
	require.NoError(t, err)
	require.Equal(t, time.Second*2, ttl)
}

func TestCacheDoXML(t *testing.T) {
	kp := NewCacheParams(&cacheimp{}, time.Second, 0, "x113")

	type model struct {
		A string `xml:"a"`
		B int    `xml:"b"`
	}

	testdata := model{A: "astr", B: 100}

	retdata := &model{}

	bcache, err := kp.DoXML(func() (interface{}, error) {
		return testdata, nil
	}, retdata, 0)
	require.NoError(t, err)
	require.Equal(t, false, bcache > 0)
	require.Equal(t, testdata.A, retdata.A)
	require.Equal(t, testdata.B, retdata.B)

	retdata = &model{}
	bcache, err = kp.DoXML(func() (interface{}, error) {
		return testdata, nil
	}, retdata, 0)
	require.NoError(t, err)
	require.Equal(t, true, bcache > 0)
	require.Equal(t, testdata.A, retdata.A)
	require.Equal(t, testdata.B, retdata.B)

	//缓存失效
	time.Sleep(time.Second * 2)

	retdata = &model{}
	bcache, err = kp.DoXML(func() (interface{}, error) {
		return testdata, nil
	}, retdata, 0)
	require.NoError(t, err)
	require.Equal(t, false, bcache > 0)
	require.Equal(t, testdata.A, retdata.A)
	require.Equal(t, testdata.B, retdata.B)
}

func TestCacheDoJSON(t *testing.T) {
	kp := NewCacheParams(&cacheimp{}, time.Second, 0, "x114")
	type model struct {
		A string `json:"a"`
		B int    `json:"b"`
	}

	testdata := model{A: "astr", B: 100}

	retdata := &model{}

	bcache, err := kp.DoJSON(func() (interface{}, error) {
		return testdata, nil
	}, retdata, 0)
	require.NoError(t, err)
	require.Equal(t, false, bcache > 0)
	require.Equal(t, testdata.A, retdata.A)
	require.Equal(t, testdata.B, retdata.B)

	retdata = &model{}
	bcache, err = kp.DoJSON(func() (interface{}, error) {
		return testdata, nil
	}, retdata, 0)
	require.NoError(t, err)
	require.Equal(t, true, bcache > 0)
	require.Equal(t, testdata.A, retdata.A)
	require.Equal(t, testdata.B, retdata.B)

	//缓存失效
	time.Sleep(time.Second * 2)

	retdata = &model{}
	bcache, err = kp.DoJSON(func() (interface{}, error) {
		return testdata, nil
	}, retdata, 0)
	require.NoError(t, err)
	require.Equal(t, false, bcache > 0)
	require.Equal(t, testdata.A, retdata.A)
	require.Equal(t, testdata.B, retdata.B)
}

func TestTryDoBytes(t *testing.T) {
	kp := NewCacheParams(&cacheimp{}, time.Second, time.Second*2, "x115")

	sb := []byte{1, 2, 34}

	bb, bcache, err := kp.DoBytes(func() ([]byte, error) {
		return []byte{1, 2, 34}, nil
	}, time.Second)
	require.NoError(t, err)
	require.Equal(t, false, bcache > 0)
	require.Equal(t, sb, bb)
	time.Sleep(time.Second * 2)

	for i := 0; i < 5; i++ {
		go func() {
			bb, bcache, err = kp.DoBytes(func() ([]byte, error) {
				return []byte{1, 2, 34}, nil
			}, time.Second)
		}()

	}
	time.Sleep(time.Second * 30)
}

func TestCacheDoBytes(t *testing.T) {
	kp := NewCacheParams(&cacheimp{}, time.Second, 0, "x115")

	sb := []byte{1, 2, 34}

	bb, bcache, err := kp.DoBytes(func() ([]byte, error) {
		return []byte{1, 2, 34}, nil
	}, time.Second)
	require.NoError(t, err)
	require.Equal(t, false, bcache > 0)
	require.Equal(t, sb, bb)

	bb, bcache, err = kp.DoBytes(func() ([]byte, error) {
		return []byte{1, 2, 34}, nil
	}, time.Second)

	require.NoError(t, err)
	require.Equal(t, true, bcache > 0)
	require.Equal(t, sb, bb)

	//缓存失效
	time.Sleep(time.Second * 2)

	bb, bcache, err = kp.DoBytes(func() ([]byte, error) {
		return []byte{1, 2, 34}, nil
	}, time.Second)

	require.NoError(t, err)
	require.Equal(t, false, bcache > 0)
	require.Equal(t, sb, bb)

	kp.Remove()
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
