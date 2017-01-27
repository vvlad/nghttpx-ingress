package controllers

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/golang/glog"
	"github.com/vvlad/nghttpx-ingress/app/options"
	"io"
	"io/ioutil"
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
	TLS                       cache.Store
	DefaultHttpBackendService string
	Controllers               []cache.Controller
	ConfigChannel             chan string
}

type TLS struct {
	CertPath string
	KeyPath  string
}

type NGHttpxConfig struct {
	Backends     []Backend
	NoTLSPort    string
	TLSPort      string
	FallbackPort string
	Certs        []TLS
}

func NewNGHttpxConfig(backends []Backend, options *options.NGHttpxConfig, certs []TLS) NGHttpxConfig {
	sort.Sort(ByBackend(backends))
	return NGHttpxConfig{
		Backends:     backends,
		NoTLSPort:    options.Port,
		TLSPort:      options.TLSPort,
		FallbackPort: options.HealthPort,
		Certs:        certs,
	}
}

type NGHttpxConfigurationController struct {
	client   *clientset.Clientset
	ticker   *time.Ticker
	config   NGHttpxConfig
	template *template.Template
	options  *options.NGHttpxConfig
	NGHttpx
}

var (
	nghttpTemplate = `frontend=*,{{.NoTLSPort}};no-tls
frontend=*,{{.TLSPort}};tls{{range $service := .Backends}}
backend={{$service.Address}},{{$service.Port}};{{$service.Hostname}}{{$service.Path}};proto=h2;{{end}}
backend=127.0.0.1,{{.FallbackPort}};;{{range $cert := .Certs}}
subcert={{$cert.KeyPath}}:{{$cert.CertPath}}{{end}}
`
)

func NewNGHttpxConfigurationController(options *options.NGHttpxConfig, config NGHttpx) *NGHttpxConfigurationController {
	backends := make([]Backend, 0)
	return &NGHttpxConfigurationController{
		NGHttpx:  config,
		client:   options.Client,
		ticker:   time.NewTicker(options.ResyncPeriod),
		config:   NewNGHttpxConfig(backends, options, []TLS{}),
		options:  options,
		template: template.Must(template.New("config").Parse(nghttpTemplate)),
	}
}

func (n *NGHttpxConfigurationController) Run(stopCh <-chan struct{}) {
	glog.Infoln("Running ...")
	n.update(n.config)
	for _ = range n.ticker.C {
		n.CheckAndReload()
	}
	<-stopCh
}

func (n *NGHttpxConfigurationController) CheckAndReload() {
	if !n.controllersInSync() {
		glog.Warningln("Delaying checking changes ...")
		return
	}

	newBackends := make([]Backend, 0)
	tlsCerts := make([]TLS, 0)
	for _, ingressObject := range n.Ingress.List() {
		ingress := ingressObject.(*v1beta1.Ingress)
		newBackends = append(newBackends, n.buildBackends(ingress)...)

		for _, tls := range ingress.Spec.TLS {
			secret := n.findTLSSecret(fmt.Sprintf("%s/%s", ingress.Namespace, tls.SecretName))
			if secret != nil {
				tlsCerts = append(tlsCerts, createTLSCert(secret, tls.Hosts))
			}
		}
	}

	newConfig := NewNGHttpxConfig(newBackends, n.options, tlsCerts)

	if !reflect.DeepEqual(n.config, newConfig) {
		n.update(newConfig)
	}
}

func (n *NGHttpxConfigurationController) update(config NGHttpxConfig) {
	glog.Warningln("Configuration changed")
	var doc bytes.Buffer
	if err := n.template.ExecuteTemplate(&doc, "config", config); err != nil {
		glog.Errorln(err)
		return
	}
	n.ConfigChannel <- doc.String()
	n.config = config
}

func (n *NGHttpxConfigurationController) controllersInSync() bool {
	for _, c := range n.Controllers {
		if !c.HasSynced() {
			return false
		}
	}
	return true
}

func (n *NGHttpxConfigurationController) buildBackends(ingress *v1beta1.Ingress) []Backend {
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

func (n *NGHttpxConfigurationController) findService(key string) *v1.Service {
	if object, exists, err := n.Service.GetByKey(key); err == nil && exists {
		return object.(*v1.Service)
	}
	return nil
}

func (n *NGHttpxConfigurationController) findTLSSecret(key string) *v1.Secret {
	if object, exists, err := n.TLS.GetByKey(key); err == nil && exists {
		return object.(*v1.Secret)
	}
	return nil
}

func createTLSCert(secret *v1.Secret, hosts Hosts) TLS {
	h := md5.New()
	sort.Sort(hosts)
	for _, host := range hosts {
		io.WriteString(h, host)
	}
	h.Write(secret.Data["tls.crt"])
	h.Write(secret.Data["tls.key"])
	sum := hex.EncodeToString(h.Sum(nil))

	certPath := fmt.Sprintf("/tmp/ssl-%s.pem", sum)
	if !fileExists(certPath) {
		ioutil.WriteFile(certPath, secret.Data["tls.crt"], 0644)
	}

	keyPath := fmt.Sprintf("/tmp/ssl-%s.key", sum)
	if !fileExists(keyPath) {
		ioutil.WriteFile(keyPath, secret.Data["tls.key"], 0644)
	}

	tlsCert := TLS{
		CertPath: certPath,
		KeyPath:  keyPath,
	}
	return tlsCert
}
