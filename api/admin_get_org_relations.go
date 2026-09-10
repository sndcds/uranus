package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

// PermissionNote: User must be authenticated.
// The endpoint returns all data connected to the organization.
// PermissionChecks: Organization permission is checked before querying.
// Verified: 2026-09-09, Roald

// Todo: Define the response of this endpoint, the current version is for informative

func (h *ApiHandler) AdminGetOrgRelations(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-org-relations")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	orgUuid := gc.Param("orgUuid")
	if orgUuid == "" {
		apiRequest.Required("orgUuid is required")
		return
	}

	apiRequest.SetMeta("org_uuid", orgUuid)

	var data []struct {
		TableName string          `json:"table_name"`
		Record    json.RawMessage `json:"record"`
	}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		// Check that the organization exists and that the user
		// has permission to access it.
		txErr := h.CheckOrgPermissionTx(
			gc,
			tx,
			userUuid,
			orgUuid,
			app.UserPermEditOrg,
		)
		if txErr != nil {
			return txErr
		}

		rows, err := tx.Query(
			ctx,
			app.UranusInstance.SqlAdminGetOrgRelations,
			orgUuid,
		)
		if err != nil {
			return &ApiTxError{
				Code:    http.StatusInternalServerError,
				Message: "failed to load organization data",
				Err:     err,
			}
		}
		defer rows.Close()

		type OrganizationRecord struct {
			TableName string          `json:"table_name"`
			Record    json.RawMessage `json:"record"`
		}

		for rows.Next() {
			var record OrganizationRecord

			err = rows.Scan(
				&record.TableName,
				&record.Record,
			)
			if err != nil {
				return &ApiTxError{
					Code:    http.StatusInternalServerError,
					Message: "failed to read organization data",
					Err:     err,
				}
			}

			data = append(data, record)
		}

		if err = rows.Err(); err != nil {
			return &ApiTxError{
				Code:    http.StatusInternalServerError,
				Message: "failed to read organization data",
				Err:     err,
			}
		}

		return nil
	})

	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.Error(txErr.Code, txErr.Message)
		return
	}

	apiRequest.Success(
		http.StatusOK,
		gin.H{
			"org_uuid": orgUuid,
			"data":     data,
		},
		"Organization data loaded successfully",
	)
}
