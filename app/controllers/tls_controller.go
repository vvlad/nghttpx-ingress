package controllers

import (
	"github.com/golang/glog"
	"github.com/vvlad/nghttpx-ingress/app/options"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
	"time"
)

type TLSController struct {
	client          *clientset.Clientset
	Store           cache.Store
	CacheController cache.Controller
}

func NewTLSController(options *options.NGHttpxConfig) *TLSController {
	c := TLSController{
		client: options.Client,
	}
	c.Store, c.CacheController = cache.NewInformer(&c, &v1.Secret{}, 1*time.Second, &c)
	return &c
}

func (c *TLSController) List(options metav1.ListOptions) (runtime.Object, error) {
	return c.client.CoreV1().Secrets(api.NamespaceAll).List(options)
}

func (c *TLSController) Watch(options metav1.ListOptions) (watch.Interface, error) {
	return c.client.CoreV1().Secrets(api.NamespaceAll).Watch(options)
}

func (c *TLSController) OnAdd(obj interface{}) {
	secret := obj.(*v1.Secret)
	if secret.Type == "kubernetes.io/tls" {
		c.Store.Add(obj)
	}
}

func (c *TLSController) OnUpdate(oldObj, newObj interface{}) {
	c.Store.Delete(oldObj)
	secret := newObj.(*v1.Secret)
	if secret.Type == "kubernetes.io/tls" {
		c.Store.Add(newObj)
	}
}

func (c *TLSController) OnDelete(obj interface{}) {
	c.Store.Delete(obj)
}

func (c *TLSController) Run(stopCh <-chan struct{}) {
	glog.Infoln("Running ...")
	c.CacheController.Run(stopCh)
}
