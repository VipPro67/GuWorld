package db

// Refine session (phiên luyện cổ) per WF-REF-05 + stage config (WF-REF-06).
// Sessions are historical snapshots: no FK to refining_recipes / gu_dictionary
// so Admin CSV re-imports can never corrupt a running session.

func schemaRefine() error {
	if err := execSchema("refine_sessions", `
	CREATE TABLE IF NOT EXISTS refine_sessions (
		id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		player_id           UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,

		recipe_id           VARCHAR(100) NOT NULL,
		target_gu_id        VARCHAR(50)  NOT NULL,
		target_rank         INTEGER      NOT NULL CHECK (target_rank BETWEEN 1 AND 9),
		variant             INTEGER      NOT NULL DEFAULT 0,

		status              VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE'
		                    CHECK (status IN ('ACTIVE', 'RESOLVED', 'ABORTED')),

		selected_gu_ids     TEXT[]       NOT NULL,
		gu_snapshot         JSONB        NOT NULL DEFAULT '[]',
		materials_deducted  JSONB        NOT NULL DEFAULT '{}',
		rate_snapshot       JSONB        NOT NULL,

		plan                JSONB        NOT NULL,
		step_results        JSONB        NOT NULL DEFAULT '{}',

		accuracy            DOUBLE PRECISION NOT NULL DEFAULT 0,
		death_penalty       DOUBLE PRECISION NOT NULL DEFAULT 0,
		death_marked_at     TIMESTAMPTZ,

		result              JSONB,

		started_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
		ends_at             TIMESTAMPTZ  NOT NULL,
		resolved_at         TIMESTAMPTZ,
		acknowledged_at     TIMESTAMPTZ,
		created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
	);`); err != nil {
		return err
	}
	if err := execSchema("refine_sessions_one_active", `
	CREATE UNIQUE INDEX IF NOT EXISTS refine_sessions_one_active
		ON refine_sessions (player_id)
		WHERE status = 'ACTIVE';`); err != nil {
		return err
	}
	if err := execSchema("refine_sessions_due_idx", `
	CREATE INDEX IF NOT EXISTS idx_refine_sessions_due
		ON refine_sessions (ends_at)
		WHERE status = 'ACTIVE';`); err != nil {
		return err
	}
	if err := execSchema("refine_sessions_player_created_idx", `
	CREATE INDEX IF NOT EXISTS idx_refine_sessions_player_created
		ON refine_sessions (player_id, created_at DESC);`); err != nil {
		return err
	}

	if err := execSchema("refine_stage_templates", `

	CREATE TABLE IF NOT EXISTS refine_stage_templates (
		rank                   INTEGER PRIMARY KEY CHECK (rank BETWEEN 1 AND 9),
		duration_seconds       INTEGER NOT NULL CHECK (duration_seconds > 0),
		step_count             INTEGER NOT NULL CHECK (step_count BETWEEN 3 AND 12),
		minigame_count         INTEGER NOT NULL CHECK (minigame_count >= 1),
		window_seconds         INTEGER NOT NULL CHECK (window_seconds > 0),

		accuracy_max_bonus     DOUBLE PRECISION NOT NULL DEFAULT 0
		                       CHECK (accuracy_max_bonus >= 0 AND accuracy_max_bonus <= 1),
		death_penalty          DOUBLE PRECISION NOT NULL DEFAULT 0.25
		                       CHECK (death_penalty >= 0 AND death_penalty <= 1),
		mastery_multiplier_max DOUBLE PRECISION NOT NULL DEFAULT 2.0
		                       CHECK (mastery_multiplier_max >= 1),

		tap_half_width         DOUBLE PRECISION NOT NULL CHECK (tap_half_width > 0 AND tap_half_width <= 0.5),
		tap_period_ms          INTEGER NOT NULL CHECK (tap_period_ms >= 300),
		sequence_length        INTEGER NOT NULL CHECK (sequence_length BETWEEN 2 AND 10),
		sequence_display_ms    INTEGER NOT NULL CHECK (sequence_display_ms BETWEEN 100 AND 2000),
		sequence_input_ms      INTEGER NOT NULL CHECK (sequence_input_ms BETWEEN 200 AND 5000),
		order_item_count       INTEGER NOT NULL CHECK (order_item_count BETWEEN 2 AND 10),
		choice_option_count    INTEGER NOT NULL CHECK (choice_option_count BETWEEN 2 AND 3),

		minigame_weights       JSONB NOT NULL DEFAULT '{"TAP":1,"SEQUENCE":1,"ORDER":1}',

		updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),

		CONSTRAINT refine_stage_minigame_le_steps CHECK (minigame_count <= step_count - 1)
	);`); err != nil {
		return err
	}

	if err := execSchema("refine_decisions", `
	CREATE TABLE IF NOT EXISTS refine_decisions (
		id             INTEGER PRIMARY KEY,
		situation      TEXT NOT NULL,
		option_1       TEXT NOT NULL,
		option_2       TEXT NOT NULL,
		option_3       TEXT NOT NULL DEFAULT '',
		correct_index  INTEGER NOT NULL CHECK (correct_index >= 0 AND correct_index <= 2),
		neutral_index  INTEGER NOT NULL DEFAULT -1 CHECK (neutral_index >= -1 AND neutral_index <= 2),
		explanation    TEXT NOT NULL DEFAULT ''
	);`); err != nil {
		return err
	}

	// Baseline pace/difficulty config (WF-REF-06 §3): seed once, Admin import
	// overrides later; DO NOTHING keeps admin edits across restarts.
	if err := seedRefineStageBaseline(); err != nil {
		return err
	}
	return nil
}
