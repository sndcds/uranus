package api

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestSocialSchedulingPostgresMigration(t *testing.T) {
	h, r, token := publishDatabase(t)
	account := createPostAccount(t, r, token, "facebook", socialOrg)
	post := previewPost(t, r, token, account)
	result := publishAndProcess(t, h, r, token, socialPostsPath+"/"+post.Uuid+"/publish").Results[0]
	publication := publicationData(t, socialRequest(r, "GET", socialPublicationsPath+"/"+result.PublicationUuid, token, ""))
	if publication.PublicationSource != "manual" {
		t.Fatal("manual attempt lost publication source")
	}
	publicationSQLFails(t, h, "UPDATE uranus.social_publication SET publication_source='scheduled' WHERE uuid=$1", "23514", publication.Uuid)
	publicationSQLFails(t, h, `INSERT INTO uranus.social_publication
		(uuid,social_post_uuid,social_post_target_uuid,social_account_uuid,platform,status,publication_source)
		VALUES ($1,$2,$3,$4,'facebook','publishing','invalid')`, "23514", socialOtherUser, post.Uuid, post.Targets[0].Uuid, account)

	down, err := os.ReadFile("../migrations/202609260002_social_scheduling.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	scheduledTargetForTest(t, h, post)
	conn, err := h.DbPool.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, err = conn.Exec(context.Background(), strings.ReplaceAll(string(down), "uranus.", h.DbSchema+"."))
	if err == nil {
		t.Fatal("rollback accepted pending schedules")
	}
	if _, err := conn.Exec(context.Background(), "ROLLBACK"); err != nil {
		t.Fatal(err)
	}
	conn.Release()
	assertSocialStatus(t, socialRequest(r, "POST", socialPostsPath+"/"+post.Uuid+"/cancel", token, ""), 200)
	dbExec(t, h, string(down))
	up, err := os.ReadFile("../migrations/202609260002_social_scheduling.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	dbExec(t, h, string(up))
	publication = publicationData(t, socialRequest(r, "GET", socialPublicationsPath+"/"+publication.Uuid, token, ""))
	if publication.PublicationSource != "manual" || publication.Status != "failed" {
		t.Fatal("migration did not preserve existing attempts")
	}
}
