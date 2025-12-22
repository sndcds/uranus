package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type GenreLookup map[int]string // genre_id -> name
type TypeLookup struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type LanguageLookup struct {
	Types  map[int]TypeLookup  `json:"types"`
	Genres map[int]GenreLookup `json:"genres"`
}

func (h *ApiHandler) GetEventTypesAndGenres(gc *gin.Context) {
	ctx := gc.Request.Context()

	// Get comma-separated languages from URL, e.g. "de,da,en"
	langsParam := gc.Query("lang")
	if langsParam == "" {
		langsParam = "de,da,en" // default fallback
	}
	langs := strings.Split(langsParam, ",") // ["de","da","en"]

	query := fmt.Sprintf(`
SELECT
  et.type_id,
  et.name,
  et.iso_639_1,
  gt.type_id,
  gt.name
FROM %s.event_type et
LEFT JOIN %s.genre_type gt
  ON gt.event_type_id = et.type_id
 AND gt.iso_639_1 = et.iso_639_1
WHERE et.iso_639_1 = ANY($1)
ORDER BY et.type_id, gt.type_id;
`, h.DbSchema, h.DbSchema)

	rows, err := h.DbPool.Query(ctx, query, langs)
	if err != nil {
		gc.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	result := map[string]*LanguageLookup{}

	for rows.Next() {
		var (
			typeID    int
			typeName  string
			lang      string
			genreID   *int
			genreName *string
		)

		if err := rows.Scan(&typeID, &typeName, &lang, &genreID, &genreName); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if result[lang] == nil {
			result[lang] = &LanguageLookup{
				Types:  map[int]TypeLookup{},
				Genres: map[int]GenreLookup{},
			}
		}

		result[lang].Types[typeID] = TypeLookup{
			ID:   typeID,
			Name: typeName,
		}

		if genreID != nil && genreName != nil {
			if result[lang].Genres[typeID] == nil {
				result[lang].Genres[typeID] = GenreLookup{}
			}
			result[lang].Genres[typeID][*genreID] = *genreName
		}
	}

	gc.JSON(http.StatusOK, result)
}
