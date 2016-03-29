package xweb

import (
	"encoding/json"
	"errors"
	. "gopkg.in/check.v1"
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
	JSONArgs
	A string `json:"a" validate:"len=5,exists=false"`
	B int    `json:"b" validate:"min=2,max=6"`
}

func (this *TestArgs) Model() IModel {
	return &TestModel{}
}

func (this *TestArgs) Handler(m *TestModel) {
	m.C = this.A + "54321"
	m.D = this.B + 10
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
	Test TestArgs `url:"/post/json" method:"POST"`
}

type BDistacher struct {
	HTTPDispatcher
	A int
}

type WebSuite struct {
}

var _ = Suite(&WebSuite{})

func (this *WebSuite) SetUpSuite(c *C) {
	m.SetValidationFunc("exists", userExists)
	m.UseRender()
	m.UseDispatcher(new(TestDistacher))
}

func (this *WebSuite) TearDownSuite(c *C) {

}

func (this *WebSuite) TestHttpPostJsonValidateSuccess(c *C) {
	res := httptest.NewRecorder()
	body := strings.NewReader(`{"a":"12345","b":3}`)
	req, _ := http.NewRequest(http.MethodPost, "/post/json", body)
	m.ServeHTTP(res, req)
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
	m.ServeHTTP(res, req)
	m := &ValidateModel{}
	c.Assert(res.Body, NotNil)
	d := res.Body.Bytes()
	c.Log(string(d))
	c.Assert(json.Unmarshal(d, m), IsNil)
	c.Assert(len(m.Errors), Equals, 2)
}

func (this *WebSuite) TestQueryArgs(c *C) {
	var v interface{} = nil

	v = &URLArgs{}
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
}
