package webdavcredential

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	entmainwebdavcredential "github.com/simpledms/simpledms/db/entmain/webdavcredential"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/util/accountutil"
	"github.com/simpledms/simpledms/util/e"
)

const (
	secretBytes   = 32
	usernameBytes = 15
	fakeSalt      = "webdav-fake-salt-for-missing-users"

	// MinimumSecretLength is the shortest supported generated WebDAV secret.
	MinimumSecretLength = 12
	// DefaultSecretLength preserves the full generated WebDAV secret length.
	DefaultSecretLength = 43
)

var fakeHash = accountutil.PasswordHash("not-the-secret", fakeSalt)

type CredentialService struct{}

func NewCredentialService() *CredentialService {
	return &CredentialService{}
}

func (qq *CredentialService) CreateOwnerCredential(
	ctx *ctxx.SpaceContext,
	label string,
	endpointURL string,
	secretLength int,
	compatibilityMode bool,
) (*CreateResult, error) {
	label = strings.TrimSpace(label)
	if label == "" {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Credential label is required.")
	}

	secret, err := randomSecret(secretLength, compatibilityMode)
	if err != nil {
		return nil, err
	}
	username, err := uniqueUsername(ctx, ctx.MainTx)
	if err != nil {
		return nil, err
	}
	salt, ok := accountutil.RandomSalt()
	if !ok {
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not create credential.")
	}

	_, err = ctx.MainTx.WebDAVCredential.Create().
		SetAccountID(ctx.Account.ID).
		SetTenantID(ctx.Tenant.ID).
		SetSpacePublicID(entx.NewCIText(ctx.Space.PublicID.String())).
		SetLabel(label).
		SetUsername(username).
		SetSecretSalt(salt).
		SetSecretHash(accountutil.PasswordHash(secret, salt)).
		Save(ctx)
	if err != nil {
		return nil, credentialConstraintError(err)
	}

	return &CreateResult{
		URL:      endpointURL,
		Username: username,
		Secret:   secret,
	}, nil
}

func (qq *CredentialService) RevokeOwnedCredential(
	ctx *ctxx.MainContext,
	credentialPublicID string,
) error {
	return qq.revoke(ctx, credentialPublicID, ctx.Account.ID, nil, &ctx.Account.ID)
}

func (qq *CredentialService) EditOwnedCredentialLabel(
	ctx *ctxx.MainContext,
	credentialPublicID string,
	label string,
) error {
	label = strings.TrimSpace(label)
	if label == "" {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Credential label is required.")
	}

	credentialx, err := ctx.MainTx.WebDAVCredential.Query().
		Where(
			entmainwebdavcredential.PublicIDEQ(entx.NewCIText(credentialPublicID)),
			entmainwebdavcredential.AccountID(ctx.Account.ID),
		).
		Only(ctx)
	if err != nil {
		if entmain.IsNotFound(err) {
			return e.NewHTTPErrorf(http.StatusNotFound, "Credential not found.")
		}
		log.Println(err)
		return err
	}

	if err := credentialx.Update().SetLabel(label).Exec(ctx); err != nil {
		return credentialConstraintError(err)
	}
	return nil
}

func (qq *CredentialService) RevokeTenantCredential(
	ctx *ctxx.MainContext,
	credentialPublicID string,
	tenantID int64,
) error {
	return qq.revoke(ctx, credentialPublicID, ctx.Account.ID, &tenantID, nil)
}

func (qq *CredentialService) VerifySecret(record *AuthRecord, secret string) bool {
	salt := fakeSalt
	hash := fakeHash
	if record != nil {
		salt = record.SecretSalt
		hash = record.SecretHash
	}

	candidate := accountutil.PasswordHash(secret, salt)
	return subtle.ConstantTimeCompare([]byte(candidate), []byte(hash)) == 1
}

func (qq *CredentialService) AuthRecordByUsername(
	ctx context.Context,
	mainDB *entmain.Client,
	username string,
) (*AuthRecord, bool, error) {
	credentialx, err := mainDB.WebDAVCredential.Query().
		Where(entmainwebdavcredential.Username(username)).
		Only(ctx)
	if err != nil {
		if entmain.IsNotFound(err) {
			return nil, false, nil
		}
		log.Println(err)
		return nil, false, err
	}

	tenantx, err := credentialx.QueryTenant().Only(ctx)
	if err != nil {
		log.Println(err)
		return nil, false, err
	}

	return &AuthRecord{
		ID:             credentialx.ID,
		PublicID:       credentialx.PublicID.String(),
		AccountID:      credentialx.AccountID,
		TenantID:       credentialx.TenantID,
		TenantPublicID: tenantx.PublicID.String(),
		SpacePublicID:  credentialx.SpacePublicID.String(),
		SecretSalt:     credentialx.SecretSalt,
		SecretHash:     credentialx.SecretHash,
		RevokedAt:      credentialx.RevokedAt,
		LastUsedAt:     credentialx.LastUsedAt,
	}, true, nil
}

