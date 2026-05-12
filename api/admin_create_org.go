package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/grains/grains_token"
	"github.com/sndcds/grains/grains_uuid"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminCreateOrg(gc *gin.Context) {
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)
	apiRequest := grains_api.NewRequest(gc, "create-org")

	type Payload struct {
		Name string `json:"org_name" binding:"required"`
	}

	payload, ok := grains_api.DecodeJSONBody[Payload](gc, apiRequest)
	if !ok {
		return
	}

	// Validate name
	payload.Name = strings.TrimSpace(payload.Name)
	if payload.Name == "" {
		apiRequest.Error(http.StatusBadRequest, "organization name cannot be empty")
		return
	}
	apiRequest.Metadata["organization_name"] = payload.Name

	var err error
	orgUuid := ""

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		orgUuid, err = grains_uuid.Uuidv7String()
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to generate uuid: %v", err),
			}
		}

		apiImportToken := grains_token.GenerateUuid()
		query := fmt.Sprintf(`
			INSERT INTO %s.organization (uuid, created_by, name, api_import_token)
			VALUES ($1::uuid, $2::uuid, $3, $4)`,
			h.DbSchema)
		_, err = tx.Exec(ctx, query, orgUuid, userUuid, payload.Name, apiImportToken)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to insert organization: %v", err),
			}
		}

		// Insert user_organization_link
		insertLinkQuery := fmt.Sprintf(
			`INSERT INTO %s.user_organization_link (user_uuid, org_uuid, permissions) VALUES ($1::uuid, $2::uuid, $3)`,
			h.DbSchema)
		_, err = tx.Exec(gc, insertLinkQuery, userUuid, orgUuid, app.UserPermCombinationAdmin)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("insert user_organization_link failed: %w", err),
			}
		}

		// Insert organization_member_link
		insertMemberQuery := fmt.Sprintf(
			`INSERT INTO %s.organization_member_link (org_uuid, user_uuid, has_joined) VALUES ($1::uuid, $2::uuid, $3)`,
			h.DbSchema)
		_, err = tx.Exec(gc, insertMemberQuery, orgUuid, userUuid, true)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("insert organization_member_link failed: %w", err),
			}
		}

		return nil
	})
	if txErr != nil {
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	apiRequest.Metadata["org_uuid"] = orgUuid
	apiRequest.SuccessNoData(http.StatusCreated, "")
}
