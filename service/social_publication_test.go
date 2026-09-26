package service

import (
	"testing"

	"github.com/sndcds/uranus/model"
)

func TestSocialContentFingerprint(t *testing.T) {
	alt := "Bühne 🎷"
	post := model.RenderedPost{Platform: "mastodon", Text: "Jazzabend · 19:00 🎶", ImageURL: "https://example.test/image", ImageAlt: &alt}
	fingerprint := SocialContentFingerprint(post)
	if fingerprint != "a310a8a70f450edda32e09ded110ebbc6c6dfd4693b0814deef7e940222c2673" || fingerprint != SocialContentFingerprint(post) {
		t.Fatal("fingerprint is not a deterministic SHA-256 hex string")
	}
	for _, field := range []string{"platform", "text", "image", "alt"} {
		t.Run(field, func(t *testing.T) {
			changed := post
			switch field {
			case "platform":
				changed.Platform = "facebook"
			case "text":
				changed.Text += "!"
			case "image":
				changed.ImageURL += "?width=1920"
			case "alt":
				value := alt + "!"
				changed.ImageAlt = &value
			}
			if fingerprint == SocialContentFingerprint(changed) {
				t.Fatal("payload change did not change fingerprint")
			}
		})
	}
	post.URL = "https://example.test/not-sent-separately"
	if fingerprint != SocialContentFingerprint(post) {
		t.Fatal("metadata URL changed fingerprint")
	}
	post.ImageURL = ""
	fingerprint = SocialContentFingerprint(post)
	post.ImageAlt = nil
	if fingerprint != SocialContentFingerprint(post) {
		t.Fatal("unused alt text changed text-only fingerprint")
	}
	a := model.RenderedPost{Platform: "ab", Text: "c"}
	b := model.RenderedPost{Platform: "a", Text: "bc"}
	if SocialContentFingerprint(a) == SocialContentFingerprint(b) {
		t.Fatal("field boundaries are ambiguous")
	}
}

func TestValidSocialRemotePostID(t *testing.T) {
	for _, tc := range []struct {
		platform, id string
		valid        bool
	}{
		{"mastodon", "123456789", true}, {"mastodon", "", false},
		{"mastodon", " 123", false}, {"mastodon", "secret-access", false},
		{"facebook", "123", false}, {"bluesky", "123", false},
	} {
		if ValidSocialRemotePostID(tc.platform, tc.id) != tc.valid {
			t.Fatal("incorrect remote ID validation")
		}
	}
}
