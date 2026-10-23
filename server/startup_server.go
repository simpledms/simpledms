package server

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"

	"golang.org/x/crypto/acme/autocert"
)

// startupServer keeps the listener alive while maintenance gives way to the app.
type startupServer struct {
	server  *http.Server
	mu      sync.RWMutex
	handler http.Handler
	done    chan error
}

func newStartupServer(addr string, handler http.Handler) *startupServer {
	server := &startupServer{
		handler: handler,
		done:    make(chan error, 1),
	}
	server.server = newHTTPServer(addr, server)
	return server
}

// ServeHTTP dispatches requests to the current maintenance or application handler.
func (qq *startupServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	qq.mu.RLock()
	handler := qq.handler
	qq.mu.RUnlock()
	// In-flight maintenance requests finish with their original handler.
	handler.ServeHTTP(rw, req)
}

func (qq *startupServer) setHandler(handler http.Handler) {
	qq.mu.Lock()
	defer qq.mu.Unlock()
	qq.handler = handler
}

func (qq *startupServer) start(
	mode listenMode,
	certFilepath, privateKeyFilepath string,
	manager *autocert.Manager,
) {
	go func() {
		var err error
		switch mode {
		case listenModeTLSAutocert:
			qq.server.TLSConfig = &tls.Config{GetCertificate: manager.GetCertificate}
			err = qq.server.ListenAndServeTLS("", "")
		case listenModeHTTP:
			err = qq.server.ListenAndServe()
		case listenModeTLSFiles:
			err = qq.server.ListenAndServeTLS(certFilepath, privateKeyFilepath)
		default:
			err = fmt.Errorf("unknown listen mode: %d", mode)
		}
		qq.done <- err
	}()
}
