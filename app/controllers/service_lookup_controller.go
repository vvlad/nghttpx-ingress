package controllers

import (
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	// "k8s.io/apimachinery/pkg/watch"
	// "k8s.io/client-go/pkg/api"
	// "k8s.io/kubernetes/pkg/apis/extensions/v1beta1"
	"k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
)

type ServiceLookupController struct {
	Requests chan BackendEvent
	Updates  chan BackendEvent
	client   *clientset.Clientset
}

func NewServiceLookupController(clientset *clientset.Clientset) *ServiceLookupController {
	return &ServiceLookupController{
		Requests: make(chan BackendEvent),
		Updates:  make(chan BackendEvent),
		client:   clientset,
	}
}

func (s *ServiceLookupController) Run() {
	glog.Infoln("Running...")
	for request := range s.Requests {
		glog.Infoln("Got request", request)
		if request.Type == Deleted {
			s.Updates <- request
			continue
		}
		service, err := s.client.Services(request.Namespace).Get(request.ServiceName, v1.GetOptions{})
		if err != nil {
			glog.Errorln(err)
			continue
		}
		request.Backend.Address = service.Spec.ClusterIP
		s.Updates <- request
	}
}
