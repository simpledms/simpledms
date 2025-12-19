package scheduler

import (
	"context"
	"crypto/tls"
	"log"
	"runtime/debug"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/privacy"
	mailer "github.com/wneessen/go-mail"

	"github.com/simpledms/simpledms/app/simpledms/entmain/mail"
	"github.com/simpledms/simpledms/app/simpledms/entmain/systemconfig"
)

func (qq *Scheduler) sendMails() {
	defer func() {
		// tested and works
		if r := recover(); r != nil {
			log.Printf("%v: %s", r, debug.Stack())
			log.Println("trying to recover")

			// TODO what is a good interval
			time.Sleep(1 * time.Minute)

			// tested and works, automatically restarts loop
			qq.sendMails()
		}
	}()
	for {
		// TODO in transaction or not? if so, don't forget rollback in recovery logic

		// it's fine to send password mail even if tenant is not initialized; because login
		// works without tenant, a message can be shown on dashboard

		// read in every loop to detect possible changes...
		//
		// TODO impl a more robust solution than FirstX
		systemConfigx := qq.mainDB.ReadOnlyConn.SystemConfig.
			Query().
			Order(systemconfig.ByUpdatedAt(sql.OrderDesc())).
			FirstX(context.Background())

		if systemConfigx.MailerHost == "" {
			log.Println("mailer host not set, skipping mail sending")
			time.Sleep(1 * time.Minute)
			continue
		}

		// TODO make safe for production use
		mailClient, err := mailer.NewClient(
			systemConfigx.MailerHost,
			mailer.WithPort(systemConfigx.MailerPort),
			mailer.WithUsername(systemConfigx.MailerUsername),
			mailer.WithPassword(systemConfigx.MailerPassword.String()),
			// TODO is there a default timeout?
			// mailer.WithTimeout()
		)
		if err != nil {
			log.Println(err)
			time.Sleep(1 * time.Minute)
			continue
		}
		if systemConfigx.MailerInsecureSkipVerify {
			mailClient.SetTLSPolicy(mailer.NoTLS)
			err = mailClient.SetTLSConfig(&tls.Config{
				InsecureSkipVerify: true,
			})
			if err != nil {
				log.Println(err)
				time.Sleep(1 * time.Minute)
				continue
			}
		} else {
			mailClient.SetSMTPAuth(mailer.SMTPAuthAutoDiscover)
		}

		ctx := context.Background()
		ctx = privacy.DecisionContext(ctx, privacy.Allow)

		mails := qq.mainDB.ReadWriteConn.Mail.
			Query().
			Where(
				mail.SentAtIsNil(),
				mail.LastTriedAtLT(time.Now().Add(-1*time.Hour)),
				mail.RetryCountLT(3),
			).
			Order(mail.ByLastTriedAt(sql.OrderDesc())).
			AllX(ctx)

		for _, mailx := range mails {
			mailx.Update().SetLastTriedAt(time.Now()).AddRetryCount(1).SaveX(ctx)

			message := mailer.NewMsg()

			err := message.From(systemConfigx.MailerFrom)
			if err != nil {
				log.Println(err)
				continue
			}

			userEmail := mailx.QueryReceiver().OnlyX(ctx).Email
			err = message.To(userEmail.String())
			if err != nil {
				log.Println(err)
				continue
			}

			message.Subject(mailx.Subject)

			// Set plain text as the main body
			message.SetBodyString(mailer.TypeTextPlain, mailx.Body)

			// If HTML body exists, add it as an alternative
			if htmlBody := mailx.HTMLBody; htmlBody != "" {
				message.AddAlternativeString(mailer.TypeTextHTML, htmlBody)
			}

			err = mailClient.DialAndSend(message)
			if err != nil {
				log.Println(err)
				continue
			}

			mailx.Update().SetSentAt(time.Now()).SaveX(ctx)
		}

		time.Sleep(15 * time.Second)
	}
}
