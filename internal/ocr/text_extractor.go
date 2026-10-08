package ocr

import (
	"context"
	"errors"
	"io"
	"log"
	"strings"

	"github.com/marcobeierer/go-tika"

	"github.com/simpledms/simpledms/internal/xberg"
)

// TODO use language of user?
const ocrLanguage = "eng+deu+fra+ita+spa"

// TextExtractor extracts text with Xberg if configured and falls back to Tika for formats
// Xberg does not support and for empty Xberg results, because Xberg's OCR quality filters can
// discard text that Tika recognizes. Other Xberg failures do not fall back, so that an Xberg
// outage results in a retry instead of silently switching the extraction engine.
//
// Xberg returns markdown, whose line breaks are kept so that tables stay readable. Tika's
// plain text is joined into a single line as before.
type TextExtractor struct {
	xbergClientNilable *xberg.XbergClient
	tikaClientNilable  *tika.Client
}

// NewNilableTextExtractor returns nil if neither Xberg nor Tika is configured.
func NewNilableTextExtractor(
	xbergClientNilable *xberg.XbergClient,
	tikaClientNilable *tika.Client,
) *TextExtractor {
	if xbergClientNilable == nil && tikaClientNilable == nil {
		return nil
	}

	return &TextExtractor{
		xbergClientNilable: xbergClientNilable,
		tikaClientNilable:  tikaClientNilable,
	}
}

// ExtractText opens the file with openFile, which may be called a second time for the Tika
// fallback because the first stream is consumed by then.
func (qq *TextExtractor) ExtractText(
	ctx context.Context,
	filename string,
	mimeType string,
	openFile func() (io.ReadCloser, error),
) (string, error) {
	if qq.xbergClientNilable != nil {
		content, err := qq.extractTextWithXberg(ctx, filename, mimeType, openFile)
		if err == nil {
			content = strings.TrimSpace(content)
			if content != "" || qq.tikaClientNilable == nil {
				return content, nil
			}
		} else if !errors.Is(err, xberg.ErrUnsupportedFormat) || qq.tikaClientNilable == nil {
			log.Println(err)
			return "", err
		}
	}

	return qq.extractTextWithTika(ctx, openFile)
}

func (qq *TextExtractor) extractTextWithXberg(
	ctx context.Context,
	filename string,
	mimeType string,
	openFile func() (io.ReadCloser, error),
) (string, error) {
	openedFile, err := openFile()
	if err != nil {
		log.Println(err)
		return "", err
	}
	defer func() {
		err := openedFile.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	return qq.xbergClientNilable.ExtractText(ctx, filename, mimeType, ocrLanguage, openedFile)
}

func (qq *TextExtractor) extractTextWithTika(
	ctx context.Context,
	openFile func() (io.ReadCloser, error),
) (string, error) {
	openedFile, err := openFile()
	if err != nil {
		log.Println(err)
		return "", err
	}
	defer func() {
		err := openedFile.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	tikaHeader := tika.NewHeader().AcceptText().SetOCRLanguage(ocrLanguage)
	content, err := qq.tikaClientNilable.Parse(ctx, openedFile, tikaHeader)
	if err != nil {
		log.Println(err)
		return "", err
	}

	return removeAllWhitespace(content), nil
}
