package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/sndcds/uranus/model"
)

// SocialPublishingAccount is internal input, never response metadata. As with
// the write-only API input, accidental JSON and formatted output are redacted.
type SocialPublishingAccount struct {
	Account     model.SocialAccount
	accessToken string
}

func NewSocialPublishingAccount(account model.SocialAccount, accessToken string) SocialPublishingAccount {
	return SocialPublishingAccount{account, accessToken}
}

func (SocialPublishingAccount) MarshalJSON() ([]byte, error) { return []byte("null"), nil }
func (SocialPublishingAccount) String() string               { return "[REDACTED]" }
func (SocialPublishingAccount) GoString() string             { return "[REDACTED]" }

// SocialPublisher sends the rendered text verbatim. It has no database or renderer.
type SocialPublisher interface {
	Publish(context.Context, SocialPublishingAccount, model.RenderedPost) (model.SocialPublishResult, error)
}

// A supplied client is a trusted dependency (e.g. a local test transport).
// Production uses the public-IP-only transport. Redirects and timeouts are
// enforced even when a caller supplies a client.
func NewSocialPublisher(platform string, client *http.Client, imageBaseURL string) (SocialPublisher, error) {
	switch platform {
	case "mastodon":
		return mastodonPublisher{socialPublishClient(client), imageBaseURL}, nil
	case "facebook", "instagram", "bluesky":
		return unimplementedSocialPublisher{platform}, nil
	default:
		return nil, fmt.Errorf("unsupported platform")
	}
}

type unimplementedSocialPublisher struct{ platform string }

func (p unimplementedSocialPublisher) Publish(context.Context, SocialPublishingAccount, model.RenderedPost) (model.SocialPublishResult, error) {
	return model.SocialPublishResult{}, fmt.Errorf("publishing is not implemented for platform %s", p.platform)
}

type socialPublishUncertainError struct{ message string }

func (e *socialPublishUncertainError) Error() string { return e.message }

// An interrupted status POST or invalid success response may already have
// created a post. Keep its claim blocked until an operator checks remotely.
func SocialPublishUncertain(err error) bool {
	var uncertain *socialPublishUncertainError
	return errors.As(err, &uncertain)
}
