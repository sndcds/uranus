package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/model"
)

// TODO: Code review

func (h *ApiHandler) AdminGetTodos(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	query := fmt.Sprintf(`
    	SELECT id, title, description, due_date, completed
    	FROM %s.todo
    	WHERE user_id = $1
    	ORDER BY due_date IS NULL, due_date DESC, id DESC`,
		h.DbSchema)
	rows, err := h.DbPool.Query(ctx, query, userId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("query failed: %v", err)})
		return
	}
	defer rows.Close()

	var todos []model.Todo

	for rows.Next() {
		var todo model.Todo
		if err := rows.Scan(
			&todo.Id,
			&todo.Title,
			&todo.Description,
			&todo.DueDate,
			&todo.Completed,
		); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("scan failed: %v", err)})
			return
		}
		todos = append(todos, todo)
	}

	if rows.Err() != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("rows error: %v", rows.Err())})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"todos":       todos,
		"total_count": len(todos),
	})
}

func (h *ApiHandler) AdminGetTodo(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	todoId, ok := ParamInt(gc, "todoId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "tidiId is required"})
		return
	}

	query := fmt.Sprintf(`
		SELECT id, title, description, due_date
		FROM %s.todo
		WHERE user_id = $1 AND id = $2`,
		h.Config.DbSchema)

	var todo model.Todo
	err := h.DbPool.QueryRow(ctx, query, userId, todoId).Scan(
		&todo.Id,
		&todo.Title,
		&todo.Description,
		&todo.DueDate,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			gc.JSON(http.StatusNotFound, gin.H{"error": "todo not found"})
		} else {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("query failed: %v", err)})
		}
		return
	}

	gc.JSON(http.StatusOK, gin.H{"todo": todo})
}

func (h *ApiHandler) AdminUpsertTodo(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	type Incoming struct {
		Id          int     `json:"id"`
		Completed   *bool   `json:"completed"`
		Title       *string `json:"title"`
		Description *string `json:"description"`
		DueDate     *string `json:"due_date"`
	}

	var payload Incoming
	if err := gc.ShouldBindJSON(&payload); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse due_date
	var duePtr *time.Time
	if payload.DueDate != nil && *payload.DueDate != "" {
		t, err := time.Parse("2006-01-02", *payload.DueDate)
		if err != nil {
			gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid due_date format (YYYY-MM-DD)"})
			return
		}
		duePtr = &t
	}

	// Insert a new entry
	if payload.Id < 0 {
		query := fmt.Sprintf(`
			INSERT INTO %s.todo
				(user_id, title, description, due_date, completed)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`, h.Config.DbSchema)

		var newId int
		err := h.DbPool.QueryRow(
			ctx,
			query,
			userId,
			payload.Title,
			payload.Description,
			duePtr,
			payload.Completed,
		).Scan(&newId)

		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		gc.JSON(http.StatusOK, gin.H{
			"message": "Todo created successfully",
			"id":      newId,
		})
		return
	}

	// Update an entry
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if payload.Completed != nil {
		setClauses = append(setClauses, fmt.Sprintf("completed = $%d", argIdx))
		args = append(args, *payload.Completed)
		argIdx++
	}

	if payload.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *payload.Title)
		argIdx++
	}

	if payload.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *payload.Description)
		argIdx++
	}

	if payload.DueDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("due_date = $%d", argIdx))
		args = append(args, duePtr)
		argIdx++
	}

	if len(setClauses) == 0 {
		gc.JSON(http.StatusOK, gin.H{"message": "nothing to update"})
		return
	}

	query := fmt.Sprintf(`
		UPDATE %s.todo
		SET %s
		WHERE user_id = $%d AND id = $%d
	`, h.Config.DbSchema, strings.Join(setClauses, ", "), argIdx, argIdx+1)

	args = append(args, userId, payload.Id)

	res, err := h.DbPool.Exec(ctx, query, args...)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if res.RowsAffected() == 0 {
		gc.JSON(http.StatusNotFound, gin.H{"error": "todo not found"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"id":      payload.Id,
		"message": "Todo updated successfully",
	})
}

func (h *ApiHandler) AdminDeleteTodo(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	todoId, ok := ParamInt(gc, "todoId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "tidiId is required"})
		return
	}

	query := fmt.Sprintf(`DELETE FROM %s.todo WHERE user_id = $1 AND id = $2`, h.DbSchema)
	res, err := h.DbPool.Exec(ctx, query, userId, todoId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("delete failed: %v", err)})
		return
	}

	if res.RowsAffected() == 0 {
		gc.JSON(http.StatusNotFound, gin.H{"error": "todo not found"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"message": "Todo deleted successfully",
		"id":      todoId,
	})
}
