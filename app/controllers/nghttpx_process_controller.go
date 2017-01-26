package controllers

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"syscall"
)

type NGHttpxProcessController struct {
	ConfigChannel chan string
	cmd           *exec.Cmd
}

func nghttpxBinaryPath() (string, error) {

	paths := []string{
		"/usr/bin/nghttpx",
		"/usr/local/bin/nghttpx",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}
		return path, nil
	}

	return "", errors.New("path not found")
}

var (
	ngHttpXConfigFile string = "/tmp/nghttpx.conf"
)

func nghttpxCommand() *exec.Cmd {
	cmdPath, err := nghttpxBinaryPath()
	if err != nil {
		glog.Fatalln(err)
	}
	return exec.Command(cmdPath, "--conf", ngHttpXConfigFile)
}

func NewNGHttpxProcessController() *NGHttpxProcessController {
	return &NGHttpxProcessController{
		ConfigChannel: make(chan string),
		cmd:           nghttpxCommand(),
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
		n.cmd.Process.Wait()
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

func (n *NGHttpxProcessController) startHTTPServer() {
	glog.Infoln("Starting default 404 server on 127.0.0.1:8080")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "NotFound")
	})
	glog.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}
