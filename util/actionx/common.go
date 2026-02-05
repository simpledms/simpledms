package actionx

import (
	"fmt"
	"net/url"
	"strings"
)

type Config struct {
	endpoint   string
	method     string
	isReadOnly bool
	// some actions open their own write transactions and should avoid long-lived
	// request transactions
	useManualTxManagement bool
}

func NewConfig(
	endpoint string,
	// method string,
	isReadOnly bool,
) *Config {
	return &Config{
		endpoint:   endpoint,
		method:     "POST",
		isReadOnly: isReadOnly,
	}
}

func (qq *Config) Route() string {
	return fmt.Sprintf("%s %s", qq.method, qq.Endpoint())
}

// TODO rename to URL?
func (qq *Config) Endpoint() string {
	return "/-/" + strings.TrimPrefix(qq.endpoint, "/")
}

// TODO rename to FormURL?
func (qq *Config) FormEndpoint() string {
	return "/-/" + strings.TrimPrefix(qq.endpoint, "/") + "-form"
}

/*
	func (qq *Config) FormEndpointWithParams2(params url.Values) string {
		return "/-/partial" + qq.endpoint + "-form?" + params.Encode()
	}
*/

func (qq *Config) FormEndpointWithParams(wrapper ResponseWrapper, hxTarget string) string {
	return qq.endpointWithParams(qq.FormEndpoint(), wrapper, hxTarget)
}

func (qq *Config) EndpointWithParams(wrapper ResponseWrapper, hxTarget string) string {
	return qq.endpointWithParams(qq.Endpoint(), wrapper, hxTarget)
}

// TODO rename to something more meaningful, also var where value is used
func (qq *Config) IsReadOnly() bool {
	return qq.isReadOnly || qq.UseManualTxManagement()
}

func (qq *Config) UseManualTxManagement() bool {
	return qq.useManualTxManagement
}

func (qq *Config) EnableManualTxManagement() *Config {
	qq.useManualTxManagement = true
	return qq
}

// TODO return url?
/*
func (qq *Config) EndpointWithState(state any, wrapper ResponseWrapper, hxTarget string) string {
	if state == nil {
		return qq.endpointWithParams(qq.Endpoint(), wrapper, hxTarget)
	}

	values := url.Values{}

	encoder := schema.NewEncoder()
	encoder.SetAliasTag("url")

	err := encoder.Encode(state, values)
	if err != nil {
		log.Println(err)
		return qq.endpointWithParams(qq.Endpoint(), wrapper, hxTarget)
	}

	values.Set("hx-target", hxTarget)

	urlx, err := url.Parse(qq.Endpoint())
	if err != nil {
		panic(err)
	}

	urlx.RawQuery = values.Encode()
	return urlx.String()
}
*/

// TODO use struct instead?
func (qq *Config) endpointWithParams(endpoint string, wrapper ResponseWrapper, hxTarget string) string {
	urlx, err := url.Parse(endpoint)
	if err != nil {
		panic(err)
	}
	query := urlx.Query()

	// Encode doesn't remove empty assignments, thus check if set
	if wrapper != "" {
		query.Set("wrapper", wrapper.String())
	}
	if hxTarget != "" {
		query.Set("hx-target", hxTarget)
	}

	urlx.RawQuery = query.Encode()
	return urlx.String()
}

func (qq *Config) FormRoute() string {
	if qq.isReadOnly && !qq.useManualTxManagement {
		return ""
	}
	return fmt.Sprintf("POST %s", qq.FormEndpoint())
}

/*
func (qq *Common) RouteHandler() func(http.ResponseWriter, *http.Request) {
	return qq.Handler
}

func (qq *Common) Register(router *http.ServeMux) {
	router.HandleFunc(qq.Route(), qq.Handler)
	// TODO handle if not available
	router.HandleFunc(qq.FormRoute(), nil) // TODO
}
*/
