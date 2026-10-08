package systemstatus

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/marcobeierer/go-tika"
	"github.com/minio/minio-go/v7"

	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain/mail"
	"github.com/simpledms/simpledms/internal/gotenberg"
	"github.com/simpledms/simpledms/internal/xberg"
	mailermodel "github.com/simpledms/simpledms/model/main/mailer"
	systemconfigmodel "github.com/simpledms/simpledms/model/main/systemconfig"
	"github.com/simpledms/simpledms/util/timex"
)

const (
	// short enough that the status page stays usable if a service does not respond
	probeTimeout = 5 * time.Second
	// the scheduler gives up sending a mail after this number of attempts, see sendMails
	maxMailAttempts                = 3
	certificateExpiryWarningPeriod = 14 * 24 * time.Hour
)

// SystemStatusChecker reports the effective configuration of the running instance and checks
// whether the configured services are reachable. Configuration changes take effect only after
// a restart, thus the values loaded on startup are reported.
type SystemStatusChecker struct {
	systemConfig              *systemconfigmodel.SystemConfig
	s3ClientNilable           *minio.Client
	isDevMode                 bool
	isFileEncryptionDisabled  bool
	isDBConfigOverrideEnabled bool
}

func NewSystemStatusChecker(
	systemConfig *systemconfigmodel.SystemConfig,
	s3ClientNilable *minio.Client,
	isDevMode bool,
	isFileEncryptionDisabled bool,
	isDBConfigOverrideEnabled bool,
) *SystemStatusChecker {
	return &SystemStatusChecker{
		systemConfig:              systemConfig,
		s3ClientNilable:           s3ClientNilable,
		isDevMode:                 isDevMode,
		isFileEncryptionDisabled:  isFileEncryptionDisabled,
		isDBConfigOverrideEnabled: isDBConfigOverrideEnabled,
	}
}

func (qq *SystemStatusChecker) Check(ctx ctxx.Context) (*SystemStatus, error) {
	probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	// started before the database queries so that all checks run in parallel
	storageProbe := qq.nilableStartStorageProbe(probeCtx)
	tikaProbe := qq.nilableStartTikaProbe(probeCtx)
	xbergProbe := qq.nilableStartXbergProbe(probeCtx)
	gotenbergProbe := qq.nilableStartGotenbergProbe(probeCtx)
	mailProbe := qq.nilableStartMailProbe(probeCtx)

	mailStatus, err := qq.mailStatus(ctx, mailProbe)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	components := []*ComponentStatus{
		qq.generalStatus(),
		qq.storageStatus(storageProbe),
		qq.ocrStatus(tikaProbe, xbergProbe),
		qq.previewStatus(gotenbergProbe),
		mailStatus,
		qq.tlsStatus(ctx),
	}

	return NewSystemStatus(components, time.Now()), nil
}

