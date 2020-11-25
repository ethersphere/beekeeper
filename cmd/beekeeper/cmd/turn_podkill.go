package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/chaos"

	"github.com/spf13/cobra"
)

func (c *command) initTurnPodkill(action string) *cobra.Command {
	return &cobra.Command{
		Use:   "podkill",
		Short: "podkill scenario",
		Long:  `podkill scenario.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			kubeconfig := c.config.GetString(optionNameKubeConfig)
			mode := c.config.GetString(optionNameMode)
			value := c.config.GetString(optionNameValue)
			namespace := c.config.GetString(optionNameChaosNamespace)
			podname := c.config.GetString(optionNamePodname)
			cron := c.config.GetString(optionNameCron)

			ctx := cmd.Context()
			err = chaos.CheckChaosMesh(ctx, kubeconfig, namespace)
			if err != nil {
				return err
			}

			err = chaos.PodKill(ctx, kubeconfig, action, mode, value, namespace, podname, cron)
			if err != nil {
				return err
			}
			if action == "create" {
				fmt.Printf("Turned on pod-kill-%s-%s\n", mode, podname)
			} else {
				fmt.Printf("Turned off pod-kill-%s-%s\n", mode, podname)
			}
			return
		},
		PreRunE: c.turnPreRunE,
	}
}
