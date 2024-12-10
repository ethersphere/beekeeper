package cmd

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/swap"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	optionNameConfigDir          = "config-dir"
	optionNameConfigGitRepo      = "config-git-repo"
	optionNameConfigGitDir       = "config-git-dir"
	optionNameConfigGitBranch    = "config-git-branch"
	optionNameConfigGitUsername  = "config-git-username"
	optionNameConfigGitPassword  = "config-git-password"
	optionNameLogVerbosity       = "log-verbosity"
	optionNameLokiEndpoint       = "loki-endpoint"
	optionNameTracingEnabled     = "tracing-enable"
	optionNameTracingEndpoint    = "tracing-endpoint"
	optionNameTracingHost        = "tracing-host"
	optionNameTracingPort        = "tracing-port"
	optionNameTracingServiceName = "tracing-service-name"
	optionNameEnableK8S          = "enable-k8s"
	optionNameInCluster          = "in-cluster"
	optionNameInClusterDomain    = "in-cluster-domain"
	optionNameKubeconfig         = "kubeconfig"
)

func init() {
	cobra.EnableCommandSorting = true
}

type command struct {
	// global configuration
	root             *cobra.Command
	globalConfig     *viper.Viper
	globalConfigFile string
	homeDir          string
	// configuration
	config *config.Config
	// kubernetes client
	k8sClient *k8s.Client
	// swap client
	swapClient swap.Client
	// log
	log logging.Logger
}

type option func(*command)

func newCommand(opts ...option) (c *command, err error) {
	c = &command{
		root: &cobra.Command{
			Use:           "beekeeper",
			Short:         "Ethereum Swarm Beekeeper",
			SilenceErrors: true,
			SilenceUsage:  true,
			PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
				return c.initConfig()
			},
		},
	}

	for _, o := range opts {
		o(c)
	}

	// find home directory
	if err := c.setHomeDir(); err != nil {
		return nil, err
	}

	c.initGlobalFlags()

	if err := c.initCheckCmd(); err != nil {
		return nil, err
	}

	if err := c.initCreateCmd(); err != nil {
		return nil, err
	}

	if err := c.initDeleteCmd(); err != nil {
		return nil, err
	}

	if err := c.initFundCmd(); err != nil {
		return nil, err
	}

	if err := c.initNodeFunderCmd(); err != nil {
		return nil, err
	}

	if err := c.initOperatorCmd(); err != nil {
		return nil, err
	}

	if err := c.initRestartCmd(); err != nil {
		return nil, err
	}

	if err := c.initPrintCmd(); err != nil {
		return nil, err
	}

	if err := c.initSimulateCmd(); err != nil {
		return nil, err
	}

	c.initVersionCmd()

	return c, nil
}

func (c *command) Execute() (err error) {
	return c.root.Execute()
}

// Execute parses command line arguments and runs appropriate functions.
func Execute() (err error) {
	c, err := newCommand()
	if err != nil {
		return err
	}
	return c.Execute()
}

func (c *command) initGlobalFlags() {
	globalFlags := c.root.PersistentFlags()
	globalFlags.StringVar(&c.globalConfigFile, "config", "", "Path to the configuration file (default is $HOME/.beekeeper.yaml)")
	globalFlags.String(optionNameConfigDir, filepath.Join(c.homeDir, "/.beekeeper/"), "Directory for configuration files")
	globalFlags.String(optionNameConfigGitRepo, "", "URL of the Git repository containing configuration files (uses the config-dir if not specified)")
	globalFlags.String(optionNameConfigGitDir, ".", "Directory within the Git repository containing configuration files. Defaults to the root directory")
	globalFlags.String(optionNameConfigGitBranch, "main", "Git branch to use for configuration files")
	globalFlags.String(optionNameConfigGitUsername, "", "Git username for authentication (required for private repositories)")
	globalFlags.String(optionNameConfigGitPassword, "", "Git password or personal access token for authentication (required for private repositories)")
	globalFlags.String(optionNameLogVerbosity, "info", "Log verbosity level (0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=trace;")
	globalFlags.String(optionNameLokiEndpoint, "", "HTTP endpoint for sending logs to Loki (e.g., http://loki.testnet.internal/loki/api/v1/push)")
	globalFlags.Bool(optionNameTracingEnabled, false, "Enable tracing for performance monitoring and debugging")
	globalFlags.String(optionNameTracingEndpoint, "127.0.0.1:6831", "Endpoint for sending tracing data, specified as host:port")
	globalFlags.String(optionNameTracingHost, "", "Host address for sending tracing data")
	globalFlags.String(optionNameTracingPort, "", "Port for sending tracing data")
	globalFlags.String(optionNameTracingServiceName, "beekeeper", "Service name identifier used in tracing data")
	globalFlags.Bool(optionNameEnableK8S, true, "Enable Kubernetes client functionality")
	globalFlags.Bool(optionNameInCluster, false, "Use the in-cluster Kubernetes client")
	globalFlags.String(optionNameInClusterDomain, "cluster.local", "In-cluster domain name for service discovery")
	globalFlags.String(optionNameKubeconfig, "~/.kube/config", "Path to the kubeconfig file")
}

