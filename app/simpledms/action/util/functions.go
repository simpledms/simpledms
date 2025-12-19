package util

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/go-playground/form"
	"github.com/go-playground/validator/v10"

	"github.com/google/uuid"

	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	"github.com/simpledms/simpledms/app/simpledms/model/common/country"
	"github.com/simpledms/simpledms/app/simpledms/model/common/fieldtype"
	"github.com/simpledms/simpledms/app/simpledms/model/common/language"
	"github.com/simpledms/simpledms/app/simpledms/model/common/spacerole"
	"github.com/simpledms/simpledms/app/simpledms/model/common/tenantrole"
	"github.com/simpledms/simpledms/app/simpledms/model/tagging/tagtype"
	"github.com/simpledms/simpledms/app/simpledms/renderable"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
	"github.com/simpledms/simpledms/util/timex"
)

func QueryHeader(endpoint string, data any) template.JS {
	values := url.Values{}

	if data != nil {
		// encoder := schema.NewEncoder()
		encoder := form.NewEncoder()
		var err error
		values, err = encoder.Encode(data)
		if err != nil {
			log.Println(err)
			panic(err)
		}
	}

	// TODO json?
	return util.JSON(struct {
		XQueryEndpoint string `json:"X-Query-Endpoint"` // TODO Partial or Route or Endpoint?
		XQueryData     any    `json:"X-Query-Data"`     // TODO Data or Form or Vals?
	}{
		XQueryEndpoint: endpoint,
		XQueryData:     values.Encode(),
	})
}

/*
// should only used rarely, for example if loading a partial is not enough or not possible;
// On the dashboard it is for example used because there is no DashboardCards command
// because of lazyness...
func RefreshPageHeader() template.JS {
	return util.JSON(struct {
		XRefreshPage bool `json:"X-Refresh-Page"`
	}{
		XRefreshPage: true,
	})
}
*/

// TODO where is a ood location for this? bind to HTMXAttrs?
func ResetStateHeader() template.JS {
	return util.JSON(struct {
		ResetState bool `json:"Reset-State"`
	}{
		ResetState: true,
	})
}

func CloseDetailsHeader() template.JS {
	return util.JSON(struct {
		CloseDetails bool `json:"Close-Details"`
	}{
		CloseDetails: true, // TODO or ID?
	})
}

func CloseDialogHeader() template.JS {
	return util.JSON(struct {
		CloseDialog bool `json:"Close-Dialog"`
	}{
		CloseDialog: true,
	})
}

// anonymous functions don't support generics...
/*
was for gorilla/schema
func converterFunc[T any](strFunc func(str string) (T, error)) func(s string) reflect.Value {
	return func(str string) reflect.Value {
		val, err := strFunc(str)
		if err != nil {
			log.Println(err)
			// FIXME how to return an error?
			//		decoder uses reflect.Value.IsValid() to decide if valid
			//		this method checks val.flag != 0
			return reflect.ValueOf(nil)
		}
		return reflect.ValueOf(val)
	}
}*/
func converterFunc[T any](strFunc func(str string) (T, error)) func(vals []string) (interface{}, error) {
	return func(vals []string) (interface{}, error) {
		val, err := strFunc(vals[0])
		if err != nil {
			log.Println(err)
			return nil, err
		}
		return val, nil
	}
}

// TODO make private
func FormData[T any](rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) (*T, error) {
	return FormDataX[T](rw, req, ctx, false)
}

