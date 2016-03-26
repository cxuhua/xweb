package xweb

type IModel interface {
	GetString() string
	GetData() []byte
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

type MVC struct {
	IMVC
	status int
	view   string
	render string
	model  IModel
}

func (this *MVC) GetView() string {
	return this.view
}

func (this *MVC) SetView(v string) {
	this.view = v
}

func (this *MVC) GetModel() IModel {
	return this.model
}

func (this *MVC) SetModel(v IModel) {
	this.model = v
}

func (this *MVC) GetRender() string {
	return this.render
}

func (this *MVC) SetRender(v string) {
	this.render = v
}

func (this *MVC) GetStatus() int {
	return this.status
}

func (this *MVC) SetStatus(v int) {
	this.status = v
}
