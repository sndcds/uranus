package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/grains/grains_uuid"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) RegisterSocialPostRoutes(admin *gin.RouterGroup) {
	admin.GET("/social/posts", h.AdminGetSocialPosts)
	admin.POST("/social/posts", h.AdminCreateSocialPost)
	admin.GET("/social/posts/:uuid", h.AdminGetSocialPost)
	admin.PUT("/social/posts/:uuid", h.AdminUpdateSocialPost)
	admin.DELETE("/social/posts/:uuid", h.AdminDeleteSocialPost)
}

type socialPostTargetInput struct {
	SocialAccountUuid string `json:"social_account_uuid"`
}

// NullableField decodes its value with json.Unmarshal; keep nested target
// fields strict as well so publication metadata cannot be submitted silently.
func (target *socialPostTargetInput) UnmarshalJSON(data []byte) error {
	type targetInput socialPostTargetInput
	var value targetInput
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	*target = socialPostTargetInput(value)
	return nil
}

type socialPostInput struct {
	OrgUuid    NullableField[string]                  `json:"org_uuid"`
	SourceType NullableField[string]                  `json:"source_type"`
	SourceUuid NullableField[string]                  `json:"source_uuid"`
	Targets    NullableField[[]socialPostTargetInput] `json:"targets"`
}

func (p *socialPostInput) validate(create bool) string {
	for _, field := range []struct {
		name  string
		value *NullableField[string]
	}{
		{"org_uuid", &p.OrgUuid}, {"source_type", &p.SourceType}, {"source_uuid", &p.SourceUuid},
	} {
		TrimNullableString(field.value)
		if (create || field.value.Set) && (field.value.Value == nil || *field.value.Value == "") {
			return field.name + " is required"
		}
		if field.name != "source_type" && field.value.Set {
			if !app.ValidUUID(*field.value.Value) {
				return field.name + " must be a valid UUID"
			}
			id, _ := uuid.Parse(*field.value.Value)
			*field.value.Value = id.String()
		}
	}
	if p.SourceType.Set {
		switch *p.SourceType.Value {
		case "event", "venue", "organization":
		default:
			return "invalid source_type"
		}
	}
	if p.Targets.Set {
		if p.Targets.Value == nil {
			return "targets cannot be null"
		}
		accounts := make(map[string]bool)
		for i := range *p.Targets.Value {
			target := &(*p.Targets.Value)[i]
			target.SocialAccountUuid = strings.TrimSpace(target.SocialAccountUuid)
			if !app.ValidUUID(target.SocialAccountUuid) {
				return "social_account_uuid must be a valid UUID"
			}
			id, _ := uuid.Parse(target.SocialAccountUuid)
			target.SocialAccountUuid = id.String()
			if accounts[target.SocialAccountUuid] {
				return "duplicate social_account_uuid"
			}
			accounts[target.SocialAccountUuid] = true
		}
	}
	return ""
}

func decodeSocialPost(gc *gin.Context, apiRequest *grains_api.Request, create bool) *socialPostInput {
	var payload *socialPostInput
	if err := grains_api.BindJSONStrict(gc, &payload); err != nil || payload == nil {
		apiRequest.PayloadError()
		return nil
	}
	if message := payload.validate(create); message != "" {
		apiRequest.Error(http.StatusBadRequest, message)
		return nil
	}
	return payload
}

func socialPostUUID(gc *gin.Context, apiRequest *grains_api.Request) (string, bool) {
	postUuid := gc.Param("uuid")
	if !app.ValidUUID(postUuid) {
		apiRequest.Error(http.StatusBadRequest, "uuid must be a valid UUID")
		return "", false
	}
	return postUuid, true
}

// Do not retain database error details or request values in responses or logs.
func socialPostDBError(err error) *ApiTxError {
	if errors.Is(err, pgx.ErrNoRows) {
		return &ApiTxError{Code: http.StatusNotFound, Message: "social post not found"}
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return &ApiTxError{Code: http.StatusConflict, Message: "social post target already exists"}
		case "23503", "23502", "23514", "22021", "22P02":
			return &ApiTxError{Code: http.StatusBadRequest, Message: "invalid social post data"}
		}
	}
	return &ApiTxError{Code: http.StatusInternalServerError, Message: "internal server error"}
}

