package server

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPublicActionSlowBodyDoesNotHoldMainWriteConnection(t *testing.T) {
	for _, tc := range []struct {
		name        string
		endpoint    string
		contentType string
		prefix      string
	}{
		{
			name:        "sign-in form",
			endpoint:    "/-/auth/sign-in-cmd",
			contentType: "application/x-www-form-urlencoded",
			prefix:      "Email=",
		},
		{
			name:        "password-reset multipart form",
			endpoint:    "/-/auth/reset-password-cmd",
			contentType: "multipart/form-data; boundary=security-test",
			prefix:      "--security-test\r\n",
		},
		{
			name:        "passkey JSON",
			endpoint:    "/-/auth/passkey-sign-in-finish-cmd",
			contentType: "application/json",
			prefix:      `{"challengeId":"`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			harness := newActionTestHarness(t)
			body, writer := io.Pipe()
			req := httptest.NewRequest(http.MethodPost, tc.endpoint, body)
			req.Header.Set("Content-Type", tc.contentType)
			done := make(chan struct{})
			go func() {
				defer close(done)
				harness.router.ServeHTTP(httptest.NewRecorder(), req)
			}()
			defer func() {
				if err := writer.Close(); err != nil {
					t.Error(err)
				}
				select {
				case <-done:
				case <-time.After(5 * time.Second):
					t.Error("request did not finish after releasing its body")
				}
			}()

			started := make(chan error, 1)
			go func() {
				_, err := io.WriteString(writer, tc.prefix)
				started <- err
			}()
			select {
			case err := <-started:
				if err != nil {
					t.Fatalf("start request body: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("handler did not start reading the public request body")
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			tx, err := harness.mainDB.Tx(ctx, false)
			if err != nil {
				t.Fatalf("incomplete public request blocked an independent writer: %v", err)
			}
			if err := tx.Rollback(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPublicActionRejectsOversizedBodyBeforeWaitingForDatabase(t *testing.T) {
	harness := newActionTestHarness(t)
	tx, err := harness.mainDB.Tx(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil {
			t.Error(err)
		}
	}()

	for _, contentLength := range []int64{1024*1024 + 1, -1} {
		t.Run(fmt.Sprintf("content-length-%d", contentLength), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			req := httptest.NewRequest(
				http.MethodPost,
				harness.actions.Auth.SignInCmd.Endpoint(),
				strings.NewReader(strings.Repeat("x", 1024*1024+1)),
			).WithContext(ctx)
			req.ContentLength = contentLength
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rr := httptest.NewRecorder()
			harness.router.ServeHTTP(rr, req)
			if rr.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("status = %d, want 413 before acquiring a transaction", rr.Code)
			}
		})
	}
}

func TestPublicActionTimesOutIncompleteBodyThroughCompression(t *testing.T) {
	harness := newActionTestHarness(t)
	handler := newPublicHandler(harness.router)
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		handler.ServeHTTP(&shortReadDeadlineResponseWriter{ResponseWriter: rw}, req)
	}))
	defer server.Close()

	conn, err := net.DialTimeout("tcp", server.Listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()
	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	_, err = fmt.Fprintf(conn,
		"POST %s HTTP/1.1\r\nHost: example.com\r\n"+
			"Content-Type: application/x-www-form-urlencoded\r\n"+
			"Content-Length: 1024\r\nAccept-Encoding: gzip\r\n\r\nEmail=",
		harness.actions.Auth.SignInCmd.Endpoint(),
	)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("waiting for incomplete request to time out: %v", err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	if response.StatusCode != http.StatusRequestTimeout {
		t.Fatalf("status = %d, want 408", response.StatusCode)
	}
	if response.Header.Get("Content-Encoding") != "gzip" {
		t.Fatal("request did not exercise the compression middleware")
	}
}
