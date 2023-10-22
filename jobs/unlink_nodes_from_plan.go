package jobs

import (
	"dvpn/internal/sentinel"
	"dvpn/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"os"
	"strconv"
)

type UnlinkNodesFromPlanJob struct {
	DB       *gorm.DB
	Logger   *zap.SugaredLogger
	Sentinel *sentinel.Sentinel
}

func (job UnlinkNodesFromPlanJob) Run() {
	var servers []models.Server
	tx := job.DB.Model(&models.Server{}).Order("created_at").Limit(10).Find(&servers, "is_included_in_plan = ?", true)
	if tx.Error != nil {
		job.Logger.Error("failed to get sentinel servers from the DB: " + tx.Error.Error())
		return
	}

	maxPricePerHour, err := strconv.ParseInt(os.Getenv("SENTINEL_NODE_MAX_PRICE_PER_HOUR"), 10, 64)
	if err != nil {
		job.Logger.Error("failed to parse SENTINEL_NODE_MAX_PRICE_PER_HOUR: " + err.Error())
		return
	}

	for _, server := range servers {
		if server.Configuration.Data().PricePerHour > maxPricePerHour || !server.IsActive || server.IsBanned {
			err := job.Sentinel.RemoveNodeFromPlan(server.Configuration.Data().Address)
			if err != nil {
				job.Logger.Error("failed to remove node from plan: " + err.Error())
				continue
			}

			server.IsIncludedInPlan = false
			tx = job.DB.Save(&server)
			if tx.Error != nil {
				job.Logger.Error("failed to update server in the DB: " + tx.Error.Error())
				continue
			}
		}
	}

	tx = job.DB.Exec("UPDATE cities AS c SET servers_available = (SELECT COUNT(s.id) FROM servers AS s WHERE s.city_id = c.id AND s.is_active = ? AND s.is_included_in_plan = ?)", true, true)
	if tx.Error != nil {
		job.Logger.Errorf("Error updating cities: %v", tx.Error)
		return
	}

	tx = job.DB.Exec("UPDATE countries AS c SET servers_available = (SELECT COUNT(s.id) FROM servers AS s WHERE s.country_id = c.id AND s.is_active = ? AND s.is_included_in_plan = ?)", true, true)
	if tx.Error != nil {
		job.Logger.Errorf("Error updating countries: %v", tx.Error)
		return
	}
}
