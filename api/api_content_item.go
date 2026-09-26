package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

// LoadContentItem prepares a source for an authorized organization editor.
// It uses the existing pool and permissions; it does not persist or render content.
func (h *ApiHandler) LoadContentItem(
	gc *gin.Context,
	orgUuid string,
	sourceType string,
	sourceUuid string,
	lang string,
) (*model.ContentItem, *ApiTxError) {
	if !app.ValidUUID(h.userUuid(gc)) {
		return nil, &ApiTxError{Code: http.StatusUnauthorized, Message: "authentication required"}
	}
	if !app.ValidUUID(orgUuid) || !app.ValidUUID(sourceUuid) {
		return nil, &ApiTxError{Code: http.StatusBadRequest, Message: "org_uuid and source_uuid must be valid UUIDs"}
	}
	switch sourceType {
	case "event", "venue", "organization":
	default:
		return nil, &ApiTxError{Code: http.StatusBadRequest, Message: "invalid source_type"}
	}

	var item *model.ContentItem
	txErr := WithTransaction(gc.Request.Context(), h.DbPool, func(tx pgx.Tx) *ApiTxError {
		if txErr := h.CheckAllOrgPermissionsTx(gc, tx, h.userUuid(gc), orgUuid, app.UserPermEditOrg); txErr != nil {
			return txErr
		}

		var txErr *ApiTxError
		item, txErr = h.loadContentItemTx(gc.Request.Context(), tx, orgUuid, sourceType, sourceUuid, lang)
		return txErr
	})
	if txErr != nil {
		return nil, contentItemError(txErr)
	}

	return item, nil
}

// LoadSocialPostContentItem resolves the persisted source reference without
// changing the post, its targets, or the existing CRUD responses.
func (h *ApiHandler) LoadSocialPostContentItem(
	gc *gin.Context,
	postUuid string,
	lang string,
) (*model.ContentItem, *ApiTxError) {
	if !app.ValidUUID(h.userUuid(gc)) {
		return nil, &ApiTxError{Code: http.StatusUnauthorized, Message: "authentication required"}
	}
	if !app.ValidUUID(postUuid) {
		return nil, &ApiTxError{Code: http.StatusBadRequest, Message: "post uuid must be a valid UUID"}
	}

	var item *model.ContentItem
	txErr := WithTransaction(gc.Request.Context(), h.DbPool, func(tx pgx.Tx) *ApiTxError {
		orgUuid, txErr := h.socialPostPermission(gc, tx, postUuid)
		if txErr != nil {
			return txErr
		}

		var sourceType, sourceUuid string
		query := fmt.Sprintf(`
			SELECT source_type, source_uuid
			FROM %s.social_post
			WHERE uuid = $1`,
			h.DbSchema)
		if err := tx.QueryRow(gc.Request.Context(), query, postUuid).Scan(&sourceType, &sourceUuid); err != nil {
			return contentItemDBError(err)
		}

		item, txErr = h.loadContentItemTx(gc.Request.Context(), tx, orgUuid, sourceType, sourceUuid, lang)
		return txErr
	})
	if txErr != nil {
		return nil, contentItemError(txErr)
	}

	return item, nil
}

// Also sanitize transaction failures: raw database errors may contain row data.
func contentItemError(txErr *ApiTxError) *ApiTxError {
	if txErr.Code >= http.StatusInternalServerError {
		return &ApiTxError{Code: http.StatusInternalServerError, Message: "internal server error"}
	}
	return txErr
}

func contentItemDBError(err error) *ApiTxError {
	if errors.Is(err, pgx.ErrNoRows) {
		return &ApiTxError{Code: http.StatusNotFound, Message: "source not found in organization"}
	}
	return &ApiTxError{Code: http.StatusInternalServerError, Message: "internal server error"}
}

// All three source queries use v for their optional venue/organization address.
const contentLocationSQL = `CASE WHEN v.uuid IS NULL THEN NULL ELSE jsonb_build_object(
	'uuid', v.uuid, 'name', v.name, 'street', v.street, 'house_number', v.house_number,
	'postal_code', v.postal_code, 'city', v.city, 'state', v.state,
	'country', v.country, 'url', v.web_link) END`

func (h *ApiHandler) loadContentItemTx(
	ctx context.Context,
	tx pgx.Tx,
	orgUuid string,
	sourceType string,
	sourceUuid string,
	lang string,
) (*model.ContentItem, *ApiTxError) {
	var query string

	// Load current source tables, as the detail handlers do. Projections are for
	// event searches and do not contain translated source text.
	switch sourceType {
	case "event":
		query = fmt.Sprintf(`
			SELECT e.uuid, e.org_uuid, e.content_iso_639_1, e.title, e.subtitle,
				e.description, e.summary, e.source_link, e.ticket_link, e.online_link, %s
			FROM %s.event e
			LEFT JOIN %s.venue v ON v.uuid = e.venue_uuid
			WHERE e.uuid = $1 AND e.org_uuid = $2
			FOR SHARE OF e`,
			contentLocationSQL, h.DbSchema, h.DbSchema)
	case "venue":
		query = fmt.Sprintf(`
			SELECT v.uuid, v.org_uuid, v.content_iso_639_1, v.name, NULL,
				v.description, v.summary, v.web_link, v.ticket_link, NULL, %s
			FROM %s.venue v
			WHERE v.uuid = $1 AND v.org_uuid = $2
			FOR SHARE OF v`,
			contentLocationSQL, h.DbSchema)
	case "organization":
		query = fmt.Sprintf(`
			SELECT v.uuid, v.uuid, v.content_iso_639_1, v.name, NULL,
				v.description, NULL, v.web_link, NULL, NULL, %s
			FROM %s.organization v
			WHERE v.uuid = $1 AND v.uuid = $2
			FOR SHARE OF v`,
			contentLocationSQL, h.DbSchema)
	default:
		return nil, &ApiTxError{Code: http.StatusBadRequest, Message: "invalid source_type"}
	}

	item := &model.ContentItem{
		SourceType: sourceType,
		Language:   app.NormalizeLocale(lang),
		Dates:      make([]model.ContentDate, 0),
		Images:     make([]model.Image, 0),
	}
	err := tx.QueryRow(ctx, query, sourceUuid, orgUuid).Scan(
		&item.SourceUuid,
		&item.OrgUuid,
		&item.ContentLanguage,
		&item.Title,
		&item.Subtitle,
		&item.Description,
		&item.Summary,
		&item.Url,
		&item.TicketLink,
		&item.OnlineLink,
		&item.Location,
	)
	if err != nil {
		return nil, contentItemDBError(err)
	}

	if sourceType == "event" {
		if txErr := h.loadContentDatesTx(ctx, tx, item); txErr != nil {
			return nil, txErr
		}
	}
	if txErr := h.loadContentImagesTx(ctx, tx, item); txErr != nil {
		return nil, txErr
	}

	return item, nil
}

