package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	optionNameKubeConfig     = "kubeconfig"
	optionNameChaosNamespace = "namespace"
	optionNameMode           = "mode"
	optionNameValue          = "value"
	optionNameDuration       = "duration"
	optionNameCron           = "cron"
	optionNamePodname        = "podname"
	optionNameMode2          = "mode2"
	optionNameValue2         = "value2"
	optionNameDirection      = "direction"
	optionNameCorrelation    = "correlation"
	optionNameLoss           = "loss"
	optionNameLatency        = "latency"
	optionNameJitter         = "jitter"
	optionNameDuplicate      = "duplicate"
	optionNameCorrupt        = "corrupt"
)

var (
	action string
)

func (c *command) initTurnOnCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "turn-on",
		Short: "Turn on chaos scenario on a Bee cluster",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return cmd.Help()
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}
	viper.AutomaticEnv()
	cmd.PersistentFlags().String(optionNameKubeConfig, viper.GetString("KUBECONFIG"), "kubernetes config file")
	cmd.PersistentFlags().StringP(optionNameChaosNamespace, "n", "bee", "kubernetes namespace")
	cmd.PersistentFlags().String(optionNameMode, "one", "defines the mode to run chaos scenario [one|all|fixed|fixed-percent|random-max-percent]")
	cmd.PersistentFlags().String(optionNameValue, "", "depends on the mode, for one and all leave empty")
	cmd.PersistentFlags().String(optionNameDuration, "", "defines the duration for each chaos scenario [15s|5m|1h]")
	cmd.PersistentFlags().String(optionNameCron, "", "defines the scheduler rules for the running time of the chaos @every [15s|5m|1h]")
	cmd.PersistentFlags().String(optionNamePodname, "", "if empty it will use random bee pod from the namespace")
	cmd.PersistentFlags().String(optionNameMode2, "one", "same as mode, used only for networkPartition scenario")
	cmd.PersistentFlags().String(optionNameValue2, "", "same as value, used only for networkPartition scenario")
	cmd.PersistentFlags().String(optionNameDirection, "both", "specifies the partition direction, used only for networkPartition scenario [from|to|both]")
	cmd.PersistentFlags().String(optionNameCorrelation, "0", "correlation is used to emulate variation for networkChaos scenarios")
	cmd.PersistentFlags().String(optionNameLoss, "", "defines the percentage of packet loss, used only for networkLoss scenario")
	cmd.PersistentFlags().String(optionNameLatency, "", "defines the delay time in sending packets, used only for networkDelay scenario")
	cmd.PersistentFlags().String(optionNameJitter, "0ms", "specifies the jitter of the delay time")
	cmd.PersistentFlags().String(optionNameDuplicate, "", "indicates the percentage of packet duplication, used only for networkDuplicate scenario")
	cmd.PersistentFlags().String(optionNameCorrupt, "", "specifies the percentage of packet corruption, used only for networkCorrupt scenario")

	cmd.AddCommand(c.initTurnPodfailure("create"))
	cmd.AddCommand(c.initTurnPodkill("create"))

	c.root.AddCommand(cmd)
	return nil
}

func (c *command) initTurnOffCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "turn-off",
		Short: "Turn off chaos scenario on a Bee cluster",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return cmd.Help()
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}
	viper.AutomaticEnv()
	cmd.PersistentFlags().String(optionNameKubeConfig, viper.GetString("KUBECONFIG"), "kubernetes config file")
	cmd.PersistentFlags().StringP(optionNameChaosNamespace, "n", "bee", "kubernetes namespace")
	cmd.PersistentFlags().String(optionNameMode, "one", "defines the mode to run chaos scenario [one|all|fixed|fixed-percent|random-max-percent]")
	cmd.PersistentFlags().String(optionNameValue, "", "depends on the mode, for one and all leave empty")

	cmd.AddCommand(c.initTurnPodfailure("delete"))
	cmd.AddCommand(c.initTurnPodkill("delete"))

	c.root.AddCommand(cmd)
	return nil
}

func (c *command) turnPreRunE(cmd *cobra.Command, args []string) (err error) {
	if err = c.config.BindPFlags(cmd.Flags()); err != nil {
		return
	}

	return
}
