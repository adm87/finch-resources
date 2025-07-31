package main

import (
	"github.com/adm87/finch-resources/cmd/generate"
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use: "finch-resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.AddCommand(generate.ManifestCmd)

	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
