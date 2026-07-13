package service

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AccessibilityLookup struct {
	mu sync.RWMutex

	// lang -> flag id -> label
	labels map[string]map[int]string
}

func NewAccessibilityLookup() *AccessibilityLookup {
	return &AccessibilityLookup{
		labels: make(map[string]map[int]string),
	}
}

func (l *AccessibilityLookup) Load(
	ctx context.Context,
	db *pgxpool.Pool,
	schema string,
) error {

	query := `
		SELECT flag, iso_639_1, name
		FROM ` + schema + `.accessibility_flag
	`

	rows, err := db.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	labels := make(map[string]map[int]string)

	for rows.Next() {
		var (
			flag int
			lang string
			name string
		)

		if err := rows.Scan(
			&flag,
			&lang,
			&name,
		); err != nil {
			return err
		}

		if labels[lang] == nil {
			labels[lang] = make(map[int]string)
		}

		labels[lang][flag] = name
	}

	if err := rows.Err(); err != nil {
		return err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.labels = labels

	return nil
}

func (l *AccessibilityLookup) LabelsForMask(
	mask int64,
	lang string,
) []string {

	l.mu.RLock()
	defer l.mu.RUnlock()

	labels := make([]string, 0)

	for flag := int64(0); flag < 64; flag++ {
		if mask&(1<<flag) == 0 {
			continue
		}
		if label, ok := l.labels[lang][int(flag)]; ok {
			labels = append(labels, label)
		}
	}

	return labels
}
