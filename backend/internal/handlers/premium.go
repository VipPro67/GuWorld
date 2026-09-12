package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

const (
	CurrencyPrimevalStone        = "PRIMEVAL_STONE"
	CurrencyImmortalEssenceStone = "IMMORTAL_ESSENCE_STONE"
	CurrencyImmortalJade         = "IMMORTAL_JADE"
)

type WalletDTO struct {
	PrimevalStone        int64 `json:"primevalStone"`
	ImmortalEssenceStone int64 `json:"immortalEssenceStone"`
	ImmortalJade         int64 `json:"immortalJade"`
}

type TopupPackage struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	AmountVND int64  `json:"amountVnd"`
	Jade      int64  `json:"jadeAmount"`
}

func handlerCSVPath(filename string) string {
	candidates := []string{
		filepath.Join("data", filename),
		filepath.Join("cmd/server/data", filename),
		filename,
		filepath.Join("backend", filename),
		filepath.Join("..", filename),
		filepath.Join("..", "..", filename),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filename
}

func loadTopupPackages() ([]TopupPackage, error) {
	var packages []TopupPackage
	if err := cache.GetJSON("topup_packages", &packages); err == nil && len(packages) > 0 {
		return packages, nil
	}

	file, err := os.Open(handlerCSVPath("TopupPackage.csv"))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) < 1 {
		return []TopupPackage{}, nil
	}
	header := rows[0]
	col := func(name string) int {
		for i, h := range header {
			if strings.TrimPrefix(strings.TrimSpace(h), "\uFEFF") == name {
				return i
			}
		}
		return -1
	}
	idIdx, nameIdx, amountIdx, jadeIdx := col("id"), col("name"), col("amount_vnd"), col("jade")
	if idIdx < 0 || nameIdx < 0 || amountIdx < 0 || jadeIdx < 0 {
		return nil, fmt.Errorf("TopupPackage.csv missing required columns")
	}
	get := func(row []string, idx int) string {
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}
	packages = []TopupPackage{}
	for _, row := range rows[1:] {
		id := get(row, idIdx)
		if id == "" {
			continue
		}
		var amount, jade int64
		fmt.Sscanf(get(row, amountIdx), "%d", &amount)
		fmt.Sscanf(get(row, jadeIdx), "%d", &jade)
		packages = append(packages, TopupPackage{ID: id, Name: get(row, nameIdx), AmountVND: amount, Jade: jade})
	}
	_ = cache.SetJSON("topup_packages", packages, 24*time.Hour)
	return packages, nil
}

type PackageReward struct {
	Currency string `json:"currency,omitempty"`
	Amount   int64  `json:"amount,omitempty"`
	ItemID   string `json:"itemId,omitempty"`
	ItemName string `json:"itemName,omitempty"`
	Quantity int64  `json:"quantity,omitempty"`
	Note     string `json:"note,omitempty"`
}

type ShopLimit struct {
	Count  int    `json:"count"`
	Period string `json:"period"`
}

type ShopRequirement struct {
	MinRank    int  `json:"minRank,omitempty"`
	MinSubRank int  `json:"minSubRank,omitempty"`
	TODO       bool `json:"todo,omitempty"`
}

type PremiumShopPackage struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	PriceJade    int64            `json:"priceJade"`
	Rewards      []PackageReward  `json:"rewards"`
	Requirements *ShopRequirement `json:"requirements,omitempty"`
	Limit        ShopLimit        `json:"limit"`
	Description  string           `json:"description"`
}

type ShopPackageDTO struct {
	PremiumShopPackage
	RemainingPurchaseCount *int `json:"remainingPurchaseCount,omitempty"`
}

