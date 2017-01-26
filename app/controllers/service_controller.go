package controllers

import (
	"github.com/golang/glog"
	"github.com/vvlad/nghttpx-ingress/app/options"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
	"time"
)

type serviceController struct {
	client          *clientset.Clientset
	Store           cache.Store
	CacheController cache.Controller
}

func NewServiceController(options *options.NGHttpxConfig) *serviceController {
	c := serviceController{
		client: options.Client,
	}
	c.Store, c.CacheController = cache.NewInformer(&c, &api.Service{}, 1*time.Second, &c)
	return &c
}

func (c *serviceController) List(options metav1.ListOptions) (runtime.Object, error) {
	return c.client.CoreV1().Services(api.NamespaceAll).List(options)
}

func (c *serviceController) Watch(options metav1.ListOptions) (watch.Interface, error) {
	return c.client.CoreV1().Services(api.NamespaceAll).Watch(options)
}

func (c *serviceController) OnAdd(obj interface{}) {
	c.Store.Add(obj)
}

func (c *serviceController) OnUpdate(oldObj, newObj interface{}) {
	c.Store.Delete(oldObj)
	c.Store.Add(newObj)
}

func (c *serviceController) OnDelete(obj interface{}) {
	c.Store.Delete(obj)
}

func (c *serviceController) Run(stopCh <-chan struct{}) {
	glog.Infoln("Running ...")
	c.CacheController.Run(stopCh)
}
