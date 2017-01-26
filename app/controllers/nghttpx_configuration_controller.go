package controllers

import (
	"bytes"
	"fmt"
	"github.com/golang/glog"
	"github.com/vvlad/nghttpx-ingress/app/options"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/apis/extensions/v1beta1"
	"k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
	"reflect"
	"sort"
	"text/template"
	"time"
)

type NGHttpx struct {
	Ingress                   cache.Store
	Service                   cache.Store
	DefaultHttpBackendService string
	Controllers               []cache.Controller
	ConfigChannel             chan string
}

type NGHttpxServerConfig struct {
	Backends  []Backend
	NoTLSPort string
	TLSPort   string
}

func NewNGHttpxServerConfig(backends []Backend, options *options.NGHttpxConfig) NGHttpxServerConfig {
	sort.Sort(ByBackend(backends))
	return NGHttpxServerConfig{
		Backends:  backends,
		NoTLSPort: options.Port,
		TLSPort:   options.TLSPort,
	}
}

type nghttpxConfigurationController struct {
	client   *clientset.Clientset
	ticker   *time.Ticker
	config   NGHttpxServerConfig
	template *template.Template
	options  *options.NGHttpxConfig
	NGHttpx
}

var (
	nghttpTemplate = `frontend=*,{{.NoTLSPort}};no-tls
frontend=*,{{.TLSPort}};no-tls{{range $service := .Backends}}
backend={{$service.Address}},{{$service.Port}};{{$service.Hostname}}{{$service.Path}};proto=h2;{{end}}
backend=127.0.0.1,8080;;
`
)

func NewNGHttpxConfigurationController(options *options.NGHttpxConfig, config NGHttpx) *nghttpxConfigurationController {
	backends := make([]Backend, 0)
	return &nghttpxConfigurationController{
		NGHttpx:  config,
		client:   options.Client,
		ticker:   time.NewTicker(options.ResyncPeriod),
		config:   NewNGHttpxServerConfig(backends, options),
		options:  options,
		template: template.Must(template.New("config").Parse(nghttpTemplate)),
	}
}

func (n *nghttpxConfigurationController) Run(stopCh <-chan struct{}) {
	glog.Infoln("Running ...")
	n.update(n.config)
	for _ = range n.ticker.C {
		n.CheckAndReload()
	}
	<-stopCh
}

func (n *nghttpxConfigurationController) CheckAndReload() {
	if !n.controllersInSync() {
		glog.Warningln("Delaying checking changes ...")
		return
	}

	newBackends := make([]Backend, 0)
	for _, ingressObject := range n.Ingress.List() {
		ingress := ingressObject.(*v1beta1.Ingress)
		newBackends = append(newBackends, n.buildBackends(ingress)...)
	}

	newConfig := NewNGHttpxServerConfig(newBackends, n.options)

	if !reflect.DeepEqual(n.config, newConfig) {
		n.update(newConfig)
	}
}

func (n *nghttpxConfigurationController) update(config NGHttpxServerConfig) {
	glog.Warningln("Configuration changed")
	var doc bytes.Buffer
	if err := n.template.ExecuteTemplate(&doc, "config", config); err != nil {
		glog.Errorln(err)
		return
	}
	n.ConfigChannel <- doc.String()
	n.config = config
}

func (n *nghttpxConfigurationController) controllersInSync() bool {
	for _, c := range n.Controllers {
		if !c.HasSynced() {
			return false
		}
	}
	return true
}

func (n *nghttpxConfigurationController) buildBackends(ingress *v1beta1.Ingress) []Backend {
	backends := make([]Backend, 0)
	for _, rule := range ingress.Spec.Rules {
		for _, path := range rule.HTTP.Paths {
			backend := Backend{
				Hostname: rule.Host,
				Path:     path.Path,
				Port:     path.Backend.ServicePort.String(),
			}

			if service := n.findService(fmt.Sprintf("%s/%s", ingress.ObjectMeta.Namespace, path.Backend.ServiceName)); service != nil {
				backend.Address = service.Spec.ClusterIP
			} else {
				backend.Address = "127.0.0.1"
				backend.Port = "8080"
				backend.Path = "/"
			}
			backends = append(backends, backend)
		}
	}
	return backends
}

func (n *nghttpxConfigurationController) findService(key string) *v1.Service {
	if object, exists, err := n.Service.GetByKey(key); err == nil && exists {
		return object.(*v1.Service)
	}
	return nil
}
