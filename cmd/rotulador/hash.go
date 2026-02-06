package main

import (
	"fmt"

	"github.com/lewtec/rotulador/annotation"
	"github.com/spf13/cobra"
)

var hashPasswordCmd = &cobra.Command{
	Use:   "hash-password [password]",
	Short: "Hash a password for the configuration file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		password := args[0]
		hash, err := annotation.HashPassword(password)
		if err != nil {
			return err
		}
		fmt.Println(hash)
		return nil
	},
}