func (qq *SystemStatusChecker) generalStatus() *ComponentStatus {
	status := NewComponentStatus(widget.T("General"))
	config := qq.systemConfig

	status.AddSetting(widget.T("Public origin"), valueOrNotSet(config.PublicOrigin()))
	status.AddSetting(widget.T("WebAuthn relying party ID"), valueOrNotSet(config.WebAuthnRPID()))
	status.AddSetting(
		widget.T("Passphrase protection"),
		enabledOrDisabled(config.IsIdentityEncryptedWithPassphrase()),
	)
	status.AddSetting(widget.T("File encryption"), enabledOrDisabled(!qq.isFileEncryptionDisabled))
	status.AddSetting(
		widget.T("Insecure cookies"),
		allowedOrNotAllowed(config.AllowInsecureCookies()),
	)
	status.AddSetting(widget.T("Max upload size"), mibOrUnlimited(config.MaxUploadSizeBytes()))
	status.AddSetting(
		widget.T("Override stored config on startup"),
		enabledOrDisabled(qq.isDBConfigOverrideEnabled),
	)
	status.AddSetting(widget.T("Development mode"), enabledOrDisabled(qq.isDevMode))

	if qq.isDevMode {
		status.AddFinding(
			StatusLevelWarning,
			widget.T("Development mode is enabled. Do not use it in production."),
		)
	}

	publicOrigin := config.PublicOrigin()
	if publicOrigin == "" {
		status.AddFinding(
			StatusLevelWarning,
			widget.T("No public origin is set. Emails cannot link to the app."),
		)
	} else {
		publicOriginURL, err := url.Parse(publicOrigin)
		if err != nil || publicOriginURL.Hostname() == "" ||
			(publicOriginURL.Scheme != "http" && publicOriginURL.Scheme != "https") {
			status.AddFinding(StatusLevelError, widget.T("The public origin is not a valid URL."))
		} else if publicOriginURL.Scheme == "http" && !isLocalHost(publicOriginURL.Hostname()) {
			status.AddFinding(StatusLevelWarning, widget.T("The public origin does not use HTTPS."))
		}
	}

	// passkeys fall back to the request host if neither is set, see PasskeyService
	rpID := strings.ToLower(config.WebAuthnRPID())
	canonicalHost := config.CanonicalHost()
	if rpID != "" && canonicalHost != "" &&
		rpID != canonicalHost && !strings.HasSuffix(canonicalHost, "."+rpID) {
		status.AddFinding(
			StatusLevelError,
			widget.T("The WebAuthn relying party ID does not match the public origin. "+
				"Passkeys do not work."),
		)
	}

	if config.AllowInsecureCookies() {
		status.AddFinding(
			StatusLevelWarning,
			widget.T("Session cookies are also sent over unencrypted connections."),
		)
	}

	return status
}

func (qq *SystemStatusChecker) nilableStartStorageProbe(ctx context.Context) *probe {
	if qq.s3ClientNilable == nil {
		return nil
	}

	bucketName := qq.systemConfig.S3().S3BucketName
	return startProbe(ctx, func(ctx context.Context) (string, error) {
		exists, err := qq.s3ClientNilable.BucketExists(ctx, bucketName)
		if err != nil {
			return "", err
		}
		if !exists {
			return "", fmt.Errorf("bucket %q does not exist", bucketName)
		}
		return "", nil
	})
}

func (qq *SystemStatusChecker) storageStatus(storageProbeNilable *probe) *ComponentStatus {
	status := NewComponentStatus(widget.T("Storage"))
	config := qq.systemConfig.S3()

	status.AddSetting(widget.T("Endpoint"), valueOrNotSet(config.S3Endpoint))
	status.AddSetting(widget.T("Bucket"), valueOrNotSet(config.S3BucketName))
	status.AddSetting(widget.T("Access key ID"), valueOrNotSet(config.S3AccessKeyID))
	status.AddSetting(widget.T("Secret access key"), setOrNotSet(config.S3SecretAccessKey))
	status.AddSetting(widget.T("SSL/TLS"), enabledOrDisabled(config.S3UseSSL))

	if storageProbeNilable == nil {
		status.AddFinding(
			StatusLevelError,
			widget.T("No storage is configured. Files cannot be uploaded."),
		)
		return status
	}

	qq.addConnection(status, widget.T("Connection"), storageProbeNilable, "S3")

	if !config.S3UseSSL && !isLocalHost(hostWithoutPort(config.S3Endpoint)) {
		status.AddFinding(
			StatusLevelWarning,
			widget.T("The connection to the storage is not encrypted."),
		)
	}

	return status
}

func (qq *SystemStatusChecker) nilableStartTikaProbe(ctx context.Context) *probe {
	tikaURL := strings.TrimSpace(qq.systemConfig.OCR().TikaURL)
	if tikaURL == "" || !isValidServiceURL(tikaURL) {
		return nil
	}

	tikaClient := tika.NewDefaultClient(tikaURL)
	return startProbe(ctx, func(ctx context.Context) (string, error) {
		version, err := tikaClient.Version(ctx)
		return strings.TrimSpace(version), err
	})
}

