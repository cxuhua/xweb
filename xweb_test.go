package xweb

import (
	"encoding/json"
	"errors"
	. "gopkg.in/check.v1"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
)

var (
	ErrUserExists = TextErr{errors.New("user exists")}
)

func userExists(v interface{}, param string) error {
	if param == "true" {
		return ErrUserExists
	}
	return nil
}

//定义输入参数
type TestArgs struct {
	JSONArgs        // `form:"Body"`
	A        string `json:"a" validate:"len=5,exists=false"`
	B        int    `json:"b" validate:"min=2,max=6"`
}

//定义输出模型
type TestModel struct {
	HTTPModel
	C string `json:"a"`
	D int    `json:"b"`
}

//定义url分发器
type TestDistacher struct {
	HTTPDispatcher
	PX TestArgs `url:"/post/json" method:"POST" handler:"P1" render:"JSON"`
	P2 IArgs    `url:"/post/test" method:"POST"`
}

func (this *TestDistacher) P2Handler() {

}

func (this *TestDistacher) P1Handler(args TestArgs, c IMVC) {
	m := &TestModel{}
	c.SetModel(m)
	m.C = args.A + "54321"
	m.D = args.B + 10
}

type BDistacher struct {
	HTTPDispatcher
	A int
}

type WebSuite struct {
}

var _ = Suite(&WebSuite{})

func (this *WebSuite) SetUpSuite(c *C) {
	Main.SetValidationFunc("exists", userExists)
	Main.UseRender()
	Main.UseDispatcher(new(TestDistacher))
	log.Println(Main.URLS)
}

func (this *WebSuite) TearDownSuite(c *C) {

}

func (this *WebSuite) TestHttpPostJsonValidateSuccess(c *C) {
	res := httptest.NewRecorder()
	body := strings.NewReader(`{"a":"12345","b":3}`)
	req, _ := http.NewRequest(http.MethodPost, "/post/json", body)
	Main.ServeHTTP(res, req)
	m := &TestModel{}
	c.Assert(res.Body, NotNil)
	d := res.Body.Bytes()
	c.Log(string(d))
	c.Assert(json.Unmarshal(d, m), IsNil)
	c.Assert(m.C, Equals, "1234554321")
	c.Assert(m.D, Equals, 13)
}

func (this *WebSuite) TestHttpPostJsonValidateError(c *C) {
	res := httptest.NewRecorder()
	body := strings.NewReader(`{"a":"123456","b":300}`)
	req, _ := http.NewRequest(http.MethodPost, "/post/json", body)
	Main.ServeHTTP(res, req)
	m := &ValidateModel{}
	c.Assert(res.Body, NotNil)
	d := res.Body.Bytes()
	c.Log(string(d))
	c.Assert(json.Unmarshal(d, m), IsNil)
	c.Assert(len(m.Errors), Equals, 2)
}

func (this *WebSuite) TestQueryArgs(c *C) {
	var v interface{} = nil

	v = &Args{}
	_, ok := v.(IArgs)
	c.Assert(ok, Equals, true)

	v = &JSONArgs{}
	_, ok = v.(IArgs)
	c.Assert(ok, Equals, true)

	v = &FORMArgs{}
	_, ok = v.(IArgs)
	c.Assert(ok, Equals, true)

	v = &XMLArgs{}
	_, ok = v.(IArgs)
	c.Assert(ok, Equals, true)

	v = &QUERYArgs{}
	_, ok = v.(IArgs)
	c.Assert(ok, Equals, true)
}
