package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) AdminGetTodos(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := gc.GetInt("user-id")

	query := fmt.Sprintf(`
    	SELECT id, title, description, due_date, completed
    	FROM %s.todo
    	WHERE user_id = $1
    	ORDER BY due_date IS NULL, due_date ASC, id ASC`,
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
		"todos": todos,
	})
}

func (h *ApiHandler) AdminGetTodo(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool

	userId := gc.GetInt("user-id")

	todoIdStr := gc.Param("todoId")
	if todoIdStr == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "todoId is required"})
		return
	}

	todoId, err := strconv.Atoi(todoIdStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "todoId must be a number"})
		return
	}

	query := fmt.Sprintf(`
		SELECT id, title, description, due_date
		FROM %s.todo
		WHERE user_id = $1 AND id = $2`,
		h.Config.DbSchema)

	var todo model.Todo
	err = pool.QueryRow(ctx, query, userId, todoId).Scan(
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
	userId := gc.GetInt("user-id")

	type Incoming struct {
		Id          int     `json:"id"`
		Completed   *bool   `json:"completed"`
		Title       *string `json:"title"`
		Description *string `json:"description"`
		DueDate     *string `json:"due_date"`
	}

	var req Incoming
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse due_date
	var duePtr *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		t, err := time.Parse("2006-01-02", *req.DueDate)
		if err != nil {
			gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid due_date format (YYYY-MM-DD)"})
			return
		}
		duePtr = &t
	}

	// Insert a new entry
	if req.Id < 0 {
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
			req.Title,
			req.Description,
			duePtr,
			req.Completed,
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

	if req.Completed != nil {
		setClauses = append(setClauses, fmt.Sprintf("completed = $%d", argIdx))
		args = append(args, *req.Completed)
		argIdx++
	}

	if req.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *req.Title)
		argIdx++
	}

	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}

	if req.DueDate != nil {
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

	args = append(args, userId, req.Id)

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
		"id":      req.Id,
		"message": "Todo updated successfully",
	})
}

func (h *ApiHandler) AdminDeleteTodo(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool
	dbSchema := h.Config.DbSchema

	userId := gc.GetInt("user-id")

	todoIdStr := gc.Param("todoId")
	if todoIdStr == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "todoId is required"})
		return
	}

	todoId, err := strconv.Atoi(todoIdStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "todoId must be a number"})
		return
	}

	query := fmt.Sprintf(`DELETE FROM %s.todo WHERE user_id = $1 AND id = $2`, dbSchema)
	res, err := pool.Exec(ctx, query, userId, todoId)
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
