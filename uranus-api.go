package main

import (
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sndcds/pluto"
	"github.com/sndcds/uranus/api"
	"github.com/sndcds/uranus/api/admin"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func signupHandler(gc *gin.Context) {
	pool := app.Singleton.MainDbPool

	// Parse incoming JSON
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := gc.BindJSON(&req); err != nil || req.Email == "" || req.Password == "" {
		// Todo: Validate
		gc.JSON(http.StatusBadRequest, gin.H{"error": "email and password required"})
		return
	}

	// Encrypt password
	passwordHash, err := app.EncryptPassword(req.Password)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt password"})
		return
	}

	// Check if user already exists
	var exists bool
	checkQuery := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s.user WHERE email_address = $1)", app.Singleton.Config.DbSchema)
	err = pool.QueryRow(gc, checkQuery, req.Email).Scan(&exists)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if exists {
		gc.JSON(http.StatusConflict, gin.H{"error": "user already exists"})
		return
	}

	// Insert new user
	var newUserId int
	insertQuery := fmt.Sprintf(`
		INSERT INTO %s.user (email_address, password_hash)
		VALUES ($1, $2)
		RETURNING id
	`, app.Singleton.Config.DbSchema)

	err = pool.QueryRow(gc, insertQuery, req.Email, passwordHash).Scan(&newUserId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	// Respond success
	gc.JSON(http.StatusCreated, gin.H{
		"message": "user created successfully",
		"user_id": newUserId,
	})
}

func loginHandler(gc *gin.Context) {
	var credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := gc.BindJSON(&credentials); err != nil || credentials.Email == "" || credentials.Password == "" {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "credentials required"})
		return
	}

	user, err := model.GetUser(app.Singleton, credentials.Email)
	if err != nil || app.ComparePasswords(user.PasswordHash, credentials.Password) != nil {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// -----------------------
	// Create tokens
	// -----------------------
	accessExp := time.Now().Add(time.Duration(app.Singleton.Config.AuthTokenExpirationTime) * time.Second)
	refreshExp := time.Now().Add(7 * 24 * time.Hour)

	accessClaims := &app.Claims{
		UserId: user.Id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenStr, err := accessToken.SignedString(app.Singleton.JwtKey)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create access token"})
		return
	}

	refreshClaims := &app.Claims{
		UserId: user.Id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExp),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenStr, err := refreshToken.SignedString(app.Singleton.JwtKey)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create refresh token"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"message":       "login successful",
		"access_token":  accessTokenStr,
		"refresh_token": refreshTokenStr,
	})
}

func refreshHandler(gc *gin.Context) {
	fmt.Println("refreshHandler()")

	// Get refresh token from Authorization header
	authHeader := gc.GetHeader("Authorization")
	if authHeader == "" {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header missing"})
		return
	}

	// Expected format: "Bearer <token>"
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
		return
	}
	refreshToken := parts[1]

	// Parse and validate token
	claims := &app.Claims{}
	tkn, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		return app.Singleton.JwtKey, nil
	})
	if err != nil || !tkn.Valid {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	// Issue new access token
	accessExp := time.Now().Add(time.Duration(app.Singleton.Config.AuthTokenExpirationTime) * time.Second)
	newClaims := &app.Claims{
		UserId: claims.UserId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	accessTokenStr, err := accessToken.SignedString(app.Singleton.JwtKey)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sign access token"})
		return
	}

	// Send new access token back in header (and optionally JSON body)
	gc.Header("Authorization", "Bearer "+accessTokenStr)
	gc.JSON(http.StatusOK, gin.H{
		"message":      "token refreshed",
		"access_token": accessTokenStr,
		"expires_in":   int(time.Until(accessExp).Seconds()),
	})
}

