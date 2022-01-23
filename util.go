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
	ic  = uint32(0)
	pid = os.Getpid()
)

func fixInc(num uint32, max int) (uint32, int) {
	org := int(math.Log10(float64(num)) + 1)
	if org > max {
		org = max
	}
	ret := num % uint32(math.Pow10(org))
	if ret <= 0 {
		ret = 1
	}
	bits := math.Log10(float64(ret)) + 1
	return ret, int(bits)
}

func fixPid(pid int, num int) uint {
	p := float64(pid)
	n := int(math.Log10(p)) + 1
	return uint(pid%int(math.Pow10(num)) + int(math.Pow10(num)*float64(n)))
}

func fixNum(pid int, num uint32) uint64 {
	inc, bit := fixInc(num, 5)
	np := fixPid(pid, 7-bit)
	return uint64(float64(np)*math.Pow10(bit)) + uint64(inc)
}

// 创建一个guid
func GenId() string {
	t := time.Now()
	hour, min, sec := t.Clock()
	z := uint32(hour*60*60 + min*60 + sec)
	i := atomic.AddUint32(&ic, 1) % 100000000
	s1 := fmt.Sprintf("%.8d", fixNum(pid, i))
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
