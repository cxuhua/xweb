package xweb

import (
	"encoding/json"
	. "gopkg.in/check.v1"
	"net/http"
	"net/http/httptest"
	"strings"
)

/////////////////////////////////////////////////////////////////////////////////////////////

type TestArgs struct {
	JsonArgs        // `form:"Body"`
	A        string `json:"a" validate:"len=5"`
	B        int    `json:"b" validate:"min=2,max=6"`
}

func (this TestArgs) Model() IModel {
	return new(TestModel)
}

type TestModel struct {
	IModel `json:"-"`
	A      string `json:"a"`
	B      int    `json:"b"`
}

// func (this *TestModel) View() string {
// 	return "index"
// }

// func (this *TestModel) HTML(args TestArgs) {
// 	this.A = args.A + "54321"
// 	this.B = args.B + 10
// }

func (this *TestModel) JSON(args TestArgs) {
	this.A = args.A + "54321"
	this.B = args.B + 10
}

//////////////////////////////////////////////////////////////////////////////////////
type TestDistacher struct {
	HTTPDispatcher
	POST struct {
		P1 TestArgs `url:"/post/json"`
	}
}

type WebSuite struct {
}

var _ = Suite(&WebSuite{})

func (this *WebSuite) SetUpSuite(c *C) {
	main.UseRender()
	main.SetDispatcher(new(TestDistacher))
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
	c.Assert(json.Unmarshal(d, m), IsNil)
	c.Assert(m.A, Equals, "1234554321")
	c.Assert(m.B, Equals, 13)
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
