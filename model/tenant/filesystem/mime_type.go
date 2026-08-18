package filesystem

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func detectMIME(reader io.Reader) (string, error) {
	buf := make([]byte, 512)
	n, err := io.ReadFull(reader, buf)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", err
	}
	// Remove zero values from short files because they can cause false detection.
	// See https://gist.github.com/rayrutjes/db9b9ea8e02255d62ce2?permalink_comment_id=3418419#gistcomment-3418419.
	buf = buf[:n]

	mimeType := http.DetectContentType(buf)
	if mimeType != "application/zip" {
		return mimeType, nil
	}

	file, err := os.CreateTemp("", "simpledms-mime-*.zip")
	if err != nil {
		return "", err
	}
	defer func() {
		if err := os.Remove(file.Name()); err != nil {
			log.Println(err)
		}
	}()
	defer func() {
		if err := file.Close(); err != nil {
			log.Println(err)
		}
	}()

	if _, err := io.Copy(file, io.MultiReader(bytes.NewReader(buf), reader)); err != nil {
		return "", err
	}
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	archive, err := zip.NewReader(file, info.Size())
	if err != nil {
		return mimeType, nil
	}

	return detectZIPMIME(archive), nil
}

func detectZIPMIME(archive *zip.Reader) string {
	for _, file := range archive.File {
		switch file.Name {
		case "mimetype":
			if mimeType := readODFMIME(file); mimeType != "" {
				return mimeType
			}
		case "word/document.xml":
			return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
		case "xl/workbook.xml":
			return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		case "ppt/presentation.xml":
			return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
		}
	}
	return "application/zip"
}

func readODFMIME(file *zip.File) string {
	reader, err := file.Open()
	if err != nil {
		return ""
	}
	defer func() {
		if err := reader.Close(); err != nil {
			log.Println(err)
		}
	}()

	value, err := io.ReadAll(io.LimitReader(reader, 256))
	if err != nil {
		return ""
	}
	mimeType := strings.TrimSpace(string(value))
	switch mimeType {
	case "application/vnd.oasis.opendocument.text",
		"application/vnd.oasis.opendocument.spreadsheet",
		"application/vnd.oasis.opendocument.presentation":
		return mimeType
	default:
		return ""
	}
}
