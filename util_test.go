package xweb

import (
	"fmt"
	. "gopkg.in/check.v1"
	"log"
	"strconv"
	"time"
)

type UtilSuite struct {
}

var _ = Suite(&UtilSuite{})

func (this *UtilSuite) SetUpSuite(c *C) {

}

func (this *UtilSuite) TearDownSuite(c *C) {

}

func (this *UtilSuite) TestRandNumber(c *C) {
	log.Println(RandNumber(4))
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
	ids := map[string]bool{}
	for i := 0; i < 10000; i++ {
		v := GenId()
		c.Assert(ids[v], Equals, false)
		ids[v] = true
	}
	id := GenId()
	num, err := strconv.ParseUint(id, 10, 64)
	c.Log(id, num)
	c.Assert(err, Equals, nil)
	s := fmt.Sprintf("%v", num)
	c.Assert(id, Equals, s)
}
