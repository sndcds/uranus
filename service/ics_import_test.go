package service

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestReadICSURL(t *testing.T) {
	feedURL := os.Getenv("ICS_URL")
	if feedURL == "" {
		t.Fatal("ICS_URL environment variable is required")
	}

	events, err := ReadICSURL(
		context.Background(),
		feedURL,
		"euf-zentral",
	)
	if err != nil {
		t.Fatalf("ReadICSURL failed: %v", err)
	}

	t.Logf("Converted %d events\n", len(events))

	for i, event := range events {
		t.Logf(
			"\n[%d] %s\n"+
				"  UID:         %s\n"+
				"  Start:       %s\n"+
				"  End:         %s\n"+
				"  All day:     %t\n"+
				"  Location:    %s\n"+
				"  Categories:  %s\n"+
				"  Status:      %s\n"+
				"  Source URL:  %s\n"+
				"  Image URL:   %s\n"+
				"  Description: %s\n",
			i+1,
			event.Title,
			event.ExternalUID,
			event.StartsAt.Format("2006-01-02 15:04 MST"),
			event.EndsAt.Format("2006-01-02 15:04 MST"),
			event.AllDay,
			event.LocationText,
			strings.Join(event.Categories, ", "),
			event.Status,
			event.SourceURL,
			event.ImageURL,
			event.Description,
		)
	}
}
