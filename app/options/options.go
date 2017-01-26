package options

import (
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/pkg/util/config"
	"time"
)

type NGHttpxConfig struct {
	configFile   string
	master       string
	Client       *clientset.Clientset
	ResyncPeriod time.Duration
	Port         string
	TLSPort      string
}

func NewNGHttpxConfig() *NGHttpxConfig {
	return &NGHttpxConfig{}
}

func (s *NGHttpxConfig) AddFlags(fs *pflag.FlagSet) {
	s.ResyncPeriod = 1 * time.Second
	s.TLSPort = "30443"
	s.Port = "30080"
	fs.StringVar(&s.configFile, "kubeconfig", s.configFile, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	fs.StringVar(&s.master, "master", s.master, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	fs.DurationVar(&s.ResyncPeriod, "resync-period", s.ResyncPeriod, "The sync interval")
	fs.StringVar(&s.Port, "port", s.Port, "no-tls port")
	fs.StringVar(&s.TLSPort, "tls-port", s.TLSPort, "tls port")
	config.DefaultFeatureGate.AddFlag(fs)
}

func (s *NGHttpxConfig) Run() {
	config, err := clientcmd.BuildConfigFromFlags(s.master, s.configFile)
	if err != nil {
		glog.Errorln(err)
	}
	s.Client, err = clientset.NewForConfig(restclient.AddUserAgent(config, "ingress-manager"))
	if err != nil {
		glog.Fatalf("Invalid API configuration: %v", err)
	}
}
