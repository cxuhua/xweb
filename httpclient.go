package xweb

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

const (
	POST_REMOTE_TIMEOUT = 15
)

type HTTPValues struct {
	url.Values
}

func (this HTTPValues) MD5Sign(key string) string {
	s := this.RawEncode()
	if len(s) > 0 {
		s += "&"
	}
	s += "key=" + key
	return MD5String(s)
}

func (this HTTPValues) RawEncode() string {
	if this.Values == nil {
		return ""
	}
	buf := bytes.Buffer{}
	keys := make([]string, 0, len(this.Values))
	for k := range this.Values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		vs := this.Values[k]
		if len(vs) == 0 {
			continue
		}
		s := k + "="
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(s)
			buf.WriteString(v)
		}
	}
	return buf.String()
}

func (this HTTPValues) AddFormat(key, format string, value interface{}) {
	this.Values.Add(key, fmt.Sprintf(format, value))
}

func (this HTTPValues) Add(key string, value interface{}) {
	this.AddFormat(key, "%v", value)
}

func (this HTTPValues) SetFormat(key, format string, value interface{}) {
	this.Values.Set(key, fmt.Sprintf(format, value))
}

func (this HTTPValues) Set(key string, value interface{}) {
	this.SetFormat(key, "%v", value)
}

func (this HTTPValues) IsEmpty() bool {
	return len(this.Values) == 0
}

func NewHTTPValues() HTTPValues {
	return HTTPValues{Values: url.Values{}}
}

