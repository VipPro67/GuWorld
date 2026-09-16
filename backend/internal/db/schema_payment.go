package db

// Premium wallet, top-up and shop/gift-code accounting schema.

func schemaPayment() error {
	if err := execSchema("topup_orders", `
	CREATE TABLE IF NOT EXISTS topup_orders (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		order_code BIGINT NOT NULL UNIQUE,
		amount_vnd BIGINT NOT NULL CHECK (amount_vnd > 0),
		jade_amount BIGINT NOT NULL CHECK (jade_amount > 0),
		status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','PAID','EXPIRED','FAILED','CANCELLED')),
		provider VARCHAR(20) NOT NULL DEFAULT 'PAYOS',
		checkout_url TEXT,
		qr_code TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		paid_at TIMESTAMP WITH TIME ZONE NULL
	);`); err != nil {
		return err
	}
	if err := execSchema("topup_orders_user_idx", `
	CREATE INDEX IF NOT EXISTS idx_topup_orders_user ON topup_orders (user_id);`); err != nil {
		return err
	}
	if err := execSchema("topup_orders_status_idx", `
	CREATE INDEX IF NOT EXISTS idx_topup_orders_status ON topup_orders (status);`); err != nil {
		return err
	}

	if err := execSchema("wallet_transactions", `
	CREATE TABLE IF NOT EXISTS wallet_transactions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		type VARCHAR(30) NOT NULL CHECK (type IN ('TOPUP','SPEND','REFUND','ADMIN_ADJUST','PACKAGE_REWARD','EXCHANGE')),
		currency VARCHAR(40) NOT NULL CHECK (currency IN ('PRIMEVAL_STONE','IMMORTAL_ESSENCE_STONE','IMMORTAL_JADE')),
		amount NUMERIC(30,3) NOT NULL,
		balance_before NUMERIC(30,3) NOT NULL,
		balance_after NUMERIC(30,3) NOT NULL,
		ref_order_code BIGINT NULL REFERENCES topup_orders(order_code) ON DELETE SET NULL,
		ref_package_id VARCHAR(80) NULL,
		note TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		return err
	}
	if err := execSchema("wallet_transactions_decimal_upgrade", `
	ALTER TABLE wallet_transactions ALTER COLUMN amount TYPE NUMERIC(30,3) USING ROUND(amount::NUMERIC, 3);
	ALTER TABLE wallet_transactions ALTER COLUMN balance_before TYPE NUMERIC(30,3) USING ROUND(balance_before::NUMERIC, 3);
	ALTER TABLE wallet_transactions ALTER COLUMN balance_after TYPE NUMERIC(30,3) USING ROUND(balance_after::NUMERIC, 3);
	`); err != nil {
		return err
	}
	if err := execSchema("wallet_transactions_user_idx", `
	CREATE INDEX IF NOT EXISTS idx_wallet_transactions_user ON wallet_transactions (user_id);`); err != nil {
		return err
	}
	// Self-heal the type CHECK on existing databases (CREATE TABLE IF NOT EXISTS
	// does not update the constraint of an already-created table).
	if err := execSchema("wallet_transactions_type_check", `
	ALTER TABLE wallet_transactions
		DROP CONSTRAINT IF EXISTS wallet_transactions_type_check,
		ADD CONSTRAINT wallet_transactions_type_check
		CHECK (type IN ('TOPUP','SPEND','REFUND','ADMIN_ADJUST','PACKAGE_REWARD','EXCHANGE'));`); err != nil {
		return err
	}
	if err := execSchema("wallet_transactions_topup_order_unique", `
	CREATE UNIQUE INDEX IF NOT EXISTS wallet_transactions_topup_order_unique ON wallet_transactions (ref_order_code) WHERE type = 'TOPUP' AND ref_order_code IS NOT NULL;`); err != nil {
		return err
	}

	if err := execSchema("shop_package_purchases", `
	CREATE TABLE IF NOT EXISTS shop_package_purchases (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		package_id VARCHAR(80) NOT NULL,
		price_jade BIGINT NOT NULL,
		rewards_snapshot JSONB NOT NULL DEFAULT '{}',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		return err
	}
	if err := execSchema("shop_package_purchases_user_package_created_idx", `
	CREATE INDEX IF NOT EXISTS shop_package_purchases_user_package_created_idx ON shop_package_purchases (user_id, package_id, created_at);`); err != nil {
		return err
	}

	if err := execSchema("gift_codes", `
	CREATE TABLE IF NOT EXISTS gift_codes (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		code VARCHAR(50) NOT NULL UNIQUE,
		rewards JSONB NOT NULL DEFAULT '[]',
		max_uses INTEGER NOT NULL DEFAULT 1,
		used_count INTEGER NOT NULL DEFAULT 0,
		max_per_player INTEGER NOT NULL DEFAULT 1,
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (NOW() + INTERVAL '30 days'),
		is_active BOOLEAN NOT NULL DEFAULT true,
		created_by UUID REFERENCES players(id) ON DELETE SET NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		return err
	}

	if err := execSchema("gift_code_redemptions", `
	CREATE TABLE IF NOT EXISTS gift_code_redemptions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		gift_code_id UUID NOT NULL REFERENCES gift_codes(id) ON DELETE CASCADE,
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		redeemed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		return err
	}
	if err := execSchema("gift_code_redemptions_gift_code_id_player_id_idx", `
	CREATE INDEX IF NOT EXISTS gift_code_redemptions_gift_code_id_player_id_idx ON gift_code_redemptions (gift_code_id, player_id);`); err != nil {
		return err
	}

	return nil
}
