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
	"strings"
	"time"
)

// Claims struct for JWT
type Claims struct {
	UserId int `json:"user_id"`
	jwt.RegisteredClaims
}

// var jwtKey = []byte("82jhdksl#") TODO: Remove!

func JWTMiddleware(gc *gin.Context) {
	// Try to get token from cookie first
	tokenStr, err := gc.Cookie("uranus_auth_token")
	if err != nil || tokenStr == "" {
		// Fallback: try to get token from Authorization header
		authHeader := gc.GetHeader("Authorization")
		if authHeader == "" {
			gc.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing Authorization token"})
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			gc.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
			return
		}
		tokenStr = parts[1]
	}

	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return app.Singleton.JwtKey, nil
	})

	fmt.Println("token: ", token)

	if err != nil || !token.Valid {
		gc.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
		return
	}

	// Token is valid, save user info in context
	gc.Set("claims", claims)
	gc.Set("userId", claims.UserId)

	gc.Next()
}

func loginHandler(gc *gin.Context) {
	var creds struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := gc.BindJSON(&creds); err != nil {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "missing credentials"})
		return
	}

	if creds.Email == "" || creds.Password == "" {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "missing credentials"})
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
	claims := &Claims{
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
			3600,                // maxAge in seconds
			"/",                 // path
			"localhost",         // domain
			true,                // secure (false in dev)
			false,               // httpOnly
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

	_, err = pluto.New(*configFileName, app.Singleton.MainDb, true)
	if err != nil {
		panic(err)
	}

	// Create a Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New() // Use `Default()` for built-in logging and recovery
	// Add middleware explicitly
	// router.Use(gin.Logger())
	// router.Use(gin.Recovery())

	if app.Singleton.Config.UseRouterMiddleware {

		origins := map[string]bool{}
		for _, origin := range app.Singleton.Config.AllowOrigins {
			origins[origin] = true
		}

		router.Use(cors.New(cors.Config{
			AllowOriginFunc: func(origin string) bool {
				return origins[origin]
			},
			AllowMethods:     []string{"GET", "POST", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Authorization", "Content-Type", "Accept"},
			ExposeHeaders:    []string{"Set-Cookie", "Origin", "Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}))
	}

	fmt.Println("AllowOrigins:", app.Singleton.Config.AllowOrigins)

	// Register routes
	apiRoute := router.Group("/api")
	{
		apiRoute.POST("/login", loginHandler)
		apiRoute.GET("/query", api.QueryHandler)
		apiRoute.POST("/query", api.QueryHandler)

		apiRoute.GET("/user", JWTMiddleware, api.UserHandler)

		apiRoute.GET("/space", api.SpaceHandler)

		apiRoute.POST("/event", JWTMiddleware, api.CreateEventHandler)

		// Inject app middleware into Pluto's image routes
		pluto.Singleton.RegisterRoutes(apiRoute, JWTMiddleware)

		// Print all registered routes
		for _, route := range router.Routes() {
			fmt.Printf("%-6s -> %s (%s)\n", route.Method, route.Path, route.Handler)
		}
	}

	// Start the server (Gin handles everything)
	port := ":" + strconv.Itoa(app.Singleton.Config.Port)
	fmt.Printf("Uranus server is running on port %s\n", port)
	err = router.Run(port)
	if err != nil {
		fmt.Println("app server error:", err)
	}
}
