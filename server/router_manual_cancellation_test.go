package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/util/httpx"
)

func TestRouterWrapManualTxCancellationAfterTransactionsClosed(t *testing.T) {
	t.Run("canceled request does not write an error response", func(t *testing.T) {
		harness := newActionTestHarness(t)
		requestCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(requestCtx)
		rr := httptest.NewRecorder()

		handler := func(_ httpx.ResponseWriter, req *httpx.Request, _ ctxx.Context) error {
			cancel()
			return req.Context().Err()
		}

		harness.router.wrapManualTx(handler)(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected no error status, got %d", rr.Code)
		}
		if rr.Body.Len() != 0 {
			t.Fatalf("expected no error body, got %q", rr.Body.String())
		}
	})

	t.Run("generic error uses the existing error response", func(t *testing.T) {
		harness := newActionTestHarness(t)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		handler := func(httpx.ResponseWriter, *httpx.Request, ctxx.Context) error {
			return errors.New("generic handler error")
		}

		harness.router.wrapManualTx(handler)(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), unexpectedErrorMessage) {
			t.Fatalf("expected existing error response body, got %q", rr.Body.String())
		}
	})
}