func (c *command) bindGlobalFlags() (err error) {
	for _, flag := range []string{optionNameConfigDir, optionNameConfigGitRepo, optionNameConfigGitBranch, optionNameConfigGitUsername, optionNameConfigGitPassword, optionNameLogVerbosity, optionNameLokiEndpoint} {
		if err := c.globalConfig.BindPFlag(flag, c.root.PersistentFlags().Lookup(flag)); err != nil {
			return err
		}
	}
	return
}

func (c *command) initConfig() (err error) {
	// set global configuration
	cfg := viper.New()
	cfgName := ".beekeeper"
	if c.globalConfigFile != "" {
		// Use config file from the flag.
		cfg.SetConfigFile(c.globalConfigFile)
	} else {
		// Search config in home directory with name ".beekeeper" (without extension).
		cfg.AddConfigPath(c.homeDir)
		cfg.SetConfigName(cfgName)
	}

	// environment
	cfg.SetEnvPrefix("beekeeper")
	cfg.AutomaticEnv() // read in environment variables that match
	cfg.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	if c.homeDir != "" && c.globalConfigFile == "" {
		c.globalConfigFile = filepath.Join(c.homeDir, cfgName+".yaml")
	}

	// if a config file is found, read it in.
	if err := cfg.ReadInConfig(); err != nil {
		var e viper.ConfigFileNotFoundError
		if !errors.As(err, &e) {
			return err
		}
	}

	c.globalConfig = cfg
	if err := c.bindGlobalFlags(); err != nil {
		return err
	}

	// init logger
	verbosity := c.globalConfig.GetString(optionNameLogVerbosity)
	lokiEndpoint := c.globalConfig.GetString(optionNameLokiEndpoint)
	c.log, err = newLogger(c.root, verbosity, lokiEndpoint)
	if err != nil {
		return fmt.Errorf("new logger: %w", err)
	}

	if c.globalConfig.GetString(optionNameConfigGitRepo) != "" {
		// read configuration from git repo
		fs := memfs.New()
		if _, err := git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
			Auth: &http.BasicAuth{
				Username: c.globalConfig.GetString(optionNameConfigGitUsername),
				Password: c.globalConfig.GetString(optionNameConfigGitPassword),
			},
			Depth:         1,
			ReferenceName: plumbing.ReferenceName("refs/heads/" + c.globalConfig.GetString(optionNameConfigGitBranch)),
			SingleBranch:  true,
			URL:           c.globalConfig.GetString(optionNameConfigGitRepo),
		}); err != nil {
			return fmt.Errorf("cloning repo %s: %w ", c.globalConfig.GetString(optionNameConfigGitRepo), err)
		}

		dir := c.globalConfig.GetString(optionNameConfigGitDir)

		files, err := fs.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("reading git config dir: %w", err)
		}

		yamlFiles := []config.YamlFile{}
		for _, file := range files {
			if file.IsDir() || (!strings.HasSuffix(file.Name(), ".yaml") && !strings.HasSuffix(file.Name(), ".yml")) {
				continue
			}
			filePath := filepath.Join(dir, file.Name())
			f, err := fs.Open(filePath)
			if err != nil {
				return fmt.Errorf("opening file %s: %w ", file.Name(), err)
			}
			defer f.Close()

			yamlFile := make([]byte, file.Size())
			if _, err := f.Read(yamlFile); err != nil {
				return fmt.Errorf("reading file %s: %w ", file.Name(), err)
			}
			yamlFiles = append(yamlFiles, config.YamlFile{
				Name:    file.Name(),
				Content: yamlFile,
			})
		}

		if c.config, err = config.Read(c.log, yamlFiles); err != nil {
			return err
		}
	} else {
		// read configuration from directory
		files, err := os.ReadDir(c.globalConfig.GetString(optionNameConfigDir))
		if err != nil {
			return fmt.Errorf("reading config dir: %w", err)
		}

		yamlFiles := []config.YamlFile{}
		for _, file := range files {
			fullPath := filepath.Join(c.globalConfig.GetString(optionNameConfigDir) + "/" + file.Name())
			fileExt := filepath.Ext(fullPath)
			if fileExt != ".yaml" && fileExt != ".yml" {
				continue
			}
			yamlFile, err := os.ReadFile(fullPath)
			if err != nil {
				return fmt.Errorf("reading file %s: %w ", file.Name(), err)
			}
			yamlFiles = append(yamlFiles, config.YamlFile{
				Name:    file.Name(),
				Content: yamlFile,
			})
		}

		if c.config, err = config.Read(c.log, yamlFiles); err != nil {
			return err
		}
	}

	return
}

