package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/httpx"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/node"
	"github.com/ethersphere/beekeeper/pkg/scheduler"
	"github.com/ethersphere/beekeeper/pkg/swap"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	httptransport "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	optionNameConfigDir          = "config-dir"
	optionNameConfigGitBranch    = "config-git-branch"
	optionNameConfigGitDir       = "config-git-dir"
	optionNameConfigGitPassword  = "config-git-password"
	optionNameConfigGitRepo      = "config-git-repo"
	optionNameConfigGitUsername  = "config-git-username"
	optionNameEnableK8S          = "enable-k8s"
	optionNameGethURL            = "geth-url"
	optionNameInCluster          = "in-cluster"
	optionNameKubeconfig         = "kubeconfig"
	optionNameLogVerbosity       = "log-verbosity"
	optionNameLokiEndpoint       = "loki-endpoint"
	optionNameTracingEnabled     = "tracing-enable"
	optionNameTracingEndpoint    = "tracing-endpoint"
	optionNameTracingHost        = "tracing-host"
	optionNameTracingPort        = "tracing-port"
	optionNameTracingServiceName = "tracing-service-name"
)

var (
	errBlockchainEndpointNotProvided = errors.New("URL of the Ethereum-compatible blockchain RPC endpoint not provided; use the --geth-url flag")
	errMissingClusterName            = errors.New("cluster name not provided")
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
	config           *config.Config // beekeeper clusters configuration (config dir)
	httpClient       *http.Client
	k8sClient        *k8s.Client // kubernetes client
	swapClient       swap.Client
	log              logging.Logger
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
				return c.initConfig(cmd.Flags().Changed(optionNameClusterName))
			},
		},
		httpClient: &http.Client{
			Transport: &httpx.HeaderRoundTripper{
				Next: http.DefaultTransport,
			},
			Timeout: 3 * time.Minute,
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

	if err := c.initStamperCmd(); err != nil {
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

	if err := c.initNukeCmd(); err != nil {
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
	globalFlags.String(optionNameConfigGitBranch, "master", "Git branch to use for configuration files")
	globalFlags.String(optionNameConfigGitUsername, "", "Git username for authentication (required for private repositories)")
	globalFlags.String(optionNameConfigGitPassword, "", "Git password or personal access token for authentication (required for private repositories)")
	globalFlags.String(optionNameGethURL, "", "URL of the ethereum compatible blockchain RPC endpoint")
	globalFlags.String(optionNameLogVerbosity, "info", "Log verbosity level (0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=trace)")
	globalFlags.String(optionNameLokiEndpoint, "", "HTTP endpoint for sending logs to Loki (e.g., http://loki.testnet.internal/loki/api/v1/push)")
	globalFlags.Bool(optionNameTracingEnabled, false, "Enable tracing for performance monitoring and debugging")
	globalFlags.String(optionNameTracingEndpoint, "127.0.0.1:6831", "Endpoint for sending tracing data, specified as host:port")
	globalFlags.String(optionNameTracingHost, "", "Host address for sending tracing data")
	globalFlags.String(optionNameTracingPort, "", "Port for sending tracing data")
	globalFlags.String(optionNameTracingServiceName, "beekeeper", "Service name identifier used in tracing data")
	globalFlags.Bool(optionNameEnableK8S, true, "Enable Kubernetes client functionality")
	globalFlags.Bool(optionNameInCluster, false, "Use the in-cluster Kubernetes client")
	globalFlags.String(optionNameKubeconfig, "~/.kube/config", "Path to the kubeconfig file")
}

func (c *command) bindGlobalFlags() error {
	for _, flag := range []string{
		optionNameConfigDir,
		optionNameConfigGitBranch,
		optionNameConfigGitDir,
		optionNameConfigGitPassword,
		optionNameConfigGitRepo,
		optionNameConfigGitUsername,
		optionNameGethURL,
		optionNameLogVerbosity,
		optionNameLokiEndpoint,
	} {
		if err := c.globalConfig.BindPFlag(flag, c.root.PersistentFlags().Lookup(flag)); err != nil {
			return fmt.Errorf("binding %s flag: %w", flag, err)
		}
	}

	return nil
}

func (c *command) initConfig(loadConfigDir bool) error {
	if err := c.initGlobalConfig(); err != nil {
		return fmt.Errorf("initializing global configuration: %w", err)
	}

	if err := c.initLogger(); err != nil {
		return fmt.Errorf("initializing logger: %w", err)
	}

	if !loadConfigDir {
		c.log.Debugf("skpping loading configuration directory as the cluster name is not used")
		return nil
	}

	if err := c.loadConfigDirectory(); err != nil {
		return fmt.Errorf("loading configuration directory: %w", err)
	}

	return nil
}

func (c *command) initGlobalConfig() error {
	cfg := viper.New()
	cfgName := ".beekeeper"

	if c.globalConfigFile != "" {
		cfg.SetConfigFile(c.globalConfigFile)
	} else {
		cfg.AddConfigPath(c.homeDir)
		cfg.SetConfigName(cfgName)
	}

	cfg.SetEnvPrefix("beekeeper")
	cfg.AutomaticEnv()
	cfg.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	if c.homeDir != "" && c.globalConfigFile == "" {
		c.globalConfigFile = filepath.Join(c.homeDir, cfgName+".yaml")
	}

	if err := cfg.ReadInConfig(); err != nil {
		if !errors.As(err, &viper.ConfigFileNotFoundError{}) {
			return err
		}
	}

	c.globalConfig = cfg

	return c.bindGlobalFlags()
}

func (c *command) initLogger() error {
	verbosity := c.globalConfig.GetString(optionNameLogVerbosity)
	lokiEndpoint := c.globalConfig.GetString(optionNameLokiEndpoint)

	log, err := newLogger(c.root, verbosity, lokiEndpoint, c.httpClient)
	if err != nil {
		return fmt.Errorf("new logger: %w", err)
	}

	c.log = log
	return nil
}

func (c *command) loadConfigDirectory() error {
	if c.globalConfig.GetString(optionNameConfigGitRepo) != "" {
		c.log.Debugf("using configuration from Git repository %s, branch %s, directory %s", c.globalConfig.GetString(optionNameConfigGitRepo), c.globalConfig.GetString(optionNameConfigGitBranch), c.globalConfig.GetString(optionNameConfigGitDir))
		// read configuration from git repo
		fs := memfs.New()
		if _, err := git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
			Auth: &httptransport.BasicAuth{
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
		c.log.Debugf("using configuration from directory %s", c.globalConfig.GetString(optionNameConfigDir))
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

	return nil
}

func (c *command) setHomeDir() error {
	if c.homeDir != "" {
		return nil
	}

	dir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("obtaining user's home dir: %w", err)
	}

	c.homeDir = dir
	return nil
}

func (c *command) preRunE(cmd *cobra.Command, args []string) (err error) {
	if err := c.globalConfig.BindPFlags(cmd.Flags()); err != nil {
		return err
	}

	if err := c.setK8sClient(); err != nil {
		return err
	}

	if err := c.setSwapClient(); err != nil {
		return err
	}

	return nil
}

func (c *command) setK8sClient() error {
	if !c.globalConfig.GetBool(optionNameEnableK8S) {
		c.log.Info("Kubernetes client disabled. Enable it with --enable-k8s=true flag if required")
		return nil
	}

	c.log.Info("Kubernetes client enabled. Disable it with --enable-k8s=false flag if not required")

	options := []k8s.ClientOption{
		k8s.WithLogger(c.log),
		k8s.WithInCluster(c.globalConfig.GetBool(optionNameInCluster)),
		k8s.WithKubeconfigPath(c.globalConfig.GetString(optionNameKubeconfig)),
	}

	k8sClient, err := k8s.NewClient(options...)
	if err != nil && !errors.Is(err, k8s.ErrKubeconfigNotSet) {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	c.k8sClient = k8sClient
	return nil
}

func (c *command) executePeriodically(ctx context.Context, action func(ctx context.Context) error) error {
	periodicCheck := c.globalConfig.GetDuration(optionNamePeriodicCheck)

	if periodicCheck == 0 {
		return action(ctx)
	}

	periodicExecutor := scheduler.NewPeriodicExecutor(periodicCheck, c.log)

	periodicExecutor.Start(ctx, func(ctx context.Context) error {
		if err := action(ctx); err != nil {
			c.log.Errorf("failed to execute action periodically: %v", err)
		}
		return nil
	})
	defer func() {
		if err := periodicExecutor.Close(); err != nil {
			c.log.Errorf("failed to close periodic executor: %v", err)
		}
	}()

	<-ctx.Done()

	return ctx.Err()
}

func (c *command) createNodeClient(ctx context.Context, useDeploymentType bool) (*node.Client, error) {
	namespace := c.globalConfig.GetString(optionNameNamespace)
	clusterName := c.globalConfig.GetString(optionNameClusterName)

	if clusterName == "" && namespace == "" {
		return nil, errors.New("either cluster name or namespace must be provided")
	}

	if c.globalConfig.IsSet(optionNameNamespace) && namespace == "" {
		return nil, errors.New("namespace cannot be empty if set")
	}

	if namespace == "" && useDeploymentType && !isValidDeploymentType(c.globalConfig.GetString(optionNameDeploymentType)) {
		return nil, errors.New("unsupported deployment type: must be 'beekeeper' or 'helm'")
	}

	if useDeploymentType {
		c.log.Infof("using deployment type %s", c.globalConfig.GetString(optionNameDeploymentType))
	}

	var beeClients map[string]*bee.Client

	if clusterName != "" {
		cluster, err := c.setupCluster(ctx, clusterName, false)
		if err != nil {
			return nil, fmt.Errorf("setting up cluster %s: %w", clusterName, err)
		}

		beeClients, err = cluster.NodesClients(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve node clients: %w", err)
		}

		namespace = cluster.Namespace()
	}

	nodeClient := node.New(&node.ClientConfig{
		Log:            c.log,
		HTTPClient:     c.httpClient,
		K8sClient:      c.k8sClient,
		BeeClients:     beeClients,
		Namespace:      namespace,
		LabelSelector:  c.globalConfig.GetString(optionNameLabelSelector),
		DeploymentType: node.DeploymentType(c.globalConfig.GetString(optionNameDeploymentType)),
		InCluster:      c.globalConfig.GetBool(optionNameInCluster),
		UseNamespace:   c.globalConfig.IsSet(optionNameNamespace),
		NodeGroups:     c.globalConfig.GetStringSlice(optionNameNodeGroups),
	})

	return nodeClient, nil
}

func (c *command) setSwapClient() (err error) {
	if c.globalConfig.IsSet(optionNameGethURL) {
		gethUrl, err := url.Parse(c.globalConfig.GetString(optionNameGethURL))
		if err != nil {
			return fmt.Errorf("parsing Geth URL: %w", err)
		}

		c.swapClient = swap.NewGethClient(gethUrl, &swap.GethClientOptions{
			BzzTokenAddress: c.globalConfig.GetString("bzz-token-address"),
			EthAccount:      c.globalConfig.GetString("eth-account"),
			HTTPClient:      c.httpClient,
		}, c.log)
	} else {
		c.swapClient = &swap.NotSet{}
	}

	return err
}

func newLogger(cmd *cobra.Command, verbosity, lokiEndpoint string, httpClient *http.Client) (logging.Logger, error) {
	var logger logging.Logger
	opts := []logging.LoggerOption{
		logging.WithLokiOption(lokiEndpoint, httpClient),
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
