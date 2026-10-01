package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/entx"
	credentialmodel "github.com/simpledms/simpledms/model/main/mcpcredential"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	storedfilemodel "github.com/simpledms/simpledms/model/tenant/storedfile"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/httpx"
)

func TestMCPDownloadBoundedContinuationPinsHistoricalVersion(t *testing.T) {
	h := newActionTestHarnessWithS3(t)
	f := newMCPFixtureWithWrites(t, h, "download-version-pin")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := f.connect(t, server.URL+"/mcp")

	original := bytes.Repeat([]byte("old-version-"), 100*1024)
	upload := callMCP(t, client, "upload_file", map[string]any{
		"filename": "version-pin.bin", "content_base64": base64.StdEncoding.EncodeToString(original),
	})
	fileID := upload["file_id"].(string)
	oldVersion := int(callMCP(t, client, "get_file", map[string]any{"file_id": fileID})["version_number"].(float64))
	first := callMCP(t, client, "download_file", map[string]any{
		"file_id": fileID, "version_number": oldVersion, "offset": int64(0), "length": 1024 * 1024,
	})
	firstChunk, err := base64.StdEncoding.DecodeString(first["content_base64"].(string))
	if err != nil {
		t.Fatal(err)
	}

	newBytes := bytes.Repeat([]byte("new-version-"), 70*1024)
	mainTx, tenantTx, tenantCtx, err := newTenantContextForUpload(h, f.account, f.tenant, f.db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = mainTx.Rollback(); _ = tenantTx.Rollback() }()
	spaceCtx := ctxx.NewSpaceContext(tenantCtx, tenantCtx.TTx.Space.Query().Where(
		space.PublicID(entx.NewCIText(f.spaceID)),
	).OnlyX(tenantCtx))
	var body bytes.Buffer
	writer := multipartWriter(t, &body, fileID, "version-pin-new.bin", newBytes)
	req := httptest.NewRequest(http.MethodPost, "/upload", &body)
	req.Header.Set("Content-Type", writer)
	if err := h.actions.Browse.UploadFileVersionCmd.Handler(
		httpx.NewResponseWriter(httptest.NewRecorder()), httpx.NewRequest(req), spaceCtx,
	); err != nil {
		t.Fatalf("upload new version: %v", err)
	}

	downloaded := append([]byte(nil), firstChunk...)
	offset := int64(first["next_offset"].(float64))
	for {
		result := callMCP(t, client, "download_file", map[string]any{
			"file_id": fileID, "version_number": oldVersion, "offset": offset, "length": 1024 * 1024,
		})
		chunk, err := base64.StdEncoding.DecodeString(result["content_base64"].(string))
		if err != nil {
			t.Fatal(err)
		}
		downloaded = append(downloaded, chunk...)
		if !result["has_more"].(bool) {
			break
		}
		offset = int64(result["next_offset"].(float64))
	}
	if !bytes.Equal(downloaded, original) {
		t.Fatalf("historical continuation returned %d bytes from the latest version", len(downloaded))
	}
	latest := callMCP(t, client, "download_file", map[string]any{
		"file_id": fileID, "length": len(newBytes),
	})
	got, err := base64.StdEncoding.DecodeString(latest["content_base64"].(string))
	if err != nil || !bytes.Equal(got, newBytes) {
		t.Fatalf("latest version bytes mismatch: err=%v", err)
	}
}

