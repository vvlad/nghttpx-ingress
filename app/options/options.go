package options

import (
	"github.com/spf13/pflag"
	"k8s.io/kubernetes/pkg/util/config"
)

type NGHttpxConfig struct {
	Kubeconfig string
	Master     string
}

func NewNGHttpxConfig() *NGHttpxConfig {
	return &NGHttpxConfig{}
}

func (s *NGHttpxConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.Kubeconfig, "kubeconfig", s.Kubeconfig, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	fs.StringVar(&s.Master, "master", s.Master, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	config.DefaultFeatureGate.AddFlag(fs)
}