func loadPremiumShopPackages() ([]PremiumShopPackage, error) {
	var packages []PremiumShopPackage
	if err := cache.GetJSON("premium_shop_packages", &packages); err == nil && len(packages) > 0 {
		return packages, nil
	}

	file, err := os.Open(handlerCSVPath("PremiumShopPackage.csv"))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) < 1 {
		return []PremiumShopPackage{}, nil
	}
	header := rows[0]
	col := func(name string) int {
		for i, h := range header {
			if strings.TrimPrefix(strings.TrimSpace(h), "\uFEFF") == name {
				return i
			}
		}
		return -1
	}
	idIdx, nameIdx, priceIdx, rewardsIdx := col("id"), col("name"), col("price_jade"), col("rewards")
	descIdx, minRankIdx, minSubRankIdx, limitCountIdx, limitPeriodIdx := col("description"), col("min_rank"), col("min_sub_rank"), col("limit_count"), col("limit_period")
	if idIdx < 0 || nameIdx < 0 || priceIdx < 0 || rewardsIdx < 0 || limitCountIdx < 0 || limitPeriodIdx < 0 {
		return nil, fmt.Errorf("PremiumShopPackage.csv missing required columns")
	}
	get := func(row []string, idx int) string {
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}
	packages = []PremiumShopPackage{}
	for _, row := range rows[1:] {
		id := get(row, idIdx)
		if id == "" {
			continue
		}
		var price int64
		var limitCount, minRank, minSubRank int
		fmt.Sscanf(get(row, priceIdx), "%d", &price)
		fmt.Sscanf(get(row, limitCountIdx), "%d", &limitCount)
		fmt.Sscanf(get(row, minRankIdx), "%d", &minRank)
		fmt.Sscanf(get(row, minSubRankIdx), "%d", &minSubRank)
		rewards := []PackageReward{}
		if raw := get(row, rewardsIdx); raw != "" {
			if err := json.Unmarshal([]byte(raw), &rewards); err != nil {
				return nil, fmt.Errorf("parse rewards for package %s: %w", id, err)
			}
		}
		var req *ShopRequirement
		if minRank > 0 || minSubRank > 0 {
			req = &ShopRequirement{MinRank: minRank, MinSubRank: minSubRank}
		}
		packages = append(packages, PremiumShopPackage{ID: id, Name: get(row, nameIdx), PriceJade: price, Rewards: rewards, Requirements: req, Limit: ShopLimit{Count: limitCount, Period: get(row, limitPeriodIdx)}, Description: get(row, descIdx)})
	}
	_ = cache.SetJSON("premium_shop_packages", packages, 24*time.Hour)
	return packages, nil
}

func findTopupPackage(id string) (TopupPackage, bool) {
	packages, err := loadTopupPackages()
	if err != nil {
		return TopupPackage{}, false
	}
	for _, pkg := range packages {
		if pkg.ID == id {
			return pkg, true
		}
	}
	return TopupPackage{}, false
}

func findPremiumShopPackage(id string) (PremiumShopPackage, bool) {
	packages, err := loadPremiumShopPackages()
	if err != nil {
		return PremiumShopPackage{}, false
	}
	for _, pkg := range packages {
		if pkg.ID == id {
			return pkg, true
		}
	}
	return PremiumShopPackage{}, false
}

func GetWallet(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	wallet, err := ensureWalletContext(c.Request.Context(), db.DB, playerID)
	if err != nil {
		logger.S.Errorf("GetWallet failed for player %s: %v", playerID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải dữ liệu ví"})
		return
	}
	c.JSON(http.StatusOK, wallet)
}

func GetTopupPackages(c *gin.Context) {
	packages, err := loadTopupPackages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải gói nạp"})
		return
	}
	c.JSON(http.StatusOK, packages)
}

type createTopupOrderRequest struct {
	PackageID string `json:"packageId" binding:"required"`
}

func CreateTopupOrder(c *gin.Context) {
	c.JSON(http.StatusBadRequest, gin.H{"error": "Chức năng nạp tiền hiện đang tạm ngưng"})
}

