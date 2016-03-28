package xweb

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type IModel interface {
	String() string
}

//内存模版输出

type TempModal struct {
	IModel
	Template string
	Modal    interface{}
}

func (this *TempModal) String() string {
	return this.Template
}

//文件输出
type FileModal struct {
	IModel
	http.Header
	Name    string        //名称
	ModTime time.Time     //修改时间
	Reader  io.ReadSeeker //读取接口
}

func (this *FileModal) String() string {
	return this.Name
}

func NewFileModal() *FileModal {
	return &FileModal{Header: http.Header{}, ModTime: time.Now()}
}

//用于TEXT输出
type StringModel struct {
	IModel
	Text string
}

func (this *StringModel) String() string {
	return this.Text
}

//用于DATA输出
type BinaryModel struct {
	IModel
	Data []byte
}

func (this *BinaryModel) String() string {
	return string(this.Data)
}

//数据模型
type DataModel struct {
	IModel `bson:"-" json:"-" xml:"-"`
}

func (this *DataModel) String() string {
	return "DataModel"
}

//渲染模型
type HTTPModel struct {
	IModel `bson:"-" json:"-" xml:"-"`
	Code   int    `json:"code" xml:"code"`
	Error  string `json:"error,omitempty" xml:"error,omitempty"`
}

func (this *HTTPModel) String() string {
	return fmt.Sprintf("%d:%s", this.Code, this.Error)
}

func NewHTTPError(code int, err string) *HTTPModel {
	return &HTTPModel{Code: code, Error: err}
}

func NewHTTPSuccess() *HTTPModel {
	return &HTTPModel{Code: 0, Error: ""}
}

type IMVC interface {
	GetView() string
	SetView(string)

	GetModel() IModel
	SetModel(IModel)

	GetRender() string
	SetRender(string)

	GetStatus() int
	SetStatus(int)
}

type mvc struct {
	IMVC
	status int
	view   string
	render string
	model  IModel
}

func (this *mvc) GetView() string {
	return this.view
}

func (this *mvc) SetView(v string) {
	this.view = v
}

func (this *mvc) GetModel() IModel {
	return this.model
}

func (this *mvc) SetModel(v IModel) {
	this.model = v
}

func (this *mvc) GetRender() string {
	return this.render
}

func (this *mvc) SetRender(v string) {
	this.render = v
}

func (this *mvc) GetStatus() int {
	return this.status
}

func (this *mvc) SetStatus(v int) {
	this.status = v
}
