package account

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	gonanoid "github.com/matoous/go-nanoid"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/passkeycredential"
	"github.com/simpledms/simpledms/db/entmain/session"
	"github.com/simpledms/simpledms/db/entmain/tenant"
	"github.com/simpledms/simpledms/db/entmain/tenantaccountassignment"
	"github.com/simpledms/simpledms/util/accountutil"
	"github.com/simpledms/simpledms/util/e"
)

type Account struct {
	Data *entmain.Account
}

const passkeyRecoveryCodeAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
const passkeyRecoveryCodeGroupLength = 5
const passkeyRecoveryCodeGroupCount = 4

func NewAccount(data *entmain.Account) *Account {
	return &Account{
		Data: data,
	}
}

/*
// Tenant instead of ID is more type safe
func (qq *Account) BelongsToTenant(ctx context.Context, tenantm *tenant2.Tenant) bool {
	return qq.Data.QueryTenants().Where(tenant.ID(tenantm.Data.ID)).ExistX(ctx)
}
*/

// TODO rename to Login?
func (qq *Account) Auth(ctx ctxx.Context, password string) (bool, error) {
	passkeyPolicy, err := qq.PasskeyPolicy(ctx)
	if err != nil {
		return false, err
	}

	return qq.AuthWithPasskeyPolicy(ctx, password, passkeyPolicy)
}

func (qq *Account) AuthWithPasskeyPolicy(
	ctx ctxx.Context,
	password string,
	passkeyPolicy *PasskeyPolicy,
) (bool, error) {
	err := qq.ensurePasswordSignInAllowedWithPasskeyPolicy(passkeyPolicy)
	if err != nil {
		return false, err
	}

	return qq.authenticatePasswordSignIn(ctx, password)
}

func (qq *Account) ensurePasswordSignInAllowedWithPasskeyPolicy(passkeyPolicy *PasskeyPolicy) error {
	if !passkeyPolicy.IsPasswordSignInAllowed() {
		return e.NewHTTPErrorf(http.StatusUnauthorized, "Passkey sign-in is required for this account.")
	}

	return nil
}

