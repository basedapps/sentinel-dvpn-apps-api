package jobs

import (
	"dvpn/internal/sentinel"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SyncFeeGrantList struct {
	DB       *gorm.DB
	Logger   *zap.SugaredLogger
	Sentinel *sentinel.Sentinel
}

func (job SyncFeeGrantList) Run() {
	job.Logger.Infof("fetching grant fee allowances from Sentinel")
	allowances, err := job.fetchAllowances()
	if err != nil {
		job.Logger.Errorw("failed to fetch grant fee allowances from Sentinel", "error", err)
		return
	}

	for _, allowance := range *allowances {
		tx := job.DB.Exec("UPDATE devices SET is_fee_granted= ? WHERE wallet_address = ?", true, allowance.Grantee)
		if tx.Error != nil {
			job.Logger.Errorf("Error updating grant fee allowances for wallet "+allowance.Grantee+": %v", tx.Error)
			return
		}
	}
}

func (job SyncFeeGrantList) fetchAllowances() (*[]sentinel.SentinelAllowance, error) {
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
