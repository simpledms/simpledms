package modelmain

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"filippo.io/age"

	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	"github.com/simpledms/simpledms/app/simpledms/encryptor"
	"github.com/simpledms/simpledms/app/simpledms/entmain"
	"github.com/simpledms/simpledms/app/simpledms/entx"
	"github.com/simpledms/simpledms/util/e"
)

// TODO remove prefix from all certs as soon as forms support fieldgroups

type MailerConfig struct {
	MailerHost               string `validate:"required"`
	MailerPort               int    `validate:"required"`
	MailerUsername           string `validate:"required"`
	MailerPassword           string `validate:"required"`
	MailerFrom               string `validate:"required"`
	MailerInsecureSkipVerify bool
}

type S3Config struct {
	S3Endpoint        string `validate:"required"`
	S3AccessKeyID     string `validate:"required"`
	S3SecretAccessKey string `validate:"required"`
	S3BucketName      string `validate:"required"`
	S3UseSSL          bool   // only allowed in dev...
}

type TLSConfig struct {
	TLSEnableAutocert     bool
	TLSCertFilepath       string
	TLSPrivateKeyFilepath string
	TLSAutocertEmail      string
	TLSAutocertHosts      []string
}

type OCRConfig struct {
	TikaURL string // optional, can also be used without OCR
}

func InitApp(
	ctx ctxx.Context,
	// isDevMode bool,
	passphrase string,
	acceptEmptyPassphrase bool, // just for safety
	s3Config S3Config,
	tlsConfig TLSConfig,
	mailerConfig MailerConfig,
	ocrConfig OCRConfig,
) error {
	return InitAppWithoutCustomContext(
		ctx,
		// TODO still within current Tx?
		//  comment can be interpreted this way...
		ctx.MainCtx().MainTx,
		// isDevMode,
		passphrase,
		acceptEmptyPassphrase,
		s3Config,
		tlsConfig,
		mailerConfig,
		ocrConfig,
	)
}

// used for dev mode initialization
func InitAppWithoutCustomContext(
	ctx context.Context,
	mainDB *entmain.Tx,
	// isDevMode bool,
	passphrase string,
	acceptEmptyPassphrase bool, // just for safety
	s3Config S3Config,
	tlsConfig TLSConfig,
	mailerConfig MailerConfig,
	ocrConfig OCRConfig,
) error {
	configCount := mainDB.SystemConfig.Query().CountX(ctx)
	if configCount > 0 {
		return e.NewHTTPErrorf(http.StatusBadRequest, "App already initialized.")
	}
	if passphrase == "" && !acceptEmptyPassphrase {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Passphrase is required.")
	}

	x25519identity, err := age.GenerateX25519Identity()
	if err != nil {
		log.Println(err)
		return err
	}
	encryptor.NilableX25519MainIdentity = x25519identity

	isIdentityPassphraseEncrypted := passphrase != ""
	var identityBytes []byte
	if isIdentityPassphraseEncrypted {
		recipient, err := age.NewScryptRecipient(passphrase)
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

		identityReader := strings.NewReader(x25519identity.String())

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

		identityBytes = identityCiphertext.Bytes()
	} else {
		identityBytes = []byte(x25519identity.String())
	}

	mainDB.SystemConfig.Create().
		// SetIsDevMode(isDevMode).
		// identity
		SetX25519Identity(identityBytes).
		SetIsIdentityEncryptedWithPassphrase(isIdentityPassphraseEncrypted).
		// s3
		SetS3Endpoint(s3Config.S3Endpoint).
		SetS3AccessKeyID(s3Config.S3AccessKeyID).
		SetS3SecretAccessKey(entx.NewEncryptedString(s3Config.S3SecretAccessKey)).
		SetS3BucketName(s3Config.S3BucketName).
		SetS3UseSsl(s3Config.S3UseSSL).
		// tls
		SetTLSEnableAutocert(tlsConfig.TLSEnableAutocert).
		SetTLSCertFilepath(tlsConfig.TLSCertFilepath).
		SetTLSPrivateKeyFilepath(tlsConfig.TLSPrivateKeyFilepath).
		SetTLSAutocertEmail(tlsConfig.TLSAutocertEmail).
		SetTLSAutocertHosts(tlsConfig.TLSAutocertHosts).
		// mailer
		SetMailerHost(mailerConfig.MailerHost).
		SetMailerPort(mailerConfig.MailerPort).
		SetMailerUsername(mailerConfig.MailerUsername).
		SetMailerPassword([]byte(mailerConfig.MailerPassword)).
		SetMailerFrom(mailerConfig.MailerFrom).
		SetMailerInsecureSkipVerify(mailerConfig.MailerInsecureSkipVerify).
		// ocr
		SetOcrTikaURL(ocrConfig.TikaURL).
		// other
		SetInitializedAt(time.Now()).
		SaveX(ctx)

	return nil

}
