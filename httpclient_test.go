package xweb

import (
	. "gopkg.in/check.v1"
	"net/http"
)

type HttpSuite struct {
}

var _ = Suite(&HttpSuite{})

func (this *HttpSuite) SetUpSuite(c *C) {

}

func (this *HttpSuite) TearDownSuite(c *C) {

}

func (this *HttpSuite) TestHttpRequest(c *C) {
	client := NewHTTPClient("http://www.baidu.com")
	req, err := client.NewRequest(http.MethodGet, "/", nil)
	c.Assert(err, IsNil)
	res, err := client.Do(req)
	c.Assert(err, IsNil)
	data, err := readResponse(res.Response)
	c.Assert(err, IsNil)
	c.Log(string(data))
	c.Assert(len(data) > 0, Equals, true)
}

func (this *HttpSuite) TestHTTPS(c *C) {
	http := NewHTTPClient("https://www.baidu.com")
	d, err := http.Get("/", HTTPValues{})
	c.Assert(err, IsNil)
	data, err := d.ToBytes()
	c.Assert(err, IsNil)
	c.Log(string(data))
	c.Assert(len(data) > 0, Equals, true)
}

func (this *HttpSuite) TestURL(c *C) {
	v := NewHTTPValues()
	v.Add("dd", 800)
	v.Set("a", "bb")
	v.Set("c", "&3847384^dfdfnjdfnj")
	c.Log(v.RawEncode())
	c.Assert(v.RawEncode(), Equals, "a=bb&c=&3847384^dfdfnjdfnj&dd=800")
	c.Assert(v.MD5Sign("3423742374823748327"), Equals, "a2fecac6295bdbd82989380ecd3166ea")
}
