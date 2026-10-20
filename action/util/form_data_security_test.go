package util

import (
	"bytes"
	"context"
	"log"
	"mime/multipart"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/i18n"
	"github.com/simpledms/simpledms/util/httpx"
)

func TestFormDataXBoundsRequestBodies(t *testing.T) {
	type form struct {
		Name string `form:"Name"`
	}
	for _, contentType := range []string{"application/x-www-form-urlencoded", "multipart/form-data"} {
		t.Run(contentType, func(t *testing.T) {
			var body []byte
			if contentType == "multipart/form-data" {
				var buf bytes.Buffer
				mw := multipart.NewWriter(&buf)
				part, err := mw.CreateFormField("Name")
				if err != nil {
					t.Fatal(err)
				}
				_, _ = part.Write([]byte(strings.Repeat("x", 1024*1024)))
				if err := mw.Close(); err != nil {
					t.Fatal(err)
				}
				body = buf.Bytes()
				contentType = mw.FormDataContentType()
			} else {
				body = []byte(url.Values{"Name": {strings.Repeat("x", 1024*1024)}}.Encode())
			}
			req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
			req.ContentLength = -1
			req.Header.Set("Content-Type", contentType)
			_, err := FormDataX[form](httpx.NewResponseWriter(httptest.NewRecorder()), httpx.NewRequest(req), ctxx.Context(nil), true)
			if err == nil {
				t.Fatal("accepted oversized body with unknown Content-Length")
			}
		})
	}
}

func TestFormDataXValidationLogsDoNotExposeInput(t *testing.T) {
	type form struct {
		Email string `form:"Email" validate:"required,email"`
	}
	const secret = "patient-private-record-secret"
	req := httptest.NewRequest("POST", "/", strings.NewReader("Email="+url.QueryEscape(secret)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := ctxx.NewVisitorContext(context.Background(), nil, i18n.NewI18n(), "en", "UTC", false, false, false)
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(previous)
	_, err := FormDataX[form](httpx.NewResponseWriter(httptest.NewRecorder()), httpx.NewRequest(req), ctx, false)
	if err == nil {
		t.Fatal("expected invalid email to fail validation")
	}
	if strings.Contains(output.String(), secret) {
		t.Fatalf("validation log exposed submitted value: %s", output.String())
	}
}

func TestFormDataXAcceptsSmallForm(t *testing.T) {
	type form struct {
		Name string `form:"Name"`
	}
	req := httptest.NewRequest("POST", "/", strings.NewReader("Name=ok"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	got, err := FormDataX[form](httpx.NewResponseWriter(httptest.NewRecorder()), httpx.NewRequest(req), ctxx.Context(nil), true)
	if err != nil || got.Name != "ok" {
		t.Fatalf("FormDataX() = %+v, %v", got, err)
	}
}