func TestMCPDownloadReportsMissingOnlyObjectAndKeepsMetadata(t *testing.T) {
	s3 := newTestS3Config(t)
	h := newActionTestHarnessWithSaaSAndS3Config(t, true, s3)
	f := newMCPFixtureWithWrites(t, h, "download-storage-error")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := f.connect(t, server.URL+"/mcp")
	upload := callMCP(t, client, "upload_file", map[string]any{
		"filename": "disposable-storage-error.txt", "content_base64": base64.StdEncoding.EncodeToString([]byte("disposable")),
	})
	fileID := upload["file_id"].(string)
	var bucket, object string
	if err := withTenantContext(t, h, f.account, f.tenant, f.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		sc := ctxx.NewSpaceContext(tc, tc.TTx.Space.Query().Where(
			space.PublicID(entx.NewCIText(f.spaceID)),
		).OnlyX(tc))
		version := sc.TTx.FileVersion.Query().Where(
			fileversion.FileID(sc.TTx.File.Query().Where(file.PublicID(entx.NewCIText(fileID))).OnlyX(sc).ID),
		).WithStoredFile().OnlyX(sc)
		bucket = version.Edges.StoredFile.BucketName
		var err error
		object, err = storedfilemodel.NewStoredFile(version.Edges.StoredFile).ObjectNameWithPrefix()
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := s3.client.RemoveObject(context.Background(), bucket, object, minio.RemoveObjectOptions{}); err != nil {
		t.Fatal(err)
	}
	assertMCPToolError(t, client, "download_file", map[string]any{"file_id": fileID, "length": 32})
	metadata := callMCP(t, client, "get_file", map[string]any{"file_id": fileID})
	if metadata["file_id"] != fileID || metadata["version_number"] == nil {
		t.Fatalf("storage failure changed file metadata: %v", metadata)
	}
}

func TestMCPDownloadCancellationDoesNotCommitPartialChunk(t *testing.T) {
	s3 := newTestS3Config(t)
	armed := new(atomic.Bool)
	entered := make(chan struct{})
	var enteredOnce sync.Once
	baseTransport := http.DefaultTransport
	endpoint := envOrDefault("SIMPLEDMS_S3_ENDPOINT", "localhost:7070")
	useSSL := envOrDefaultBool("SIMPLEDMS_S3_USE_SSL", false)
	accessKey := envOrDefault("SIMPLEDMS_S3_ACCESS_KEY_ID", "unsafe-placeholder-access-key-id")
	secretKey := envOrDefault("SIMPLEDMS_S3_SECRET_ACCESS_KEY", "unsafe-placeholder-secret-access-key")
	wrapped := mcpRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestPath := req.URL.Path
		isObjectGet := req.Method == http.MethodGet &&
			strings.Contains(requestPath, "/"+s3.bucketName+"/")
		if armed.Load() && isObjectGet {
			enteredOnce.Do(func() { close(entered) })
			<-req.Context().Done()
			return nil, req.Context().Err()
		}
		return baseTransport.RoundTrip(req)
	})
	client, err := minio.New(endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(accessKey, secretKey, ""), Secure: useSSL, Transport: wrapped,
	})
	if err != nil {
		t.Fatal(err)
	}
	s3.client = client
	h := newActionTestHarnessWithSaaSAndS3Config(t, true, s3)
	f := newMCPFixtureWithWrites(t, h, "download-cancel")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	session := f.connect(t, server.URL+"/mcp")
	content := bytes.Repeat([]byte("cancel-me-"), 100*1024)
	upload := callMCP(t, session, "upload_file", map[string]any{
		"filename": "cancel-me.bin", "content_base64": base64.StdEncoding.EncodeToString(content),
	})
	fileID := upload["file_id"].(string)
	if got := callMCP(t, session, "download_file", map[string]any{"file_id": fileID, "length": 32}); got["file_id"] != fileID {
		t.Fatalf("setup download metadata: %v", got)
	}

	armed.Store(true)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	resultCh := make(chan struct {
		result *sdk.CallToolResult
		err    error
	}, 1)
	go func() {
		result, err := session.CallTool(ctx, &sdk.CallToolParams{
			Name: "download_file", Arguments: map[string]any{"file_id": fileID, "length": 1024 * 1024},
		})
		resultCh <- struct {
			result *sdk.CallToolResult
			err    error
		}{result, err}
	}()
	select {
	case <-entered:
		cancel()
	case <-time.After(5 * time.Second):
		t.Fatal("download transport was not entered")
	}
	select {
	case outcome := <-resultCh:
		if outcome.err == nil && outcome.result != nil && !outcome.result.IsError {
			t.Fatal("cancelled download returned a successful chunk")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancelled download did not return")
	}
	armed.Store(false)
	fresh := callMCP(t, session, "download_file", map[string]any{"file_id": fileID, "length": 32})
	if fresh["file_id"] != fileID || fresh["content_base64"] == "" {
		t.Fatalf("fresh download after cancellation failed: %v", fresh)
	}
}

