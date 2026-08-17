package gotenberg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/simpledms/simpledms/model/tenant/previewconversion"
)

const (
	gotenbergRequestTimeout = 2 * time.Minute
	gotenbergResponseLimit  = 256 * 1024 * 1024
)

const (
	FailureCategoryNetwork          = "network"
	FailureCategoryTimeout          = "timeout"
	FailureCategoryService          = "service_unavailable"
	FailureCategoryRejected         = "converter_rejected"
	FailureCategoryInvalidResponse  = "invalid_response"
	FailureCategorySourceUnreadable = "source_unreadable"
)

type ConversionError struct {
	Category  string
	Retryable bool
	Status    int
	TraceID   string
	Err       error
}

func (qq *ConversionError) Error() string {
	if qq.Err == nil {
		if qq.Status != 0 {
			return fmt.Sprintf("gotenberg conversion failed with status %d", qq.Status)
		}
		return "gotenberg conversion failed"
	}
	return qq.Err.Error()
}

func (qq *ConversionError) Unwrap() error {
	return qq.Err
}

type GotenbergClient struct {
	baseURL          string
	httpClient       *http.Client
	maxResponseBytes int64
}

func NewGotenbergClient(rawURL string) (*GotenbergClient, error) {
	rawURL = strings.TrimRight(strings.TrimSpace(rawURL), "/")
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Gotenberg URL: %w", err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("invalid Gotenberg URL")
	}

	return &GotenbergClient{
		baseURL: rawURL,
		httpClient: &http.Client{
			Timeout: gotenbergRequestTimeout,
		},
		maxResponseBytes: gotenbergResponseLimit,
	}, nil
}

func IsValidGotenbergURL(rawURL string) bool {
	_, err := NewGotenbergClient(rawURL)
	return err == nil
}

func (qq *GotenbergClient) Convert(
	ctx context.Context,
	classification *previewconversion.Classification,
	source io.Reader,
) (io.ReadCloser, error) {
	pipeReader, pipeWriter := io.Pipe()
	multipartWriter := multipart.NewWriter(pipeWriter)
	writerDone := make(chan error, 1)
	go func() {
		err := writeConversionMultipart(multipartWriter, pipeWriter, classification, source)
		writerDone <- err
	}()

	requestContext, cancel := context.WithTimeout(ctx, gotenbergRequestTimeout)
	requestURL := qq.baseURL + path.Clean("/"+classification.Route)
	request, err := http.NewRequestWithContext(requestContext, http.MethodPost, requestURL, pipeReader)
	if err != nil {
		cancel()
		_ = pipeReader.CloseWithError(err)
		return nil, &ConversionError{Category: FailureCategoryNetwork, Retryable: true, Err: err}
	}

	contentType := multipartWriter.FormDataContentType()
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Gotenberg-Output-Filename", classification.OutputFilename)

	response, err := qq.httpClient.Do(request)
	if err != nil {
		cancel()
		_ = pipeReader.CloseWithError(err)
		category := FailureCategoryNetwork
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(requestContext.Err(), context.DeadlineExceeded) {
			category = FailureCategoryTimeout
		}
		var netError net.Error
		if errors.As(err, &netError) && netError.Timeout() {
			category = FailureCategoryTimeout
		}
		return nil, &ConversionError{Category: category, Retryable: true, Err: err}
	}

	traceID := response.Header.Get("Gotenberg-Trace-Id")
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_ = response.Body.Close()
		cancel()
		_ = pipeReader.CloseWithError(errors.New("gotenberg rejected the request"))
		return nil, &ConversionError{
			Category:  FailureCategoryRejected,
			Retryable: response.StatusCode >= http.StatusInternalServerError || response.StatusCode == http.StatusRequestTimeout || response.StatusCode == http.StatusTooManyRequests,
			Status:    response.StatusCode,
			TraceID:   traceID,
		}
	}

	contentType = strings.ToLower(strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0]))
	if contentType != "application/pdf" {
		_ = response.Body.Close()
		cancel()
		_ = pipeReader.CloseWithError(errors.New("gotenberg returned an invalid content type"))
		return nil, &ConversionError{
			Category: FailureCategoryInvalidResponse,
			TraceID:  traceID,
			Err:      fmt.Errorf("unexpected Gotenberg content type %q", contentType),
		}
	}

	return &pdfResponseReader{
		body:       response.Body,
		cancel:     cancel,
		maxBytes:   qq.maxResponseBytes,
		traceID:    traceID,
		writerDone: writerDone,
	}, nil
}