func (qq *SystemStatusChecker) nilableStartXbergProbe(ctx context.Context) *probe {
	xbergURL := strings.TrimSpace(qq.systemConfig.OCR().XbergURL)
	if xbergURL == "" {
		return nil
	}

	xbergClient, err := xberg.NewXbergClient(xbergURL)
	if err != nil {
		// reported as invalid URL by ocrStatus
		return nil
	}

	return startProbe(ctx, xbergClient.Health)
}

func (qq *SystemStatusChecker) ocrStatus(
	tikaProbeNilable *probe,
	xbergProbeNilable *probe,
) *ComponentStatus {
	status := NewComponentStatus(widget.T("OCR"))
	config := qq.systemConfig.OCR()
	tikaURL := strings.TrimSpace(config.TikaURL)
	xbergURL := strings.TrimSpace(config.XbergURL)

	// mirrors ocr.TextExtractor
	switch {
	case xbergURL != "" && tikaURL != "":
		status.AddSetting(
			widget.T("Text extraction"),
			widget.T("Xberg, Tika for unsupported formats"),
		)
	case xbergURL != "":
		status.AddSetting(widget.T("Text extraction"), widget.Tu("Xberg"))
	case tikaURL != "":
		status.AddSetting(widget.T("Text extraction"), widget.Tu("Tika"))
	default:
		status.AddSetting(widget.T("Text extraction"), widget.T("Disabled"))
		status.AddFinding(
			StatusLevelWarning,
			widget.T("Neither Xberg nor Tika is configured. Text in documents is not extracted "+
				"and cannot be searched."),
		)
	}
	status.AddSetting(widget.T("Max file size"), widget.Tuf("%d MiB", config.MaxFileSizeMiB))

	status.AddSetting(widget.T("Xberg URL"), valueOrNotSet(xbergURL))
	if xbergURL != "" {
		if xbergProbeNilable == nil {
			status.AddFinding(StatusLevelError, widget.T("The Xberg URL is not valid."))
		} else {
			qq.addConnection(status, widget.T("Xberg connection"), xbergProbeNilable, "Xberg")
		}
	}

	status.AddSetting(widget.T("Tika URL"), valueOrNotSet(tikaURL))
	if tikaURL != "" {
		if tikaProbeNilable == nil {
			status.AddFinding(StatusLevelError, widget.T("The Tika URL is not valid."))
		} else {
			qq.addConnection(status, widget.T("Tika connection"), tikaProbeNilable, "Tika")
		}
	}

	return status
}

func (qq *SystemStatusChecker) nilableStartGotenbergProbe(ctx context.Context) *probe {
	gotenbergURL := strings.TrimSpace(qq.systemConfig.GotenbergURL())
	if gotenbergURL == "" {
		return nil
	}

	gotenbergClient, err := gotenberg.NewGotenbergClient(gotenbergURL)
	if err != nil {
		// reported as invalid URL by previewStatus
		return nil
	}

	return startProbe(ctx, func(ctx context.Context) (string, error) {
		return "", gotenbergClient.Health(ctx)
	})
}

func (qq *SystemStatusChecker) previewStatus(gotenbergProbeNilable *probe) *ComponentStatus {
	status := NewComponentStatus(widget.T("PDF previews"))
	gotenbergURL := strings.TrimSpace(qq.systemConfig.GotenbergURL())

	status.AddSetting(widget.T("Gotenberg URL"), valueOrNotSet(gotenbergURL))

	if gotenbergURL == "" {
		status.AddFinding(
			StatusLevelDisabled,
			widget.T("Gotenberg is not configured. PDF previews of office, HTML and Markdown "+
				"files are disabled."),
		)
		return status
	}
	if gotenbergProbeNilable == nil {
		status.AddFinding(StatusLevelError, widget.T("The Gotenberg URL is not valid."))
		return status
	}

	qq.addConnection(status, widget.T("Connection"), gotenbergProbeNilable, "Gotenberg")

	return status
}

