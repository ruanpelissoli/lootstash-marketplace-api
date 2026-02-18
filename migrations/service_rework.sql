-- Service Feature Rework Migration
-- Decouples services from listings into standalone entities
-- Creates service_runs table for service engagements
-- Makes offers, chats, and transactions polymorphic

BEGIN;

-- ============================================================
-- 1. Create d2.services table
-- ============================================================

CREATE TABLE d2.services (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id   UUID NOT NULL REFERENCES d2.profiles(id),
    service_type  TEXT NOT NULL,
    name          TEXT NOT NULL,
    description   TEXT,
    asking_price  TEXT,
    asking_for    JSONB NOT NULL DEFAULT '[]',
    game          TEXT NOT NULL DEFAULT 'diablo2',
    ladder        BOOLEAN NOT NULL DEFAULT FALSE,
    hardcore      BOOLEAN NOT NULL DEFAULT FALSE,
    is_non_rotw   BOOLEAN NOT NULL DEFAULT FALSE,
    platforms     TEXT[] NOT NULL DEFAULT '{pc}',
    region        TEXT NOT NULL DEFAULT 'americas',
    notes         TEXT,
    status        TEXT NOT NULL DEFAULT 'active',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- One service per type per provider per game
    CONSTRAINT uq_services_provider_type_game UNIQUE (provider_id, service_type, game)
);

CREATE INDEX idx_services_provider_id ON d2.services(provider_id);
CREATE INDEX idx_services_status ON d2.services(status);
CREATE INDEX idx_services_game ON d2.services(game);
CREATE INDEX idx_services_service_type ON d2.services(service_type);

-- ============================================================
-- 2. Create d2.service_runs table
-- ============================================================

CREATE TABLE d2.service_runs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_id    UUID NOT NULL REFERENCES d2.services(id),
    offer_id      UUID NOT NULL REFERENCES d2.offers(id),
    provider_id   UUID NOT NULL REFERENCES d2.profiles(id),
    client_id     UUID NOT NULL REFERENCES d2.profiles(id),
    status        TEXT NOT NULL DEFAULT 'active',
    cancel_reason TEXT,
    cancelled_by  UUID REFERENCES d2.profiles(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at  TIMESTAMPTZ,
    cancelled_at  TIMESTAMPTZ
);

CREATE INDEX idx_service_runs_service_id ON d2.service_runs(service_id);
CREATE INDEX idx_service_runs_offer_id ON d2.service_runs(offer_id);
CREATE INDEX idx_service_runs_provider_id ON d2.service_runs(provider_id);
CREATE INDEX idx_service_runs_client_id ON d2.service_runs(client_id);
CREATE INDEX idx_service_runs_status ON d2.service_runs(status);

-- ============================================================
-- 3. Alter d2.offers: add type, service_id, make listing_id nullable
-- ============================================================

-- Add type column (default 'item' for existing rows)
ALTER TABLE d2.offers ADD COLUMN type TEXT NOT NULL DEFAULT 'item';

-- Add service_id column
ALTER TABLE d2.offers ADD COLUMN service_id UUID REFERENCES d2.services(id);

-- Make listing_id nullable (was NOT NULL before)
ALTER TABLE d2.offers ALTER COLUMN listing_id DROP NOT NULL;

-- Check constraint: item offers must have listing_id, service offers must have service_id
ALTER TABLE d2.offers ADD CONSTRAINT chk_offers_type CHECK (
    (type = 'item' AND listing_id IS NOT NULL AND service_id IS NULL) OR
    (type = 'service' AND service_id IS NOT NULL AND listing_id IS NULL)
);

CREATE INDEX idx_offers_type ON d2.offers(type);
CREATE INDEX idx_offers_service_id ON d2.offers(service_id);

-- ============================================================
-- 4. Alter d2.chats: make trade_id nullable, add service_run_id
-- ============================================================

-- Make trade_id nullable (was NOT NULL before)
ALTER TABLE d2.chats ALTER COLUMN trade_id DROP NOT NULL;

-- Add service_run_id column
ALTER TABLE d2.chats ADD COLUMN service_run_id UUID REFERENCES d2.service_runs(id);

-- Check constraint: must have exactly one of trade_id or service_run_id
ALTER TABLE d2.chats ADD CONSTRAINT chk_chats_context CHECK (
    (trade_id IS NOT NULL AND service_run_id IS NULL) OR
    (trade_id IS NULL AND service_run_id IS NOT NULL)
);

CREATE UNIQUE INDEX idx_chats_service_run_id ON d2.chats(service_run_id) WHERE service_run_id IS NOT NULL;

-- ============================================================
-- 5. Alter d2.transactions: add service_run_id
-- ============================================================

ALTER TABLE d2.transactions ADD COLUMN service_run_id UUID REFERENCES d2.service_runs(id);

CREATE INDEX idx_transactions_service_run_id ON d2.transactions(service_run_id) WHERE service_run_id IS NOT NULL;

-- ============================================================
-- 6. Add new notification types to enum (if using enum)
-- If notification type is a text column, no migration needed.
-- If it's an enum, uncomment and run:
-- ============================================================

-- ALTER TYPE d2.notification_type ADD VALUE IF NOT EXISTS 'service_run_created';
-- ALTER TYPE d2.notification_type ADD VALUE IF NOT EXISTS 'service_run_completed';
-- ALTER TYPE d2.notification_type ADD VALUE IF NOT EXISTS 'service_run_cancelled';

-- ============================================================
-- 7. Clean d2.listings: remove service-type rows, drop service columns
-- ============================================================

-- Delete any existing service-type listings
DELETE FROM d2.listings WHERE listing_type = 'service';

-- Drop the old columns (only if they exist)
ALTER TABLE d2.listings DROP COLUMN IF EXISTS listing_type;
ALTER TABLE d2.listings DROP COLUMN IF EXISTS service_type;
ALTER TABLE d2.listings DROP COLUMN IF EXISTS description;

-- ============================================================
-- 8. Enable RLS on new tables
-- ============================================================

ALTER TABLE d2.services ENABLE ROW LEVEL SECURITY;
ALTER TABLE d2.service_runs ENABLE ROW LEVEL SECURITY;

-- Service policies: anyone can read active, providers can manage own
CREATE POLICY "Anyone can view active services"
    ON d2.services FOR SELECT
    USING (status = 'active');

CREATE POLICY "Providers can manage own services"
    ON d2.services FOR ALL
    USING (auth.uid() = provider_id);

-- Service run policies: only participants can view/manage
CREATE POLICY "Participants can view own service runs"
    ON d2.service_runs FOR SELECT
    USING (auth.uid() = provider_id OR auth.uid() = client_id);

CREATE POLICY "Participants can manage own service runs"
    ON d2.service_runs FOR ALL
    USING (auth.uid() = provider_id OR auth.uid() = client_id);

-- ============================================================
-- 9. updated_at triggers for new tables
-- ============================================================

CREATE TRIGGER set_services_updated_at
    BEFORE UPDATE ON d2.services
    FOR EACH ROW
    EXECUTE FUNCTION d2.set_updated_at();

CREATE TRIGGER set_service_runs_updated_at
    BEFORE UPDATE ON d2.service_runs
    FOR EACH ROW
    EXECUTE FUNCTION d2.set_updated_at();

COMMIT;
