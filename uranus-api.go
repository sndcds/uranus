package main

import (
	"flag"
	"fmt"
	"net/smtp"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/pluto"
	"github.com/sndcds/uranus/api"
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

	apiHandler := &api.ApiHandler{
		DbPool: app.Singleton.MainDbPool,
		Config: &app.Singleton.Config,
	}

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
	publicRoute.GET("/events", apiHandler.GetEvents)
	publicRoute.GET("/events/types-count", apiHandler.GetEventsPerType)
	publicRoute.GET("/event/:eventId", apiHandler.GetEvent)
	publicRoute.GET("/event/:eventId/date/:dateId", apiHandler.GetEventByDateId)

	publicRoute.GET("/geojson/venues", apiHandler.GetGeojsonVenues)

	publicRoute.GET("/organizers", apiHandler.GetOrganizers)

	publicRoute.GET("/user/:userId/avatar/:size", apiHandler.GetUserAvatar)
	publicRoute.GET("/user/:userId/avatar", apiHandler.GetUserAvatar)

	publicRoute.GET("/choosable-venues/organizer/:id", apiHandler.GetChoosableOrganizerVenues)
	publicRoute.GET("/choosable-spaces/venue/:id", apiHandler.GetChoosableVenueSpaces)
	publicRoute.GET("/choosable-event-types", apiHandler.GetChoosableEventTypes)
	publicRoute.GET("/choosable-event-genres/event-type/:id", apiHandler.GetChoosableEventGenres)
	publicRoute.GET("/choosable-states", apiHandler.GetChoosableStates)
	publicRoute.GET("/choosable-licenses", apiHandler.GetChoosableLicenses)
	publicRoute.GET("/choosable-legal-forms", apiHandler.GetChoosableLegalForms)
	publicRoute.GET("/choosable-countries", apiHandler.GetChoosableCountries)
	publicRoute.GET("/choosable-release-states", apiHandler.GetChoosableReleaseStates)
	publicRoute.GET("/choosable-languages", apiHandler.GetChoosableLanguages)
	publicRoute.GET("/choosable-url-types/event", apiHandler.GetChoosableEventUrlTypes)

	publicRoute.GET("/accessibility/flags", apiHandler.GetAccessibilityFlags)

	publicRoute.GET("/query", api.QueryHandler)                  // TODO: Refactor QueryVenueForMap
	publicRoute.GET("/user", app.JWTMiddleware, api.UserHandler) // Todo: To be removed
	publicRoute.GET("/user/events", app.JWTMiddleware, api.AdminHandlerUserEvents)
	publicRoute.GET("/space", apiHandler.GetSpace)
	publicRoute.GET("/space/types", apiHandler.GetSpaceTypes)

	// Check ...
	// publicRoute.GET("/event/images/:event-id", api.EventImagesHandler)

	// Inject app middleware into Pluto's image routes
	pluto.Singleton.RegisterRoutes(publicRoute, app.JWTMiddleware)

	// Authorized endpoints, user must be logged in
	adminRoute := router.Group("/api/admin")

	// OK
	adminRoute.POST("/signup", apiHandler.Signup)
	adminRoute.POST("/activate", apiHandler.Activate)
	adminRoute.POST("/login", apiHandler.Login)
	adminRoute.POST("/refresh", apiHandler.Refresh)
	adminRoute.POST("/forgot-password", apiHandler.ForgotPassword)
	adminRoute.POST("/reset-password", apiHandler.ResetPassword)

	adminRoute.POST("/send-message", app.JWTMiddleware, apiHandler.AdminSendMessage)
	adminRoute.GET("/messages", app.JWTMiddleware, apiHandler.AdminGetMessages)

	adminRoute.GET("/todos", app.JWTMiddleware, apiHandler.AdminGetTodos)
	adminRoute.GET("/todo/:todoId", app.JWTMiddleware, apiHandler.AdminGetTodo)
	adminRoute.POST("/todo", app.JWTMiddleware, apiHandler.AdminCreateTodo)
	adminRoute.PUT("/todo/:todoId", app.JWTMiddleware, apiHandler.AdminUpdateTodo)
	adminRoute.DELETE("/todo/:todoId", app.JWTMiddleware, apiHandler.AdminDeleteTodo)

	adminRoute.GET("/user/me", app.JWTMiddleware, apiHandler.AdminGetUserProfil)
	adminRoute.PUT("/user/me", app.JWTMiddleware, apiHandler.AdminUpdateUserProfile)
	adminRoute.POST("/user/me/avatar", app.JWTMiddleware, apiHandler.AdminUploadUserAvatar)
	adminRoute.DELETE("/user/me/avatar", app.JWTMiddleware, apiHandler.AdminDeleteUserAvatar)
	adminRoute.GET("/user/me/permissions", app.JWTMiddleware, apiHandler.AdminUserPermissions)

	adminRoute.GET("/user/event/notification", app.JWTMiddleware, apiHandler.AdminGetUserEventNotification)

	adminRoute.GET("/event/:eventId", app.JWTMiddleware, apiHandler.AdminGetEvent)
	adminRoute.DELETE("/event/:eventId", app.JWTMiddleware, apiHandler.AdminDeleteEvent)

	adminRoute.GET("/choosable-organizers", app.JWTMiddleware, apiHandler.AdminGetChoosableOrganizers)
	adminRoute.GET("/organizer/:organizerId", app.JWTMiddleware, apiHandler.AdminGetOrganizer)
	adminRoute.PUT("/organizer/:organizerId", app.JWTMiddleware, apiHandler.AdminUpdateOrganizer)
	adminRoute.DELETE("/organizer/:organizerId", app.JWTMiddleware, apiHandler.AdminDeleteOrganizer)

	adminRoute.GET("/organizer/dashboard", app.JWTMiddleware, apiHandler.AdminGetOrganizerDashboard)
	adminRoute.GET("/organizer/:organizerId/venues", app.JWTMiddleware, apiHandler.AdminGetOrganizerVenues)
	adminRoute.GET("/organizer/:organizerId/events", app.JWTMiddleware, apiHandler.AdminGetOrganizerEvents)
	adminRoute.GET("/organizer/:organizerId/event/permission", app.JWTMiddleware, apiHandler.AdminGetOrganizerEventPermissions)

	adminRoute.POST("/organizer/create", app.JWTMiddleware, apiHandler.AdminCreateOrganizer)

	adminRoute.POST("/venue/create", app.JWTMiddleware, apiHandler.AdminCreateVenue)
	adminRoute.GET("/venue/:venueId", app.JWTMiddleware, apiHandler.AdminGetVenue)
	adminRoute.PUT("/venue/:venueId", app.JWTMiddleware, apiHandler.AdminUpdateVenue)

	adminRoute.POST("/space/create", app.JWTMiddleware, apiHandler.AdminCreateSpace)
	adminRoute.GET("/space/:spaceId", app.JWTMiddleware, apiHandler.AdminGetSpace)
	adminRoute.PUT("/space/:spaceId", app.JWTMiddleware, apiHandler.AdminUpdateSpace)

	adminRoute.GET("/user/choosable-event-organizers/organizer/:organizerId", app.JWTMiddleware, apiHandler.AdminChoosableUserEventOrganizers)
	adminRoute.POST("/event/create", app.JWTMiddleware, apiHandler.AdminCreateEvent)
	adminRoute.PUT("/event/:eventId/header", app.JWTMiddleware, apiHandler.AdminUpdateEventHeader)
	adminRoute.PUT("/event/:eventId/description", app.JWTMiddleware, apiHandler.AdminUpdateEventDescription)
	adminRoute.PUT("/event/:eventId/teaser", app.JWTMiddleware, apiHandler.AdminUpdateEventTeaser)
	adminRoute.PUT("/event/:eventId/types", app.JWTMiddleware, apiHandler.AdminUpdateEventTypes)
	adminRoute.PUT("/event/:eventId/space", app.JWTMiddleware, apiHandler.AdminUpdateEventSpace)
	adminRoute.PUT("/event/:eventId/links", app.JWTMiddleware, apiHandler.AdminUpdateEventLinks)
	adminRoute.PUT("/event/:eventId/dates", app.JWTMiddleware, apiHandler.AdminUpdateEventDates)
	adminRoute.PUT("/event/:eventId/tags", app.JWTMiddleware, apiHandler.AdminUpdateEventTags)
	adminRoute.PUT("/event/:eventId/languages", app.JWTMiddleware, apiHandler.AdminUpdateEventLanguages)
	adminRoute.PUT("/event/:eventId/release-status", app.JWTMiddleware, apiHandler.AdminUpdateEventReleaseStatus)

	adminRoute.POST("/event/:eventId/image", app.JWTMiddleware, apiHandler.AdminUpdateEventImage)
	adminRoute.DELETE("/event/:eventId/image", app.JWTMiddleware, apiHandler.AdminDeleteEventMainImage)

	adminRoute.POST("/event/:eventId/teaser/image", app.JWTMiddleware, apiHandler.AdminUpdateEventTeaserImage)

	// Check ...
	// adminRoute.POST("image/upload", app.JWTMiddleware, api.AdminAddImageHandler) TODO: Unused
	// adminRoute.GET("/user/spaces/:mode", app.JWTMiddleware, api.AdminUserSpacesHandler) TODO: Unused
	// adminRoute.GET("/events", app.JWTMiddleware, api.AdminEventsHandler) TODO: Unused

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