func (qq *SystemStatusChecker) nilableStartMailProbe(ctx context.Context) *probe {
	config := qq.systemConfig.Mailer()
	if config.MailerHost == "" {
		return nil
	}

	return startProbe(ctx, func(ctx context.Context) (string, error) {
		mailClient, err := mailermodel.NewSMTPClient(config)
		if err != nil {
			return "", err
		}

		// also negotiates TLS and authenticates, thus wrong credentials are detected too
		err = mailClient.DialWithContext(ctx)
		if err != nil {
			return "", err
		}

		return "", mailClient.Close()
	})
}

func (qq *SystemStatusChecker) mailStatus(
	ctx ctxx.Context,
	mailProbeNilable *probe,
) (*ComponentStatus, error) {
	status := NewComponentStatus(widget.T("Email"))
	config := qq.systemConfig.Mailer()

	pendingMailCount, err := ctx.MainCtx().MainTx.Mail.Query().
		Where(mail.SentAtIsNil(), mail.RetryCountLT(maxMailAttempts)).
		Count(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	failedMailCount, err := ctx.MainCtx().MainTx.Mail.Query().
		Where(mail.SentAtIsNil(), mail.RetryCountGTE(maxMailAttempts)).
		Count(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	status.AddSetting(widget.T("Host"), valueOrNotSet(config.MailerHost))
	status.AddSetting(widget.T("Port"), widget.Tuf("%d", config.MailerPort))
	status.AddSetting(widget.T("Username"), valueOrNotSet(config.MailerUsername))
	status.AddSetting(widget.T("Password"), setOrNotSet(config.MailerPassword))
	status.AddSetting(widget.T("Sender address"), valueOrNotSet(config.MailerFrom))
	status.AddSetting(
		widget.T("Implicit SSL/TLS"),
		enabledOrDisabled(config.MailerUseImplicitSSLTLS),
	)
	status.AddSetting(
		widget.T("Certificate verification"),
		enabledOrDisabled(!config.MailerInsecureSkipVerify),
	)
	status.AddSetting(widget.T("Emails waiting to be sent"), widget.Tuf("%d", pendingMailCount))
	status.AddSetting(widget.T("Emails that could not be sent"), widget.Tuf("%d", failedMailCount))

	if mailProbeNilable == nil {
		status.AddFinding(
			StatusLevelWarning,
			widget.T("No mail server is configured. Emails, for example for password resets, "+
				"are not sent."),
		)
	} else {
		qq.addConnection(status, widget.T("Connection"), mailProbeNilable, "SMTP")
	}

	if config.MailerHost != "" && config.MailerFrom == "" {
		status.AddFinding(
			StatusLevelError,
			widget.T("No sender address is configured. Emails cannot be sent."),
		)
	}
	if config.MailerInsecureSkipVerify {
		status.AddFinding(
			StatusLevelWarning,
			widget.T("The certificate of the mail server is not verified."),
		)
	}
	if failedMailCount > 0 {
		status.AddFinding(
			StatusLevelWarning,
			widget.Tf(
				"%d emails could not be sent after %d attempts.",
				failedMailCount,
				maxMailAttempts,
			),
		)
	}

	return status, nil
}

func (qq *SystemStatusChecker) tlsStatus(ctx ctxx.Context) *ComponentStatus {
	status := NewComponentStatus(widget.T("TLS"))
	config := qq.systemConfig.TLS()

	// mirrors shouldUseAutocert and resolveListenMode in server
	if config.TLSEnableAutocert && !qq.isDevMode {
		var hosts []string
		for _, host := range config.TLSAutocertHosts {
			if strings.TrimSpace(host) != "" {
				hosts = append(hosts, strings.TrimSpace(host))
			}
		}

		status.AddSetting(widget.T("Mode"), widget.T("Automatic certificates from Let's Encrypt"))
		status.AddSetting(widget.T("Hosts"), valueOrNotSet(strings.Join(hosts, ", ")))
		status.AddSetting(widget.T("Email"), valueOrNotSet(config.TLSAutocertEmail))

		if len(hosts) == 0 {
			status.AddFinding(
				StatusLevelError,
				widget.T("No hosts are configured for automatic certificates."),
			)
		}
		return status
	}

	if config.TLSCertFilepath == "" || config.TLSPrivateKeyFilepath == "" {
		status.AddSetting(
			widget.T("Mode"),
			widget.T("HTTP, HTTPS must be provided by a reverse proxy"),
		)
		if config.TLSCertFilepath != "" || config.TLSPrivateKeyFilepath != "" {
			status.AddFinding(
				StatusLevelWarning,
				widget.T("Only one of certificate file and private key file is set. "+
					"TLS is disabled."),
			)
		}
		return status
	}

	status.AddSetting(widget.T("Mode"), widget.T("Certificate files"))
	status.AddSetting(widget.T("Certificate file"), widget.Tu(config.TLSCertFilepath))
	status.AddSetting(widget.T("Private key file"), widget.Tu(config.TLSPrivateKeyFilepath))

	certificate, err := tls.LoadX509KeyPair(config.TLSCertFilepath, config.TLSPrivateKeyFilepath)
	if err != nil {
		log.Println(err)
		status.AddFinding(
			StatusLevelError,
			widget.Tf("The certificate could not be loaded: %s", err.Error()),
		)
		return status
	}
	leaf := certificate.Leaf
	if leaf == nil {
		leaf, err = x509.ParseCertificate(certificate.Certificate[0])
		if err != nil {
			log.Println(err)
			status.AddFinding(
				StatusLevelError,
				widget.Tf("The certificate could not be loaded: %s", err.Error()),
			)
			return status
		}
	}

	validUntil := timex.NewDateTime(leaf.NotAfter).String(ctx.MainCtx().LanguageBCP47)
	status.AddSetting(widget.T("Certificate valid until"), widget.Tu(validUntil))

	if time.Now().After(leaf.NotAfter) {
		status.AddFinding(StatusLevelError, widget.Tf("The certificate expired on %s.", validUntil))
	} else if time.Until(leaf.NotAfter) < certificateExpiryWarningPeriod {
		status.AddFinding(
			StatusLevelWarning,
			widget.Tf("The certificate expires on %s.", validUntil),
		)
	}

	return status
}

// addConnection waits for the probe and adds its result as setting and as finding on failure.
func (qq *SystemStatusChecker) addConnection(
	status *ComponentStatus,
	label *widget.Text,
	probex *probe,
	serviceName string, // product or protocol name, thus not translated
) {
	detail, err := probex.wait()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("no response within %s", probeTimeout)
		}
		status.AddSetting(label, widget.T("Not reachable"))
		status.AddFinding(
			StatusLevelError,
			widget.Tf("%s is not reachable: %s", serviceName, err.Error()),
		)
		return
	}

	if detail == "" {
		status.AddSetting(label, widget.T("Connected"))
		return
	}
	status.AddSetting(label, widget.Tf("Connected (%s)", detail))
}

