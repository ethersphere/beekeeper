package cmd

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/chaos"

	"github.com/spf13/cobra"
)

func (c *command) initTurnPodfailure(action string) *cobra.Command {
	return &cobra.Command{
		Use:   "podfailure",
		Short: "podfailure scenario",
		Long:  `podfailure scenario.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			ctx := context.Background()
			kubeconfig := c.config.GetString(optionNameKubeConfig)
			mode := c.config.GetString(optionNameMode)
			value := c.config.GetString(optionNameValue)
			namespace := c.config.GetString(optionNameChaosNamespace)
			podname := c.config.GetString(optionNamePodname)
			duration := c.config.GetString(optionNameDuration)
			cron := c.config.GetString(optionNameCron)

			err = chaos.CheckChaosMesh(ctx, kubeconfig, namespace)
			if err != nil {
				return err
			}

			err = chaos.PodFailure(ctx, kubeconfig, action, mode, value, namespace, podname, duration, cron)
			if err != nil {
				return err
			}
			if action == "create" {
				fmt.Printf("Turned on pod-failure-%s-%s\n", mode, podname)
			} else {
				fmt.Printf("Turned off pod-failure-%s-%s\n", mode, podname)
			}
			return
		},
		PreRunE: c.turnPreRunE,
	}
}
