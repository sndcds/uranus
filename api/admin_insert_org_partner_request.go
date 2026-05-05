package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminInsertOrgPartnerRequest(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-insert-org-partner-request")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	fromOrgUuid := gc.Param("orgUuid")
	if fromOrgUuid == "" {
		apiRequest.Required("missing or invalid orgUuid")
		return
	}

	var body struct {
		ToOrgUuid string  `json:"to_org_uuid" binding:"required"`
		Message   *string `json:"message"`
	}

	if err := gc.ShouldBindJSON(&body); err != nil {
		apiRequest.InvalidJSONInput()
		return
	}

	if fromOrgUuid == body.ToOrgUuid {
		apiRequest.Error(http.StatusConflict, "(#1) An organization cannot assign permissions to itself")
		return
	}

	var message any
	if body.Message != nil {
		message = strings.TrimSpace(*body.Message)
	} else {
		message = nil
	}

	apiStatus := http.StatusBadRequest
	apiMessage := ""

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		// Check if partnership already exists
		partnershipExistQuery := fmt.Sprintf(`
			SELECT EXISTS(
				SELECT 1 FROM %s.organization_access_grants g
				WHERE g.src_org_uuid = $1::uuid AND g.dst_org_uuid = $2::uuid
		    )`,
			h.DbSchema)

		var partnershipExists bool
		err := tx.QueryRow(
			ctx, partnershipExistQuery, body.ToOrgUuid, fromOrgUuid).Scan(&partnershipExists)
		if err != nil {
			return TxInternalError(err)
		}
		if partnershipExists {
			apiStatus = http.StatusConflict
			apiMessage = "(#1) partnership already exists"
			return nil
		}

		// Check if partnership already has been requested

		partnershipRequestedQuery := fmt.Sprintf(`
			SELECT EXISTS(
				SELECT 1 FROM %s.organization_partner_request r
				WHERE r.from_org_uuid = $1::uuid AND r.to_org_uuid = $2::uuid
		    )`,
			h.DbSchema)

		var partnershipRequested bool
		err = tx.QueryRow(
			ctx, partnershipRequestedQuery, fromOrgUuid, body.ToOrgUuid).Scan(&partnershipRequested)
		if err != nil {
			return TxInternalError(err)
		}
		if partnershipRequested {
			apiStatus = http.StatusConflict
			apiMessage = "(#2) partner request already exists"
			return nil
		}

		//

		res, err := tx.Exec(
			ctx, app.UranusInstance.SqlAdminInsertOrgPartnerRequest,
			userUuid, fromOrgUuid, body.ToOrgUuid, message)
		if err != nil {
			debugf(err.Error())
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				debugf("pgErr.Code: %s", pgErr.Code)
				apiStatus = http.StatusConflict
				apiMessage = "(#2) partner request already exists"
				return nil
			}
			return TxInternalError(err)
		}

		rows := res.RowsAffected()
		if rows == 0 {
			apiStatus = http.StatusNotFound
			apiMessage = "(#3) target organization does not exist or request was not created"
			return nil
		}

		apiStatus = http.StatusCreated
		apiMessage = "request created successfully"
		return nil
	})
	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.InternalServerError()
		return
	}

	apiRequest.SuccessNoData(apiStatus, apiMessage)
}

func (h *ApiHandler) AdminInsertOrgPartnerAccept(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-insert-org-partner-request")
	ctx := gc.Request.Context()

	orgUuid := gc.Param("orgUuid")
	if orgUuid == "" {
		apiRequest.Required("missing or invalid orgUuid")
		return
	}
	apiRequest.SetMeta("org_uuid", orgUuid)

	partnerUuid := gc.Param("partnerUuid")
	if partnerUuid == "" {
		apiRequest.Required("missing or invalid partnerUuid")
		return
	}
	apiRequest.SetMeta("partner_uuid", partnerUuid)

	tx, err := h.DbPool.Begin(ctx)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}
	defer tx.Rollback(ctx)

	// Accept request only if still pending
	updateQuery := fmt.Sprintf(`
		UPDATE %s.organization_partner_request
		SET status = 'accepted'
		WHERE from_org_uuid = $1::uuid
		  AND to_org_uuid = $2::uuid
		  AND status = 'pending'
	`, h.DbSchema)

	res, err := tx.Exec(ctx, updateQuery, partnerUuid, orgUuid)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	rowsAffected := res.RowsAffected()
	if rowsAffected == 0 {
		apiRequest.Error(http.StatusConflict, "(#1) request is not pending or already processed")
		return
	}

	// Insert access grant
	insertQuery := fmt.Sprintf(`
		INSERT INTO %s.organization_access_grants (
			src_org_uuid,
			dst_org_uuid,
			permissions
		)
		VALUES ($1::uuid, $2::uuid, 0)
		ON CONFLICT (src_org_uuid, dst_org_uuid) DO NOTHING
	`, h.DbSchema)

	_, err = tx.Exec(ctx, insertQuery, orgUuid, partnerUuid)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "OK")
}

func (h *ApiHandler) AdminInsertOrgPartnerReject(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-insert-org-partner-request")
	// ctx := gc.Request.Context()

	orgUuid := gc.Param("orgUuid")
	if orgUuid == "" {
		apiRequest.Required("missing or invalid orgUuid")
		return
	}
	apiRequest.SetMeta("org_uuid", orgUuid)

	partnerUuid := gc.Param("partnerUuid")
	if partnerUuid == "" {
		apiRequest.Required("missing or invalid partnerUuid")
		return
	}
	apiRequest.SetMeta("partner_uuid", partnerUuid)

	apiRequest.SuccessNoData(http.StatusOK, "OK")
}
