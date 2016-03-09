package xweb

import (
	. "gopkg.in/check.v1"
)

type CryptoSuite struct {
	S string
}

var _ = Suite(&CryptoSuite{})

func (this *CryptoSuite) SetUpSuite(c *C) {
	TokenKey = []byte("Hjdfd(&#&*&")
	this.S = "12232938293293**&S*^DKSJBDKJSBDKJBSD"
}

func (this *CryptoSuite) TearDownSuite(c *C) {

}

func (this *CryptoSuite) TestAesCrypto(c *C) {
	block, err := NewAESChpher([]byte("skdjfs9u3294jsfsdofuosdufos"))
	c.Assert(err, IsNil)
	d1, err := AesEncrypt(block, []byte(this.S))
	c.Assert(err, IsNil)
	d2, err := AesDecrypt(block, d1)
	c.Assert(err, IsNil)
	c.Assert(string(d2), Equals, this.S)
}

func (this *CryptoSuite) TestTokenCrypto(c *C) {
	d1, err := TokenEncrypt(this.S)
	c.Assert(err, IsNil)
	d2, err := TokenDecrypt(d1)
	c.Assert(err, IsNil)
	c.Assert(d2, Equals, this.S)
}
