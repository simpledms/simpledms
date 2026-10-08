package xberg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"time"
)

const (
	// OCR of large scanned PDFs runs page by page and can take several minutes.
	xbergRequestTimeout = 10 * time.Minute
	xbergResponseLimit  = 256 * 1024 * 1024

	unsupportedFormatErrorType = "unsupported_format"
)

// ErrUnsupportedFormat is returned if Xberg has no extractor for the file format.
var ErrUnsupportedFormat = errors.New("xberg: unsupported format")

type XbergClient struct {
	baseURL          string
	httpClient       *http.Client
	maxResponseBytes int64
}

func NewXbergClient(rawURL string) (*XbergClient, error) {
	rawURL = strings.TrimRight(strings.TrimSpace(rawURL), "/")
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Xberg URL: %w", err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Host == "" ||
		parsed.User != nil ||
		parsed.RawQuery != "" ||
		parsed.Fragment != "" {
		return nil, errors.New("invalid Xberg URL")
	}

	return &XbergClient{
		baseURL: rawURL,
		httpClient: &http.Client{
			Timeout: xbergRequestTimeout,
		},
		maxResponseBytes: xbergResponseLimit,
	}, nil
}

// ExtractText returns the plain text of source. mimeType is optional; if empty, Xberg detects
// the format from the filename extension and the content.
func (qq *XbergClient) ExtractText(
	ctx context.Context,
	filename string,
	mimeType string,
	ocrLanguage string,
	source io.Reader,
) (string, error) {
	configJSON, err := json.Marshal(newExtractConfig(ocrLanguage))
	if err != nil {
		return "", err
	}

	pipeReader, pipeWriter := io.Pipe()
	multipartWriter := multipart.NewWriter(pipeWriter)
	go func() {
		err := writeExtractMultipart(multipartWriter, filename, mimeType, configJSON, source)
		_ = pipeWriter.CloseWithError(err)
	}()

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		qq.baseURL+"/extract",
		pipeReader,
	)
	if err != nil {
		_ = pipeReader.CloseWithError(err)
		return "", err
	}
	request.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	request.Header.Set("Accept", "application/json")

	response, err := qq.httpClient.Do(request)
	if err != nil {
		_ = pipeReader.CloseWithError(err)
		return "", fmt.Errorf("xberg request failed: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()
	// unblocks the multipart writer if the server responded before reading the whole body
	defer func() {
		_ = pipeReader.CloseWithError(errors.New("xberg response received"))
	}()

	body := io.LimitReader(response.Body, qq.maxResponseBytes)

	if response.StatusCode != http.StatusOK {
		var errorResp errorResponse
		err = json.NewDecoder(body).Decode(&errorResp)
		if err != nil {
			return "", fmt.Errorf("xberg extraction failed with status %d", response.StatusCode)
		}
		return "", fmt.Errorf(
			"xberg extraction failed with status %d: %s: %s",
			response.StatusCode,
			errorResp.ErrorType,
			errorResp.Message,
		)
	}

	var extractResp extractResponse
	err = json.NewDecoder(body).Decode(&extractResp)
	if err != nil {
		return "", fmt.Errorf("invalid xberg response: %w", err)
	}

	// per-input errors are reported with status 200, for example for unsupported formats
	if len(extractResp.Errors) > 0 {
		extractErr := extractResp.Errors[0]
		if extractErr.ErrorType == unsupportedFormatErrorType {
			return "", fmt.Errorf("%w: %s", ErrUnsupportedFormat, extractErr.Message)
		}
		return "", fmt.Errorf(
			"xberg extraction failed: %s: %s",
			extractErr.ErrorType,
			extractErr.Message,
		)
	}
	if len(extractResp.Results) != 1 {
		return "", fmt.Errorf("xberg returned %d results, expected 1", len(extractResp.Results))
	}

	return extractResp.Results[0].Content, nil
}

type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// Health returns the Xberg version if the server reports itself as healthy.
func (qq *XbergClient) Health(ctx context.Context) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, qq.baseURL+"/health", nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/json")

	response, err := qq.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("xberg request failed: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("xberg health check failed with status %d", response.StatusCode)
	}

	var healthResp healthResponse
	err = json.NewDecoder(io.LimitReader(response.Body, 64*1024)).Decode(&healthResp)
	if err != nil {
		return "", fmt.Errorf("invalid xberg health response: %w", err)
	}
	if healthResp.Status != "healthy" {
		return "", fmt.Errorf("xberg reports status %q", healthResp.Status)
	}

	return healthResp.Version, nil
}

func writeExtractMultipart(
	multipartWriter *multipart.Writer,
	filename string,
	mimeType string,
	configJSON []byte,
	source io.Reader,
) error {
	err := multipartWriter.WriteField("config", string(configJSON))
	if err != nil {
		return err
	}

	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	partHeader := make(textproto.MIMEHeader)
	partHeader.Set(
		"Content-Disposition",
		multipart.FileContentDisposition("files", filename),
	)
	partHeader.Set("Content-Type", mimeType)

	part, err := multipartWriter.CreatePart(partHeader)
	if err != nil {
		return err
	}
	_, err = io.Copy(part, source)
	if err != nil {
		return err
	}

	return multipartWriter.Close()
}