func (qq *Account) PasskeyPolicy(ctx ctxx.Context) (*PasskeyPolicy, error) {
	if qq.Data.PasskeyLoginEnabled {
		return NewPasskeyPolicy(true, false, false), nil
	}

	isTenantPasskeyRequired, err := qq.isTenantPasskeyAuthEnforced(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if !isTenantPasskeyRequired {
		return NewPasskeyPolicy(false, false, false), nil
	}

	hasPasskeyCredentials, err := ctx.VisitorCtx().MainTx.PasskeyCredential.Query().
		Where(passkeycredential.AccountID(qq.Data.ID)).
		Exist(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return NewPasskeyPolicy(false, true, hasPasskeyCredentials), nil
}

func (qq *Account) IsTenantPasskeyEnrollmentRequired(ctx ctxx.Context) (bool, error) {
	passkeyPolicy, err := qq.PasskeyPolicy(ctx)
	if err != nil {
		log.Println(err)
		return false, err
	}

	return passkeyPolicy.IsTenantPasskeyEnrollmentRequired(), nil
}

func (qq *Account) authenticatePasswordSignIn(
	ctx ctxx.Context,
	password string,
) (bool, error) {
	// TODO make sure account isn't deleted

	err := qq.EnsureRecentLoginAttemptAllowed(
		ctx,
		10*time.Second,
		http.StatusUnauthorized,
		"Too many login attempts. Please try again in 10 seconds.",
	)
	if err != nil {
		return false, err
	}
	qq.RecordLoginAttempt(ctx)

	if qq.IsPasswordValid(ctx, password) {
		return true, nil
	}

	isValid, err := qq.isTemporaryPasswordValid(password)
	if err != nil {
		log.Println(err)
		return false, err
	}
	return isValid, nil
}

func (qq *Account) EnsureRecentLoginAttemptAllowed(
	ctx ctxx.Context,
	window time.Duration,
	statusCode int,
	message string,
) error {
	if qq.Data.LastLoginAttemptAt.After(time.Now().Add(-window)) {
		return e.NewHTTPErrorf(statusCode, message)
	}

	return nil
}

func (qq *Account) RecordLoginAttempt(ctx ctxx.Context) {
	qq.Data = qq.Data.Update().SetLastLoginAttemptAt(time.Now()).SaveX(ctx)
}

func (qq *Account) IsPasskeyLoginRequired(ctx ctxx.Context) (bool, error) {
	passkeyPolicy, err := qq.PasskeyPolicy(ctx)
	if err != nil {
		log.Println(err)
		return false, err
	}

	return passkeyPolicy.IsPasskeyLoginRequired(), nil
}

func (qq *Account) IsTenantPasskeyAuthEnforced(ctx ctxx.Context) (bool, error) {
	return qq.isTenantPasskeyAuthEnforced(ctx)
}

func (qq *Account) isTenantPasskeyAuthEnforced(ctx ctxx.Context) (bool, error) {
	return qq.Data.QueryTenantAssignment().
		Where(
			tenantaccountassignment.Or(
				tenantaccountassignment.ExpiresAtIsNil(),
				tenantaccountassignment.ExpiresAtGT(time.Now()),
			),
		).
		QueryTenant().
		Where(tenant.PasskeyAuthEnforced(true)).
		Exist(ctx)
}

func (qq *Account) IsPasskeyLoginEnabled() bool {
	return qq.Data.PasskeyLoginEnabled
}

func (qq *Account) GenerateAndSetPasskeyRecoveryCodes(ctx ctxx.Context, count int) ([]string, error) {
	if count <= 0 {
		count = 10
	}

	salt, ok := accountutil.RandomSalt()
	if !ok {
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not generate backup codes.")
	}

	recoveryCodes := make([]string, 0, count)
	recoveryCodeHashes := make([]string, 0, count)

	for i := 0; i < count; i++ {
		recoveryCode, err := gonanoid.Generate(
			passkeyRecoveryCodeAlphabet,
			passkeyRecoveryCodeGroupLength*passkeyRecoveryCodeGroupCount,
		)
		if err != nil {
			log.Println(err)
			return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not generate backup codes.")
		}

		formattedCode := qq.formatPasskeyRecoveryCode(recoveryCode)
		recoveryCodes = append(recoveryCodes, formattedCode)

		normalizedCode := qq.normalizePasskeyRecoveryCode(formattedCode)
		recoveryCodeHashes = append(recoveryCodeHashes, accountutil.PasswordHash(normalizedCode, salt))
	}

	qq.Data = qq.Data.Update().
		SetPasskeyRecoveryCodeSalt(salt).
		SetPasskeyRecoveryCodeHashes(recoveryCodeHashes).
		SaveX(ctx)

	return recoveryCodes, nil
}

func (qq *Account) ConsumePasskeyRecoveryCode(ctx ctxx.Context, recoveryCode string) (bool, error) {
	salt := qq.Data.PasskeyRecoveryCodeSalt
	if salt == "" {
		return false, nil
	}

	normalizedCode := qq.normalizePasskeyRecoveryCode(recoveryCode)
	if normalizedCode == "" {
		return false, nil
	}

	codeHash := accountutil.PasswordHash(normalizedCode, salt)
	recoveryCodeHashes := qq.Data.PasskeyRecoveryCodeHashes

	for qi, existingHash := range recoveryCodeHashes {
		if codeHash == existingHash {
			recoveryCodeHashes = append(recoveryCodeHashes[:qi], recoveryCodeHashes[qi+1:]...)
			qq.Data = qq.Data.Update().SetPasskeyRecoveryCodeHashes(recoveryCodeHashes).SaveX(ctx)
			return true, nil
		}
	}

	return false, nil
}

func (qq *Account) SetPasskeyLoginEnabled(ctx ctxx.Context, isEnabled bool) {
	qq.Data = qq.Data.Update().SetPasskeyLoginEnabled(isEnabled).SaveX(ctx)
}

func (qq *Account) EnablePasskeyLogin(ctx ctxx.Context) {
	if qq.Data.PasskeyLoginEnabled {
		return
	}

	qq.SetPasskeyLoginEnabled(ctx, true)
}

func (qq *Account) DisablePasskeyLoginAndClearRecoveryCodes(ctx ctxx.Context) {
	qq.Data = qq.Data.Update().
		SetPasskeyLoginEnabled(false).
		SetPasskeyRecoveryCodeSalt("").
		SetPasskeyRecoveryCodeHashes([]string{}).
		SaveX(ctx)
}

func (qq *Account) DisablePasskeyLoginAndClearRecoveryCodesIfNoCredentials(
	ctx ctxx.Context,
) (bool, error) {
	hasPasskeyCredentials, err := ctx.VisitorCtx().MainTx.PasskeyCredential.Query().
		Where(passkeycredential.AccountID(qq.Data.ID)).
		Exist(ctx)
	if err != nil {
		log.Println(err)
		return false, err
	}
	if hasPasskeyCredentials {
		return false, nil
	}

	qq.DisablePasskeyLoginAndClearRecoveryCodes(ctx)

	return true, nil
}

func (qq *Account) ClearPasskeyRecoveryCodes(ctx ctxx.Context) {
	qq.Data = qq.Data.Update().
		SetPasskeyRecoveryCodeSalt("").
		SetPasskeyRecoveryCodeHashes([]string{}).
		SaveX(ctx)
}

func (qq *Account) normalizePasskeyRecoveryCode(recoveryCode string) string {
	recoveryCode = strings.TrimSpace(recoveryCode)
	recoveryCode = strings.ReplaceAll(recoveryCode, "-", "")
	recoveryCode = strings.ToUpper(recoveryCode)
	return recoveryCode
}

func (qq *Account) formatPasskeyRecoveryCode(recoveryCode string) string {
	groups := make([]string, 0, passkeyRecoveryCodeGroupCount)
	for qi := 0; qi < len(recoveryCode); qi += passkeyRecoveryCodeGroupLength {
		groups = append(groups, recoveryCode[qi:qi+passkeyRecoveryCodeGroupLength])
	}

	return strings.Join(groups, "-")
}

func (qq *Account) IsPasswordValid(ctx ctxx.Context, password string) bool {
	passwordHash := accountutil.PasswordHash(password, qq.Data.PasswordSalt)
	return passwordHash == qq.Data.PasswordHash
}

func (qq *Account) ClearTemporaryPassword(ctx ctxx.Context) {
	qq.Data.Update().
		SetTemporaryPasswordSalt("").
		SetTemporaryPasswordHash("").
		SetTemporaryPasswordExpiresAt(time.Time{}).
		SaveX(ctx)
}

// Unsafe because must be used with care...
func (qq *Account) UnsafeDelete(ctx ctxx.Context) {
	qq.ClearTemporaryPassword(ctx)

	qq.Data = qq.Data.Update().
		SetPasswordSalt("").
		SetPasswordHash("").
		SetPasskeyLoginEnabled(false).
		SetPasskeyRecoveryCodeSalt("").
		SetPasskeyRecoveryCodeHashes([]string{}).
		SetDeletedAt(time.Now()).
		SetDeletedBy(ctx.MainCtx().Account.ID). // TODO SetDeleter
		SaveX(ctx)

	ctx.MainCtx().MainTx.Session.Delete().Where(session.AccountID(qq.Data.ID)).ExecX(ctx)
	ctx.MainCtx().MainTx.PasskeyCredential.Delete().Where(passkeycredential.AccountID(qq.Data.ID)).ExecX(ctx)
}

func (qq *Account) isTemporaryPasswordValid(password string) (bool, error) {
	if qq.Data.TemporaryPasswordHash == "" || qq.Data.TemporaryPasswordSalt == "" {
		return false, nil
	}
	if qq.Data.TemporaryPasswordExpiresAt.Before(time.Now()) {
		// TODO detailed error message? probably leaks to much info
		return false, e.NewHTTPErrorf(http.StatusUnauthorized, "Temporary password expired.")
	}

	passwordHash := accountutil.PasswordHash(password, qq.Data.TemporaryPasswordSalt)
	return passwordHash == qq.Data.TemporaryPasswordHash, nil
}

func (qq *Account) GenerateTemporaryPassword(ctx ctxx.Context) (string, time.Time, error) {
	password, err := gonanoid.Generate("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_", 16)
	if err != nil {
		log.Println(err)
		return "", time.Time{}, e.NewHTTPErrorf(http.StatusInternalServerError, "could not generate temporary password")
	}

	expiresAt, err := qq.SetTemporaryPassword(ctx, password)
	if err != nil {
		log.Println(err)
		return "", time.Time{}, err
	}

	return password, expiresAt, nil
}

// should only be used on app initialization
func (qq *Account) SetTemporaryPassword(ctx context.Context, password string) (time.Time, error) {
	salt, ok := accountutil.RandomSalt()
	if !ok {
		return time.Time{}, e.NewHTTPErrorf(http.StatusInternalServerError, "could not generate salt")
	}

	passwordHash := accountutil.PasswordHash(password, salt)

	expiresAt := time.Now().Add(time.Hour * 24 * 7)

	qq.Data.Update().
		SetTemporaryPasswordSalt(salt).
		SetTemporaryPasswordHash(passwordHash).
		SetTemporaryPasswordExpiresAt(expiresAt).
		SaveX(ctx)

	return expiresAt, nil

}

func (qq *Account) ChangePassword(ctx ctxx.Context, currentPassword, newPassword, confirmPassword string) error {
	isTemporaryPasswordValid, err := qq.isTemporaryPasswordValid(currentPassword)
	if err != nil {
		log.Println(err)
		return err
	}
	if !qq.IsPasswordValid(ctx, currentPassword) && !isTemporaryPasswordValid {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Current password is invalid.")
	}

	if currentPassword == newPassword {
		return e.NewHTTPErrorf(http.StatusBadRequest, "New password must be different from current password.")
	}

	return qq.SetPassword(ctx, newPassword, confirmPassword)
}

func (qq *Account) SetPassword(ctx ctxx.Context, password, confirmPassword string) error {
	// TODO add additional password rules?
	if len(password) < 12 {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Password must be at least twelve characters long.")
	}

	if password != confirmPassword {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Passwords do not match.")
	}

	salt, ok := accountutil.RandomSalt()
	if !ok {
		return e.NewHTTPErrorf(http.StatusInternalServerError, "could not generate salt")
	}

	passwordHash := accountutil.PasswordHash(password, salt)

	qq.Data.Update().
		SetPasswordSalt(salt).
		SetPasswordHash(passwordHash).
		// Clear temporary password fields
		SetTemporaryPasswordSalt("").
		SetTemporaryPasswordHash("").
		SetTemporaryPasswordExpiresAt(time.Time{}).
		SaveX(ctx)

	return nil
}

func (qq *Account) HasPassword() bool {
	return qq.Data.PasswordHash != "" && qq.Data.PasswordSalt != ""
}

func (qq *Account) HasTemporaryPassword() bool {
	return qq.Data.TemporaryPasswordHash != "" && qq.Data.TemporaryPasswordSalt != "" &&
		qq.Data.TemporaryPasswordExpiresAt.After(time.Now())
}

// same in User.Name()
func (qq *Account) Name() string {
	var elems []string
	if qq.Data.FirstName != "" {
		elems = append(elems, qq.Data.FirstName)
	}
	if qq.Data.LastName != "" {
		elems = append(elems, qq.Data.LastName)
	}
	if len(elems) > 0 {
		return strings.Join(elems, " ")
	}
	// TODO does this expose to many details?
	return qq.Data.Email.String()
}
