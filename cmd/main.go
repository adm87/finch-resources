package main

import (
	"resources/cmd/generate"

	"github.com/spf13/cobra"
)

var root = &cobra.Command{
	Use: "resources",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func main() {
	root.AddCommand(generate.Generate())

	if err := root.Execute(); err != nil {
		panic(err)
	}
}
