package main

import (
	"flag"
	"fmt"
	"net/smtp"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/pluto"
	"github.com/sndcds/uranus/api"
	"github.com/sndcds/uranus/api/admin"
	"github.com/sndcds/uranus/app"
)

func sendTestMail() {
	from := "oklab_noreply@grain.one"
	password := app.Singleton.Config.SmtpPassword

	// List of recipients
	to := []string{
		"pippa@grain.one",
	}

	// Your SMTP server configuration
	smtpHost := app.Singleton.Config.SmtpHost
	smtpPort := app.Singleton.Config.SmtpPort

	// Message body (RFC 822 format)
	message := []byte("Subject: Hello from Go!\r\n" +
		"To: undisclosed-recipients:;\r\n" +
		"From: roald@grain.one\r\n" +
		"\r\n" +
		"This is a test email sent from Go.\r\nUranus API is prepared to reset password :-)")

	// Authentication
	auth := smtp.PlainAuth("", from, password, smtpHost)
	addr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)

	// Send the email
	err := smtp.SendMail(addr, auth, from, to, message)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Email sent successfully!")
}

func main() {
	fmt.Println("start")

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

	// OK
	publicRoute.GET("/events", api.GetEventsHandler)
	publicRoute.GET("/event/:id", api.GetEventHandler)

	publicRoute.GET("/user/:userId/avatar/:size", api.GetUserAvatarHandler)
	publicRoute.GET("/user/:userId/avatar", api.GetUserAvatarHandler)

	publicRoute.GET("/choosable-venues/organizer/:id", api.ChoosableOrganizerVenuesHandler)
	publicRoute.GET("/choosable-spaces/venue/:id", api.ChoosableVenueSpacesHandler)
	publicRoute.GET("/choosable-event-types", api.ChoosableEventTypesHandler)
	publicRoute.GET("/choosable-event-genres/event-type/:id", api.ChoosableEventGenresHandler)
	publicRoute.GET("/choosable-licenses", api.ChoosableLicensesHandler)
	publicRoute.GET("/choosable-legal-forms", api.ChoosableLegalFormsHandler)
	publicRoute.GET("/choosable-countries", api.ChoosableCountriesHandler)
	publicRoute.GET("/choosable-states", api.ChoosableStatesHandler)

	publicRoute.GET("/query", api.QueryHandler)                  // TODO: Refactor QueryVenueForMap
	publicRoute.GET("/user", app.JWTMiddleware, api.UserHandler) // Todo: To be removed
	publicRoute.GET("/user/events", app.JWTMiddleware, api.AdminHandlerUserEvents)
	publicRoute.GET("/space", api.SpaceHandler)
	publicRoute.GET("/space/types", api.SpaceTypesHandler)

	// Check ...
	// publicRoute.GET("/event/images/:event-id", api.EventImagesHandler)

	// Inject app middleware into Pluto's image routes
	pluto.Singleton.RegisterRoutes(publicRoute, app.JWTMiddleware)

	// Authorized endpoints, user must be logged in
	adminRoute := router.Group("/api/admin")

	// OK
	adminRoute.POST("/signup", api_admin.SignupHandler)
	adminRoute.POST("/login", api_admin.LoginHandler)
	adminRoute.POST("/refresh", api_admin.RefreshHandler)
	adminRoute.POST("/forgot-password", api_admin.ForgotPasswordHandler)
	adminRoute.POST("/reset-password", api_admin.ResetPasswordHandler)

	adminRoute.GET("/user/me", api_admin.GetUserProfileHandler)
	adminRoute.PUT("/user/me", api_admin.UpdateUserProfileHandler)
	adminRoute.POST("/user/me/avatar", api_admin.UploadUserAvatarHandler)
	adminRoute.DELETE("/user/me/avatar", api_admin.DeleteUserAvatarHandler)
	adminRoute.GET("/user/me/permissions", app.JWTMiddleware, api_admin.AdminUserPermissionsHandler)

	adminRoute.GET("/event/:eventId", app.JWTMiddleware, api_admin.GetAdminEventHandler)

	adminRoute.GET("/choosable-organizers", app.JWTMiddleware, api_admin.ChoosableOrganizersHandler)
	adminRoute.GET("/organizer/:organizerId", app.JWTMiddleware, api_admin.GetAdminOrganizerHandler)
	adminRoute.PUT("/organizer/:organizerId", app.JWTMiddleware, api_admin.UpdateOrganizerHandler)
	adminRoute.GET("/organizer/dashboard", app.JWTMiddleware, api_admin.OrganizerDashboardHandler)
	adminRoute.GET("/organizer/:organizerId/venues", app.JWTMiddleware, api_admin.OrganizerVenuesHandler)
	adminRoute.GET("/organizer/:organizerId/events", app.JWTMiddleware, api_admin.OrganizerEventsHandler)

	adminRoute.POST("/organizer/create", app.JWTMiddleware, api_admin.CreateOrganizerHandler)

	adminRoute.POST("/venue/create", app.JWTMiddleware, api_admin.CreateVenueHandler)
	adminRoute.GET("/venue/:venueId", app.JWTMiddleware, api_admin.GetAdminVenueHandler)
	adminRoute.PUT("/venue/:venueId", app.JWTMiddleware, api_admin.UpdateVenueHandler)

	adminRoute.POST("/space/create", app.JWTMiddleware, api_admin.CreateSpaceHandler)
	adminRoute.GET("/space/:spaceId", app.JWTMiddleware, api_admin.GetAdminSpaceHandler)
	adminRoute.PUT("/space/:spaceId", app.JWTMiddleware, api_admin.UpdateSpaceHandler)

	adminRoute.GET("/user/choosable-event-organizers/organizer/:organizerId", app.JWTMiddleware, api_admin.ChoosableUserEventOrganizersHandler)
	adminRoute.POST("/event/create", app.JWTMiddleware, api_admin.CreateEventHandler)
	adminRoute.PUT("/event/:eventId/header", app.JWTMiddleware, api_admin.UpdateEventHeaderHandler)
	adminRoute.PUT("/event/:eventId/description", app.JWTMiddleware, api_admin.UpdateEventDescriptionHandler)
	adminRoute.PUT("/event/:eventId/teaser", app.JWTMiddleware, api_admin.UpdateEventTeaserHandler)
	adminRoute.PUT("/event/:eventId/types", app.JWTMiddleware, api_admin.UpdateEventTypesHandler)
	adminRoute.PUT("/event/:eventId/space", app.JWTMiddleware, api_admin.UpdateEventSpaceHandler)
	adminRoute.PUT("/event/:eventId/links", app.JWTMiddleware, api_admin.UpdateEventLinksHandler)
	adminRoute.POST("/event/:eventId/image", app.JWTMiddleware, api_admin.UpdateEventImageHandler)
	adminRoute.PUT("/event/:eventId/dates", app.JWTMiddleware, api_admin.UpdateEventDatesHandler)

	// Check ...
	adminRoute.POST("image/upload", app.JWTMiddleware, api.AdminAddImageHandler)
	adminRoute.GET("/user/spaces/:mode", app.JWTMiddleware, api.AdminUserSpacesHandler)
	adminRoute.GET("/events", app.JWTMiddleware, api.AdminEventsHandler)

	fmt.Println("Gin mode:", gin.Mode())
	fmt.Println("Total routes:", len(router.Routes()))

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
