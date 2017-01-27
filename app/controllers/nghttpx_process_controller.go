package controllers

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/vvlad/nghttpx-ingress/app/options"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

type NGHttpxProcessController struct {
	ConfigChannel chan string
	cmd           *exec.Cmd
	options       *options.NGHttpxConfig
}

func nghttpxBinaryPath() (string, error) {

	paths := []string{
		"/usr/bin/nghttpx",
		"/usr/local/bin/nghttpx",
	}

	for _, path := range paths {
		if fileExists(path) {
			return path, nil
		}
	}

	return "", errors.New("path not found")
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

var (
	ngHttpXConfigFile string = "/tmp/nghttpx.conf"
)

func nghttpxCommand() *exec.Cmd {
	cmdPath, err := nghttpxBinaryPath()
	if err != nil {
		glog.Fatalln(err)
	}
	if fileExists("/etc/ssl/private/ssl-cert-snakeoil.key") {
		return exec.Command(cmdPath, "--conf", ngHttpXConfigFile, "/etc/ssl/private/ssl-cert-snakeoil.key", "/etc/ssl/certs/ssl-cert-snakeoil.pem")
	} else {
		return exec.Command(cmdPath, "--conf", ngHttpXConfigFile)
	}
}

func NewNGHttpxProcessController(options *options.NGHttpxConfig) *NGHttpxProcessController {
	return &NGHttpxProcessController{
		ConfigChannel: make(chan string),
		cmd:           nghttpxCommand(),
		options:       options,
	}
}

func (n *NGHttpxProcessController) Start() bool {
	glog.Warningln("Starting ")
	err := n.cmd.Start()
	if err != nil {
		glog.Errorln(err)
		return false
	}
	return true
}

func (n *NGHttpxProcessController) Stop() {
	if n.cmd.Process != nil {
		n.cmd.Process.Kill()
		n.cmd.Wait()
	}
}

func (n *NGHttpxProcessController) Reload() bool {
	if n.cmd.Process != nil {
		return n.cmd.Process.Signal(syscall.SIGHUP) == nil
	}

	return false
}

func (n *NGHttpxProcessController) Restart() bool {
	n.Stop()
	return n.Start()
}

func (n *NGHttpxProcessController) Run(stopCh <-chan struct{}) {
	go n.startHTTPServer()
	for config := range n.ConfigChannel {
		glog.Info("Reloading config ...")
		err := ioutil.WriteFile(ngHttpXConfigFile, []byte(config), 0644)
		if err != nil {
			glog.Errorln(err)
			return
		}
		if !n.Reload() {
			if !n.Restart() {
				glog.Errorln("Unable to restart http worker")
			}
		}
	}
	<-stopCh
}

func (n *NGHttpxProcessController) redirectToHttps(w http.ResponseWriter, req *http.Request) {

	if req.URL.Path == "/healthz" {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "ok")
		return
	}

	host := req.Host
	if n.options.Port != "80" {
		host = strings.Replace(host, n.options.Port, n.options.TLSPort, 1)
	}
	target := "https://" + host + req.URL.Path
	if len(req.URL.RawQuery) > 0 {
		target += "?" + req.URL.RawQuery
	}
	http.Redirect(w, req, target, http.StatusPermanentRedirect)
}

func (n *NGHttpxProcessController) startHTTPServer() {
	address := fmt.Sprintf("0.0.0.0:%s", n.options.HealthPort)
	glog.Infoln("Starting healthz service on ", address)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "404 Not Found")
	})
	go http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", n.options.Port), http.HandlerFunc(n.redirectToHttps))
	http.ListenAndServe(address, nil)
}
