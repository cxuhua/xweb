package xweb

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/cxuhua/lzma"
	"github.com/cxuhua/xweb/logging"
	"github.com/cxuhua/xweb/martini"
)

type IModel interface {
	Finished()   //处理完成
	Render() int //输出模式
	GetHeader() http.Header
}

type xModel struct {
	IModel
	Header http.Header
}

func (this *xModel) InitHeader() {
	this.Header = http.Header{}
}

func (this *xModel) GetHeader() http.Header {
	return this.Header
}

func (this *xModel) Finished() {

}

func (this *xModel) Render() int {
	return HTML_RENDER
}

type RedirectModel struct {
	xModel
	Url string
}

func (this *RedirectModel) Render() int {
	return REDIRECT_RENDER
}

//html model
type HtmlModel struct {
	xModel
}

func (this *HtmlModel) Finished() {

}

func (this *HtmlModel) Render() int {
	return HTML_RENDER
}

//内存模版输出

type TempModel struct {
	xModel
	Template string
	Model    interface{}
}

func (this *TempModel) Finished() {

}

func (this *TempModel) Render() int {
	return TEMP_RENDER
}

//http文件下载
type IHttpFile interface {
	io.Reader
	io.Seeker
	io.Closer
}

//文件输出
type FileModel struct {
	xModel
	Name    string    //名称
	ModTime time.Time //修改时间
	File    IHttpFile //读取接口
}

func (this *FileModel) Render() int {
	return FILE_RENDER
}

func (this *FileModel) Finished() {
	if this.File != nil {
		this.File.Close()
	}
}

func NewFileModel() *FileModel {
	m := &FileModel{ModTime: time.Now()}
	m.InitHeader()
	return m
}

//脚本输出
type ScriptModel struct {
	xModel
	Script string
}

func (this *ScriptModel) Render() int {
	return SCRIPT_RENDER
}

func (this *ScriptModel) Finished() {

}

func NewScriptModel() *ScriptModel {
	m := &ScriptModel{}
	m.InitHeader()
	return m
}

//用于TEXT输出
type StringModel struct {
	xModel
	Text string
}

func (this *StringModel) Finished() {

}

func (this *StringModel) Render() int {
	return TEXT_RENDER
}

func NewStringModel() *StringModel {
	m := &StringModel{}
	m.InitHeader()
	return m
}

//content model
type ContentModel struct {
	xModel
	Key  string
	Type string
	Data []byte
}

func (this *ContentModel) Finished() {

}

func (this *ContentModel) Render() int {
	return CONTENT_RENDER
}

func NewContentModel(b []byte, k string, t string) *ContentModel {
	m := &ContentModel{Data: b, Key: k, Type: t}
	m.InitHeader()
	return m
}

//data render model
type BinaryModel struct {
	xModel
	Data []byte
}

func (this *BinaryModel) Finished() {

}

func (this *BinaryModel) Render() int {
	return DATA_RENDER
}

func NewBinaryModel() *BinaryModel {
	m := &BinaryModel{}
	m.InitHeader()
	return m
}

//json render model
type JSONModel struct {
	xModel
}

func (this *JSONModel) Finished() {

}

func (this *JSONModel) Render() int {
	return JSON_RENDER
}

//xml render model
type XMLModel struct {
	xModel
}

func (this *XMLModel) Finished() {

}

func (this *XMLModel) Render() int {
	return XML_RENDER
}

//渲染模型
type HTTPModel struct {
	JSONModel `bson:"-" json:"-" xml:"-"`
	Code      int    `bson:"code" json:"code" xml:"code"`
	Error     string `bson:"error,omitempty" json:"error,omitempty" xml:"error,omitempty"`
	File      string `bson:"file,omitempty" json:"file,omitempty" xml:"file,omitempty"`
}

func (this *HTTPModel) SetCode(code int) {
	this.Code = code
	if this.Code == 0 {
		return
	}
	if martini.Env == martini.Dev {
		_, file, line, _ := runtime.Caller(1)
		this.File = fmt.Sprintf("%s:%d", file, line)
	}
}

func (this *HTTPModel) SetFormat(code int, format string, args ...interface{}) {
	this.Code = code
	this.Error = fmt.Sprintf(format, args...)
	if martini.Env == martini.Dev {
		_, file, line, _ := runtime.Caller(1)
		this.File = fmt.Sprintf("%s:%d", file, line)
	}
}

