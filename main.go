package main

import (
	"flag"
	"net/http"
	"os"
	"os/user"
	"runtime"
	"time"

	"context"

	"github.com/gin-gonic/gin"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/middleware"
	"github.com/goadesign/goa/middleware/gzip"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/api"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/controller"
	witmiddleware "github.com/fabric8-services/fabric8-wit/goamiddleware"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/notification"
	"github.com/fabric8-services/fabric8-wit/remoteworkitem"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/space/authz"
	"github.com/fabric8-services/fabric8-wit/token"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	goalogrus "github.com/goadesign/goa/logging/logrus"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	_ "github.com/lib/pq"
)

func main() {
	flag.Parse() // moved here because to avoid preventing execution of tests with the ginkgo tool. See https://github.com/onsi/ginkgo/issues/296#issuecomment-249924522
	config := configuration.LoadDefault()
	// Initialized developer mode flag and log level for the logger
	log.InitializeLogger(config.IsLogJSON(), config.GetLogLevel())
	log.Logger().Debugf("Config: %v\n", config.String()) // will print all config settings including DB credentials, so make sure this is only visible in DEBUG mode

	printUserInfo()

	db := connectToDB(config)
	defer db.Close()

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

	// Create service
	service := goa.New("wit")

	// Mount middleware
	service.Use(middleware.RequestID())
	// Use our own log request to inject identity id and modify other properties
	service.Use(log.LogRequest(config.IsPostgresDeveloperModeEnabled()))
	service.Use(gzip.Middleware(9))
	service.Use(jsonapi.ErrorHandler(service, true))
	service.Use(middleware.Recover())

	service.WithLogger(goalogrus.New(log.Logger()))

	// Setup Account/Login/Security
	identityRepository := account.NewIdentityRepository(db)
	userRepository := account.NewUserRepository(db)

	var notificationChannel notification.Channel = &notification.DevNullChannel{}
	if config.GetNotificationServiceURL() != "" {
		log.Logger().Infof("Enabling Notification service %v", config.GetNotificationServiceURL())
		channel, err := notification.NewServiceChannel(config)
		if err != nil {
			log.Panic(nil, map[string]interface{}{
				"err": err,
				"url": config.GetNotificationServiceURL(),
			}, "failed to parse notification service url")
		}
		notificationChannel = channel
	}

	appDB := gormapplication.NewGormDB(db)

	publicKey, err := config.GetTokenPublicKey()
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"err": err,
		}, "failed to parse public token")
	}
	tokenManager := token.NewManager(publicKey)
	// Middleware that extracts and stores the token in the context
	jwtMiddlewareTokenContext := witmiddleware.TokenContext(publicKey, nil, app.NewJWTSecurity())
	service.Use(jwtMiddlewareTokenContext)

	service.Use(login.InjectTokenManager(tokenManager))
	service.Use(log.LogRequest(config.IsPostgresDeveloperModeEnabled()))
	app.UseJWTMiddleware(service, goajwt.New(publicKey, nil, app.NewJWTSecurity()))

	spaceAuthzService := authz.NewAuthzService(config, appDB)
	service.Use(authz.InjectAuthzService(spaceAuthzService))

	loginService := login.NewKeycloakOAuthProvider(identityRepository, userRepository, tokenManager, appDB)
	loginCtrl := controller.NewLoginController(service, loginService, tokenManager, config, identityRepository)
	app.MountLoginController(service, loginCtrl)

	logoutCtrl := controller.NewLogoutController(service, &login.KeycloakLogoutService{}, config)
	app.MountLogoutController(service, logoutCtrl)

	// Mount "status" controller
	statusCtrl := controller.NewStatusController(service, db)
	app.MountStatusController(service, statusCtrl)

	// Mount "workitem" controller
	//workitemCtrl := controller.NewWorkitemController(service, appDB, config)
	workitemCtrl := controller.NewNotifyingWorkitemController(service, appDB, notificationChannel, config)
	app.MountWorkitemController(service, workitemCtrl)

	// Mount "named workitem" controller
	namedWorkitemsCtrl := controller.NewNamedWorkItemsController(service, appDB)
	app.MountNamedWorkItemsController(service, namedWorkitemsCtrl)

	// Mount "workitems" controller
	//workitemsCtrl := controller.NewWorkitemsController(service, appDB, config)
	workitemsCtrl := controller.NewNotifyingWorkitemsController(service, appDB, notificationChannel, config)
	app.MountWorkitemsController(service, workitemsCtrl)

	// Mount "workitemtype" controller
	workitemtypeCtrl := controller.NewWorkitemtypeController(service, appDB, config)
	app.MountWorkitemtypeController(service, workitemtypeCtrl)

	// Mount "work item link category" controller
	workItemLinkCategoryCtrl := controller.NewWorkItemLinkCategoryController(service, appDB)
	app.MountWorkItemLinkCategoryController(service, workItemLinkCategoryCtrl)

	// Mount "work item link type" controller
	workItemLinkTypeCtrl := controller.NewWorkItemLinkTypeController(service, appDB, config)
	app.MountWorkItemLinkTypeController(service, workItemLinkTypeCtrl)

	// Mount "work item link" controller
	workItemLinkCtrl := controller.NewWorkItemLinkController(service, appDB, config)
	app.MountWorkItemLinkController(service, workItemLinkCtrl)

	// Mount "work item comments" controller
	//workItemCommentsCtrl := controller.NewWorkItemCommentsController(service, appDB, config)
	workItemCommentsCtrl := controller.NewNotifyingWorkItemCommentsController(service, appDB, notificationChannel, config)
	app.MountWorkItemCommentsController(service, workItemCommentsCtrl)

	// Mount "work item relationships links" controller
	workItemRelationshipsLinksCtrl := controller.NewWorkItemRelationshipsLinksController(service, appDB, config)
	app.MountWorkItemRelationshipsLinksController(service, workItemRelationshipsLinksCtrl)

	// Mount "comments" controller
	//commentsCtrl := controller.NewCommentsController(service, appDB, config)
	commentsCtrl := controller.NewNotifyingCommentsController(service, appDB, notificationChannel, config)
	app.MountCommentsController(service, commentsCtrl)

	var scheduler *remoteworkitem.Scheduler
	if config.GetFeatureWorkitemRemote() {
		// Scheduler to fetch and import remote tracker items
		scheduler = remoteworkitem.NewScheduler(db)
		defer scheduler.Stop()

		accessTokens := controller.GetAccessTokens(config)
		scheduler.ScheduleAllQueries(service.Context, accessTokens)

		// Mount "tracker" controller
		c5 := controller.NewTrackerController(service, appDB, scheduler, config)
		app.MountTrackerController(service, c5)

		// Mount "trackerquery" controller
		c6 := controller.NewTrackerqueryController(service, appDB, scheduler, config)
		app.MountTrackerqueryController(service, c6)
	}

	// Mount "space" controller
	spaceCtrl := controller.NewSpaceController(service, appDB, config, auth.NewAuthzResourceManager(config))
	app.MountSpaceController(service, spaceCtrl)

	// Mount "user" controller
	userCtrl := controller.NewUserController(service, appDB, tokenManager, config)
	if config.GetTenantServiceURL() != "" {
		log.Logger().Infof("Enabling Init Tenant service %v", config.GetTenantServiceURL())
		userCtrl.InitTenant = account.NewInitTenant(config)
	}
	app.MountUserController(service, userCtrl)

	userServiceCtrl := controller.NewUserServiceController(service)
	userServiceCtrl.UpdateTenant = account.NewUpdateTenant(config)
	userServiceCtrl.CleanTenant = account.NewCleanTenant(config)
	userServiceCtrl.ShowTenant = account.NewShowTenant(config)
	app.MountUserServiceController(service, userServiceCtrl)

	// Mount "search" controller
	searchCtrl := controller.NewSearchController(service, appDB, config)
	app.MountSearchController(service, searchCtrl)

	// Mount "users" controller
	keycloakProfileService := login.NewKeycloakUserProfileClient()
	usersCtrl := controller.NewUsersController(service, appDB, config, keycloakProfileService)
	app.MountUsersController(service, usersCtrl)

	// Mount "labels" controller
	labelCtrl := controller.NewLabelController(service, appDB, config)
	app.MountLabelController(service, labelCtrl)

	// Mount "iterations" controller
	iterationCtrl := controller.NewIterationController(service, appDB, config)
	app.MountIterationController(service, iterationCtrl)

	// Mount "spaceiterations" controller
	spaceIterationCtrl := controller.NewSpaceIterationsController(service, appDB, config)
	app.MountSpaceIterationsController(service, spaceIterationCtrl)

	// Mount "userspace" controller
	userspaceCtrl := controller.NewUserspaceController(service, db)
	app.MountUserspaceController(service, userspaceCtrl)

	// Mount "render" controller
	renderCtrl := controller.NewRenderController(service)
	app.MountRenderController(service, renderCtrl)

	// Mount "areas" controller
	areaCtrl := controller.NewAreaController(service, appDB, config)
	app.MountAreaController(service, areaCtrl)

	spaceAreaCtrl := controller.NewSpaceAreasController(service, appDB, config)
	app.MountSpaceAreasController(service, spaceAreaCtrl)

	filterCtrl := controller.NewFilterController(service, config)
	app.MountFilterController(service, filterCtrl)

	// Mount "namedspaces" controller
	namedSpacesCtrl := controller.NewNamedspacesController(service, appDB)
	app.MountNamedspacesController(service, namedSpacesCtrl)

	// Mount "plannerBacklog" controller
	plannerBacklogCtrl := controller.NewPlannerBacklogController(service, appDB, config)
	app.MountPlannerBacklogController(service, plannerBacklogCtrl)

	// Mount "codebase" controller
	codebaseCtrl := controller.NewCodebaseController(service, appDB, config)
	codebaseCtrl.ShowTenant = account.NewShowTenant(config)
	app.MountCodebaseController(service, codebaseCtrl)

	// Mount "spacecodebases" controller
	spaceCodebaseCtrl := controller.NewSpaceCodebasesController(service, appDB)
	app.MountSpaceCodebasesController(service, spaceCodebaseCtrl)

	// Mount "collaborators" controller
	collaboratorsCtrl := controller.NewCollaboratorsController(service, appDB, config, auth.NewKeycloakPolicyManager(config))
	app.MountCollaboratorsController(service, collaboratorsCtrl)

	// Mount "space template" controller
	spaceTemplateCtrl := controller.NewSpaceTemplateController(service, appDB)
	app.MountSpaceTemplateController(service, spaceTemplateCtrl)

	// Mount "type hierarchy" controller
	workItemTypeGroupCtrl := controller.NewWorkItemTypeGroupController(service, appDB)
	app.MountWorkItemTypeGroupController(service, workItemTypeGroupCtrl)

	log.Logger().Infoln("Git Commit SHA: ", controller.Commit)
	log.Logger().Infoln("UTC Build Time: ", controller.BuildTime)
	log.Logger().Infoln("UTC Start Time: ", controller.StartTime)
	log.Logger().Infoln("Dev mode:       ", config.IsPostgresDeveloperModeEnabled())
	log.Logger().Infoln("GOMAXPROCS:     ", runtime.GOMAXPROCS(-1))
	log.Logger().Infoln("NumCPU:         ", runtime.NumCPU())

	engine := api.NewGinEngine(appDB, notificationChannel, config)
	engine.Any("/legacyapi/*w", gin.WrapH(service.Mux))
	engine.GET("/", gin.WrapH(http.FileServer(assetFS())))
	engine.GET("/favicon.ico", gin.WrapH(http.NotFoundHandler()))
	engine.Run(config.GetHTTPAddress())
}

func connectToDB(config *configuration.ConfigurationData) *gorm.DB {
	var db *gorm.DB
	var err error
	for {
		db, err = gorm.Open("postgres", config.GetPostgresConfigString())
		if err != nil {
			db.Close()
			log.Logger().Errorf("ERROR: Unable to open connection to database %v", err)
			log.Logger().Infof("Retrying to connect in %v...", config.GetPostgresConnectionRetrySleep())
			time.Sleep(config.GetPostgresConnectionRetrySleep())
		} else {
			break
		}
	}

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
	log.Logger().Infof("DB connection: %v...", db)
	return db
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
