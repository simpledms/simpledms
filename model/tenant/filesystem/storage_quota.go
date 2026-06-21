package filesystem

import (
	"math"
	"net/http"
	"time"

	"entgo.io/ent/privacy"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/entmain/tenantaccountassignment"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/storedfile"
	"github.com/simpledms/simpledms/model/main/common/plan"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/fileutil"
)

const tenantQuotaTrialBytes int64 = 1 * 1024 * 1024 * 1024
const tenantQuotaProPerUserBytes int64 = 5 * 1024 * 1024 * 1024
const tenantQuotaUnlimitedBytes int64 = 500 * 1024 * 1024 * 1024

type StorageQuota struct {
	isSaaSModeEnabled bool
}

func NewStorageQuota(isSaaSModeEnabled bool) *StorageQuota {
	return &StorageQuota{
		isSaaSModeEnabled: isSaaSModeEnabled,
	}
}

func (qq *StorageQuota) TenantUsageBytes(ctx ctxx.Context) (int64, int64, error) {
	tenantCtx := ctx.TenantCtx()
	return qq.TenantUsageBytesForTenant(
		ctx,
		tenantCtx.TTx,
		tenantCtx.Tenant.ID,
		tenantCtx.Tenant.Plan,
	)
}

// TenantUsageBytesForTenant returns the current storage usage and storage limit for a tenant.
func (qq *StorageQuota) TenantUsageBytesForTenant(
	ctx ctxx.Context,
	tenantTx *enttenant.Tx,
	tenantID int64,
	tenantPlan plan.Plan,
) (int64, int64, error) {
	limitBytes, err := qq.tenantStorageLimitBytes(ctx, tenantID, tenantPlan)
	if err != nil {
		return 0, 0, err
	}

	usedBytes, err := qq.currentUsedTenantStorageBytes(ctx, tenantTx)
	if err != nil {
		return 0, 0, err
	}

	return usedBytes, limitBytes, nil
}

func (qq *StorageQuota) EnsureTenantStorageLimit(ctx ctxx.Context, incomingUploadedBytes int64) error {
	if !qq.isSaaSModeEnabled {
		return nil
	}
	if incomingUploadedBytes <= 0 {
		return nil
	}

	usedBytes, limitBytes, err := qq.TenantUsageBytes(ctx)
	if err != nil {
		return e.NewHTTPErrorf(http.StatusInternalServerError, "Could not verify storage limit.")
	}

	if qq.exceedsStorageLimit(usedBytes, incomingUploadedBytes, limitBytes) {
		return e.NewHTTPErrorf(
			http.StatusRequestEntityTooLarge,
			"Storage limit reached for this organization. Used: %s of %s.",
			fileutil.FormatSize(usedBytes),
			fileutil.FormatSize(limitBytes),
		)
	}

	return nil
}

func (qq *StorageQuota) tenantStorageLimitBytes(
	ctx ctxx.Context,
	tenantID int64,
	tenantPlan plan.Plan,
) (int64, error) {
	if !qq.planNeedsActiveUserCount(tenantPlan) {
		return qq.LimitBytesForPlan(tenantPlan, 0), nil
	}

	activeUserCount, err := qq.activeTenantUserCount(ctx, tenantID)
	if err != nil {
		return 0, err
	}

	return qq.LimitBytesForPlan(tenantPlan, activeUserCount), nil
}

func (qq *StorageQuota) activeTenantUserCount(ctx ctxx.Context, tenantID int64) (int, error) {
	activeUserCount, err := ctx.MainCtx().MainTx.TenantAccountAssignment.Query().
		Where(
			tenantaccountassignment.TenantID(tenantID),
			tenantaccountassignment.Or(
				tenantaccountassignment.ExpiresAtIsNil(),
				tenantaccountassignment.ExpiresAtGT(time.Now()),
			),
			tenantaccountassignment.HasAccountWith(account.DeletedAtIsNil()),
		).
		Count(ctx)
	if err != nil {
		return 0, err
	}

	return activeUserCount, nil
}

func (qq *StorageQuota) planNeedsActiveUserCount(tenantPlan plan.Plan) bool {
	return tenantPlan == plan.Pro
}

func (qq *StorageQuota) LimitBytesForPlan(tenantPlan plan.Plan, activeUserCount int) int64 {
	if tenantPlan == plan.Trial {
		return tenantQuotaTrialBytes
	}
	if tenantPlan == plan.Unlimited {
		return tenantQuotaUnlimitedBytes
	}
	if tenantPlan != plan.Pro {
		return tenantQuotaTrialBytes
	}

	if activeUserCount <= 0 {
		return 0
	}
	if int64(activeUserCount) > math.MaxInt64/tenantQuotaProPerUserBytes {
		return math.MaxInt64
	}

	return int64(activeUserCount) * tenantQuotaProPerUserBytes
}

func (qq *StorageQuota) currentUsedTenantStorageBytes(
	ctx ctxx.Context,
	tenantTx *enttenant.Tx,
) (int64, error) {
	type tenantUsedStorageRow struct {
		TenantUsedBytes *int64 `json:"tenant_used_bytes"`
	}

	rows := make([]tenantUsedStorageRow, 0, 1)
	ctxWithPrivacyBypass := privacy.DecisionContext(ctx, privacy.Allow)
	err := tenantTx.StoredFile.Query().
		// Legacy files created before upload status tracking have no upload timestamps.
		Where(storedfile.Or(
			storedfile.UploadSucceededAtNotNil(),
			storedfile.UploadStartedAtIsNil(),
		)).
		Aggregate(enttenant.As(enttenant.Sum(storedfile.FieldSize), "tenant_used_bytes")).
		Scan(ctxWithPrivacyBypass, &rows)
	if err != nil {
		return 0, err
	}

	if len(rows) == 0 || rows[0].TenantUsedBytes == nil {
		return 0, nil
	}
	if *rows[0].TenantUsedBytes < 0 {
		return 0, nil
	}

	return *rows[0].TenantUsedBytes, nil
}

func (qq *StorageQuota) exceedsStorageLimit(usedBytes, incomingUploadedBytes, limitBytes int64) bool {
	if incomingUploadedBytes <= 0 {
		return false
	}
	if limitBytes <= 0 {
		return true
	}
	if usedBytes >= limitBytes {
		return true
	}

	remainingBytes := limitBytes - usedBytes
	return incomingUploadedBytes > remainingBytes
}
