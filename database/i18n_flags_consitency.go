package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DatabaseFlagsCheckI18nResult holds the counts and any inconsistencies
type DatabaseFlagsCheckI18nResult struct {
	TopicCount      int
	FlagCount       int
	Inconsistencies []string
}

// DatabaseFlagsCheckI18nConsistency checks that all flags have the expected languages
// and that their topic_id exists in the topic table.
// Returns counts of unique topics, flags, and any inconsistencies.
func DatabaseFlagsCheckI18nConsistency(
	ctx context.Context,
	dbPool *pgxpool.Pool,
	flagTable,
	topicTable string,
	languages []string,
) (*DatabaseFlagsCheckI18nResult, error) {

	// Build quoted language array for SQL
	langArray := make([]string, len(languages))
	for i, l := range languages {
		langArray[i] = fmt.Sprintf("'%s'", l)
	}

	query := fmt.Sprintf(`
SELECT
    f.flag,
    f.key,
    COUNT(DISTINCT f.iso_639_1) = %d AS all_languages_present,
    ARRAY(
        SELECT unnest(ARRAY[%s]::text[])
        EXCEPT
        SELECT unnest(array_agg(f.iso_639_1))
    ) AS missing_languages,
    (t.topic_id IS NOT NULL) AS topic_exists,
	t.topic_id AS topic_id
FROM %s f
LEFT JOIN %s t
    ON f.topic_id = t.topic_id
WHERE f.iso_639_1 = ANY(ARRAY[%s]::text[])
GROUP BY f.flag, f.key, t.topic_id
HAVING COUNT(DISTINCT f.iso_639_1) <> %d OR t.topic_id IS NULL
ORDER BY f.flag;
`,
		len(languages),
		strings.Join(langArray, ","),
		flagTable,
		topicTable,
		strings.Join(langArray, ","),
		len(languages),
	)

	rows, err := dbPool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	result := &DatabaseFlagsCheckI18nResult{}
	flagSet := make(map[int]struct{})
	topicSet := make(map[int]struct{})

	for rows.Next() {
		var flag int
		var key string
		var allLangs bool
		var missingLangs []string
		var topicExists bool
		var topicID *int // may be null if topic missing

		if err := rows.Scan(&flag, &key, &allLangs, &missingLangs, &topicExists, &topicID); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Track unique flags and topics
		flagSet[flag] = struct{}{}
		if topicID != nil {
			topicSet[*topicID] = struct{}{}
		}

		if !allLangs || !topicExists {
			var parts []string
			parts = append(parts, fmt.Sprintf("flag %d (%s)", flag, key))
			if !allLangs {
				parts = append(parts, fmt.Sprintf("missing languages: %v", missingLangs))
			}
			if !topicExists {
				parts = append(parts, "topic_id missing in topic table")
			}
			result.Inconsistencies = append(result.Inconsistencies, strings.Join(parts, "; "))
		}
	}

	result.FlagCount = len(flagSet)
	result.TopicCount = len(topicSet)

	// Return error if inconsistencies found
	if len(result.Inconsistencies) > 0 {
		return result, fmt.Errorf("i18n consistency errors detected")
	}

	return result, nil
}
