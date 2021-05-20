package cmd

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/swap"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const optionNameConfigDir = "config-dir"

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
	globalFlags.StringVar(&c.globalConfigFile, "config", "", "config file (default is $HOME/.beekeeper.yaml)")
	globalFlags.String(optionNameConfigDir, filepath.Join(c.homeDir, "/.beekeeper/"), "config directory (default is $HOME/.beekeeper/)")
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

	// bind flag for configuration directory
	if err := cfg.BindPFlag(optionNameConfigDir, c.root.PersistentFlags().Lookup(optionNameConfigDir)); err != nil {
		return err
	}

	// read configuration directory
	c.config, err = config.ReadDir(c.globalConfig.GetString(optionNameConfigDir))
	if err != nil {
		return err
	}

	return nil
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
	if c.globalConfig.GetBool("enable-k8s") {
		if c.k8sClient, err = k8s.NewClient(&k8s.ClientOptions{
			InCluster:      c.globalConfig.GetBool("in-cluster"),
			KubeconfigPath: c.globalConfig.GetString("kubeconfig"),
		}); err != nil && err != k8s.ErrKubeconfigNotSet {
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
		})
	} else {
		c.swapClient = &swap.NotSet{}
	}

	return
}
