package api

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *ApiHandler) Sitemap(gc *gin.Context) {
	ctx := gc.Request.Context()
	xml, err := generateUpcomingEventsSitemap(
		h,
		ctx,
		"https://kulturbytes.de/event/",
	)
	if err != nil {
		gc.String(500, "failed to generate sitemap")
		return
	}
	gc.Header("Content-Type", "application/xml; charset=utf-8")
	gc.String(200, xml)
}

func generateUpcomingEventsSitemap(h *ApiHandler, ctx context.Context, baseUrl string) (string, error) {
	rows, err := h.DbPool.Query(ctx, `
		SELECT event_uuid, start_date::text, start_time::text,
		modified_at::date::text AS lastmod
		FROM uranus.event_date_projection
		WHERE (start_date > CURRENT_DATE)
		   OR (start_date = CURRENT_DATE AND start_time >= CURRENT_TIME)
		ORDER BY start_date ASC, start_time ASC
		LIMIT 50000
	`)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var b strings.Builder
	b.Grow(50_000 * 120) // Rough preallocation

	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)

	for rows.Next() {
		var (
			uuid      string
			startDate string
			startTime string
			lastMod   string
		)

		if err := rows.Scan(&uuid, &startDate, &startTime, &lastMod); err != nil {
			continue
		}

		slug := BuildDateSlug(startDate, startTime)

		b.WriteString(`<url><loc>`)
		b.WriteString(baseUrl)
		b.WriteString(uuid)
		b.WriteString("/")
		b.WriteString(slug)
		b.WriteString(`</loc>`)

		// SEO IMPORTANT: lastmod improves indexing
		b.WriteString(`<lastmod>`)
		b.WriteString(lastMod)
		b.WriteString(`</lastmod>`)

		// best practice for event pages
		b.WriteString(`<changefreq>weekly</changefreq>`)
		b.WriteString(`<priority>0.8</priority></url>`)
	}

	b.WriteString(`</urlset>`)

	return b.String(), nil
}
