package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"
)

const notificationInviteAccepted = "organization_team_invite_accepted"
const notificationTeamJoined = "organization_team_joined"

// Only server configuration determines mail destinations; public Accept requests
// cannot supply an origin. Frontend is the existing dashboard base URL setting.
func (h *ApiHandler) dashboardURL(path string) (string, error) {
	base, err := url.Parse(strings.TrimRight(h.Config.Frontend, "/"))
	if err != nil || base == nil || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return "", fmt.Errorf("frontend must be an absolute HTTP(S) dashboard URL")
	}
	return base.String() + path, nil
}

func (h *ApiHandler) enqueueTeamOnboarding(ctx context.Context, tx pgx.Tx, token, org, orgName, member string, inviter *string) error {
	// An internal fingerprint deduplicates the acceptance event without retaining
	// its bearer token. It is never returned by the notification API.
	digest := sha256.Sum256([]byte(token))
	eventKey := hex.EncodeToString(digest[:])
	var memberName string
	err := tx.QueryRow(ctx, fmt.Sprintf(`SELECT COALESCE(NULLIF(display_name, ''), NULLIF(CONCAT_WS(' ', first_name, last_name), ''), username, '') FROM %s."user" WHERE uuid = $1::uuid`, h.DbSchema), member).Scan(&memberName)
	if err != nil {
		return err
	}
	metadata, err := json.Marshal(map[string]string{"organization_name": orgName, "member_name": memberName})
	if err != nil {
		return err
	}
	insert := func(recipient, kind, template, action string) error {
		query := fmt.Sprintf(`WITH notification AS (
    INSERT INTO %s.user_notification (user_uuid, type, event_key, organization_uuid, actor_user_uuid, target_user_uuid, action_url, metadata)
    VALUES ($1::uuid, $2, $3, $4::uuid, $5::uuid, $5::uuid, $6, $7::jsonb)
    ON CONFLICT (user_uuid, type, event_key) DO NOTHING RETURNING uuid)
   INSERT INTO %s.notification_email_outbox (notification_uuid, template_context)
   SELECT uuid, $8 FROM notification ON CONFLICT (notification_uuid) DO NOTHING`, h.DbSchema, h.DbSchema)
		_, err := tx.Exec(ctx, query, recipient, kind, eventKey, org, member, action, metadata, template)
		return err
	}
	if inviter != nil {
		if err := insert(*inviter, notificationInviteAccepted, "team-invite-accepted", fmt.Sprintf("/admin/org/%s/member/%s/permissions", org, member)); err != nil {
			return err
		}
	}
	return insert(member, notificationTeamJoined, "team-joined", "/admin/orgs")
}

func emailLocale(locale string) string {
	switch locale {
	case "de", "da":
		return locale
	default:
		return "en"
	}
}

// Templates use the existing {{variable}} convention. Values are escaped in
// HTML, and subject values cannot inject mail headers.
func renderNotificationEmail(subject, body string, data map[string]string) (string, string) {
	var plain, escaped []string
	for key, value := range data {
		plain = append(plain, "{{"+key+"}}", strings.NewReplacer("\r", " ", "\n", " ").Replace(value))
		escaped = append(escaped, "{{"+key+"}}", html.EscapeString(value))
	}
	return strings.NewReplacer(plain...).Replace(subject), strings.NewReplacer(escaped...).Replace(body)
}
