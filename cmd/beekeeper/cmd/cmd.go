package cmd

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/swap"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	optionNameConfigDir         = "config-dir"
	optionNameConfigGitRepo     = "config-git-repo"
	optionNameConfigGitBranch   = "config-git-branch"
	optionNameConfigGitUsername = "config-git-username"
	optionNameConfigGitPassword = "config-git-password"
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
	globalFlags.String(optionNameConfigGitRepo, "", "Git repository with configurations (uses config directory when Git repo is not specified) (default \"\")")
	globalFlags.String(optionNameConfigGitBranch, "main", "Git branch")
	globalFlags.String(optionNameConfigGitUsername, "", "Git username (needed for private repos)")
	globalFlags.String(optionNameConfigGitPassword, "", "Git password or personal access tokens (needed for private repos)")
}

func (c *command) bindGlobalFlags() (err error) {
	for _, flag := range []string{optionNameConfigDir, optionNameConfigGitRepo, optionNameConfigGitBranch, optionNameConfigGitUsername, optionNameConfigGitPassword} {
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

		files, err := fs.ReadDir(".")
		if err != nil {
			return err
		}
		yamlFiles := [][]byte{}
		for _, file := range files {
			if file.IsDir() || (!strings.HasSuffix(file.Name(), ".yaml") && !strings.HasSuffix(file.Name(), ".yml")) {
				continue
			}
			f, err := fs.Open(file.Name())
			if err != nil {
				return fmt.Errorf("opening file %s: %w ", file.Name(), err)
			}
			defer f.Close()

			yamlFile := make([]byte, file.Size())
			if _, err := f.Read(yamlFile); err != nil {
				return fmt.Errorf("reading file %s: %w ", file.Name(), err)
			}
			yamlFiles = append(yamlFiles, yamlFile)
		}

		if c.config, err = config.Read(yamlFiles...); err != nil {
			return err
		}
	} else {
		// read configuration from directory
		files, err := os.ReadDir(c.globalConfig.GetString(optionNameConfigDir))
		if err != nil {
			return fmt.Errorf("reading config dir: %w", err)
		}

		yamlFiles := [][]byte{}
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
			yamlFiles = append(yamlFiles, yamlFile)
		}

		if c.config, err = config.Read(yamlFiles...); err != nil {
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
	if c.globalConfig.GetBool("enable-k8s") {
		funcs := k8s.ClientFunctions{
			NewForConfig:         kubernetes.NewForConfig,
			InClusterConfig:      rest.InClusterConfig,
			BuildConfigFromFlags: clientcmd.BuildConfigFromFlags,
			FlagString:           flag.String,
			FlagParse:            flag.Parse,
			OsUserHomeDir:        os.UserHomeDir,
		}
		if c.k8sClient, err = k8s.NewClient(funcs, &k8s.ClientOptions{
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
