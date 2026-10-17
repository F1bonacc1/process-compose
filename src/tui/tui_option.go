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
		// Mark the column in the header, as setTableSorter does when its
		// shortcut is pressed. The header starts out unmarked.
		if title, ok := view.procColumns[column]; ok {
			order := "[pink]↓[-:-:-]"
			if !isAscending {
				order = "[pink]↑[-:-:-]"
			}
			view.procTable.GetCell(0, int(column)).SetText(title + order)
		}
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

func WithDetachOnSuccess(detachOnSuccess bool) Option {
	return func(view *pcView) error {
		view.detachOnSuccess = detachOnSuccess
		return nil
	}
}

func LoadExtraShortCutsPaths(paths []string) Option {
	return func(view *pcView) error {
		for _, path := range paths {
			_ = view.shortcuts.loadFromFile(path)
		}
		view.recreateHelpDialog()
		return nil
	}
}
