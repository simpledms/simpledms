package server

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/entx"
)

func TestMCPReadFileTextPrefersXbergMarkdownOverTikaText(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixture(t, h, "markdown")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := f.connect(t, server.URL+"/mcp")
	const markdown = "| Product | Units |\n| --- | --- |\n| Basic | 1,204 |"

	expectText := func(wantText string, wantFormat string) {
		t.Helper()
		text := callMCP(t, client, "read_file_text", map[string]any{"file_id": f.fileID})
		if text["available"] != true || text["text"] != wantText || text["format"] != wantFormat {
			t.Fatalf("want %s %q, got %v", wantFormat, wantText, text)
		}
	}

	expectText("ä🙂abcdef", "text")

	updateMCPFixtureFile(t, f, func(update *enttenant.FileUpdate) {
		update.SetOcrMarkdown(markdown).SetOcrMarkdownSuccessAt(time.Now())
	})
	expectText(markdown, "markdown")

	// empty markdown, for example for formats Xberg does not support
	updateMCPFixtureFile(t, f, func(update *enttenant.FileUpdate) {
		update.SetOcrMarkdown("")
	})
	expectText("ä🙂abcdef", "text")

	// Xberg can finish before Tika
	updateMCPFixtureFile(t, f, func(update *enttenant.FileUpdate) {
		update.SetOcrMarkdown(markdown).ClearOcrSuccessAt()
	})
	expectText(markdown, "markdown")
	fileData := callMCP(t, client, "get_file", map[string]any{"file_id": f.fileID})
	if fileData["ocr_available"] != true {
		t.Fatalf("markdown-only file reported without OCR: %v", fileData)
	}
}

func updateMCPFixtureFile(t *testing.T, f *mcpFixture, apply func(*enttenant.FileUpdate)) {
	t.Helper()
	if err := withTenantContext(t, f.h, f.account, f.tenant, f.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		sc := ctxx.NewSpaceContext(tc, tc.TTx.Space.Query().Where(
			space.PublicID(entx.NewCIText(f.spaceID)),
		).OnlyX(tc))
		update := sc.TTx.File.Update().Where(file.PublicID(entx.NewCIText(f.fileID)))
		apply(update)
		return update.Exec(sc)
	}); err != nil {
		t.Fatal(err)
	}
}
