package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
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

var topupPackages = []TopupPackage{
	{ID: "jade_60", Name: "60 Tiên Ngọc", AmountVND: 10000, Jade: 60},
	{ID: "jade_130", Name: "130 Tiên Ngọc", AmountVND: 20000, Jade: 130},
	{ID: "jade_350", Name: "350 Tiên Ngọc", AmountVND: 50000, Jade: 350},
	{ID: "jade_750", Name: "750 Tiên Ngọc", AmountVND: 100000, Jade: 750},
	{ID: "jade_1600", Name: "1600 Tiên Ngọc", AmountVND: 200000, Jade: 1600},
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

var premiumShopPackages = []PremiumShopPackage{
	{
		ID: "STARTER_CULTIVATION_PACK", Name: "Gói Tân Thủ Tu Luyện", PriceJade: 60,
		Description: "Tài nguyên nhập môn để ổn định tu hành. TODO: thay green_bamboo_wine bằng MANA_RECOVERY_POTION khi seed item đó tồn tại.",
		Rewards: []PackageReward{
			{Currency: CurrencyPrimevalStone, Amount: 30000},
			{ItemID: "green_bamboo_wine", ItemName: "Rượu Thanh Trúc", Quantity: 5},
			{ItemID: "grass_puppet", ItemName: "Hình nộm người cỏ", Quantity: 3},
		},
		Limit: ShopLimit{Count: 1, Period: "ACCOUNT"},
	},
	{
		ID: "MORTAL_ADVANCE_PACK", Name: "Gói Phàm Tu Tinh Tiến", PriceJade: 180,
		Description: "Gói hỗ trợ phàm tu. TODO: RELIC_GU_SAME_RANK đang được biểu diễn bằng GU_RELIC nếu dictionary hỗ trợ.",
		Rewards: []PackageReward{
			{Currency: CurrencyPrimevalStone, Amount: 120000},
			{ItemID: "moon_orchid_petal", ItemName: "Cánh hoa Nguyệt Lan", Quantity: 10},
			{ItemID: "GU_RELIC", ItemName: "Xá Lợi Cổ", Quantity: 1, Note: "TODO: chỉ trao nếu Gu dictionary có GU_RELIC."},
		},
		Limit: ShopLimit{Count: 2, Period: "MONTH"},
	},
	{
		ID: "ASCENSION_PREP_PACK", Name: "Gói Thăng Tiên Dự Bị", PriceJade: 680,
		Description: "Chuẩn bị tài nguyên trước lúc thăng tiên. TODO: item tăng Nhân Khí chưa có seed riêng.",
		Rewards: []PackageReward{
			{Currency: CurrencyPrimevalStone, Amount: 500000},
			{Currency: CurrencyImmortalEssenceStone, Amount: 1},
			{ItemID: "azure_jade_bamboo", ItemName: "Ngọc Trúc Bích Không", Quantity: 5},
		},
		Requirements: &ShopRequirement{MinRank: 5},
		Limit:        ShopLimit{Count: 1, Period: "MONTH"},
	},
	{
		ID: "IMMORTAL_BEGINNER_PACK", Name: "Gói Tiên Nhân Sơ Cấp", PriceJade: 980,
		Description: "Gói dành cho Cổ Tiên mới tấn thăng. TODO: item giảm tổn thất luyện Tiên Cổ chưa có seed riêng.",
		Rewards: []PackageReward{
			{Currency: CurrencyImmortalEssenceStone, Amount: 3},
			{ItemID: "ice_tear", ItemName: "Lệ Băng", Quantity: 10},
		},
		Requirements: &ShopRequirement{MinRank: 6},
		Limit:        ShopLimit{Count: 2, Period: "MONTH"},
	},
}

func findTopupPackage(id string) (TopupPackage, bool) {
	for _, pkg := range topupPackages {
		if pkg.ID == id {
			return pkg, true
		}
	}
	return TopupPackage{}, false
}

func findPremiumShopPackage(id string) (PremiumShopPackage, bool) {
	for _, pkg := range premiumShopPackages {
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
	wallet, err := ensureWallet(db.DB, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải ví"})
		return
	}
	c.JSON(http.StatusOK, wallet)
}

func GetTopupPackages(c *gin.Context) {
	c.JSON(http.StatusOK, topupPackages)
}

type createTopupOrderRequest struct {
	PackageID string `json:"packageId" binding:"required"`
}

func CreateTopupOrder(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req createTopupOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pkg, ok := findTopupPackage(req.PackageID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gói nạp không hợp lệ"})
		return
	}

	orderCode := time.Now().UnixMilli()%900000000000 + 100000000000
	frontendURL := strings.TrimRight(getEnvOrDefault("FRONTEND_URL", "http://localhost:3000"), "/")
	returnURL := fmt.Sprintf("%s/topup/success?orderCode=%d", frontendURL, orderCode)
	cancelURL := fmt.Sprintf("%s/topup?cancelled=1&orderCode=%d", frontendURL, orderCode)
	description := fmt.Sprintf("GU%d", orderCode%1000000000)

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	if _, err = tx.Exec(`INSERT INTO wallets (user_id, primeval_stone, immortal_essence_stone) SELECT id, COALESCE(primeval_stones,0)::BIGINT, FLOOR(COALESCE(immortal_stones,0))::BIGINT FROM players WHERE id=$1 ON CONFLICT (user_id) DO NOTHING`, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể khởi tạo ví"})
		return
	}
	if _, err = tx.Exec(`INSERT INTO topup_orders (user_id, order_code, amount_vnd, jade_amount, status, provider) VALUES ($1,$2,$3,$4,'PENDING','PAYOS')`, playerID, orderCode, pkg.AmountVND, pkg.Jade); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo đơn nạp"})
		return
	}
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu đơn nạp"})
		return
	}

	checkoutURL, qrCode, payErr := createPayOSPaymentLink(orderCode, pkg, description, returnURL, cancelURL)
	if payErr != nil {
		_, _ = db.DB.Exec(`UPDATE topup_orders SET status='FAILED' WHERE order_code=$1 AND status='PENDING'`, orderCode)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Không thể tạo link thanh toán payOS", "details": payErr.Error()})
		return
	}
	_, err = db.DB.Exec(`UPDATE topup_orders SET checkout_url=$1, qr_code=$2 WHERE order_code=$3`, checkoutURL, qrCode, orderCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu link thanh toán"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"orderCode": orderCode, "amountVnd": pkg.AmountVND, "jadeAmount": pkg.Jade, "checkoutUrl": checkoutURL, "qrCode": qrCode, "status": "PENDING"})
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
	err = db.DB.QueryRow(`SELECT order_code, amount_vnd, jade_amount, status, checkout_url, qr_code, created_at, paid_at FROM topup_orders WHERE order_code=$1 AND user_id=$2`, orderCode, playerID).Scan(&row.OrderCode, &row.AmountVND, &row.Jade, &row.Status, &row.Checkout, &row.QR, &row.CreatedAt, &row.PaidAt)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func GetPremiumShopPackages(c *gin.Context) {
	var playerID uuid.UUID
	_, authed := middleware.GetPlayerID(c)
	if authed {
		playerID, _ = middleware.GetPlayerID(c)
	}
	result := make([]ShopPackageDTO, 0, len(premiumShopPackages))
	for _, pkg := range premiumShopPackages {
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

	if player.MaterialsInventory == nil {
		player.MaterialsInventory = make(models.MaterialsInventory)
	}
	for _, reward := range pkg.Rewards {
		switch reward.Currency {
		case CurrencyPrimevalStone:
			before := int64(player.PrimevalStones)
			player.PrimevalStones += int(reward.Amount)
			wallet.PrimevalStone = int64(player.PrimevalStones)
			if err = insertWalletTx(tx, playerID, "PACKAGE_REWARD", CurrencyPrimevalStone, reward.Amount, before, int64(player.PrimevalStones), nil, &pkg.ID, "Thưởng gói tu luyện"); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể ghi giao dịch thưởng Nguyên Thạch"})
				return
			}
		case CurrencyImmortalEssenceStone:
			before := int64(player.ImmortalStones)
			player.ImmortalStones += float64(reward.Amount)
			wallet.ImmortalEssenceStone = int64(player.ImmortalStones)
			if err = insertWalletTx(tx, playerID, "PACKAGE_REWARD", CurrencyImmortalEssenceStone, reward.Amount, before, int64(player.ImmortalStones), nil, &pkg.ID, "Thưởng gói tu luyện"); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể ghi giao dịch thưởng Tiên Nguyên Thạch"})
				return
			}
		default:
			if reward.ItemID != "" {
				if strings.HasPrefix(reward.ItemID, "GU_") {
					if spec, exists := models.GuRegistry[reward.ItemID]; exists {
						for i := int64(0); i < reward.Quantity; i++ {
							player.GuInventory = append(player.GuInventory, models.GuInstance{InstanceID: uuid.New().String(), GuID: spec.ID, Hunger: 100, Status: "ACTIVE", Equipped: false})
						}
					}
				} else {
					player.MaterialsInventory[reward.ItemID] += int(reward.Quantity)
				}
			}
		}
	}

	if _, err = tx.Exec(`UPDATE players SET primeval_stones=$1, immortal_stones=$2, gu_inventory=$3, materials_inventory=$4 WHERE id=$5`, player.PrimevalStones, player.ImmortalStones, player.GuInventory, player.MaterialsInventory, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật phần thưởng"})
		return
	}
	if _, err = tx.Exec(`UPDATE wallets SET primeval_stone=$1, immortal_essence_stone=$2, immortal_jade=$3, updated_at=CURRENT_TIMESTAMP WHERE user_id=$4`, wallet.PrimevalStone, wallet.ImmortalEssenceStone, wallet.ImmortalJade, playerID); err != nil {
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
	c.JSON(http.StatusOK, gin.H{"message": "Mua gói thành công", "wallet": wallet, "rewards": pkg.Rewards})
}

func ensureWallet(database *sql.DB, playerID uuid.UUID) (WalletDTO, error) {
	if _, err := database.Exec(`INSERT INTO wallets (user_id, primeval_stone, immortal_essence_stone) SELECT id, COALESCE(primeval_stones,0)::BIGINT, FLOOR(COALESCE(immortal_stones,0))::BIGINT FROM players WHERE id=$1 ON CONFLICT (user_id) DO NOTHING`, playerID); err != nil {
		return WalletDTO{}, err
	}
	var w WalletDTO
	err := database.QueryRow(`SELECT primeval_stone, immortal_essence_stone, immortal_jade FROM wallets WHERE user_id=$1`, playerID).Scan(&w.PrimevalStone, &w.ImmortalEssenceStone, &w.ImmortalJade)
	return w, err
}

func ensureWalletTx(tx *sql.Tx, playerID uuid.UUID) (WalletDTO, error) {
	if _, err := tx.Exec(`INSERT INTO wallets (user_id, primeval_stone, immortal_essence_stone) SELECT id, COALESCE(primeval_stones,0)::BIGINT, FLOOR(COALESCE(immortal_stones,0))::BIGINT FROM players WHERE id=$1 ON CONFLICT (user_id) DO NOTHING`, playerID); err != nil {
		return WalletDTO{}, err
	}
	var w WalletDTO
	err := tx.QueryRow(`SELECT primeval_stone, immortal_essence_stone, immortal_jade FROM wallets WHERE user_id=$1 FOR UPDATE`, playerID).Scan(&w.PrimevalStone, &w.ImmortalEssenceStone, &w.ImmortalJade)
	return w, err
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
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("payOS status %d", resp.StatusCode)
	}
	var decoded struct {
		Code string `json:"code"`
		Desc string `json:"desc"`
		Data struct {
			CheckoutURL string `json:"checkoutUrl"`
			QRCode      string `json:"qrCode"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return "", "", err
	}
	if decoded.Code != "00" {
		return "", "", fmt.Errorf("payOS error: %s", decoded.Desc)
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
	rows, err := db.DB.Query(`SELECT order_code, user_id, amount_vnd, jade_amount, status, provider, checkout_url, created_at, paid_at FROM topup_orders ORDER BY created_at DESC LIMIT 100`)
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
	rows, err := db.DB.Query(query, args...)
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
