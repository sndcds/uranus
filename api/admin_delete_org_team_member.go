package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminDeleteOrgTeamMember(gc *gin.Context) {
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)
	apiRequest := grains_api.NewRequest(gc, "admin-delete-org-team-member")

	err := h.VerifyUserPassword(gc, userUuid)
	if err != nil {
		apiRequest.Error(http.StatusUnauthorized, err.Error())
		return
	}

	orgUuid := gc.Param("orgUuid")
	if orgUuid == "" {
		apiRequest.Required("orgUuid is required")
		return
	}
	apiRequest.SetMeta("org_uuid", orgUuid)

	memberUuid := gc.Param("memberUuid")
	if memberUuid == "" {
		apiRequest.Required("memberUuid is required")
		return
	}
	apiRequest.SetMeta("member_uuid", memberUuid)

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		txErr := h.CheckAllOrgPermissionsTx(
			gc, tx, userUuid, orgUuid,
			app.UserPermManageTeam)
		if txErr != nil {
			return txErr
		}

		query := fmt.Sprintf(
			`DELETE FROM %s.organization_member_link
			 WHERE org_uuid = $1::uuid
			   AND user_uuid = $2::uuid`,
			h.DbSchema)

		result, err := tx.Exec(ctx, query, orgUuid, memberUuid)
		if err != nil {
			return TxInternalError(err)
		}

		if result.RowsAffected() == 0 {
			return &ApiTxError{
				Code: http.StatusNotFound,
				Err:  fmt.Errorf("organization team member not found"),
			}
		}

		query = fmt.Sprintf(
			`DELETE FROM %s.user_organization_link
			 WHERE org_uuid = $1::uuid
			   AND user_uuid = $2::uuid`,
			h.DbSchema)

		_, err = tx.Exec(ctx, query, orgUuid, memberUuid)
		if err != nil {
			return TxInternalError(err)
		}

		return nil
	})

	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	apiRequest.SuccessNoData(
		http.StatusOK,
		"organization team member deleted successfully",
	)
}
