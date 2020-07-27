package helm3

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/strvals"

	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/util/homedir"
)

var settings *cli.EnvSettings

func Upgrade(kubeconfig string, namespace string, release string, chart string, args map[string]string) (err error) {
	os.Setenv("HELM_NAMESPACE", namespace)
	settings = cli.New()

	if kubeconfig == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		} else {
			kubeconfig = "kubeconfig"
		}
	}

	os.Setenv("KUBECONFIG", kubeconfig)

	if release == "" {
		return fmt.Errorf("error release has to be specified")
	}

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), noDebug); err != nil {
		log.Fatal(err)
	}
	client := action.NewUpgrade(actionConfig)

	client.Namespace = namespace
	// client.ReleaseName = release
	cp, err := client.ChartPathOptions.LocateChart(chart, settings)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: Add reuse values and wait as cli arguments
	client.ReuseValues = true
	client.Wait = true

	p := getter.All(settings)
	valueOpts := &values.Options{}
	vals, err := valueOpts.MergeValues(p)
	if err != nil {
		log.Fatal(err)
	}

	// Add args
	if err := strvals.ParseInto(args["set"], vals); err != nil {
		log.Fatal(errors.Wrap(err, "failed parsing --set data"))
	}

	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := loader.Load(cp)
	if err != nil {
		log.Fatal(err)
	}

	validInstallableChart, err := isChartInstallable(chartRequested)
	if !validInstallableChart {
		log.Fatal(err)
	}

	_, err = client.Run(release, chartRequested, vals)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println(rel.Manifest)

	return nil
}

func isChartInstallable(ch *chart.Chart) (bool, error) {
	switch ch.Metadata.Type {
	case "", "application":
		return true, nil
	}
	return false, errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}

// func debug(format string, v ...interface{}) {
// 	format = fmt.Sprintf("[debug] %s\n", format)
// 	_ = log.Output(2, fmt.Sprintf(format, v...))
// }

func noDebug(format string, v ...interface{}) {
	// format = fmt.Sprintf("[debug] %s\n", format)
	// _ = log.Output(2, fmt.Sprintf(format, v...))
}
