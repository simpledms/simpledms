package gotenberg

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/model/tenant/previewconversion"
)

func TestGotenbergClientBuildsHTMLMultipart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path != previewconversion.HTMLRoute {
			t.Fatalf("path = %q", req.URL.Path)
		}
		if got := req.Header.Get("Gotenberg-Output-Filename"); got != "report.pdf" {
			t.Fatalf("output filename = %q", got)
		}
		reader, err := req.MultipartReader()
		if err != nil {
			t.Fatal(err)
		}
		part, err := reader.NextPart()
		if err != nil {
			t.Fatal(err)
		}
		if part.FileName() != "index.html" {
			t.Fatalf("input filename = %q", part.FileName())
		}
		body, err := io.ReadAll(part)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "<h1>Report</h1>" {
			t.Fatalf("body = %q", body)
		}
		rw.Header().Set("Content-Type", "application/pdf")
		_, _ = rw.Write([]byte("%PDF-1.7\n"))
	}))
	defer server.Close()

	client, err := NewGotenbergClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	classification, eligible := previewconversion.Classify("text/html", "report.html", false)
	if !eligible {
		t.Fatal("expected eligible classification")
	}
	pdf, err := client.Convert(context.Background(), classification, strings.NewReader("<h1>Report</h1>"))
	if err != nil {
		t.Fatal(err)
	}
	defer pdf.Close()
	data, err := io.ReadAll(pdf)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "%PDF-1.7\n" {
		t.Fatalf("PDF = %q", data)
	}
}

func TestGotenbergClientBuildsMarkdownMultipart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		reader, err := req.MultipartReader()
		if err != nil {
			t.Fatal(err)
		}
		filenames := make([]string, 0, 2)
		bodies := make([]string, 0, 2)
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			filenames = append(filenames, part.FileName())
			body, err := io.ReadAll(part)
			if err != nil {
				t.Fatal(err)
			}
			bodies = append(bodies, string(body))
		}
		if len(filenames) != 2 || filenames[0] != "index.html" || filenames[1] != "source.md" {
			t.Fatalf("filenames = %#v", filenames)
		}
		if !strings.Contains(bodies[0], `{{ toHTML "source.md" }}`) {
			t.Fatalf("markdown template = %q", bodies[0])
		}
		if bodies[1] != "# Heading" {
			t.Fatalf("markdown source = %q", bodies[1])
		}
		rw.Header().Set("Content-Type", "application/pdf; charset=binary")
		_, _ = rw.Write([]byte("%PDF-1.7\n"))
	}))
	defer server.Close()

	client, err := NewGotenbergClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	classification, eligible := previewconversion.Classify("text/markdown", "README.md", false)
	if !eligible {
		t.Fatal("expected eligible classification")
	}
	pdf, err := client.Convert(context.Background(), classification, strings.NewReader("# Heading"))
	if err != nil {
		t.Fatal(err)
	}
	defer pdf.Close()
	if _, err := io.ReadAll(pdf); err != nil {
		t.Fatal(err)
	}
}

func TestGotenbergClientClassifiesResponses(t *testing.T) {
	for _, test := range []struct {
		name      string
		status    int
		content   string
		wantRetry bool
	}{
		{name: "service unavailable", status: http.StatusServiceUnavailable, wantRetry: true},
		{name: "server error", status: http.StatusInternalServerError, wantRetry: true},
		{name: "rejected", status: http.StatusBadRequest, wantRetry: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				rw.WriteHeader(test.status)
				_, _ = rw.Write([]byte(test.content))
			}))
			defer server.Close()

			client, err := NewGotenbergClient(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			classification, _ := previewconversion.Classify("text/html", "page.html", false)
			pdf, err := client.Convert(context.Background(), classification, strings.NewReader("page"))
			if pdf != nil {
				_ = pdf.Close()
			}
			if err == nil {
				t.Fatal("expected conversion error")
			}
			conversionErr, ok := err.(*ConversionError)
			if !ok {
				t.Fatalf("error type = %T", err)
			}
			if conversionErr.Retryable != test.wantRetry {
				t.Fatalf("retryable = %t, want %t", conversionErr.Retryable, test.wantRetry)
			}
		})
	}
}

func TestNewGotenbergClientRejectsUnsafeURLs(t *testing.T) {
	for _, rawURL := range []string{"", "localhost:3000", "ftp://localhost:3000", "http://user@localhost:3000", "http://localhost:3000?token=secret"} {
		if _, err := NewGotenbergClient(rawURL); err == nil {
			t.Fatalf("URL %q was accepted", rawURL)
		}
	}
}

func TestWriteConversionMultipartRejectsUnknownFamily(t *testing.T) {
	reader, writer := io.Pipe()
	multipartWriter := multipart.NewWriter(writer)
	go func() {
		_ = writeConversionMultipart(multipartWriter, writer, previewconversion.NewClassification("unknown", "", "", ""), strings.NewReader("source"))
	}()
	_, err := io.ReadAll(reader)
	if err == nil {
		t.Fatal("expected multipart error")
	}
	_ = reader.Close()
}
