package xberg

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestXbergClientRequestsLayoutMarkdownWithPaddleOCR(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost || req.URL.Path != "/extract" {
			t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
		}

		// t.Fatal must not be called from the handler goroutine
		file, fileHeader, err := req.FormFile("files")
		if err != nil {
			t.Error(err)
			return
		}
		defer file.Close()
		fileContent, err := io.ReadAll(file)
		if err != nil {
			t.Error(err)
			return
		}
		if string(fileContent) != "file content" {
			t.Errorf("unexpected file content %q", fileContent)
		}
		if fileHeader.Filename != "scan.pdf" {
			t.Errorf("unexpected filename %q", fileHeader.Filename)
		}
		if contentType := fileHeader.Header.Get("Content-Type"); contentType != "application/pdf" {
			t.Errorf("unexpected content type %q", contentType)
		}
		expectedConfig := `{"output_format":"markdown","use_layout_for_markdown":true,` +
			`"layout":{"table_model":"tatr"},"ocr":{"backend":"paddle-ocr","language":"latin"}}`
		if config := req.FormValue("config"); config != expectedConfig {
			t.Errorf("unexpected config %q", config)
		}

		_, _ = rw.Write([]byte(`{"results":[{"content":"extracted"}],"errors":[]}`))
	}))
	defer server.Close()

	client, err := NewXbergClient(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}

	content, err := client.ExtractMarkdown(
		context.Background(),
		"scan.pdf",
		"application/pdf",
		strings.NewReader("file content"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if content != "extracted" {
		t.Fatalf("unexpected content %q", content)
	}
}

func TestXbergClientMapsUnsupportedFormatError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, _ = io.Copy(io.Discard, req.Body)
		_, _ = rw.Write([]byte(`{"results":[],"errors":[{"index":0,"code":1010,` +
			`"error_type":"unsupported_format","source":"file","message":"Unsupported format"}]}`))
	}))
	defer server.Close()

	client, err := NewXbergClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.ExtractMarkdown(context.Background(), "file", "", strings.NewReader("x"))
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("expected unsupported format error, got %v", err)
	}
}

func TestNewXbergClientRejectsInvalidURLs(t *testing.T) {
	for _, rawURL := range []string{
		"",
		"xberg:8000",
		"ftp://xberg:8000",
		"http://user:pass@xberg:8000",
		"http://xberg:8000?x=1",
	} {
		_, err := NewXbergClient(rawURL)
		if err == nil {
			t.Errorf("expected %q to be rejected", rawURL)
		}
	}
}
