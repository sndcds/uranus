package main

import (
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sndcds/pluto"
	"github.com/sndcds/uranus/api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func loginHandler(gc *gin.Context) {
	var creds struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := gc.BindJSON(&creds); err != nil || creds.Email == "" || creds.Password == "" {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "credentials required"})
		return
	}

	user, err := model.GetUser(app.Singleton, gc, creds.Email)
	if err != nil || app.ComparePasswords(user.PasswordHash, creds.Password) != nil {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// -----------------------
	// Create tokens
	// -----------------------
	accessExp := time.Now().Add(30 * time.Minute)
	refreshExp := time.Now().Add(7 * 24 * time.Hour)

	accessClaims := &app.Claims{
		UserId: user.Id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenStr, _ := accessToken.SignedString(app.Singleton.JwtKey)

	refreshClaims := &app.Claims{
		UserId: user.Id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExp),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenStr, _ := refreshToken.SignedString(app.Singleton.JwtKey)

	gc.JSON(http.StatusOK, gin.H{
		"message":       "login successful",
		"access_token":  accessTokenStr,
		"refresh_token": refreshTokenStr,
	})
}

func refreshHandler(gc *gin.Context) {
	fmt.Println("refreshHandler() ......................")
	// Get refresh token from cookie
	refreshToken, err := gc.Cookie("refresh_token")
	if err != nil {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token missing"})
		return
	}

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
	accessExp := time.Now().Add(30 * time.Minute)
	newClaims := &app.Claims{
		UserId: claims.UserId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	accessTokenStr, _ := accessToken.SignedString(app.Singleton.JwtKey)

	domain := app.Singleton.Config.DbHost
	if app.Singleton.Config.DevMode {
		domain = "localhost"
	}
	secure := !app.Singleton.Config.DevMode

	// Set new cookie
	gc.SetCookie(
		"access_token",
		accessTokenStr,
		int(time.Until(accessExp).Seconds()),
		"/",
		domain,
		secure,
		true, // HttpOnly
	)

	gc.JSON(http.StatusOK, gin.H{"message": "token refreshed"})
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

	if app.Singleton.Config.UseRouterMiddleware {
		router.Use(cors.New(cors.Config{
			AllowOrigins:     []string{"*"}, // app.Singleton.Config.AllowOrigins,
			AllowMethods:     []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Authorization", "Content-Type", "Accept"},
			ExposeHeaders:    []string{"Set-Cookie", "Origin", "Content-Length"},
			AllowCredentials: false,
			MaxAge:           12 * time.Hour,
		}))
	}

	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Public endpoints
	publicRoute := router.Group("/api")

	publicRoute.GET("/query", api.QueryHandler)
	publicRoute.GET("/user", app.JWTMiddleware, api.UserHandler)
	publicRoute.GET("/user/events", app.JWTMiddleware, api.AdminHandlerUserEvents)
	publicRoute.GET("/space", api.SpaceHandler)
	// publicRoute.POST("/query", api.QueryHandler)
	publicRoute.GET("/test", app.JWTMiddleware, testHandler)
	publicRoute.GET("/meta/:mode", api.GetMetaHandler)
	publicRoute.GET("/event/images/:event-id", api.EventImagesHandler)

	// Inject app middleware into Pluto's image routes
	pluto.Singleton.RegisterRoutes(publicRoute, app.JWTMiddleware)

	// Authorized endpoints, user must be logged in
	adminRoute := router.Group("/api/admin")

	adminRoute.POST("/login", loginHandler)
	adminRoute.POST("/refresh", refreshHandler)

	adminRoute.GET("/user/permissions/:mode", app.JWTMiddleware, api.AdminUserPermissionsHandler)
	adminRoute.GET("/user/event-organizers", app.JWTMiddleware, api.AdminUserEventOrganizersHandler)
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
