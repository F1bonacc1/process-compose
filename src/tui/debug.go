package tui

import (
	"os"
)

func (pv *pcView) dumpStyles() {
	f, _ := os.Create("/tmp/styles.yaml")
	defer f.Close()
	pv.styles.Dump(f)
}
