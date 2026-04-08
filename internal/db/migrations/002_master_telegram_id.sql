-- Add master's own Telegram ID so we know where to send admin notifications
ALTER TABLE masters ADD COLUMN IF NOT EXISTS master_telegram_id BIGINT;
