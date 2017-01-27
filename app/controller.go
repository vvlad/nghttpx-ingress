package app

import (
	"github.com/golang/glog"
	"github.com/vvlad/nghttpx-ingress/app/controllers"
	"github.com/vvlad/nghttpx-ingress/app/options"
	"k8s.io/client-go/tools/cache"
)

func Run(config *options.NGHttpxConfig) error {

	glog.Info("Starting nginx-ingress")
	stopCh := make(chan struct{})

	ingressController := controllers.NewIngressController(config)
	go ingressController.Run(stopCh)
	serviceController := controllers.NewServiceController(config)
	go serviceController.Run(stopCh)
	tlsController := controllers.NewTLSController(config)
	go tlsController.Run(stopCh)

	nghttpxProcessController := controllers.NewNGHttpxProcessController(config)
	go nghttpxProcessController.Run(stopCh)

	nghttpxConfigurationController := controllers.NewNGHttpxConfigurationController(config, controllers.NGHttpx{
		ConfigChannel: nghttpxProcessController.ConfigChannel,
		Ingress:       ingressController.Store,
		Service:       serviceController.Store,
		TLS:           tlsController.Store,
		Controllers: []cache.Controller{
			ingressController.CacheController,
			serviceController.CacheController,
			tlsController.CacheController,
		},
	})
	go nghttpxConfigurationController.Run(stopCh)

	<-stopCh

	return nil
}
