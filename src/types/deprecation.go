package types

import (
	"github.com/rs/zerolog/log"
	"os"
	"time"
)

const (
	month = time.Hour * 24 * 30
)

type DeprecationParams struct {
	StartTime time.Time
}

func deprecationHandler(start, proc, deprecated, new, scope string) {
	startTime, _ := time.Parse("2006-01-02", start)
	if time.Now().Before(startTime.Add(month)) {
		//month not passed since start
		log.Warn().Msgf("Process %s uses deprecated %s '%s' please change to '%s'", proc, scope, deprecated, new)
	} else if time.Now().Before(startTime.Add(2 * month)) {
		//2 months not passed
		log.Error().Msgf("Process %s uses deprecated %s '%s' please change to '%s'", proc, scope, deprecated, new)
		time.Sleep(5 * time.Second)
	} else {
		log.Error().Msgf("Process %s uses deprecated %s '%s' please change to '%s' exiting...", proc, scope, deprecated, new)
		os.Exit(1)
	}
}