func socialPostRespondError(apiRequest *grains_api.Request, err *ApiTxError) {
	if err.Code >= http.StatusInternalServerError {
		apiRequest.InternalServerError()
	} else {
		apiRequest.Error(err.Code, err.Error())
	}
}

// A single query gives posts and targets from the same snapshot. Account rows,
// including credentials, are never selected into a response.
func (h *ApiHandler) socialPostQuery() string {
	return fmt.Sprintf(`SELECT p.uuid, p.org_uuid, p.source_type, p.source_uuid,
		p.created_by, p.created_at, p.updated_at,
		COALESCE((SELECT jsonb_agg(to_jsonb(t) ORDER BY t.created_at, t.uuid)
			FROM %s.social_post_target t WHERE t.social_post_uuid = p.uuid), '[]'::jsonb)
		FROM %s.social_post p`, h.DbSchema, h.DbSchema)
}

func scanSocialPost(row pgx.Row) (model.SocialPost, error) {
	var post model.SocialPost
	err := row.Scan(&post.Uuid, &post.OrgUuid, &post.SourceType, &post.SourceUuid,
		&post.CreatedBy, &post.CreatedAt, &post.UpdatedAt, &post.Targets)
	return post, err
}

func (h *ApiHandler) socialPostPermission(gc *gin.Context, tx pgx.Tx, postUuid string) (string, *ApiTxError) {
	var orgUuid string
	err := tx.QueryRow(gc.Request.Context(), fmt.Sprintf(
		"SELECT org_uuid FROM %s.social_post WHERE uuid = $1 FOR UPDATE", h.DbSchema), postUuid).Scan(&orgUuid)
	if err != nil {
		return "", socialPostDBError(err)
	}
	return orgUuid, h.CheckAllOrgPermissionsTx(gc, tx, h.userUuid(gc), orgUuid, app.UserPermEditOrg)
}

func (h *ApiHandler) updateSocialPostTargets(gc *gin.Context, tx pgx.Tx, postUuid, orgUuid string, targets []socialPostTargetInput) *ApiTxError {
	ctx := gc.Request.Context()
	// Stable lock order prevents opposing target lists from deadlocking.
	accounts := make([]string, 0, len(targets))
	for _, target := range targets {
		accounts = append(accounts, target.SocialAccountUuid)
	}
	sort.Strings(accounts)
	for _, accountUuid := range accounts {
		var accountOrgUuid string
		var enabled bool
		err := tx.QueryRow(ctx, fmt.Sprintf(
			"SELECT org_uuid, enabled FROM %s.social_account WHERE uuid = $1 FOR SHARE", h.DbSchema), accountUuid).Scan(&accountOrgUuid, &enabled)
		if errors.Is(err, pgx.ErrNoRows) {
			return &ApiTxError{Code: http.StatusBadRequest, Message: "social account not found"}
		}
		if err != nil {
			return socialPostDBError(err)
		}
		if accountOrgUuid != orgUuid {
			return &ApiTxError{Code: http.StatusBadRequest, Message: "social account belongs to another organization"}
		}
		if !enabled {
			return &ApiTxError{Code: http.StatusBadRequest, Message: "social account is disabled"}
		}
	}
	_, err := tx.Exec(ctx, fmt.Sprintf(`DELETE FROM %s.social_post_target
		WHERE social_post_uuid = $1 AND NOT (social_account_uuid = ANY($2::uuid[]))`, h.DbSchema), postUuid, accounts)
	if err != nil {
		return socialPostDBError(err)
	}
	for _, accountUuid := range accounts {
		targetUuid, err := grains_uuid.Uuidv7String()
		if err != nil {
			return socialPostDBError(err)
		}
		// Existing targets retain their UUID, timestamps and publication metadata.
		_, err = tx.Exec(ctx, fmt.Sprintf(`INSERT INTO %s.social_post_target
			(uuid, social_post_uuid, social_account_uuid) VALUES ($1, $2, $3)
			ON CONFLICT (social_post_uuid, social_account_uuid) DO NOTHING`, h.DbSchema), targetUuid, postUuid, accountUuid)
		if err != nil {
			return socialPostDBError(err)
		}
	}
	return nil
}
