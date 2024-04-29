package cmd

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/f1bonacc1/process-compose/src/tui"
	"github.com/spf13/pflag"
	"strings"
)

var pcFlags *config.Flags
var commonFlags *pflag.FlagSet

const (
	flagTheme   = "theme"
	flagReverse = "reverse"
	flagSort    = "sort"
)

func init() {
	pcFlags = config.NewFlags()
	commonFlags = pflag.NewFlagSet("", pflag.ContinueOnError)
	commonFlags.BoolVarP(pcFlags.IsReverseSort, flagReverse, "R", *pcFlags.IsReverseSort, "sort in reverse order")
	commonFlags.StringVarP(
		pcFlags.SortColumn,
		flagSort,
		"S",
		*pcFlags.SortColumn,
		fmt.Sprintf("sort column name. legal values (case insensitive): [%s]", strings.Join(tui.ColumnNames(), ", ")),
	)
	commonFlags.StringVar(pcFlags.PcTheme, flagTheme, *pcFlags.PcTheme, "select process compose theme")
}
