-- ============================================================
-- Скрипт добавления нового мастера
-- Запускать: psql $DATABASE_URL -f scripts/add_master.sql
-- ============================================================

-- 1. Вставьте данные мастера
INSERT INTO masters (
    name,
    address,
    master_telegram_id,
    client_bot_token,
    admin_bot_token,
    welcome_text,
    is_active,
    slot_interval_min,
    min_hours_before_booking,
    cancel_limit_hours,
    mon_start, mon_end,
    tue_start, tue_end,
    wed_start, wed_end,
    thu_start, thu_end,
    fri_start, fri_end,
    sat_start, sat_end,
    trial_started_at,
    trial_ends_at
) VALUES (
    'Алина',                          -- имя мастера
    'ул. Ленина 15, каб. 203',        -- адрес
    123456789,                         -- telegram ID мастера (узнать через @userinfobot)
    'BOT_TOKEN_CLIENT_HERE',           -- токен клиентского бота (от @BotFather)
    'BOT_TOKEN_ADMIN_HERE',            -- токен админ-бота (от @BotFather)
    'Привет! 👋 Я бот мастера Алины. Записывайтесь быстро и удобно!',
    FALSE,                             -- is_active = FALSE до вашей проверки
    30,                                -- интервал слотов (минуты)
    3,                                 -- минимум часов до записи
    12,                                -- лимит отмены (часы)
    '10:00', '19:00',                 -- Понедельник
    '10:00', '19:00',                 -- Вторник
    '10:00', '19:00',                 -- Среда
    '10:00', '19:00',                 -- Четверг
    '10:00', '19:00',                 -- Пятница
    '10:00', '15:00',                 -- Суббота
    NOW(),                             -- trial_started_at
    NOW() + INTERVAL '14 days'        -- trial_ends_at
);

-- 2. Получаем ID нового мастера
-- SELECT id FROM masters ORDER BY id DESC LIMIT 1;

-- 3. Добавляем услуги (замените master_id на реальный)
-- INSERT INTO services (master_id, name, price, duration_min)
-- VALUES
--   (1, 'Покрытие гель-лак', 5000, 90),
--   (1, 'Снятие + покрытие', 7000, 120),
--   (1, 'Маникюр без покрытия', 3000, 60),
--   (1, 'Педикюр классический', 6000, 90);

-- 4. После проверки активируйте бота:
-- UPDATE masters SET is_active = TRUE WHERE id = 1;