func GetTopupOrder(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	orderCode, err := strconv.ParseInt(c.Param("orderCode"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "orderCode không hợp lệ"})
		return
	}
	var row struct {
		OrderCode int64
		AmountVND int64
		Jade      int64
		Status    string
		Checkout  sql.NullString
		QR        sql.NullString
		CreatedAt time.Time
		PaidAt    sql.NullTime
	}
	err = db.DB.QueryRowContext(c.Request.Context(), `SELECT order_code, amount_vnd, jade_amount, status, checkout_url, qr_code, created_at, paid_at FROM topup_orders WHERE order_code=$1 AND user_id=$2`, orderCode, playerID).Scan(&row.OrderCode, &row.AmountVND, &row.Jade, &row.Status, &row.Checkout, &row.QR, &row.CreatedAt, &row.PaidAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy đơn nạp"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải đơn nạp"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"orderCode": row.OrderCode, "amountVnd": row.AmountVND, "jadeAmount": row.Jade, "status": row.Status, "checkoutUrl": row.Checkout.String, "qrCode": row.QR.String, "createdAt": row.CreatedAt, "paidAt": nullableTime(row.PaidAt)})
}

type payOSWebhook struct {
	Code      string                 `json:"code"`
	Desc      string                 `json:"desc"`
	Success   bool                   `json:"success"`
	Data      map[string]interface{} `json:"data"`
	Signature string                 `json:"signature"`
}