func (this *HTTPModel) SetError(code int, err interface{}) {
	this.Code = code
	if err != nil {
		switch err.(type) {
		case string:
			this.Error = err.(string)
		case error:
			this.Error = err.(error).Error()
		default:
			this.Error = fmt.Sprintf("%v", err)
		}
	}
	if martini.Env == martini.Dev {
		_, file, line, _ := runtime.Caller(1)
		this.File = fmt.Sprintf("%s:%d", file, line)
	}
}

func (this *HTTPModel) Finished() {
	//
}

func NewHTTPError(code int, err string) *HTTPModel {
	m := &HTTPModel{Code: code, Error: err}
	m.InitHeader()
	return m
}

func NewHTTPSuccess() *HTTPModel {
	m := &HTTPModel{Code: 0}
	m.InitHeader()
	return m
}

//数据参数校验器是吧输出

type ValidateError struct {
	Field string `xml:"field,attr" json:"field"`
	Error string `xml:",chardata" json:"error"`
}

type ValidateModel struct {
	JSONModel `json:"-" xml:"-" form:"-" url:"-"`
	XMLName   struct{}        `xml:"xml" json:"-" form:"-" url:"-"`
	Code      int             `xml:"code" json:"code"`
	Fileds    []ValidateError `xml:"fileds>item" json:"fileds"`
	Error     string          `xml:"error" json:"error"`
}

func (this *ValidateModel) ToJSON() string {
	d, err := json.Marshal(this)
	if err != nil {
		return ""
	}
	return string(d)
}

func (this *ValidateModel) ToXML() string {
	d, err := xml.Marshal(this)
	if err != nil {
		return ""
	}
	return string(d)
}

func (this *ValidateModel) ToTEXT() string {
	s := []string{}
	for _, i := range this.Fileds {
		s = append(s, i.Field)
	}
	return strings.Join(s, ",")
}

func (this *ValidateModel) Init(e error) {
	this.Error = "args error,look fileds"
	this.Fileds = []ValidateError{}
	this.Code = ValidateErrorCode
	err, ok := e.(ErrorMap)
	if !ok {
		return
	}
	for k, v := range err {
		e := ValidateError{Field: k, Error: v.Error()}
		this.Fileds = append(this.Fileds, e)
	}
}

//NewValidateModel 校验器
func NewValidateModel(err error) *ValidateModel {
	m := &ValidateModel{}
	m.InitHeader()
	m.Init(err)
	return m
}

//ILocker 缓存锁，需要基于分布式锁实现
type ILocker interface {
	//Release 释放锁
	Release()
	//TTL 锁超时时间 返回0表示锁已经释放
	TTL() (time.Duration, error)
	//Refresh 更新锁超时时间
	Refresh(ttl time.Duration) error
}

//ICache 缓存接口
type ICache interface {
	//获取key超时时间
	//-1 未设置过期时间
	//-2 key不存在
	TTL(k string) (time.Duration, error)
	//Set 设置值
	Set(k string, v interface{}, exp ...time.Duration) error
	//Get 获取值
	Get(k string, v interface{}) error
	//Del 删除值
	Del(k ...string) (int64, error)
	//Locker 创建缓存锁,锁存在将返回错误
	//如果创建成功，锁将在ttl时间后过期释放
	Locker(key string, ttl time.Duration, meta ...string) (ILocker, error)
}

var (
	//CacheOn 是否开启缓存
	CacheOn = true
	//MinZipSize 最小压缩大小
	MinZipSize = 2048
)

//IsCacheOn 是否开启缓存
func IsCacheOn() bool {
	return CacheOn
}

//CacheParams 缓存参数
type CacheParams struct {
	//缓存实现
	Imp ICache
	//超时时间
	TTL time.Duration
	//缓存key
	Key string
	//延迟超时时间，如果设置ttl和dtl>0，当key得时间少于dtl时就算过期
	DTL time.Duration
}

//NewCacheParams 传教缓存参数
func NewCacheParams(imp ICache, ttl time.Duration, dtl time.Duration, kfmt string, vs ...interface{}) *CacheParams {
	return &CacheParams{
		Imp: imp,
		TTL: ttl,
		Key: fmt.Sprintf(kfmt, vs...),
		DTL: dtl,
	}
}

//Remove 删除缓存
func (cp *CacheParams) Remove() {
	cp.Imp.Del(cp.Key)
}