func main() {
	// Configuration
	configFileName := flag.String("config", "config.json", "Path to config file")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()
	fmt.Println("Config file:", *configFileName)

	_, err := app.New(*configFileName)
	if err != nil {
		panic(err)
	}

	if *verbose {
		app.Singleton.Config.Verbose = true
	}

	app.Singleton.Config.Print()

	_, err = pluto.New(*configFileName, app.Singleton.MainDbPool, true)
	if err != nil {
		panic(err)
	}

	// Create a Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New() // Use `Default()` for built-in logging and recovery

	/*
		if app.Singleton.Config.UseRouterMiddleware {
			router.Use(cors.New(cors.Config{
				AllowOrigins:     []string{"*"}, // any origin
				AllowMethods:     []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
				AllowHeaders:     []string{"Origin", "Authorization", "Content-Type", "Accept"},
				ExposeHeaders:    []string{"Authorization"},
				AllowCredentials: false, // no cookies needed
				MaxAge:           12 * time.Hour,
			}))
		}
	*/

	if app.Singleton.Config.UseRouterMiddleware {
		router.Use(func(gc *gin.Context) {
			origin := gc.GetHeader("Origin")
			if origin != "" {
				gc.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				gc.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
				gc.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Authorization, Content-Type, Accept")
				gc.Writer.Header().Set("Access-Control-Expose-Headers", "Authorization")
				gc.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if gc.Request.Method == "OPTIONS" {
				gc.AbortWithStatus(204)
				return
			}

			gc.Next()
		})
	}

	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Public endpoints
	publicRoute := router.Group("/api")

	publicRoute.GET("/choosable-venues/organizer/:id", api.ChoosableOrganizerVenuesHandler)
	publicRoute.GET("/choosable-spaces/venue/:id", api.ChoosableVenueSpacesHandler)

	publicRoute.GET("/query", api.QueryHandler)
	publicRoute.GET("/user", app.JWTMiddleware, api.UserHandler) // Todo: To be removed
	publicRoute.GET("/user/events", app.JWTMiddleware, api.AdminHandlerUserEvents)
	publicRoute.GET("/space", api.SpaceHandler)
	publicRoute.GET("/space/types", api.SpaceTypesHandler)

	publicRoute.GET("/test", app.JWTMiddleware, testHandler)
	publicRoute.GET("/meta/:mode", api.GetMetaHandler)
	publicRoute.GET("/event/images/:event-id", api.EventImagesHandler)

	// Inject app middleware into Pluto's image routes
	pluto.Singleton.RegisterRoutes(publicRoute, app.JWTMiddleware)

	// Authorized endpoints, user must be logged in
	adminRoute := router.Group("/api/admin")

	adminRoute.POST("/login", loginHandler)
	adminRoute.POST("/signup", signupHandler)
	adminRoute.POST("/refresh", refreshHandler)

	adminRoute.GET("/organizer/dashboard", app.JWTMiddleware, api_admin.OrganizerDashboardHandler)
	adminRoute.GET("/organizer/:id/events", app.JWTMiddleware, api_admin.OrganizerEventsHandler)

	adminRoute.POST("/organizer/create", app.JWTMiddleware, api_admin.OrganizerCreateHandler)
	adminRoute.POST("/venue/create", app.JWTMiddleware, api_admin.VenueCreateHandler)
	adminRoute.POST("/space/create", app.JWTMiddleware, api_admin.SpaceCreateHandler)

	adminRoute.GET("/user/choosable-event-organizers/organizer/:id", app.JWTMiddleware, api_admin.ChoosableUserEventOrganizersHandler)

	adminRoute.GET("/user/permissions/:mode", app.JWTMiddleware, api.AdminUserPermissionsHandler)
	adminRoute.GET("/event/:id", app.JWTMiddleware, api.AdminEventHandler)

	adminRoute.GET("/user/stats", app.JWTMiddleware, testHandler)
	adminRoute.GET("/user/spaces/:mode", app.JWTMiddleware, api.AdminUserSpacesHandler)
	adminRoute.GET("/events", app.JWTMiddleware, api.AdminEventsHandler)
	adminRoute.POST("/event/update", app.JWTMiddleware, api.AdminPostEventHandler)

	adminRoute.POST("image/upload", app.JWTMiddleware, api.AdminAddImageHandler)

	// Print all registered routes
	for _, route := range router.Routes() {
		fmt.Printf("%-6s -> %s (%s)\n", route.Method, route.Path, route.Handler)
	}

	// Start the server (Gin handles everything)
	port := ":" + strconv.Itoa(app.Singleton.Config.Port)
	fmt.Printf("Uranus server is running on port %s\n", port)
	err = router.Run(port)
	if err != nil {
		fmt.Println("app server error:", err)
	}
}

func testHandler(gc *gin.Context) {
	modeStr, _ := api.GetContextParam(gc, "mode")
	fmt.Println(modeStr)
	switch modeStr {
	case "dashboard":
		model.TestQuery(gc)
		break
	default:
		gc.JSON(http.StatusBadRequest, gin.H{"error": "unknown mode"})
	}
}
