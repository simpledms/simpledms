package systemconfig

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"filippo.io/age"

	appmodel "github.com/simpledms/simpledms/core/model/app"
	"github.com/simpledms/simpledms/core/util/e"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/encryptor"
)

const bytesPerMiB int64 = 1024 * 1024

type SystemConfig struct {
	// not exported so that nobody has write access
	data              *entmain.SystemConfig
	isSaaSModeEnabled bool
	// also stored in VisitorContext
	commercialLicenseEnabled bool
	// nilableX25519Identity *age.X25519Identity
	allowInsecureCookies bool
	publicOrigin         string
	webauthnRPID         string
	webauthnRPName       string
}

func NewSystemConfig(
	data *entmain.SystemConfig,
	isSaaSModeEnabled, commercialLicenseEnabled, allowInsecureCookies bool,
	publicOrigin, webauthnRPID, webauthnRPName string,
) *SystemConfig {
	return &SystemConfig{
		data:                     data,
		isSaaSModeEnabled:        isSaaSModeEnabled,
		commercialLicenseEnabled: commercialLicenseEnabled,
		allowInsecureCookies:     allowInsecureCookies,
		publicOrigin:             strings.TrimSpace(publicOrigin),
		webauthnRPID:             strings.TrimSpace(webauthnRPID),
		webauthnRPName:           strings.TrimSpace(webauthnRPName),
	}
}

/*
func (qq *SystemConfig) IsDevMode() bool {
	return qq.data.IsDevMode
}
*/

func (qq *SystemConfig) IsIdentityEncryptedWithPassphrase() bool {
	return qq.data.IsIdentityEncryptedWithPassphrase
}

func (qq *SystemConfig) Unlock(passphrase string) error {
	if !qq.IsAppLocked() {
		return e.NewHTTPErrorf(http.StatusBadRequest, "App is already unlocked.")
	}

	x25519Identity, err := qq.decryptMainIdentity(passphrase)
	if err != nil {
		log.Println(err)
		return err
	}

	encryptor.NilableX25519MainIdentity = x25519Identity
	// qq.nilableX25519Identity = x25519Identity
	return nil
}

// identity is not set at encryptor.X25519MainIdentity
// TODO where is a good location for this?
func DecryptMainIdentity(encryptedIdentity []byte, passphrase string) (*age.X25519Identity, error) {
	passphraseIdentity, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	plaintextIdentity := &bytes.Buffer{}
	identityReader := bytes.NewReader(encryptedIdentity)

	decryptor, err := age.Decrypt(identityReader, passphraseIdentity)
	if err != nil {
		var noMatchErr *age.NoIdentityMatchError
		if errors.As(err, &noMatchErr) { // errors.Is seems not to work
			// TODO correct status code?
			return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid passphrase.")
		}
		log.Println(err)
		return nil, err
	}

	_, err = io.Copy(plaintextIdentity, decryptor)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	x25519Identity, err := age.ParseX25519Identity(plaintextIdentity.String())
	if err != nil {
		log.Println(err, "could not parse identity")
		return nil, nil
	}

	return x25519Identity, nil

}

func (qq *SystemConfig) decryptMainIdentity(passphrase string) (*age.X25519Identity, error) {
	if !qq.data.IsIdentityEncryptedWithPassphrase {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "App is not encrypted with passphrase.") // TODO?
	}

	return DecryptMainIdentity(qq.data.X25519Identity, passphrase)
}

func (qq *SystemConfig) RemovePassphrase(ctx ctxx.Context, currentPassphrase string) error {
	if !qq.data.IsIdentityEncryptedWithPassphrase {
		return e.NewHTTPErrorf(http.StatusBadRequest, "No passphrase set.")
	}

	// oldPassphrase is technically not required, because we could also reencrypt encryptor.X25519MainIdentity
	// but is safer against accidental change...
	// this way it will also work if app is still locked...
	mainX25519Identity, err := qq.decryptMainIdentity(currentPassphrase)
	if err != nil {
		log.Println(err)
		return err
	}

	// updating encryptor.X25519MainIdentity not necessary because has not changed, just
	// got encrypted in database

	qq.data = qq.data.Update().
		SetIsIdentityEncryptedWithPassphrase(false).
		SetX25519Identity([]byte(mainX25519Identity.String())).
		SaveX(ctx)

	return nil
}

