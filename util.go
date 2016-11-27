package xweb

import (
	"crypto/rand"
	"fmt"
	"github.com/cxuhua/xweb/now"
	"math/big"
	"strconv"
	"time"
)

// 获取当前时间
func TimeNow() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// 获取一天的开始时间
func ZeroTime() time.Time {
	return now.BeginningOfDay()
}

// 随机获取0-9数字，l为长度
func RandNumber(l int) string {
	max := big.NewInt(10)
	ret := ""
	for i := 0; i < l; i++ {
		r, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(err)
		}
		ret += fmt.Sprintf("%d", r.Uint64())
	}
	return ret
}

// 创建一个guid
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