func PayOSWebhook(c *gin.Context) {
	var payload payOSWebhook
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if !verifyPayOSSignature(payload.Data, payload.Signature) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook"})
		return
	}
	dataCode := fmt.Sprint(payload.Data["code"])
	if !payload.Success || payload.Code != "00" || dataCode != "00" {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Webhook verified; payment not successful"})
		return
	}
	orderCode, err := numberFromAny(payload.Data["orderCode"])
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid orderCode"})
		return
	}
	amount, err := numberFromAny(payload.Data["amount"])
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid amount"})
		return
	}

	if err = applyPaidTopup(orderCode, amount); err != nil {
		auditlog.Log(db.DB, auditlog.Entry{
			Action: "topup.webhook_failed", EntityType: "topup_order",
			EntityID:       fmt.Sprint(orderCode),
			RequestSummary: fmt.Sprintf("orderCode=%d amount=%d err=%s", orderCode, amount, err.Error()),
			IPAddress:      c.ClientIP(), Success: false, ErrorMessage: err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	auditlog.Log(db.DB, auditlog.Entry{
		Action: "topup.webhook_applied", EntityType: "topup_order",
		EntityID:       fmt.Sprint(orderCode),
		RequestSummary: fmt.Sprintf("orderCode=%d amount=%d", orderCode, amount),
		IPAddress:      c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func GetPremiumShopPackages(c *gin.Context) {
	var playerID uuid.UUID
	_, authed := middleware.GetPlayerID(c)
	if authed {
		playerID, _ = middleware.GetPlayerID(c)
	}
	packages, err := loadPremiumShopPackages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải gói shop"})
		return
	}
	result := make([]ShopPackageDTO, 0, len(packages))
	for _, pkg := range packages {
		dto := ShopPackageDTO{PremiumShopPackage: pkg}
		if authed {
			remaining, _ := remainingPurchases(db.DB, playerID, pkg)
			dto.RemainingPurchaseCount = &remaining
		}
		result = append(result, dto)
	}
	c.JSON(http.StatusOK, result)
}

type buyPremiumPackageRequest struct {
	PackageID string `json:"packageId" binding:"required"`
}

func BuyPremiumPackage(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req buyPremiumPackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pkg, ok := findPremiumShopPackage(req.PackageID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gói shop không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể khóa người chơi"})
		return
	}
	before := append(models.GuInventory(nil), player.GuInventory...)
	if pkg.Requirements != nil {
		if player.Rank < pkg.Requirements.MinRank || (player.Rank == pkg.Requirements.MinRank && pkg.Requirements.MinSubRank > 0 && player.SubRank < pkg.Requirements.MinSubRank) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Chưa đạt yêu cầu tu vi để mua gói"})
			return
		}
	}

	wallet, err := ensureWalletTx(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải ví"})
		return
	}
	remaining, err := remainingPurchases(tx, playerID, pkg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra giới hạn mua"})
		return
	}
	if remaining <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đã vượt giới hạn mua gói"})
		return
	}
	if wallet.ImmortalJade < pkg.PriceJade {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Tiên Ngọc"})
		return
	}

	jadeBefore := wallet.ImmortalJade
	wallet.ImmortalJade -= pkg.PriceJade
	if _, err = tx.Exec(`UPDATE wallets SET immortal_jade=$1, updated_at=CURRENT_TIMESTAMP WHERE user_id=$2`, wallet.ImmortalJade, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể trừ Tiên Ngọc"})
		return
	}
	if err = insertWalletTx(tx, playerID, "SPEND", CurrencyImmortalJade, -pkg.PriceJade, jadeBefore, wallet.ImmortalJade, nil, &pkg.ID, "Mua gói tu luyện"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể ghi giao dịch trừ Tiên Ngọc"})
		return
	}

	for _, reward := range pkg.Rewards {
		switch reward.Currency {
		case CurrencyPrimevalStone:
			before := player.PrimevalStones
			player.PrimevalStones += reward.Amount
			wallet.PrimevalStone = player.PrimevalStones
			if err = insertWalletTx(tx, playerID, "PACKAGE_REWARD", CurrencyPrimevalStone, reward.Amount, before, player.PrimevalStones, nil, &pkg.ID, "Thưởng gói tu luyện"); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể ghi giao dịch thưởng Nguyên Thạch"})
				return
			}
		case CurrencyImmortalEssenceStone:
			before := player.ImmortalStones
			player.ImmortalStones += reward.Amount
			wallet.ImmortalEssenceStone = player.ImmortalStones
			if err = insertWalletTx(tx, playerID, "PACKAGE_REWARD", CurrencyImmortalEssenceStone, reward.Amount, before, player.ImmortalStones, nil, &pkg.ID, "Thưởng gói tu luyện"); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể ghi giao dịch thưởng Tiên Nguyên Thạch"})
				return
			}
		default:
			if reward.ItemID != "" {
				if strings.HasPrefix(reward.ItemID, "GU_") {
					if spec, exists := models.GetGuAny(reward.ItemID); exists {
						for i := int64(0); i < reward.Quantity; i++ {
							player.GuInventory = append(player.GuInventory, models.GuInstance{InstanceID: uuid.New().String(), GuID: spec.ID, Rank: spec.Rank, Hunger: 100, Status: "ACTIVE", Equipped: false})
						}
					}
				} else {
					InventorySvc.AddItem(tx, playerID.String(), reward.ItemID, "materials", reward.Quantity)
				}
			}
		}
	}

	if err := db.SyncPlayerGus(tx, playerID.String(), before, player.GuInventory); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật phần thưởng"})
		return
	}
	if _, err = tx.Exec(`UPDATE players SET primeval_stones=$1, immortal_stones=$2 WHERE id=$3`, player.PrimevalStones, player.ImmortalStones, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật phần thưởng"})
		return
	}
	if _, err = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 10, $1) WHERE id = $2", MaxHumanQi, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật Nhân Khí"})
		return
	}
	if _, err = tx.Exec(`UPDATE wallets SET immortal_jade=$1, updated_at=CURRENT_TIMESTAMP WHERE user_id=$2`, wallet.ImmortalJade, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật ví"})
		return
	}
	rewardsJSON, _ := json.Marshal(pkg.Rewards)
	if _, err = tx.Exec(`INSERT INTO shop_package_purchases (user_id, package_id, price_jade, rewards_snapshot) VALUES ($1,$2,$3,$4)`, playerID, pkg.ID, pkg.PriceJade, string(rewardsJSON)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể ghi lịch sử mua gói"})
		return
	}
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "shop.buy_package", EntityType: "premium_package",
		EntityID:       pkg.ID,
		RequestSummary: fmt.Sprintf("Gói %s | giá %d Tiên Ngọc | %d phần thưởng", pkg.ID, pkg.PriceJade, len(pkg.Rewards)),
		Metadata:       map[string]interface{}{"price_jade": pkg.PriceJade, "jade_before": jadeBefore, "jade_after": wallet.ImmortalJade},
		IPAddress:      c.ClientIP(), Success: true,
	})
	responseWithPlayer(c, http.StatusOK, player, gin.H{"message": "Mua gói thành công. Nhân Khí +10.", "wallet": wallet, "rewards": pkg.Rewards})
}

