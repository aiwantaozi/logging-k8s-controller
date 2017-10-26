package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/aiwantaozi/logging-k8s-controller/api"
	"github.com/aiwantaozi/logging-k8s-controller/k8sutils"
	"github.com/urfave/cli"
	"github.com/urfave/negroni"
)

var VERSION = "v0.0.0-dev"

/* Todo
1. field validate like host ip could access
*/

func main() {
	app := cli.NewApp()
	app.Name = "logging-k8s-controller"
	app.Version = VERSION
	app.Usage = "You need help!"
	app.Action = startServer
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name: "debug",
			Usage: fmt.Sprintf(
				"Set true to get debug logs",
			),
		},
		cli.StringFlag{
			Name:  "listen",
			Value: ":8090",
			Usage: fmt.Sprintf(
				"Address to listen to (TCP)",
			),
		},
		cli.StringFlag{
			Name:  "k8s-config-path",
			Usage: "k8s config path",
			Value: "/Users/fengcaixiao/.kube/config",
		},
	}
	app.Run(os.Args)
}

func startServer(c *cli.Context) error {
	k8sut := k8sutils.K8sClientConfig{ConfigPath: c.String("k8s-config-path")}
	cfg, err := k8sut.New()
	if err != nil {
		return err
	}
	if err := k8sut.IsReachable(); err != nil {
		return err
	}
	listen := c.GlobalString("listen")
	server, err := api.NewServer(cfg)
	if err != nil {
		return err
	}
	router := http.Handler(api.NewRouter(server))

	n := negroni.New()
	n.Use(negroni.NewLogger())
	n.UseHandler(router)

	logrus.Infof("Listening on %s", listen)

	return http.ListenAndServe(listen, n)
}
