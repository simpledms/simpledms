package server

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/crypto/acme/autocert"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/model/modelmain"
)

type PreparedServer struct {
	server          *Server
	mainDB          *sqlx.MainDB
	tenantDBs       *tenantdbs.TenantDBs
	router          *Router
	handler         http.Handler
	systemConfig    *modelmain.SystemConfig
	autocertManager *autocert.Manager
}

func closePreparedResources(mainDB *sqlx.MainDB, tenantDBs *tenantdbs.TenantDBs) {
	// FIXME is it okay to keep all databases open all the time?

	if mainDB != nil {
		err := mainDB.Close()
		if err != nil {
			log.Println(err)
		}
	}

	if tenantDBs != nil {
		tenantDBs.Range(func(tenantID int64, tenantDB *sqlx.TenantDB) bool {
			err := tenantDB.Close()
			if err != nil {
				log.Println(err)
			}
			return true
		})
	}
}

// Router returns the prepared router so callers can register custom handlers
// before the server starts listening.
func (qq *PreparedServer) Router() *Router {
	return qq.router
}

// Infra returns the initialized infrastructure so wrappers can build actions
// that depend on renderer, config, and other runtime services.
func (qq *PreparedServer) Infra() *common.Infra {
	return qq.router.infra
}

// Start starts background services and begins listening for HTTP requests.
func (qq *PreparedServer) Start() error {
	defer closePreparedResources(qq.mainDB, qq.tenantDBs)

	tlsConfig := qq.systemConfig.TLS()
	useAutocert := tlsConfig.TLSEnableAutocert && !qq.server.devMode

	mainListenMode := resolveListenMode(
		useAutocert,
		tlsConfig.TLSCertFilepath,
		tlsConfig.TLSPrivateKeyFilepath,
	)

	var err error

	switch mainListenMode {
	case listenModeTLSAutocert:
		server := &http.Server{
			Addr: fmt.Sprintf(":%d", qq.server.port(
				useAutocert,
				tlsConfig.TLSCertFilepath,
				tlsConfig.TLSPrivateKeyFilepath,
			)),
			TLSConfig: &tls.Config{GetCertificate: qq.autocertManager.GetCertificate},
			Handler:   qq.handler,
		}
		err = server.ListenAndServeTLS("", "")
	case listenModeHTTP:
		err = http.ListenAndServe(
			fmt.Sprintf(":%d", qq.server.port(
				useAutocert,
				tlsConfig.TLSCertFilepath,
				tlsConfig.TLSPrivateKeyFilepath,
			)),
			qq.handler,
		)
	case listenModeTLSFiles:
		err = http.ListenAndServeTLS(
			fmt.Sprintf(":%d", qq.server.port(
				useAutocert,
				tlsConfig.TLSCertFilepath,
				tlsConfig.TLSPrivateKeyFilepath,
			)),
			tlsConfig.TLSCertFilepath,
			tlsConfig.TLSPrivateKeyFilepath,
			qq.handler,
		)
	default:
		log.Fatalln("unknown main listen mode")
	}
	if err != nil {
		log.Fatalln(err)
	}

	return nil
}