const (
	// Exchange rates between Nguyên Thạch (primeval) and Tiên Nguyên Thạch (immortal).
	ExchangeImmortalToPrimevalRate = 100    // 1 Tiên Nguyên Thạch = 100 Nguyên Thạch
	ExchangePrimevalToImmortalCost = 100000 // 100000 Nguyên Thạch = 1 Tiên Nguyên Thạch
)

type exchangeStonesRequest struct {
	// Currency is the stone the player spends: PRIMEVAL_STONE or IMMORTAL_ESSENCE_STONE.
	Currency string `json:"currency" binding:"required"`
	Amount   int64  `json:"amount" binding:"required,gt=0"`
}

func ExchangeStones(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req exchangeStonesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// immortalGain / primevalGain track the amounts added in each direction.
	var immortalGain, primevalGain int64
	switch req.Currency {
	case CurrencyImmortalEssenceStone:
		if req.Amount > math.MaxInt64/ExchangeImmortalToPrimevalRate {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Số lượng quá lớn"})
			return
		}
		primevalGain = req.Amount * ExchangeImmortalToPrimevalRate
	case CurrencyPrimevalStone:
		if req.Amount%ExchangePrimevalToImmortalCost != 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Số Nguyên Thạch phải là bội số của 100.000"})
			return
		}
		immortalGain = req.Amount / ExchangePrimevalToImmortalCost
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại tiền tệ không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể khóa người chơi"})
		return
	}
	wallet, err := ensureWalletTx(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải ví"})
		return
	}

	primevalBefore := player.PrimevalStones
	immortalBefore := player.ImmortalStones
	switch req.Currency {
	case CurrencyImmortalEssenceStone:
		if player.ImmortalStones < req.Amount {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Tiên Nguyên Thạch"})
			return
		}
		player.ImmortalStones -= req.Amount
		player.PrimevalStones += primevalGain
	default:
		if player.PrimevalStones < req.Amount {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Nguyên Thạch"})
			return
		}
		player.PrimevalStones -= req.Amount
		player.ImmortalStones += immortalGain
	}
	wallet.PrimevalStone = player.PrimevalStones
	wallet.ImmortalEssenceStone = player.ImmortalStones

	if _, err = tx.Exec(`UPDATE players SET primeval_stones=$1, immortal_stones=$2 WHERE id=$3`, player.PrimevalStones, player.ImmortalStones, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật đá"})
		return
	}
	// Stones are authoritative in the players table, no need to update wallets

	if err = insertWalletTx(tx, playerID, "EXCHANGE", req.Currency, -req.Amount, func() int64 {
		if req.Currency == CurrencyImmortalEssenceStone {
			return immortalBefore
		}
		return primevalBefore
	}(), func() int64 {
		if req.Currency == CurrencyImmortalEssenceStone {
			return int64(wallet.ImmortalEssenceStone)
		}
		return int64(wallet.PrimevalStone)
	}(), nil, nil, "Đổi sang loại đá khác"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể ghi giao dịch trừ đá"})
		return
	}
	if err = insertWalletTx(tx, playerID, "EXCHANGE", func() string {
		if req.Currency == CurrencyImmortalEssenceStone {
			return CurrencyPrimevalStone
		}
		return CurrencyImmortalEssenceStone
	}(), func() int64 {
		if req.Currency == CurrencyImmortalEssenceStone {
			return primevalGain
		}
		return immortalGain
	}(), func() int64 {
		if req.Currency == CurrencyImmortalEssenceStone {
			return primevalBefore
		}
		return immortalBefore
	}(), func() int64 {
		if req.Currency == CurrencyImmortalEssenceStone {
			return int64(wallet.PrimevalStone)
		}
		return int64(wallet.ImmortalEssenceStone)
	}(), nil, nil, "Nhận từ đổi đá"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể ghi giao dịch nhận đá"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "wallet.exchange", EntityType: "currency",
		RequestSummary: fmt.Sprintf("%s %d -> %s %d", req.Currency, req.Amount, func() string {
			if req.Currency == CurrencyImmortalEssenceStone {
				return fmt.Sprintf("%s %d", CurrencyPrimevalStone, primevalGain)
			}
			return fmt.Sprintf("%s %d", CurrencyImmortalEssenceStone, immortalGain)
		}(), func() int64 {
			if req.Currency == CurrencyImmortalEssenceStone {
				return primevalGain
			}
			return immortalGain
		}()),
		IPAddress: c.ClientIP(), Success: true,
	})
	responseWithPlayer(c, http.StatusOK, player, gin.H{"message": "Đổi đá thành công", "wallet": wallet})
}

