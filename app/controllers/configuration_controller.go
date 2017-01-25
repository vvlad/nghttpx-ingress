package controllers

import (
	"fmt"
	"github.com/golang/glog"
	"os"
	"text/template"
)

type Backend struct {
	Address  string
	Port     string
	Hostname string
	Path     string
}

type BackendEventType int

const (
	Unknown BackendEventType = iota
	Added
	Deleted
)

type BackendEvent struct {
	Type        BackendEventType
	Namespace   string
	ServiceName string
	ServicePort string
	Backend     Backend
}

type backendConfig struct {
	Backends map[string]Backend
}

type ConfigurationController struct {
	Updates chan BackendEvent
	config  backendConfig
}

func NewConfigurationController() *ConfigurationController {
	return &ConfigurationController{
		Updates: make(chan BackendEvent),
		config:  backendConfig{make(map[string]Backend)},
	}
}

func (c *ConfigurationController) Run() {
	c.Restart()
	for update := range c.Updates {

		switch {
		case update.Type == Added:
			{
				c.config.Backends[update.hash()] = update.Backend
			}
		case update.Type == Deleted:
			{
				delete(c.config.Backends, update.hash())
			}
		}
		glog.Info(c.config.Backends)
		c.Restart()
	}
}

var (
	nghttpTemplate = `frontend=*,30080;no-tls
frontend=*,30443;no-tls
{{range $service := .Backends}}
backend={{$service.Address}},{{$service.Port}};{{$service.Hostname}}{{$service.Path}};proto=h2;
{{end}}`
)

func (c *ConfigurationController) Restart() {
	t := template.Must(template.New("nghttpdx").Parse(nghttpTemplate))
	f, err := os.Create("/tmp/nghttpdx.conf")
	if err != nil {
		glog.Errorln(err)
		return
	}
	if err := t.ExecuteTemplate(f, "nghttpdx", c.config); err != nil {
		panic(err)
	}
	f.Close()
}

func (b BackendEvent) hash() string {
	return fmt.Sprintf("%s/%s:%s", b.Namespace, b.ServiceName, b.ServicePort)
}

// import (
//  "github.com/golang/glog"
// )

// func (c *controllerImpl) reconfigure() {
//  c.writeConfig()
// }

// func (c *controllerImpl) writeConfig() {
// }
