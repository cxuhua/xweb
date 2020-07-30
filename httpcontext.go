package xweb

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"runtime/pprof"
	"sort"
	"sync"
	"time"

	"github.com/cxuhua/xweb/logging"
	"github.com/cxuhua/xweb/martini"
)

func checkWriteHeaderCode(code int) {
	if code < 100 || code > 999 {
		panic(fmt.Sprintf("invalid WriteHeader code %v", code))
	}
}

func TimeoutHandler(h http.Handler, dt time.Duration, msg string) http.Handler {
	logger := h.(*HttpContext).Logger()
	return &timeoutHandler{
		logger:  logger,
		handler: h,
		body:    msg,
		dt:      dt,
	}
}

var ErrHandlerTimeout = errors.New("http: Handler timeout")

type timeoutHandler struct {
	logger      *logging.Logger
	handler     http.Handler
	body        string
	dt          time.Duration
	testContext context.Context
}

func (h *timeoutHandler) errorBody() string {
	if h.body != "" {
		return h.body
	}
	return "<html><head><title>Timeout</title></head><body><h1>Timeout</h1></body></html>"
}

func (h *timeoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := h.testContext
	if ctx == nil {
		var cancelCtx context.CancelFunc
		ctx, cancelCtx = context.WithTimeout(r.Context(), h.dt)
		defer cancelCtx()
	}
	r = r.WithContext(ctx)
	done := make(chan struct{})
	tw := &timeoutWriter{
		w: w,
		h: make(http.Header),
	}
	panicChan := make(chan interface{}, 1)
	now := time.Now().UnixNano()
	go func() {
		defer func() {
			if p := recover(); p != nil {
				panicChan <- p
			}
		}()
		h.handler.ServeHTTP(tw, r)
		close(done)
	}()
	select {
	case p := <-panicChan:
		panic(p)
	case <-done:
		tw.mu.Lock()
		defer tw.mu.Unlock()
		dst := w.Header()
		for k, vv := range tw.h {
			dst[k] = vv
		}
		if !tw.wroteHeader {
			tw.code = http.StatusOK
		}
		w.WriteHeader(tw.code)
		_, _ = w.Write(tw.wbuf.Bytes())
	case <-ctx.Done():
		tw.mu.Lock()
		defer tw.mu.Unlock()
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, h.errorBody())
		tw.timedOut = true
		h.logger.Println(r.RequestURI, "do timeout", time.Now().UnixNano()-now, " status=", http.StatusServiceUnavailable)
	}
}

type timeoutWriter struct {
	w    http.ResponseWriter
	h    http.Header
	wbuf bytes.Buffer

	mu          sync.Mutex
	timedOut    bool
	wroteHeader bool
	code        int
}

func (tw *timeoutWriter) Header() http.Header { return tw.h }

func (tw *timeoutWriter) Write(p []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut {
		return 0, ErrHandlerTimeout
	}
	if !tw.wroteHeader {
		tw.writeHeader(http.StatusOK)
	}
	return tw.wbuf.Write(p)
}

func (tw *timeoutWriter) WriteHeader(code int) {
	checkWriteHeaderCode(code)
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut || tw.wroteHeader {
		return
	}
	tw.writeHeader(code)
}

func (tw *timeoutWriter) writeHeader(code int) {
	tw.wroteHeader = true
	tw.code = code
}

//default context

var (
	m            = NewHttpContext()
	LoggerFormat = logging.MustStringFormatter(`%{color}%{time:15:04:05.000} %{shortfile} %{shortfunc} ▶ %{level:.5s} %{id:d}%{color:reset} %{message}`)
	LoggerPrefix = ""
	UserPprof    = flag.Bool("usepprof", false, "write cpu pprof and heap pprof file")
	HttpTimeout  = time.Second * 30
)

func AddExtType(ext string, typ string) {
	_ = mime.AddExtensionType(ext, typ)
}

func InitLogger(w io.Writer) {
	m.InitDefaultLogger(w)
}

func ServeHTTP(res http.ResponseWriter, req *http.Request) {
	m.ServeHTTP(res, req)
}

func SetValidationFunc(name string, vf ValidationFunc) error {
	return m.Validator.SetValidationFunc(name, vf)
}

func Validate(v IArgs) error {
	return m.Validator.Validate(v)
}

func SetEnv(env string) {
	martini.Env = env
}

func Map(v interface{}) {
	m.Map(v)
}

func GetBody(req *http.Request) ([]byte, error) {
	return m.GetBody(req)
}

func MapTo(v interface{}, t interface{}) {
	m.MapTo(v, t)
}

func UseDispatcher(c IDispatcher, in ...martini.Handler) {
	m.UseDispatcher(c, in...)
}

func Use(h martini.Handler) {
	m.Use(h)
}

func UseRender(opts ...RenderOptions) {
	m.UseRender(opts...)
}

func Serve(addr string) error {
	return m.ListenAndServe(addr)
}

func Shutdown() {
	m.Shutdown()
}

func ListenAndServe(addr string) error {
	return m.ListenAndServe(addr)
}

func ServeTLS(addr string, cert, key string) error {
	return m.ListenAndServeTLS(addr, cert, key)
}

func ListenAndServeTLS(addr string, cert, key string) error {
	return m.ListenAndServeTLS(addr, cert, key)
}

