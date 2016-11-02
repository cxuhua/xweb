package xweb

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strings"
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

func readResponse(res *http.Response) ([]byte, error) {
	if res.Body == nil {
		return nil, NoDataError
	}
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return data, errors.New("http error,status=" + res.Status)
	}
	return data, err
}

type HttpResponse struct {
	*http.Response
}

func (this HttpResponse) ToReader() (io.Reader, error) {
	data, err := readResponse(this.Response)
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

func (this HttpResponse) ToBytes() ([]byte, error) {
	return readResponse(this.Response)
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
	MapFormBindType(v, fv, nil, nil, nil)
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

func (this HTTPClient) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, this.Host+path, body)
}

func (this HTTPClient) NewGet(path string, q HTTPValues) (*http.Request, error) {
	url, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	qv := url.Query()
	for kv, vv := range q.Values {
		for _, v := range vv {
			qv.Add(kv, v)
		}
	}
	return http.NewRequest(http.MethodGet, this.Host+url.Path+"?"+qv.Encode(), nil)
}

func (this HTTPClient) NewForm(path string, v HTTPValues) (*http.Request, error) {
	body := strings.NewReader(v.Encode())
	req, err := http.NewRequest(http.MethodPost, this.Host+path, body)
	if err != nil {
		return req, nil
	}
	req.Header.Set(ContentType, ContentURLEncoded)
	return req, nil
}

func (this HTTPClient) NewPost(path string, bt string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodPost, this.Host+path, body)
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

func NewHTTPClient(host string, confs ...*tls.Config) HTTPClient {
	host = strings.ToLower(host)
	ret := HTTPClient{}
	ret.Host = host
	ret.IsSecure = strings.HasPrefix(host, "https")
	tr := &http.Transport{}
	if len(confs) > 0 {
		tr.TLSClientConfig = confs[0]
	} else if ret.IsSecure {
		tr.TLSClientConfig = TLSSkipVerifyConfig()
	}
	ret.Client = http.Client{Transport: tr}
	return ret
}
