package martini

import (
	"github.com/cxuhua/xweb/logging"
	"net/http"
	"time"
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
		if Env == Dev {
			log.Infof("%s %s for %s Completed %v %s in %v,length:%d\n", req.Method, req.URL.Path, addr,rw.Status(), http.StatusText(rw.Status()), time.Since(start), rw.Size())
		}
	}
}