func valueOrNotSet(value string) *widget.Text {
	if value == "" {
		return widget.T("Not set")
	}
	return widget.Tu(value)
}

// setOrNotSet must be used for secrets so that they are never shown.
func setOrNotSet(secret string) *widget.Text {
	if secret == "" {
		return widget.T("Not set")
	}
	return widget.T("Set")
}

func enabledOrDisabled(isEnabled bool) *widget.Text {
	if isEnabled {
		return widget.T("Enabled")
	}
	return widget.T("Disabled")
}

func allowedOrNotAllowed(isAllowed bool) *widget.Text {
	if isAllowed {
		return widget.T("Allowed")
	}
	return widget.T("Not allowed")
}

func mibOrUnlimited(bytes int64) *widget.Text {
	if bytes <= 0 {
		return widget.T("Unlimited")
	}
	return widget.Tuf("%d MiB", bytes/(1024*1024))
}

func isValidServiceURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func hostWithoutPort(hostPort string) string {
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return hostPort
	}
	return host
}

// isLocalHost reports whether unencrypted traffic to host stays on the machine or in a private
// network, for example a Docker Compose service name without dots.
func isLocalHost(host string) bool {
	if host == "localhost" || (!strings.Contains(host, ".") && !strings.Contains(host, ":")) {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate())
}
