package systemstatus

import (
	"net/netip"
	"os"
	"strings"

	"github.com/simpledms/simpledms/core/ui/widget"
)

// RuntimeStatus snapshots settings that are not stored in SystemConfig at startup.
type RuntimeStatus struct {
	listenPort          int
	dataDirectory       string
	trustedProxyCIDRs   []string
	environmentSettings []*Setting
}

func NewRuntimeStatus(
	listenPort int,
	dataDirectory string,
	trustedProxies []netip.Prefix,
) *RuntimeStatus {
	status := &RuntimeStatus{
		listenPort:    listenPort,
		dataDirectory: dataDirectory,
	}
	for _, prefix := range trustedProxies {
		status.trustedProxyCIDRs = append(status.trustedProxyCIDRs, prefix.String())
	}

	// Use an explicit allowlist: bootstrap credentials and unrelated process secrets
	// must never appear on the status page. These are overrides, not parsed defaults.
	for _, name := range []string{
		"SIMPLEDMS_OPENCLOUD_ORIGIN",
		"SIMPLEDMS_DB_READ_ONLY_MAX_OPEN_CONNS",
		"SIMPLEDMS_DEV_DISABLE_SQL_LOG",
		"SIMPLEDMS_DEV_PWA_NAME",
		"CONTENT_HASH_BACKFILL_ENABLED",
		"CONTENT_HASH_BACKFILL_MAX_FILES_PER_RUN",
		"CONTENT_HASH_BACKFILL_MAX_BYTES_PER_RUN",
		"CONTENT_HASH_BACKFILL_MAX_DURATION",
		"CONTENT_HASH_BACKFILL_INTERVAL",
	} {
		status.environmentSettings = append(status.environmentSettings,
			NewSetting(widget.Tu(name), valueOrNotSet(os.Getenv(name))),
		)
	}
	for _, name := range []string{
		"SIMPLEDMS_OPENCLOUD_PUBLIC_LINK_PASSWORD",
		"SIMPLEDMS_PPROF_PATH",
	} {
		status.environmentSettings = append(status.environmentSettings,
			NewSetting(widget.Tu(name), setOrNotSet(os.Getenv(name))),
		)
	}
	return status
}

func (qq *RuntimeStatus) componentStatus() *ComponentStatus {
	status := NewComponentStatus(widget.T("Runtime"))
	status.AddSetting(widget.T("Listen port"), widget.Tuf("%d", qq.listenPort))
	status.AddSetting(widget.T("Data directory"), valueOrNotSet(qq.dataDirectory))
	status.AddSetting(
		widget.T("Trusted proxy CIDRs"),
		valueOrNotSet(strings.Join(qq.trustedProxyCIDRs, ", ")),
	)
	return status
}

func (qq *RuntimeStatus) environmentStatus() *ComponentStatus {
	status := NewComponentStatus(widget.T("Startup environment overrides"))
	status.AddFinding(StatusLevelOK, widget.T(
		"These additional environment values were captured at startup. "+
			"Unset values use component defaults; invalid overrides may fall back to defaults. "+
			"Development settings apply only in development mode. "+
			"Passwords and the profiling path are shown only as set or not set.",
	))
	for _, setting := range qq.environmentSettings {
		status.AddSetting(setting.Label(), setting.Value())
	}
	return status
}
