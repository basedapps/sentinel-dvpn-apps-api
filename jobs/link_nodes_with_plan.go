package jobs

import (
	"dvpn/internal/sentinel"
	"dvpn/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"os"
	"strconv"
)

type LinkNodesWithPlanJob struct {
	DB       *gorm.DB
	Logger   *zap.SugaredLogger
	Sentinel *sentinel.Sentinel
}

func (job LinkNodesWithPlanJob) Run() {
	var servers []models.Server
	tx := job.DB.Model(&models.Server{}).Order("created_at").Limit(10).Find(&servers, "is_included_in_plan = ? AND is_banned = ?", false, false)
	if tx.Error != nil {
		job.Logger.Error("failed to get sentinel servers from the DB: " + tx.Error.Error())
		return
	}

	maxPricePerHour, err := strconv.ParseInt(os.Getenv("SENTINEL_NODE_MAX_PRICE_PER_HOUR"), 10, 64)
	if err != nil {
		job.Logger.Error("failed to parse SENTINEL_NODE_MAX_PRICE_PER_HOUR: " + err.Error())
		return
	}

	var nodeAddresses []string = make([]string, 0)

	for _, server := range servers {
		if server.Configuration.Data().PricePerHour <= maxPricePerHour && server.IsActive {
			nodeAddresses = append(nodeAddresses, server.Configuration.Data().Address)
		}
	}

	if len(nodeAddresses) == 0 {
		return
	}

	err = job.Sentinel.AddNodeToPlan(nodeAddresses)
	if err != nil {
		job.Logger.Error("failed to add nodes to plan: " + err.Error())
		return
	}

	for _, server := range servers {
		server.IsIncludedInPlan = true
		tx = job.DB.Save(&server)
		if tx.Error != nil {
			job.Logger.Error("failed to update server in the DB: " + tx.Error.Error())
			continue
		}
	}

	tx = job.DB.Exec("UPDATE cities AS c SET servers_available = (SELECT COUNT(s.id) FROM servers AS s WHERE s.city_id = c.id AND s.is_active = ? AND s.is_included_in_plan = ? AND s.is_banned = ?)", true, true, false)
	if tx.Error != nil {
		job.Logger.Errorf("Error updating cities: %v", tx.Error)
		return
	}

	tx = job.DB.Exec("UPDATE countries AS c SET servers_available = (SELECT COUNT(s.id) FROM servers AS s WHERE s.country_id = c.id AND s.is_active = ? AND s.is_included_in_plan = ? AND s.is_banned = ?)", true, true, false)
	if tx.Error != nil {
		job.Logger.Errorf("Error updating countries: %v", tx.Error)
		return
	}
}
