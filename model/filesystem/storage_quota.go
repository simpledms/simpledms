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
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/fileutil"
)

// TODO default should be set in env var;
//
//	overrideable per tenant, maybe based on subscription plan
const tenantQuotaPerUserBytes int64 = 5 * 1024 * 1024 * 1024

type StorageQuota struct {
	isSaaSModeEnabled bool
}

func NewStorageQuota(isSaaSModeEnabled bool) *StorageQuota {
	return &StorageQuota{
		isSaaSModeEnabled: isSaaSModeEnabled,
	}
}

func (qq *StorageQuota) EnsureTenantStorageLimit(ctx ctxx.Context, incomingUploadedBytes int64) error {
	if !qq.isSaaSModeEnabled {
		return nil
	}
	if incomingUploadedBytes <= 0 {
		return nil
	}

	limitBytes, err := qq.tenantStorageLimitBytes(ctx)
	if err != nil {
		return e.NewHTTPErrorf(http.StatusInternalServerError, "Could not verify storage limit.")
	}

	usedBytes, err := qq.currentUsedTenantStorageBytes(ctx)
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

func (qq *StorageQuota) tenantStorageLimitBytes(ctx ctxx.Context) (int64, error) {
	activeUserCount, err := ctx.MainCtx().MainTx.TenantAccountAssignment.Query().
		Where(
			tenantaccountassignment.TenantID(ctx.TenantCtx().Tenant.ID),
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

	if activeUserCount <= 0 {
		return 0, nil
	}
	if int64(activeUserCount) > math.MaxInt64/tenantQuotaPerUserBytes {
		return math.MaxInt64, nil
	}

	return int64(activeUserCount) * tenantQuotaPerUserBytes, nil
}

func (qq *StorageQuota) currentUsedTenantStorageBytes(ctx ctxx.Context) (int64, error) {
	type tenantUsedStorageRow struct {
		TenantUsedBytes *int64 `json:"tenant_used_bytes"`
	}

	rows := make([]tenantUsedStorageRow, 0, 1)
	ctxWithPrivacyBypass := privacy.DecisionContext(ctx, privacy.Allow)
	err := ctx.TenantCtx().TTx.StoredFile.Query().
		Where(storedfile.UploadSucceededAtNotNil()).
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
