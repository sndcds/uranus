package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

type orgPartnerRequestEmailRecipient struct {
	Email       string
	Locale      string
	DisplayName *string
	FirstName   *string
	LastName    *string
}

type orgPartnerRequestEmailData struct {
	Language     string
	DisplayName  string
	FromOrgName  string
	ToOrgName    string
	Message      string
	RedirectLink string
}

func (h *ApiHandler) loadOrgPartnerRequestEmailData(ctx context.Context, fromOrgUuid, toOrgUuid string) (orgPartnerRequestEmailData, error) {
	query := fmt.Sprintf(`
		SELECT from_org.name, to_org.name, COALESCE(r.message, '')
		FROM %[1]s.organization_partner_request r
		JOIN %[1]s.organization from_org ON from_org.uuid = r.from_org_uuid
		JOIN %[1]s.organization to_org ON to_org.uuid = r.to_org_uuid
		WHERE r.from_org_uuid = $1::uuid AND r.to_org_uuid = $2::uuid
	`, h.DbSchema)

	var data orgPartnerRequestEmailData
	err := h.DbPool.QueryRow(ctx, query, fromOrgUuid, toOrgUuid).Scan(
		&data.FromOrgName, &data.ToOrgName, &data.Message,
	)
	data.RedirectLink = strings.TrimRight(h.Config.FrontendDashboard, "/") + "/admin/dashboard"
	return data, err
}

// Called only after the request transaction has committed. A failed delivery
// must not prevent the other authorized team members from being notified.
func (h *ApiHandler) sendOrgPartnerRequestEmails(fromOrgUuid, toOrgUuid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	data, err := h.loadOrgPartnerRequestEmailData(ctx, fromOrgUuid, toOrgUuid)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`
		SELECT u.email, COALESCE(u.locale, ''), u.display_name, u.first_name, u.last_name
		FROM %[1]s."user" u
		WHERE EXISTS (
			SELECT 1
			FROM %[1]s.user_organization_link uol
			JOIN %[1]s.organization_member_link oml
			  ON oml.user_uuid = uol.user_uuid AND oml.org_uuid = uol.org_uuid
			WHERE uol.user_uuid = u.uuid
			  AND uol.org_uuid = $1::uuid
			  AND oml.has_joined = TRUE
			  AND (uol.permissions & $2::bigint) <> 0
		)
	`, h.DbSchema)

	rows, err := h.DbPool.Query(ctx, query, toOrgUuid, int64(app.UserPermAnswerPartnerRequest))
	if err != nil {
		return err
	}
	// Finish reading and release the connection before sending any emails.
	recipients, err := pgx.CollectRows(rows, pgx.RowToStructByPos[orgPartnerRequestEmailRecipient])
	if err != nil {
		return err
	}

	var sendErrors []error
	for _, recipient := range recipients {
		if err := sendOrgPartnerRequestEmail("partner-request", recipient, data); err != nil {
			sendErrors = append(sendErrors, err)
		}
	}
	return errors.Join(sendErrors...)
}

// The requester is taken from UPDATE ... RETURNING, never from the accepting user.
func (h *ApiHandler) sendOrgPartnerRequestAcceptedEmail(requestingUserUuid, fromOrgUuid, toOrgUuid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	data, err := h.loadOrgPartnerRequestEmailData(ctx, fromOrgUuid, toOrgUuid)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`
		SELECT email, COALESCE(locale, ''), display_name, first_name, last_name
		FROM %s."user"
		WHERE uuid = $1::uuid
	`, h.DbSchema)
	var recipient orgPartnerRequestEmailRecipient
	err = h.DbPool.QueryRow(ctx, query, requestingUserUuid).Scan(
		&recipient.Email, &recipient.Locale, &recipient.DisplayName, &recipient.FirstName, &recipient.LastName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		// The original requester may have deleted their account in the meantime.
		return nil
	}
	if err != nil {
		return err
	}

	return sendOrgPartnerRequestEmail("partner-request-accepted", recipient, data)
}

func sendOrgPartnerRequestEmail(templateName string, recipient orgPartnerRequestEmailRecipient, data orgPartnerRequestEmailData) error {
	data.Language = app.NormalizeLocale(recipient.Locale)
	data.DisplayName = BuildUserLabel(recipient.Email, recipient.DisplayName, recipient.FirstName, recipient.LastName)

	subject, body, err := app.RenderEmailTemplate(
		fmt.Sprintf("template/email/layout/%s.html", data.Language),
		fmt.Sprintf("template/email/%s/%s.html", templateName, data.Language),
		data,
	)
	if err != nil {
		return err
	}
	return app.SendEmailWithTimeout(recipient.Email, subject, body, 20*time.Second)
}
