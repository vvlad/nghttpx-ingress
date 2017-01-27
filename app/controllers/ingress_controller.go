package controllers

import (
	"github.com/golang/glog"
	"github.com/vvlad/nghttpx-ingress/app/options"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/apis/extensions/v1beta1"
	"k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
	"time"
)

type ingressController struct {
	client          *clientset.Clientset
	Store           cache.Store
	CacheController cache.Controller
}

func NewIngressController(options *options.NGHttpxConfig) *ingressController {
	c := ingressController{
		client: options.Client,
	}

	c.Store, c.CacheController = cache.NewInformer(&c, &v1beta1.Ingress{}, 1*time.Second, &c)
	return &c
}

func (c *ingressController) List(options metav1.ListOptions) (runtime.Object, error) {
	return c.client.ExtensionsV1beta1().Ingresses(api.NamespaceAll).List(options)
}

func (c *ingressController) Watch(options metav1.ListOptions) (watch.Interface, error) {
	return c.client.ExtensionsV1beta1().Ingresses(api.NamespaceAll).Watch(options)
}

func (c *ingressController) OnAdd(obj interface{}) {
	ingress := obj.(*v1beta1.Ingress)
	if ingress.ObjectMeta.GetAnnotations()["kubernetes.io/ingress.class"] == "nghttpx" {
		c.Store.Add(obj)
	}
}
func (c *ingressController) OnUpdate(oldObj, newObj interface{}) {
	c.Store.Delete(oldObj)
	ingress := newObj.(*v1beta1.Ingress)
	if ingress.ObjectMeta.GetAnnotations()["kubernetes.io/ingress.class"] == "nghttpx" {
		c.Store.Add(newObj)
	}
}

func (c *ingressController) OnDelete(obj interface{}) {
	c.Store.Delete(obj)
}

func (c *ingressController) Run(stopCh <-chan struct{}) {
	glog.Infoln("Running ...")
	c.CacheController.Run(stopCh)
}
