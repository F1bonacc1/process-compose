package tui

import "time"

type Option func(view *pcView) error

func WithRefreshRate(rate time.Duration) Option {
	return func(view *pcView) error {
		view.refreshRate = rate
		return nil
	}
}

func WithStateSorter(column ColumnID, isAscending bool) Option {
	return func(view *pcView) error {
		view.stateSorter = StateSorter{sortByColumn: column, isAsc: isAscending}
		return nil
	}
}
