package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sndcds/uranus/model"
)

const publishImageUUID = "01994126-6680-7000-8000-000000000016"

func publishAccount(base string) SocialPublishingAccount {
	return NewSocialPublishingAccount(model.SocialAccount{Uuid: "account", OrgUuid: "org", Platform: "mastodon",
		RemoteAccountID: "42", BaseURL: &base, Enabled: true}, "secret-access")
}

func TestSocialPublisherText(t *testing.T) {
	text := "Unchanged 👩‍💻\n" + strings.Repeat("x", 510)
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "POST" || r.URL.Path != "/api/v1/statuses" || r.Header.Get("Authorization") != "Bearer secret-access" ||
			r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Error("incorrect status request metadata")
		}
		if err := r.ParseForm(); err != nil || r.Form.Get("status") != text || r.Form.Get("visibility") != "public" || len(r.Form) != 2 {
			t.Error("publisher modified rendered text or form")
		}
		io.WriteString(w, `{"id":"123","url":"https://example.test/@test/123"}`)
	}))
	defer server.Close()
	publisher, err := NewSocialPublisher("mastodon", server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	result, err := publisher.Publish(context.Background(), publishAccount(server.URL), model.RenderedPost{Platform: "mastodon", Text: text})
	if err != nil || result.RemotePostID != "123" || calls.Load() != 1 {
		t.Fatal("text publishing failed")
	}
}

func TestSocialPublisherMedia(t *testing.T) {
	for _, async := range []bool{false, true} {
		t.Run(fmt.Sprint(async), func(t *testing.T) {
			var data bytes.Buffer
			if err := png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
				t.Fatal(err)
			}
			var steps []string
			polls := 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				steps = append(steps, r.Method+" "+r.URL.Path)
				switch r.URL.Path {
				case "/api/image/" + publishImageUUID:
					if r.Header.Get("Authorization") != "" {
						t.Error("credential sent to image endpoint")
					}
					w.Header().Set("Content-Type", "image/png")
					w.Write(data.Bytes())
				case "/api/v2/media":
					if r.Header.Get("Authorization") != "Bearer secret-access" {
						t.Error("missing media authorization")
					}
					if err := r.ParseMultipartForm(socialImageMaxBytes); err != nil {
						t.Error("invalid multipart upload")
						return
					}
					defer r.MultipartForm.RemoveAll()
					file, header, err := r.FormFile("file")
					if err != nil {
						t.Error("missing upload file")
						return
					}
					defer file.Close()
					got, _ := io.ReadAll(file)
					if !bytes.Equal(got, data.Bytes()) || header.Header.Get("Content-Type") != "image/png" || r.FormValue("description") != "Original alt" {
						t.Error("media bytes, MIME or alt text changed")
					}
					if async {
						w.WriteHeader(202)
						io.WriteString(w, `{"id":"77","url":null}`)
					} else {
						io.WriteString(w, `{"id":"77","url":"https://media.test/77"}`)
					}
				case "/api/v1/media/77":
					polls++
					if polls == 1 {
						w.WriteHeader(206)
					} else {
						io.WriteString(w, `{"id":"77","url":"https://media.test/77"}`)
					}
				case "/api/v1/statuses":
					if r.ParseForm() != nil || r.Form.Get("media_ids[]") != "77" || r.Form.Get("status") != "Exact text" {
						t.Error("incorrect media status")
					}
					io.WriteString(w, `{"id":"123"}`)
				default:
					t.Error("unexpected network request")
					w.WriteHeader(404)
				}
			}))
			defer server.Close()
			publisher, _ := NewSocialPublisher("mastodon", server.Client(), server.URL)
			post := model.RenderedPost{Platform: "mastodon", Text: "Exact text", ImageURL: server.URL + "/api/image/" + publishImageUUID + "?type=jpg&width=1920", ImageAlt: renderPtr("Original alt")}
			result, err := publisher.Publish(context.Background(), publishAccount(server.URL), post)
			if err != nil || result.RemotePostID != "123" {
				t.Fatal("media publishing failed")
			}
			if len(steps) < 3 || steps[0] != "GET /api/image/"+publishImageUUID || steps[1] != "POST /api/v2/media" || steps[len(steps)-1] != "POST /api/v1/statuses" {
				t.Fatal("incorrect media request order")
			}
		})
	}
}

