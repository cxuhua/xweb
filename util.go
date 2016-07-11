package xweb

import (
	"fmt"
	"math/rand"
	"time"
)

//get 00:00:00 unix time
func ZeroTime() time.Time {
	now := time.Now()
	hour, min, sec := now.Clock()
	t := now.Unix() - int64(hour*60*60+min*60+sec)
	return time.Unix(t, 0)
}

//auto uuid
func GenId() string {
	t := time.Now()
	rand.Seed(t.UnixNano())
	hour, min, sec := t.Clock()
	z := int64(hour*60*60 + min*60 + sec)
	x := rand.Int() % 100000
	v := t.Format("20060102")
	return fmt.Sprintf("%s%.5d%.5d", v, z, x)
}
