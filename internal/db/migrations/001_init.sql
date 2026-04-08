-- Masters table
CREATE TABLE IF NOT EXISTS masters (
    id                  SERIAL PRIMARY KEY,
    name                TEXT NOT NULL,
    address             TEXT,
    client_bot_token    TEXT NOT NULL UNIQUE,
    admin_bot_token     TEXT NOT NULL UNIQUE,
    client_bot_username TEXT,
    admin_bot_username  TEXT,
    welcome_text        TEXT,
    is_active           BOOLEAN NOT NULL DEFAULT FALSE,
    trial_started_at    TIMESTAMP WITH TIME ZONE,
    trial_ends_at       TIMESTAMP WITH TIME ZONE,
    paid_until          TIMESTAMP WITH TIME ZONE,
    -- Booking settings
    slot_interval_min       INTEGER NOT NULL DEFAULT 30,
    min_hours_before_booking INTEGER NOT NULL DEFAULT 3,
    cancel_limit_hours      INTEGER NOT NULL DEFAULT 12,
    -- Schedule: working hours per day (null = day off)
    mon_start TIME, mon_end TIME,
    tue_start TIME, tue_end TIME,
    wed_start TIME, wed_end TIME,
    thu_start TIME, thu_end TIME,
    fri_start TIME, fri_end TIME,
    sat_start TIME, sat_end TIME,
    sun_start TIME, sun_end TIME,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Service categories
CREATE TABLE IF NOT EXISTS service_categories (
    id        SERIAL PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES masters(id) ON DELETE CASCADE,
    name      TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0
);

-- Services
CREATE TABLE IF NOT EXISTS services (
    id          SERIAL PRIMARY KEY,
    master_id   INTEGER NOT NULL REFERENCES masters(id) ON DELETE CASCADE,
    category_id INTEGER REFERENCES service_categories(id) ON DELETE SET NULL,
    name        TEXT NOT NULL,
    price       INTEGER NOT NULL, -- in tenge
    price_from  BOOLEAN NOT NULL DEFAULT FALSE, -- "from X tenge"
    duration_min INTEGER NOT NULL, -- duration in minutes
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order  INTEGER NOT NULL DEFAULT 0
);

-- Clients
CREATE TABLE IF NOT EXISTS clients (
    id               SERIAL PRIMARY KEY,
    master_id        INTEGER NOT NULL REFERENCES masters(id) ON DELETE CASCADE,
    telegram_id      BIGINT NOT NULL,
    telegram_username TEXT,
    name             TEXT,
    phone            TEXT,
    consent_given    BOOLEAN NOT NULL DEFAULT FALSE,
    consent_given_at TIMESTAMP WITH TIME ZONE,
    no_broadcast     BOOLEAN NOT NULL DEFAULT FALSE,
    is_blocked       BOOLEAN NOT NULL DEFAULT FALSE, -- blocked the bot
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(master_id, telegram_id)
);

-- Bookings
CREATE TABLE IF NOT EXISTS bookings (
    id           SERIAL PRIMARY KEY,
    master_id    INTEGER NOT NULL REFERENCES masters(id) ON DELETE CASCADE,
    client_id    INTEGER NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    service_id   INTEGER NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    starts_at    TIMESTAMP WITH TIME ZONE NOT NULL,
    ends_at      TIMESTAMP WITH TIME ZONE NOT NULL,
    status       TEXT NOT NULL DEFAULT 'pending',
    -- pending | confirmed | cancelled_by_client | cancelled_by_master | completed
    confirmed_by TEXT, -- 'master' | 'auto'
    cancel_reason TEXT,
    reminder_24h_sent BOOLEAN NOT NULL DEFAULT FALSE,
    reminder_2h_sent  BOOLEAN NOT NULL DEFAULT FALSE,
    review_requested  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Blocked time slots
CREATE TABLE IF NOT EXISTS blocked_slots (
    id         SERIAL PRIMARY KEY,
    master_id  INTEGER NOT NULL REFERENCES masters(id) ON DELETE CASCADE,
    starts_at  TIMESTAMP WITH TIME ZONE NOT NULL,
    ends_at    TIMESTAMP WITH TIME ZONE NOT NULL,
    reason     TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Reviews
CREATE TABLE IF NOT EXISTS reviews (
    id         SERIAL PRIMARY KEY,
    master_id  INTEGER NOT NULL REFERENCES masters(id) ON DELETE CASCADE,
    client_id  INTEGER NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    booking_id INTEGER NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    text       TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Broadcast logs
CREATE TABLE IF NOT EXISTS broadcast_logs (
    id          SERIAL PRIMARY KEY,
    master_id   INTEGER NOT NULL REFERENCES masters(id) ON DELETE CASCADE,
    message     TEXT NOT NULL,
    segment     TEXT NOT NULL, -- 'inactive_1m' | 'inactive_2m' | 'inactive_3m'
    sent_count  INTEGER NOT NULL DEFAULT 0,
    fail_count  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_bookings_master_starts ON bookings(master_id, starts_at);
CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings(status);
CREATE INDEX IF NOT EXISTS idx_clients_master_telegram ON clients(master_id, telegram_id);
CREATE INDEX IF NOT EXISTS idx_blocked_slots_master ON blocked_slots(master_id, starts_at, ends_at);