func ensureWallet(database *sql.DB, playerID uuid.UUID) (WalletDTO, error) {
	return ensureWalletContext(context.Background(), database, playerID)
}

func ensureWalletContext(ctx context.Context, database *sql.DB, playerID uuid.UUID) (WalletDTO, error) {
	var w WalletDTO
	// Stone balances are authoritative in players. If the player row is missing
	// (e.g. the account is not present in this DB), fall back to the wallets
	// mirror so the display never breaks.
	err := database.QueryRowContext(ctx, `SELECT COALESCE(primeval_stones,0)::BIGINT, COALESCE(immortal_stones,0)::BIGINT FROM players WHERE id=$1`, playerID).Scan(&w.PrimevalStone, &w.ImmortalEssenceStone)
	if err != nil && err != sql.ErrNoRows {
		return WalletDTO{}, err
	}
	// Jade only lives in wallets.
	err = database.QueryRowContext(ctx, `SELECT immortal_jade FROM wallets WHERE user_id=$1`, playerID).Scan(&w.ImmortalJade)
	if err != nil {
		if err == sql.ErrNoRows {
			if _, ierr := database.ExecContext(ctx, `INSERT INTO wallets (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`, playerID); ierr != nil {
				logger.S.Warnf("ensureWallet backfill %s: %v", playerID, ierr)
			}
			w.ImmortalJade = 0
		} else {
			logger.S.Warnf("ensureWallet jade read %s: %v", playerID, err)
			w.ImmortalJade = 0
		}
	}
	return w, nil
}
func ensureWalletTx(tx *sql.Tx, playerID uuid.UUID) (WalletDTO, error) {
	var w WalletDTO

	if err := tx.QueryRow(`
		SELECT
			COALESCE(primeval_stones, 0)::BIGINT,
			COALESCE(immortal_stones, 0)::BIGINT
		FROM players
		WHERE id = $1
	`, playerID).Scan(
		&w.PrimevalStone,
		&w.ImmortalEssenceStone,
	); err != nil {
		return WalletDTO{}, err
	}

	if _, err := tx.Exec(`
		INSERT INTO wallets (user_id)
		VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING
	`, playerID); err != nil {
		return WalletDTO{}, err
	}

	if err := tx.QueryRow(`
		SELECT immortal_jade
		FROM wallets
		WHERE user_id = $1
		FOR UPDATE
	`, playerID).Scan(&w.ImmortalJade); err != nil {
		return WalletDTO{}, err
	}

	return w, nil
}
func createPayOSPaymentLink(orderCode int64, pkg TopupPackage, description, returnURL, cancelURL string) (string, string, error) {
	clientID := os.Getenv("PAYOS_CLIENT_ID")
	apiKey := os.Getenv("PAYOS_API_KEY")
	checksumKey := os.Getenv("PAYOS_CHECKSUM_KEY")
	if clientID == "" || apiKey == "" || checksumKey == "" {
		return "", "", fmt.Errorf("missing PAYOS_CLIENT_ID/PAYOS_API_KEY/PAYOS_CHECKSUM_KEY")
	}
	sigPayload := fmt.Sprintf("amount=%d&cancelUrl=%s&description=%s&orderCode=%d&returnUrl=%s", pkg.AmountVND, cancelURL, description, orderCode, returnURL)
	signature := hmacSHA256Hex(sigPayload, checksumKey)
	body := map[string]interface{}{
		"orderCode": orderCode, "amount": pkg.AmountVND, "description": description,
		"items":     []map[string]interface{}{{"name": pkg.Name, "quantity": 1, "price": pkg.AmountVND}},
		"cancelUrl": cancelURL, "returnUrl": returnURL, "signature": signature,
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, "https://api-merchant.payos.vn/v2/payment-requests", bytes.NewReader(raw))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-client-id", clientID)
	req.Header.Set("x-api-key", apiKey)
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var decoded struct {
		Code string `json:"code"`
		Desc string `json:"desc"`
		Data struct {
			CheckoutURL string `json:"checkoutUrl"`
			QRCode      string `json:"qrCode"`
		} `json:"data"`
	}
	_ = json.Unmarshal(respBody, &decoded)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.S.Errorf("payOS Payment Link API error status=%d body=%s", resp.StatusCode, string(respBody))
		if decoded.Desc != "" {
			return "", "", fmt.Errorf("payOS error (%s): %s", decoded.Code, decoded.Desc)
		}
		return "", "", fmt.Errorf("payOS HTTP status %d: %s", resp.StatusCode, string(respBody))
	}
	if decoded.Code != "00" {
		logger.S.Errorf("payOS payment link code error code=%s desc=%s", decoded.Code, decoded.Desc)
		return "", "", fmt.Errorf("payOS error (%s): %s", decoded.Code, decoded.Desc)
	}
	return decoded.Data.CheckoutURL, decoded.Data.QRCode, nil
}

