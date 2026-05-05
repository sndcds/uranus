package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

// AdminGetOrgChoosableVenues returns all venues that can be chosen for events of an organization.
func (h *ApiHandler) AdminGetOrgChoosableVenues(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-org-choosable-venues")
	ctx := gc.Request.Context()

	type VenueSpaceRow struct {
		OrgUuid     *string         `db:"org_uuid"`
		OrgName     *string         `db:"org_name"`
		VenueUuid   *string         `db:"venue_uuid"`
		VenueName   *string         `db:"venue_name"`
		SpaceUuid   *string         `db:"space_uuid"`
		SpaceName   *string         `db:"space_name"`
		City        *string         `db:"city"`
		Country     *string         `db:"country"`
		Permissions app.Permissions `db:"permissions"`
	}

	type SpaceDTO struct {
		Uuid string `json:"uuid"`
		Name string `json:"name"`
	}

	type VenueDTO struct {
		Uuid    string     `json:"uuid"`
		Name    string     `json:"name"`
		OrgUuid string     `json:"org_uuid"`
		City    string     `json:"city"`
		Country string     `json:"country"`
		Spaces  []SpaceDTO `json:"spaces"`
	}

	type Response struct {
		Venues []VenueDTO `json:"venues"`
	}

	orgUuid := gc.Param("orgUuid")
	if orgUuid == "" {
		apiRequest.Required("orgUuid is required")
		return
	}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		rows, err := tx.Query(ctx, app.UranusInstance.SqlAdminChoosableVenues, orgUuid, app.OrgPermChooseVenue)
		if err != nil {
			debugf(err.Error())
			return TxInternalError(nil)
		}
		defer rows.Close()

		venuesMap := make(map[string]*VenueDTO)
		for rows.Next() {
			var vs VenueSpaceRow
			err := rows.Scan(
				&vs.OrgUuid,
				&vs.OrgName,
				&vs.VenueUuid,
				&vs.VenueName,
				&vs.SpaceUuid,
				&vs.SpaceName,
				&vs.City,
				&vs.Country,
				&vs.Permissions,
			)
			if err != nil {
				debugf(err.Error())
				return TxInternalError(nil)
			}

			if vs.OrgUuid != nil && vs.VenueUuid != nil && vs.VenueName != nil {
				venue, ok := venuesMap[*vs.VenueUuid]

				if !ok {
					city := ""
					if vs.City != nil {
						city = *vs.City
					}
					country := ""
					if vs.Country != nil {
						country = *vs.Country
					}
					venue = &VenueDTO{
						Uuid:    *vs.VenueUuid,
						Name:    *vs.VenueName,
						OrgUuid: *vs.OrgUuid,
						City:    city,
						Country: country,
						Spaces:  []SpaceDTO{},
					}
					venuesMap[*vs.VenueUuid] = venue
				}
				// Add space (if exists)
				if vs.SpaceUuid != nil && vs.SpaceName != nil {
					venue.Spaces = append(venue.Spaces, SpaceDTO{
						Uuid: *vs.SpaceUuid,
						Name: *vs.SpaceName,
					})
				}
			}

		}

		venues := make([]VenueDTO, 0, len(venuesMap))
		for _, v := range venuesMap {
			venues = append(venues, *v)
		}

		apiRequest.Success(http.StatusOK, Response{Venues: venues}, "")
		return nil
	})
	if txErr != nil {
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}
}
