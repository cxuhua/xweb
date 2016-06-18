package xweb

import (
	"time"
)

//get 00:00:00 unix time
func ZeroTime() time.Time {
	now := time.Now()
	hour, min, sec := now.Clock()
	t := now.Unix() - int64(hour*60*60+min*60+sec)
	return time.Unix(t, 0)
}
