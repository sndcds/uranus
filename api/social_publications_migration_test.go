package api

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sndcds/uranus/model"
)

func publicationSQLFails(t *testing.T, h *ApiHandler, query, code string, args ...any) {
	t.Helper()
	_, err := h.DbPool.Exec(context.Background(), strings.ReplaceAll(query, "uranus.", h.DbSchema+"."), args...)
	pgErr, ok := err.(*pgconn.PgError)
	if !ok || (pgErr.Code != code && !(code == "23503" && pgErr.Code == "23001")) {
		t.Fatalf("expected SQLSTATE %s, got %v", code, err)
	}
}

func publicationRollbackFails(t *testing.T, h *ApiHandler) {
	t.Helper()
	sql, err := os.ReadFile("../migrations/202609260001_social_publication.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := h.DbPool.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	_, err = conn.Exec(context.Background(), strings.ReplaceAll(string(sql), "uranus.", h.DbSchema+"."))
	if err == nil {
		t.Fatal("rollback removed unresolved history")
	}
	if _, err := conn.Exec(context.Background(), "ROLLBACK"); err != nil {
		t.Fatal(err)
	}
}

func TestSocialPublicationsPostgresMigrationLegacyAndRollback(t *testing.T) {
	// The harness applies Parts 1 and 2 before Part 5 and Part 6.
	h, r, token := contentDatabase(t)
	applySocialPublishMigration(t, h, "up")
	account := createPostAccount(t, r, token, "mastodon", socialOrg)
	post := previewPost(t, r, token, account)
	dbExec(t, h, "UPDATE uranus.social_post_target SET status='publishing' WHERE uuid=$1", post.Targets[0].Uuid)
	before := socialPostData(t, socialRequest(r, "GET", socialPostsPath+"/"+post.Uuid, token, ""))
	applySocialPublicationMigration(t, h, "up")
	after := socialPostData(t, socialRequest(r, "GET", socialPostsPath+"/"+post.Uuid, token, ""))
	if !reflect.DeepEqual(before, after) {
		t.Fatal("migration changed target state")
	}
	history := publicationList(t, socialRequest(r, "GET", socialPublicationsPath, token, ""))
	if len(history) != 1 || history[0].Status != "uncertain" || history[0].ContentFingerprint != nil || history[0].RenderedText != nil || history[0].RemotePostID != nil ||
		history[0].Error == nil || !strings.Contains(*history[0].Error, "legacy Part 5") || !history[0].StartedAt.Equal(before.Targets[0].UpdatedAt) {
		t.Fatal("migration invented a legacy snapshot or lost unresolved target")
	}
	publicationRollbackFails(t, h)
	path := socialPublicationsPath + "/" + history[0].Uuid
	publicationData(t, socialRequest(r, "POST", path+"/reconcile", token, `{"outcome":"failed"}`))
	repaired := socialPostData(t, socialRequest(r, "GET", socialPostsPath+"/"+post.Uuid, token, ""))
	if repaired.Targets[0].Status != "failed" {
		t.Fatal("legacy target was not repaired")
	}
	// Deliberate rollback destroys resolved history but preserves current state.
	applySocialPublicationMigration(t, h, "down")
	applySocialPublicationMigration(t, h, "up")
	if !reflect.DeepEqual(repaired, socialPostData(t, socialRequest(r, "GET", socialPostsPath+"/"+post.Uuid, token, ""))) ||
		len(publicationList(t, socialRequest(r, "GET", socialPublicationsPath, token, ""))) != 0 {
		t.Fatal("migration roundtrip changed target state or invented attempts")
	}
	// Exercise persisted publishing rows and rollback protection, too.
	dbExec(t, h, "UPDATE uranus.social_account SET base_url='https://social.test' WHERE uuid=$1", account)
	gc := contentContext(authTestUser)
	result := model.SocialPostPublish{PostUuid: post.Uuid}
	if _, txErr := h.claimSocialPublish(gc, post.Uuid, &result); txErr != nil {
		t.Fatal(txErr)
	}
	publicationRollbackFails(t, h)
	publicationData(t, socialRequest(r, "POST", socialPublicationsPath+"/"+result.Results[0].PublicationUuid+"/reconcile", token,
		`{"outcome":"failed","confirm_inactive":true}`))
	applySocialPublicationMigration(t, h, "down")
	// Canonical DDL must install the same constraints and indexes.
	ddl, err := os.ReadFile("../ddl/social_publication.ddl")
	if err != nil {
		t.Fatal(err)
	}
	dbExec(t, h, string(ddl))
	var indexes int
	if err := h.DbPool.QueryRow(context.Background(), "SELECT count(*) FROM pg_indexes WHERE schemaname=$1 AND tablename='social_publication'", h.DbSchema).Scan(&indexes); err != nil || indexes != 8 {
		t.Fatalf("unexpected canonical publication indexes: %d, %v", indexes, err)
	}
}

func TestSocialPublicationsPostgresConstraints(t *testing.T) {
	h, r, token := publishDatabase(t)
	account := createPostAccount(t, r, token, "facebook", socialOrg)
	post := previewPost(t, r, token, account)
	result := publishData(t, socialRequest(r, "POST", socialPostsPath+"/"+post.Uuid+"/publish", token, ""), 207).Results[0]
	for _, query := range []string{
		"DELETE FROM uranus.social_post WHERE uuid=$1",
		"DELETE FROM uranus.social_post_target WHERE social_post_uuid=$1",
	} {
		publicationSQLFails(t, h, query, "23503", post.Uuid)
	}
	publicationSQLFails(t, h, "DELETE FROM uranus.organization WHERE uuid=$1", "23503", socialOrg)
	publicationSQLFails(t, h, "DELETE FROM uranus.social_account WHERE uuid=$1", "23503", account)
	for _, query := range []string{
		"DELETE FROM uranus.social_publication WHERE uuid=$1",
		"UPDATE uranus.social_publication SET rendered_text='rewritten' WHERE uuid=$1",
		"UPDATE uranus.social_publication SET status='published',remote_post_id='123' WHERE uuid=$1",
	} {
		publicationSQLFails(t, h, query, "23514", result.PublicationUuid)
	}
	insert := `INSERT INTO uranus.social_publication
		(uuid,social_post_uuid,social_post_target_uuid,social_account_uuid,platform,status,
		content_fingerprint,rendered_text,rendered_image_url,finished_at,remote_post_id)
		VALUES ($1,$2,$3,$4,'facebook',$5,$6,'text','',CURRENT_TIMESTAMP,$7)`
	for _, tc := range []struct{ status, fingerprint, remote, code string }{
		{"invalid", strings.Repeat("a", 64), "", "23514"},
		{"published", "bad", "123", "23514"},
		{"published", strings.Repeat("a", 64), "", "23514"},
		{"failed", strings.Repeat("a", 64), "123", "23514"},
		{"publishing", strings.Repeat("a", 64), "", "23514"},
	} {
		publicationSQLFails(t, h, insert, tc.code, socialOtherUser, post.Uuid, post.Targets[0].Uuid, account, tc.status, tc.fingerprint, tc.remote)
	}
	publicationSQLFails(t, h, insert, "23503", socialOtherUser, socialOtherOrg, post.Targets[0].Uuid, account, "published", strings.Repeat("b", 64), "123")
	dbExec(t, h, insert, socialOtherUser, post.Uuid, post.Targets[0].Uuid, account, "published", strings.Repeat("b", 64), "123")
	publicationSQLFails(t, h, insert, "23505", socialOtherOrg, post.Uuid, post.Targets[0].Uuid, account, "published", strings.Repeat("b", 64), "124")
	unresolved := `INSERT INTO uranus.social_publication (uuid,social_post_uuid,social_post_target_uuid,social_account_uuid,platform,status)
		VALUES ($1,$2,$3,$4,'facebook','publishing')`
	dbExec(t, h, unresolved, socialOtherOrg, post.Uuid, post.Targets[0].Uuid, account)
	publicationSQLFails(t, h, unresolved, "23505", contentOtherVenue, post.Uuid, post.Targets[0].Uuid, account)
	publicationSQLFails(t, h, "UPDATE uranus.social_publication SET rendered_text='changed' WHERE uuid=$1", "23514", socialOtherOrg)
	publicationSQLFails(t, h, "UPDATE uranus.social_publication SET status='failed',finished_at=CURRENT_TIMESTAMP,error=$2 WHERE uuid=$1", "23514", socialOtherOrg, strings.Repeat("x", 1025))
}