func (qq *CredentialService) TouchLastUsed(
	ctx context.Context,
	mainDB *entmain.Client,
	credentialID int64,
	lastUsedAt *time.Time,
) {
	now := time.Now()
	if lastUsedAt != nil && lastUsedAt.After(now.Add(-15*time.Minute)) {
		return
	}
	if err := mainDB.WebDAVCredential.UpdateOneID(credentialID).
		SetLastUsedAt(now).
		Exec(ctx); err != nil && !entmain.IsNotFound(err) {
		log.Println(err)
	}
}

func (qq *CredentialService) revoke(
	ctx *ctxx.MainContext,
	credentialPublicID string,
	revokedByAccountID int64,
	tenantID *int64,
	accountID *int64,
) error {
	query := ctx.MainTx.WebDAVCredential.Query().
		Where(entmainwebdavcredential.PublicIDEQ(entx.NewCIText(credentialPublicID)))
	if tenantID != nil {
		query.Where(entmainwebdavcredential.TenantID(*tenantID))
	}
	if accountID != nil {
		query.Where(entmainwebdavcredential.AccountID(*accountID))
	}

	credentialx, err := query.Only(ctx)
	if err != nil {
		if entmain.IsNotFound(err) {
			return e.NewHTTPErrorf(http.StatusNotFound, "Credential not found.")
		}
		log.Println(err)
		return err
	}
	if credentialx.RevokedAt != nil {
		return nil
	}

	return credentialx.Update().
		SetRevokedAt(time.Now()).
		SetRevokedByAccountID(revokedByAccountID).
		Exec(ctx)
}

func credentialConstraintError(err error) error {
	log.Println(err)
	if entmain.IsConstraintError(err) {
		return e.NewHTTPErrorf(http.StatusBadRequest, "A similar entity already exists.")
	}
	return err
}

func uniqueUsername(ctx context.Context, mainTx *entmain.Tx) (string, error) {
	for range 10 {
		username, err := randomUsername()
		if err != nil {
			return "", err
		}
		exists, err := mainTx.WebDAVCredential.Query().
			Where(entmainwebdavcredential.Username(username)).
			Exist(ctx)
		if err != nil {
			return "", err
		}
		if !exists {
			return username, nil
		}
	}

	return "", errors.New("could not generate unique WebDAV username")
}

func randomUsername() (string, error) {
	bytes := make([]byte, usernameBytes)
	if _, err := rand.Read(bytes); err != nil {
		log.Println(err)
		return "", err
	}
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes)
	return "dav_" + strings.ToLower(encoded), nil
}

func randomSecret(length int, compatibilityMode bool) (string, error) {
	if length == 0 {
		length = DefaultSecretLength
	}
	if length < MinimumSecretLength || length > DefaultSecretLength {
		return "", e.NewHTTPErrorf(
			http.StatusBadRequest,
			"Secret length must be between %d and %d characters.",
			MinimumSecretLength,
			DefaultSecretLength,
		)
	}
	if !compatibilityMode {
		return randomPrintableASCIISecret(length)
	}
	bytes := make([]byte, secretBytes)
	if _, err := rand.Read(bytes); err != nil {
		log.Println(err)
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes)[:length], nil
}

func randomPrintableASCIISecret(length int) (string, error) {
	characterLimit := big.NewInt(94)
	for {
		secret := make([]byte, length)
		hasSpecialCharacter := false
		for index := range secret {
			characterIndex, err := rand.Int(rand.Reader, characterLimit)
			if err != nil {
				log.Println(err)
				return "", err
			}
			secret[index] = byte(33 + characterIndex.Int64())
			hasSpecialCharacter = hasSpecialCharacter || !isASCIIAlphaNumeric(secret[index])
		}
		if hasSpecialCharacter {
			return string(secret), nil
		}
	}
}

func isASCIIAlphaNumeric(char byte) bool {
	return char >= '0' && char <= '9' ||
		char >= 'A' && char <= 'Z' ||
		char >= 'a' && char <= 'z'
}