func writeConversionMultipart(
	writer *multipart.Writer,
	pipeWriter *io.PipeWriter,
	classification *previewconversion.Classification,
	source io.Reader,
) error {
	writeFile := func(filename string, content io.Reader) error {
		part, err := writer.CreateFormFile("files", filename)
		if err != nil {
			return err
		}
		_, err = io.Copy(part, content)
		return err
	}

	var err error
	switch classification.Family {
	case previewconversion.FamilyHTML, previewconversion.FamilyOffice:
		err = writeFile(classification.InputFilename, source)
	case previewconversion.FamilyMarkdown:
		err = writeFile("index.html", strings.NewReader(`<!doctype html>
<html><head><meta charset="utf-8"><title>Preview</title></head><body>{{ toHTML "source.md" }}</body></html>`))
		if err == nil {
			err = writeFile("source.md", source)
		}
	default:
		err = fmt.Errorf("unsupported preview conversion family %q", classification.Family)
	}
	if err == nil {
		err = writer.Close()
	}
	if err != nil {
		_ = pipeWriter.CloseWithError(err)
		return err
	}
	return pipeWriter.Close()
}

type pdfResponseReader struct {
	body       io.ReadCloser
	cancel     context.CancelFunc
	maxBytes   int64
	bytesRead  int64
	prefix     []byte
	prefixPos  int
	validated  bool
	traceID    string
	writerDone <-chan error
}

func (qq *pdfResponseReader) Read(p []byte) (int, error) {
	if !qq.validated {
		qq.prefix = make([]byte, 5)
		_, err := io.ReadFull(qq.body, qq.prefix)
		if err != nil {
			return 0, &ConversionError{Category: FailureCategoryInvalidResponse, TraceID: qq.traceID, Err: err}
		}
		if !bytes.Equal(qq.prefix, []byte("%PDF-")) {
			return 0, &ConversionError{
				Category: FailureCategoryInvalidResponse,
				TraceID:  qq.traceID,
				Err:      errors.New("Gotenberg response is not a PDF"),
			}
		}
		qq.validated = true
	}

	if qq.prefixPos < len(qq.prefix) {
		n := copy(p, qq.prefix[qq.prefixPos:])
		qq.prefixPos += n
		qq.bytesRead += int64(n)
		return n, nil
	}

	if qq.bytesRead >= qq.maxBytes {
		return 0, &ConversionError{
			Category: FailureCategoryInvalidResponse,
			TraceID:  qq.traceID,
			Err:      errors.New("Gotenberg response exceeds the configured size limit"),
		}
	}
	remaining := qq.maxBytes - qq.bytesRead
	if int64(len(p)) > remaining {
		p = p[:int(remaining)]
	}
	n, err := qq.body.Read(p)
	qq.bytesRead += int64(n)
	if err != nil && !errors.Is(err, io.EOF) {
		return n, &ConversionError{Category: FailureCategoryInvalidResponse, TraceID: qq.traceID, Err: err}
	}
	if err == io.EOF && qq.bytesRead <= 5 {
		return n, &ConversionError{
			Category: FailureCategoryInvalidResponse,
			TraceID:  qq.traceID,
			Err:      errors.New("Gotenberg response is empty"),
		}
	}
	if err == io.EOF {
		return n, err
	}
	return n, nil
}

func (qq *pdfResponseReader) Close() error {
	qq.cancel()
	if err := qq.body.Close(); err != nil {
		return err
	}
	if qq.writerDone != nil {
		if err := <-qq.writerDone; err != nil {
			return &ConversionError{Category: FailureCategorySourceUnreadable, Err: err}
		}
	}
	return nil
}
