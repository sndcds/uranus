package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

// PermissionNote: User must be authenticated.
// The endpoint only returns the organization if the authenticated user
// is linked to it via the user_organization_link table.
// Purpose: Retrieves details of a specific organization for authorized users.
// PermissionChecks: Unnecessary.
// Verified: 2026-01-12, Roald

func (h *ApiHandler) AdminGetOrg(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-org")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	orgUuid := gc.Param("orgUuid")
	if orgUuid == "" {
		apiRequest.Required("orgUuid is required")
		return
	}

	var data map[string]interface{}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		txErr := h.CheckOrgPermissionTx(gc, tx, userUuid, orgUuid, app.UserPermEditOrg)
		if txErr != nil {
			return txErr
		}

		query := app.UranusInstance.SqlAdminGetOrg
		rows, err := tx.Query(ctx, query, orgUuid, userUuid)
		if err != nil {
			return TxInternalError(err)
		}
		defer rows.Close()

		if !rows.Next() {
			return &ApiTxError{
				Code: http.StatusNotFound,
				Err:  fmt.Errorf("organization not found"),
			}
		}

		fieldDescriptions := rows.FieldDescriptions()
		values, err := rows.Values()
		if err != nil {
			return TxInternalError(err)
		}

		data = make(map[string]interface{}, len(values))
		for i, fd := range fieldDescriptions {
			data[string(fd.Name)] = values[i]
		}

		return nil
	})

	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	apiRequest.Success(http.StatusOK, data)
}
