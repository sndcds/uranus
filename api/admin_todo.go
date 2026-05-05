package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) AdminUserGetTodos(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-user-todos")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	query := fmt.Sprintf(`
    	SELECT id, title, description, due_date, completed, importance
    	FROM %s.todo
    	WHERE user_uuid = $1
    	ORDER BY due_date IS NULL, due_date DESC, id DESC`,
		h.DbSchema)
	rows, err := h.DbPool.Query(ctx, query, userUuid)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	var todos []model.Todo

	for rows.Next() {
		var todo model.Todo
		err := rows.Scan(
			&todo.Id,
			&todo.Title,
			&todo.Description,
			&todo.DueDate,
			&todo.Completed,
			&todo.Importance,
		)
		if err != nil {
			debugf(err.Error())
			apiRequest.InternalServerError()
			return
		}
		todos = append(todos, todo)
	}

	if rows.Err() != nil {
		debugf(rows.Err().Error())
		apiRequest.InternalServerError()
		return
	}

	apiRequest.Success(http.StatusOK, gin.H{"todos": todos, "total_count": len(todos)}, "")
}

func (h *ApiHandler) AdminGetTodo(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-user-todo")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	todoId, ok := ParamInt(gc, "todoId")
	if !ok {
		apiRequest.Required("todoId is required")
		return
	}

	query := fmt.Sprintf(`SELECT id, title, description, due_date,completed, importance FROM %s.todo WHERE user_uuid = $1 AND id = $2`, h.DbSchema)
	var todo model.Todo
	err := h.DbPool.QueryRow(ctx, query, userUuid, todoId).Scan(
		&todo.Id,
		&todo.Title,
		&todo.Description,
		&todo.DueDate,
		&todo.Completed,
		&todo.Importance,
	)
	if err != nil {
		debugf(err.Error())
		if err == pgx.ErrNoRows {
			apiRequest.Error(http.StatusNotFound, "todo not found")
		} else {
			apiRequest.InternalServerError()
		}
		return
	}

	apiRequest.Success(http.StatusOK, gin.H{"todo": todo}, "")
}

func (h *ApiHandler) AdminUpsertTodo(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "upsert-user-todo")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	var req struct {
		Id          int     `json:"id"`
		Title       *string `json:"title"`
		Description *string `json:"description"`
		DueDate     *string `json:"due_date"`
		Completed   *bool   `json:"completed"`
		Importance  *string `json:"importance"`
	}

	if err := gc.ShouldBindJSON(&req); err != nil {
		debugf(err.Error())
		apiRequest.InvalidJSONInput()
		return
	}

	// Parse due_date
	var duePtr *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		t, err := time.Parse("2006-01-02", *req.DueDate)
		if err != nil {
			apiRequest.Error(http.StatusBadRequest, "due_date_format_error")
			return
		}
		if t.Before(time.Now()) {
			apiRequest.Error(http.StatusBadRequest, "due_date_in_past_error")
			return
		}
		duePtr = &t
	}

	// Insert a new entry
	if req.Id < 0 {
		query := fmt.Sprintf(
			`INSERT INTO %s.todo (user_uuid, title, description, due_date, completed, importance)
			VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
			h.DbSchema)

		var newTodoId int
		err := h.DbPool.QueryRow(
			ctx,
			query,
			userUuid,
			req.Title,
			req.Description,
			duePtr,
			req.Completed,
			req.Importance,
		).Scan(&newTodoId)

		if err != nil {
			debugf(err.Error())
			apiRequest.InternalServerError()
			return
		}

		apiRequest.SuccessNoData(http.StatusCreated, "todo created successfully")
		return
	}

	// Update an entry
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

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

	if req.Completed != nil {
		setClauses = append(setClauses, fmt.Sprintf("completed = $%d", argIdx))
		args = append(args, *req.Completed)
		argIdx++
	}

	if req.Importance != nil {
		setClauses = append(setClauses, fmt.Sprintf("importance = $%d", argIdx))
		args = append(args, *req.Importance)
		argIdx++
	}

	if len(setClauses) == 0 {
		apiRequest.SuccessNoData(http.StatusOK, "nothing to update")
		return
	}

	query := fmt.Sprintf(
		`UPDATE %s.todo SET %s WHERE user_uuid = $%d AND id = $%d`,
		h.DbSchema,
		strings.Join(setClauses, ", "),
		argIdx, argIdx+1)

	fmt.Println(query)
	args = append(args, userUuid, req.Id)

	cmdTag, err := h.DbPool.Exec(ctx, query, args...)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	if cmdTag.RowsAffected() == 0 {
		apiRequest.Error(http.StatusNotFound, "todo not found")
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "todo updated successfully")
}

func (h *ApiHandler) AdminDeleteTodo(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "delete-todo")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	totoId, hasTodoId := ParamInt(gc, "todoId")
	if !hasTodoId {
		apiRequest.Required("todoId is required")
		return
	}

	query := fmt.Sprintf(`DELETE FROM %s.todo WHERE user_uuid = $1 AND id = $2`, h.DbSchema)
	cmdTag, err := h.DbPool.Exec(ctx, query, userUuid, totoId)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	if cmdTag.RowsAffected() == 0 {
		apiRequest.Error(http.StatusNotFound, "todo not found")
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "todo deleted successfully")
}
