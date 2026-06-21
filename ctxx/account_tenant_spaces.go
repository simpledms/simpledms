package ctxx

import "github.com/simpledms/simpledms/model/main/common/plan"

type AccountTenantSpaces struct {
	TenantID            int64
	TenantPublicID      string
	TenantName          string
	TenantPlan          plan.Plan
	IsTenantInitialized bool
	IsTenantOwner       bool
	Spaces              []AccountSpace
}

func NewAccountTenantSpaces(
	tenantID int64,
	tenantPublicID string,
	tenantName string,
	tenantPlan plan.Plan,
	isTenantInitialized bool,
	isTenantOwner bool,
	spaces []AccountSpace,
) AccountTenantSpaces {
	return AccountTenantSpaces{
		TenantID:            tenantID,
		TenantPublicID:      tenantPublicID,
		TenantName:          tenantName,
		TenantPlan:          tenantPlan,
		IsTenantInitialized: isTenantInitialized,
		IsTenantOwner:       isTenantOwner,
		Spaces:              spaces,
	}
}