func TestSocialPublisherRemoteErrors(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    int
		body      string
		uncertain bool
	}{
		{"unauthorized", 401, "secret-access", false}, {"forbidden", 403, "secret-access", false},
		{"missing", 404, "secret-access", false}, {"unprocessable", 422, "secret-access", false},
		{"rate limit", 429, "secret-access", false}, {"server error", 500, "secret-access", true},
		{"invalid JSON", 200, "<html>secret-access</html>", true}, {"missing id", 200, `{}`, true},
		{"echoed id", 200, `{"id":"secret-access"}`, true}, {"empty id", 200, `{"id":""}`, true},
		{"oversized", 200, strings.Repeat("x", socialResponseMaxBytes+1), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); io.WriteString(w, tc.body) }))
			defer server.Close()
			publisher, _ := NewSocialPublisher("mastodon", server.Client(), server.URL)
			result, err := publisher.Publish(context.Background(), publishAccount(server.URL), model.RenderedPost{Platform: "mastodon", Text: "text"})
			if err == nil || strings.Contains(err.Error(), "secret-access") || len(err.Error()) > 200 || result.RemotePostID != "" || SocialPublishUncertain(err) != tc.uncertain {
				t.Fatal("unsafe or incorrect remote error")
			}
		})
	}
}

type socialTestTransport func(*http.Request) (*http.Response, error)

func (f socialTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSocialPublisherCancellationAndConnectionFailure(t *testing.T) {
	for _, timeout := range []bool{false, true} {
		client := &http.Client{Timeout: 20 * time.Millisecond, Transport: socialTestTransport(func(r *http.Request) (*http.Response, error) {
			if timeout {
				<-r.Context().Done()
			}
			return nil, fmt.Errorf("connection error containing secret-access")
		})}
		publisher, _ := NewSocialPublisher("mastodon", client, "https://image.test")
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		_, err := publisher.Publish(ctx, publishAccount("https://social.test"), model.RenderedPost{Platform: "mastodon", Text: "text"})
		cancel()
		if err == nil || strings.Contains(err.Error(), "secret-access") || !SocialPublishUncertain(err) {
			t.Fatal("unsafe connection error")
		}
	}
}

func TestSocialPublisherRejectsRedirects(t *testing.T) {
	var redirected atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirected" {
			redirected.Add(1)
		}
		w.Header().Set("Location", "/redirected")
		w.WriteHeader(307)
	}))
	defer server.Close()
	publisher, _ := NewSocialPublisher("mastodon", server.Client(), server.URL)
	for _, media := range []bool{false, true} {
		post := model.RenderedPost{Platform: "mastodon", Text: "text"}
		if media {
			post.ImageURL = server.URL + "/api/image/" + publishImageUUID
		}
		if _, err := publisher.Publish(context.Background(), publishAccount(server.URL), post); err == nil {
			t.Fatal("accepted redirect")
		}
	}
	if redirected.Load() != 0 {
		t.Fatal("followed publishing redirect")
	}
}

func TestSocialPublisherImageValidation(t *testing.T) {
	for _, tc := range []struct {
		name, contentType, body string
		status                  int
	}{
		{"status", "image/png", "", 404}, {"HTML", "text/html", "<html>secret-access</html>", 200},
		{"false MIME", "image/png", "<html>fake</html>", 200}, {"empty", "image/jpeg", "", 200},
		{"large", "image/png", strings.Repeat("x", socialImageMaxBytes+1), 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var uploads atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					uploads.Add(1)
				}
				w.Header().Set("Content-Type", tc.contentType)
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.body)
			}))
			defer server.Close()
			publisher, _ := NewSocialPublisher("mastodon", server.Client(), server.URL)
			_, err := publisher.Publish(context.Background(), publishAccount(server.URL), model.RenderedPost{Platform: "mastodon", ImageURL: server.URL + "/api/image/" + publishImageUUID})
			if err == nil || uploads.Load() != 0 || strings.Contains(err.Error(), "secret-access") {
				t.Fatal("image validation failed")
			}
		})
	}
}