//DoXML 缓存为xml
func (cp *CacheParams) DoXML(fn func() (interface{}, error), vp interface{}, ttl time.Duration, try ...int) (bool, error) {
	if !IsCacheOn() {
		return false, fmt.Errorf("cache disabled")
	}
	//测试是否从缓存获取数据
	lck, bb, fbc, err := cp.prepare(ttl, try...)
	//如果有缓存数据
	if fbc {
		err = xml.Unmarshal(bb, vp)
		return true, err
	}
	//错误了
	if err != nil {
		return false, err
	}
	if lck != nil {
		defer lck.Release()
	}
	//没有执行处理函数返回数据
	vptr, err := fn()
	if err != nil {
		return false, err
	}
	//序列化保存
	bb, err = xml.Marshal(vptr)
	if err != nil {
		return false, err
	}
	//有数据就保存并且返回
	err = cp.SetBytes(bb)
	//反序列化到vp返回
	if err == nil {
		err = xml.Unmarshal(bb, vp)
	}
	return false, err
}

//DoJSON 缓存为json
func (cp *CacheParams) DoJSON(fn func() (interface{}, error), vp interface{}, ttl time.Duration, try ...int) (bool, error) {
	if !IsCacheOn() {
		return false, fmt.Errorf("cache disabled")
	}
	//测试是否从缓存获取数据
	lck, bb, fbc, err := cp.prepare(ttl, try...)
	//如果有缓存数据
	if fbc {
		err = json.Unmarshal(bb, vp)
		return true, err
	}
	//错误了
	if err != nil {
		return false, err
	}
	if lck != nil {
		defer lck.Release()
	}
	//没有执行处理函数返回数据
	vptr, err := fn()
	if err != nil {
		return false, err
	}
	//序列化保存
	bb, err = json.Marshal(vptr)
	if err != nil {
		return false, err
	}
	//有数据就保存并且返回
	err = cp.SetBytes(bb)
	//反序列化到vp返回
	if err == nil {
		err = json.Unmarshal(bb, vp)
	}
	return false, err
}

//PTP 获取尝试次数和延迟时间
func PTP(try ...int) (int, time.Duration) {
	//默认3 次每次100毫秒
	tc := 3
	tv := time.Millisecond * 100
	if len(try) > 0 {
		tc = try[0]
	}
	if len(try) > 1 {
		tv = time.Millisecond * time.Duration(try[1])
	}
	return tc, tv
}

//预处理数据
func (cp *CacheParams) prepare(ttl time.Duration, try ...int) (ILocker, []byte, bool, error) {
	//从缓存获取数据
	bb, err := cp.GetBytes()
	//如果有数据并且设置dtl，数据可能要过期
	hasbb := err == nil
	//如果有并且没有过期就直接返回
	if hasbb && !cp.IsExpire() {
		return nil, bb, true, nil
	}
	//如果不启用锁并且没有数据
	if ttl == 0 && !hasbb {
		return nil, nil, false, nil
	}
	//加锁确保后续fn不会重复被执行
	lck, err := cp.Imp.Locker(cp.Key, ttl)
	//锁失败并且有旧数据返回旧数据
	if err != nil && hasbb {
		return nil, bb, true, nil
	}
	//尝试再次获取数据和锁
	for tc, tv := PTP(try...); err != nil && tc > 0; tc-- {
		//休眠后尝试
		time.Sleep(tv)
		//尝试期间如果有缓存数据
		bb, err = cp.GetBytes()
		if err == nil {
			return nil, bb, true, nil
		}
		lck, err = cp.Imp.Locker(cp.Key, ttl)
	}
	//尝试多次未获取锁失败返回错误
	if err != nil {
		return nil, nil, false, err
	}
	return lck, nil, false, nil
}

//DoBytes 缓存fn返回的二进制数据
//返回参数2为true表示来自缓存
//ttl 锁超时时间,try锁尝试次数和延迟时间(毫秒)
func (cp *CacheParams) DoBytes(fn func() ([]byte, error), ttl time.Duration, try ...int) ([]byte, bool, error) {
	if !IsCacheOn() {
		return nil, false, fmt.Errorf("cache disabled")
	}
	//测试是否从缓存获取数据
	lck, bb, fbc, err := cp.prepare(ttl, try...)
	//如果有缓存数据
	if fbc {
		return bb, true, nil
	}
	//错误了
	if err != nil {
		return nil, false, err
	}
	if lck != nil {
		defer lck.Release()
	}
	//没有执行处理函数返回数据
	bb, err = fn()
	if err != nil {
		return nil, false, err
	}
	//有数据就保存并且返回
	err = cp.SetBytes(bb)
	return bb, false, err
}

