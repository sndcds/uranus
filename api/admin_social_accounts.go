package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/grains/grains_uuid"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

// RegisterSocialAccountRoutes uses the existing JWT-protected admin group.
func (h *ApiHandler) RegisterSocialAccountRoutes(admin *gin.RouterGroup) {
	admin.GET("/social/accounts", h.AdminGetSocialAccounts)
	admin.POST("/social/accounts", h.AdminCreateSocialAccount)
	admin.GET("/social/accounts/:uuid", h.AdminGetSocialAccount)
	admin.PUT("/social/accounts/:uuid", h.AdminUpdateSocialAccount)
	admin.DELETE("/social/accounts/:uuid", h.AdminDeleteSocialAccount)
}

// Write-only credentials retain NullableField's omitted/null/value semantics.
// Defense in depth: accidental JSON or formatted output cannot reveal a value.
type socialAccountSecret struct {
	NullableField[string]
}

func (socialAccountSecret) MarshalJSON() ([]byte, error) { return []byte("null"), nil }
func (socialAccountSecret) String() string               { return "[REDACTED]" }
func (socialAccountSecret) GoString() string             { return "[REDACTED]" }

type socialAccountInput struct {
	OrgUuid           NullableField[string]    `json:"org_uuid"`
	Platform          NullableField[string]    `json:"platform"`
	Name              NullableField[string]    `json:"name"`
	RemoteAccountID   NullableField[string]    `json:"remote_account_id"`
	RemoteAccountName NullableField[string]    `json:"remote_account_name"`
	AccessToken       socialAccountSecret      `json:"access_token"`
	RefreshToken      socialAccountSecret      `json:"refresh_token"`
	TokenExpiresAt    NullableField[time.Time] `json:"token_expires_at"`
	BaseURL           NullableField[string]    `json:"base_url"`
	Enabled           NullableField[bool]      `json:"enabled"`
}

func (p *socialAccountInput) validate(create bool) string {
	for _, field := range []struct {
		name  string
		value *NullableField[string]
	}{
		{"org_uuid", &p.OrgUuid}, {"platform", &p.Platform},
		{"name", &p.Name}, {"remote_account_id", &p.RemoteAccountID},
	} {
		TrimNullableString(field.value)
		if (create || field.value.Set) && (field.value.Value == nil || *field.value.Value == "") {
			return field.name + " is required"
		}
	}
	if p.OrgUuid.Set && !app.ValidUUID(*p.OrgUuid.Value) {
		return "org_uuid must be a valid UUID"
	}
	if p.Platform.Set {
		switch *p.Platform.Value {
		case "facebook", "instagram", "mastodon", "bluesky":
		default:
			return "invalid platform"
		}
	}
	if p.Enabled.Set && p.Enabled.Value == nil {
		return "enabled cannot be null"
	}
	// Explicit empty strings clear credentials; other values are kept verbatim.
	for _, secret := range []*socialAccountSecret{&p.AccessToken, &p.RefreshToken} {
		if secret.Value != nil && *secret.Value == "" {
			secret.Value = nil
		}
	}
	return ""
}

func decodeSocialAccount(gc *gin.Context, req *grains_api.Request, create bool) *socialAccountInput {
	var payload *socialAccountInput
	// Do not use DecodeJSONBody: some decoder errors include submitted values.
	if err := grains_api.BindJSONStrict(gc, &payload); err != nil || payload == nil {
		req.PayloadError()
		return nil
	}
	if message := payload.validate(create); message != "" {
		req.Error(http.StatusBadRequest, message)
		return nil
	}
	return payload
}

// No raw credential is selected or returned, even on writes.
const socialAccountColumns = `uuid, org_uuid, platform, name, remote_account_id,
	remote_account_name, token_expires_at, base_url, enabled,
	COALESCE(access_token <> '', false), COALESCE(refresh_token <> '', false),
	created_at, updated_at`

func scanSocialAccount(row pgx.Row) (model.SocialAccount, error) {
	var account model.SocialAccount
	err := row.Scan(&account.Uuid, &account.OrgUuid, &account.Platform, &account.Name,
		&account.RemoteAccountID, &account.RemoteAccountName, &account.TokenExpiresAt,
		&account.BaseURL, &account.Enabled, &account.HasAccessToken,
		&account.HasRefreshToken, &account.CreatedAt, &account.UpdatedAt)
	return account, err
}

