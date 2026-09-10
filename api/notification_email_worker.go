package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

type notificationEmailSender func(string, string, string, time.Duration) error

// RunNotificationEmailWorker resumes pending work after restarts. Sending/failed
// rows need operator review: SMTP cannot provide exactly-once delivery, and the
// existing timeout helper may leave its underlying send running after timeout.
func (h *ApiHandler) RunNotificationEmailWorker(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		if err := h.processNotificationEmails(ctx, sendEmailWithTimeout); err != nil && ctx.Err() == nil {
			log.Print("notification email worker: database/preparation failure; pending work retained")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (h *ApiHandler) processNotificationEmails(ctx context.Context, send notificationEmailSender) error {
	rows, err := h.DbPool.Query(ctx, fmt.Sprintf(`SELECT uuid FROM %s.notification_email_outbox WHERE status = 'pending' ORDER BY created_at LIMIT 20`, h.DbSchema))
	if err != nil {
		return err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	var firstErr error
	for _, id := range ids {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := h.deliverNotificationEmail(ctx, id, send); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (h *ApiHandler) deliverNotificationEmail(ctx context.Context, id string, send notificationEmailSender) error {
	var recipient, displayName, locale, templateContext, action string
	var metadata []byte
	query := fmt.Sprintf(`SELECT u.email, COALESCE(NULLIF(u.display_name, ''), NULLIF(CONCAT_WS(' ', u.first_name, u.last_name), ''), u.email),
 COALESCE(u.locale, 'en'), e.template_context, n.action_url, n.metadata
 FROM %s.notification_email_outbox e JOIN %s.user_notification n ON n.uuid = e.notification_uuid
 JOIN %s."user" u ON u.uuid = n.user_uuid WHERE e.uuid = $1::uuid AND e.status = 'pending'`, h.DbSchema, h.DbSchema, h.DbSchema)
	err := h.DbPool.QueryRow(ctx, query, id).Scan(&recipient, &displayName, &locale, &templateContext, &action, &metadata)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	link, err := h.dashboardURL(action)
	if err != nil {
		return err
	} // Still pending; correcting configuration resumes it.
	var data map[string]string
	if err := json.Unmarshal(metadata, &data); err != nil {
		return err
	}
	data["display_name"] = displayName
	data["action_link"] = link
	var subject, body string
	err = h.DbPool.QueryRow(ctx, app.UranusInstance.SqlGetSystemEmailTemplate, templateContext, emailLocale(locale)).Scan(&subject, &body)
	if err == pgx.ErrNoRows && emailLocale(locale) != "en" {
		err = h.DbPool.QueryRow(ctx, app.UranusInstance.SqlGetSystemEmailTemplate, templateContext, "en").Scan(&subject, &body)
	}
	if err != nil {
		return err
	} // Missing templates can be installed without losing jobs.
	subject, body = renderNotificationEmail(subject, body, data)
	// Commit the exclusive claim BEFORE calling SMTP. Concurrent workers cannot
	// send the same job, even if SMTP times out or a process restarts.
	result, err := h.DbPool.Exec(ctx, fmt.Sprintf(`UPDATE %s.notification_email_outbox SET status = 'sending', attempted_at = now() WHERE uuid = $1::uuid AND status = 'pending'`, h.DbSchema), id)
	if err != nil || result.RowsAffected() == 0 {
		return err
	}
	status := "sent"
	if err := send(recipient, subject, body, 20*time.Second); err != nil {
		status = "failed"
		log.Printf("notification email %s: delivery unconfirmed; operator review required", id)
	}
	// Persist the outcome even if shutdown/request cancellation happened during SMTP.
	finishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = h.DbPool.Exec(finishCtx, fmt.Sprintf(`UPDATE %s.notification_email_outbox SET status = $2, sent_at = CASE WHEN $2 = 'sent' THEN now() END WHERE uuid = $1::uuid AND status = 'sending'`, h.DbSchema), id, status)
	return err
}
