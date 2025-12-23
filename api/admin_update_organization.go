package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

type organizationReq struct {
	Name                  string   `json:"name"`
	Description           *string  `json:"description"`
	LegalFormID           *int     `json:"legal_form_id"`
	HoldingOrganizationID *int     `json:"holding_organization_id"`
	Nonprofit             *bool    `json:"nonprofit"`
	ContactEmail          *string  `json:"contact_email"`
	ContactPhone          *string  `json:"contact_phone"`
	WebsiteUrl            *string  `json:"website_url"`
	Street                *string  `json:"street"`
	HouseNumber           *string  `json:"house_number"`
	PostalCode            *string  `json:"postal_code"`
	City                  *string  `json:"city"`
	StateCode             *string  `json:"state_code"`
	CountryCode           *string  `json:"country_code"`
	AddressAddition       *string  `json:"address_addition"`
	Longitude             *float64 `json:"longitude"`
	Latitude              *float64 `json:"latitude"`
}

func (h *ApiHandler) AdminUpsertOrganization(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := gc.GetInt("user-id")

	organizationId := ParamIntDefault(gc, "organizationId", -1)

	var req organizationReq
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var newOrganizationId int

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		// Insert
		if organizationId < 0 {
			err := tx.QueryRow(
				ctx,
				app.Singleton.SqlInsertOrganization,
				req.Name,
				req.Description,
				req.LegalFormID,
				req.Nonprofit,
				req.ContactEmail,
				req.ContactPhone,
				req.WebsiteUrl,
				req.Street,
				req.HouseNumber,
				req.PostalCode,
				req.City,
				req.CountryCode,
				req.StateCode,
				req.AddressAddition,
				req.Longitude,
				req.Latitude,
				userId,
			).Scan(&newOrganizationId)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("insert organization failed: %w", err),
				}
			}

			// Insert user_organization_link
			insertLinkQuery := `
INSERT INTO {{schema}}.user_organization_link (user_id, organization_id, permissions)
VALUES ($1, $2, $3)
`
			insertLinkQuery = strings.Replace(insertLinkQuery, "{{schema}}", h.Config.DbSchema, 1)

			_, err = tx.Exec(gc, insertLinkQuery, userId, newOrganizationId, app.PermCombinationAdmin)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("insert user_organization_link failed: %w", err),
				}
			}

			// Insert organization_member_link
			insertMemberQuery := fmt.Sprintf(`
INSERT INTO %s.organization_member_link
(organization_id, user_id, has_joined, member_role_id)
VALUES ($1, $2, $3, $4)`,
				h.Config.DbSchema)
			_, err = tx.Exec(gc, insertMemberQuery, newOrganizationId, userId, true, 1)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("insert organization_member_link failed: %w", err),
				}
			}

			return nil
		}

		// Update
		_, err := tx.Exec(
			ctx,
			app.Singleton.SqlUpdateOrganization,
			organizationId,
			req.Name,
			req.Description,
			req.LegalFormID,
			req.Nonprofit,
			req.ContactEmail,
			req.ContactPhone,
			req.WebsiteUrl,
			req.Street,
			req.HouseNumber,
			req.PostalCode,
			req.City,
			req.CountryCode,
			req.StateCode,
			req.AddressAddition,
			req.Longitude,
			req.Latitude,
			userId,
		)

		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("update organization failed: %w", err),
			}
		}

		err = RefreshEventProjections(ctx, tx, "organization", []int{organizationId})
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("refresh projections failed: %w", err),
			}
		}

		return nil
	})
	if txErr != nil {
		gc.JSON(txErr.Code, gin.H{"error": txErr.Error()})
		return
	}

	if organizationId < 0 {
		gc.JSON(http.StatusOK, gin.H{"message": "Organization created successfully", "organization_id": newOrganizationId})
	} else {
		gc.JSON(http.StatusOK, gin.H{"message": "Organization updated successfully"})
	}
}