func TestMCPNotesInTrashReadOnlyAndConcurrentReplaceIsSingleSuccessor(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixtureWithWrites(t, h, "notes-trash-concurrent")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := f.connect(t, server.URL+"/mcp")
	note := callMCP(t, client, "create_document_note", map[string]any{
		"file_id": f.fileID, "title": "Trash readable", "body": "preserved",
	})
	trash := f.browserAt(
		route.Inbox(f.tenant.PublicID.String(), f.spaceID, f.fileID),
		h.actions.Browse.DeleteFileCmd.Endpoint(), map[string][]string{"FileID": {f.fileID}},
	)
	if trash.Code != http.StatusOK {
		t.Fatalf("move file to trash: %d %s", trash.Code, trash.Body.String())
	}
	read := callMCP(t, client, "get_document_note", map[string]any{
		"file_id": f.fileID, "note_id": note["note_id"],
	})
	if read["body"] != "preserved" {
		t.Fatalf("trash note was not readable: %v", read)
	}
	assertMCPToolErrorCode(t, client, "replace_document_note", map[string]any{
		"file_id": f.fileID, "note_id": note["note_id"], "title": "denied", "body": "denied",
	}, "forbidden", "Trash")

	otherFile := fixtureFileID(t, f)
	original := callMCP(t, client, "create_document_note", map[string]any{
		"file_id": otherFile, "title": "Predecessor", "body": "original",
	})
	first, second := f.connect(t, server.URL+"/mcp"), f.connect(t, server.URL+"/mcp")
	type outcome struct {
		result *sdk.CallToolResult
		err    error
	}
	outcomes := make(chan outcome, 2)
	start := make(chan struct{})
	var group sync.WaitGroup
	for i, session := range []*sdk.ClientSession{first, second} {
		group.Add(1)
		go func(i int, session *sdk.ClientSession) {
			defer group.Done()
			<-start
			result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "replace_document_note", Arguments: map[string]any{
				"file_id": otherFile, "note_id": original["note_id"], "title": fmt.Sprintf("winner %d", i), "body": fmt.Sprintf("winner %d", i),
			}})
			outcomes <- outcome{result, err}
		}(i, session)
	}
	close(start)
	group.Wait()
	close(outcomes)
	successes := 0
	for result := range outcomes {
		if result.err == nil && result.result != nil && !result.result.IsError {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent replacements succeeded %d times", successes)
	}
	history := callMCP(t, client, "list_document_notes", map[string]any{
		"file_id": otherFile, "show_history": true, "limit": 50,
	})
	if len(history["notes"].([]any)) != 2 {
		t.Fatalf("concurrent replacement produced unexpected history: %v", history)
	}
	for _, item := range history["notes"].([]any) {
		if item.(map[string]any)["note_id"] == "" {
			t.Fatal("history returned an empty public note ID")
		}
	}
}

func TestMCPNoteReplacementCommitErrorLeavesNoHistoryOrOrphan(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixtureWithWrites(t, h, "notes-commit-error")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := f.connect(t, server.URL+"/mcp")
	original := callMCP(t, client, "create_document_note", map[string]any{
		"file_id": f.fileID, "title": "Original", "body": "unchanged",
	})
	service := credentialmodel.NewCredentialService()
	ok, err := service.ExecuteWrite(context.Background(), h.mainDB, h.tenantDBs, h.i18n, false, f.token,
		func(spaceCtx *ctxx.SpaceContext, _ *entmain.MCPCredential) error {
			spaceCtx.TTx.OnCommit(func(enttenant.Committer) enttenant.Committer {
				return enttenant.CommitFunc(func(context.Context, *enttenant.Tx) error {
					return errors.New("injected note commit failure")
				})
			})
			_, err := filemodel.NewDocumentNotes().Replace(
				spaceCtx, f.fileID, original["note_id"].(string), "Successor", "orphan",
			)
			return err
		})
	if ok || err == nil {
		t.Fatalf("replacement unexpectedly committed: ok=%t err=%v", ok, err)
	}
	history := callMCP(t, client, "list_document_notes", map[string]any{
		"file_id": f.fileID, "show_history": true, "limit": 50,
	})
	notes := history["notes"].([]any)
	if len(notes) != 1 || notes[0].(map[string]any)["note_id"] != original["note_id"] ||
		notes[0].(map[string]any)["body"] != "unchanged" {
		t.Fatalf("failed replacement left history or orphan: %v", history)
	}
}

func TestMCPMoveRollsBackCreatedChildOnDescendantError(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixtureWithWrites(t, h, "move-child-rollback")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := f.connect(t, server.URL+"/mcp")
	a := stringField(t, callMCP(t, client, "create_directory", map[string]any{"parent_directory_id": f.rootID, "name": "A"}), "directory_id")
	b := stringField(t, callMCP(t, client, "create_directory", map[string]any{"parent_directory_id": a, "name": "B"}), "directory_id")
	assertMCPToolError(t, client, "move_file", map[string]any{
		"file_id": a, "destination_directory_id": b, "filename": "A", "new_directory_name": "created-then-rejected",
	})
	if containsMCPChild(t, callMCP(t, client, "list_directory", map[string]any{"directory_id": b})["children"], "", "created-then-rejected") {
		t.Fatal("descendant move left its newly created child directory behind")
	}
	if !containsMCPChild(t, callMCP(t, client, "list_directory", map[string]any{"directory_id": a})["children"], b, "B") {
		t.Fatal("failed descendant move damaged the source subtree")
	}
}

