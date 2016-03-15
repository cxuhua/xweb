package xweb

import (
	. "gopkg.in/check.v1"
)

type ContextSuite struct {
}

var _ = Suite(&ContextSuite{})

func (this *ContextSuite) SetUpSuite(c *C) {

}

func (this *ContextSuite) TearDownSuite(c *C) {

}

func (this *ContextSuite) TestHTTPS(c *C) {
	http := NewHTTPClient(true)
	d, err := http.Get("https://git.sportingcool.com", HTTPValues{})
	c.Assert(err, IsNil)
	c.Assert(len(d) > 0, Equals, true)
}

func (this *ContextSuite) TestURL(c *C) {
	v := NewHTTPValues()
	v.Add("dd", 800)
	v.Set("a", "bb")
	v.Set("c", "&3847384^dfdfnjdfnj")
	c.Log(v.RawEncode())
	c.Assert(v.RawEncode(), Equals, "a=bb&c=&3847384^dfdfnjdfnj&dd=800")
	c.Assert(v.MD5Sign("3423742374823748327"), Equals, "a2fecac6295bdbd82989380ecd3166ea")
}
