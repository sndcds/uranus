package main

import (
	"flag"
	"fmt"
	"strconv"

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

	// Authorized endpoints, user must be logged in
	adminRoute := router.Group("/api/admin")

	adminRoute.POST("/signup", apiHandler.Signup)
	adminRoute.POST("/activate", apiHandler.Activate)
	adminRoute.POST("/login", apiHandler.Login)
	adminRoute.POST("/refresh", apiHandler.Refresh)
	adminRoute.POST("/forgot-password", apiHandler.ForgotPassword)
	adminRoute.POST("/reset-password", apiHandler.ResetPassword)

	adminRoute.GET("/user/:userId", app.JWTMiddleware, apiHandler.AdminGetUser)

	adminRoute.POST("/send-message", app.JWTMiddleware, apiHandler.AdminSendMessage)
	adminRoute.GET("/messages", app.JWTMiddleware, apiHandler.AdminGetMessages)

	adminRoute.GET("/todos", app.JWTMiddleware, apiHandler.AdminGetTodos)
	adminRoute.GET("/todo/:todoId", app.JWTMiddleware, apiHandler.AdminGetTodo)
	adminRoute.PUT("/todo", app.JWTMiddleware, apiHandler.AdminUpsertTodo)
	adminRoute.DELETE("/todo/:todoId", app.JWTMiddleware, apiHandler.AdminDeleteTodo)

	adminRoute.GET("/user/me", app.JWTMiddleware, apiHandler.AdminGetUserProfile)
	adminRoute.PUT("/user/me", app.JWTMiddleware, apiHandler.AdminUpdateUserProfile)
	adminRoute.PUT("/user/me/settings", app.JWTMiddleware, apiHandler.AdminUpdateUserProfileSettings)
	adminRoute.POST("/user/me/avatar", app.JWTMiddleware, apiHandler.AdminUploadUserAvatar)
	adminRoute.DELETE("/user/me/avatar", app.JWTMiddleware, apiHandler.AdminDeleteUserAvatar)
	adminRoute.GET("/user/me/permissions", app.JWTMiddleware, apiHandler.AdminUserPermissions)

	adminRoute.GET("/permission/list", app.JWTMiddleware, apiHandler.AdminGetPermissionList)

	adminRoute.GET("/organization/:organizationId/member/:memberId/permissions", app.JWTMiddleware, apiHandler.AdminGetOrganizationMemberPermissions)
	adminRoute.PUT("/organization/:organizationId/member/:memberId/permission", app.JWTMiddleware, apiHandler.AdminUpdateOrganizationMemberPermission)

	adminRoute.GET("/user/event/notification", app.JWTMiddleware, apiHandler.AdminGetUserEventNotification)

	adminRoute.GET("/choosable-organizations", app.JWTMiddleware, apiHandler.AdminGetChoosableOrganizations)
	adminRoute.GET("/user/choosable-event-organizations/organiationr/:organizationId", app.JWTMiddleware, apiHandler.AdminChoosableUserEventOrganizations)
	adminRoute.GET("/user/choosable-venues-spaces", app.JWTMiddleware, apiHandler.AdminChoosableUserVenuesSpaces)
	adminRoute.GET("/user/choosable-event-venues", app.JWTMiddleware, apiHandler.AdminChoosableUserEventVenues) // TODO: Remove!

	adminRoute.GET("/organization/:organizationId", app.JWTMiddleware, apiHandler.AdminGetOrganization)
	adminRoute.PUT("/organization", app.JWTMiddleware, apiHandler.AdminUpsertOrganization)
	adminRoute.PUT("/organization/:organizationId", app.JWTMiddleware, apiHandler.AdminUpsertOrganization)
	adminRoute.DELETE("/organization/:organizationId", app.JWTMiddleware, apiHandler.AdminDeleteOrganization)

	adminRoute.GET("/organization/dashboard", app.JWTMiddleware, apiHandler.AdminGetOrganizationDashboard)
	adminRoute.GET("/organization/:organizationId/venues", app.JWTMiddleware, apiHandler.AdminGetOrganizationVenues)
	adminRoute.GET("/organization/:organizationId/events", app.JWTMiddleware, apiHandler.AdminGetOrganizationEvents)

	adminRoute.GET("/organization/:organizationId/team", app.JWTMiddleware, apiHandler.AdminGetOrganizationTeam)
	adminRoute.POST("/organization/:organizationId/team/invite", app.JWTMiddleware, apiHandler.AdminOrganizationTeamInvite)
	adminRoute.DELETE("/organization/:organizationId/team/member/:memberId", app.JWTMiddleware, apiHandler.AdminDeleteOrganizationTeamMember)
	adminRoute.POST("/organization/team/invite/accept", apiHandler.AdminOrganizationTeamInviteAccept)

	adminRoute.GET("/venue/:venueId", app.JWTMiddleware, apiHandler.AdminGetVenue)
	adminRoute.PUT("/venue", app.JWTMiddleware, apiHandler.AdminUpsertVenue)
	adminRoute.PUT("/venue/:venueId", app.JWTMiddleware, apiHandler.AdminUpsertVenue)
	adminRoute.DELETE("/venue/:venueId", app.JWTMiddleware, apiHandler.AdminDeleteVenue)

	adminRoute.GET("/space/:spaceId", app.JWTMiddleware, apiHandler.AdminGetSpace)
	adminRoute.PUT("/space", app.JWTMiddleware, apiHandler.AdminUpsertSpace)
	adminRoute.PUT("/space/:spaceId", app.JWTMiddleware, apiHandler.AdminUpsertSpace)
	adminRoute.DELETE("/space/:spaceId", app.JWTMiddleware, apiHandler.AdminDeleteSpace)

	adminRoute.GET("/event/:eventId", app.JWTMiddleware, apiHandler.AdminGetEvent)
	adminRoute.DELETE("/event/:eventId", app.JWTMiddleware, apiHandler.AdminDeleteEvent)

	adminRoute.POST("/event/:eventId/date", app.JWTMiddleware, apiHandler.AdminUpsertEventDate)
	adminRoute.PUT("/event/:eventId/date/:dateId", app.JWTMiddleware, apiHandler.AdminUpsertEventDate)
	adminRoute.DELETE("/event/:eventId/date/:dateId", app.JWTMiddleware, apiHandler.AdminDeleteEventDate)

	adminRoute.POST("/event/create", app.JWTMiddleware, apiHandler.AdminCreateEvent)
	adminRoute.PUT("/event/:eventId/release-status", app.JWTMiddleware, apiHandler.AdminUpdateEventReleaseStatus)
	adminRoute.PUT("/event/:eventId/header", app.JWTMiddleware, apiHandler.AdminUpdateEventHeader)
	adminRoute.PUT("/event/:eventId/description", app.JWTMiddleware, apiHandler.AdminUpdateEventDescription)
	adminRoute.PUT("/event/:eventId/teaser", app.JWTMiddleware, apiHandler.AdminUpdateEventTeaser)
	adminRoute.PUT("/event/:eventId/types", app.JWTMiddleware, apiHandler.AdminUpdateEventTypes)
	adminRoute.PUT("/event/:eventId/place", app.JWTMiddleware, apiHandler.AdminUpdateEventPlace)
	adminRoute.PUT("/event/:eventId/link", app.JWTMiddleware, apiHandler.AdminUpsertEventLink)
	adminRoute.PUT("/event/:eventId/link/:linkId", app.JWTMiddleware, apiHandler.AdminUpsertEventLink)
	adminRoute.PUT("/event/:eventId/tags", app.JWTMiddleware, apiHandler.AdminUpdateEventTags)
	adminRoute.PUT("/event/:eventId/languages", app.JWTMiddleware, apiHandler.AdminUpdateEventLanguages)
	adminRoute.PUT("/event/:eventId/participation-infos", app.JWTMiddleware, apiHandler.AdminUpdateEventParticipationInfos)

	adminRoute.GET("/event/:eventId/image/:imageIndex/meta", app.JWTMiddleware, apiHandler.AdminGetImageMeta)
	adminRoute.POST("/event/:eventId/image/:imageIndex", app.JWTMiddleware, apiHandler.AdminUpsertEventImage)
	adminRoute.DELETE("/event/:eventId/image", app.JWTMiddleware, apiHandler.AdminDeleteEventMainImage)

	adminRoute.POST("/event/:eventId/teaser/image", app.JWTMiddleware, apiHandler.AdminUpdateEventTeaserImage)

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