func (h *ApiHandler) loadContentDatesTx(ctx context.Context, tx pgx.Tx, item *model.ContentItem) *ApiTxError {
	// Match get-event-dates.sql: date venue overrides event venue; an explicit
	// date venue also controls space selection. Keep every occurrence.
	query := fmt.Sprintf(`
		SELECT ed.uuid, TO_CHAR(ed.start_date, 'YYYY-MM-DD'), TO_CHAR(ed.start_time, 'HH24:MI'),
			TO_CHAR(ed.end_date, 'YYYY-MM-DD'), TO_CHAR(ed.end_time, 'HH24:MI'),
			TO_CHAR(ed.entry_time, 'HH24:MI'), ed.duration, ed.all_day,
			ed.ticket_link, %s, s.uuid, s.name
		FROM %s.event_date ed
		JOIN %s.event e ON e.uuid = ed.event_uuid
		LEFT JOIN %s.venue v ON v.uuid = COALESCE(ed.venue_uuid, e.venue_uuid)
		LEFT JOIN %s.space s ON s.uuid = CASE
			WHEN ed.venue_uuid IS NOT NULL THEN ed.space_uuid ELSE e.space_uuid END
		WHERE e.uuid = $1
		ORDER BY ed.start_date, ed.start_time, ed.uuid`,
		contentLocationSQL, h.DbSchema, h.DbSchema, h.DbSchema, h.DbSchema)

	rows, err := tx.Query(ctx, query, item.SourceUuid)
	if err != nil {
		return contentItemDBError(err)
	}
	defer rows.Close()

	for rows.Next() {
		var date model.ContentDate
		err := rows.Scan(
			&date.Uuid, &date.StartDate, &date.StartTime, &date.EndDate, &date.EndTime,
			&date.EntryTime, &date.Duration, &date.AllDay, &date.TicketLink,
			&date.Location, &date.SpaceUuid, &date.SpaceName,
		)
		if err != nil {
			return contentItemDBError(err)
		}
		item.Dates = append(item.Dates, date)
	}
	if err := rows.Err(); err != nil {
		return contentItemDBError(err)
	}

	return nil
}

func (h *ApiHandler) loadContentImagesTx(ctx context.Context, tx pgx.Tx, item *model.ContentItem) *ApiTxError {
	// Use the existing image links and requested-language license lookup,
	// including the all-rights-reserved fallback from the public detail queries.
	query := fmt.Sprintf(`
		SELECT pi.uuid, pil.identifier, pi.alt_text, pi.width, pi.height,
			pi.creator_name, pi.copyright, pi.description,
			COALESCE(lic.key, fallback.key), COALESCE(lic.name, fallback.name),
			COALESCE(lic.description, fallback.description), COALESCE(pi.ai_label, 'none'),
			pi.focus_x, pi.focus_y
		FROM %s.pluto_image_link pil
		JOIN %s.pluto_image pi ON pi.uuid = pil.pluto_image_uuid
		LEFT JOIN %s.license_i18n lic ON lic.key = pi.license AND lic.iso_639_1 = $3
		LEFT JOIN %s.license_i18n fallback ON fallback.key = 'all-rights-reserved' AND fallback.iso_639_1 = $3
		WHERE pil.context = $1 AND pil.context_uuid = $2
		ORDER BY pil.identifier, pi.uuid`,
		h.DbSchema, h.DbSchema, h.DbSchema, h.DbSchema)

	rows, err := tx.Query(ctx, query, item.SourceType, item.SourceUuid, item.Language)
	if err != nil {
		return contentItemDBError(err)
	}
	defer rows.Close()

	for rows.Next() {
		var image model.Image
		err := rows.Scan(
			&image.Uuid, &image.Identifier, &image.Alt, &image.Width, &image.Height,
			&image.Creator, &image.Copyright, &image.Description, &image.License,
			&image.LicenseName, &image.LicenseDescription, &image.AiLabel, &image.FocusX, &image.FocusY,
		)
		if err != nil {
			return contentItemDBError(err)
		}
		image.Url = ImageUrl(image.Uuid)
		item.Images = append(item.Images, image)
	}
	if err := rows.Err(); err != nil {
		return contentItemDBError(err)
	}

	return nil
}
