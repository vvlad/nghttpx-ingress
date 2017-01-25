package main

import (
	"fmt"
	"github.com/spf13/pflag"
	"github.com/vvlad/nghttpx-ingress/app"
	"github.com/vvlad/nghttpx-ingress/app/options"
	"k8s.io/apiserver/pkg/server/healthz"
	"k8s.io/kubernetes/pkg/util/flag"
	"k8s.io/kubernetes/pkg/util/logs"
	"k8s.io/kubernetes/pkg/version/verflag"
	"os"
	// "k8s.io/client-go/tools/clientcmd"
	// "k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
)

func init() {
	healthz.DefaultHealthz()
}

func main() {
	s := options.NewNGHttpxConfig()
	s.AddFlags(pflag.CommandLine)

	flag.InitFlags()
	logs.InitLogs()

	verflag.PrintAndExitIfRequested()

	if err := app.Run(s); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

}