// skipValidation is necessary because function is also used when opening form, not just when it
// gets submitted
func FormDataX[T any](
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
	skipValidation bool,
) (*T, error) {
	data := new(T)

	// necessary because can be a value like:
	// multipart/form-data; boundary=----WebKitFormBoundaryAb4I2KBu0OZAKU8h
	contentType := req.Header.Get("Content-Type")
	contentTypeArr := strings.Split(contentType, ";")
	contentType = contentTypeArr[0]

	if contentType == "multipart/form-data" {
		// 50 MB in memory, rest in tmp file
		err := req.ParseMultipartForm(50 * 1024)
		if err != nil {
			log.Println(err)
			return data, e.NewHTTPErrorf(http.StatusBadRequest, "cannot parse file")
		}
	} else {
		err := req.ParseForm()
		if err != nil {
			log.Println(err)
			return data, e.NewHTTPErrorf(http.StatusBadRequest, "cannot parse form")
		}
	}

	decoder := form.NewDecoder()
	decoder.RegisterCustomTypeFunc(func(vals []string) (interface{}, error) {
		return timex.ParseDate(vals[0])
	}, timex.Date{})
	decoder.RegisterCustomTypeFunc(converterFunc[tagtype.TagType](tagtype.TagTypeString), tagtype.Simple)
	decoder.RegisterCustomTypeFunc(converterFunc[country.Country](country.CountryString), country.Unknown)
	decoder.RegisterCustomTypeFunc(converterFunc[language.Language](language.LanguageString), language.Unknown)
	decoder.RegisterCustomTypeFunc(converterFunc[fieldtype.FieldType](fieldtype.FieldTypeString), fieldtype.Unknown)
	decoder.RegisterCustomTypeFunc(converterFunc[tenantrole.TenantRole](tenantrole.TenantRoleString), tenantrole.User)
	decoder.RegisterCustomTypeFunc(converterFunc[spacerole.SpaceRole](spacerole.SpaceRoleString), spacerole.User)

	/*
		decoder := schema.NewDecoder()
		decoder.IgnoreUnknownKeys(true)

		// TODO find a better (generic) solution that works with all enums without manually defining them;
		//		maybe switch to a library to supports implementing an interface for custom decoding
		decoder.RegisterConverter(tagtype.Simple, converterFunc[tagtype.TagType](tagtype.TagTypeString))
		decoder.RegisterConverter(country.Unknown, converterFunc[country.Country](country.CountryString))
		decoder.RegisterConverter(language.Unknown, converterFunc[language.Language](language.LanguageString))
		decoder.RegisterConverter(propertytype.Unknown, converterFunc[propertytype.PropertyType](propertytype.PropertyTypeString))
		decoder.RegisterConverter(timex.Date{}, func(str string) reflect.Value {
			datex, err := timex.ParseDate(str)
			if err != nil {
				// FIXME how to return an error?
				//		decoder uses reflect.Value.IsValid() to decide if valid
				//		this method checks val.flag != 0
				return reflect.ValueOf(nil)
			}
			return reflect.ValueOf(datex)
		})
		// decoder.RegisterConverter()

	*/

	err := decoder.Decode(data, req.PostForm)
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "cannot decode form")
	}

	if !skipValidation {
		// withRequiredStructEnabled becomes default in v11
		validatorx := validator.New(validator.WithRequiredStructEnabled())
		/*
			enx := en.New()
			dex := de.New()
			translator := ut.New(enx, dex, enx)
			// FIXME use from account
			ctx.IsTenantCtx()
			translatorx, _ := translator.GetTranslator(dex.Locale()) // found not checked because we have fallback

			if err := en_translations.RegisterDefaultTranslations(validatorx, translatorx); err != nil {
				log.Println(err)
			}
			// if err := de_translations.RegisterDefaultTranslations(validatorx, translatorx); err != nil {
			// log.Println(err)
			// }
		*/

		validationErrsUntyped := validatorx.Struct(data)
		if validationErrsUntyped != nil {
			var invalidValidationError *validator.InvalidValidationError
			if errors.As(validationErrsUntyped, &invalidValidationError) {
				log.Println(validationErrsUntyped)
				// TODO status correct? error means incorrectly configured?
				return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Cannot validate form.")
			}

			var validationErrs validator.ValidationErrors
			if errors.As(validationErrsUntyped, &validationErrs) {
				for _, erry := range validationErrs {
					log.Println(erry.Namespace())
					log.Println(erry.Field())
					log.Println(erry.StructNamespace())
					log.Println(erry.StructField())
					log.Println(erry.Tag())
					log.Println(erry.ActualTag())
					log.Println(erry.Kind())
					log.Println(erry.Type())
					log.Println(erry.Value())
					log.Println(erry.Param())
					log.Println()
				}

				// TODO is it a good choice to only show first error?
				//		more might be overwhelming, but is it guaranteed that the first one
				//		is the most relevant?
				/*
					errMessage := wx.T("Form validation failed.").String(ctx)
					if len(validationErrs) > 0 {
						errMessage = validationErrs[0].Translate(translatorx)
					}
				*/

				// return nil, e.NewHTTPErrorf(http.StatusBadRequest, errMessage)
			}

			return nil, e.NewHTTPErrorf(http.StatusBadRequest, wx.T("Form validation failed.").String(ctx))
		}
	}

	/*
		// Check if data has field SpaceID and set it
		// TODO can this be done more type safe?
		if field := reflect.ValueOf(data).Elem().FieldByName("SpaceID"); field.IsValid() && field.CanSet() {
			ctx.SpaceID = field.Int()
		}
	*/

	return data, nil
}