func (c *command) setHomeDir() (err error) {
	if c.homeDir != "" {
		return
	}
	dir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	c.homeDir = dir
	return nil
}

func (c *command) preRunE(cmd *cobra.Command, args []string) (err error) {
	if err := c.globalConfig.BindPFlags(cmd.Flags()); err != nil {
		return err
	}

	// set Kubernetes client
	if err := c.setK8S(); err != nil {
		return err
	}
	// set Swap client
	if err := c.setSwapClient(); err != nil {
		return err
	}
	return nil
}

func (c *command) setK8S() (err error) {
	if c.globalConfig.GetBool(optionNameEnableK8S) {
		options := []k8s.ClientOption{
			k8s.WithLogger(c.log),
			k8s.WithInCluster(c.globalConfig.GetBool(optionNameInCluster)),
			k8s.WithKubeconfigPath(c.globalConfig.GetString(optionNameKubeconfig)),
			k8s.WithInClusterDomain(c.globalConfig.GetString(optionNameInClusterDomain)),
		}

		if c.k8sClient, err = k8s.NewClient(options...); err != nil && !errors.Is(err, k8s.ErrKubeconfigNotSet) {
			return fmt.Errorf("creating Kubernetes client: %w", err)
		}
	}

	return
}

func (c *command) setSwapClient() (err error) {
	if len(c.globalConfig.GetString("geth-url")) > 0 {
		gethUrl, err := url.Parse(c.globalConfig.GetString("geth-url"))
		if err != nil {
			return fmt.Errorf("parsing Geth URL: %w", err)
		}

		c.swapClient = swap.NewGethClient(gethUrl, &swap.GethClientOptions{
			BzzTokenAddress: c.globalConfig.GetString("bzz-token-address"),
			EthAccount:      c.globalConfig.GetString("eth-account"),
		}, c.log)
	} else {
		c.swapClient = &swap.NotSet{}
	}

	return
}

func newLogger(cmd *cobra.Command, verbosity, lokiEndpoint string) (logging.Logger, error) {
	var logger logging.Logger
	opts := []logging.LoggerOption{
		logging.WithLokiOption(lokiEndpoint),
		logging.WithMetricsOption(),
	}
	switch strings.ToLower(verbosity) {
	case "0", "silent":
		logger = logging.New(io.Discard, 0)
	case "1", "error":
		logger = logging.New(cmd.OutOrStdout(), logrus.ErrorLevel, opts...)
	case "2", "warn":
		logger = logging.New(cmd.OutOrStdout(), logrus.WarnLevel, opts...)
	case "3", "info":
		logger = logging.New(cmd.OutOrStdout(), logrus.InfoLevel, opts...)
	case "4", "debug":
		logger = logging.New(cmd.OutOrStdout(), logrus.DebugLevel, opts...)
	case "5", "trace":
		logger = logging.New(cmd.OutOrStdout(), logrus.TraceLevel, opts...)
	default:
		return nil, fmt.Errorf("unknown %s level %q, use help to check flag usage options", optionNameLogVerbosity, verbosity)
	}
	return logger, nil
}