func (qq *SystemConfig) ChangePassphrase(
	ctx ctxx.Context,
	currentPassphrase, newPassphrase, confirmNewPassphrase string,
) error {
	if newPassphrase == "" {
		return e.NewHTTPErrorf(http.StatusBadRequest, "New passphrase is required.")
	}
	if newPassphrase != confirmNewPassphrase {
		return e.NewHTTPErrorf(http.StatusBadRequest, "New passphrase does not match confirmation.")
	}
	if qq.data.IsIdentityEncryptedWithPassphrase && currentPassphrase == "" {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Old passphrase is required.")
	}

	var mainX25519Identity *age.X25519Identity
	var err error
	if qq.data.IsIdentityEncryptedWithPassphrase {
		// oldPassphrase is technically not required, because we could also reencrypt encryptor.X25519MainIdentity
		// but is safer against accidental change...
		// this way it will also work if app is still locked...
		mainX25519Identity, err = qq.decryptMainIdentity(currentPassphrase)
		if err != nil {
			log.Println(err)
			return err
		}
	} else {
		mainX25519Identity, err = age.ParseX25519Identity(string(qq.data.X25519Identity))
		if err != nil {
			log.Println(err, "could not parse identity")
			return err
		}
	}

	recipient, err := age.NewScryptRecipient(newPassphrase)
	if err != nil {
		log.Println(err)
		return err
	}

	identityCiphertext := &bytes.Buffer{}
	encryptorx, err := age.Encrypt(identityCiphertext, recipient)
	if err != nil {
		log.Println(err)
		return err
	}

	identityReader := strings.NewReader(mainX25519Identity.String())

	_, err = io.Copy(encryptorx, identityReader)
	if err != nil {
		log.Println(err)
		return err
	}

	// TODO close in defer on error? or cleaned up on return anyway?
	err = encryptorx.Close()
	if err != nil {
		log.Println(err)
		return err
	}

	identityBytes := identityCiphertext.Bytes()
	qq.data = qq.data.Update().
		SetIsIdentityEncryptedWithPassphrase(true).
		SetX25519Identity(identityBytes).
		SaveX(ctx)

	// updating encryptor.X25519MainIdentity not necessary because has not changed, just
	// got encrypted in database

	return nil
}

// TODO best location? App struct might be better
func (qq *SystemConfig) IsAppLocked() bool {
	return qq.NilableX25519Identity() == nil
}

func (qq *SystemConfig) NilableX25519Identity() *age.X25519Identity {
	if encryptor.NilableX25519MainIdentity != nil {
		return encryptor.NilableX25519MainIdentity
	}

	if qq.data.IsIdentityEncryptedWithPassphrase {
		log.Println("identity is encrypted with passphrase") // TODO infoln
		return nil
	}

	identityBytes := qq.data.X25519Identity
	if len(identityBytes) == 0 {
		log.Println("no identity")
		return nil
	}
	x25519Identity, err := age.ParseX25519Identity(string(identityBytes))
	if err != nil {
		log.Println(err, "could not parse identity")
		return nil
	}

	// qq.nilableX25519Identity = x25519Identity
	encryptor.NilableX25519MainIdentity = x25519Identity
	return encryptor.NilableX25519MainIdentity
}

func (qq *SystemConfig) IsSaaSModeEnabled() bool {
	return qq.isSaaSModeEnabled
}

func (qq *SystemConfig) CommercialLicenseEnabled() bool {
	return qq.commercialLicenseEnabled
}

func (qq *SystemConfig) AllowInsecureCookies() bool {
	return qq.allowInsecureCookies
}

func (qq *SystemConfig) MaxUploadSizeBytes() int64 {
	if qq.data.MaxUploadSizeMib <= 0 {
		return 0
	}

	return qq.data.MaxUploadSizeMib * bytesPerMiB
}

func (qq *SystemConfig) SetMaxUploadSizeMib(ctx ctxx.Context, maxUploadSizeMib int64) error {
	if maxUploadSizeMib < 0 {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Max upload size must be greater than or equal to 0.")
	}

	qq.data = ctx.MainCtx().MainTx.SystemConfig.UpdateOneID(qq.data.ID).
		SetMaxUploadSizeMib(maxUploadSizeMib).
		SaveX(ctx)

	return nil
}

func (qq *SystemConfig) PublicOrigin() string {
	return qq.publicOrigin
}

func (qq *SystemConfig) CanonicalHost() string {
	if qq.publicOrigin == "" {
		return ""
	}

	publicOriginURL, err := url.Parse(qq.publicOrigin)
	if err != nil {
		log.Println(err)
		return ""
	}

	return strings.ToLower(publicOriginURL.Hostname())
}

func (qq *SystemConfig) WebAuthnRPID() string {
	if qq.webauthnRPID != "" {
		return qq.webauthnRPID
	}

	return qq.CanonicalHost()
}

func (qq *SystemConfig) WebAuthnRPName() string {
	if qq.webauthnRPName != "" {
		return qq.webauthnRPName
	}
	return "SimpleDMS"
}

func (qq *SystemConfig) AbsoluteURL(path string) string {
	if qq.publicOrigin == "" {
		return path
	}

	baseURL, err := url.Parse(qq.publicOrigin)
	if err != nil {
		log.Println(err)
		return path
	}

	resolvedURL, err := baseURL.Parse(path)
	if err != nil {
		log.Println(err)
		return path
	}

	return resolvedURL.String()
}

func (qq *SystemConfig) S3() *appmodel.S3Config {
	return &appmodel.S3Config{
		S3Endpoint:        qq.data.S3Endpoint,
		S3AccessKeyID:     qq.data.S3AccessKeyID,
		S3SecretAccessKey: qq.data.S3SecretAccessKey.String(),
		S3BucketName:      qq.data.S3BucketName,
		S3UseSSL:          qq.data.S3UseSsl,
	}
}

func (qq *SystemConfig) TLS() *appmodel.TLSConfig {
	return &appmodel.TLSConfig{
		TLSEnableAutocert:     qq.data.TLSEnableAutocert,
		TLSCertFilepath:       qq.data.TLSCertFilepath,
		TLSPrivateKeyFilepath: qq.data.TLSPrivateKeyFilepath,
		TLSAutocertEmail:      qq.data.TLSAutocertEmail,
		TLSAutocertHosts:      qq.data.TLSAutocertHosts,
	}
}
