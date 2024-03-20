package tui

import (
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
	"slices"
)

func (pv *pcView) showNsFilter() {
	filter := pv.createNsFilterPrimitive()
	_, _, w, h := filter.GetRect()
	pv.pages.AddPage(PageDialog, createDialogPage(filter, w, h), true, true)
	pv.appView.SetFocus(filter)
}

func (pv *pcView) createNsFilterPrimitive() tview.Primitive {
	selectAllNsLbl := "Select all the namespaces"
	list := tview.NewList().
		AddItem("All", selectAllNsLbl, '0', func() {
			pv.setSelectedNs(AllNS)
			pv.pages.RemovePage(PageDialog)
		})
	nsList, err := pv.getSortedNsList()
	if err != nil {
		log.Err(err).Msg("failed to get sorted ns list")
	}
	maxLblLen := len(selectAllNsLbl)
	const scList = "123456789abcdefghijklmnopqrstuvwyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	for i, ns := range nsList {
		nsToAdd := ns
		selectNsLbl := "Select " + ns
		r := '0'
		if i < len(scList) {
			r = rune(scList[i])
		}
		list.AddItem(ns, selectNsLbl, r, func() {
			pv.setSelectedNs(nsToAdd)
			pv.pages.RemovePage(PageDialog)
		})
		if len(selectNsLbl) > maxLblLen {
			maxLblLen = len(selectNsLbl)
		}
	}
	list.AddItem("Cancel", "Select to close", 'x', func() {
		pv.pages.RemovePage(PageDialog)
	})
	list.SetBorder(true).SetTitle("Namespaces")

	flex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(list, (len(nsList)+3)*2, 1, true), maxLblLen+10, 1, true).
		AddItem(nil, 0, 1, false)
	list.SetBackgroundColor(pv.styles.BgColor())
	return flex
}

func (pv *pcView) getSortedNsList() ([]string, error) {
	states, err := pv.project.GetProcessesState()
	if err != nil {
		log.Err(err).Msg("failed to get processes state")
		return []string{}, err
	}
	nsList := make(map[string]struct{})
	for _, state := range states.States {
		nsList[state.Namespace] = struct{}{}
	}
	var nsListSorted []string
	for ns := range nsList {
		nsListSorted = append(nsListSorted, ns)
	}
	slices.Sort(nsListSorted)
	return nsListSorted, nil
}

func (pv *pcView) isNsSelected(ns string) bool {
	pv.selectedNsMtx.Lock()
	defer pv.selectedNsMtx.Unlock()
	if pv.selectedNs == AllNS || pv.selectedNs == ns {
		return true
	}

	return false
}

func (pv *pcView) setSelectedNs(ns string) {
	pv.selectedNsMtx.Lock()
	defer pv.selectedNsMtx.Unlock()
	pv.selectedNsChanged.Store(pv.selectedNs != ns)
	pv.selectedNs = ns
}

func (pv *pcView) getSelectedNs() string {
	pv.selectedNsMtx.Lock()
	defer pv.selectedNsMtx.Unlock()
	return pv.selectedNs
}
