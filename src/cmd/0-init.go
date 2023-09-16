package cmd

import "github.com/f1bonacc1/process-compose/src/config"

var pcFlags *config.Flags

func init() {
	pcFlags = config.NewFlags()
}
