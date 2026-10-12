package db

// Merchant system: data-driven merchant profiles with a server-wide (shared)
// buy quota and a dynamic buy price that decays as demand is filled. Profiles
// are seeded in code for now; CSV/Admin plumbing is the upgrade path.
//
// ponytail: profiles/stock seeded in code, add MerchantProfile.csv + Admin
// dataset when merchants need live tuning.

func schemaMerchants() error {
	if err := execSchema("merchant_profiles", `
	CREATE TABLE IF NOT EXISTS merchant_profiles (
		id VARCHAR(50) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		kind VARCHAR(20) NOT NULL DEFAULT 'FIXED',
		buy_ratio NUMERIC(10,4) NOT NULL DEFAULT 0,
		buy_floor_ratio NUMERIC(10,4) NOT NULL DEFAULT 0,
		buy_tags JSONB NOT NULL DEFAULT '{}',
		quota_per_item INTEGER NOT NULL DEFAULT 0,
		quota_period VARCHAR(10) NOT NULL DEFAULT 'DAY',
		sell_markup NUMERIC(10,4) NOT NULL DEFAULT 0,
		enabled BOOLEAN NOT NULL DEFAULT TRUE,
		updated_at TIMESTAMP DEFAULT NOW()
	);`); err != nil {
		return err
	}
	if err := execSchema("merchants", `
	CREATE TABLE IF NOT EXISTS merchants (
		id VARCHAR(80) PRIMARY KEY,
		profile_id VARCHAR(50) NOT NULL REFERENCES merchant_profiles(id) ON DELETE CASCADE,
		region_id VARCHAR(50) NOT NULL REFERENCES regions(id) ON DELETE CASCADE,
		x INTEGER NOT NULL DEFAULT 0,
		y INTEGER NOT NULL DEFAULT 0,
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		updated_at TIMESTAMP DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_merchants_region ON merchants (region_id);`); err != nil {
		return err
	}
	// Hàng merchant bán ra. Rỗng = merchant chỉ thu mua.
	if err := execSchema("merchant_stock", `
	CREATE TABLE IF NOT EXISTS merchant_stock (
		merchant_id VARCHAR(80) NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
		item_id VARCHAR(50) NOT NULL,
		rank INTEGER NOT NULL DEFAULT 0,
		price NUMERIC(30,3) NOT NULL,
		stock INTEGER NOT NULL DEFAULT -1,
		sort_order INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (merchant_id, item_id, rank)
	);`); err != nil {
		return err
	}
	// Trạng thái cung-cầu. PK theo period_start nên quota tự reset mỗi kỳ.
	if err := execSchema("merchant_demand", `
	CREATE TABLE IF NOT EXISTS merchant_demand (
		merchant_id VARCHAR(80) NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
		item_id VARCHAR(50) NOT NULL,
		period_start TIMESTAMPTZ NOT NULL,
		bought INTEGER NOT NULL DEFAULT 0,
		sold INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (merchant_id, item_id, period_start)
	);`); err != nil {
		return err
	}
	return nil
}

// SeedMerchantDefaults installs the built-in profiles and places a merchant in
// every HUB/SAFE_ZONE region that already exists. Regions imported later get
// their merchant lazily on first visit (see handlers.ensureRegionMerchants).
func SeedMerchantDefaults() error {
	profiles := []struct {
		id, name, desc, kind, tags, period string
		buyRatio, floor, markup            float64
		quota                              int
	}{
		{
			id: "village_general", name: "Thương Nhân Thôn Làng", kind: "FIXED",
			desc:     "Thu mua hầu hết hàng hoá với giá thấp, số lượng lớn.",
			tags:     `{"types":["Gu","Material","Consumable","Recipe"]}`,
			period:   "DAY",
			buyRatio: 0.10, floor: 0.10, markup: 1.50, quota: 300,
		},
		{
			id: "hub_fence", name: "Tiêu Thương Thành Trấn", kind: "FIXED",
			desc:     "Chỉ thu mua nguyên liệu phẩm chất thấp nhưng trả giá cao; mua nhiều thì giá tụt.",
			tags:     `{"types":["Material","Consumable"],"min_rank":1,"max_rank":5}`,
			period:   "HOUR",
			buyRatio: 0.45, floor: 0.15, markup: 1.40, quota: 15,
		},
	}
	for _, p := range profiles {
		if _, err := schemaDB().Exec(`
			INSERT INTO merchant_profiles (id,name,description,kind,buy_ratio,buy_floor_ratio,buy_tags,quota_per_item,quota_period,sell_markup,updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8,$9,$10,NOW())
			ON CONFLICT (id) DO NOTHING`,
			p.id, p.name, p.desc, p.kind, p.buyRatio, p.floor, p.tags, p.quota, p.period, p.markup); err != nil {
			return err
		}
	}
	// Đặt merchant cố định: village ở HUB + SAFE_ZONE, fence chỉ ở HUB.
	if _, err := schemaDB().Exec(`
		INSERT INTO merchants (id, profile_id, region_id, is_active)
		SELECT 'village_'||r.id, 'village_general', r.id, TRUE
		FROM regions r WHERE r.zone_type IN ('HUB','SAFE_ZONE')
		ON CONFLICT (id) DO NOTHING`); err != nil {
		return err
	}
	if _, err := schemaDB().Exec(`
		INSERT INTO merchants (id, profile_id, region_id, is_active)
		SELECT 'fence_'||r.id, 'hub_fence', r.id, TRUE
		FROM regions r WHERE r.zone_type = 'HUB'
		ON CONFLICT (id) DO NOTHING`); err != nil {
		return err
	}
	// Demo stock cho fence để tab "mua" có dữ liệu ngay.
	if _, err := schemaDB().Exec(`
		INSERT INTO merchant_stock (merchant_id, item_id, rank, price, stock, sort_order)
		SELECT m.id, i.id, COALESCE(i.rank,1),
		       GREATEST(0.001, COALESCE(NULLIF(i.base_price,0),1) * 1.5), 20, 0
		FROM merchants m
		JOIN item_dictionary i ON i.id IN ('ITEM_HEALING_PELLET_SMALL','bitter_water','ginseng','boar_meat')
		WHERE m.profile_id = 'hub_fence'
		ON CONFLICT (merchant_id, item_id, rank) DO NOTHING`); err != nil {
		return err
	}
	return nil
}
