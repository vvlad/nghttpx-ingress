package app

import (
	"github.com/golang/glog"
	"github.com/vvlad/nghttpx-ingress/app/controllers"
	"github.com/vvlad/nghttpx-ingress/app/options"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
)

func Run(s *options.NGHttpxConfig) error {
	kubeconfig, err := clientcmd.BuildConfigFromFlags(s.Master, s.Kubeconfig)
	if err != nil {
		return err
	}
	kubeClient, err := clientset.NewForConfig(restclient.AddUserAgent(kubeconfig, "ingress-manager"))
	if err != nil {
		glog.Fatalf("Invalid API configuration: %v", err)
	}

	ingressController := controllers.NewIngressController(kubeClient)
	go ingressController.Run()

	serviceLookupController := controllers.NewServiceLookupController(kubeClient)
	go serviceLookupController.Run()

	configurationController := controllers.NewConfigurationController()
	go configurationController.Run()

	go func() {
		for event := range ingressController.Updates {
			serviceLookupController.Requests <- event
		}
	}()

	for event := range serviceLookupController.Updates {
		configurationController.Updates <- event
	}

	return nil
}
