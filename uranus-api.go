package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strconv"

	"html/template"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/pluto"
	"github.com/sndcds/uranus/api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/service"
)

func main() {
	configFileName := flag.String("config", "config.json", "Path to config file")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()

	grains_api.Init(grains_api.Config{
		ServiceName: "Uranus API",
		APIVersion:  "1.0",
		TimeFormat:  "", // leave empty to use default RFC3339
	})

	// TODO: Validate required properties!

	var err error
	app.UranusInstance, err = app.Initialize(*configFileName)
	if err != nil {
		log.Fatal(err)
	}

	err = app.UranusInstance.CheckAllDatabaseConsistency(context.Background())
	if err != nil {
		fmt.Println("Uranus database not consistent")
		fmt.Println(err.Error())
		panic(err)
	}
	app.UranusInstance.Log("CheckAllDatabaseConsistency succeeded")

	if *verbose {
		app.UranusInstance.Config.Verbose = true
	}

	// Accessibility Lookup

	accessibilityLookup := service.NewAccessibilityLookup()
	err = accessibilityLookup.Load(
		context.Background(),
		app.UranusInstance.MainDbPool,
		app.UranusInstance.Config.DbSchema,
	)
	if err != nil {

		log.Fatal(err)

	}

	//

	app.UranusInstance.Config.Print()

	eventTemplate := template.Must(template.ParseFiles("templates/event.html"))

	apiHandler := &api.ApiHandler{
		Config:        &app.UranusInstance.Config,
		DbPool:        app.UranusInstance.MainDbPool,
		DbSchema:      app.UranusInstance.Config.DbSchema,
		EventTemplate: eventTemplate,
		Accessibility: accessibilityLookup,
	}

	_, err = pluto.Initialize(*configFileName, app.UranusInstance.MainDbPool, true)
	if err != nil {
		panic(err)
	}

	// Create a Gin router

	gin.SetMode(gin.ReleaseMode)
	router := gin.New() // Use `Default()` for built-in logging and recovery
	router.SetTrustedProxies([]string{"127.0.0.1", "::1"})

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

	//
	// Event endpoints
	//

	eventRoute := router.Group("/event")
	eventRoute.GET("/:eventUuid", apiHandler.InternalTest)
	eventRoute.GET("/:eventUuid/date/:dateIdentifier", apiHandler.InternalTest)

	//
	// Public endpoints
	//

	publicRoute := router.Group("/api")

	publicRoute.GET("/health", apiHandler.GetHealth)

	publicRoute.GET("/event/release-status-i18n", apiHandler.GetEventReleaseStatusI18n)

	publicRoute.GET("/events", apiHandler.GetEvents)
	publicRoute.GET("/events/week", apiHandler.GetEventsWeek)
	publicRoute.GET("/events/type-summary", apiHandler.GetEventTypeSummary)
	publicRoute.GET("/events/venue-summary", apiHandler.GetEventVenueSummary) // TODO: check!
	publicRoute.GET("/events/geojson", apiHandler.GetEventsGeoJSON)           // TODO: Reduce data

	publicRoute.GET("/event/:eventUuid", apiHandler.GetEvent)
	publicRoute.GET("/event/:eventUuid/date/:dateIdentifier", apiHandler.GetEventByDate)
	publicRoute.GET("/event/:eventUuid/date/:dateIdentifier/ics", apiHandler.GetEventDateICS)

	publicRoute.GET("/portal/:uuid", apiHandler.GetPortal)

	publicRoute.GET("/venues", apiHandler.GetVenues)
	publicRoute.GET("/venues/type-summary", apiHandler.GetVenuesSummary)
	publicRoute.GET("/venues/geojson", apiHandler.GetVenuesGeoJSON)

	publicRoute.GET("/org/:orgUuid", apiHandler.GetOrg)
	publicRoute.GET("/orgs", apiHandler.GetOrgs)

	publicRoute.GET("/venue/:venueUuid", apiHandler.GetVenue)
	publicRoute.GET("/venue/slug/:slug/uuid", apiHandler.GetVenueUuidBySlug)
	publicRoute.GET("/venue/:venueUuid/space/:spaceUuid/label", apiHandler.GetVenueSpaceLabel)

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
	publicRoute.GET("/choosable-languages", apiHandler.GetChoosableLanguages)
	publicRoute.GET("/choosable-price-types", apiHandler.GetChoosablePriceTypes)         // TODO: check!
	publicRoute.GET("/choosable-currencies", apiHandler.GetChoosableCurrencies)          // TODO: check!
	publicRoute.GET("/choosable-event-ocassions", apiHandler.GetChoosableEventOccasions) // TODO: check!

	publicRoute.GET("/choosable-venues", apiHandler.GetChoosableVenues)
	publicRoute.GET("/choosable-orgs", apiHandler.GetChoosableOrgs)
	publicRoute.GET("/choosable-venues/org/:orgUuid", apiHandler.GetChoosableOrgVenues)           // TODO: check!
	publicRoute.GET("/choosable-spaces/venue/:venueUuid", apiHandler.GetChoosableVenueSpaces)     // TODO: check!
	publicRoute.GET("/choosable-event-genres/event-type/:id", apiHandler.GetChoosableEventGenres) // TODO: check!

	publicRoute.GET("/accessibility/flags", apiHandler.GetAccessibilityFlags) // TODO: check!

	// Inject app middleware into Pluto's image routes
	pluto.PlutoInstance.RegisterRoutes(publicRoute, app.JWTMiddleware) // TODO: check!

	publicRoute.POST("/signup", apiHandler.Signup)
	publicRoute.POST("/login", apiHandler.Login)
	publicRoute.POST("/activate", apiHandler.Activate)
	publicRoute.POST("/forgot-password", apiHandler.ForgotPassword)
	publicRoute.POST("/reset-password", apiHandler.ResetPassword)

	publicRoute.GET("/sitemap", apiHandler.Sitemap)

	publicRoute.GET("/geo/countries", apiHandler.GetGeoCountries)
	publicRoute.GET("/geo/countries/:country_slug/states", apiHandler.GetGeoCountryStates)
	publicRoute.GET("/geo/countries/:country_slug/states/:state_slug", apiHandler.GetGeoStateRegions)

	//
	// Authorized endpoints, user must be logged in
	//

	adminRoute := router.Group("/api/admin", app.JWTMiddleware)
	adminRoute.GET("/event/:eventUuid/date/:dateIdentifier", apiHandler.GetEventByDate) // TODO: Permission check
	adminRoute.GET("/permissions/list", apiHandler.AdminGetPermissionsList)             // TODO: Permission check
	adminRoute.POST("/refresh", apiHandler.Refresh)                                     // TODO: Permission check

	// User
	adminRoute.GET("/user/profile", apiHandler.AdminGetUserProfile)             // TODO: Permission check
	adminRoute.PUT("/user/profile", apiHandler.AdminUpdateUserProfile)          // TODO: Permission check
	adminRoute.PUT("/user/settings", apiHandler.AdminUpdateUserProfileSettings) // TODO: Permission check
	adminRoute.POST("/user/avatar", apiHandler.AdminUploadUserAvatar)           // TODO: Permission check
	adminRoute.DELETE("/user/avatar", apiHandler.AdminDeleteUserAvatar)         // TODO: Permission check

	adminRoute.GET("/user/todos", apiHandler.AdminUserGetTodos)         // TODO: Permission check
	adminRoute.GET("/user/todo/:todoId", apiHandler.AdminGetTodo)       // TODO: Permission check
	adminRoute.PUT("/user/todo", apiHandler.AdminUpsertTodo)            // TODO: Permission check
	adminRoute.DELETE("/user/todo/:todoId", apiHandler.AdminDeleteTodo) // TODO: Permission check

	adminRoute.GET("/user/messages", apiHandler.AdminGetMessages)      // TODO: Permission check
	adminRoute.POST("/user/send-message", apiHandler.AdminSendMessage) // TODO: Permission check

	adminRoute.GET("/user/org/:orgUuid/event/notifications", apiHandler.AdminGetUserEventNotifications)
	adminRoute.GET("/user/choosable-orgs", apiHandler.AdminGetChoosableOrgs)                    // TODO: Permission check
	adminRoute.GET("/user/choosable-event-venues", apiHandler.AdminGetChoosableUserEventVenues) // TODO: Unused, can be removed!

	// Organization
	adminRoute.GET("/org/:orgUuid/member/:memberUuid/permissions", apiHandler.AdminGetOrgMemberPermissions)    // TODO: Permission check
	adminRoute.PUT("/org/:orgUuid/member/:memberUuid/permissions", apiHandler.AdminUpdateOrgMemberPermissions) // TODO: Permission check

	adminRoute.POST("/org/create", apiHandler.AdminCreateOrg)          // TODO: Permission check
	adminRoute.GET("/org/:orgUuid", apiHandler.AdminGetOrg)            // TODO: Permission check
	adminRoute.PUT("/org/:orgUuid/fields", apiHandler.UpdateOrgFields) // TODO: Permission check
	adminRoute.DELETE("/org/:orgUuid", apiHandler.AdminDeleteOrg)      // TODO: Permission check

	adminRoute.GET("/org/list", apiHandler.AdminGetOrgList)                // TODO: Permission check
	adminRoute.GET("/org/:orgUuid/venues", apiHandler.AdminGetOrgVenues)   // TODO: Permission check
	adminRoute.GET("/org/:orgUuid/events", apiHandler.AdminGetOrgEvents)   // TODO: Permission check
	adminRoute.GET("/org/:orgUuid/portals", apiHandler.AdminGetOrgPortals) // TODO: Permission check

	adminRoute.GET("/org/:orgUuid/team", apiHandler.AdminGetOrgTeam)                              // TODO: Permission check
	adminRoute.POST("/org/:orgUuid/team/invite", apiHandler.AdminOrgTeamInvite)                   // TODO: Permission check
	adminRoute.POST("/org/team/invite/accept", apiHandler.AdminOrgTeamInviteAccept)               // TODO: Permission check
	adminRoute.DELETE("/org/:orgUuid/team/member/:memberId", apiHandler.AdminDeleteOrgTeamMember) // TODO: Permission check
	adminRoute.GET("/org/:orgUuid/choosable-venues", apiHandler.AdminGetOrgChoosableVenues)

	// Partner
	adminRoute.GET("/org/:orgUuid/partnership-connections", apiHandler.AdminOrgPartnershipConnections)           // TODO: Permission check
	adminRoute.GET("/org/partnership-connections-by-user", apiHandler.AdminOrgPartnershipConnectionsByUser)      // TODO: Permission check
	adminRoute.GET("/org/:orgUuid/partner/grants", apiHandler.AdminGetOrgPartnerGrants)                          // TODO: Permission check
	adminRoute.GET("/org/:orgUuid/partner/requests", apiHandler.AdminGetOrgPartnerRequest)                       // TODO: Permission check
	adminRoute.POST("/org/:orgUuid/partner/:partnerUuid/grants", apiHandler.AdminUpdateOrgPartnerGrants)         // TODO: Permission check
	adminRoute.POST("/org/:orgUuid/partner/request", apiHandler.AdminInsertOrgPartnerRequest)                    // TODO: Permission check
	adminRoute.POST("/org/:orgUuid/partner/request/:partnerUuid/accept", apiHandler.AdminInsertOrgPartnerAccept) // TODO: Permission check
	adminRoute.POST("/org/:orgUuid/partner/request/:partnerUuid/reject", apiHandler.AdminOrgPartnerReject)       // TODO: Permission check

	// Venue
	adminRoute.GET("/venue/:venueUuid", apiHandler.AdminGetVenue) // TODO: Permission check
	adminRoute.POST("/venue/create", apiHandler.AdminCreateVenue) // TODO: Permission check
	// adminRoute.PUT("/venue", apiHandler.AdminUpsertVenue) // TODO: refactor to be create with complete data set
	adminRoute.PUT("/venue/:venueUuid/fields", apiHandler.AdminUpdateVenueFields) // TODO: Permission check
	adminRoute.DELETE("/venue/:venueUuid", apiHandler.AdminDeleteVenue)           // TODO: Permission check

	// Space
	adminRoute.GET("/space/:spaceUuid", apiHandler.AdminGetSpace) // Permission check ok
	adminRoute.POST("/space/create", apiHandler.AdminCreateSpace) // TODO: Permission check
	// adminRoute.PUT("/space", apiHandler.AdminUpsertSpace) // TODO: refactor to be create with complete data set
	adminRoute.PUT("/space/:spaceUuid/fields", apiHandler.AdminUpdateSpaceFields) // Permission check ok
	adminRoute.DELETE("/space/:spaceUuid", apiHandler.AdminDeleteSpace)           // Permission check ok

	// Event
	adminRoute.GET("/event/:eventUuid", apiHandler.AdminGetEvent)                          // TODO: Permission check
	adminRoute.POST("/event/:eventUuid/date", apiHandler.AdminUpsertEventDate)             // TODO: Permission check
	adminRoute.PUT("/event/:eventUuid/date/:dateUuid", apiHandler.AdminUpsertEventDate)    // TODO: Permission check
	adminRoute.DELETE("/event/:eventUuid", apiHandler.AdminDeleteEvent)                    // Permission check ok
	adminRoute.DELETE("/event/:eventUuid/date/:dateUuid", apiHandler.AdminDeleteEventDate) // Permission check ok

	adminRoute.POST("/event/initial", apiHandler.AdminInitialEvent)                     // TODO: Permission check
	adminRoute.POST("/event/create", apiHandler.AdminCreateEvent)                       // TODO: Permission check
	adminRoute.PUT("/event/:eventUuid/dates", apiHandler.AdminUpdateEventDates)         // TODO: Permission check
	adminRoute.PUT("/event/:eventUuid/types", apiHandler.AdminUpdateEventTypes)         // TODO: Permission check
	adminRoute.PUT("/event/:eventUuid/languages", apiHandler.AdminUpdateEventLanguages) // TODO: Permission check
	adminRoute.PUT("/event/:eventUuid/links", apiHandler.AdminUpdateEventLinks)         // TODO: Permission check
	adminRoute.PUT("/event/:eventUuid/venue", apiHandler.AdminUpdateEventVenue)         // TODO: Permission check
	adminRoute.PUT("/event/:eventUuid/fields", apiHandler.AdminUpdateEventFields)       // TODO: Permission check

	adminRoute.PUT("/event/:eventUuid/release-status", apiHandler.AdminUpdateEventReleaseStatus)           // TODO: Permission check
	adminRoute.PUT("/event/:eventUuid/header", apiHandler.AdminUpdateEventHeader)                          // TODO: Permission check
	adminRoute.PUT("/event/:eventUuid/description", apiHandler.AdminUpdateEventDescription)                // TODO: Permission check
	adminRoute.PUT("/event/:eventUuid/summary", apiHandler.AdminUpdateEventSummary)                        // TODO: Permission check
	adminRoute.PUT("/event/:eventUuid/participation-infos", apiHandler.AdminUpdateEventParticipationInfos) // TODO: Permission check

	// Portal
	adminRoute.GET("/portal/:portalUuid", apiHandler.AdminGetPortal)                 // TODO: Permission check
	adminRoute.POST("/portal/create", apiHandler.AdminCreatePortal)                  // TODO: Permission check
	adminRoute.PUT("/portal/:portalUuid/fields", apiHandler.AdminUpdatePortalFields) // TODO: Permission check
	adminRoute.PUT("/portal/:portalUuid/filter", apiHandler.AdminUpdatePortalFilter) // TODO: Permission check
	adminRoute.PUT("/portal/:portalUuid/style", apiHandler.AdminUpdatePortalStyle)   // TODO: Permission check
	adminRoute.PUT("/portal/:portalUuid/header", apiHandler.AdminUpdatePortalHeader) // TODO: Permission check
	adminRoute.PUT("/portal/:portalUuid/footer", apiHandler.AdminUpdatePortalFooter) // TODO: Permission check

	// Favorites
	adminRoute.GET("/org/:orgUuid/favorite-lists", apiHandler.AdminGetFavoriteLists)               // TODO: Permission check
	adminRoute.POST("/favorite-list/create", apiHandler.AdminCreateFavoriteList)                   // TODO: Permission check
	adminRoute.POST("/favorite-list/toggle-event-date", apiHandler.AdminToggleFavoriteEventDate)   // TODO: Permission check
	adminRoute.POST("/favorite-list/check-event-date", apiHandler.AdminCheckFavoriteListEventDate) // TODO: Permission check

	// Pluto Image
	adminRoute.POST("/image/:context/:contextUuid/:identifier", apiHandler.AdminUpsertPlutoImage)   // TODO: Permission check
	adminRoute.DELETE("/image/:context/:contextUuid/:identifier", apiHandler.AdminDeletePlutoImage) // TODO: Permission check

	//
	// Internal endpoints, callable only from localhost
	//

	internalRoute := router.Group("/api/internal", app.LocalhostOnlyMiddleware)

	internalRoute.POST("/event/:eventUuid/refresh-projections", apiHandler.AdminRefreshEventProjections) // TODO: Check!
	internalRoute.GET("/image/cleanup", apiHandler.InternalCleanupImages)                                // TODO: Check!
	internalRoute.GET("/test", apiHandler.InternalTest)                                                  // TODO: Check!
	internalRoute.GET("/migrate-venues", apiHandler.InternalMigrateVenues)                               // TODO: Check!

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