func TestSocialPublisherSSRF(t *testing.T) {
	for _, raw := range []string{"127.0.0.1", "10.0.0.1", "172.16.0.1", "192.168.0.1", "169.254.169.254", "100.64.0.1", "0.0.0.0", "224.0.0.1", "::1", "::ffff:127.0.0.1", "fc00::1", "fe80::1", "64:ff9b::a00:1", "2002:a00:1::", "2001:db8::1", "ff02::1"} {
		if socialPublicIP(netip.MustParseAddr(raw)) {
			t.Errorf("accepted non-public IP %s", raw)
		}
	}
	for _, raw := range []string{"8.8.8.8", "2606:4700:4700::1111"} {
		if !socialPublicIP(netip.MustParseAddr(raw)) {
			t.Error("rejected public IP")
		}
	}
	if socialPublishTransport.Proxy != nil {
		t.Fatal("environment proxy bypasses IP binding")
	}
	var calls atomic.Int32
	client := &http.Client{Transport: socialTestTransport(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, fmt.Errorf("forbidden network") })}
	for _, raw := range []string{"http://image.test/api/image/" + publishImageUUID, "https://user:secret-access@image.test/api/image/" + publishImageUUID,
		"https://localhost/api/image/" + publishImageUUID, "https://127.0.0.1/api/image/" + publishImageUUID, "https://image.test/other/" + publishImageUUID,
		"https://image.test/api/image/not-a-uuid", "https://image.test/api/image/" + publishImageUUID + "?url=http://localhost", "https://image.test/api/image/" + publishImageUUID + "#fragment"} {
		p := mastodonPublisher{socialPublishClient(client), "https://image.test"}
		if _, _, err := p.downloadImage(context.Background(), raw); err == nil {
			t.Fatal("accepted unsafe image URL")
		}
	}
	if calls.Load() != 0 {
		t.Fatal("unsafe image URL reached transport")
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("production transport reached loopback") }))
	defer server.Close()
	// No injected transport: the normal publisher blocks even an allowlisted
	// origin when its hostname/IP resolves to a local destination.
	p := mastodonPublisher{socialPublishClient(nil), server.URL}
	if _, _, err := p.downloadImage(context.Background(), server.URL+"/api/image/"+publishImageUUID); err == nil {
		t.Fatal("default transport accepted loopback")
	}
}

func TestSocialPublisherDNSBinding(t *testing.T) {
	var addresses []string
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		addresses = append(addresses, address)
		return nil, fmt.Errorf("mock connection failure")
	}
	public := netip.MustParseAddr("8.8.8.8")
	private := netip.MustParseAddr("10.0.0.1")
	// Reject the complete resolution before dialing, even if the first answer
	// is public. Mixed DNS answers cannot steer a later attempt to a private IP.
	dialSocialAddresses(context.Background(), "tcp", "443", []netip.Addr{public, private}, dial)
	if len(addresses) != 0 {
		t.Fatal("mixed DNS answer reached dialer")
	}
	dialSocialAddresses(context.Background(), "tcp", "443", []netip.Addr{public}, dial)
	if len(addresses) != 1 || addresses[0] != "8.8.8.8:443" {
		t.Fatal("dialer did not pin the validated numeric address")
	}
	// The production caller supplies no hostname here, so a subsequent DNS
	// change cannot affect the peer chosen by the dialer.
}

