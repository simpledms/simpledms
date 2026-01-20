package ctxx

// ctxx instead of ctx and context prevents naming conflicts with var names and on import

import (
	"context"
	"log"
	"time"

	"golang.org/x/text/language"

	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/i18n"
)

// having TTx in Context allows for easier replacement of ent with jet later
// TODO VisitorContext or Context?
type VisitorContext struct {
	context.Context
	MainTx        *entmain.Tx
	Printer       *i18n.Printer
	IsHTMXRequest bool
	// IsAppLocked   bool
	LanguageBCP47 string // used in widgets
	Location      *time.Location
	// also stored in system config
	CommercialLicenseEnabled bool
}

func NewVisitorContext(
	ctx context.Context,
	mainTx *entmain.Tx,
	i18nx *i18n.I18n,
	acceptLanguageStr string,
	clientTimezone string,
	isHTMXRequest bool,
	commercialLicenseEnabled bool,
) *VisitorContext {
	var langTag language.Tag
	if acceptLanguageStr != "" {
		langMatcher := language.NewMatcher([]language.Tag{
			language.English,
			language.German,
			language.French,
			language.Italian,
		})
		tag, _, err := language.ParseAcceptLanguage(acceptLanguageStr)
		if err != nil {
			log.Println(err)
			langTag = language.English
		} else {
			langTag, _, _ = langMatcher.Match(tag...)
		}
	} else {
		langTag = language.English // Default to English if no 'Accept-Language' is provided
	}
	langTagBase, _ := langTag.Base() // TODO evaluate confidence?
	printer := i18nx.Printer(langTag)

	location, err := time.LoadLocation(clientTimezone)
	if err != nil {
		log.Println("failed to load location", clientTimezone, err)
		location = time.Local
	}

	visitorCtx := &VisitorContext{
		Context:       ctx,
		MainTx:        mainTx,
		Printer:       printer,
		IsHTMXRequest: isHTMXRequest,
		// IsAppLocked:   isAppLocked,
		LanguageBCP47:            langTagBase.String(),
		Location:                 location,
		CommercialLicenseEnabled: commercialLicenseEnabled,
	}
	visitorCtx.Context = context.WithValue(ctx, visitorCtxKey, visitorCtx)
	return visitorCtx
}

func (qq *VisitorContext) VisitorCtx() *VisitorContext {
	return qq
}

func (qq *VisitorContext) MainCtx() *MainContext {
	panic("context not available")
}

func (qq *VisitorContext) TenantCtx() *TenantContext {
	panic("context not available")
}

func (qq *VisitorContext) SpaceCtx() *SpaceContext {
	panic("context not available")
}

func (qq *VisitorContext) IsVisitorCtx() bool {
	return true
}

func (qq *VisitorContext) IsMainCtx() bool {
	return false
}

func (qq *VisitorContext) IsTenantCtx() bool {
	return false
}

func (qq *VisitorContext) IsSpaceCtx() bool {
	return false
}
