package xweb

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
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
	hour, min, sec := t.Clock()
	z := int64(hour*60*60 + min*60 + sec)
	r, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		panic(err)
	}
	x := r.Uint64() % 1000000
	return fmt.Sprintf("%s%.5d%.6d", t.Format("20060102"), z, x)
}

func GenUInt64() uint64 {
	id := GenId()
	num, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		panic(err)
	}
	return num
}
