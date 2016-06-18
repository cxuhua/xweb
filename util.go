package xweb

import (
	"time"
)

//获得当天0点时间
func ZeroTime() time.Time {
	now := time.Now()
	fmt := now.Format("2006-01-02")
	zv, err := time.ParseInLocation("2006-01-02", fmt, time.Local)
	if err != nil {
		panic(err)
	}
	return zv
}
