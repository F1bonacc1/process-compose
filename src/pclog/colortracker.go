package pclog

import (
	"github.com/fatih/color"
	"hash/fnv"
	"sync"
)

type ColorTracker struct {
	clrMtx    sync.Mutex
	colors    map[string]func(a ...interface{}) string
	maxColors int
}

func NewColorTracker() *ColorTracker {
	return &ColorTracker{
		colors:    map[string]func(a ...interface{}) string{},
		maxColors: int(color.FgHiWhite) - int(color.FgHiBlack),
	}
}

// GetColor returns the color for the given name.
func (c *ColorTracker) GetColor(name string) func(a ...interface{}) string {
	c.clrMtx.Lock()
	defer c.clrMtx.Unlock()
	if fn, ok := c.colors[name]; ok {
		return fn
	}
	fn := color.New(color.FgHiBlack+color.Attribute(stringToInt(name)%c.maxColors), color.Bold).SprintFunc()
	c.colors[name] = fn
	return fn
}

// Name2Color returns the color for the given name.
func Name2Color(name string) func(a ...interface{}) string {
	maxColors := int(color.FgHiWhite) - int(color.FgHiBlack)
	return color.New(color.FgHiBlack+color.Attribute(stringToInt(name)%maxColors), color.Bold).SprintFunc()
}

func stringToInt(s string) int {
	// Create a hash of the string using FNV-1a algorithm
	hash := fnv.New32a()
	hash.Write([]byte(s))

	// Convert the hash to an integer
	return int(hash.Sum32())
}
