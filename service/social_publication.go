package service

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"

	"github.com/sndcds/uranus/model"
)

// SocialContentFingerprint hashes length-prefixed UTF-8 fields in payload order.
// URL is already in Text; alt text without an image is not sent to the platform.
func SocialContentFingerprint(post model.RenderedPost) string {
	alt := ""
	if post.ImageURL != "" && post.ImageAlt != nil {
		alt = *post.ImageAlt
	}
	hash := sha256.New()
	for _, field := range []string{post.Platform, post.Text, post.ImageURL, alt} {
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(len(field)))
		hash.Write(length[:])
		hash.Write([]byte(field))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func ValidSocialRemotePostID(platform, id string) bool {
	// Only Mastodon currently publishes. Other platforms need their own ID
	// validation when their publisher is implemented.
	return platform == "mastodon" && mastodonID.MatchString(id)
}
