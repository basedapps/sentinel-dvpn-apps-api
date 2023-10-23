package jobs

import (
	"dvpn/internal/sentinel"
	"dvpn/models"
	"errors"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"strconv"
	"time"
)

type SyncNodesWithSentinelJob struct {
	DB       *gorm.DB
	Logger   *zap.SugaredLogger
	Sentinel *sentinel.Sentinel

	planNodes *[]sentinel.SentinelNode
}

func (job SyncNodesWithSentinelJob) Run() {
	job.Logger.Infof("fetching nodes from Sentinel")
	nodes, err := job.fetchActiveNodes()
	if err != nil {
		job.Logger.Errorw("failed to fetch nodes from Sentinel", "error", err)
		return
	}

	job.Logger.Infof("fetching nodes listed on plan from Sentinel")
	job.planNodes, err = job.fetchNodesOnPlan()
	if err != nil {
		job.Logger.Errorw("failed to fetch nodes listed on plan from Sentinel", "error", err)
		return
	}

	job.Logger.Infof("processing %d nodes", len(*nodes))
	job.processNodes(nodes)
}

func (job SyncNodesWithSentinelJob) processNodes(nodes *[]sentinel.SentinelNode) {
	revision := time.Now().Unix()

	for _, node := range *nodes {
		job.Logger.Infof("requesting status for node %s", node.Address)

		status, err := job.Sentinel.FetchNodeStatus(node)
		if err == nil {
			protocols := datatypes.NewJSONType(job.parseNodeProtocols(status))
			configuration := datatypes.NewJSONType(job.parseNodeConfiguration(&node, status))
			currentLoad := job.parseCurrentLoad(status)
			countryId, err := job.parseCountryId(status)
			if err != nil {
				job.Logger.Errorf("failed to determine country id for %s: %s", status.Address, err)
				continue
			}

			cityId, err := job.parseCityId(status, countryId)
			if err != nil {
				job.Logger.Errorf("failed to determine city id in country %d for %s: %s", countryId, status.Address, err)
				continue
			}

			var server models.Server
			tx := job.DB.First(&server, "\"configuration\"->>'address' = ?", status.Address)
			if tx.Error == nil {
				server.Name = status.Moniker
				server.CountryID = countryId
				server.CityID = cityId
				server.Protocols = protocols
				server.Configuration = configuration
				server.CurrentLoad = currentLoad
				server.IsActive = true
				server.IsIncludedInPlan = job.checkIfIncludedInPlan(&node)
				server.Revision = revision

				tx = job.DB.Save(&server)
				if tx.Error != nil {
					job.Logger.Errorf("failed to update server %s in the DB: %s", status.Address, tx.Error)
				} else {
					job.Logger.Infof("updated DB record for server %s", status.Address)
				}
			} else {
				if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
					server = models.Server{
						CountryID:        countryId,
						CityID:           cityId,
						Name:             status.Moniker,
						IsActive:         true,
						IsBanned:         false,
						IsIncludedInPlan: job.checkIfIncludedInPlan(&node),
						CurrentLoad:      currentLoad,
						Protocols:        protocols,
						Configuration:    configuration,
						Revision:         revision,
					}

					tx = job.DB.Create(&server)
					if tx.Error != nil {
						job.Logger.Errorf("failed to create server %s in the DB: %s", status.Address, tx.Error)
					} else {
						job.Logger.Infof("created DB record for server %s", status.Address)
					}
				} else {
					job.Logger.Errorf("failed to fetch server %s from database: %s", status.Address, tx.Error)
				}
			}
		} else {
			job.Logger.Warnw("failed to fetch Sentinel node status for "+node.Address+": "+err.Error(), "url", node.RemoteURL)
		}
	}

	tx := job.DB.Model(&models.Server{}).Where("revision != ?", revision).Update("is_active", false)
	if tx.Error != nil {
		job.Logger.Errorf("failed to deactivate inactive servers: %s", tx.Error)
	} else {
		job.Logger.Infof("deactivated %d inactive servers", tx.RowsAffected)
	}
}

func (job SyncNodesWithSentinelJob) fetchActiveNodes() (*[]sentinel.SentinelNode, error) {
	var syncInProgress bool
	var limit int
	var offset int

	syncInProgress = true
	limit = 100
	offset = 0

	var nodes []sentinel.SentinelNode

	for syncInProgress {
		n, err := job.Sentinel.FetchNodes(limit, offset)
		if err != nil {
			return nil, err
		}

		if n == nil {
			syncInProgress = false
		} else {
			nodes = append(nodes, *n...)
		}

		offset += limit
	}

	return &nodes, nil
}