// PostgreSQL errors can contain the failing row, including credentials.
// Neither wrap nor log the original error in this feature.
func socialAccountDBError(err error) *ApiTxError {
	if errors.Is(err, pgx.ErrNoRows) {
		return &ApiTxError{Code: http.StatusNotFound, Message: "social account not found"}
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return &ApiTxError{Code: http.StatusConflict, Message: "social account already exists"}
		case "23001", "23503":
			if pgErr.ConstraintName == "social_post_target_social_account_uuid_fkey" {
				return &ApiTxError{Code: http.StatusConflict, Message: "social account is used by social post targets"}
			}
			return &ApiTxError{Code: http.StatusBadRequest, Message: "invalid organization"}
		case "23502", "23514", "22001", "22007", "22008", "22021", "22P02":
			return &ApiTxError{Code: http.StatusBadRequest, Message: "invalid social account data"}
		}
	}
	return &ApiTxError{Code: http.StatusInternalServerError, Message: "internal server error"}
}

func socialAccountRespondError(req *grains_api.Request, err *ApiTxError) {
	// Also sanitize Begin/Commit errors from WithTransaction and permission errors.
	if err.Code >= http.StatusInternalServerError {
		req.InternalServerError()
	} else {
		req.Error(err.Code, err.Error())
	}
}

func socialAccountUUID(gc *gin.Context, req *grains_api.Request) (string, bool) {
	id := gc.Param("uuid")
	if !app.ValidUUID(id) {
		req.Error(http.StatusBadRequest, "uuid must be a valid UUID")
		return "", false
	}
	return id, true
}

func (h *ApiHandler) AdminCreateSocialAccount(gc *gin.Context) {
	req := grains_api.NewRequest(gc, "admin-create-social-account")
	p := decodeSocialAccount(gc, req, true)
	if p == nil {
		return
	}
	id, err := grains_uuid.Uuidv7String()
	if err != nil {
		req.InternalServerError()
		return
	}
	enabled := true
	if p.Enabled.Set {
		enabled = *p.Enabled.Value
	}
	ctx := gc.Request.Context()
	var result model.SocialAccount
	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		if err := h.CheckAllOrgPermissionsTx(gc, tx, h.userUuid(gc), *p.OrgUuid.Value, app.UserPermEditOrg); err != nil {
			return err
		}
		query := fmt.Sprintf(`INSERT INTO %s.social_account
			(uuid, org_uuid, platform, name, remote_account_id, remote_account_name,
			 access_token, refresh_token, token_expires_at, base_url, enabled)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING %s`, h.DbSchema, socialAccountColumns)
		var err error
		result, err = scanSocialAccount(tx.QueryRow(ctx, query, id, p.OrgUuid.Value,
			p.Platform.Value, p.Name.Value, p.RemoteAccountID.Value, p.RemoteAccountName.Value,
			p.AccessToken.Value, p.RefreshToken.Value, p.TokenExpiresAt.Value, p.BaseURL.Value, enabled))
		if err != nil {
			return socialAccountDBError(err)
		}
		return nil
	})
	if txErr != nil {
		socialAccountRespondError(req, txErr)
		return
	}
	req.Success(http.StatusCreated, result)
}

func (h *ApiHandler) AdminGetSocialAccounts(gc *gin.Context) {
	req := grains_api.NewRequest(gc, "admin-get-social-accounts")
	org, filtered := gc.GetQuery("org_uuid")

	if filtered && !app.ValidUUID(org) {
		req.Error(http.StatusBadRequest, "org_uuid must be a valid UUID")
		return
	}
	// EXISTS prevents duplicate results when multiple organization links exist.
	query := fmt.Sprintf(`SELECT %s FROM %s.social_account a
		WHERE EXISTS (SELECT 1 FROM %s.user_organization_link l
			JOIN %s.organization_member_link m
				ON m.org_uuid = l.org_uuid AND m.user_uuid = l.user_uuid AND m.has_joined = true
			WHERE l.org_uuid = a.org_uuid AND l.user_uuid = $1
			AND (l.permissions & $2) = $2)`, socialAccountColumns, h.DbSchema, h.DbSchema, h.DbSchema)

	args := []any{h.userUuid(gc), int64(app.UserPermEditOrg)}
	if filtered {
		query += " AND a.org_uuid = $3"
		args = append(args, org)
	}
	query += " ORDER BY a.created_at, a.uuid"
	rows, err := h.DbPool.Query(gc.Request.Context(), query, args...)
	if err != nil {
		req.InternalServerError()
		return
	}
	defer rows.Close()

	accounts := make([]model.SocialAccount, 0)
	for rows.Next() {
		account, err := scanSocialAccount(rows)
		if err != nil {
			req.InternalServerError()
			return
		}
		accounts = append(accounts, account)
	}
	if rows.Err() != nil {
		req.InternalServerError()
		return
	}
	req.Success(http.StatusOK, gin.H{"accounts": accounts})
}

