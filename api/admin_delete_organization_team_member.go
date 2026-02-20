package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) AdminDeleteOrganizationTeamMember(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)
	apiRequest := grains_api.NewRequest(gc, "admin-delete-organization-team-member")

	err := h.VerifyUserPassword(gc, userId)
	if err != nil {
		apiRequest.Error(http.StatusUnauthorized, err.Error())
		return
	}

	organizationId, ok := ParamInt(gc, "organizationId")
	if !ok {
		apiRequest.Error(http.StatusBadRequest, "organizationId is required")
		return
	}
	apiRequest.SetMeta("organization_id", organizationId)

	memberUserId, ok := ParamInt(gc, "memberId")
	if !ok {
		apiRequest.Error(http.StatusBadRequest, "memberId is required")
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
	_, err = tx.Exec(ctx, query, organizationId, memberUserId)
	if err != nil {
		apiRequest.Error(http.StatusInternalServerError, "failed to delete team member (#1)")
		return
	}

	query = fmt.Sprintf(
		`DELETE FROM %s.user_organization_link WHERE organization_id = $1 AND user_id = $2`,
		h.DbSchema)
	_, err = tx.Exec(ctx, query, organizationId, memberUserId)
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
