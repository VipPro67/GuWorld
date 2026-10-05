package db

// Trading / social economy schema: auction, trades, parties, mine slots.
// parties is split into schemaParties (created before players so
// players.party_id can FK inline); its leader FK is added by schemaPlayers.
// auction.item_data is polymorphic JSONB (Gu array / Material id / Recipe).

func schemaParties() error {
	if err := execSchema("parties", `
	CREATE TABLE IF NOT EXISTS parties (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		leader_id UUID NOT NULL,
		members JSONB NOT NULL DEFAULT '[]',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		return err
	}
	// Party battle formation set by the leader: ordered [{kind,id}] where kind
	// is player|pet|npc. Empty = legacy auto behavior.
	if err := execSchema("parties_formation", `
	ALTER TABLE parties ADD COLUMN IF NOT EXISTS formation JSONB NOT NULL DEFAULT '[]';`); err != nil {
		return err
	}
	return nil
}

func schemaTrade() error {
	if err := execSchema("auction", `
	CREATE TABLE IF NOT EXISTS auction (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		seller_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		item_type VARCHAR(20) NOT NULL,
		item_data JSONB NOT NULL,
		quantity BIGINT NOT NULL,
		currency VARCHAR(20) NOT NULL,
		start_price NUMERIC(30,3) NOT NULL,
		current_price NUMERIC(30,3) NOT NULL,
		highest_bidder_id UUID REFERENCES players(id) ON DELETE SET NULL,
		end_time TIMESTAMP WITH TIME ZONE NOT NULL,
		status VARCHAR(20) NOT NULL DEFAULT 'active',
		listing_type VARCHAR(20) NOT NULL DEFAULT 'AUCTION',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		duration_hours INTEGER NOT NULL DEFAULT 24,
		target_buyer_id UUID DEFAULT NULL
	);`); err != nil {
		return err
	}
	if err := execSchema("auction_seller_idx", `
	CREATE INDEX IF NOT EXISTS idx_auction_seller ON auction (seller_id);`); err != nil {
		return err
	}
	if err := execSchema("auction_bidder_idx", `
	CREATE INDEX IF NOT EXISTS idx_auction_bidder ON auction (highest_bidder_id);`); err != nil {
		return err
	}
	if err := execSchema("auction_status_idx", `
	CREATE INDEX IF NOT EXISTS idx_auction_status ON auction (status);`); err != nil {
		return err
	}
	if err := execSchema("auction_status_endtime_buyer_idx", `
	CREATE INDEX IF NOT EXISTS idx_auction_status_endtime_buyer ON auction (status, end_time, target_buyer_id);`); err != nil {
		return err
	}

	if err := execSchema("auction_price_type_upgrade", `
	DO $$
	BEGIN
		ALTER TABLE auction ALTER COLUMN start_price TYPE NUMERIC(30,3) USING ROUND(COALESCE(start_price, 0)::NUMERIC, 3);
		ALTER TABLE auction ALTER COLUMN current_price TYPE NUMERIC(30,3) USING ROUND(COALESCE(current_price, 0)::NUMERIC, 3);
		IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='auction' AND column_name='quantity' AND data_type IN ('integer', 'int')) THEN
			ALTER TABLE auction ALTER COLUMN quantity TYPE BIGINT USING COALESCE(quantity, 0)::BIGINT;
		END IF;
	END $$;`); err != nil {
		return err
	}

	if err := execSchema("trades", `
	CREATE TABLE IF NOT EXISTS trades (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		sender_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		receiver_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		sender_offer JSONB NOT NULL DEFAULT '{"primeval_stones":0,"immortal_stones":0,"gus":[],"materials":{}}',
		receiver_offer JSONB NOT NULL DEFAULT '{"primeval_stones":0,"immortal_stones":0,"gus":[],"materials":{}}',
		sender_locked BOOLEAN NOT NULL DEFAULT false,
		receiver_locked BOOLEAN NOT NULL DEFAULT false,
		sender_confirmed BOOLEAN NOT NULL DEFAULT false,
		receiver_confirmed BOOLEAN NOT NULL DEFAULT false,
		status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		return err
	}
	if err := execSchema("trades_sender_idx", `
	CREATE INDEX IF NOT EXISTS idx_trades_sender ON trades (sender_id);`); err != nil {
		return err
	}
	if err := execSchema("trades_receiver_idx", `
	CREATE INDEX IF NOT EXISTS idx_trades_receiver ON trades (receiver_id);`); err != nil {
		return err
	}
	if err := execSchema("trades_status_idx", `
	CREATE INDEX IF NOT EXISTS idx_trades_status ON trades (status);`); err != nil {
		return err
	}

	if err := execSchema("party_invitations", `
	CREATE TABLE IF NOT EXISTS party_invitations (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		party_id UUID NOT NULL REFERENCES parties(id) ON DELETE CASCADE,
		inviter_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		invitee_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (party_id, invitee_id)
	);`); err != nil {
		return err
	}

	if err := execSchema("mine_slots", `
	CREATE TABLE IF NOT EXISTS mine_slots (
		id INTEGER PRIMARY KEY,
		rarity VARCHAR(50) NOT NULL,
		occupier_id UUID REFERENCES players(id) ON DELETE SET NULL,
		monster_id VARCHAR(50) REFERENCES monster_templates(id) ON DELETE SET NULL,
		occupied_at TIMESTAMPTZ NULL,
		last_claimed_at TIMESTAMPTZ NULL,
		abandoned_at TIMESTAMPTZ DEFAULT NOW(),
		monster_rank INT NOT NULL DEFAULT 1,
		monster_sub_rank INT NOT NULL DEFAULT 1,
		mine_type VARCHAR(20) NOT NULL DEFAULT 'mortal'
	);`); err != nil {
		return err
	}
	execSchema("mine_slots_type", `ALTER TABLE mine_slots ADD COLUMN IF NOT EXISTS mine_type VARCHAR(20) NOT NULL DEFAULT 'mortal';`)
	if err := execSchema("mine_slots_occupier_idx", `
	CREATE INDEX IF NOT EXISTS idx_mine_slots_occupier ON mine_slots (occupier_id);`); err != nil {
		return err
	}
	if err := execSchema("mine_slots_monster_idx", `
	CREATE INDEX IF NOT EXISTS idx_mine_slots_monster ON mine_slots (monster_id);`); err != nil {
		return err
	}

	if err := execSchema("buyback_orders", `
	CREATE TABLE IF NOT EXISTS buyback_orders (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		buyer_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		order_type VARCHAR(10) NOT NULL DEFAULT 'BUY',
		item_type VARCHAR(20) NOT NULL,
		item_target JSONB NOT NULL,
		item_data JSONB DEFAULT NULL,
		unit_price NUMERIC(30,3) NOT NULL,
		quantity_requested BIGINT NOT NULL,
		quantity_filled BIGINT NOT NULL DEFAULT 0,
		currency VARCHAR(20) NOT NULL DEFAULT 'primeval',
		upfront_fee NUMERIC(30,3) NOT NULL DEFAULT 0,
		total_locked NUMERIC(30,3) NOT NULL DEFAULT 0,
		status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
		duration_hours INTEGER NOT NULL DEFAULT 72,
		end_time TIMESTAMP WITH TIME ZONE NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		return err
	}
	execSchema("buyback_orders_order_type", `ALTER TABLE buyback_orders ADD COLUMN IF NOT EXISTS order_type VARCHAR(10) NOT NULL DEFAULT 'BUY';`)
	execSchema("buyback_orders_item_data", `ALTER TABLE buyback_orders ADD COLUMN IF NOT EXISTS item_data JSONB DEFAULT NULL;`)
	if err := execSchema("buyback_buyer_idx", `
	CREATE INDEX IF NOT EXISTS idx_buyback_buyer ON buyback_orders (buyer_id);`); err != nil {
		return err
	}
	if err := execSchema("buyback_status_idx", `
	CREATE INDEX IF NOT EXISTS idx_buyback_status ON buyback_orders (status);`); err != nil {
		return err
	}
	if err := execSchema("buyback_status_end_time_idx", `
	CREATE INDEX IF NOT EXISTS idx_buyback_status_end_time ON buyback_orders (status, end_time);`); err != nil {
		return err
	}

	if err := execSchema("trade_offers", `
	CREATE TABLE IF NOT EXISTS trade_offers (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		creator_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		offered_data JSONB NOT NULL DEFAULT '{"primeval_stones":0,"immortal_stones":0,"gus":[],"materials":{}}',
		requested_data JSONB NOT NULL DEFAULT '{"primeval_stones":0,"immortal_stones":0,"gus":[],"materials":{}}',
		status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
		duration_hours INTEGER NOT NULL DEFAULT 168,
		end_time TIMESTAMP WITH TIME ZONE NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		return err
	}
	execSchema("trade_offers_player_id", `ALTER TABLE trade_offers ADD COLUMN IF NOT EXISTS player_id UUID REFERENCES players(id) ON DELETE CASCADE;`)
	if err := execSchema("trade_offers_creator_idx", `
	CREATE INDEX IF NOT EXISTS idx_trade_offers_creator ON trade_offers (creator_id);`); err != nil {
		return err
	}
	if err := execSchema("trade_offers_player_idx", `
	CREATE INDEX IF NOT EXISTS idx_trade_offers_player ON trade_offers (player_id);`); err != nil {
		return err
	}
	if err := execSchema("trade_offers_status_idx", `
	CREATE INDEX IF NOT EXISTS idx_trade_offers_status ON trade_offers (status);`); err != nil {
		return err
	}
	if err := execSchema("trade_offers_status_end_time_idx", `
	CREATE INDEX IF NOT EXISTS idx_trade_offers_status_end_time ON trade_offers (status, end_time);`); err != nil {
		return err
	}

	execSchema("deprecate_market_listings", `UPDATE auction SET status = 'cancelled' WHERE listing_type = 'MARKET' AND status = 'active';`)

	return nil
}
