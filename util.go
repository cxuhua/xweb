package xweb

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/cxuhua/xweb/now"
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

var (
	ic = uint32(0)
)

func fixNum(pid int, num uint32) uint64 {
	p := float64(pid)
	n := int(math.Log10(100000000)) - int(math.Log10(p))
	return uint64(math.Pow10(n-1)*p) + uint64(num)
}

// 创建一个guid
func GenId() string {
	t := time.Now()
	hour, min, sec := t.Clock()
	z := uint64(hour*60*60 + min*60 + sec)
	i := atomic.AddUint32(&ic, 1) % 1000000000
	p := os.Getpid()
	s1 := fmt.Sprintf("%.8d", fixNum(p, i))
	s2 := fmt.Sprintf("%.5d", z)
	return t.Format("060102") + s2 + s1
}

func GenUInt64() uint64 {
	id := GenId()
	num, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		panic(err)
	}
	return num
}
