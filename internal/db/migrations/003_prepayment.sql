-- Prepayment settings for masters
ALTER TABLE masters
ADD COLUMN prepayment_enabled BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN prepayment_amount  INTEGER NOT NULL DEFAULT 0,
ADD COLUMN prepayment_details TEXT    NOT NULL DEFAULT '';

-- Prepayment status for bookings
ALTER TABLE bookings
ADD COLUMN prepayment_status TEXT NOT NULL DEFAULT 'not_required';
-- not_required | pending | claimed | confirmed | expired