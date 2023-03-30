package main

import (
	"context"
	"log"

	"github.com/spf13/cobra"

	"github.com/ftl/clusterix"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Connect to the given DX cluster host and log the incoming messages to stdout.",
	Run:   runWithClient(monitor),
}

func init() {
	rootCmd.AddCommand(monitorCmd)
}

func monitor(ctx context.Context, c *clusterix.Client, _ *cobra.Command, _ []string) {
	c.Notify(new(dxMonitor))
	<-ctx.Done()
}

type dxMonitor struct{}

func (m *dxMonitor) DX(msg clusterix.DXMessage) {
	log.Printf("%+v", msg)
}
