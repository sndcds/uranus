package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

/*
adminRoute.POST("/todo", apiHandler.AdminCreateTodo)
adminRoute.PUT("/todo/:todoId", apiHandler.AdminUpdateTodo)
*/

type Todo struct {
	Id          int        `json:"id"`
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	DueDate     *time.Time `json:"due_date"`
}

func (h *ApiHandler) AdminGetTodos(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool

	userId := UserIdFromAccessToken(gc)
	if userId == 0 {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	sql := fmt.Sprintf(`
		SELECT id AS todo_id, title, description, due_date
		FROM %s.todo
		WHERE user_id = $1 AND done = FALSE
		ORDER BY due_date ASC`,
		h.Config.DbSchema)

	rows, err := pool.Query(ctx, sql, userId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("query failed: %v", err)})
		return
	}
	defer rows.Close()

	var todos []Todo

	for rows.Next() {
		var todo Todo
		if err := rows.Scan(
			&todo.Id,
			&todo.Title,
			&todo.Description,
			&todo.DueDate,
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
		"messages": todos,
	})
}

func (h *ApiHandler) AdminGetTodo(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool

	userId := UserIdFromAccessToken(gc)
	if userId == 0 {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	todoIdStr := gc.Param("todo_id")
	if todoIdStr == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "todo_id is required"})
		return
	}

	todoId, err := strconv.Atoi(todoIdStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "todo_id must be a number"})
		return
	}

	sql := fmt.Sprintf(`
		SELECT id, title, description, due_date
		FROM %s.todo
		WHERE user_id = $1 AND id = $2`,
		h.Config.DbSchema)

	var todo Todo
	err = pool.QueryRow(ctx, sql, userId, todoId).Scan(
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

func (h *ApiHandler) AdminCreateTodo(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool
	dbSchema := h.Config.DbSchema

	userId := UserIdFromAccessToken(gc)
	if userId == 0 {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	type Incoming struct {
		Title       string  `json:"title" binding:"required"`
		Description *string `json:"description"`
		DueDate     *string `json:"due_date"`
	}

	var req Incoming
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var due time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		t, err := time.Parse("2006-01-02", *req.DueDate)
		if err != nil {
			gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid due_date format, expected YYYY-MM-DD"})
			return
		}
		due = t
	}

	sql := fmt.Sprintf(`
		INSERT INTO %s.todo (user_id, title, description, due_date)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		dbSchema)

	var duePtr *time.Time
	if !due.IsZero() {
		duePtr = &due
	}

	var todoId int
	err := pool.QueryRow(ctx, sql, userId, req.Title, req.Description, duePtr).Scan(&todoId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create todo: %v", err)})
		return
	}

	gc.JSON(http.StatusCreated, gin.H{
		"todo_id": todoId,
		"message": "Todo saved successfully",
	})
}

func (h *ApiHandler) AdminUpdateTodo(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool
	dbSchema := h.Config.DbSchema

	userId := UserIdFromAccessToken(gc)
	if userId == 0 {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	todoIdStr := gc.Param("todo_id")
	if todoIdStr == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "todo_id is required"})
		return
	}

	todoId, err := strconv.Atoi(todoIdStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "todo_id must be a number"})
		return
	}

	type Incoming struct {
		Title       string  `json:"title"`
		Description string  `json:"description"`
		DueDate     *string `json:"due_date"`
	}

	var req Incoming
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var duePtr *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		t, err := time.Parse("2006-01-02", *req.DueDate)
		if err != nil {
			gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid due_date format, expected YYYY-MM-DD"})
			return
		}
		duePtr = &t
	}

	sql := fmt.Sprintf(`
		UPDATE %s.todo
		SET title = $1, description = $2, due_date = $3
		WHERE user_id = $4 AND id = $5`,
		dbSchema,
	)

	res, err := pool.Exec(ctx, sql, req.Title, req.Description, duePtr, userId, todoId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("update failed: %v", err)})
		return
	}

	if res.RowsAffected() == 0 {
		gc.JSON(http.StatusNotFound, gin.H{"error": "todo not found"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"todo_id": todoId,
		"message": "Todo updated successfully",
	})
}
