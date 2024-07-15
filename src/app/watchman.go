package app

import (
	"fmt"
	"os"
	"path"

	"github.com/cdmistman/watchman"
	"github.com/cdmistman/watchman/protocol"
	"github.com/cdmistman/watchman/protocol/query"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
)

type watchmanState int

const (
	uninit watchmanState = iota
	initing
	errored
	ready
)

type Watchman struct {
	subscriptions []*WatchmanSub
}

// TODO: proper error handling
func (w *Watchman) MaybeStart() error {
	if len(w.subscriptions) == 0 {
		return nil
	}

	globs := []string{}
	for _, sub := range w.subscriptions {
		globs = append(globs, sub.config.Glob)
	}

	q := query.Query{
		Generators: query.Generators{query.GGlob: globs},
		Fields:     query.Fields{query.FName},
	}

	client, err := watchman.Connect()
	if err != nil {
		return err
	}

	if !client.HasCapability("glob_generator") {
		return fmt.Errorf("watchman does not have the glob_generator capability")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	watch, err := client.AddWatch(cwd)
	if err != nil {
		return err
	}

	if watch.RelativePath() != "" {
		if !client.HasCapability("relative_root") {
			return fmt.Errorf("watchman does not have the relative_root capability but gave a relative root")
		}

		q.RelativeRoot = watch.RelativePath()
	}

	if _, err = watch.Subscribe("process-compose", &q); err != nil {
		return err
	}

	go loop(client, w.subscriptions)
	return nil
}

// assumes that all subscriptions have already been created
func loop(client *watchman.Client, subs []*WatchmanSub) {
	for {
		rawNotif, ok := <-client.Notifications()
		if !ok {
			log.Warn().Msg("watchman client shut down, exiting watch loop")
			return
		}

		notif, ok := rawNotif.(*protocol.Subscription)
		if !ok {
			log.Debug().Msg("got a non-subscription notification, ignoring")
		}

		if notif.IsFreshInstance() {
			log.Debug().Msg("fresh instance notification, ignoring")
		}

		files := []string{}
		for _, rawFile := range notif.Files() {
			// if the query fields change, this will need to be a map[string]any
			file := rawFile.(string)
			files = append(files, file)
		}

		for _, sub := range subs {
			sub.updates <- files
		}
	}
}

func (w *Watchman) Subscribe(config *types.Watch) *WatchmanSub {
	ch := make(chan []string)
	w.subscriptions = append(w.subscriptions, &WatchmanSub{
		updates: ch,
		config:  config,
	})

	return &WatchmanSub{ch, config}
}

type WatchmanSub struct {
	updates chan []string
	config  *types.Watch
}

func (s *WatchmanSub) Recv() ([]string, bool) {
	for {
		files, ok := <-s.updates
		if !ok {
			return nil, false
		}

		matches := []string{}

	Files:
		for ii := 0; ii < len(files); ii++ {
			file := files[ii]

			matched, err := path.Match(s.config.Glob, file)
			if err != nil {
				// TODO: close the channel?
				log.Warn().
					Fields(map[string]any{
						"glob": s.config.Glob,
						"path": file,
					}).
					Err(err).
					Msg("invalid glob config, reporting as shut down")

				return nil, false
			} else if !matched {
				continue Files
			}

		Ignore:
			for jj := 0; jj < len(s.config.Ignore); jj++ {
				matched, err := path.Match(s.config.Ignore[jj], file)
				if err != nil {
					log.
						Warn().
						Fields(map[string]any{
							"glob":   s.config.Glob,
							"ignore": s.config.Ignore[jj],
							"path":   file,
						}).
						Err(err).
						Msg("invalid ignore config, removing")

					s.config.Ignore = append(s.config.Ignore[:jj], s.config.Ignore[jj+1:]...)
					jj -= 1
					continue Ignore
				}

				if matched {
					continue Files
				}
			}

			matches = append(matches, file)
		}

		if len(matches) > 0 {
			return matches, true
		}
	}
}
