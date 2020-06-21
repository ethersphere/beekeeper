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

	if release == "" {
		return fmt.Errorf("error release has to be specified")
	}

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), debug); err != nil {
		log.Fatal(err)
	}
	client := action.NewUpgrade(actionConfig)

	client.Namespace = namespace
	// client.ReleaseName = release
	cp, err := client.ChartPathOptions.LocateChart(chart, settings)
	if err != nil {
		log.Fatal(err)
	}

	client.ReuseValues = true

	p := getter.All(settings)
	valueOpts := &values.Options{}
	vals, err := valueOpts.MergeValues(p)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("VALUES: %+v", vals)

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

	// if req := chartRequested.Metadata.Dependencies; req != nil {
	// 	if err := action.CheckDependencies(chartRequested, req); err != nil {
	// 		if client.DependencyUpdate {
	// 			man := &downloader.Manager{
	// 				Out:              os.Stdout,
	// 				ChartPath:        cp,
	// 				Keyring:          client.ChartPathOptions.Keyring,
	// 				SkipUpdate:       false,
	// 				Getters:          p,
	// 				RepositoryConfig: settings.RepositoryConfig,
	// 				RepositoryCache:  settings.RepositoryCache,
	// 			}
	// 			if err := man.Update(); err != nil {
	// 				log.Fatal(err)
	// 			}
	// 		} else {
	// 			log.Fatal(err)
	// 		}
	// 	}
	// }

	rel, err := client.Run(release, chartRequested, vals)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(rel.Manifest)

	return nil
}

func isChartInstallable(ch *chart.Chart) (bool, error) {
	switch ch.Metadata.Type {
	case "", "application":
		return true, nil
	}
	return false, errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}

func debug(format string, v ...interface{}) {
	format = fmt.Sprintf("[debug] %s\n", format)
	log.Output(2, fmt.Sprintf(format, v...))
}
