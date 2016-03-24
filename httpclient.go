package xweb

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
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
	var buf bytes.Buffer
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

func (this HTTPValues) Add(key string, value interface{}) {
	this.Values.Add(key, fmt.Sprintf("%v", value))
}

func (this HTTPValues) Set(key string, value interface{}) {
	this.Values.Set(key, fmt.Sprintf("%v", value))
}

func (this HTTPValues) IsEmpty() bool {
	return len(this.Values) == 0
}

func NewHTTPValues() HTTPValues {
	return HTTPValues{Values: url.Values{}}
}

type HTTPClient struct {
	http.Client
	IsSecure bool
	Host     string
}

var (
	NoDataError = errors.New("http not response data")
)

func (this HTTPClient) ReadResponse(res *http.Response) ([]byte, error) {
	if res.Body == nil {
		return nil, NoDataError
	}
	defer res.Body.Close()
	return ioutil.ReadAll(res.Body)
}

func (this HTTPClient) Get(path string, q HTTPValues) ([]byte, error) {
	if !q.IsEmpty() {
		path = path + "?" + q.Encode()
	}
	if res, err := this.Client.Get(this.Host + path); err != nil {
		return nil, err
	} else {
		return this.ReadResponse(res)
	}
}

func (this HTTPClient) Post(path string, bt string, body io.Reader) ([]byte, error) {
	if res, err := this.Client.Post(this.Host+path, bt, body); err != nil {
		return nil, err
	} else {
		return this.ReadResponse(res)
	}
}

func (this HTTPClient) Form(path string, v HTTPValues) ([]byte, error) {
	if res, err := this.Client.PostForm(this.Host+path, v.Values); err != nil {
		return nil, err
	} else {
		return this.ReadResponse(res)
	}
}

//http.MethodPost http.MethodGet http.MethodHead...
func (this HTTPClient) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, this.Host+path, body)
}

func (this HTTPClient) Do(req *http.Request) (*http.Response, error) {
	return this.Client.Do(req)
}

//not verify config
func TLSSkipVerifyConfig() *tls.Config {
	return &tls.Config{InsecureSkipVerify: true}
}

func MustLoadTLSFileConfig(casFile, crtFile, keyFile string) *tls.Config {
	if casFile == "" {
		panic(errors.New("casFile miss"))
	}
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
	pem, err := ioutil.ReadFile(casFile)
	if err != nil {
		panic(err)
	}
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pem) {
		panic("Failed appending certs")
	}
	config := &tls.Config{}
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0] = cert
	config.RootCAs = certPool
	return config
}

func NewHTTPClient(host string, config ...*tls.Config) HTTPClient {
	host = strings.ToLower(host)
	ret := HTTPClient{}
	ret.Host = host
	ret.IsSecure = strings.HasPrefix(host, "https")
	tr := &http.Transport{}
	if len(config) > 0 {
		tr.TLSClientConfig = config[0]
	} else if ret.IsSecure {
		tr.TLSClientConfig = TLSSkipVerifyConfig()
	}
	ret.Client = http.Client{Transport: tr}
	return ret
}
