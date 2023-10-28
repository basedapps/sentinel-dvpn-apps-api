package jobs

import (
	"dvpn/internal/sentinel"
	"dvpn/models"
	"errors"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"time"
)

type EnrollWalletsJob struct {
	DB       *gorm.DB
	Logger   *zap.SugaredLogger
	Sentinel *sentinel.Sentinel
}

func (job EnrollWalletsJob) Run() {

	now := time.Now()
	inactivityThreshold := now.Add(time.Duration(24) * time.Hour)

	var sentinelPlanSubscription *models.SentinelPlanSubscription
	tx := job.DB.Model(&models.SentinelPlanSubscription{}).Order("id desc").First(&sentinelPlanSubscription, "inactive_at > ?", inactivityThreshold)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			s, err := job.Sentinel.CreatePlanSubscription()
			if err != nil {
				job.Logger.Error("failed to create sentinel plan subscription: " + err.Error())
				return
			}

			sentinelPlanSubscription = &models.SentinelPlanSubscription{
				ID:         s.Base.ID,
				InactiveAt: s.Base.InactiveAt,
			}

			tx := job.DB.Create(sentinelPlanSubscription)
			if tx.Error != nil {
				job.Logger.Error("failed to save create sentinel plan subscription to the DB: " + tx.Error.Error())
				return
			}
		} else {
			job.Logger.Error("failed to get sentinel plan subscription from the DB: " + tx.Error.Error())
			return
		}
	}

	var devices []models.Device
	tx = job.DB.Model(&models.Device{}).Order("created_at").Limit(10).Where("subscription_id IS DISTINCT FROM ?", sentinelPlanSubscription.ID).Find(&devices)
	if tx.Error != nil {
		job.Logger.Error("failed to get sentinel wallets from the DB: " + tx.Error.Error())
		return
	}

	var walletAddresses []string = make([]string, 0)

	for _, device := range devices {
		walletAddresses = append(walletAddresses, device.WalletAddress)
	}

	if len(walletAddresses) == 0 {
		return
	}

	err := job.Sentinel.EnrollWalletToSubscription(walletAddresses, sentinelPlanSubscription.ID)
	if err != nil {
		job.Logger.Error("failed to enroll sentinel wallets to subscription: " + err.Error())
		return
	}

	for _, device := range devices {
		device.SubscriptionId = &sentinelPlanSubscription.ID
		tx := job.DB.Save(&device)
		if tx.Error != nil {
			job.Logger.Error("failed to update device subscription ID: " + tx.Error.Error())
			continue
		}
	}
}
