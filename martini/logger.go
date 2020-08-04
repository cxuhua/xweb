package martini

import (
	"net/http"
	"time"

	"github.com/cxuhua/xweb/logging"
)

// Logger returns a middleware handler that logs the request as it goes in and the response as it goes out.
func Logger() Handler {
	return func(res http.ResponseWriter, req *http.Request, c Context, log *logging.Logger) {
		start := time.Now()
		addr := req.Header.Get("X-Real-IP")
		if addr == "" {
			addr = req.Header.Get("X-Forwarded-For")
			if addr == "" {
				addr = req.RemoteAddr
			}
		}
		rw := res.(ResponseWriter)
		c.Next()
		if Env != Dev {
			return
		}
		log.Infof(
			"%s %s for %s Completed %v %s time=%.3f ms,length=%d\n",
			req.Method,
			req.URL.Path,
			addr,
			rw.Status(),
			http.StatusText(rw.Status()),
			float64(time.Since(start))/float64(time.Millisecond),
			rw.Size(),
		)
	}
}
