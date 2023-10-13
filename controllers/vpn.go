package controllers

import (
	"dvpn/internal/sentinel"
	"dvpn/middleware"
	"dvpn/models"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/tyler-smith/go-bip39"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type VPNController struct {
	DB       *gorm.DB
	Logger   *zap.SugaredLogger
	Auth     *middleware.AuthMiddleware
	Sentinel *sentinel.Sentinel
}

func (vc VPNController) GetIPAddress(c *gin.Context) {
	type result struct {
		Ip        string  `json:"ip"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}

	var ipAddr string

	realIp := c.GetHeader("X-Real-IP")
	if realIp != "" {
		ipAddr = realIp
	} else {
		forwardedIp := c.GetHeader("X-Forwarded-For")
		if forwardedIp != "" {
			parts := strings.Split(forwardedIp, ",")
			if len(parts) > 0 {
				ipAddr = strings.TrimSpace(parts[0])
			}
		} else {
			remoteIp := c.RemoteIP()
			if remoteIp != "" {
				ipAddr = remoteIp
			} else {
				middleware.RespondErr(c, middleware.APIErrorUnknown, "failed to get IP address")
				return
			}
		}
	}

	url := "http://ip-api.com/json/" + ipAddr
	method := "GET"

	client := &http.Client{}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		middleware.RespondErr(c, middleware.APIErrorUnknown, "failed to form request to get IP address location")
		return
	}

	res, err := client.Do(req)
	if err != nil {
		middleware.RespondErr(c, middleware.APIErrorUnknown, "failed to perform request to get IP address location")
		return
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		middleware.RespondErr(c, middleware.APIErrorUnknown, "failed to read response body to get IP address location")
		return
	}

	type coordinates struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	}

	var coordinatesObject coordinates

	err = json.Unmarshal(body, &coordinatesObject)
	if err != nil {
		middleware.RespondErr(c, middleware.APIErrorUnknown, "failed to unmarshal response body to get IP address location")
		return
	}

	resultObject := result{
		Ip:        ipAddr,
		Latitude:  coordinatesObject.Lat,
		Longitude: coordinatesObject.Lon,
	}

	middleware.RespondOK(c, resultObject)
}

func (vc VPNController) GetCountries(c *gin.Context) {
	var countries []models.Country
	tx := vc.DB.Model(&models.Country{}).Order("name").Find(&countries, "servers_available > ?", 0)
	if tx.Error != nil {
		reason := "failed to get countries: " + tx.Error.Error()
		middleware.RespondErr(c, middleware.APIErrorUnknown, reason)
		vc.Logger.Error(reason)
		return
	}

	middleware.RespondOK(c, countries)
}

func (vc VPNController) GetCities(c *gin.Context) {
	countryId, err := strconv.ParseUint(c.Params.ByName("country_id"), 10, 64)
	if err != nil {
		middleware.RespondErr(c, middleware.APIErrorInvalidRequest, "invalid country id: "+err.Error())
		return
	}

	var cities []models.City

	err = vc.DB.Order("name").Find(&cities, "country_id = ? AND servers_available > 0", countryId).Order("").Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			middleware.RespondOK(c, []models.City{})
			return
		} else {
			reason := "failed to get cities: " + err.Error()
			middleware.RespondErr(c, middleware.APIErrorUnknown, reason)
			vc.Logger.Error(reason)
			return
		}
	}

	middleware.RespondOK(c, cities)
}

func (vc VPNController) GetServers(c *gin.Context) {
	var servers []models.Server

	query := vc.DB.Model(&models.Server{}).Where("is_active = ?", true)

	sortBy := c.Query("sortBy")
	if sortBy != "" {
		switch sortBy {
		case "CURRENT_LOAD":
			query = query.Order("current_load desc")
			break
		default:
			middleware.RespondErr(c, middleware.APIErrorInvalidRequest, "invalid sortBy")
			return
		}
	}

	offset := c.Query("offset")
	if offset != "" {
		offset, err := strconv.Atoi(offset)
		if err != nil {
			middleware.RespondErr(c, middleware.APIErrorInvalidRequest, "invalid offset: "+err.Error())
			return
		}

		query = query.Offset(offset)
	}

	limit := c.Query("limit")
	if limit != "" {
		limit, err := strconv.Atoi(limit)
		if err != nil {
			middleware.RespondErr(c, middleware.APIErrorInvalidRequest, "invalid limit: "+err.Error())
			return
		}

		query = query.Limit(limit)
	}

	country := c.Query("country")
	if country != "" {
		query = query.Where("country_id = ?", strings.ToUpper(country))
	}

	city := c.Query("city")
	if city != "" {
		query = query.Where("city_id = ?", strings.ToUpper(city))
	}

	protocol := c.Query("protocol")
	if protocol != "" && protocol != "ALL" {
		switch protocol {
		case "WIREGUARD", "V2RAY":
			query = query.Where(`protocols @> ?`, "[\""+protocol+"\"]")
			break
		default:
			middleware.RespondErr(c, middleware.APIErrorInvalidRequest, "invalid protocol")
			return
		}
	}

	tx := query.Find(&servers)
	if tx.Error != nil {
		reason := "failed to get servers: " + tx.Error.Error()
		middleware.RespondErr(c, middleware.APIErrorUnknown, reason)
		vc.Logger.Error(reason)
		return
	}

	middleware.RespondOK(c, servers)
}

func (vc VPNController) ConnectToCity(c *gin.Context) {
	device, err := vc.Auth.CurrentDevice(c)
	if err != nil {
		reason := "failed to retrieve device: " + err.Error()
		middleware.RespondErr(c, middleware.APIErrorUnknown, reason)
		vc.Logger.Error(reason)
		return
	}

	countryId, err := strconv.ParseUint(c.Params.ByName("country_id"), 10, 64)
	if err != nil {
		middleware.RespondErr(c, middleware.APIErrorInvalidRequest, "invalid country id: "+err.Error())
		return
	}

	cityId, err := strconv.ParseUint(c.Params.ByName("city_id"), 10, 64)
	if err != nil {
		middleware.RespondErr(c, middleware.APIErrorInvalidRequest, "invalid city id: "+err.Error())
		return
	}

	protocol := c.Params.ByName("protocol")

	var server models.Server

	if protocol != "" {
		err = vc.DB.Order("id desc").First(&server, "current_load < ? AND country_id = ? AND city_id = ? AND is_included_in_plan = ? AND is_active = ? AND protocols->>0 = ?", 0.9, countryId, cityId, true, true, protocol).Error
		if err != nil {
			middleware.RespondErr(c, middleware.APIErrorUnknown, "failed to get server: "+err.Error())
			return
		}

		vc.createCredentials(device, &server, c)
		return
	}

	err = vc.DB.Order("id desc").First(&server, "current_load < ? AND country_id = ? AND city_id = ? AND is_included_in_plan = ? AND is_active = ?", 0.9, countryId, cityId, true, true).Error
	if err != nil {
		middleware.RespondErr(c, middleware.APIErrorUnknown, "failed to get server: "+err.Error())
		return
	}

	vc.createCredentials(device, &server, c)
}

func (vc VPNController) ConnectToServer(c *gin.Context) {
	device, err := vc.Auth.CurrentDevice(c)
	if err != nil {
		reason := "failed to retrieve device: " + err.Error()
		middleware.RespondErr(c, middleware.APIErrorUnknown, reason)
		vc.Logger.Error(reason)
		return
	}

	countryId, err := strconv.ParseUint(c.Params.ByName("country_id"), 10, 64)
	if err != nil {
		middleware.RespondErr(c, middleware.APIErrorInvalidRequest, "invalid country id: "+err.Error())
		return
	}

	cityId, err := strconv.ParseUint(c.Params.ByName("city_id"), 10, 64)
	if err != nil {
		middleware.RespondErr(c, middleware.APIErrorInvalidRequest, "invalid city id: "+err.Error())
		return
	}

	serverId, err := strconv.ParseUint(c.Params.ByName("server_id"), 10, 64)
	if err != nil {
		middleware.RespondErr(c, middleware.APIErrorInvalidRequest, "invalid server id: "+err.Error())
		return
	}

	var server models.Server
	tx := vc.DB.First(&server, "id = ? AND city_id = ? AND country_id = ?", &serverId, &cityId, &countryId)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			middleware.RespondErr(c, middleware.APIErrorNotFound, "server not found: "+tx.Error.Error())
		} else {
			reason := "failed to get server: " + tx.Error.Error()
			middleware.RespondErr(c, middleware.APIErrorUnknown, reason)
			vc.Logger.Error(reason)
		}
		return
	}

	if !server.IsActive {
		middleware.RespondErr(c, middleware.APIErrorServerInactive, "server is not active")
		return
	}

	if !server.IsIncludedInPlan {
		middleware.RespondErr(c, middleware.APIErrorServerNotCovered, "server is not available with subscription")
		return
	}

	vc.createCredentials(device, &server, c)
}

func (vc VPNController) createCredentials(device *models.Device, server *models.Server, c *gin.Context) {
	deviceMnemonic, _ := bip39.NewMnemonic(device.WalletEntropy)

	if device.SubscriptionId == nil || device.CurrentBalance <= 0 {
		reason := "wallet is not yet enrolled"
		middleware.RespondErr(c, middleware.APIErrorUnknown, reason)
		vc.Logger.Error(reason)
		return
	}

	var sentinelNodeSubscription *models.SentinelNodeSubscription
	tx := vc.DB.Model(&models.SentinelNodeSubscription{}).Order("id desc").First(&sentinelNodeSubscription, "node_address = ? AND inactive_at > ?", server.Configuration.Data().Address, time.Now())
	if tx.Error != nil {
		if tx.Error != nil {
			if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
				hours, err := strconv.ParseInt(os.Getenv("SENTINEL_NODE_HOURS"), 10, 64)
				if err != nil {
					reason := fmt.Sprintf("failed to parse sentinel node hours: %s", err.Error())
					middleware.RespondErr(c, middleware.APIErrorUnknown, reason)
					vc.Logger.Error(reason)
					return
				}

				s, err := vc.Sentinel.CreateNodeSubscription(vc.Sentinel.ProviderWalletAddress, vc.Sentinel.ProviderMnemonic, server.Configuration.Data().Address, 0, hours)
				if err != nil {
					reason := "failed to create sentinel node subscription: " + err.Error()
					middleware.RespondErr(c, middleware.APIErrorUnknown, reason)
					vc.Logger.Error(reason)
					return
				}

				sentinelNodeSubscription = &models.SentinelNodeSubscription{
					ID:          s.Base.ID,
					NodeAddress: s.NodeAddress,
					InactiveAt:  s.Base.InactiveAt,
				}

				tx := vc.DB.Create(sentinelNodeSubscription)
				if tx.Error != nil {
					reason := "failed to save create sentinel node subscription to the DB: " + tx.Error.Error()
					middleware.RespondErr(c, middleware.APIErrorUnknown, reason)
					vc.Logger.Error(reason)
					return
				}

			} else {
				reason := "failed to get sentinel node subscription from DB: " + tx.Error.Error()
				middleware.RespondErr(c, middleware.APIErrorUnknown, reason)
				vc.Logger.Error(reason)
				return
			}
		}
	}

	credentials, err := vc.Sentinel.CreateCredentials(server.Configuration.Data().Address, *device.SubscriptionId, deviceMnemonic)
	if err != nil {
		reason := "failed to create sentinel credentials: " + err.Error()
		middleware.RespondErr(c, middleware.APIErrorUnknown, reason)
		vc.Logger.Error(reason)
		return
	}

	balance, err := vc.Sentinel.FetchBalance(device.WalletAddress)
	if err != nil {
		reason := "failed to fetch sentinel wallet balance: " + err.Error()
		middleware.RespondErr(c, middleware.APIErrorUnknown, reason)
		vc.Logger.Error(reason)
		return
	}

	device.CurrentBalance = balance
	tx = vc.DB.Save(&device)
	if tx.Error != nil {
		reason := "failed to update device: " + tx.Error.Error()
		middleware.RespondErr(c, middleware.APIErrorUnknown, reason)
		vc.Logger.Error(reason)
		return
	}

	middleware.RespondOK(c, &struct {
		Protocol   string  `json:"protocol"`
		Payload    string  `json:"payload,omitempty"`
		PrivateKey string  `json:"private_key,omitempty"`
		Uid        string  `json:"uid,omitempty"`
		Latitude   float64 `json:"latitude,omitempty"`
		Longitude  float64 `json:"longitude,omitempty"`
	}{
		Protocol:   string(server.Protocols.Data()[0]),
		Payload:    credentials.DTO().Payload,
		PrivateKey: credentials.DTO().PrivateKey,
		Uid:        credentials.DTO().Uid,
		Latitude:   server.Configuration.Data().LocationLat,
		Longitude:  server.Configuration.Data().LocationLon,
	})

}
