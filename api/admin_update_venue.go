package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

type venueReq struct {
	OrganizationId int      `json:"organization_id" binding:"required"`
	Name           string   `json:"name"`
	Description    *string  `json:"description"`
	OpenedAt       *string  `json:"opened_at"`
	ClosedAt       *string  `json:"closed_at"`
	ContactEmail   *string  `json:"contact_email"`
	ContactPhone   *string  `json:"contact_phone"`
	WebLink        *string  `json:"web_link"`
	Street         *string  `json:"street"`
	HouseNumber    *string  `json:"house_number"`
	PostalCode     *string  `json:"postal_code"`
	City           *string  `json:"city"`
	State          *string  `json:"state"`
	Country        *string  `json:"country"`
	Longitude      *float64 `json:"longitude"`
	Latitude       *float64 `json:"latitude"`
}

func (h *ApiHandler) AdminUpsertVenue(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-upsert-venue")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	venueUuid := gc.Param("venueUuid")

	var req venueReq
	if err := gc.ShouldBindJSON(&req); err != nil {
		debugf(err.Error())
		apiRequest.InvalidJSONInput()
		return
	}

	newVenueId := -1

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		// Insert
		if venueUuid == "" {
			err := tx.QueryRow(
				ctx,
				app.UranusInstance.SqlInsertVenue,
				req.OrganizationId,
				req.Name,
				req.Description,
				req.OpenedAt,
				req.ClosedAt,
				req.ContactEmail,
				req.ContactPhone,
				req.WebLink,
				req.Street,
				req.HouseNumber,
				req.PostalCode,
				req.City,
				req.Country,
				req.State,
				req.Longitude,
				req.Latitude,
				userUuid,
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
				app.UranusInstance.SqlUpdateVenue,
				venueUuid,
				req.Name,
				req.Description,
				req.OpenedAt,
				req.ClosedAt,
				req.ContactEmail,
				req.ContactPhone,
				req.WebLink,
				req.Street,
				req.HouseNumber,
				req.PostalCode,
				req.City,
				req.Country,
				req.State,
				req.Longitude,
				req.Latitude,
				userUuid,
			)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("update venue failed: %v", err),
				}
			}

			err = RefreshEventProjections(ctx, tx, "venue", []string{venueUuid})
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