func Logger() *logging.Logger {
	return m.GetLogger()
}

type URLS struct {
	Method  string
	Pattern string
	View    string
	Render  string
	Args    IArgs
}
type HttpContext struct {
	martini.ClassicMartini
	Validator      *Validator
	URLS           []URLS
	heapPPROFFiles []string
	cpuPPROFFiles  []string
	http           *http.Server
}

func (this *HttpContext) InitDefaultLogger(w io.Writer) {
	lw := logging.NewLogBackend(w, LoggerPrefix, 0)
	backend := logging.NewBackendFormatter(lw, LoggerFormat)
	logging.SetBackend(backend)
}

func (this *HttpContext) UseRender(opts ...RenderOptions) {
	this.Use(Renderer(opts...))
}

func (this *HttpContext) SetValidationFunc(name string, vf ValidationFunc) error {
	return this.Validator.SetValidationFunc(name, vf)
}

func (this *HttpContext) Validate(v IArgs) error {
	if !v.IsValidate() {
		return nil
	}
	return this.Validator.Validate(v)
}

func (this *HttpContext) Logger() *logging.Logger {
	return this.GetLogger()
}

func (this *HttpContext) startCPUPprof() (*os.File, string) {
	file := fmt.Sprintf("cpu-%s.prof", time.Now().Format("20060102150405"))
	log.Println("create cpu prof ", file)
	cpuFile, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(cpuFile)
	return cpuFile, file
}

func (this *HttpContext) writeHeapPprof() {
	for {
		select {
		case <-time.After(time.Minute * 10):
			file := fmt.Sprintf("heap-%s.prof", time.Now().Format("20060102150405"))
			f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				log.Fatal(err)
			}
			pprof.WriteHeapProfile(f)
			f.Close()
			this.heapPPROFFiles = append(this.heapPPROFFiles, file)
			if len(this.heapPPROFFiles) > 10 {
				os.Remove(this.heapPPROFFiles[0])
				this.heapPPROFFiles = this.heapPPROFFiles[1:]
			}
			log.Println("create heap prof ", file)
		}
	}
}

func (this *HttpContext) writeCPUPprof() {
	cpuFile, file := this.startCPUPprof()
	for {
		select {
		case <-time.After(time.Minute * 30):
			pprof.StopCPUProfile()
			cpuFile.Close()
			this.cpuPPROFFiles = append(this.cpuPPROFFiles, file)
			if len(this.cpuPPROFFiles) > 10 {
				os.Remove(this.cpuPPROFFiles[0])
				this.cpuPPROFFiles = this.cpuPPROFFiles[1:]
			}
			cpuFile, file = this.startCPUPprof()
		}
	}
}
func (this *HttpContext) Shutdown() {
	if this.http == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	_ = this.http.Shutdown(ctx)
}

func (this *HttpContext) ListenAndServe(addr string) error {
	this.PrintURLS()
	this.Logger().Infof("http listening on %s (%s)\n", addr, martini.Env)

	if *UserPprof {
		go this.writeHeapPprof()
		go this.writeCPUPprof()
	}
	this.http = &http.Server{
		Addr:    addr,
		Handler: TimeoutHandler(this, HttpTimeout, "time out"),
	}
	return this.http.ListenAndServe()
}

func (this *HttpContext) ListenAndServeTLS(addr string, cert, key string) error {
	this.PrintURLS()
	this.Logger().Infof("https listening on %s (%s)\n", addr, martini.Env)

	if *UserPprof {
		go this.writeHeapPprof()
		go this.writeCPUPprof()
	}
	this.http = &http.Server{
		Addr:    addr,
		Handler: TimeoutHandler(this, HttpTimeout, "time out"),
	}
	return this.http.ListenAndServeTLS(cert, key)
}

func (this *HttpContext) PrintURLS() {
	log := this.GetLogger()
	sort.Slice(this.URLS, func(i, j int) bool {
		return this.URLS[i].Pattern < this.URLS[j].Pattern
	})
	mc, pc, vc, rc := 0, 0, 0, 0
	for _, u := range this.URLS {
		if len(u.Method) > mc {
			mc = len(u.Method)
		}
		if len(u.Pattern) > pc {
			pc = len(u.Pattern)
		}
		if len(u.View) > vc {
			vc = len(u.View)
		}
		if u.Render == "" {
			u.Render = RenderToString(u.Args.Model().Render())
		}
		if len(u.Render) > rc {
			rc = len(u.Render)
		}
	}
	fs := fmt.Sprintf("+ %%-%ds %%-%ds %%-%ds %%-%ds\n", mc, pc, vc, rc)
	for _, u := range this.URLS {
		log.Infof(fs, u.Method, u.Pattern, u.View, u.Render)
	}
}

func NewHttpContext() *HttpContext {
	h := &HttpContext{}
	r := martini.NewRouter()
	m := martini.New()
	m.Use(martini.Logger())
	m.Use(martini.Recovery())
	m.Use(martini.Static("public"))
	m.MapTo(r, (*martini.Routes)(nil))
	m.Action(r.Handle)
	h.Validator = NewValidator()
	h.URLS = []URLS{}
	h.Martini = m
	h.Router = r
	return h
}
