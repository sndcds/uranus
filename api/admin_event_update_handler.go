package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
	"net/http"
	"strconv"
	"strings"
)

func AdminPostEventHandler(gc *gin.Context) {
	pool := app.Singleton.MainDbPool

	fmt.Printf("AdminPostEventHandler ...")

	type UpdateEventRequest struct {
		EventIDStr        *string   `json:"event_id"`
		OrganizerIDStr    *string   `json:"organizer_id"`
		SpaceIDStr        *string   `json:"space_id"`
		Title             *string   `json:"title"`
		Subtitle          *string   `json:"subtitle"`
		Description       *string   `json:"description"`
		TeaserText        *string   `json:"teaser_text"`
		SourceURL         *string   `json:"source_url"`
		ParticipationInfo *string   `json:"participation_info"`
		MinAgeStr         *string   `json:"min_age"`
		MaxAgeStr         *string   `json:"max_age"`
		EventTypeIDs      *[]string `json:"event_type_ids"`
		GenreTypeIDs      *[]string `json:"genre_type_ids"`
	}

	var req UpdateEventRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	/*
		{
			b, err := json.MarshalIndent(req, "", "  ")
			if err != nil {
				fmt.Println("Error marshaling:", err)
				return http.StatusBadRequest, fmt.Errorf("MarshalIndent error")
			}
			fmt.Println(string(b))
			return http.StatusOK, nil
		}
	*/

	// Validate

	eventID, ok := StringToIntWithDefault(req.EventIDStr, -1)
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"message": "invalid event-id"})
		return
	}

	organizerID, ok := StringToIntWithDefault(req.OrganizerIDStr, -1)
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"message": "invalid organizer-id"})
		return
	}

	spaceID, ok := StringToIntWithDefault(req.SpaceIDStr, -1)
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"message": "invalid space-id"})
		return
	}

	minAge, ok := StringToIntWithDefault(req.MinAgeStr, -1)
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"message": "invalid min-age"})
	}

	maxAge, ok := StringToIntWithDefault(req.MaxAgeStr, -1)
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"message": "invalid max-age"})
	}

	var eventTypeIDsInt []int
	var genreTypeIDsInt []int

	if req.EventTypeIDs != nil {
		for _, s := range *req.EventTypeIDs {
			i, err := strconv.Atoi(s)
			if err != nil {
				gc.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("invalid event-type id: %v", s)})
				return
			}
			eventTypeIDsInt = append(eventTypeIDsInt, i)
		}
	}

	if req.GenreTypeIDs != nil {
		for _, s := range *req.GenreTypeIDs {
			i, err := strconv.Atoi(s)
			if err != nil {
				gc.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("invalid genre-type id: %v", s)})
				return
			}
			genreTypeIDsInt = append(genreTypeIDsInt, i)
		}
	}

	// Build sql query

	var sqlBuilder strings.Builder
	var args []interface{}
	argPos := 1 // placeholder counter

	sqlBuilder.WriteString(fmt.Sprintf("UPDATE %s.event SET ", app.Singleton.Config.DbSchema))

	if spaceID >= 0 {
		sqlBuilder.WriteString(fmt.Sprintf("space_id = $%d, ", argPos))
		args = append(args, spaceID)
		argPos++
	}
	if req.Title != nil {
		sqlBuilder.WriteString(fmt.Sprintf("title = $%d, ", argPos))
		args = append(args, *req.Title)
		argPos++
	}
	if req.Subtitle != nil {
		sqlBuilder.WriteString(fmt.Sprintf("subtitle = $%d, ", argPos))
		args = append(args, *req.Subtitle)
		argPos++
	}
	if req.Description != nil {
		sqlBuilder.WriteString(fmt.Sprintf("description = $%d, ", argPos))
		args = append(args, *req.Description)
		argPos++
	}
	if req.TeaserText != nil {
		sqlBuilder.WriteString(fmt.Sprintf("teaser_text = $%d, ", argPos))
		args = append(args, *req.TeaserText)
		argPos++
	}
	if req.SourceURL != nil {
		sqlBuilder.WriteString(fmt.Sprintf("source_url = $%d, ", argPos))
		args = append(args, *req.SourceURL)
		argPos++
	}
	if req.ParticipationInfo != nil {
		sqlBuilder.WriteString(fmt.Sprintf("participation_info = $%d, ", argPos))
		args = append(args, *req.ParticipationInfo)
		argPos++
	}
	if minAge >= 0 {
		sqlBuilder.WriteString(fmt.Sprintf("min_age = $%d, ", argPos))
		args = append(args, minAge)
		argPos++
	}
	if maxAge >= 0 {
		sqlBuilder.WriteString(fmt.Sprintf("max_age = $%d, ", argPos))
		args = append(args, maxAge)
		argPos++
	}
	if organizerID >= 0 {
		sqlBuilder.WriteString(fmt.Sprintf("organizer_id = $%d, ", argPos))
		args = append(args, organizerID)
		argPos++
	}

	fmt.Println("organizerID:", organizerID)
	fmt.Println("eventID:", eventID)
	fmt.Println("minAge:", minAge)
	fmt.Println("maxAge:", maxAge)

	// Remove trailing comma and space
	sql := strings.TrimSuffix(sqlBuilder.String(), ", ")

	// Add WHERE clause, assume eventID is last parameter
	sql += fmt.Sprintf(" WHERE id = $%d", argPos)
	args = append(args, eventID)

	fmt.Println("argPos:", argPos)
	fmt.Println("sql:", sql)

	// Transaction
	tx, err := pool.Begin(gc)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	defer tx.Rollback(gc)

	// Basic event data
	_, err = tx.Exec(gc, sql, args...)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	// Event types
	{
		sql = "DELETE FROM uranus.event_type_links WHERE event_id = $1"
		_, err := tx.Exec(gc, sql, eventID)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		queryTemplate := `INSERT INTO {{schema}}.event_type_links (event_id, type_id) VALUES ($1, $2)`
		query := strings.Replace(queryTemplate, "{{schema}}", app.Singleton.Config.DbSchema, 1)

		for _, typeId := range eventTypeIDsInt {
			// fmt.Println("eventId", eventId, "typeId:", typeId)
			_, err := tx.Exec(gc, query, eventID, typeId)
			if err != nil {
				gc.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
		}
	}

	// Genres
	{
		sql = "DELETE FROM uranus.event_genre_links WHERE event_id = $1"
		_, err := tx.Exec(gc, sql, eventID)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		queryTemplate := `INSERT INTO {{schema}}.event_genre_links (event_id, type_id) VALUES ($1, $2)`
		query := strings.Replace(queryTemplate, "{{schema}}", app.Singleton.Config.DbSchema, 1)
		// fmt.Println("query:", query)
		for _, genreId := range genreTypeIDsInt {
			// fmt.Println("eventId", eventId, "genreId:", genreId)
			_, err := tx.Exec(gc, query, eventID, genreId)
			if err != nil {
				gc.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
		}
	}

	err = tx.Commit(gc)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"message": "OK"})
}
