package main

import (
	"flag"
	"fmt"
	"strconv"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/sndcds/pluto"
	"github.com/sndcds/uranus/api"
	"github.com/sndcds/uranus/app"
)

func main() {
	fmt.Println("start")

	// Configuration
	configFileName := flag.String("config", "config.json", "Path to config file")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()
	fmt.Println("Config file:", *configFileName)

	var err error
	app.UranusInstance, err = app.Initialize(*configFileName)
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}

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

	// Public endpoints
	publicRoute := router.Group("/api")

	publicRoute.GET("/events", apiHandler.GetEvents)
	publicRoute.GET("/events/type-summary", apiHandler.GetEventTypeSummary)
	publicRoute.GET("/events/venue-summary", apiHandler.GetEventVenueSummary)

	publicRoute.GET("/event/:eventId/date/:dateId", apiHandler.GetEventByDateId)
	publicRoute.GET("/event/:eventId/date/:dateId/ics", apiHandler.GetEventDateICS)

	publicRoute.GET("/geojson/venues", apiHandler.GetGeojsonVenues)

	publicRoute.GET("/organizations", apiHandler.GetOrganizations)

	publicRoute.GET("/user/:userId/avatar/:size", apiHandler.GetUserAvatar)
	publicRoute.GET("/user/:userId/avatar", apiHandler.GetUserAvatar)

	publicRoute.GET("/event/types-genres-lookup", apiHandler.GetEventTypesAndGenres)

	publicRoute.GET("/choosable-venues", apiHandler.GetChoosableVenues)
	publicRoute.GET("/choosable-organizations", apiHandler.GetChoosableOrganizations)
	publicRoute.GET("/choosable-venues/organization/:organizationId", apiHandler.GetChoosableOrganizationVenues)
	publicRoute.GET("/choosable-space-types", apiHandler.GetSpaceTypes)
	publicRoute.GET("/choosable-spaces/venue/:venueId", apiHandler.GetChoosableVenueSpaces)
	publicRoute.GET("/choosable-event-types", apiHandler.GetChoosableEventTypes)
	publicRoute.GET("/choosable-event-genres/event-type/:id", apiHandler.GetChoosableEventGenres)
	publicRoute.GET("/choosable-states", apiHandler.GetChoosableStates)
	publicRoute.GET("/choosable-licenses", apiHandler.GetChoosableLicenses)
	publicRoute.GET("/choosable-legal-forms", apiHandler.GetChoosableLegalForms)
	publicRoute.GET("/choosable-countries", apiHandler.GetChoosableCountries)
	publicRoute.GET("/choosable-release-states", apiHandler.GetChoosableReleaseStates)
	publicRoute.GET("/choosable-languages", apiHandler.GetChoosableLanguages)
	publicRoute.GET("/choosable-url-types/event", apiHandler.GetChoosableEventUrlTypes)

	publicRoute.GET("/choosable-price-types", apiHandler.GetChoosablePriceTypes)
	publicRoute.GET("/choosable-currencies", apiHandler.GetChoosableCurrencies)
	publicRoute.GET("/choosable-event-ocassions", apiHandler.GetChoosableEventOccasions)

	publicRoute.GET("/accessibility/flags", apiHandler.GetAccessibilityFlags)

	publicRoute.GET("/organization/:organizationId", apiHandler.GetOrganization)

	// Inject app middleware into Pluto's image routes
	pluto.PlutoInstance.RegisterRoutes(publicRoute, app.JWTMiddleware)

	publicRoute.POST("/signup", apiHandler.Signup)
	publicRoute.POST("/activate", apiHandler.Activate)
	publicRoute.POST("/login", apiHandler.Login)
	publicRoute.POST("/forgot-password", apiHandler.ForgotPassword)
	publicRoute.POST("/reset-password", apiHandler.ResetPassword)

	// Authorized endpoints, user must be logged in
	adminRoute := router.Group("/api/admin", app.JWTMiddleware)

	adminRoute.POST("/refresh", apiHandler.Refresh)

	adminRoute.GET("/user/me", apiHandler.AdminGetUserProfile)
	adminRoute.PUT("/user/me", apiHandler.AdminUpdateUserProfile)
	adminRoute.PUT("/user/me/settings", apiHandler.AdminUpdateUserProfileSettings)
	adminRoute.POST("/user/me/avatar", apiHandler.AdminUploadUserAvatar)
	adminRoute.DELETE("/user/me/avatar", apiHandler.AdminDeleteUserAvatar)
	adminRoute.GET("/user/me/permissions", apiHandler.AdminUserPermissions)

	adminRoute.POST("/send-message", apiHandler.AdminSendMessage)
	adminRoute.GET("/messages", apiHandler.AdminGetMessages)

	adminRoute.GET("/todos", apiHandler.AdminGetTodos)
	adminRoute.GET("/todo/:todoId", apiHandler.AdminGetTodo)
	adminRoute.PUT("/todo", apiHandler.AdminUpsertTodo)
	adminRoute.DELETE("/todo/:todoId", apiHandler.AdminDeleteTodo)

	adminRoute.GET("/permission/list", apiHandler.AdminGetPermissionList)

	adminRoute.GET("/organization/:organizationId/member/:memberId/permissions", apiHandler.AdminGetOrganizationMemberPermissions)
	adminRoute.PUT("/organization/:organizationId/member/:memberId/permission", apiHandler.AdminUpdateOrganizationMemberPermission)

	adminRoute.GET("/user/event/notification", apiHandler.AdminGetUserEventNotification)

	adminRoute.GET("/choosable-organizations", apiHandler.AdminGetChoosableOrganizations)
	adminRoute.GET("/user/choosable-venues-spaces", apiHandler.AdminChoosableUserVenuesSpaces)

	adminRoute.GET("/organization/:organizationId", apiHandler.AdminGetOrganization)
	adminRoute.PUT("/organization", apiHandler.AdminUpsertOrganization)
	adminRoute.PUT("/organization/:organizationId", apiHandler.AdminUpsertOrganization)
	adminRoute.DELETE("/organization/:organizationId", apiHandler.AdminDeleteOrganization)

	adminRoute.GET("/organization/dashboard", apiHandler.AdminGetOrganizationDashboard)
	adminRoute.GET("/organization/:organizationId/venues", apiHandler.AdminGetOrganizationVenues)
	adminRoute.GET("/organization/:organizationId/events", apiHandler.AdminGetOrganizationEvents)

	adminRoute.GET("/organization/:organizationId/team", apiHandler.AdminGetOrganizationTeam)
	adminRoute.POST("/organization/:organizationId/team/invite", apiHandler.AdminOrganizationTeamInvite)
	adminRoute.DELETE("/organization/:organizationId/team/member/:memberId", apiHandler.AdminDeleteOrganizationTeamMember)
	adminRoute.POST("/organization/team/invite/accept", apiHandler.AdminOrganizationTeamInviteAccept)

	adminRoute.GET("/venue/:venueId", apiHandler.AdminGetVenue)
	adminRoute.PUT("/venue", apiHandler.AdminUpsertVenue)
	adminRoute.PUT("/venue/:venueId", apiHandler.AdminUpsertVenue)
	adminRoute.DELETE("/venue/:venueId", apiHandler.AdminDeleteVenue)

	adminRoute.GET("/space/:spaceId", apiHandler.AdminGetSpace)
	adminRoute.PUT("/space", apiHandler.AdminUpsertSpace)
	adminRoute.PUT("/space/:spaceId", apiHandler.AdminUpsertSpace)
	adminRoute.DELETE("/space/:spaceId", apiHandler.AdminDeleteSpace)

	adminRoute.GET("/event/:eventId", apiHandler.AdminGetEvent) // .....
	adminRoute.DELETE("/event/:eventId", apiHandler.AdminDeleteEvent)

	adminRoute.POST("/event/:eventId/date", apiHandler.AdminUpsertEventDate)
	adminRoute.PUT("/event/:eventId/date/:dateId", apiHandler.AdminUpsertEventDate)
	adminRoute.DELETE("/event/:eventId/date/:dateId", apiHandler.AdminDeleteEventDate)

	adminRoute.POST("/event/create", apiHandler.AdminCreateEvent)
	adminRoute.PUT("/event/:eventId/release-status", apiHandler.AdminUpdateEventReleaseStatus)
	adminRoute.PUT("/event/:eventId/header", apiHandler.AdminUpdateEventHeader)
	adminRoute.PUT("/event/:eventId/description", apiHandler.AdminUpdateEventDescription)
	adminRoute.PUT("/event/:eventId/summary", apiHandler.AdminUpdateEventSummary)
	adminRoute.PUT("/event/:eventId/types", apiHandler.AdminUpdateEventTypes)
	adminRoute.PUT("/event/:eventId/place", apiHandler.AdminUpdateEventPlace)
	adminRoute.PUT("/event/:eventId/link", apiHandler.AdminUpsertEventLink)
	adminRoute.PUT("/event/:eventId/link/:linkId", apiHandler.AdminUpsertEventLink)
	adminRoute.PUT("/event/:eventId/tags", apiHandler.AdminUpdateEventTags)
	adminRoute.PUT("/event/:eventId/languages", apiHandler.AdminUpdateEventLanguages)
	adminRoute.PUT("/event/:eventId/participation-infos", apiHandler.AdminUpdateEventParticipationInfos)

	adminRoute.GET("/event/:eventId/image/:imageIndex/meta", apiHandler.AdminGetImageMeta)
	adminRoute.POST("/event/:eventId/image/:imageIndex", apiHandler.AdminUpsertEventImage)
	adminRoute.DELETE("/event/:eventId/image", apiHandler.AdminDeleteEventMainImage)

	adminRoute.POST("/event/:eventId/teaser/image", apiHandler.AdminUpdateEventTeaserImage)

	fmt.Println("Gin mode:", gin.Mode())
	fmt.Println("Total routes:", len(router.Routes()))

	// Print all registered routes
	for _, route := range router.Routes() {
		fmt.Printf("%-6s -> %s (%s)\n", route.Method, route.Path, route.Handler)
	}

	// Start the server (Gin handles everything)
	port := ":" + strconv.Itoa(app.UranusInstance.Config.Port)
	fmt.Printf("Uranus server is running on port %s\n", port)
	err = router.Run(port)
	if err != nil {
		fmt.Println("app server error:", err)
	}
}
