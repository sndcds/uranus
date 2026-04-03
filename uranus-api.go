package main

import (
	"context"
	"flag"
	"fmt"
	"strconv"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/pluto"
	"github.com/sndcds/uranus/api"
	"github.com/sndcds/uranus/app"
)

func main() {
	configFileName := flag.String("config", "config.json", "Path to config file")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()
	fmt.Println("Config file:", *configFileName)

	grains_api.Init(grains_api.Config{
		ServiceName: "Uranus API",
		APIVersion:  "1.0",
		TimeFormat:  "", // leave empty to use default RFC3339
	})

	var err error
	app.UranusInstance, err = app.Initialize(*configFileName)
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}

	err = app.UranusInstance.CheckAllDatabaseConsistency(context.Background())
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}
	app.UranusInstance.Log("CheckAllDatabaseConsistency succeded")

	if *verbose {
		app.UranusInstance.Config.Verbose = true
	}

	app.UranusInstance.Config.Print()

	apiHandler := &api.ApiHandler{
		Config:   &app.UranusInstance.Config,
		DbPool:   app.UranusInstance.MainDbPool,
		DbSchema: app.UranusInstance.Config.DbSchema,
	}

	_, err = pluto.Initialize(*configFileName, app.UranusInstance.MainDbPool, true)
	if err != nil {
		panic(err)
	}

	// Create a Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New() // Use `Default()` for built-in logging and recovery

	// Enable gzip compression (recommended level), exclude images and already-compressed data
	router.Use(gzip.Gzip(
		gzip.DefaultCompression,
		gzip.WithExcludedExtensions([]string{".png", ".jpg", ".jpeg", ".webp"}),
	))

	if app.UranusInstance.Config.UseRouterMiddleware {
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

	// Serve all files in ./static under /static
	router.Static("/api/info", "./static")

	// Public endpoints
	publicRoute := router.Group("/api")

	publicRoute.GET("/health", apiHandler.GetHealth)

	publicRoute.GET("/event/release-status-i18n", apiHandler.GetEventReleaseStatusI18n)

	publicRoute.GET("/events", apiHandler.GetEvents)                          // TODO: check!
	publicRoute.GET("/events/ics", apiHandler.GetEventsICS)                   // TODO: check!
	publicRoute.GET("/events/type-summary", apiHandler.GetEventTypeSummary)   // TODO: check!
	publicRoute.GET("/events/venue-summary", apiHandler.GetEventVenueSummary) // TODO: check!
	publicRoute.GET("/events/geojson", apiHandler.GetEventsGeoJSON)           // TODO: Reduce data

	publicRoute.GET("/event/:eventUuid/date/:dateUuid", apiHandler.GetEventByDateUuid)
	publicRoute.GET("/event/:eventUuid/date/:dateUuid/ics", apiHandler.GetEventDateICS) // TODO: check!

	publicRoute.GET("/venues/geojson", apiHandler.GetVenuesGeoJSON)

	publicRoute.GET("/organization/:orgUuid", apiHandler.GetOrganization)
	publicRoute.GET("/organizations", apiHandler.GetOrganizations)

	publicRoute.GET("/venue/:venueUuid", apiHandler.GetVenue)

	publicRoute.GET("/transport/stations", apiHandler.GetTransportStations)

	publicRoute.GET("/user/:userUuid/avatar/:size", apiHandler.GetUserAvatar)
	publicRoute.GET("/user/:userUuid/avatar", apiHandler.GetUserAvatar)

	publicRoute.GET("/event/type-genre-lookup", apiHandler.GetEventTypeGenreLookup)
	publicRoute.GET("/event/category-lookup", apiHandler.GetEventCategoryLookup)

	publicRoute.GET("/choosable-link-types", apiHandler.GetChoosableLinkTypes)
	publicRoute.GET("/choosable-venue-types", apiHandler.GetChoosableVenueTypes)
	publicRoute.GET("/choosable-space-types", apiHandler.GetChoosableSpaceTypes)
	publicRoute.GET("/choosable-legal-forms", apiHandler.GetChoosableLegalForms)
	publicRoute.GET("/choosable-license-types", apiHandler.GetChoosableLicenseTypes)
	publicRoute.GET("/choosable-countries", apiHandler.GetChoosableCountries)
	publicRoute.GET("/choosable-states", apiHandler.GetChoosableStates)
	publicRoute.GET("/choosable-languages", apiHandler.GetChoosableLanguages)            // TODO: check!
	publicRoute.GET("/choosable-price-types", apiHandler.GetChoosablePriceTypes)         // TODO: check!
	publicRoute.GET("/choosable-currencies", apiHandler.GetChoosableCurrencies)          // TODO: check!
	publicRoute.GET("/choosable-event-ocassions", apiHandler.GetChoosableEventOccasions) // TODO: check!

	publicRoute.GET("/choosable-venues", apiHandler.GetChoosableVenues)                                   // TODO: check!
	publicRoute.GET("/choosable-organizations", apiHandler.GetChoosableOrganizations)                     // TODO: check!
	publicRoute.GET("/choosable-venues/organization/:orgUuid", apiHandler.GetChoosableOrganizationVenues) // TODO: check!
	publicRoute.GET("/choosable-spaces/venue/:venueUuid", apiHandler.GetChoosableVenueSpaces)             // TODO: check!
	publicRoute.GET("/choosable-event-genres/event-type/:id", apiHandler.GetChoosableEventGenres)         // TODO: check!

	publicRoute.GET("/accessibility/flags", apiHandler.GetAccessibilityFlags) // TODO: check!

	// Inject app middleware into Pluto's image routes
	pluto.PlutoInstance.RegisterRoutes(publicRoute, app.JWTMiddleware) // TODO: check!

	publicRoute.POST("/signup", apiHandler.Signup)
	publicRoute.POST("/login", apiHandler.Login)
	publicRoute.POST("/activate", apiHandler.Activate)              // TODO: check!
	publicRoute.POST("/forgot-password", apiHandler.ForgotPassword) // TODO: check!
	publicRoute.POST("/reset-password", apiHandler.ResetPassword)   // TODO: check!

	// Authorized endpoints, user must be logged in
	adminRoute := router.Group("/api/admin", app.JWTMiddleware) // TODO: check!

	adminRoute.GET("/permissions/list", apiHandler.AdminGetPermissionsList) // TODO: check!

	adminRoute.POST("/refresh", apiHandler.Refresh) // TODO: check!

	// User
	adminRoute.GET("/user/profile", apiHandler.AdminGetUserProfile)
	adminRoute.PUT("/user/profile", apiHandler.AdminUpdateUserProfile)
	adminRoute.PUT("/user/settings", apiHandler.AdminUpdateUserProfileSettings)
	adminRoute.POST("/user/avatar", apiHandler.AdminUploadUserAvatar)
	adminRoute.DELETE("/user/avatar", apiHandler.AdminDeleteUserAvatar)

	adminRoute.GET("/user/todos", apiHandler.AdminUserGetTodos)
	adminRoute.GET("/user/todo/:todoId", apiHandler.AdminGetTodo)
	adminRoute.PUT("/user/todo", apiHandler.AdminUpsertTodo)
	adminRoute.DELETE("/user/todo/:todoId", apiHandler.AdminDeleteTodo)

	adminRoute.GET("/user/messages", apiHandler.AdminGetMessages)      // TODO: check!
	adminRoute.POST("/user/send-message", apiHandler.AdminSendMessage) // TODO: check!

	adminRoute.GET("/user/event/notifications", apiHandler.AdminGetUserEventNotifications)
	adminRoute.GET("/user/choosable-organizations", apiHandler.AdminGetChoosableOrganizations) // TODO: check!
	adminRoute.GET("/user/choosable-event-venues", apiHandler.AdminGetChoosableUserEventVenues)

	// Organisation
	adminRoute.GET("/organization/:orgUuid/member/:memberId/permissions", apiHandler.AdminGetOrganizationMemberPermissions)    // TODO: check!
	adminRoute.PUT("/organization/:orgUuid/member/:memberId/permissions", apiHandler.AdminUpdateOrganizationMemberPermissions) // TODO: check!

	adminRoute.POST("/organization/create", apiHandler.AdminCreateOrganization)
	adminRoute.GET("/organization/:orgUuid", apiHandler.AdminGetOrganization)
	adminRoute.PUT("/organization/:orgUuid/fields", apiHandler.UpdateOrganizationFields)
	adminRoute.DELETE("/organization/:orgUuid", apiHandler.AdminDeleteOrganization)

	adminRoute.GET("/organization/list", apiHandler.AdminGetOrganizationList)
	adminRoute.GET("/organization/:orgUuid/venues", apiHandler.AdminGetOrganizationVenues)
	adminRoute.GET("/organization/:orgUuid/events", apiHandler.AdminGetOrganizationEvents)

	adminRoute.GET("/organization/:orgUuid/team", apiHandler.AdminGetOrganizationTeam)                              // TODO: check!
	adminRoute.POST("/organization/:orgUuid/team/invite", apiHandler.AdminOrganizationTeamInvite)                   // TODO: check!
	adminRoute.DELETE("/organization/:orgUuid/team/member/:memberId", apiHandler.AdminDeleteOrganizationTeamMember) // TODO: check!
	adminRoute.POST("/organization/team/invite/accept", apiHandler.AdminOrganizationTeamInviteAccept)               // TODO: check!

	// Venue
	adminRoute.GET("/venue/:venueUuid", apiHandler.AdminGetVenue)
	adminRoute.POST("/venue/create", apiHandler.AdminCreateVenue)
	// adminRoute.PUT("/venue", apiHandler.AdminUpsertVenue) // TODO: refactor to be create with complete data set
	adminRoute.PUT("/venue/:venueUuid/fields", apiHandler.UpdateVenueFields)
	adminRoute.DELETE("/venue/:venueUuid", apiHandler.AdminDeleteVenue)

	// Space
	adminRoute.GET("/space/:spaceUuid", apiHandler.AdminGetSpace)
	adminRoute.POST("/space/create", apiHandler.AdminCreateSpace)
	// adminRoute.PUT("/space", apiHandler.AdminUpsertSpace) // TODO: refactor to be create with complete data set
	adminRoute.PUT("/space/:spaceUuid/fields", apiHandler.UpdateSpaceFields)
	adminRoute.DELETE("/space/:spaceUuid", apiHandler.AdminDeleteSpace)

	// Event
	adminRoute.GET("/event/:eventUuid", apiHandler.AdminGetEvent)
	adminRoute.POST("/delete/event/:eventUuid", apiHandler.AdminDeleteEvent)                    // TODO: check!
	adminRoute.POST("/event/:eventUuid/date", apiHandler.AdminUpsertEventDate)                  // TODO: check!
	adminRoute.PUT("/event/:eventUuid/date/:dateUuid", apiHandler.AdminUpsertEventDate)         // TODO: check!
	adminRoute.POST("/delete/event/:eventUuid/date/:dateUuid", apiHandler.AdminDeleteEventDate) // TODO: check!

	adminRoute.POST("/event/initial", apiHandler.AdminInitialEvent)
	adminRoute.POST("/event/create", apiHandler.AdminCreateEvent)
	adminRoute.PUT("/event/:eventUuid/dates", apiHandler.AdminUpdateEventDates)
	adminRoute.PUT("/event/:eventUuid/types", apiHandler.AdminUpdateEventTypes)
	adminRoute.PUT("/event/:eventUuid/languages", apiHandler.AdminUpdateEventLanguages)
	adminRoute.PUT("/event/:eventUuid/links", apiHandler.AdminUpdateEventLinks)
	adminRoute.PUT("/event/:eventUuid/venue", apiHandler.AdminUpdateEventVenue)
	adminRoute.PUT("/event/:eventUuid/fields", apiHandler.UpdateEventFields)

	adminRoute.PUT("/event/:eventUuid/release-status", apiHandler.AdminUpdateEventReleaseStatus)           // TODO: check!
	adminRoute.PUT("/event/:eventUuid/header", apiHandler.AdminUpdateEventHeader)                          // TODO: check!
	adminRoute.PUT("/event/:eventUuid/description", apiHandler.AdminUpdateEventDescription)                // TODO: check!
	adminRoute.PUT("/event/:eventUuid/summary", apiHandler.AdminUpdateEventSummary)                        // TODO: check!
	adminRoute.PUT("/event/:eventUuid/participation-infos", apiHandler.AdminUpdateEventParticipationInfos) // TODO: check!

	// Pluto Image
	adminRoute.POST("/image/:context/:contextUuid/:identifier", apiHandler.AdminUpsertPlutoImage)
	adminRoute.DELETE("/image/:context/:contextUuid/:identifier", apiHandler.AdminDeletePlutoImage)

	/*
		fmt.Println("Gin mode:", gin.Mode())
		fmt.Println("Total routes:", len(router.Routes()))

		// Print all registered routes
		for _, route := range router.Routes() {
			fmt.Printf("%-6s -> %s (%s)\n", route.Method, route.Path, route.Handler)
		}
	*/

	// Start the server (Gin handles everything)
	port := ":" + strconv.Itoa(app.UranusInstance.Config.Port)
	fmt.Printf("Uranus server is running on port %s\n", port)
	err = router.Run(port)
	if err != nil {
		fmt.Println("app server error:", err)
	}
}
