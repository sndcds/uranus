package main

import (
	"flag"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sndcds/pluto"
	"github.com/sndcds/uranus/api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
	"net/http"
	"strconv"
	"time"
)

func loginHandler(gc *gin.Context) {
	var creds struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := gc.BindJSON(&creds); err != nil {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "credentials required"})
		return
	}

	if creds.Email == "" || creds.Password == "" {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "credentials required"})
		return
	}

	fmt.Println("email address:", creds.Email)
	fmt.Println("password:", creds.Password)
	hashedPassword, err := app.EncryptPassword(creds.Password)
	fmt.Println("hashedPassword:", hashedPassword)

	user, err := model.GetUser(app.Singleton, gc, creds.Email)
	if err != nil {
		http.Error(gc.Writer, "No user", http.StatusBadRequest)
		return
	}
	user.Print()

	err = app.ComparePasswords(user.PasswordHash, creds.Password)
	if err != nil {
		fmt.Println("Passwords do NOT match!")
		return
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &app.Claims{
		UserId: user.Id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(app.Singleton.JwtKey)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate token"})
		return
	}

	fmt.Println("token:", tokenStr)
	fmt.Println("host:", app.Singleton.Config.DbHost)

	// Send token as JSON
	gc.JSON(http.StatusOK, gin.H{"token": tokenStr})

	// Set cookie for authentication
	if app.Singleton.Config.DevMode {
		// Dev mode: no Secure flag, SameSite=None still needed for cross-origin
		gc.SetCookie(
			"uranus_auth_token", // name
			tokenStr,            // value
			3600*4,              // maxAge in seconds
			"/",                 // path
			"localhost",         // domain
			true,                // secure (false in dev)
			true,                // httpOnly
		)

		// Required for cross-origin: Manually set SameSite=None (since gin's SetCookie doesn't support SameSite explicitly)
		gc.Writer.Header().Add("Set-Cookie",
			fmt.Sprintf("uranus_auth_token=%s; Path=/; Max-Age=3600; HttpOnly; SameSite=None", tokenStr),
		)
	} else {
		// Production: secure cookie
		gc.SetCookie(
			"uranus_auth_token",
			tokenStr,
			app.Singleton.Config.AuthTokenExpirationTime,
			"/",
			app.Singleton.Config.DbHost,
			true, // secure
			true, // httpOnly
		)

		// SameSite=None needed for cross-origin in modern browsers
		gc.Writer.Header().Add("Set-Cookie",
			fmt.Sprintf("uranus_auth_token=%s; Path=/; Max-Age=3600; HttpOnly; Secure; SameSite=None", tokenStr),
		)
	}
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

	// Before defining any routes
	origins := map[string]bool{}
	for _, origin := range app.Singleton.Config.AllowOrigins {
		origins[origin] = true
	}

	// Create a Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New() // Use `Default()` for built-in logging and recovery

	if app.Singleton.Config.UseRouterMiddleware {
		router.Use(cors.New(cors.Config{
			AllowOriginFunc: func(origin string) bool {
				return origins[origin]
			},
			AllowMethods:     []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Authorization", "Content-Type", "Accept"},
			ExposeHeaders:    []string{"Set-Cookie", "Origin", "Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}))
	}

	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	fmt.Println("AllowOrigins:", app.Singleton.Config.AllowOrigins)

	/*
		GET    /spaces/{id}         → get one space
		POST   /spaces              → create a space
		PUT    /spaces/{id}         → replace a space
		PATCH  /spaces/{id}         → update certain fields
		DELETE /spaces/{id}         → delete a space
	*/

	// Public endpoints
	publicRoute := router.Group("/api")

	publicRoute.GET("/query", api.QueryHandler)
	publicRoute.GET("/user", app.JWTMiddleware, api.UserHandler)
	publicRoute.GET("/user/events", app.JWTMiddleware, api.AdminHandlerUserEvents)
	publicRoute.GET("/space", api.SpaceHandler)
	publicRoute.POST("/query", api.QueryHandler)
	publicRoute.GET("/test", app.JWTMiddleware, testHandler)
	publicRoute.GET("/meta/:mode", api.GetMetaHandler)
	publicRoute.GET("/event/images/:event-id", api.EventImagesHandler)

	// Inject app middleware into Pluto's image routes
	pluto.Singleton.RegisterRoutes(publicRoute, app.JWTMiddleware)

	// Authorized endpoints, user must be logged in
	adminRoute := router.Group("/api/admin")

	adminRoute.POST("/login", loginHandler)
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
