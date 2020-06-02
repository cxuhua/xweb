package xweb

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cxuhua/xweb/martini"
	"github.com/stretchr/testify/require"
)

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
	kp := NewCacheParams(&cacheimp{}, time.Second, "x113")

	type model struct {
		A string `xml:"a"`
		B int    `xml:"b"`
	}

	testdata := model{A: "astr", B: 100}

	retdata := &model{}

	bcache, err := kp.DoXML(func() (interface{}, error) {
		return testdata, nil
	}, retdata)
	require.NoError(t, err)
	require.Equal(t, false, bcache)
	require.Equal(t, testdata.A, retdata.A)
	require.Equal(t, testdata.B, retdata.B)

	retdata = &model{}
	bcache, err = kp.DoXML(func() (interface{}, error) {
		return testdata, nil
	}, retdata)
	require.NoError(t, err)
	require.Equal(t, true, bcache)
	require.Equal(t, testdata.A, retdata.A)
	require.Equal(t, testdata.B, retdata.B)

	//缓存失效
	time.Sleep(time.Second * 2)

	retdata = &model{}
	bcache, err = kp.DoXML(func() (interface{}, error) {
		return testdata, nil
	}, retdata)
	require.NoError(t, err)
	require.Equal(t, false, bcache)
	require.Equal(t, testdata.A, retdata.A)
	require.Equal(t, testdata.B, retdata.B)
}

func TestCacheDoJSON(t *testing.T) {
	kp := NewCacheParams(&cacheimp{}, time.Second, "x114")
	type model struct {
		A string `json:"a"`
		B int    `json:"b"`
	}

	testdata := model{A: "astr", B: 100}

	retdata := &model{}

	bcache, err := kp.DoJSON(func() (interface{}, error) {
		return testdata, nil
	}, retdata)
	require.NoError(t, err)
	require.Equal(t, false, bcache)
	require.Equal(t, testdata.A, retdata.A)
	require.Equal(t, testdata.B, retdata.B)

	retdata = &model{}
	bcache, err = kp.DoJSON(func() (interface{}, error) {
		return testdata, nil
	}, retdata)
	require.NoError(t, err)
	require.Equal(t, true, bcache)
	require.Equal(t, testdata.A, retdata.A)
	require.Equal(t, testdata.B, retdata.B)

	//缓存失效
	time.Sleep(time.Second * 2)

	retdata = &model{}
	bcache, err = kp.DoJSON(func() (interface{}, error) {
		return testdata, nil
	}, retdata)
	require.NoError(t, err)
	require.Equal(t, false, bcache)
	require.Equal(t, testdata.A, retdata.A)
	require.Equal(t, testdata.B, retdata.B)
}

func TestCacheDoBytes(t *testing.T) {
	kp := NewCacheParams(&cacheimp{}, time.Second, "x115")

	sb := []byte{1, 2, 34}

	bb, bcache, err := kp.DoBytes(func() ([]byte, error) {
		return []byte{1, 2, 34}, nil
	})
	require.NoError(t, err)
	require.Equal(t, false, bcache)
	require.Equal(t, sb, bb)

	bb, bcache, err = kp.DoBytes(func() ([]byte, error) {
		return []byte{1, 2, 34}, nil
	})

	require.NoError(t, err)
	require.Equal(t, true, bcache)
	require.Equal(t, sb, bb)

	//缓存失效
	time.Sleep(time.Second * 2)

	bb, bcache, err = kp.DoBytes(func() ([]byte, error) {
		return []byte{1, 2, 34}, nil
	})

	require.NoError(t, err)
	require.Equal(t, false, bcache)
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
