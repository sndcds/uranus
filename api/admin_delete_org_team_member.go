package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
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

	memberUserId, ok := ParamInt(gc, "memberId")
	if !ok {
		apiRequest.Required("memberId is required")
		return
	}
	apiRequest.SetMeta("member_id", memberUserId)

	tx, err := h.DbPool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		apiRequest.Error(http.StatusInternalServerError, "transaction failed (#1)")
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	query := fmt.Sprintf(
		`DELETE FROM %s.organization_member_link WHERE organization_id = $1 AND user_id = $2`,
		h.DbSchema)
	_, err = tx.Exec(ctx, query, orgUuid, memberUserId)
	if err != nil {
		apiRequest.Error(http.StatusInternalServerError, "failed to delete team member (#1)")
		return
	}

	query = fmt.Sprintf(
		`DELETE FROM %s.user_organization_link WHERE organization_id = $1 AND user_id = $2`,
		h.DbSchema)
	_, err = tx.Exec(ctx, query, orgUuid, memberUserId)
	if err != nil {
		apiRequest.Error(http.StatusInternalServerError, "failed to delete team member (#2)")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		apiRequest.Error(http.StatusInternalServerError, "transaction failed (#2)")
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "organization team member deleted successfully")
}
