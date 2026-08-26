package dashboard

import (
	"fmt"
	"sort"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/main/systemconfig"
	"github.com/simpledms/simpledms/util/httpx"
)

type webDAVCredentialDestination struct {
	tenantID       int64
	tenantPublicID string
	tenantName     string
	spacePublicID  string
	spaceName      string
	label          string
}

func webDAVCredentialDestinations(ctx ctxx.Context) ([]*webDAVCredentialDestination, error) {
	spacesByTenant, err := ctx.MainCtx().ReadOnlyAccountSpacesByTenant()
	if err != nil {
		return nil, err
	}

	var destinations []*webDAVCredentialDestination
	for tenantx, spaces := range spacesByTenant {
		for _, spacex := range spaces {
			destinations = append(destinations, &webDAVCredentialDestination{
				tenantID:       tenantx.ID,
				tenantPublicID: tenantx.PublicID.String(),
				tenantName:     tenantx.Name,
				spacePublicID:  spacex.PublicID.String(),
				spaceName:      spacex.Name,
				label:          fmt.Sprintf("%s: %s", tenantx.Name, spacex.Name),
			})
		}
	}
	sort.Slice(destinations, func(i, j int) bool {
		return destinations[i].label < destinations[j].label
	})
	return destinations, nil
}

func (qq *webDAVCredentialDestination) key() string {
	return webDAVCredentialDestinationKey(qq.tenantID, qq.spacePublicID)
}

func (qq *webDAVCredentialDestination) value() string {
	return qq.tenantPublicID + ":" + qq.spacePublicID
}

func (qq *webDAVCredentialDestination) url(
	config *systemconfig.SystemConfig,
	req *httpx.Request,
) string {
	return webDAVCredentialURL(config, req, qq.tenantPublicID, qq.spacePublicID)
}

func webDAVCredentialDestinationKey(tenantID int64, spacePublicID string) string {
	return fmt.Sprintf("%d:%s", tenantID, spacePublicID)
}

func webDAVCredentialURL(
	config *systemconfig.SystemConfig,
	req *httpx.Request,
	tenantPublicID string,
	spacePublicID string,
) string {
	path := fmt.Sprintf("/webdav/%s/%s/", tenantPublicID, spacePublicID)
	if absoluteURL := config.AbsoluteURL(path); absoluteURL != path {
		return absoluteURL
	}
	if req == nil || req.Host == "" {
		return path
	}
	scheme := "https"
	if req.TLS == nil {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s%s", scheme, req.Host, path)
}
