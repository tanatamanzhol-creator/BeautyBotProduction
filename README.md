# Beauty Bot Platform

Telegram-бот для записи клиентов к мастерам красоты.
**Стек:** Go + PostgreSQL + Telegram Bot API

---

## Быстрый старт

### 1. Требования
- Go 1.22+
- PostgreSQL 14+

### 2. Клонируйте и настройте
```bash
git clone <your-repo>
cd beauty-bot
cp .env.example .env
```

Заполните `.env`:
```
DATABASE_URL=postgres://user:password@localhost:5432/beautybot?sslmode=disable
ADMIN_TELEGRAM_ID=ваш_telegram_id
```

### 3. Установите зависимости
```bash
go mod tidy
```

### 4. Создайте базу данных
```bash
createdb beautybot
```

### 5. Запустите
```bash
go run cmd/bot/main.go
```
Миграции применятся автоматически при первом запуске.

---

## Добавление нового мастера

### Шаг 1 — Создайте двух ботов в @BotFather
```
/newbot → alina_nails_bot        (клиентский)
/newbot → alina_nails_admin_bot  (админский)
```
Скопируйте оба токена.

### Шаг 2 — Узнайте Telegram ID мастера
Попросите мастера написать @userinfobot — он пришлёт ID.

### Шаг 3 — Добавьте в базу
Отредактируйте `scripts/add_master.sql` и запустите:
```bash
psql $DATABASE_URL -f scripts/add_master.sql
```

### Шаг 4 — Добавьте услуги
```sql
INSERT INTO services (master_id, name, price, duration_min)
VALUES
  (1, 'Покрытие гель-лак', 5000, 90),
  (1, 'Снятие + покрытие', 7000, 120);
```

### Шаг 5 — Активируйте бота
```sql
UPDATE masters SET is_active = TRUE WHERE id = 1;
```

### Шаг 6 — Перезапустите платформу
```bash
# Ctrl+C, затем снова:
go run cmd/bot/main.go
```

Бот автоматически подхватит нового мастера.

---

## Структура проекта

```
beauty-bot/
├── cmd/bot/main.go              ← точка входа
├── internal/
│   ├── config/                  ← конфиг из .env
│   ├── db/                      ← подключение + миграции
│   ├── models/                  ← структуры данных
│   ├── repository/              ← все DB запросы
│   ├── bot/
│   │   ├── manager.go           ← управление всеми ботами
│   │   ├── notify.go            ← уведомления между ботами
│   │   ├── client_bot/          ← бот для клиентов мастера
│   │   └── admin_bot/           ← бот для мастера
│   └── scheduler/               ← напоминания, автоподтверждение
├── scripts/
│   └── add_master.sql           ← скрипт добавления мастера
└── .env
```

---

## Запуск на сервере (production)

### Через systemd
Создайте `/etc/systemd/system/beautybot.service`:
```ini
[Unit]
Description=Beauty Bot Platform
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/home/ubuntu/beauty-bot
ExecStart=/usr/local/go/bin/go run cmd/bot/main.go
Restart=always
RestartSec=5
EnvironmentFile=/home/ubuntu/beauty-bot/.env

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable beautybot
sudo systemctl start beautybot
sudo systemctl status beautybot
```

### Мониторинг
```bash
# Логи в реальном времени
sudo journalctl -u beautybot -f
```

---

## Тарифы и оплата

Платежи принимаются вручную через Kaspi.
После оплаты обновите поле `paid_until`:
```sql
UPDATE masters
SET paid_until = NOW() + INTERVAL '1 month'
WHERE id = 1;
```

---

## Важные команды PostgreSQL

```bash
# Подключиться к базе
psql $DATABASE_URL

# Посмотреть всех мастеров
SELECT id, name, is_active, trial_ends_at FROM masters;

# Посмотреть записи на сегодня
SELECT b.id, c.name, s.name, b.starts_at, b.status
FROM bookings b
JOIN clients c ON c.id = b.client_id
JOIN services s ON s.id = b.service_id
WHERE b.starts_at::date = CURRENT_DATE
ORDER BY b.starts_at;
```
