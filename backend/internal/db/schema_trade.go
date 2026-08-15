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
	return nil
}

func schemaTrade() error {
	if err := execSchema("auction", `
	CREATE TABLE IF NOT EXISTS auction (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		seller_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		item_type VARCHAR(20) NOT NULL,
		item_data JSONB NOT NULL,
		quantity INTEGER NOT NULL,
		currency VARCHAR(20) NOT NULL,
		start_price DOUBLE PRECISION NOT NULL,
		current_price DOUBLE PRECISION NOT NULL,
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
		monster_sub_rank INT NOT NULL DEFAULT 1
	);`); err != nil {
		return err
	}
	if err := execSchema("mine_slots_occupier_idx", `
	CREATE INDEX IF NOT EXISTS idx_mine_slots_occupier ON mine_slots (occupier_id);`); err != nil {
		return err
	}
	if err := execSchema("mine_slots_monster_idx", `
	CREATE INDEX IF NOT EXISTS idx_mine_slots_monster ON mine_slots (monster_id);`); err != nil {
		return err
	}

	return nil
}
