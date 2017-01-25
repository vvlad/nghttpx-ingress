package controllers

import (
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions/v1beta1"
	"k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
)

type IngressController struct {
	Updates chan BackendEvent
	client  *clientset.Clientset
}

func NewIngressController(clientset *clientset.Clientset) *IngressController {
	return &IngressController{
		Updates: make(chan BackendEvent),
		client:  clientset,
	}
}

func (i *IngressController) updateIngress(ingress *v1beta1.Ingress) {

	for _, rule := range ingress.Spec.Rules {
		for _, path := range rule.HTTP.Paths {
			i.Updates <- BackendEvent{
				Type:        Added,
				Namespace:   ingress.ObjectMeta.Namespace,
				ServiceName: path.Backend.ServiceName,
				ServicePort: path.Backend.ServicePort.String(),
				Backend: Backend{
					Port:     path.Backend.ServicePort.String(),
					Hostname: rule.Host,
					Path:     path.Path,
				},
			}
		}
	}
}

func (i *IngressController) removeIngress(ingress *v1beta1.Ingress) {

}

func (i *IngressController) Run() {
	watcher, err := i.client.Ingresses(api.NamespaceAll).Watch(v1.ListOptions{})
	if err != nil {
		glog.Fatalln(err)
	}

	methods := map[watch.EventType]func(*v1beta1.Ingress){
		watch.Added:    i.updateIngress,
		watch.Modified: i.updateIngress,
		watch.Deleted:  i.removeIngress,
	}

	for event := range watcher.ResultChan() {
		ingress, ok := event.Object.(*v1beta1.Ingress)
		if !ok {
			continue
		}
		if ingress.ObjectMeta.GetAnnotations()["kubernetes.io/ingress.class"] != "nghttpx" {
			glog.Warningf("discarding ingress `%s' due to missing annotation\n", ingress.Name)
			continue
		}
		if method := methods[event.Type]; method != nil {
			method(ingress)
		}
	}
}
