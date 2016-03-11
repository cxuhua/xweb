package xweb

import (
	"encoding/json"
	. "gopkg.in/check.v1"
	"net/http"
	"net/http/httptest"
	"strings"
)

//定义输入参数
type TestArgs struct {
	JsonArgs        // `form:"Body"`
	A        string `json:"a" validate:"len=5"`
	B        int    `json:"b" validate:"min=2,max=6"`
}

//定义输出模型
type TestModel struct {
	IModel `json:"-"`
	C      string `json:"a"`
	D      int    `json:"b"`
}

//使用返回的模型处理参数
func (this TestArgs) Model() IModel {
	return new(TestModel)
}

// func (this *TestModel) View() string {
// 	return "index"
// }

//处理参数并返回 html类型(如果定义了HTML方法)
// func (this *TestModel) HTML(args TestArgs) {
// 	this.A = args.A + "54321"
// 	this.B = args.B + 10
// }

//处理参数并返回 json类型(如果定义了JSON方法)
//JSON HTML XML ANY
func (this *TestModel) JSON(args TestArgs) {
	this.C = args.A + "54321"
	this.D = args.B + 10
}

//定义url分发器
type TestDistacher struct {
	HTTPDispatcher
	POST struct {
		P1 TestArgs `url:"/json"`
	} `url:"/post"`
}

type BDistacher struct {
	HTTPDispatcher
	A int
}

type WebSuite struct {
}

var _ = Suite(&WebSuite{})

func (this *WebSuite) SetUpSuite(c *C) {
	main.UseRender()
	main.UseDispatcher(new(TestDistacher))
}

func (this *WebSuite) TearDownSuite(c *C) {

}

func (this *WebSuite) TestHttpPostJsonValidateSuccess(c *C) {
	res := httptest.NewRecorder()
	body := strings.NewReader(`{"a":"12345","b":3}`)
	req, _ := http.NewRequest(http.MethodPost, "/post/json", body)
	main.ServeHTTP(res, req)
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
	main.ServeHTTP(res, req)
	m := &HTTPValidateModel{}
	c.Assert(res.Body, NotNil)
	d := res.Body.Bytes()
	c.Log(string(d))
	c.Assert(json.Unmarshal(d, m), IsNil)
	c.Assert(len(m.Errors), Equals, 2)
}

func (this *WebSuite) TestQueryArgs(c *C) {
	var v interface{} = nil

	v = QueryArgs{}
	_, ok := v.(IArgs)
	c.Assert(ok, Equals, true)

	v = JsonArgs{}
	_, ok = v.(IArgs)
	c.Assert(ok, Equals, true)

	v = FormArgs{}
	_, ok = v.(IArgs)
	c.Assert(ok, Equals, true)

	v = XmlArgs{}
	_, ok = v.(IArgs)
	c.Assert(ok, Equals, true)
}
