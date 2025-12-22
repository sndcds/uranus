package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

type venueReq struct {
	OrganizationId int      `json:"organizationId" binding:"required"`
	Name           string   `json:"name"`
	Description    *string  `json:"description"`
	OpenedAt       *string  `json:"opened_at"`
	ClosedAt       *string  `json:"closed_at"`
	ContactEmail   *string  `json:"contact_email"`
	ContactPhone   *string  `json:"contact_phone"`
	WebsiteUrl     *string  `json:"website_url"`
	Street         *string  `json:"street"`
	HouseNumber    *string  `json:"house_number"`
	PostalCode     *string  `json:"postal_code"`
	City           *string  `json:"city"`
	StateCode      *string  `json:"state_code"`
	CountryCode    *string  `json:"country_code"`
	Longitude      *float64 `json:"longitude"`
	Latitude       *float64 `json:"latitude"`
}

func (h *ApiHandler) AdminUpsertVenue(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := gc.GetInt("user-id")

	venueId := ParamIntDefault(gc, "venueId", -1)

	var req venueReq
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newVenueId := -1

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		// Insert
		if venueId < 0 {
			err := tx.QueryRow(
				ctx,
				app.Singleton.SqlInsertVenue,
				req.OrganizationId,
				req.Name,
				req.Description,
				req.OpenedAt,
				req.ClosedAt,
				req.ContactEmail,
				req.ContactPhone,
				req.WebsiteUrl,
				req.Street,
				req.HouseNumber,
				req.PostalCode,
				req.City,
				req.CountryCode,
				req.StateCode,
				req.Longitude,
				req.Latitude,
				userId,
			).Scan(&newVenueId)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusBadRequest,
					Err:  fmt.Errorf("insert venue failed: %v", err),
				}
			}
		} else {
			_, err := tx.Exec(
				ctx,
				app.Singleton.SqlUpdateVenue,
				venueId,
				req.Name,
				req.Description,
				req.OpenedAt,
				req.ClosedAt,
				req.ContactEmail,
				req.ContactPhone,
				req.WebsiteUrl,
				req.Street,
				req.HouseNumber,
				req.PostalCode,
				req.City,
				req.CountryCode,
				req.StateCode,
				req.Longitude,
				req.Latitude,
				userId,
			)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("update venue failed: %v", err),
				}
			}

			err = RefreshEventProjections(ctx, tx, "venue", []int{venueId})
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("refresh projection tables failed: %v", err),
				}
			}
		}

		return nil
	})

	if txErr != nil {
		gc.JSON(txErr.Code, gin.H{"error": txErr.Error()})
		return
	}

	if newVenueId >= 0 {
		gc.JSON(http.StatusOK, gin.H{
			"message":         "Venue created successfully",
			"organization_id": newVenueId,
		})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"message": "Venue updated successfully"})
}
