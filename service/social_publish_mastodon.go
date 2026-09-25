package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sndcds/uranus/model"
)

const socialImageMaxBytes = 10 << 20
const socialResponseMaxBytes = 1 << 20

type mastodonPublisher struct {
	client       *http.Client
	imageBaseURL string
}

// Mastodon IDs are decimal strings. Never persist arbitrary remote text or
// reflect an echoed credential as an ID (including in a later media GET URL).
var mastodonID = regexp.MustCompile(`^[0-9]{1,64}$`)

type mastodonResponse struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

func (p mastodonPublisher) Publish(ctx context.Context, account SocialPublishingAccount, post model.RenderedPost) (model.SocialPublishResult, error) {
	result := model.SocialPublishResult{}
	if err := ValidateSocialRenderAccount(account.Account, account.Account.OrgUuid); err != nil {
		return result, err
	}
	if account.Account.Platform != "mastodon" || post.Platform != "mastodon" {
		return result, fmt.Errorf("publishing platform mismatch")
	}
	base, err := socialURL(value(account.Account.BaseURL))
	if err != nil || base.Scheme != "https" || base.ForceQuery {
		return result, fmt.Errorf("publishing base_url must be an HTTPS origin")
	}
	if strings.TrimSpace(account.accessToken) == "" || strings.ContainsAny(account.accessToken, "\r\n") {
		return result, fmt.Errorf("valid access_token is required")
	}
	// This overall bound also limits media polling and multipart preparation.
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	form := url.Values{"status": {post.Text}, "visibility": {"public"}}
	if post.ImageURL != "" {
		id, err := p.uploadMedia(ctx, strings.TrimRight(base.String(), "/"), account.accessToken, post)
		if err != nil {
			return result, err
		}
		form.Set("media_ids[]", id)
	}
	response, _, err := p.request(ctx, "POST", strings.TrimRight(base.String(), "/")+"/api/v1/statuses", account.accessToken,
		"application/x-www-form-urlencoded", strings.NewReader(form.Encode()), true)
	if err != nil {
		return result, err
	}
	result.RemotePostID = response.ID
	return result, nil
}

func (p mastodonPublisher) request(ctx context.Context, method, endpoint, token, contentType string, body io.Reader, statusPost bool) (mastodonResponse, int, error) {
	var result mastodonResponse
	fail := func(message string, uncertain bool) (mastodonResponse, int, error) {
		if statusPost && uncertain {
			return mastodonResponse{}, 0, &socialPublishUncertainError{message}
		}
		return mastodonResponse{}, 0, fmt.Errorf("%s", message)
	}
	ctx, cancel := context.WithTimeout(ctx, socialRequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return fail("Mastodon request could not be prepared", false)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return fail("Mastodon request failed or timed out; remote outcome may be unknown", true)
	}
	defer resp.Body.Close()
	if method == "GET" && resp.StatusCode == http.StatusPartialContent {
		return result, resp.StatusCode, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// No raw response body, URL, header or transport error is retained.
		return fail(fmt.Sprintf("Mastodon returned HTTP %d", resp.StatusCode), resp.StatusCode >= 500)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, socialResponseMaxBytes+1))
	if err != nil || len(data) > socialResponseMaxBytes || json.Unmarshal(data, &result) != nil {
		return fail("Mastodon returned an invalid response", true)
	}
	if !mastodonID.MatchString(result.ID) || strings.Contains(result.ID, token) {
		kind := "media"
		if statusPost {
			kind = "status"
		}
		return fail("Mastodon response did not contain a valid "+kind+" id", true)
	}
	return result, resp.StatusCode, nil
}

func (p mastodonPublisher) uploadMedia(ctx context.Context, base, token string, post model.RenderedPost) (string, error) {
	data, contentType, err := p.downloadImage(ctx, post.ImageURL)
	if err != nil {
		return "", err
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	extension := map[string]string{"image/jpeg": "jpg", "image/png": "png", "image/webp": "webp"}[contentType]
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="file"; filename="image.`+extension+`"`)
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return "", fmt.Errorf("Mastodon media upload could not be prepared")
	}
	if _, err = part.Write(data); err != nil {
		return "", fmt.Errorf("Mastodon media upload could not be prepared")
	}
	if post.ImageAlt != nil {
		if err = writer.WriteField("description", *post.ImageAlt); err != nil {
			return "", fmt.Errorf("Mastodon media upload could not be prepared")
		}
	}
	if err = writer.Close(); err != nil {
		return "", fmt.Errorf("Mastodon media upload could not be prepared")
	}
	media, status, err := p.request(ctx, "POST", base+"/api/v2/media", token, writer.FormDataContentType(), &body, false)
	if err != nil {
		return "", err
	}
	id := media.ID
	if status == http.StatusOK && media.URL != "" {
		return id, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	for attempt := 0; attempt < 5; attempt++ {
		timer := time.NewTimer(500 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", fmt.Errorf("Mastodon media processing cancelled")
		case <-timer.C:
		}
		media, status, err = p.request(ctx, "GET", base+"/api/v1/media/"+id, token, "", nil, false)
		if err != nil {
			return "", err
		}
		if status == http.StatusOK && media.ID == id && media.URL != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("Mastodon media processing did not finish")
}

func (p mastodonPublisher) downloadImage(ctx context.Context, raw string) ([]byte, string, error) {
	u, err := socialURL(raw)
	base, baseErr := socialURL(p.imageBaseURL)
	if err != nil || baseErr != nil || u.Scheme != "https" || base.Scheme != "https" ||
		u.Host != base.Host || u.Fragment != "" || u.RawPath != "" || base.RawQuery != "" || base.Fragment != "" {
		return nil, "", fmt.Errorf("image URL must use the configured HTTPS Image API")
	}
	prefix := strings.TrimRight(base.Path, "/") + "/api/image/"
	id, err := uuid.Parse(strings.TrimPrefix(u.Path, prefix))
	if err != nil || id == uuid.Nil || u.Path != prefix+id.String() {
		return nil, "", fmt.Errorf("image URL must contain an Image API UUID")
	}
	// Only the parameters emitted for Mastodon by the renderer are needed.
	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, "", fmt.Errorf("invalid image transformation")
	}
	for key, values := range query {
		if len(values) != 1 || (key != "type" && key != "width" && key != "height") ||
			(key == "type" && values[0] != "jpg") || (key != "type" && values[0] != "1920") {
			return nil, "", fmt.Errorf("invalid image transformation")
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, "", fmt.Errorf("image download could not be prepared")
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("image download failed or timed out")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("image download returned HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > socialImageMaxBytes {
		return nil, "", fmt.Errorf("image exceeds 10 MiB limit")
	}
	contentType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil || (contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp") {
		return nil, "", fmt.Errorf("unsupported image content type")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, socialImageMaxBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("image download failed")
	}
	if len(data) > socialImageMaxBytes {
		return nil, "", fmt.Errorf("image exceeds 10 MiB limit")
	}
	if http.DetectContentType(data) != contentType {
		return nil, "", fmt.Errorf("image content does not match its content type")
	}
	return data, contentType, nil
}
