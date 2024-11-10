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

func WithTheme(theme string) Option {
	return func(view *pcView) error {
		view.themes.SelectStyles(theme)
		return nil
	}
}

func WithReadOnlyMode(isReadOnly bool) Option {
	return func(view *pcView) error {
		view.isReadOnlyMode = isReadOnly
		return nil
	}
}

func WithFullScreen(isFullScreen bool) Option {
	return func(view *pcView) error {
		view.setFullScreen(isFullScreen)
		return nil
	}
}

func WithDisabledHidden(isHidden bool) Option {
	return func(view *pcView) error {
		view.hideDisabled.Store(isHidden)
		return nil
	}
}

func WithDisabledExitConfirm(isDisabled bool) Option {
	return func(view *pcView) error {
		view.isExitConfirmDisabled = isDisabled
		return nil
	}
}