// Lock before checking ownership so an organization change cannot race an update
// or delete authorized against the previous organization.
func (h *ApiHandler) socialAccountPermission(gc *gin.Context, tx pgx.Tx, id string) *ApiTxError {
	var org string
	err := tx.QueryRow(gc.Request.Context(),
		fmt.Sprintf("SELECT org_uuid FROM %s.social_account WHERE uuid = $1 FOR UPDATE", h.DbSchema), id).Scan(&org)
	if err != nil {
		return socialAccountDBError(err)
	}
	return h.CheckAllOrgPermissionsTx(gc, tx, h.userUuid(gc), org, app.UserPermEditOrg)
}

func (h *ApiHandler) AdminGetSocialAccount(gc *gin.Context) {
	req := grains_api.NewRequest(gc, "admin-get-social-account")
	id, ok := socialAccountUUID(gc, req)
	if !ok {
		return
	}
	ctx := gc.Request.Context()
	var result model.SocialAccount
	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		if err := h.socialAccountPermission(gc, tx, id); err != nil {
			return err
		}
		var err error
		result, err = scanSocialAccount(tx.QueryRow(ctx,
			fmt.Sprintf("SELECT %s FROM %s.social_account WHERE uuid = $1", socialAccountColumns, h.DbSchema), id))
		if err != nil {
			return socialAccountDBError(err)
		}
		return nil
	})
	if txErr != nil {
		socialAccountRespondError(req, txErr)
		return
	}
	req.Success(http.StatusOK, result)
}

func (h *ApiHandler) AdminUpdateSocialAccount(gc *gin.Context) {
	req := grains_api.NewRequest(gc, "admin-update-social-account")
	id, ok := socialAccountUUID(gc, req)
	if !ok {
		return
	}
	p := decodeSocialAccount(gc, req, false)
	if p == nil {
		return
	}
	clauses := []string{"updated_at = CURRENT_TIMESTAMP"}
	args := []any{}
	pos := 1
	pos = addUpdateClauseNullable("org_uuid", p.OrgUuid, &clauses, &args, pos)
	pos = addUpdateClauseNullable("platform", p.Platform, &clauses, &args, pos)
	pos = addUpdateClauseNullable("name", p.Name, &clauses, &args, pos)
	pos = addUpdateClauseNullable("remote_account_id", p.RemoteAccountID, &clauses, &args, pos)
	pos = addUpdateClauseNullable("remote_account_name", p.RemoteAccountName, &clauses, &args, pos)
	pos = addUpdateClauseNullable("access_token", p.AccessToken.NullableField, &clauses, &args, pos)
	pos = addUpdateClauseNullable("refresh_token", p.RefreshToken.NullableField, &clauses, &args, pos)
	pos = addUpdateClauseNullable("token_expires_at", p.TokenExpiresAt, &clauses, &args, pos)
	pos = addUpdateClauseNullable("base_url", p.BaseURL, &clauses, &args, pos)
	pos = addUpdateClauseNullable("enabled", p.Enabled, &clauses, &args, pos)
	args = append(args, id)
	query := fmt.Sprintf("UPDATE %s.social_account SET %s WHERE uuid = $%d RETURNING %s",
		h.DbSchema, strings.Join(clauses, ", "), pos, socialAccountColumns)
	ctx := gc.Request.Context()
	var result model.SocialAccount
	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		if err := h.socialAccountPermission(gc, tx, id); err != nil {
			return err
		}
		if p.OrgUuid.Set {
			if err := h.CheckAllOrgPermissionsTx(gc, tx, h.userUuid(gc), *p.OrgUuid.Value, app.UserPermEditOrg); err != nil {
				return err
			}
		}
		var err error
		result, err = scanSocialAccount(tx.QueryRow(ctx, query, args...))
		if err != nil {
			return socialAccountDBError(err)
		}
		return nil
	})
	if txErr != nil {
		socialAccountRespondError(req, txErr)
		return
	}
	req.Success(http.StatusOK, result)
}

func (h *ApiHandler) AdminDeleteSocialAccount(gc *gin.Context) {
	req := grains_api.NewRequest(gc, "admin-delete-social-account")
	id, ok := socialAccountUUID(gc, req)
	if !ok {
		return
	}
	ctx := gc.Request.Context()
	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		if err := h.socialAccountPermission(gc, tx, id); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, fmt.Sprintf("DELETE FROM %s.social_account WHERE uuid = $1", h.DbSchema), id)
		if err != nil {
			return socialAccountDBError(err)
		}
		return nil
	})
	if txErr != nil {
		socialAccountRespondError(req, txErr)
		return
	}
	req.SuccessNoData(http.StatusOK, "social account deleted")
}
