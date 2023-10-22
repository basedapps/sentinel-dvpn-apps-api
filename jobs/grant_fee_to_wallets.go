package jobs

import (
	"dvpn/internal/sentinel"
	"dvpn/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type GrantFeeToWalletsJob struct {
	DB       *gorm.DB
	Logger   *zap.SugaredLogger
	Sentinel *sentinel.Sentinel
}

func (job GrantFeeToWalletsJob) Run() {
	var devices []models.Device
	tx := job.DB.Model(&models.Device{}).Order("created_at").Limit(10).Find(&devices, "is_fee_granted = ?", false)
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

	err := job.Sentinel.GrantFeeToWallet(walletAddresses)
	if err != nil {
		job.Logger.Error("failed to grant fee to sentinel wallets: " + err.Error())
		return
	}

	for _, device := range devices {
		device.IsFeeGranted = true
		tx = job.DB.Save(&device)
		if tx.Error != nil {
			job.Logger.Error("failed to update device `is_fee_grant` status: " + tx.Error.Error())
			continue
		}
	}
}