func (job SyncNodesWithSentinelJob) fetchNodesOnPlan() (*[]sentinel.SentinelNode, error) {
	var syncInProgress bool
	var limit int
	var offset int

	syncInProgress = true
	limit = 100
	offset = 0

	var nodes []sentinel.SentinelNode

	for syncInProgress {
		n, err := job.Sentinel.FetchPlanNodes(limit, offset)
		if err != nil {
			return nil, err
		}

		if n == nil {
			syncInProgress = false
		} else {
			nodes = append(nodes, *n...)
		}

		offset += limit
	}

	return &nodes, nil
}

func (job SyncNodesWithSentinelJob) parseNodePrices(node *sentinel.SentinelNode) (int64, int64) {
	var pricePerGB int64
	for _, gigabytePrice := range node.GigabytePrices {
		if gigabytePrice.Denom == job.Sentinel.DefaultDenom {
			pricePerGB, _ = strconv.ParseInt(gigabytePrice.Amount, 10, 64)
		}
	}

	var pricePerHour int64
	for _, hourlyPrice := range node.HourlyPrices {
		if hourlyPrice.Denom == job.Sentinel.DefaultDenom {
			pricePerHour, _ = strconv.ParseInt(hourlyPrice.Amount, 10, 64)
		}
	}

	return pricePerGB, pricePerHour
}

func (job SyncNodesWithSentinelJob) parseNodeProtocols(status *sentinel.SentinelNodeStatus) []models.ServerProtocol {
	var supportedProtocols []models.ServerProtocol

	if status.Type == 1 {
		supportedProtocols = append(supportedProtocols, models.ServerProtocolWireGuard)
	}

	if status.Type == 2 {
		supportedProtocols = append(supportedProtocols, models.ServerProtocolV2Ray)
	}

	return supportedProtocols
}

func (job SyncNodesWithSentinelJob) parseNodeConfiguration(node *sentinel.SentinelNode, status *sentinel.SentinelNodeStatus) models.ServerConfiguration {
	pricePerGB, pricePerHour := job.parseNodePrices(node)

	return models.ServerConfiguration{
		RemoteURL:         node.RemoteURL,
		Address:           status.Address,
		BandwidthDownload: status.Bandwidth.Download,
		BandwidthUpload:   status.Bandwidth.Upload,
		LocationCity:      status.Location.City,
		LocationCountry:   status.Location.Country,
		LocationLat:       status.Location.Latitude,
		LocationLon:       status.Location.Longitude,
		PricePerGB:        pricePerGB,
		PricePerHour:      pricePerHour,
		Version:           status.Version,
	}
}

func (job SyncNodesWithSentinelJob) parseCurrentLoad(status *sentinel.SentinelNodeStatus) float64 {
	currentLoad := float64(status.Peers) / float64(status.QoS.MaxPeers)
	if currentLoad > 1 {
		currentLoad = 1
	}

	return currentLoad
}

func (job SyncNodesWithSentinelJob) parseCountryId(status *sentinel.SentinelNodeStatus) (uint, error) {
	countryName := status.Location.Country

	var country models.Country
	tx := job.DB.First(&country, "name = ?", countryName)
	if tx.Error != nil {
		return 0, tx.Error
	}

	return country.ID, nil
}

func (job SyncNodesWithSentinelJob) parseCityId(status *sentinel.SentinelNodeStatus, countryId uint) (uint, error) {
	cityName := status.Location.City

	var city models.City
	tx := job.DB.First(&city, "name = ? AND country_id = ?", cityName, countryId)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			city = models.City{
				Name:             status.Location.City,
				CountryID:        countryId,
				ServersAvailable: 0,
			}
			tx = job.DB.Create(&city)
			if tx.Error != nil {
				job.Logger.Errorf("Error creating city %s: %v", city.Name, tx.Error)
				return 0, tx.Error
			}

			return city.ID, nil
		}
		return 0, tx.Error
	}

	return city.ID, nil
}

func (job SyncNodesWithSentinelJob) checkIfIncludedInPlan(node *sentinel.SentinelNode) bool {
	for _, planNode := range *job.planNodes {
		if planNode.Address == node.Address {
			return true
		}
	}

	return false
}
