package inbox

import (
	"testing"

	"github.com/simpledms/simpledms/model/main/common/filesource"
)

func TestFilesListPartialStateSources(t *testing.T) {
	state := &FilesListPartialState{SourceValues: []string{
		filesource.WebDAV.String(),
		filesource.URLImport.String(),
		filesource.WebDAV.String(),
	}}

	sources, err := state.sources()
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 2 {
		t.Fatalf("expected 2 unique sources, got %d", len(sources))
	}
	if sources[0] != filesource.WebDAV || sources[1] != filesource.URLImport {
		t.Fatalf("unexpected sources: %#v", sources)
	}
}

func TestFilesListPartialStateSourcesRejectsInvalidSource(t *testing.T) {
	state := &FilesListPartialState{SourceValues: []string{"not-a-source"}}
	if _, err := state.sources(); err == nil {
		t.Fatal("expected invalid source error")
	}
}
