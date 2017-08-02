package main

import (
	"os"
	"os/user"
	"runtime"
	"time"

	"context"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"

	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/notification"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
)

func main() {

	config := configuration.Get()
	// Initialized developer mode flag and log level for the logger
	log.InitializeLogger(config.IsLogJSON(), config.GetLogLevel())
	log.Logger().Debugf("Config: %v\n", config.String()) // will print all config settings including DB credentials, so make sure this is only visible in DEBUG mode

	printUserInfo()

	var db *gorm.DB
	for {
		db, err := gorm.Open("postgres", config.GetPostgresConfigString())
		if err != nil {
			db.Close()
			log.Logger().Errorf("ERROR: Unable to open connection to database %v", err)
			log.Logger().Infof("Retrying to connect in %v...", config.GetPostgresConnectionRetrySleep())
			time.Sleep(config.GetPostgresConnectionRetrySleep())
		} else {
			defer db.Close()
			break
		}
	}
	log.Logger().Infof("DB connection: %v...", db)

	if config.IsPostgresDeveloperModeEnabled() && log.IsDebug() {
		db = db.Debug()
	}

	if config.GetPostgresConnectionMaxIdle() > 0 {
		log.Logger().Infof("Configured connection pool max idle %v", config.GetPostgresConnectionMaxIdle())
		db.DB().SetMaxIdleConns(config.GetPostgresConnectionMaxIdle())
	}
	if config.GetPostgresConnectionMaxOpen() > 0 {
		log.Logger().Infof("Configured connection pool max open %v", config.GetPostgresConnectionMaxOpen())
		db.DB().SetMaxOpenConns(config.GetPostgresConnectionMaxOpen())
	}

	// Set the database transaction timeout
	application.SetDatabaseTransactionTimeout(config.GetPostgresTransactionTimeout())
	// Migrate the schema
	err := migration.Migrate(db.DB(), config.GetPostgresDatabase())
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"err": err,
		}, "failed migration")
	}

	// Nothing to here except exit, since the migration is already performed.
	if config.MigrateDB() {
		os.Exit(0)
	}

	// Make sure the database is populated with the correct types (e.g. bug etc.)
	if config.GetPopulateCommonTypes() {
		ctx := migration.NewMigrationContext(context.Background())

		if err := gormsupport.Transactional(db, func(tx *gorm.DB) error {
			return migration.PopulateCommonTypes(ctx, tx, workitem.NewWorkItemTypeRepository(tx))
		}); err != nil {
			log.Panic(ctx, map[string]interface{}{
				"err": err,
			}, "failed to populate common types")
		}
		if err := gormsupport.Transactional(db, func(tx *gorm.DB) error {
			return migration.BootstrapWorkItemLinking(ctx, link.NewWorkItemLinkCategoryRepository(tx), space.NewRepository(tx), link.NewWorkItemLinkTypeRepository(tx))
		}); err != nil {
			log.Panic(ctx, map[string]interface{}{
				"err": err,
			}, "failed to bootstap work item linking")
		}
	}

	// // Create service
	// service := goa.New("wit")

	// // Mount middleware
	// service.Use(middleware.RequestID())
	// // Use our own log request to inject identity id and modify other properties
	// service.Use(log.LogRequest(config.IsPostgresDeveloperModeEnabled()))
	// service.Use(gzip.Middleware(9))
	// service.Use(jsonapi.ErrorHandler(service, true))
	// service.Use(middleware.Recover())

	// service.WithLogger(goalogrus.New(log.Logger()))

	// publicKey, err := token.ParsePublicKey(config.GetTokenPublicKey())
	// if err != nil {
	// 	log.Panic(nil, map[string]interface{}{
	// 		"err": err,
	// 	}, "failed to parse public token")
	// }

	// Setup Account/Login/Security
	// identityRepository := account.NewIdentityRepository(db)
	// userRepository := account.NewUserRepository(db)

	var notificationChannel notification.Channel = &notification.DevNullChannel{}
	if configuration.GetNotificationServiceURL() != "" {
		log.Logger().Infof("Enabling Notification service %v", configuration.GetNotificationServiceURL())
		channel, err := notification.NewServiceChannel(configuration)
		if err != nil {
			log.Panic(nil, map[string]interface{}{
				"err": err,
				"url": configuration.GetNotificationServiceURL(),
			}, "failed to parse notification service url")
		}
		notificationChannel = channel
	}

	appDB := gormapplication.NewGormDB(db)

	// tokenManager := token.NewManager(publicKey)
	// // Middleware that extracts and stores the token in the context
	// jwtMiddlewareTokenContext := witmiddleware.TokenContext(publicKey, nil, app.NewJWTSecurity())
	// service.Use(jwtMiddlewareTokenContext)

	// service.Use(login.InjectTokenManager(tokenManager))
	// service.Use(log.LogRequest(configuration.IsPostgresDeveloperModeEnabled()))
	// app.UseJWTMiddleware(service, goajwt.New(publicKey, nil, app.NewJWTSecurity()))

	// spaceAuthzService := authz.NewAuthzService(configuration, appDB)
	// service.Use(authz.InjectAuthzService(spaceAuthzService))

	// loginService := login.NewKeycloakOAuthProvider(identityRepository, userRepository, tokenManager, appDB)
	// loginCtrl := controller.NewLoginController(service, loginService, tokenManager, configuration, identityRepository)
	// app.MountLoginController(service, loginCtrl)

	// logoutCtrl := controller.NewLogoutController(service, &login.KeycloakLogoutService{}, configuration)
	// app.MountLogoutController(service, logoutCtrl)

	// // Mount "status" controller
	// statusCtrl := controller.NewStatusController(service, db)
	// app.MountStatusController(service, statusCtrl)

	// // Mount "workitem" controller
	// //workitemCtrl := controller.NewWorkitemController(service, appDB, configuration)
	// workitemCtrl := controller.NewNotifyingWorkitemController(service, appDB, notificationChannel, configuration)
	// app.MountWorkitemController(service, workitemCtrl)

	// // Mount "named workitem" controller
	// namedWorkitemsCtrl := controller.NewNamedWorkItemsController(service, appDB)
	// app.MountNamedWorkItemsController(service, namedWorkitemsCtrl)

	// // Mount "workitems" controller
	// //workitemsCtrl := controller.NewWorkitemsController(service, appDB, configuration)
	// workitemsCtrl := controller.NewNotifyingWorkitemsController(service, appDB, notificationChannel, configuration)
	// app.MountWorkitemsController(service, workitemsCtrl)

	// // Mount "workitemtype" controller
	// workitemtypeCtrl := controller.NewWorkitemtypeController(service, appDB, configuration)
	// app.MountWorkitemtypeController(service, workitemtypeCtrl)

	// // Mount "work item link category" controller
	// workItemLinkCategoryCtrl := controller.NewWorkItemLinkCategoryController(service, appDB)
	// app.MountWorkItemLinkCategoryController(service, workItemLinkCategoryCtrl)

	// // Mount "work item link type" controller
	// workItemLinkTypeCtrl := controller.NewWorkItemLinkTypeController(service, appDB, configuration)
	// app.MountWorkItemLinkTypeController(service, workItemLinkTypeCtrl)

	// // Mount "render" controller
	// renderCtrl := controller.NewRenderController(service)
	// app.MountRenderController(service, renderCtrl)

	// // Mount "areas" controller
	// areaCtrl := controller.NewAreaController(service, appDB, configuration)
	// app.MountAreaController(service, areaCtrl)

	// Mount "work item comments" controller
	// workItemCommentsCtrl := controller.NewNotifyingWorkItemCommentsController(service, appDB, notificationChannel, configuration)
	// app.MountWorkItemCommentsController(service, workItemCommentsCtrl)

	// Mount "space area" controller
	// spaceAreaCtrl := controller.NewSpaceAreasController(service, appDB, configuration)
	// app.MountSpaceAreasController(service, spaceAreaCtrl)

	// filterCtrl := controller.NewFilterController(service, configuration)
	// app.MountFilterController(service, filterCtrl)

	// // Mount "namedspaces" controller
	// namedSpacesCtrl := controller.NewNamedspacesController(service, appDB)
	// app.MountNamedspacesController(service, namedSpacesCtrl)

	// Mount "comments" controller
	// commentsCtrl := controller.NewNotifyingCommentsController(service, appDB, notificationChannel, configuration)
	// app.MountCommentsController(service, commentsCtrl)

	// // Mount "plannerBacklog" controller
	// plannerBacklogCtrl := controller.NewPlannerBacklogController(service, appDB, configuration)
	// app.MountPlannerBacklogController(service, plannerBacklogCtrl)

	// // Mount "codebase" controller
	// codebaseCtrl := controller.NewCodebaseController(service, appDB, configuration)
	// app.MountCodebaseController(service, codebaseCtrl)

	// // Mount "spacecodebases" controller
	// spaceCodebaseCtrl := controller.NewSpaceCodebasesController(service, appDB)
	// app.MountSpaceCodebasesController(service, spaceCodebaseCtrl)

	// // Mount "collaborators" controller
	// collaboratorsCtrl := controller.NewCollaboratorsController(service, appDB, configuration, auth.NewKeycloakPolicyManager(configuration))
	// app.MountCollaboratorsController(service, collaboratorsCtrl)

	log.Logger().Infoln("Git Commit SHA: ", controller.Commit)
	log.Logger().Infoln("UTC Build Time: ", controller.BuildTime)
	log.Logger().Infoln("UTC Start Time: ", controller.StartTime)
	log.Logger().Infoln("Dev mode:       ", config.IsPostgresDeveloperModeEnabled())
	log.Logger().Infoln("GOMAXPROCS:     ", runtime.GOMAXPROCS(-1))
	log.Logger().Infoln("NumCPU:         ", runtime.NumCPU())

	// http.Handle("/api/", service.Mux)
	// http.Handle("/", http.FileServer(assetFS()))
	// http.Handle("/favicon.ico", http.NotFoundHandler())

	// // Start http
	// if err := http.ListenAndServe(config.GetHTTPAddress(), nil); err != nil {
	// 	log.Error(nil, map[string]interface{}{
	// 		"addr": config.GetHTTPAddress(),
	// 		"err":  err,
	// 	}, "unable to connect to server")
	// 	service.LogError("startup", "err", err)
	// }

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})
	// api := api2go.NewAPIWithRouting(
	// 	"api",
	// 	api2go.NewStaticResolver("/"),
	// 	gingonicsupport.New(r),
	// )
	// api.AddResource(model.Space{}, resource.NewSpaceResource(appDB))
	spaceResource := resource.NewSpaceResource(appDB)
	r.GET("/api/spaces/:spaceID", spaceResource.GetByID)
	r.Run(config.GetHTTPAddress())
}

func printUserInfo() {
	u, err := user.Current()
	if err != nil {
		log.Warn(nil, map[string]interface{}{
			"err": err,
		}, "failed to get current user")
	} else {
		log.Info(nil, map[string]interface{}{
			"username": u.Username,
			"uuid":     u.Uid,
		}, "Running as user name '%s' with UID %s.", u.Username, u.Uid)
		g, err := user.LookupGroupId(u.Gid)
		if err != nil {
			log.Warn(nil, map[string]interface{}{
				"err": err,
			}, "failed to lookup group")
		} else {
			log.Info(nil, map[string]interface{}{
				"groupname": g.Name,
				"gid":       g.Gid,
			}, "Running as as group '%s' with GID %s.", g.Name, g.Gid)
		}
	}

}