func verifyPayOSSignature(data map[string]interface{}, signature string) bool {
	key := os.Getenv("PAYOS_CHECKSUM_KEY")
	if key == "" || signature == "" {
		return false
	}
	keys := make([]string, 0, len(data))
	for k := range data {
		if k == "signature" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, payOSValueString(data[k])))
	}
	computed := hmacSHA256Hex(strings.Join(parts, "&"), key)
	return hmac.Equal([]byte(strings.ToLower(computed)), []byte(strings.ToLower(signature)))
}

func payOSValueString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		if val == "undefined" || val == "null" {
			return ""
		}
		return val
	case []interface{}:
		for _, ele := range val {
			if m, ok := ele.(map[string]interface{}); ok {
				_ = m
			}
		}
		b, _ := json.Marshal(val)
		return string(b)
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	default:
		return fmt.Sprint(val)
	}
}

func hmacSHA256Hex(message, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

func applyPaidTopup(orderCode, amount int64) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var userID uuid.UUID
	var orderAmount, jadeAmount int64
	var status string
	err = tx.QueryRow(`SELECT user_id, amount_vnd, jade_amount, status FROM topup_orders WHERE order_code=$1 FOR UPDATE`, orderCode).Scan(&userID, &orderAmount, &jadeAmount, &status)
	if err == sql.ErrNoRows {
		return fmt.Errorf("order not found")
	}
	if err != nil {
		return err
	}
	if amount != orderAmount {
		return fmt.Errorf("webhook amount mismatch")
	}
	if status == "PAID" {
		return tx.Commit()
	}
	if status != "PENDING" {
		return fmt.Errorf("order is not pending")
	}
	wallet, err := ensureWalletTx(tx, userID)
	if err != nil {
		return err
	}
	before := wallet.ImmortalJade
	after := before + jadeAmount
	if _, err = tx.Exec(`UPDATE wallets SET immortal_jade=$1, updated_at=CURRENT_TIMESTAMP WHERE user_id=$2`, after, userID); err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE topup_orders SET status='PAID', paid_at=CURRENT_TIMESTAMP WHERE order_code=$1`, orderCode); err != nil {
		return err
	}
	if err = insertWalletTx(tx, userID, "TOPUP", CurrencyImmortalJade, jadeAmount, before, after, &orderCode, nil, "Nạp Tiên Ngọc qua payOS"); err != nil {
		return err
	}
	return tx.Commit()
}

func insertWalletTx(tx *sql.Tx, userID uuid.UUID, txType, currency string, amount, before, after int64, refOrder *int64, refPackage *string, note string) error {
	_, err := tx.Exec(`INSERT INTO wallet_transactions (user_id, type, currency, amount, balance_before, balance_after, ref_order_code, ref_package_id, note) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, userID, txType, currency, amount, before, after, refOrder, refPackage, note)
	return err
}

