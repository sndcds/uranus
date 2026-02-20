package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/grains/grains_token"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminCreateOrganization(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)
	apiRequest := grains_api.NewRequest(gc, "create-organization")

	type Payload struct {
		Name string `json:"name" binding:"required"`
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

	apiKey := grains_token.GenerateUuid()

	var newOrgId int
	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		query := fmt.Sprintf(`INSERT INTO %s.organization (name, api_key) VALUES ($1, $2) RETURNING id`, h.DbSchema)
		err := tx.QueryRow(ctx, query, payload.Name, apiKey).Scan(&newOrgId)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to insert organization: %v", err),
			}
		}

		// Insert user_organization_link
		insertLinkQuery := fmt.Sprintf(
			`INSERT INTO %s.user_organization_link (user_id, organization_id, permissions) VALUES ($1, $2, $3)`,
			h.DbSchema)
		_, err = tx.Exec(gc, insertLinkQuery, userId, newOrgId, app.PermCombinationAdmin)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("insert user_organization_link failed: %w", err),
			}
		}

		// Insert organization_member_link
		insertMemberQuery := fmt.Sprintf(
			`INSERT INTO %s.organization_member_link (organization_id, user_id, has_joined) VALUES ($1, $2, $3)`,
			h.DbSchema)
		_, err = tx.Exec(gc, insertMemberQuery, newOrgId, userId, true)
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

	// Success response
	apiRequest.Metadata["organization_id"] = newOrgId
	apiRequest.SuccessNoData(http.StatusCreated, "")
}