func readResponse(res *http.Response, is200 bool) ([]byte, error) {
	if res.StatusCode != http.StatusOK && is200 {
		return nil, errors.New("http error,status=" + res.Status)
	}
	if res.Body == nil {
		return nil, NoDataError
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	return data, nil
}

type HttpResponse struct {
	*http.Response
}

func (this HttpResponse) Close() {
	this.Response.Body.Close()
}

func (this HttpResponse) ToReader() (io.Reader, error) {
	data, err := readResponse(this.Response, true)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (this HttpResponse) ToString() (string, error) {
	bytes, err := this.ToBytes()
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (this HttpResponse) GetBody() ([]byte, error) {
	return readResponse(this.Response, false)
}

func (this HttpResponse) ToBytes() ([]byte, error) {
	return readResponse(this.Response, true)
}

func (this HttpResponse) ToJson(v interface{}) error {
	data, err := this.ToBytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func (this HttpResponse) ToForm(v interface{}) error {
	data, err := this.ToBytes()
	if err != nil {
		return err
	}
	fv, err := url.ParseQuery(string(data))
	if err != nil {
		return err
	}
	MapFormBindType(v, fv)
	return nil
}

func (this HttpResponse) ToXml(v interface{}) error {
	data, err := this.ToBytes()
	if err != nil {
		return err
	}
	return xml.Unmarshal(data, v)
}

type HTTPClient struct {
	http.Client
	IsSecure bool
	Host     string
	ctx      context.Context
}

var (
	NoDataError = errors.New("http not response data")
)

func (this HTTPClient) GetBytes(path string) (HttpResponse, error) {
	ret := HttpResponse{}
	res, err := this.Client.Get(this.Host + path)
	if err != nil {
		return ret, err
	}
	ret.Response = res
	return ret, nil
}

func HttpForm(url string, q HTTPValues) (HttpResponse, error) {
	ret := HttpResponse{}
	res, err := http.PostForm(url, q.Values)
	ret.Response = res
	return ret, err
}

func HttpPost(url string, bt string, body io.Reader) (HttpResponse, error) {
	ret := HttpResponse{}
	res, err := http.Post(url, bt, body)
	ret.Response = res
	return ret, err
}

func HttpGet(url string) (HttpResponse, error) {
	ret := HttpResponse{}
	res, err := http.Get(url)
	ret.Response = res
	return ret, err
}

func (this HTTPClient) Get(path string, q HTTPValues) (HttpResponse, error) {
	ret := HttpResponse{}
	url, err := url.Parse(path)
	if err != nil {
		return ret, err
	}
	qv := url.Query()
	for kv, vv := range q.Values {
		for _, v := range vv {
			qv.Add(kv, v)
		}
	}
	res, err := this.Client.Get(this.Host + url.Path + "?" + qv.Encode())
	if err != nil {
		return ret, err
	}
	ret.Response = res
	return ret, nil
}

func (this HTTPClient) PostBytes(path string, ct string, data []byte) (HttpResponse, error) {
	return this.Post(path, ct, bytes.NewReader(data))
}

func (this HTTPClient) Post(path string, ct string, body io.Reader) (HttpResponse, error) {
	ret := HttpResponse{}
	res, err := this.Client.Post(this.Host+path, ct, body)
	if err != nil {
		return ret, err
	}
	ret.Response = res
	return ret, nil
}

func (this HTTPClient) Form(path string, v HTTPValues) (HttpResponse, error) {
	ret := HttpResponse{}
	res, err := this.Client.PostForm(this.Host+path, v.Values)
	if err != nil {
		return ret, err
	}
	ret.Response = res
	return ret, nil
}

//自动识别是否启用context
func (this HTTPClient) request(method, url string, body io.Reader) (*http.Request, error) {
	if this.ctx != nil {
		return http.NewRequestWithContext(this.ctx, method, url, body)
	}
	return http.NewRequest(method, url, body)
}

func (this HTTPClient) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	return this.request(method, this.Host+path, body)
}

func (this HTTPClient) NewGet(path string, qs ...HTTPValues) (*http.Request, error) {
	url, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	qv := url.Query()
	for _, q := range qs {
		for kv, vv := range q.Values {
			for _, v := range vv {
				qv.Add(kv, v)
			}
		}
	}
	return this.request(http.MethodGet, this.Host+url.Path+"?"+qv.Encode(), nil)
}

func (this HTTPClient) NewForm(path string, v HTTPValues) (*http.Request, error) {
	body := strings.NewReader(v.Encode())
	req, err := this.request(http.MethodPost, this.Host+path, body)
	if err != nil {
		return req, nil
	}
	req.Header.Set(ContentType, ContentURLEncoded)
	return req, nil
}

func dialTimeout(ctx context.Context,network, addr string) (net.Conn, error) {
	d := net.Dialer{}
	conn, err := d.DialContext(ctx,network, addr)
	if err != nil {
		return conn, err
	}
	tcp := conn.(*net.TCPConn)
	err = tcp.SetKeepAlive(false)
	return tcp, err
}

func (this HTTPClient) NewPost(path string, bt string, body io.Reader) (*http.Request, error) {
	req, err := this.request(http.MethodPost, this.Host+path, body)
	if err != nil {
		return req, nil
	}
	if bt != "" {
		req.Header.Set(ContentType, bt)
	}
	return req, nil
}

func (this HTTPClient) Do(req *http.Request) (HttpResponse, error) {
	ret := HttpResponse{}
	res, err := this.Client.Do(req)
	if err != nil {
		return ret, err
	}
	ret.Response = res
	return ret, nil
}

//not verify config
func TLSSkipVerifyConfig() *tls.Config {
	return &tls.Config{InsecureSkipVerify: true}
}

func MustLoadTLSConfig(ca, crt, key string) *tls.Config {
	if ca == "" {
		panic(errors.New("ca data miss"))
	}
	if crt == "" {
		panic(errors.New("crt data miss"))
	}
	if key == "" {
		panic(errors.New("key data miss"))
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM([]byte(ca)) {
		panic("Failed appending certs")
	}
	cert, err := tls.X509KeyPair([]byte(crt), []byte(key))
	if err != nil {
		panic(err)
	}
	conf := &tls.Config{}
	conf.Certificates = []tls.Certificate{cert}
	conf.RootCAs = pool
	return conf
}

func MustLoadTLSFile(crtFile, keyFile string) *tls.Config {
	if crtFile == "" {
		panic(errors.New("crtFile miss"))
	}
	if keyFile == "" {
		panic(errors.New("keyFile miss"))
	}
	cert, err := tls.LoadX509KeyPair(crtFile, keyFile)
	if err != nil {
		panic(err)
	}
	conf := &tls.Config{}
	conf.Certificates = []tls.Certificate{cert}
	return conf
}

func MustLoadTLSFileConfig(rootFile, crtFile, keyFile string) *tls.Config {
	if rootFile == "" {
		panic(errors.New("rootFile miss"))
	}
	if crtFile == "" {
		panic(errors.New("crtFile miss"))
	}
	if keyFile == "" {
		panic(errors.New("keyFile miss"))
	}
	pem, err := ioutil.ReadFile(rootFile)
	if err != nil {
		panic(err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		panic("Failed appending certs")
	}
	cert, err := tls.LoadX509KeyPair(crtFile, keyFile)
	if err != nil {
		panic(err)
	}
	conf := &tls.Config{}
	conf.Certificates = []tls.Certificate{cert}
	conf.RootCAs = pool
	return conf
}

func NewHTTPClientWithContext(ctx context.Context, host string, confs ...*tls.Config) HTTPClient {
	host = strings.ToLower(host)
	ret := HTTPClient{}
	ret.Host = host
	ret.IsSecure = strings.HasPrefix(host, "https")
	tr := &http.Transport{
		DialContext:              dialTimeout,
		DisableKeepAlives: true,
	}
	if len(confs) > 0 {
		tr.TLSClientConfig = confs[0]
	} else if ret.IsSecure {
		tr.TLSClientConfig = TLSSkipVerifyConfig()
	}
	ret.Client = http.Client{Transport: tr}
	ret.ctx = ctx
	return ret
}

func NewHTTPClient(host string, confs ...*tls.Config) HTTPClient {
	return NewHTTPClientWithContext(nil, host, confs...)
}
