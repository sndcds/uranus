package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// TODO: Review code

type TeamInviteClaims struct {
	UserId         int `json:"user_id"`
	OrganizationId int `json:"organization_id"`
	jwt.RegisteredClaims
}

func (h *ApiHandler) AdminOrganizationTeamInvite(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool
	userId := gc.GetInt("user-id")
	langStr := gc.DefaultQuery("lang", "en")

	organizationIdStr := gc.Param("organizationId")
	organizationId, err := strconv.Atoi(organizationIdStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	// Parse request JSON
	var payload struct {
		Email  string `json:"email" binding:"required,email"`
		RoleId int    `json:"role_id" binding:"required"`
	}
	if err := gc.ShouldBindJSON(&payload); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: validate email

	tx, err := pool.Begin(ctx)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var memberUserId int
	var memberUserDisplayName string
	userSQL := fmt.Sprintf(`SELECT id, COALESCE(display_name, first_name || ' ' || last_name) AS display_name FROM %s."user" WHERE email_address = $1`, h.Config.DbSchema)
	err = tx.QueryRow(ctx, userSQL, payload.Email).Scan(&memberUserId, &memberUserDisplayName)
	if err != nil {
		if err == pgx.ErrNoRows {
			gc.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var organizationName string
	organizationSQL := fmt.Sprintf(`SELECT name FROM %s."organization" WHERE id = $1`, h.Config.DbSchema)
	err = tx.QueryRow(ctx, organizationSQL, organizationId).Scan(&organizationName)
	if err != nil {
		if err == pgx.ErrNoRows {
			gc.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
			return
		}
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var roleName string
	var roleDescription string
	roleSQL := fmt.Sprintf(`SELECT name, description FROM %s.team_member_role WHERE type_id = $1 AND iso_639_1 = $2`, h.Config.DbSchema)
	err = tx.QueryRow(ctx, roleSQL, payload.RoleId, langStr).Scan(&roleName, &roleDescription)
	if err != nil {
		if err == pgx.ErrNoRows {
			gc.JSON(http.StatusNotFound, gin.H{"error": "team member role not found"})
			return
		}
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Generate token and send email to users
	expiryHour := 1
	tokenExp := time.Now().Add(time.Duration(expiryHour) * time.Hour)
	tokenClaims := &TeamInviteClaims{
		UserId:         memberUserId,
		OrganizationId: organizationId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(tokenExp),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims)
	tokenString, err := token.SignedString([]byte(h.Config.JwtSecret))
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	messageQuery := fmt.Sprintf(`SELECT subject, template FROM %s.system_email_template WHERE context = 'team-invite-email' AND iso_639_1 = $1`, h.Config.DbSchema)
	_, err = pool.Exec(gc, messageQuery, langStr)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get message template"})
		return
	}
	var subject string
	var template string
	err = tx.QueryRow(ctx, messageQuery, langStr).Scan(&subject, &template)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			gc.JSON(http.StatusNotFound, gin.H{"error": "email template not found"})
		} else {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get email template"})
		}
		return
	}

	insertSql := fmt.Sprintf(`
	           INSERT INTO %s.organization_member_link (organization_id, user_id, member_role_id, accept_token, invited_at, invited_by_user_id)
	           VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, $5)`,
		h.Config.DbSchema)
	_, err = pool.Exec(ctx, insertSql, organizationId, memberUserId, payload.RoleId, tokenString, userId)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// Unique violation -> conflict
			gc.JSON(http.StatusConflict, gin.H{"error": "user is already invited"})
			return
		}
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	inviteAcceptUrl := gc.Request.Referer() + "admin/invite/accept?token=" + tokenString
	emailMessage := strings.Replace(template, "{{invite_link}}", inviteAcceptUrl, -1)
	emailMessage = strings.Replace(emailMessage, "{{expiry_hours}}", strconv.Itoa(expiryHour), -1)
	emailMessage = strings.Replace(emailMessage, "{{display_name}}", memberUserDisplayName, -1)
	emailMessage = strings.Replace(emailMessage, "{{organization_name}}", organizationName, -1)
	emailMessage = strings.Replace(emailMessage, "{{role_name}}", roleName, -1)
	emailMessage = strings.Replace(emailMessage, "{{role_description}}", roleDescription, -1)
	go func() {
		sendEmailErr := sendEmail(payload.Email, subject, emailMessage)
		if sendEmailErr != nil {
			gc.JSON(http.StatusOK, gin.H{
				"message":    "Unable to send invitation email.",
				"error_code": -1,
			})
		}
	}()

	// TODO: Error handling if email couldn´t be send

	gc.JSON(http.StatusCreated, gin.H{
		"message": "member invitation sent successfully",
	})
}

func (h *ApiHandler) AdminOrganizationTeamInviteAccept(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool

	var req struct {
		Token string `json:"token"`
	}
	if err := gc.BindJSON(&req); err != nil || req.Token == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "token required"})
		return
	}

	// Parse JWT token
	token, err := jwt.ParseWithClaims(req.Token, &TeamInviteClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(h.Config.JwtSecret), nil
	})
	if err != nil {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	claims, ok := token.Claims.(*TeamInviteClaims)
	if !ok || !token.Valid {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
		return
	}

	userId := claims.UserId
	organizationId := claims.OrganizationId

	// Start a transaction
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer func() {
		// Rollback if not already committed
		_ = tx.Rollback(ctx)
	}()

	// Query stored activation token
	var storedToken string
	var organizationMemberLinkId int
	query := fmt.Sprintf(`SELECT id, accept_token FROM %s.organization_member_link WHERE user_id = $1 AND organization_id = $2 FOR UPDATE`, h.Config.DbSchema)
	err = tx.QueryRow(ctx, query, userId, organizationId).Scan(&organizationMemberLinkId, &storedToken)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			gc.JSON(http.StatusNotFound, gin.H{"error": "organization member link not found"})
		} else {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		}
		return
	}

	// Compare tokens
	if storedToken != req.Token {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "token mismatch"})
		return
	}

	// Activate account
	updateQuery := fmt.Sprintf(`UPDATE %s.organization_member_link SET has_joined = true, accept_token = NULL WHERE id = $1`, h.Config.DbSchema)
	if _, err := tx.Exec(ctx, updateQuery, organizationMemberLinkId); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to accept invite, " + err.Error()})
		return
	}

	// Create user organization link
	uolSql := fmt.Sprintf(`
		INSERT INTO %s.user_organization_link (user_id, organization_id, permissions)
		VALUES ($1, $2, $3)`, h.Config.DbSchema)
	if _, err := tx.Exec(ctx, uolSql, userId, organizationId, 0); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to accept invite, " + err.Error()})
		return
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"message": "user joined successfully"})
}