//GetBytes 获取字符串类型
func (cp *CacheParams) GetBytes() ([]byte, error) {
	var b []byte
	err := cp.Imp.Get(cp.Key, &b)
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return nil, fmt.Errorf("empty content")
	}
	//非SetBytes设置的
	if b[0] != 0 && b[0] != 1 {
		return b, nil
	}
	if b[0] == 0 {
		return b[1:], nil
	}
	v, err := lzma.Uncompress(b[1:])
	if err != nil {
		return nil, err
	}
	return v, nil
}

//IsExpire 是否过期
//返回 true 表示已经过期
func (cp *CacheParams) IsExpire() bool {
	//如果未设置延迟过期时间返回未过期
	if cp.DTL == 0 {
		return false
	}
	//dv单位毫秒
	dv, err := cp.Imp.TTL(cp.Key)
	//错误当作过期
	if err != nil {
		return true
	}
	//未设置过期时间
	if dv == -1 {
		return false
	}
	//不存在
	if dv == -2 {
		return true
	}
	//如果设置了延迟超时
	if dv < cp.DTL {
		return true
	}
	return false
}

//SetBytes 保存字符串,第一字节存放是否被压缩
func (cp *CacheParams) SetBytes(sb []byte) error {
	var vb []byte
	if len(sb) > MinZipSize {
		zb, err := lzma.Compress(sb)
		if err != nil {
			return err
		}
		vb = make([]byte, len(zb)+1)
		vb[0] = 1
		copy(vb[1:], zb)
	} else {
		vb = make([]byte, len(sb)+1)
		vb[0] = 0
		copy(vb[1:], sb)
	}
	return cp.Imp.Set(cp.Key, vb, cp.TTL+cp.DTL)
}

//IMVC mvc控制接口
type IMVC interface {
	SetView(string)
	SetTemplate(string)
	SetViewModel(string, IModel)
	SetModel(IModel)
	SetRender(int)
	SetStatus(int)

	Redirect(string)

	//设置cookie
	SetCookie(cookie *http.Cookie)

	//context
	Map(v interface{})
	MapTo(v interface{}, t interface{})
	Next()
	SkipNext()
	// skip all handler, and run render
	SkipAll()
	// skip count
	Skip(c int)
	SkipRender(bool)
	//render content
	RunRender()
	//logger
	Logger() *logging.Logger
	//http request
	Cookie(name string) (*http.Cookie, error)
	Request() *http.Request
	RemoteAddr() string
	URL() *url.URL
	Header() http.Header
	Method() string
	Host() string
	//error put
	Error(string, ...interface{})
}

//xmvc 默认mvc控制器
type xmvc struct {
	IMVC
	status   int
	view     string
	render   int
	model    IModel
	cookies  []*http.Cookie
	req      *http.Request
	rev      Render
	ctx      martini.Context
	log      *logging.Logger
	rw       martini.ResponseWriter
	isrender bool
}

//error put
func (this *xmvc) Error(format string, args ...interface{}) {
	m := NewStringModel()
	m.Text = fmt.Sprintf(format, args...)
	this.SetStatus(http.StatusInternalServerError)
	this.SetModel(m)
}

func (this *xmvc) Method() string {
	return this.req.Method
}

func (this *xmvc) Host() string {
	return this.req.Host
}

func (this *xmvc) Header() http.Header {
	return this.req.Header
}

func (this *xmvc) Cookie(name string) (*http.Cookie, error) {
	return this.req.Cookie(name)
}

func (this *xmvc) URL() *url.URL {
	return this.req.URL
}

func (this *xmvc) RemoteAddr() string {
	return GetRemoteAddr(this.req)
}

func (this *xmvc) Render() Render {
	return this.rev
}

func (this *xmvc) Request() *http.Request {
	return this.req
}

func (this *xmvc) Logger() *logging.Logger {
	return this.log
}

// 跳过所有中间件并执行默认render
func (this *xmvc) SkipAll() {
	this.ctx.SkipAll()
}

// skip count
func (this *xmvc) Skip(c int) {
	this.ctx.Skip(c)
}

func (this *xmvc) SkipNext() {
	this.ctx.SkipNext()
}

func (this *xmvc) SkipRender(v bool) {
	this.isrender = !v
}

func (this *xmvc) merageHeaderAndCookie() {
	for ik, iv := range this.model.GetHeader() {
		for _, vv := range iv {
			this.rev.Header().Add(ik, vv)
		}
	}
	for _, cv := range this.cookies {
		this.rev.SetCookie(cv)
	}
}

