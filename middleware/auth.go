package middleware

import (
	"dvpn/core"
	"dvpn/models"
	"errors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AuthMiddleware struct {
	DB     *gorm.DB
	Logger *zap.SugaredLogger
}

func (am AuthMiddleware) RequireAuth(c *gin.Context) {
	token := c.GetHeader("x-device-token")
	if len(token) == 0 {
		RespondErr(c, APIErrorUnauthorizedDevice, "device token is required")
		return
	}

	db, err := core.GetDB()
	if err != nil {
		reason := "failed to get db: " + err.Error()
		RespondErr(c, APIErrorUnknown, reason)
		am.Logger.Error(reason)
		return
	}

	var device models.Device
	tx := db.First(&device, "token = ?", token)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			RespondErr(c, APIErrorUnauthorizedDevice, "invalid device token")
			return
		}

		reason := "failed to find device by token: " + tx.Error.Error()
		RespondErr(c, APIErrorUnknown, reason)
		am.Logger.Error(reason)
		return
	}

	if device.IsBanned {
		RespondErr(c, APIErrorBannedDevice, "device is banned")
		return
	}

	c.Set("currentDeviceID", device.ID)
	c.Next()
}

func (am AuthMiddleware) CurrentDeviceID(c *gin.Context) (*uint, error) {
	value, exist := c.Get("currentDeviceID")
	if exist == false {
		return nil, errors.New("deviceID not found in context")
	}

	deviceID := value.(uint)

	return &deviceID, nil
}

func (am AuthMiddleware) CurrentDevice(c *gin.Context) (*models.Device, error) {
	deviceID, err := am.CurrentDeviceID(c)
	if err != nil {
		return nil, err
	}

	var device models.Device
	tx := am.DB.First(&device, "id = ?", *deviceID)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return &device, nil
}
