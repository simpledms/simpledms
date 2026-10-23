package server

import (
	"fmt"
	"log"
	"net/http"

	"golang.org/x/crypto/acme/autocert"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/db/sqlx"
	systemconfigmodel "github.com/simpledms/simpledms/model/main/systemconfig"
)

type PreparedServer struct {
	server               *Server
	mainDB               *sqlx.MainDB
	tenantDBs            *tenantdbs.TenantDBs
	router               *Router
	handler              http.Handler
	systemConfig         *systemconfigmodel.SystemConfig
	autocertManager      *autocert.Manager
	nilableStartupServer *startupServer
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
// before the server starts serving application requests.
func (qq *PreparedServer) Router() *Router {
	return qq.router
}

// Infra returns the initialized infrastructure so wrappers can build actions
// that depend on renderer, config, and other runtime services.
func (qq *PreparedServer) Infra() *common.Infra {
	return qq.router.infra
}

// Start activates application requests, reusing the maintenance listener when locked.
func (qq *PreparedServer) Start() error {
	defer closePreparedResources(qq.mainDB, qq.tenantDBs)

	if qq.nilableStartupServer != nil {
		qq.nilableStartupServer.setHandler(qq.handler)
		err := <-qq.nilableStartupServer.done
		if err != nil {
			log.Println(err)
		}
		return err
	}

	tlsConfig := qq.systemConfig.TLS()
	useAutocert := tlsConfig.TLSEnableAutocert && !qq.server.devMode

	mainListenMode := resolveListenMode(
		useAutocert,
		tlsConfig.TLSCertFilepath,
		tlsConfig.TLSPrivateKeyFilepath,
	)

	server := newStartupServer(fmt.Sprintf(":%d", qq.server.port(
		useAutocert,
		tlsConfig.TLSCertFilepath,
		tlsConfig.TLSPrivateKeyFilepath,
	)), qq.handler)

	server.start(
		mainListenMode,
		tlsConfig.TLSCertFilepath,
		tlsConfig.TLSPrivateKeyFilepath,
		qq.autocertManager,
	)
	err := <-server.done
	if err != nil {
		log.Println(err)
	}

	return err
}
