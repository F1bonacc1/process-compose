package tui

import "testing"

// The sort column from the settings file or the command line must be marked
// in the header from the start, not only after its shortcut is pressed (#547).
func TestWithStateSorter_MarksSortColumnInHeader(t *testing.T) {
	pv := &pcView{}
	pv.procTable = pv.createProcTable()

	if err := WithStateSorter(ProcessStateCPU, false)(pv); err != nil {
		t.Fatal(err)
	}

	for id, title := range pv.procColumns {
		want := title
		if id == ProcessStateCPU {
			want = title + "[pink]↑[-:-:-]"
		}
		if got := pv.procTable.GetCell(0, int(id)).Text; got != want {
			t.Errorf("header of %s = %q, want %q", id, got, want)
		}
	}
}
