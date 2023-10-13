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

	sentinel := &sentinelAPI.Sentinel{
		APIEndpoint:                 os.Getenv("SENTINEL_API_ENDPOINT"),
		RPCEndpoint:                 os.Getenv("SENTINEL_RPC_ENDPOINT"),
		ProviderWalletAddress:       os.Getenv("SENTINEL_PROVIDER_WALLET_ADDRESS"),
		ProviderMnemonic:            os.Getenv("SENTINEL_PROVIDER_WALLET_MNEMONIC"),
		ProviderPlanID:              os.Getenv("SENTINEL_PROVIDER_PLAN_ID"),
		MainSubscriberWalletAddress: os.Getenv("SENTINEL_MAIN_SUBSCRIBER_WALLET_ADDRESS"),
		MainSubscriberMnemonic:      os.Getenv("SENTINEL_MAIN_SUBSCRIBER_WALLET_MNEMONIC"),
		DefaultDenom:                os.Getenv("SENTINEL_DEFAULT_DENOM"),
		ChainID:                     os.Getenv("SENTINEL_CHAIN_ID"),
		GasPrice:                    os.Getenv("SENTINEL_GAS_PRICE"),
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
		syncWithSentinelJob := jobs.SyncWithSentinelJob{
			DB:       db,
			Logger:   logger,
			Sentinel: sentinel,
		}

		topUpWalletsJob := jobs.TopUpWalletsJob{
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

		nodesScheduler := gocron.NewScheduler(time.UTC)
		nodesScheduler.SetMaxConcurrentJobs(1, gocron.RescheduleMode)
		nodesScheduler.Every(1).Hour().Do(func() {
			syncWithSentinelJob.Run()
		})
		nodesScheduler.StartAsync()

		walletsScheduler := gocron.NewScheduler(time.UTC)
		walletsScheduler.SetMaxConcurrentJobs(1, gocron.RescheduleMode)
		walletsScheduler.Every(5).Seconds().Do(func() {
			topUpWalletsJob.Run()
		})
		walletsScheduler.Every(10).Seconds().Do(func() {
			enrollWalletJob.Run()
		})
		walletsScheduler.StartAsync()

		planScheduler := gocron.NewScheduler(time.UTC)
		planScheduler.SetMaxConcurrentJobs(1, gocron.WaitMode)
		planScheduler.Every(10).Minute().Do(func() {
			linkNodesWithPlanJob.Run()
		})
		planScheduler.Every(20).Minute().Do(func() {
			unlinkNodesFromPlanJob.Run()
		})
		planScheduler.StartAsync()
	}

	logger.Info("Registering routes...")
	router.RegisterRoutes(engine)

	logger.Info("Launching API server...")
	engine.Run()
}
