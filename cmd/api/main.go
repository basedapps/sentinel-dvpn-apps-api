package main

import (
	"dvpn/controllers"
	"dvpn/core"
	sentinelAPI "dvpn/internal/sentinel"
	"dvpn/jobs"
	"dvpn/middleware"
	"dvpn/models"
	"dvpn/routers"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"github.com/joho/godotenv"
	"os"
	"strconv"
	"time"
)

func main() {
	godotenv.Load()

	db, err := core.InitDB()
	if err != nil {
		panic(err)
	}

	err = db.Debug().AutoMigrate(
		&models.Device{},
		&models.Country{},
		&models.Server{},
		&models.SentinelPlanSubscription{},
		&models.SentinelNodeSubscription{},
	)
	if err != nil {
		panic(err)
	}

	err = core.PopulateDB(db)
	if err != nil {
		panic(err)
	}

	engine := gin.Default()
	err = engine.SetTrustedProxies(nil)
	if err != nil {
		panic(err)
	}

	logger, err := core.NewLogger()
	if err != nil {
		panic(err)
	}

	auth := &middleware.AuthMiddleware{
		DB:     db,
		Logger: logger.With("middleware", "auth"),
	}

	gasBase, err := strconv.ParseInt(os.Getenv("SENTINEL_GAS_BASE"), 10, 64)
	if err != nil {
		panic(err)
	}

	sentinel := &sentinelAPI.Sentinel{
		APIEndpoint:                      os.Getenv("SENTINEL_API_ENDPOINT"),
		RPCEndpoint:                      os.Getenv("SENTINEL_RPC_ENDPOINT"),
		ProviderPlanID:                   os.Getenv("SENTINEL_PROVIDER_PLAN_ID"),
		ProviderWalletAddress:            os.Getenv("SENTINEL_PROVIDER_WALLET_ADDRESS"),
		ProviderMnemonic:                 os.Getenv("SENTINEL_PROVIDER_WALLET_MNEMONIC"),
		NodeSubscriberWalletAddress:      os.Getenv("SENTINEL_NODE_SUBSCRIBER_WALLET_ADDRESS"),
		NodeSubscriberMnemonic:           os.Getenv("SENTINEL_NODE_SUBSCRIBER_WALLET_MNEMONIC"),
		NodeLinkerWalletAddress:          os.Getenv("SENTINEL_NODE_LINKER_WALLET_ADDRESS"),
		NodeLinkerMnemonic:               os.Getenv("SENTINEL_NODE_LINKER_WALLET_MNEMONIC"),
		NodeRemoverWalletAddress:         os.Getenv("SENTINEL_NODE_REMOVER_WALLET_ADDRESS"),
		NodeRemoverMnemonic:              os.Getenv("SENTINEL_NODE_REMOVER_WALLET_MNEMONIC"),
		FeeGranterWalletAddress:          os.Getenv("SENTINEL_FEE_GRANTER_WALLET_ADDRESS"),
		FeeGranterMnemonic:               os.Getenv("SENTINEL_FEE_GRANTER_WALLET_MNEMONIC"),
		MainSubscriberWalletAddress:      os.Getenv("SENTINEL_MAIN_SUBSCRIBER_WALLET_ADDRESS"),
		MainSubscriberMnemonic:           os.Getenv("SENTINEL_MAIN_SUBSCRIBER_WALLET_MNEMONIC"),
		SubscriptionUpdaterWalletAddress: os.Getenv("SENTINEL_SUBSCRIPTION_UPDATER_WALLET_ADDRESS"),
		SubscriptionUpdaterMnemonic:      os.Getenv("SENTINEL_SUBSCRIPTION_UPDATER_WALLET_MNEMONIC"),
		WalletEnrollerWalletAddress:      os.Getenv("SENTINEL_WALLET_ENROLLER_WALLET_ADDRESS"),
		WalletEnrollerMnemonic:           os.Getenv("SENTINEL_WALLET_ENROLLER_WALLET_MNEMONIC"),
		DefaultDenom:                     os.Getenv("SENTINEL_DEFAULT_DENOM"),
		ChainID:                          os.Getenv("SENTINEL_CHAIN_ID"),
		GasPrice:                         os.Getenv("SENTINEL_GAS_PRICE"),
		GasBase:                          gasBase,
	}

	router := routers.Router{
		Auth: auth,
		HealthController: &controllers.HealthController{
			DB:     db,
			Logger: logger.With("controller", "health"),
		},
		DevicesController: &controllers.DevicesController{
			DB:     db,
			Logger: logger.With("controller", "devices"),
			Auth:   auth,
		},
		VPNController: &controllers.VPNController{
			DB:       db,
			Logger:   logger.With("controller", "vpn"),
			Auth:     auth,
			Sentinel: sentinel,
		},
	}

	logger.Info("Initializing jobs...")
	if os.Getenv("ENVIRONMENT") != "debug" {
		syncWithSentinelJob := jobs.SyncNodesWithSentinelJob{
			DB:       db,
			Logger:   logger,
			Sentinel: sentinel,
		}

		syncGrantsWithSentinelJob := jobs.SyncGrantsWithSentinelJob{
			DB:       db,
			Logger:   logger,
			Sentinel: sentinel,
		}

		grantFeeToWalletsJob := jobs.GrantFeeToWalletsJob{
			DB:       db,
			Logger:   logger,
			Sentinel: sentinel,
		}

		enrollWalletJob := jobs.EnrollWalletsJob{
			DB:       db,
			Logger:   logger,
			Sentinel: sentinel,
		}

		linkNodesWithPlanJob := jobs.LinkNodesWithPlanJob{
			DB:       db,
			Logger:   logger,
			Sentinel: sentinel,
		}

		unlinkNodesFromPlanJob := jobs.UnlinkNodesFromPlanJob{
			DB:       db,
			Logger:   logger,
			Sentinel: sentinel,
		}

		sentinelScheduler := gocron.NewScheduler(time.UTC)
		sentinelScheduler.SetMaxConcurrentJobs(1, gocron.RescheduleMode)
		sentinelScheduler.Every(1).Hour().Do(func() {
			syncWithSentinelJob.Run()
			syncGrantsWithSentinelJob.Run()
		})
		sentinelScheduler.StartAsync()

		grantFeeScheduler := gocron.NewScheduler(time.UTC)
		grantFeeScheduler.SetMaxConcurrentJobs(1, gocron.RescheduleMode)
		grantFeeScheduler.Every(1).Seconds().Do(func() {
			grantFeeToWalletsJob.Run()
		})
		grantFeeScheduler.StartAsync()

		enrollWalletScheduler := gocron.NewScheduler(time.UTC)
		enrollWalletScheduler.SetMaxConcurrentJobs(1, gocron.RescheduleMode)
		enrollWalletScheduler.Every(1).Seconds().Do(func() {
			enrollWalletJob.Run()
		})
		enrollWalletScheduler.StartAsync()

		planScheduler := gocron.NewScheduler(time.UTC)
		planScheduler.SetMaxConcurrentJobs(1, gocron.RescheduleMode)
		planScheduler.Every(5).Seconds().Do(func() {
			linkNodesWithPlanJob.Run()
			unlinkNodesFromPlanJob.Run()
		})
		planScheduler.StartAsync()
	}

	logger.Info("Registering routes...")
	router.RegisterRoutes(engine)

	logger.Info("Launching API server...")
	engine.Run()
}