func TestMCPOrganizationNonFolderRenamePreservesStateAndRejectsMove(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixtureWithWrites(t, h, "non-folder-organization-verification")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := f.connect(t, server.URL+"/mcp")
	callMCP(t, client, "mark_inbox_file_done", map[string]any{"file_id": f.fileID})
	if err := withTenantContext(t, h, f.account, f.tenant, f.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		return tc.TTx.Space.Update().Where(
			space.PublicID(entx.NewCIText(f.spaceID)),
		).SetIsFolderMode(false).Exec(tc)
	}); err != nil {
		t.Fatal(err)
	}
	before := callMCP(t, client, "get_file", map[string]any{"file_id": f.fileID})
	renamed := callMCP(t, client, "rename_file", map[string]any{
		"file_id": f.fileID, "new_filename": "non-folder-renamed.txt",
	})
	if renamed["file_id"] != f.fileID || renamed["parent_id"] != f.rootID ||
		renamed["is_in_inbox"] != false {
		t.Fatalf("non-folder rename changed lifecycle/location: %v", renamed)
	}
	assertMCPToolErrorCode(t, client, "move_file", map[string]any{
		"file_id": f.fileID, "destination_directory_id": f.rootID,
	}, "invalid_input", "Folder mode")
	after := callMCP(t, client, "get_file", map[string]any{"file_id": f.fileID})
	if after["name"] != "non-folder-renamed.txt" || after["parent_id"] != before["parent_id"] ||
		after["source"] != before["source"] || after["version_number"] != before["version_number"] {
		t.Fatalf("non-folder organization changed preserved state: %v", after)
	}
}

func TestMCPMoveCreatesChildOfCurrentParent(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixtureWithWrites(t, h, "move-current-parent-child")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := f.connect(t, server.URL+"/mcp")
	callMCP(t, client, "mark_inbox_file_done", map[string]any{"file_id": f.fileID})
	before := callMCP(t, client, "get_file", map[string]any{"file_id": f.fileID})
	moved := callMCP(t, client, "move_file", map[string]any{
		"file_id":                  f.fileID,
		"destination_directory_id": before["parent_id"],
		"new_directory_name":       "child-of-current-parent",
	})
	if moved["file_id"] != f.fileID || moved["parent_id"] == before["parent_id"] ||
		moved["is_in_inbox"] != false || moved["name"] != before["name"] {
		t.Fatalf("same-parent child creation did not move the filed entry: %v", moved)
	}
	children := callMCP(t, client, "list_directory", map[string]any{
		"directory_id": before["parent_id"],
	})["children"]
	if !containsMCPChild(t, children, moved["parent_id"].(string), "child-of-current-parent") {
		t.Fatalf("new child was not created below the current parent: %v", children)
	}
}

func TestMCPMetadataBrowserCreateAndAssignReflectsThroughMCP(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixtureWithWrites(t, h, "browser-create-assign-verification")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := f.connect(t, server.URL+"/mcp")
	response := f.browserAt(
		route.Inbox(f.tenant.PublicID.String(), f.spaceID, f.fileID),
		h.actions.Browse.Tagging.AssignedTags.CreateAndAssignTagCmd.Endpoint(),
		url.Values{
			"FileID": {f.fileID},
			"Name":   {"Created through browser"},
			"Type":   {"Simple"},
		},
	)
	if response.Code != http.StatusOK {
		t.Fatalf("shared browser create-and-assign failed: %d %s", response.Code, response.Body.String())
	}
	fileData := callMCP(t, client, "get_file", map[string]any{"file_id": f.fileID})
	assigned := fileData["direct_tags"].([]any)
	if len(assigned) != 1 || assigned[0].(map[string]any)["name"] != "Created through browser" ||
		assigned[0].(map[string]any)["tag_id"] == "" {
		t.Fatalf("browser-created assignment was not visible through MCP: %v", fileData)
	}
}

func multipartWriter(t *testing.T, body *bytes.Buffer, fileID, filename string, content []byte) string {
	t.Helper()
	w := multipart.NewWriter(body)
	if err := w.WriteField("FileID", fileID); err != nil {
		t.Fatal(err)
	}
	part, err := w.CreateFormFile("File", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return w.FormDataContentType()
}
