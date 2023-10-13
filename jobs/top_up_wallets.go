package jobs

import (
	"dvpn/internal/sentinel"
	"dvpn/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"os"
	"strconv"
)

type TopUpWalletsJob struct {
	DB       *gorm.DB
	Logger   *zap.SugaredLogger
	Sentinel *sentinel.Sentinel
}

func (job TopUpWalletsJob) Run() {
	job.updateWalletsWithUnknownBalance()

	minimalBalance, err := strconv.ParseInt(os.Getenv("SENTINEL_MIN_BALANCE"), 10, 64)

	var devices []models.Device
	tx := job.DB.Model(&models.Device{}).Order("created_at").Limit(100).Find(&devices, "current_balance < ?", minimalBalance)
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

	err = job.Sentinel.GrantTokens(walletAddresses, minimalBalance*5)
	if err != nil {
		job.Logger.Error("failed to top up sentinel wallets: " + err.Error())
		return
	}

	for _, device := range devices {
		balance, err := job.Sentinel.FetchBalance(device.WalletAddress)
		if err != nil {
			job.Logger.Error("failed to fetch sentinel wallet balance: " + err.Error())
			continue
		}

		device.CurrentBalance = balance
		tx = job.DB.Save(&device)
		if tx.Error != nil {
			job.Logger.Error("failed to update device subscription ID: " + tx.Error.Error())
			continue
		}
	}
}

func (job TopUpWalletsJob) updateWalletsWithUnknownBalance() {
	var devices []models.Device
	tx := job.DB.Model(&models.Device{}).Order("created_at").Find(&devices, "current_balance = ?", -1)
	if tx.Error != nil {
		job.Logger.Error("failed to get sentinel wallets from the DB: " + tx.Error.Error())
		return
	}

	for _, device := range devices {
		balance, err := job.Sentinel.FetchBalance(device.WalletAddress)
		if err != nil {
			job.Logger.Error("failed to fetch sentinel wallet balance: " + err.Error())
			continue
		}

		device.CurrentBalance = balance
		tx = job.DB.Save(&device)
		if tx.Error != nil {
			job.Logger.Error("failed to update device subscription ID: " + tx.Error.Error())
			continue
		}
	}
}