// / -> index
// /goods/ -> goods/index.tmpl
// /goods/list -> goods/list.tmpl
// /goods/list.html -> goods/list.html.tmpl
func (this *xmvc) template(url *url.URL) string {
	path := url.Path
	if path == "" {
		return "index"
	}
	l := len(path)
	if path[l-1] == '/' {
		return path[1:] + "index"
	}
	return path[1:]
}

func (this *xmvc) RunRender() {
	//
	if this.model != nil {
		defer this.model.Finished()
	}
	if !this.isrender {
		return
	}
	//合并http 头
	this.merageHeaderAndCookie()
	//如果未设置渲染方式从model获取
	if this.render == NONE_RENDER {
		this.render = this.model.Render()
	}
	//执行不同类型的渲染
	switch this.render {
	case CONTENT_RENDER:
		v, b := this.model.(*ContentModel)
		if !b {
			panic("RENDER Model error:must set ContentModel")
		}
		this.rev.Header().Set("X-Cache-Key", v.Key)
		this.rev.Header().Set(ContentLength, fmt.Sprintf("%d", len(v.Data)))
		this.rev.Header().Set(ContentType, v.Type)
		this.rev.Data(this.status, v.Data)
	case HTML_RENDER:
		if this.view == "" {
			this.view = this.template(this.req.URL)
		}
		this.rev.HTML(this.status, this.view, this.model)
	// json渲染输出
	case JSON_RENDER:
		this.rev.JSON(this.status, this.model)
	// xml渲染输出
	case XML_RENDER:
		this.rev.XML(this.status, this.model)
	// 脚本渲染输出
	case SCRIPT_RENDER:
		v, b := this.model.(*ScriptModel)
		if !b {
			panic("RENDER Model error:must set ScriptModel")
		}
		this.rev.Header().Set(ContentType, ContentHTML)
		this.rev.Text(this.status, v.Script)
	// 文本渲染输出
	case TEXT_RENDER:
		v, b := this.model.(*StringModel)
		if !b {
			panic("RENDER Model error:must set StringModel")
		}
		this.rev.Text(this.status, v.Text)
	// 二进制渲染输出
	case DATA_RENDER:
		v, b := this.model.(*BinaryModel)
		if !b {
			panic("RENDER Model error:must set BinaryModel")
		}
		this.rev.Data(this.status, v.Data)
	// 文件下载
	case FILE_RENDER:
		v, b := this.model.(*FileModel)
		if !b {
			panic("RENDER Model error:must set FileModel")
		}
		this.rev.File(v.Name, v.ModTime, v.File)
	// 模版内容+数据渲染输出
	case TEMP_RENDER:
		v, b := this.model.(*TempModel)
		if !b {
			panic("RENDER Model error:must set TempModel")
		}
		this.rev.TEMP(this.status, v.Template, v.Model)
	// 重定向
	case REDIRECT_RENDER:
		v, b := this.model.(*RedirectModel)
		if !b {
			panic("RENDER Model error:must set RedirectModel")
		}
		this.rev.Redirect(v.Url)
	default:
		panic(errors.New(RenderToString(this.render) + " not process"))
	}
}

func (this *xmvc) Map(v interface{}) {
	this.ctx.Map(v)
}
func (this *xmvc) MapTo(v interface{}, t interface{}) {
	this.ctx.MapTo(v, t)
}

func (this *xmvc) Next() {
	this.ctx.Next()
}

func (this *xmvc) SetCookie(cookie *http.Cookie) {
	this.cookies = append(this.cookies, cookie)
}

func (this *xmvc) Redirect(url string) {
	m := &RedirectModel{Url: url}
	m.InitHeader()
	this.SetModel(m)
	this.render = REDIRECT_RENDER
}

func (this *xmvc) String() string {
	return fmt.Sprintf("Status:%d,View:%s,Render:%s,Model:%v,IsRender:%v", this.status, this.view, RenderToString(this.render), reflect.TypeOf(this.model).Elem(), this.isrender)
}

func (this *xmvc) SetView(v string) {
	this.view = v
}

func (this *xmvc) SetModel(v IModel) {
	this.model = v
}

func (this *xmvc) SetTemplate(v string) {
	this.view = v
	this.render = HTML_RENDER
}

func (this *xmvc) SetViewModel(v string, m IModel) {
	this.view = v
	this.model = m
	this.render = HTML_RENDER
}

func (this *xmvc) SetRender(v int) {
	this.render = v
}

func (this *xmvc) SetStatus(v int) {
	this.status = v
}
