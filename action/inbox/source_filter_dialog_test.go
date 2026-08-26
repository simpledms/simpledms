package inbox

import (
	"testing"

	"github.com/simpledms/simpledms/model/main/common/filesource"
)

func TestSourceFilterDialogChipsReflectSelection(t *testing.T) {
	state := &FilesListPartialState{
		SourceValues: []string{filesource.WebDAV.String()},
	}
	chips := new(SourceFilterDialog).chips(state)

	if len(chips) != len(filesource.Values()) {
		t.Fatalf("expected %d source chips, got %d", len(filesource.Values()), len(chips))
	}
	for _, chip := range chips {
		if chip.Value == filesource.WebDAV.String() && !chip.IsChecked {
			t.Fatal("expected WebDAV source chip to be checked")
		}
	}
}
