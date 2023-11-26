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

	job.Logger.Infof("fetching grant fee allowances from Sentinel")
	existingAllowances, err := job.fetchAllowances()
	if err != nil {
		job.Logger.Errorw("failed to fetch grant fee allowances from Sentinel", "error", err)
		return
	}

	var walletAddresses []string = make([]string, 0)

	for _, device := range devices {
		for _, allowance := range *existingAllowances {
			if allowance.Grantee == device.WalletAddress {
				device.IsFeeGranted = true
				tx = job.DB.Save(&device)
				if tx.Error != nil {
					job.Logger.Error("failed to update device Sentinel existing `is_fee_grant` status: " + tx.Error.Error())
					continue
				}
			}
		}

		job.Logger.Infof("Sentinel wallet %s will be granted fee.", device.WalletAddress)
		walletAddresses = append(walletAddresses, device.WalletAddress)
	}

	if len(walletAddresses) == 0 {
		return
	}

	err = job.Sentinel.GrantFeeToWallet(walletAddresses)
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

		job.Logger.Infof("Sentinel wallet %s was granted fee.", device.WalletAddress)
	}
}

func (job GrantFeeToWalletsJob) fetchAllowances() (*[]sentinel.SentinelAllowance, error) {
	var syncInProgress bool
	var limit int
	var offset int

	syncInProgress = true
	limit = 100
	offset = 0

	var allowances []sentinel.SentinelAllowance

	for syncInProgress {
		n, err := job.Sentinel.FetchFeeGrantAllowances(limit, offset)
		if err != nil {
			return nil, err
		}

		if n == nil {
			syncInProgress = false
		} else {
			allowances = append(allowances, *n...)
		}

		offset += limit
	}

	return &allowances, nil
}
