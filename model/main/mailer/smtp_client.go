package mailer

import (
	"crypto/tls"
	"log"

	gomail "github.com/wneessen/go-mail"

	appmodel "github.com/simpledms/simpledms/model/main/app"
)

// NewSMTPClient is shared by mail sending and the system status check, so that the status
// check connects exactly like mail sending does.
func NewSMTPClient(config *appmodel.MailerConfig) (*gomail.Client, error) {
	// TODO make safe for production use
	mailClient, err := gomail.NewClient(
		config.MailerHost,
		gomail.WithPort(config.MailerPort),
		gomail.WithUsername(config.MailerUsername),
		gomail.WithPassword(config.MailerPassword),
		gomail.WithSMTPAuth(gomail.SMTPAuthAutoDiscover),
		// TODO is there a default timeout?
		// mailer.WithTimeout()
	)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if config.MailerInsecureSkipVerify {
		err = mailClient.SetTLSConfig(&tls.Config{
			// ServerName and MinVersion are set in mailer.NewClient() too;
			// could not find a way to modify default tlsConfig
			ServerName:         config.MailerHost,
			MinVersion:         gomail.DefaultTLSMinVersion,
			InsecureSkipVerify: true,
		})
		if err != nil {
			log.Println(err)
			return nil, err
		}
	}
	if config.MailerUseImplicitSSLTLS {
		// cannot use WithSSL in initialization because it doesn't
		// accept a value as input
		mailClient.SetSSL(true)
	}

	return mailClient, nil
}