func TestSocialPublisherMediaFailuresAndPollingBound(t *testing.T) {
	for _, failure := range []string{"upload", "missing id", "redirect", "poll", "pending", "cancel"} {
		t.Run(failure, func(t *testing.T) {
			var statusCalls, polls atomic.Int32
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/image/" + publishImageUUID:
					w.Header().Set("Content-Type", "image/png")
					png.Encode(w, image.NewRGBA(image.Rect(0, 0, 2, 2)))
				case "/api/v2/media":
					switch failure {
					case "upload":
						w.WriteHeader(401)
						io.WriteString(w, "secret-access")
					case "missing id":
						io.WriteString(w, `{}`)
					case "redirect":
						w.Header().Set("Location", "/api/v1/statuses")
						w.WriteHeader(307)
					default:
						w.WriteHeader(202)
						io.WriteString(w, `{"id":"77"}`)
						if failure == "cancel" {
							cancel()
						}
					}
				case "/api/v1/media/77":
					polls.Add(1)
					if failure == "poll" {
						w.WriteHeader(500)
					} else {
						w.WriteHeader(206)
					}
				default:
					statusCalls.Add(1)
				}
			}))
			defer server.Close()
			publisher, _ := NewSocialPublisher("mastodon", server.Client(), server.URL)
			_, err := publisher.Publish(ctx, publishAccount(server.URL), model.RenderedPost{Platform: "mastodon", ImageURL: server.URL + "/api/image/" + publishImageUUID})
			if err == nil || statusCalls.Load() != 0 || SocialPublishUncertain(err) || strings.Contains(err.Error(), "secret-access") {
				t.Fatal("failed media produced a status or unsafe error")
			}
			if failure == "pending" && polls.Load() != 5 {
				t.Fatal("media processing was not bounded to five polls")
			}
		})
	}
}

func TestSocialPublisherFactoryCredentialsAndAccountValidation(t *testing.T) {
	if _, err := NewSocialPublisher("unknown", nil, ""); err == nil || err.Error() != "unsupported platform" {
		t.Fatal("unknown platform accepted")
	}
	for _, platform := range []string{"facebook", "instagram", "bluesky"} {
		publisher, err := NewSocialPublisher(platform, nil, "")
		if err != nil {
			t.Fatal(err)
		}
		result, err := publisher.Publish(context.Background(), SocialPublishingAccount{}, model.RenderedPost{})
		if err == nil || err.Error() != "publishing is not implemented for platform "+platform || result.RemotePostID != "" {
			t.Fatal("unimplemented publisher succeeded")
		}
	}
	account := publishAccount("https://social.test")
	encoded, _ := json.Marshal(account)
	for _, text := range []string{string(encoded), fmt.Sprint(account), fmt.Sprintf("%+v", account), fmt.Sprintf("%#v", account)} {
		if strings.Contains(text, "secret-access") {
			t.Fatal("publishing credentials were exposed")
		}
	}
	for _, change := range []func(*SocialPublishingAccount){
		func(a *SocialPublishingAccount) { a.accessToken = "" }, func(a *SocialPublishingAccount) { a.Account.Enabled = false },
		func(a *SocialPublishingAccount) { a.Account.BaseURL = nil }, func(a *SocialPublishingAccount) { a.Account.BaseURL = renderPtr("http://social.test") },
		func(a *SocialPublishingAccount) { a.Account.BaseURL = renderPtr("https://social.test/path") }, func(a *SocialPublishingAccount) { a.Account.BaseURL = renderPtr("https://social.test?x=1") },
		func(a *SocialPublishingAccount) {
			a.Account.BaseURL = renderPtr("https://user:secret-access@social.test")
		}, func(a *SocialPublishingAccount) { a.Account.RemoteAccountID = "" },
	} {
		invalid := account
		change(&invalid)
		publisher, _ := NewSocialPublisher("mastodon", &http.Client{Transport: socialTestTransport(func(*http.Request) (*http.Response, error) {
			t.Error("invalid account reached transport")
			return nil, fmt.Errorf("forbidden")
		})}, "")
		if _, err := publisher.Publish(context.Background(), invalid, model.RenderedPost{Platform: "mastodon"}); err == nil {
			t.Fatal("accepted invalid publishing account")
		}
	}
}
