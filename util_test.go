package xweb

import (
	// "crypto/rand"
	// "encoding/binary"
	. "gopkg.in/check.v1"
	"log"
	"time"
)

type UtilSuite struct {
}

var _ = Suite(&UtilSuite{})

func (this *UtilSuite) SetUpSuite(c *C) {

}

func (this *UtilSuite) TearDownSuite(c *C) {

}

func (this *UtilSuite) TestZeroTime(c *C) {
	v1 := ZeroTime().Unix()
	time.Sleep(time.Second)
	v2 := ZeroTime().Unix()
	c.Assert(v1, Equals, v2)
	ns := time.Now().Format("2006-01-02")
	c.Assert(ns, Equals, ZeroTime().Format("2006-01-02"))
}

func (this *UtilSuite) TestGenId(c *C) {
	log.Println(GenId())
}
