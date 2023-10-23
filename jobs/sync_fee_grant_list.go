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

}