func StateX[T any](rw httpx.ResponseWriter, req *httpx.Request) *T {
	state, err := State[T](rw, req)
	if err != nil {
		log.Println(err)
		// don't do anything, just discard state, shouldn't be a big deal in most cases
	}
	return state
}

func State[T any](rw httpx.ResponseWriter, req *httpx.Request) (*T, error) {
	// TODO is req.URL.Query() fallback necessary?

	data := new(T)
	values := url.Values{}

	if req.Header.Get("Reset-State") != "" {
		// Hx-Trigger check is workaround to check if the user actively clicked on a button, especially the
		// reset button; without this check, the message is also shown when the state is reset indirectly,
		// for example by switching folders; not 100 percent sure if this works in all use cases...
		if req.Header.Get("Hx-Trigger") != "" {
			rw.AddRenderables(wx.NewSnackbarf("Filters successfully reset."))
		}
		return data, nil
	}

	// check is necessary for example for boosted link from /select-space/ to /inbox/ with upload_token as query
	// param because HX-Current-URL is set to /select-space/ url in this scenario when the /inbox/ page gets loaded
	// added on 1 April 2025;
	// commented on 12 April 2025 because it had the side effect that switching via list between files
	// did not preserve the url params, for example activeTab
	// isCommand := strings.HasPrefix(req.URL.Path, "/-/cmd/")

	currentURLStr := req.Header.Get("HX-Current-URL")
	if currentURLStr != "" {
		currentURL, err := url.Parse(currentURLStr)
		if err != nil {
			log.Println(err)
			return data, e.NewHTTPErrorf(http.StatusBadRequest, "cannot parse current url")
		}
		values = currentURL.Query()

		// for GET requests the params of the requested URL should be processed to, this is for example necessary
		// for boosted link from /select-space/ to /inbox/ with upload_token as query
		// param because HX-Current-URL is set to /select-space/ url in this scenario when the /inbox/ page gets loaded
		// added on 12 April 2025
		if req.Method == http.MethodGet {
			for key, value := range req.URL.Query() {
				values[key] = value
			}
		}
	} else {
		// initial direct request, non htmx
		values = req.URL.Query()
	}

	return stateFromQuery[T](values)
}

func stateFromQuery[T any](urlValues url.Values) (*T, error) {
	data := new(T)

	decoder := form.NewDecoder()
	decoder.SetTagName("url")

	/*
		decoder := schema.NewDecoder()
		decoder.IgnoreUnknownKeys(true)

		// TODO would it be better to do default (route) first?
		decoder.SetAliasTag("url")
	*/
	err := decoder.Decode(data, urlValues)
	if err != nil {
		log.Println(err)
		return data, e.NewHTTPErrorf(http.StatusBadRequest, "Cannot decode url query.")
	}

	return data, nil
}

var regexpAlpha = regexp.MustCompile("[^a-zA-Z]+")

// never use uuid.NewString() directly because it doesn't generate valid IDs all the time,
// HTML IDs cannot start with a number, but UUIDs can
func GenerateID(title string) string {
	// FIXME is this random enough?
	return regexpAlpha.ReplaceAllString(title, "") + "-" + uuid.NewString() // uuid.NewString() panics if not successful
}

func WrapWidget(
	headline *wx.Text,
	submitLabel *wx.Text,
	form renderable.Renderable,
	wrapper actionx.ResponseWrapper,
	dialogLayout wx.DialogLayout,
) renderable.Renderable {
	return WrapWidgetWithID(headline, submitLabel, form, wrapper, dialogLayout, "", "")
}

func WrapWidgetWithID(
	headline *wx.Text,
	submitLabel *wx.Text,
	form renderable.Renderable,
	wrapper actionx.ResponseWrapper,
	dialogLayout wx.DialogLayout,
	id string,
	formID string,
) renderable.Renderable {
	if wrapper == actionx.ResponseWrapperDialog {
		return &wx.Dialog{
			Widget: wx.Widget[wx.Dialog]{
				ID: id,
			},
			Headline:     headline,
			SubmitLabel:  submitLabel,
			FormID:       formID,
			IsOpenOnLoad: true,
			Layout:       dialogLayout,
			// ContentID:    id,
			Child: form,
		}
	}

	return form
}
