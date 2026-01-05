package account

import (
	"log"
	"net/http"
	"strings"
	"time"

	gonanoid "github.com/matoous/go-nanoid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/session"
	"github.com/simpledms/simpledms/util/accountutil"
	"github.com/simpledms/simpledms/util/e"
)

type Account struct {
	Data *entmain.Account
}

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
func (qq *Account) Auth(ctx ctxx.Context, password, twoFactorAuthCode string) (bool, error) {
	// TODO make sure account isn't deleted

	// this solution has negative side effect that user cannot login while brute force
	// attack is going on // TODO find another solution
	if qq.Data.LastLoginAttemptAt.After(time.Now().Add(-10 * time.Second)) {
		return false, e.NewHTTPErrorf(http.StatusUnauthorized, "Too many login attempts. Please try again in 10 seconds.")
	}
	qq.Data.Update().SetLastLoginAttemptAt(time.Now()).SaveX(ctx)

	isValid, err := qq.isPasswordAnd2FACodeValid(ctx, password, twoFactorAuthCode)
	if err != nil {
		log.Println(err)
		return false, err
	}
	if isValid {
		return true, nil
	}

	isValid, err = qq.isTemporaryPasswordValid(password)
	if err != nil {
		log.Println(err)
		return false, err
	}
	return isValid, nil
}

func (qq *Account) isPasswordAnd2FACodeValid(ctx ctxx.Context, password, twoFactorAuthCode string) (bool, error) {
	if qq.Data.PasswordHash == "" || qq.Data.PasswordSalt == "" {
		return false, nil
	}

	isValid, err := qq.isTwoFactorAuthCodeValid(ctx, twoFactorAuthCode)
	if err != nil {
		log.Println(err)
		return false, err
	}
	if !isValid {
		return false, nil
	}

	return qq.IsPasswordValid(ctx, password), nil
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
		SetTwoFactoryAuthKeyEncrypted("").
		SetTwoFactorAuthRecoveryCodeSalt("").
		SetTwoFactorAuthRecoveryCodeHashes([]string{}).
		SetDeletedAt(time.Now()).
		SetDeletedBy(ctx.MainCtx().Account.ID). // TODO SetDeleter
		SaveX(ctx)

	ctx.MainCtx().MainTx.Session.Delete().Where(session.AccountID(qq.Data.ID)).ExecX(ctx)
}

func (qq *Account) isTwoFactorAuthCodeValid(ctx ctxx.Context, twoFactorAuthCode string) (bool, error) {
	twoFactorAuthKey := qq.Data.TwoFactoryAuthKeyEncrypted // TODO decrypt

	if twoFactorAuthKey == "" {
		if twoFactorAuthCode != "" {
			return false, e.NewHTTPErrorf(http.StatusBadRequest, "2FA code was provided, but 2FA is not enabled for this account.")
		}
		return true, nil
	}

	otpKey, err := otp.NewKeyFromURL(twoFactorAuthKey)
	if err != nil {
		log.Println(err)
		return false, err
	}

	isValid := totp.Validate(twoFactorAuthCode, otpKey.Secret())
	if isValid {
		return true, nil
	}

	// check recovery codes
	codeHash := accountutil.PasswordHash(twoFactorAuthCode, qq.Data.TwoFactorAuthRecoveryCodeSalt)
	recoveryCodeHashes := qq.Data.TwoFactorAuthRecoveryCodeHashes

	for qi, recoveryHash := range recoveryCodeHashes {
		if codeHash == recoveryHash {
			recoveryCodeHashes = append(recoveryCodeHashes[:qi], recoveryCodeHashes[qi+1:]...)
			qq.Data.Update().SetTwoFactorAuthRecoveryCodeHashes(recoveryCodeHashes).SaveX(ctx)
			return true, nil
		}
	}

	return false, nil
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

	salt, ok := accountutil.RandomSalt()
	if !ok {
		return "", time.Time{}, e.NewHTTPErrorf(http.StatusInternalServerError, "could not generate salt")
	}

	passwordHash := accountutil.PasswordHash(password, salt)

	expiresAt := time.Now().Add(time.Hour * 24 * 7)

	qq.Data.Update().
		SetTemporaryPasswordSalt(salt).
		SetTemporaryPasswordHash(passwordHash).
		SetTemporaryPasswordExpiresAt(expiresAt).
		SaveX(ctx)

	return password, expiresAt, nil
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

// generated by Junie
func (qq *Account) SetPassword(ctx ctxx.Context, password, confirmPassword string) error {
	// TODO add additional password rules?
	if len(password) < 8 { // TODO is 8 chars enough? with 2fa probably
		return e.NewHTTPErrorf(http.StatusBadRequest, "Password must be at least eight characters long.")
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

// generated by Junie
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
