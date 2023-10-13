package controllers

import (
	"crypto/sha256"
	"dvpn/middleware"
	"dvpn/models"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/ripemd160"
	"gorm.io/gorm"
	"math/rand"
	"strconv"
	"strings"
	"time"

	bech32 "github.com/cosmos/btcutil/bech32"
	bip32 "github.com/tyler-smith/go-bip32"
	bip39 "github.com/tyler-smith/go-bip39"
)

type DevicesController struct {
	DB     *gorm.DB
	Logger *zap.SugaredLogger
	Auth   *middleware.AuthMiddleware
}

func (dc DevicesController) CreateDevice(c *gin.Context) {
	type requestPayload struct {
		Platform models.DevicePlatform `json:"platform"`
	}

	var payload requestPayload
	if err := c.BindJSON(&payload); err != nil {
		middleware.RespondErr(c, middleware.APIErrorInvalidRequest, "invalid request payload: "+err.Error())
		return
	}

	entropy, _ := bip39.NewEntropy(256)
	mnemonic, _ := bip39.NewMnemonic(entropy)

	seed := bip39.NewSeed(mnemonic, "")
	masterKey, _ := bip32.NewMasterKey(seed)

	childKey, err := deriveKeyFromPath(masterKey, "m/44'/118'/0'/0/0")

	sha256hash := sha256.Sum256(childKey.PublicKey().Key)
	hash := ripemd160.New()
	hash.Write(sha256hash[:])

	converted, err := bech32.ConvertBits(hash.Sum(nil), 8, 5, true)
	if err != nil {
		reason := "failed to convert bits: " + err.Error()
		middleware.RespondErr(c, middleware.APIErrorUnknown, reason)
		dc.Logger.Error(reason)
		return
	}

	walletAddress, err := bech32.Encode("sent", converted)
	if err != nil {
		reason := "failed to encode wallet address: " + err.Error()
		middleware.RespondErr(c, middleware.APIErrorUnknown, reason)
		dc.Logger.Error(reason)
		return
	}

	device := models.Device{
		Platform:       payload.Platform,
		Token:          generateDeviceToken(128),
		WalletAddress:  walletAddress,
		WalletEntropy:  entropy,
		CurrentBalance: 0,
	}

	tx := dc.DB.Create(&device)
	if tx.Error != nil {
		reason := "failed to create device: " + tx.Error.Error()
		middleware.RespondErr(c, middleware.APIErrorUnknown, reason)
		dc.Logger.Error(reason)
		return
	}

	middleware.RespondOK(c, device)
}

func (dc DevicesController) GetDevice(c *gin.Context) {
	device, err := dc.Auth.CurrentDevice(c)
	if err != nil {
		reason := "failed to retrieve device: " + err.Error()
		middleware.RespondErr(c, middleware.APIErrorUnknown, reason)
		dc.Logger.Error(reason)
		return
	}

	middleware.RespondOK(c, device)
}

func generateDeviceToken(l int) string {
	var charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, l)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}

	return string(b)
}

func deriveKeyFromPath(masterKey *bip32.Key, path string) (*bip32.Key, error) {
	const (
		hardenedKeyStart = 0x80000000
	)

	var indexes []uint32

	keys := []string{"Purpose", "CoinType", "Account", "Change", "AccountIndex"}
	segments := strings.Split(path, "/")
	if len(segments) == 0 || segments[0] != "m" {
		return nil, fmt.Errorf("invalid path")
	}

	for _, segment := range segments[1:] {
		segment = strings.TrimRight(segment, "'")
		index, err := strconv.Atoi(segment)
		if err != nil {
			return nil, fmt.Errorf(" segment %s invalid path: %s", segment, err)
		}
		indexes = append(indexes, uint32(index))
	}

	if len(indexes) != 5 {
		return nil, fmt.Errorf("invalid path length")
	}

	pathIndex := make(map[string]uint32)
	for i, k := range keys {
		pathIndex[k] = indexes[i]
	}

	purpose, _ := masterKey.NewChildKey(pathIndex["Purpose"] + hardenedKeyStart)
	coinType, _ := purpose.NewChildKey(pathIndex["CoinType"] + hardenedKeyStart)
	account, _ := coinType.NewChildKey(pathIndex["Account"] + hardenedKeyStart)
	change, _ := account.NewChildKey(pathIndex["Change"])
	child, _ := change.NewChildKey(pathIndex["AccountIndex"])

	return child, nil
}
