package ocr

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/marcobeierer/go-tika"

	"github.com/simpledms/simpledms/internal/xberg"
)

const unsupportedFormatResponse = `{"results":[],"errors":[{"index":0,"code":1010,` +
	`"error_type":"unsupported_format","source":"file.xyz","message":"Unsupported format"}],` +
	`"summary":{"inputs":1,"results":0,"errors":1}}`

type fakeService struct {
	server       *httptest.Server
	requestCount int
}

func newFakeService(t *testing.T, status int, body string) *fakeService {
	t.Helper()

	service := &fakeService{}
	service.server = httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			service.requestCount++
			_, _ = io.Copy(io.Discard, req.Body)
			rw.WriteHeader(status)
			_, _ = rw.Write([]byte(body))
		},
	))
	t.Cleanup(service.server.Close)
	return service
}

func newTestXbergClient(t *testing.T, service *fakeService) *xberg.XbergClient {
	t.Helper()

	client, err := xberg.NewXbergClient(service.server.URL)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

type countingOpener struct {
	openCount int
}

func (qq *countingOpener) Open() (io.ReadCloser, error) {
	qq.openCount++
	return io.NopCloser(strings.NewReader("file content")), nil
}

func TestTextExtractorPrefersXbergOverTika(t *testing.T) {
	xbergService := newFakeService(t, http.StatusOK, `{"results":[{"content":"from xberg"}]}`)
	tikaService := newFakeService(t, http.StatusOK, "from tika")
	extractor := NewNilableTextExtractor(
		newTestXbergClient(t, xbergService),
		tika.NewClient(nil, tikaService.server.URL),
	)

	content, err := extractor.ExtractText(
		context.Background(),
		"scan.pdf",
		"application/pdf",
		(&countingOpener{}).Open,
	)
	if err != nil {
		t.Fatal(err)
	}
	if content != "from xberg" {
		t.Fatalf("expected Xberg content, got %q", content)
	}
	if tikaService.requestCount != 0 {
		t.Fatalf("expected Tika not to be called, got %d requests", tikaService.requestCount)
	}
}

func TestTextExtractorKeepsXbergLineBreaksButJoinsTikaLines(t *testing.T) {
	xbergService := newFakeService(
		t,
		http.StatusOK,
		`{"results":[{"content":"\n# Title\n\n| a | b |\n| --- | --- |\n"}]}`,
	)
	content, err := NewNilableTextExtractor(newTestXbergClient(t, xbergService), nil).ExtractText(
		context.Background(),
		"table.pdf",
		"application/pdf",
		(&countingOpener{}).Open,
	)
	if err != nil {
		t.Fatal(err)
	}
	if content != "# Title\n\n| a | b |\n| --- | --- |" {
		t.Fatalf("expected trimmed Xberg markdown, got %q", content)
	}

	tikaService := newFakeService(t, http.StatusOK, "Title\n\nfirst line\nsecond line\n")
	content, err = NewNilableTextExtractor(nil, tika.NewClient(nil, tikaService.server.URL)).
		ExtractText(context.Background(), "table.pdf", "application/pdf", (&countingOpener{}).Open)
	if err != nil {
		t.Fatal(err)
	}
	if content != "Title. first line. second line." {
		t.Fatalf("expected Tika lines joined, got %q", content)
	}
}

func TestTextExtractorFallsBackToTikaForFormatsUnsupportedByXberg(t *testing.T) {
	xbergService := newFakeService(t, http.StatusOK, unsupportedFormatResponse)
	tikaService := newFakeService(t, http.StatusOK, "from tika")
	extractor := NewNilableTextExtractor(
		newTestXbergClient(t, xbergService),
		tika.NewClient(nil, tikaService.server.URL),
	)
	opener := &countingOpener{}

	content, err := extractor.ExtractText(context.Background(), "file.xyz", "", opener.Open)
	if err != nil {
		t.Fatal(err)
	}
	if content != "from tika." {
		t.Fatalf("expected Tika content, got %q", content)
	}
	// the stream consumed by Xberg cannot be reused for Tika
	if opener.openCount != 2 {
		t.Fatalf("expected file to be opened twice, got %d", opener.openCount)
	}
}

func TestTextExtractorFallsBackToTikaForEmptyXbergResults(t *testing.T) {
	xbergService := newFakeService(t, http.StatusOK, `{"results":[{"content":" \n "}]}`)
	tikaService := newFakeService(t, http.StatusOK, "from tika")
	extractor := NewNilableTextExtractor(
		newTestXbergClient(t, xbergService),
		tika.NewClient(nil, tikaService.server.URL),
	)
	opener := &countingOpener{}

	content, err := extractor.ExtractText(
		context.Background(),
		"scan.png",
		"image/png",
		opener.Open,
	)
	if err != nil {
		t.Fatal(err)
	}
	if content != "from tika." {
		t.Fatalf("expected Tika content, got %q", content)
	}
	if opener.openCount != 2 {
		t.Fatalf("expected file to be opened twice, got %d", opener.openCount)
	}
}

func TestTextExtractorKeepsEmptyXbergResultWithoutTika(t *testing.T) {
	xbergService := newFakeService(t, http.StatusOK, `{"results":[{"content":""}]}`)
	extractor := NewNilableTextExtractor(newTestXbergClient(t, xbergService), nil)

	content, err := extractor.ExtractText(
		context.Background(),
		"blank.png",
		"image/png",
		(&countingOpener{}).Open,
	)
	if err != nil {
		t.Fatal(err)
	}
	if content != "" {
		t.Fatalf("expected empty content, got %q", content)
	}
}

func TestTextExtractorReturnsUnsupportedFormatWithoutTika(t *testing.T) {
	xbergService := newFakeService(t, http.StatusOK, unsupportedFormatResponse)
	extractor := NewNilableTextExtractor(newTestXbergClient(t, xbergService), nil)

	_, err := extractor.ExtractText(
		context.Background(),
		"file.xyz",
		"",
		(&countingOpener{}).Open,
	)
	if !errors.Is(err, xberg.ErrUnsupportedFormat) {
		t.Fatalf("expected unsupported format error, got %v", err)
	}
}

func TestTextExtractorDoesNotFallBackToTikaOnOtherXbergFailures(t *testing.T) {
	testCases := []struct {
		name   string
		status int
		body   string
	}{
		{
			name:   "service error",
			status: http.StatusInternalServerError,
			body:   `{"error_type":"InternalError","message":"boom","status_code":500}`,
		},
		{
			name:   "parsing error",
			status: http.StatusOK,
			body: `{"results":[],"errors":[{"index":0,"code":1001,"error_type":"parsing",` +
				`"source":"bad.pdf","message":"Invalid cross-reference table"}]}`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			xbergService := newFakeService(t, testCase.status, testCase.body)
			tikaService := newFakeService(t, http.StatusOK, "from tika")
			extractor := NewNilableTextExtractor(
				newTestXbergClient(t, xbergService),
				tika.NewClient(nil, tikaService.server.URL),
			)

			_, err := extractor.ExtractText(
				context.Background(),
				"bad.pdf",
				"application/pdf",
				(&countingOpener{}).Open,
			)
			if err == nil {
				t.Fatal("expected error")
			}
			if tikaService.requestCount != 0 {
				t.Fatalf("expected Tika not to be called, got %d requests", tikaService.requestCount)
			}
		})
	}
}

func TestTextExtractorUsesTikaIfXbergIsNotConfigured(t *testing.T) {
	tikaService := newFakeService(t, http.StatusOK, "from tika")
	extractor := NewNilableTextExtractor(nil, tika.NewClient(nil, tikaService.server.URL))

	content, err := extractor.ExtractText(
		context.Background(),
		"scan.pdf",
		"application/pdf",
		(&countingOpener{}).Open,
	)
	if err != nil {
		t.Fatal(err)
	}
	if content != "from tika." {
		t.Fatalf("expected Tika content, got %q", content)
	}
}

func TestNewNilableTextExtractorIsNilWithoutBackends(t *testing.T) {
	if NewNilableTextExtractor(nil, nil) != nil {
		t.Fatal("expected nil text extractor without Xberg and Tika")
	}
}
