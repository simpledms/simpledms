package modelmain

import (
	"log"
	"net/http"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/tenant"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/mainrole"
	"github.com/simpledms/simpledms/util/e"
)

type UploadLimitService struct{}

func NewUploadLimitService() *UploadLimitService {
	return &UploadLimitService{}
}

func (qq *UploadLimitService) SetGlobalUploadLimit(
	ctx ctxx.Context,
	systemConfig *SystemConfig,
	isUnlimited bool,
	maxUploadSizeMib int64,
) (*UploadLimit, error) {
	err := qq.ensureAdminCtx(ctx)
	if err != nil {
		return nil, err
	}

	uploadLimit, err := NewUploadLimitFromForm(isUnlimited, maxUploadSizeMib)
	if err != nil {
		return nil, err
	}

	err = systemConfig.SetMaxUploadSizeMib(ctx, uploadLimit.MiB())
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return uploadLimit, nil
}

func (qq *UploadLimitService) SetTenantUploadLimitOverride(
	ctx ctxx.Context,
	tenantPublicID string,
	useGlobalDefault bool,
	isUnlimited bool,
	maxUploadSizeMib int64,
) (*UploadLimit, error) {
	err := qq.ensureAdminCtx(ctx)
	if err != nil {
		return nil, err
	}
	if tenantPublicID == "" {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Tenant is required.")
	}

	tenantx, err := ctx.MainCtx().MainTx.Tenant.Query().
		Where(tenant.PublicID(entx.NewCIText(tenantPublicID))).
		Only(ctx)
	if err != nil {
		log.Println(err)
		if entmain.IsNotFound(err) {
			return nil, e.NewHTTPErrorf(http.StatusNotFound, "Tenant not found.")
		}
		return nil, err
	}

	updateQuery := tenantx.Update()
	if useGlobalDefault {
		updateQuery.ClearMaxUploadSizeMibOverride()
	} else {
		uploadLimit, err := NewUploadLimitFromForm(isUnlimited, maxUploadSizeMib)
		if err != nil {
			return nil, err
		}

		updateQuery.SetMaxUploadSizeMibOverride(uploadLimit.MiB())
	}

	tenantx, err = updateQuery.Save(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if ctx.IsTenantCtx() && ctx.TenantCtx().Tenant.ID == tenantx.ID {
		ctx.TenantCtx().Tenant = tenantx
	}
	if tenantx.MaxUploadSizeMibOverride == nil {
		return nil, nil
	}

	return NewUploadLimitFromMiB(*tenantx.MaxUploadSizeMibOverride)
}

func (qq *UploadLimitService) ensureAdminCtx(ctx ctxx.Context) error {
	if !ctx.IsMainCtx() {
		return e.NewHTTPErrorf(http.StatusUnauthorized, "You must be logged in to manage upload limits.")
	}
	if ctx.MainCtx().Account.Role != mainrole.Admin {
		return e.NewHTTPErrorf(http.StatusForbidden, "You must be an admin to manage upload limits.")
	}

	return nil
}
