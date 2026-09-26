package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sndcds/uranus/app"
)

func TestSocialWorkerPostgresConcurrentClaims(t *testing.T) {
	h, r, token := publishDatabase(t)
	dbExec(t, h, "DELETE FROM uranus.pluto_image")
	entered, release := make(chan struct{}, 1), make(chan struct{})
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls.Add(1)
		entered <- struct{}{}
		select {
		case <-release:
			io.WriteString(w, `{"id":"1"}`)
		case <-req.Context().Done():
		}
	}))
	defer server.Close()
	defer close(release)
	account := publishAccountForTest(t, h, r, token, server)
	post := previewPost(t, r, token, account)
	scheduledTargetForTest(t, h, post)
	other := *h
	start := make(chan struct{})
	type outcome struct {
		count int
		err   error
	}
	done := make(chan outcome, 2)
	for _, worker := range []*ApiHandler{h, &other} {
		go func() {
			<-start
			count, err := worker.ProcessDueSocialTargets(context.Background(), 20)
			done <- outcome{count, err}
		}()
	}
	close(start)
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not reach remote")
	}
	select {
	case result := <-done:
		if result.count != 0 || result.err != nil {
			t.Fatal("second worker claimed the same target")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("second worker waited for remote publishing")
	}
	path := socialPostsPath + "/" + post.Uuid
	assertSocialStatus(t, socialRequest(r, "POST", path+"/cancel", token, ""), 409)
	assertSocialStatus(t, socialRequest(r, "POST", path+"/schedule", token, socialScheduleBody(time.Now().Add(time.Hour))), 409)
	publishData(t, socialRequest(r, "POST", path+"/publish", token, ""), 409)
	release <- struct{}{}
	select {
	case result := <-done:
		if result.count != 1 || result.err != nil {
			t.Fatal("first worker did not complete")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not finish")
	}
	history := publicationList(t, socialRequest(r, "GET", socialPublicationsPath, token, ""))
	if calls.Load() != 1 || len(history) != 1 || history[0].Status != "published" {
		t.Fatal("two workers did not produce exactly one remote post and attempt")
	}
}

func TestSocialWorkerPostgresCancelAndRescheduleRace(t *testing.T) {
	for _, action := range []string{"cancel", "schedule"} {
		t.Run(action, func(t *testing.T) {
			h, r, token := publishDatabase(t)
			dbExec(t, h, "DELETE FROM uranus.pluto_image")
			var calls atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)
				io.WriteString(w, `{"id":"1"}`)
			}))
			defer server.Close()
			account := publishAccountForTest(t, h, r, token, server)
			for range 10 {
				post := previewPost(t, r, token, account)
				scheduledTargetForTest(t, h, post)
				path := socialPostsPath + "/" + post.Uuid
				before := calls.Load()
				start := make(chan struct{})
				response := make(chan *httptest.ResponseRecorder, 1)
				errCh := make(chan error, 1)
				go func() {
					<-start
					response <- socialRequest(r, "POST", path+"/"+action, token, socialScheduleBody(time.Now().Add(time.Hour)))
				}()
				go func() {
					<-start
					_, err := h.ProcessDueSocialTargets(context.Background(), 20)
					errCh <- err
				}()
				close(start)
				var w *httptest.ResponseRecorder
				select {
				case w = <-response:
				case <-time.After(5 * time.Second):
					t.Fatal("scheduling action deadlocked with claim")
				}
				select {
				case err := <-errCh:
					if err != nil {
						t.Fatal(err)
					}
				case <-time.After(5 * time.Second):
					t.Fatal("claim deadlocked with scheduling action")
				}
				target := socialPostData(t, socialRequest(r, "GET", path, token, "")).Targets[0]
				delta := calls.Load() - before
				if delta == 0 {
					assertSocialStatus(t, w, 200)
					want := "cancelled"
					if action == "schedule" {
						want = "scheduled"
					}
					if target.Status != want {
						t.Fatal("scheduling won but target was claimed")
					}
				} else if delta == 1 {
					// Rescheduling a completed publication is explicitly allowed.
					if action == "schedule" && w.Code == 200 {
						if target.Status != "scheduled" || !target.ScheduledAt.After(time.Now()) {
							t.Fatal("post-publication schedule has inconsistent state")
						}
					} else {
						assertSocialStatus(t, w, 409)
						if target.Status != "published" {
							t.Fatal("claim won but target was reset")
						}
					}
				} else {
					t.Fatal("race caused duplicate remote calls")
				}
			}
		})
	}
}

func TestSocialWorkerPostgresShutdownAndOnce(t *testing.T) {
	for _, mode := range []string{"idle", "remote", "once"} {
		t.Run(mode, func(t *testing.T) {
			h, r, token := publishDatabase(t)
			config := app.DefaultConfig()
			h.Config = &config
			dbExec(t, h, "DELETE FROM uranus.pluto_image")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			entered := make(chan struct{}, 1)
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				io.Copy(io.Discard, req.Body)
				entered <- struct{}{}
				if mode == "remote" {
					<-req.Context().Done()
					return
				}
				io.WriteString(w, `{"id":"1"}`)
			}))
			defer server.Close()
			account := publishAccountForTest(t, h, r, token, server)
			if mode != "idle" {
				post := previewPost(t, r, token, account)
				scheduledTargetForTest(t, h, post)
			}
			done := make(chan error, 1)
			go func() { done <- h.RunSocialWorker(ctx, mode == "once") }()
			if mode == "idle" {
				// Let the initial empty poll complete and enter the interval wait.
				time.AfterFunc(100*time.Millisecond, cancel)
			} else {
				select {
				case <-entered:
				case <-time.After(5 * time.Second):
					cancel()
					t.Fatal("worker delayed the initial poll")
				}
				if mode == "remote" {
					cancel()
				}
			}
			select {
			case err := <-done:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(5 * time.Second):
				cancel()
				t.Fatal("worker did not stop promptly")
			}
			if mode == "remote" {
				history := publicationList(t, socialRequest(r, "GET", socialPublicationsPath, token, ""))
				if len(history) != 1 || history[0].Status != "uncertain" {
					t.Fatal("interrupted remote publish was not retained for reconciliation")
				}
				processSocialTargetsForTest(t, h, 20, 0)
			}
		})
	}
}
