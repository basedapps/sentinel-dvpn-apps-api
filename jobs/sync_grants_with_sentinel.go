package jobs

import (
	"dvpn/internal/sentinel"
	"dvpn/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SyncGrantsWithSentinelJob struct {
	DB       *gorm.DB
	Logger   *zap.SugaredLogger
	Sentinel *sentinel.Sentinel
}

func (job SyncGrantsWithSentinelJob) Run() {
	job.Logger.Infof("fetching grant fee allowances from Sentinel")
	allowances, err := job.fetchAllowances()
	if err != nil {
		job.Logger.Errorw("failed to fetch grant fee allowances from Sentinel", "error", err)
		return
	}

	var devices []models.Device
	tx := job.DB.Model(&models.Device{}).Order("created_at").Find(&devices, "sentinel_is_fee_granted = ?", false)
	if tx.Error != nil {
		job.Logger.Error("failed to get sentinel wallets from the DB: " + tx.Error.Error())
		return
	}

	for _, allowance := range *allowances {
		for _, device := range devices {
			if allowance.Grantee == device.WalletAddress {
				device.IsFeeGranted = true
				tx = job.DB.Save(&device)
				if tx.Error != nil {
					job.Logger.Error("failed to update device Sentinel `is_fee_grant` status: " + tx.Error.Error())
					continue
				}
			}
		}
	}
}

func (job SyncGrantsWithSentinelJob) fetchAllowances() (*[]sentinel.SentinelAllowance, error) {
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