type queryer interface {
	QueryRow(string, ...interface{}) *sql.Row
}

func remainingPurchases(q queryer, playerID uuid.UUID, pkg PremiumShopPackage) (int, error) {
	periodStart := time.Time{}
	switch pkg.Limit.Period {
	case "ACCOUNT":
		periodStart = time.Unix(0, 0)
	case "WEEK":
		now := time.Now()
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		periodStart = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
	case "MONTH":
		now := time.Now()
		periodStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	default:
		periodStart = time.Unix(0, 0)
	}
	var count int
	err := q.QueryRow(`SELECT COUNT(*) FROM shop_package_purchases WHERE user_id=$1 AND package_id=$2 AND created_at >= $3`, playerID, pkg.ID, periodStart).Scan(&count)
	if err != nil {
		return 0, err
	}
	return pkg.Limit.Count - count, nil
}

func requireAdmin(c *gin.Context) (uuid.UUID, bool) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return uuid.Nil, false
	}
	if db.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database unavailable"})
		return uuid.Nil, false
	}
	var isAdmin bool
	if err := db.DB.QueryRow(`SELECT COALESCE(is_admin, false) FROM players WHERE id=$1`, playerID).Scan(&isAdmin); err != nil || !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin required"})
		return uuid.Nil, false
	}
	return playerID, true
}

func AdminListTopupOrders(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT order_code, user_id, amount_vnd, jade_amount, status, provider, checkout_url, created_at, paid_at FROM topup_orders ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách đơn nạp"})
		return
	}
	defer rows.Close()
	orders := []gin.H{}
	for rows.Next() {
		var orderCode, amountVND, jadeAmount int64
		var userID uuid.UUID
		var status, provider string
		var checkout sql.NullString
		var createdAt time.Time
		var paidAt sql.NullTime
		if err := rows.Scan(&orderCode, &userID, &amountVND, &jadeAmount, &status, &provider, &checkout, &createdAt, &paidAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc danh sách đơn nạp"})
			return
		}
		orders = append(orders, gin.H{"orderCode": orderCode, "userId": userID, "amountVnd": amountVND, "jadeAmount": jadeAmount, "status": status, "provider": provider, "checkoutUrl": checkout.String, "createdAt": createdAt, "paidAt": nullableTime(paidAt)})
	}
	c.JSON(http.StatusOK, orders)
}

func AdminListWalletTransactions(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	userID := c.Query("userId")
	query := `SELECT id, user_id, type, currency, amount, balance_before, balance_after, ref_order_code, ref_package_id, note, created_at FROM wallet_transactions`
	args := []interface{}{}
	if userID != "" {
		query += ` WHERE user_id=$1`
		args = append(args, userID)
	}
	query += ` ORDER BY created_at DESC LIMIT 200`
	rows, err := db.DB.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải giao dịch ví"})
		return
	}
	defer rows.Close()
	transactions := []gin.H{}
	for rows.Next() {
		var id, uid uuid.UUID
		var txType, currency string
		var amount, before, after int64
		var refOrder sql.NullInt64
		var refPackage, note sql.NullString
		var createdAt time.Time
		if err := rows.Scan(&id, &uid, &txType, &currency, &amount, &before, &after, &refOrder, &refPackage, &note, &createdAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc giao dịch ví"})
			return
		}
		transactions = append(transactions, gin.H{"id": id, "userId": uid, "type": txType, "currency": currency, "amount": amount, "balanceBefore": before, "balanceAfter": after, "refOrderCode": refOrder.Int64, "refPackageId": refPackage.String, "note": note.String, "createdAt": createdAt})
	}
	c.JSON(http.StatusOK, transactions)
}

func numberFromAny(v interface{}) (int64, error) {
	switch val := v.(type) {
	case float64:
		return int64(val), nil
	case int64:
		return val, nil
	case json.Number:
		return val.Int64()
	case string:
		return strconv.ParseInt(val, 10, 64)
	default:
		return 0, fmt.Errorf("not a number")
	}
}

func nullableTime(t sql.NullTime) interface{} {
	if t.Valid {
		return t.Time
	}
	return nil
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
