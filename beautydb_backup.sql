--
-- PostgreSQL database dump
--

\restrict sWSwTKf5QdUe66K0yqskB7LGOl52t0QgqHJA2PMLOJeFHjn1MzqvllLqbFjC97G

-- Dumped from database version 16.13 (Ubuntu 16.13-0ubuntu0.24.04.1)
-- Dumped by pg_dump version 16.13 (Ubuntu 16.13-0ubuntu0.24.04.1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: blocked_slots; Type: TABLE; Schema: public; Owner: beauty
--

CREATE TABLE public.blocked_slots (
    id integer NOT NULL,
    master_id integer NOT NULL,
    starts_at timestamp with time zone NOT NULL,
    ends_at timestamp with time zone NOT NULL,
    reason text,
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.blocked_slots OWNER TO beauty;

--
-- Name: blocked_slots_id_seq; Type: SEQUENCE; Schema: public; Owner: beauty
--

CREATE SEQUENCE public.blocked_slots_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.blocked_slots_id_seq OWNER TO beauty;

--
-- Name: blocked_slots_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: beauty
--

ALTER SEQUENCE public.blocked_slots_id_seq OWNED BY public.blocked_slots.id;


--
-- Name: bookings; Type: TABLE; Schema: public; Owner: beauty
--

CREATE TABLE public.bookings (
    id integer NOT NULL,
    master_id integer NOT NULL,
    client_id integer NOT NULL,
    service_id integer NOT NULL,
    starts_at timestamp with time zone NOT NULL,
    ends_at timestamp with time zone NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    confirmed_by text,
    cancel_reason text,
    reminder_24h_sent boolean DEFAULT false NOT NULL,
    reminder_2h_sent boolean DEFAULT false NOT NULL,
    review_requested boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.bookings OWNER TO beauty;

--
-- Name: bookings_id_seq; Type: SEQUENCE; Schema: public; Owner: beauty
--

CREATE SEQUENCE public.bookings_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.bookings_id_seq OWNER TO beauty;

--
-- Name: bookings_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: beauty
--

ALTER SEQUENCE public.bookings_id_seq OWNED BY public.bookings.id;


--
-- Name: broadcast_logs; Type: TABLE; Schema: public; Owner: beauty
--

CREATE TABLE public.broadcast_logs (
    id integer NOT NULL,
    master_id integer NOT NULL,
    message text NOT NULL,
    segment text NOT NULL,
    sent_count integer DEFAULT 0 NOT NULL,
    fail_count integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.broadcast_logs OWNER TO beauty;

--
-- Name: broadcast_logs_id_seq; Type: SEQUENCE; Schema: public; Owner: beauty
--

CREATE SEQUENCE public.broadcast_logs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.broadcast_logs_id_seq OWNER TO beauty;

--
-- Name: broadcast_logs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: beauty
--

ALTER SEQUENCE public.broadcast_logs_id_seq OWNED BY public.broadcast_logs.id;


--
-- Name: clients; Type: TABLE; Schema: public; Owner: beauty
--

CREATE TABLE public.clients (
    id integer NOT NULL,
    master_id integer NOT NULL,
    telegram_id bigint NOT NULL,
    telegram_username text,
    name text,
    phone text,
    consent_given boolean DEFAULT false NOT NULL,
    consent_given_at timestamp with time zone,
    no_broadcast boolean DEFAULT false NOT NULL,
    is_blocked boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    visit_count integer DEFAULT 0 NOT NULL,
    last_visit_at timestamp with time zone
);


ALTER TABLE public.clients OWNER TO beauty;

--
-- Name: clients_id_seq; Type: SEQUENCE; Schema: public; Owner: beauty
--

CREATE SEQUENCE public.clients_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.clients_id_seq OWNER TO beauty;

--
-- Name: clients_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: beauty
--

ALTER SEQUENCE public.clients_id_seq OWNED BY public.clients.id;


--
-- Name: masters; Type: TABLE; Schema: public; Owner: beauty
--

CREATE TABLE public.masters (
    id integer NOT NULL,
    name text NOT NULL,
    address text,
    client_bot_token text NOT NULL,
    admin_bot_token text NOT NULL,
    client_bot_username text,
    admin_bot_username text,
    welcome_text text,
    is_active boolean DEFAULT false NOT NULL,
    trial_started_at timestamp with time zone,
    trial_ends_at timestamp with time zone,
    paid_until timestamp with time zone,
    slot_interval_min integer DEFAULT 30 NOT NULL,
    min_hours_before_booking integer DEFAULT 3 NOT NULL,
    cancel_limit_hours integer DEFAULT 12 NOT NULL,
    mon_start time without time zone,
    mon_end time without time zone,
    tue_start time without time zone,
    tue_end time without time zone,
    wed_start time without time zone,
    wed_end time without time zone,
    thu_start time without time zone,
    thu_end time without time zone,
    fri_start time without time zone,
    fri_end time without time zone,
    sat_start time without time zone,
    sat_end time without time zone,
    sun_start time without time zone,
    sun_end time without time zone,
    created_at timestamp with time zone DEFAULT now(),
    master_telegram_id bigint,
    longitude double precision,
    latitude double precision,
    poi_id text
);


ALTER TABLE public.masters OWNER TO beauty;

--
-- Name: masters_id_seq; Type: SEQUENCE; Schema: public; Owner: beauty
--

CREATE SEQUENCE public.masters_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.masters_id_seq OWNER TO beauty;

--
-- Name: masters_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: beauty
--

ALTER SEQUENCE public.masters_id_seq OWNED BY public.masters.id;


--
-- Name: reviews; Type: TABLE; Schema: public; Owner: beauty
--

CREATE TABLE public.reviews (
    id integer NOT NULL,
    master_id integer NOT NULL,
    client_id integer NOT NULL,
    booking_id integer,
    text text NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.reviews OWNER TO beauty;

--
-- Name: reviews_id_seq; Type: SEQUENCE; Schema: public; Owner: beauty
--

CREATE SEQUENCE public.reviews_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.reviews_id_seq OWNER TO beauty;

--
-- Name: reviews_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: beauty
--

ALTER SEQUENCE public.reviews_id_seq OWNED BY public.reviews.id;


--
-- Name: service_categories; Type: TABLE; Schema: public; Owner: beauty
--

CREATE TABLE public.service_categories (
    id integer NOT NULL,
    master_id integer NOT NULL,
    name text NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL
);


ALTER TABLE public.service_categories OWNER TO beauty;

--
-- Name: service_categories_id_seq; Type: SEQUENCE; Schema: public; Owner: beauty
--

CREATE SEQUENCE public.service_categories_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.service_categories_id_seq OWNER TO beauty;

--
-- Name: service_categories_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: beauty
--

ALTER SEQUENCE public.service_categories_id_seq OWNED BY public.service_categories.id;


--
-- Name: services; Type: TABLE; Schema: public; Owner: beauty
--

CREATE TABLE public.services (
    id integer NOT NULL,
    master_id integer NOT NULL,
    category_id integer,
    name text NOT NULL,
    price integer NOT NULL,
    price_from boolean DEFAULT false NOT NULL,
    duration_min integer NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL
);


ALTER TABLE public.services OWNER TO beauty;

--
-- Name: services_id_seq; Type: SEQUENCE; Schema: public; Owner: beauty
--

CREATE SEQUENCE public.services_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.services_id_seq OWNER TO beauty;

--
-- Name: services_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: beauty
--

ALTER SEQUENCE public.services_id_seq OWNED BY public.services.id;


--
-- Name: blocked_slots id; Type: DEFAULT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.blocked_slots ALTER COLUMN id SET DEFAULT nextval('public.blocked_slots_id_seq'::regclass);


--
-- Name: bookings id; Type: DEFAULT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.bookings ALTER COLUMN id SET DEFAULT nextval('public.bookings_id_seq'::regclass);


--
-- Name: broadcast_logs id; Type: DEFAULT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.broadcast_logs ALTER COLUMN id SET DEFAULT nextval('public.broadcast_logs_id_seq'::regclass);


--
-- Name: clients id; Type: DEFAULT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.clients ALTER COLUMN id SET DEFAULT nextval('public.clients_id_seq'::regclass);


--
-- Name: masters id; Type: DEFAULT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.masters ALTER COLUMN id SET DEFAULT nextval('public.masters_id_seq'::regclass);


--
-- Name: reviews id; Type: DEFAULT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.reviews ALTER COLUMN id SET DEFAULT nextval('public.reviews_id_seq'::regclass);


--
-- Name: service_categories id; Type: DEFAULT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.service_categories ALTER COLUMN id SET DEFAULT nextval('public.service_categories_id_seq'::regclass);


--
-- Name: services id; Type: DEFAULT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.services ALTER COLUMN id SET DEFAULT nextval('public.services_id_seq'::regclass);


--
-- Data for Name: blocked_slots; Type: TABLE DATA; Schema: public; Owner: beauty
--

COPY public.blocked_slots (id, master_id, starts_at, ends_at, reason, created_at) FROM stdin;
\.


--
-- Data for Name: bookings; Type: TABLE DATA; Schema: public; Owner: beauty
--

COPY public.bookings (id, master_id, client_id, service_id, starts_at, ends_at, status, confirmed_by, cancel_reason, reminder_24h_sent, reminder_2h_sent, review_requested, created_at) FROM stdin;
110	1	1374	4	2026-04-20 13:00:00+05	2026-04-20 14:00:00+05	cancelled_by_master	\N		f	f	f	2026-04-19 11:22:13.608742+05
111	1	1374	4	2026-04-20 13:00:00+05	2026-04-20 14:00:00+05	cancelled_by_client	master	Перенос	f	f	f	2026-04-19 11:22:25.280404+05
112	1	1374	4	2026-04-24 13:00:00+05	2026-04-24 14:00:00+05	cancelled_by_client	master	Не смогу прийти	f	f	f	2026-04-19 11:25:23.316773+05
113	1	1374	4	2026-04-28 14:30:00+05	2026-04-28 15:30:00+05	confirmed	master	\N	f	f	f	2026-04-20 13:58:34.627347+05
115	1	1374	3	2026-04-29 16:00:00+05	2026-04-29 17:12:00+05	cancelled_by_master	\N		f	f	f	2026-04-20 15:10:35.430288+05
148	1	1374	2	2026-04-25 11:30:00+05	2026-04-25 13:30:00+05	confirmed	master	\N	f	f	f	2026-04-20 19:52:49.058853+05
149	1	1374	1	2026-04-21 10:00:00+05	2026-04-21 11:00:00+05	cancelled_by_master	\N		f	f	f	2026-04-20 19:53:24.904854+05
150	1	1374	2	2026-04-21 10:00:00+05	2026-04-21 12:00:00+05	cancelled_by_master	\N		f	f	f	2026-04-20 19:53:57.148982+05
114	1	1374	1	2026-04-20 16:30:00+05	2026-04-20 17:30:00+05	confirmed	master	\N	f	f	t	2026-04-20 15:01:14.78809+05
152	1	1374	4	2026-04-27 13:30:00+05	2026-04-27 14:30:00+05	confirmed	master	\N	f	f	f	2026-04-20 21:04:02.940279+05
153	1	1374	3	2026-04-21 10:00:00+05	2026-04-21 11:12:00+05	confirmed	master	\N	f	f	f	2026-04-20 21:04:20.66563+05
151	1	1374	1	2026-04-21 13:00:00+05	2026-04-21 14:00:00+05	confirmed	master	\N	f	f	t	2026-04-20 20:02:42.318344+05
154	1	1374	6	2026-04-30 14:30:00+05	2026-04-30 16:30:00+05	confirmed	master	\N	f	f	f	2026-04-21 16:51:40.209501+05
155	1	1374	5	2026-04-29 13:30:00+05	2026-04-29 15:30:00+05	confirmed	master	\N	f	f	f	2026-04-21 16:59:32.267674+05
156	1	1374	4	2026-04-22 13:00:00+05	2026-04-22 14:00:00+05	confirmed	master	\N	f	f	f	2026-04-21 17:00:05.791273+05
157	1	1374	6	2026-04-24 11:30:00+05	2026-04-24 13:30:00+05	confirmed	master	\N	f	f	f	2026-04-21 17:03:55.671985+05
159	1	1374	13	2026-05-04 16:00:00+05	2026-05-04 17:00:00+05	confirmed	master	\N	f	f	f	2026-04-21 17:08:50.273281+05
160	1	1374	6	2026-05-01 15:00:00+05	2026-05-01 17:00:00+05	cancelled_by_master	\N		f	f	f	2026-04-21 17:09:34.860212+05
161	1	1374	5	2026-04-22 10:00:00+05	2026-04-22 12:00:00+05	confirmed	master	\N	f	f	f	2026-04-21 17:09:55.12538+05
158	1	1374	13	2026-05-04 17:30:00+05	2026-05-04 18:30:00+05	cancelled_by_client	master	Перенос	f	f	f	2026-04-21 17:07:05.432899+05
162	1	1374	13	2026-04-30 12:00:00+05	2026-04-30 13:00:00+05	confirmed	master	\N	f	f	f	2026-04-21 17:10:12.239177+05
194	1	1374	4	2026-04-25 10:30:00+05	2026-04-25 11:30:00+05	confirmed	master	\N	f	f	f	2026-04-21 17:59:59.849743+05
195	1	1374	2	2026-04-29 16:00:00+05	2026-04-29 18:00:00+05	confirmed	master	\N	f	f	f	2026-04-21 18:00:14.298936+05
196	1	1652	2	2026-05-01 10:00:00+05	2026-05-01 12:00:00+05	cancelled_by_master	\N		f	f	f	2026-04-21 18:01:48.191472+05
197	1	1652	4	2026-04-25 13:30:00+05	2026-04-25 14:30:00+05	cancelled_by_client	master	Не смогу прийти	f	f	f	2026-04-21 18:02:09.093609+05
198	1	1652	6	2026-05-04 14:00:00+05	2026-05-04 16:00:00+05	cancelled_by_client	master	Перенос	f	f	f	2026-04-21 18:02:40.722484+05
199	1	1652	6	2026-04-30 10:00:00+05	2026-04-30 12:00:00+05	confirmed	master	\N	f	f	f	2026-04-21 18:02:56.192564+05
\.


--
-- Data for Name: broadcast_logs; Type: TABLE DATA; Schema: public; Owner: beauty
--

COPY public.broadcast_logs (id, master_id, message, segment, sent_count, fail_count, created_at) FROM stdin;
\.


--
-- Data for Name: clients; Type: TABLE DATA; Schema: public; Owner: beauty
--

COPY public.clients (id, master_id, telegram_id, telegram_username, name, phone, consent_given, consent_given_at, no_broadcast, is_blocked, created_at, visit_count, last_visit_at) FROM stdin;
1374	1	7946285475	amantanat	Аманжол тест	+77051919255	t	2026-04-19 11:17:50.70828+05	f	f	2026-04-19 11:17:08.046865+05	3	2026-04-21 14:00:00+05
1652	1	949349625	dianatanat	Диана	87953412108	t	2026-04-21 18:01:15.377212+05	f	f	2026-04-21 18:01:09.071343+05	0	\N
\.


--
-- Data for Name: masters; Type: TABLE DATA; Schema: public; Owner: beauty
--

COPY public.masters (id, name, address, client_bot_token, admin_bot_token, client_bot_username, admin_bot_username, welcome_text, is_active, trial_started_at, trial_ends_at, paid_until, slot_interval_min, min_hours_before_booking, cancel_limit_hours, mon_start, mon_end, tue_start, tue_end, wed_start, wed_end, thu_start, thu_end, fri_start, fri_end, sat_start, sat_end, sun_start, sun_end, created_at, master_telegram_id, longitude, latitude, poi_id) FROM stdin;
1	Аманжол	ул. тест, 1	8218210782:AAHKdacvL18xtAUB50WtG8wzyNFAj8MjfY8	8662879357:AAGQcxntZuZcbKin_T7iwEbg5lWnpZlrI-0	\N	@amanzholtanat	\N	t	2026-04-07 15:24:00.83284+05	2026-04-21 15:24:00.83284+05	\N	30	1	12	10:00:00	19:00:00	10:00:00	19:00:00	10:00:00	19:00:00	10:00:00	19:00:00	10:00:00	19:00:00	10:00:00	15:00:00	\N	\N	2026-04-07 15:24:00.83284+05	568615354	76.947448	52.261124	15622496862613037
\.


--
-- Data for Name: reviews; Type: TABLE DATA; Schema: public; Owner: beauty
--

COPY public.reviews (id, master_id, client_id, booking_id, text, created_at) FROM stdin;
44	1	1374	113	Отличный мастер!	2026-04-20 14:10:03.794851+05
45	1	1374	113	Отличный мастер!	2026-04-20 14:32:03.353908+05
46	1	1374	113	Отличный мастер!	2026-04-20 14:32:04.911237+05
47	1	1374	113	Отличный мастер!	2026-04-20 14:32:05.516376+05
48	1	1374	113	Отличный мастер!	2026-04-20 14:32:06.052849+05
49	1	1374	113	Отличный мастер!	2026-04-20 14:32:06.512433+05
50	1	1374	113	Отличный мастер!	2026-04-20 14:32:06.936598+05
51	1	1374	113	Отличный мастер!	2026-04-20 14:32:07.357829+05
52	1	1374	113	Отличный мастер!	2026-04-20 14:32:07.764004+05
53	1	1374	114	Неплохо	2026-04-20 20:00:10.053448+05
\.


--
-- Data for Name: service_categories; Type: TABLE DATA; Schema: public; Owner: beauty
--

COPY public.service_categories (id, master_id, name, sort_order) FROM stdin;
\.


--
-- Data for Name: services; Type: TABLE DATA; Schema: public; Owner: beauty
--

COPY public.services (id, master_id, category_id, name, price, price_from, duration_min, is_active, sort_order) FROM stdin;
2	1	\N	Снятие + покрытие	7000	f	120	t	0
4	1	\N	Педикюр	10000	f	60	t	0
6	1	\N	Массаж новый	6000	f	120	t	0
3	1	\N	Маникюр без покрытия	3000	f	72	t	0
1	1	\N	Покрытие гельлак	300	f	60	t	0
13	1	\N	yjdsq	3000	f	60	t	0
5	1	\N	📅 Расписание	20000	f	120	t	0
\.


--
-- Name: blocked_slots_id_seq; Type: SEQUENCE SET; Schema: public; Owner: beauty
--

SELECT pg_catalog.setval('public.blocked_slots_id_seq', 1, false);


--
-- Name: bookings_id_seq; Type: SEQUENCE SET; Schema: public; Owner: beauty
--

SELECT pg_catalog.setval('public.bookings_id_seq', 199, true);


--
-- Name: broadcast_logs_id_seq; Type: SEQUENCE SET; Schema: public; Owner: beauty
--

SELECT pg_catalog.setval('public.broadcast_logs_id_seq', 1, false);


--
-- Name: clients_id_seq; Type: SEQUENCE SET; Schema: public; Owner: beauty
--

SELECT pg_catalog.setval('public.clients_id_seq', 1685, true);


--
-- Name: masters_id_seq; Type: SEQUENCE SET; Schema: public; Owner: beauty
--

SELECT pg_catalog.setval('public.masters_id_seq', 1, true);


--
-- Name: reviews_id_seq; Type: SEQUENCE SET; Schema: public; Owner: beauty
--

SELECT pg_catalog.setval('public.reviews_id_seq', 53, true);


--
-- Name: service_categories_id_seq; Type: SEQUENCE SET; Schema: public; Owner: beauty
--

SELECT pg_catalog.setval('public.service_categories_id_seq', 1, false);


--
-- Name: services_id_seq; Type: SEQUENCE SET; Schema: public; Owner: beauty
--

SELECT pg_catalog.setval('public.services_id_seq', 13, true);


--
-- Name: blocked_slots blocked_slots_pkey; Type: CONSTRAINT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.blocked_slots
    ADD CONSTRAINT blocked_slots_pkey PRIMARY KEY (id);


--
-- Name: bookings bookings_pkey; Type: CONSTRAINT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.bookings
    ADD CONSTRAINT bookings_pkey PRIMARY KEY (id);


--
-- Name: broadcast_logs broadcast_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.broadcast_logs
    ADD CONSTRAINT broadcast_logs_pkey PRIMARY KEY (id);


--
-- Name: clients clients_master_id_telegram_id_key; Type: CONSTRAINT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.clients
    ADD CONSTRAINT clients_master_id_telegram_id_key UNIQUE (master_id, telegram_id);


--
-- Name: clients clients_pkey; Type: CONSTRAINT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.clients
    ADD CONSTRAINT clients_pkey PRIMARY KEY (id);


--
-- Name: masters masters_admin_bot_token_key; Type: CONSTRAINT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.masters
    ADD CONSTRAINT masters_admin_bot_token_key UNIQUE (admin_bot_token);


--
-- Name: masters masters_client_bot_token_key; Type: CONSTRAINT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.masters
    ADD CONSTRAINT masters_client_bot_token_key UNIQUE (client_bot_token);


--
-- Name: masters masters_pkey; Type: CONSTRAINT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.masters
    ADD CONSTRAINT masters_pkey PRIMARY KEY (id);


--
-- Name: reviews reviews_pkey; Type: CONSTRAINT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.reviews
    ADD CONSTRAINT reviews_pkey PRIMARY KEY (id);


--
-- Name: service_categories service_categories_pkey; Type: CONSTRAINT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.service_categories
    ADD CONSTRAINT service_categories_pkey PRIMARY KEY (id);


--
-- Name: services services_pkey; Type: CONSTRAINT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.services
    ADD CONSTRAINT services_pkey PRIMARY KEY (id);


--
-- Name: idx_blocked_slots_master; Type: INDEX; Schema: public; Owner: beauty
--

CREATE INDEX idx_blocked_slots_master ON public.blocked_slots USING btree (master_id, starts_at, ends_at);


--
-- Name: idx_bookings_master_starts; Type: INDEX; Schema: public; Owner: beauty
--

CREATE INDEX idx_bookings_master_starts ON public.bookings USING btree (master_id, starts_at);


--
-- Name: idx_bookings_status; Type: INDEX; Schema: public; Owner: beauty
--

CREATE INDEX idx_bookings_status ON public.bookings USING btree (status);


--
-- Name: idx_clients_master_telegram; Type: INDEX; Schema: public; Owner: beauty
--

CREATE INDEX idx_clients_master_telegram ON public.clients USING btree (master_id, telegram_id);


--
-- Name: blocked_slots blocked_slots_master_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.blocked_slots
    ADD CONSTRAINT blocked_slots_master_id_fkey FOREIGN KEY (master_id) REFERENCES public.masters(id) ON DELETE CASCADE;


--
-- Name: bookings bookings_client_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.bookings
    ADD CONSTRAINT bookings_client_id_fkey FOREIGN KEY (client_id) REFERENCES public.clients(id) ON DELETE CASCADE;


--
-- Name: bookings bookings_master_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.bookings
    ADD CONSTRAINT bookings_master_id_fkey FOREIGN KEY (master_id) REFERENCES public.masters(id) ON DELETE CASCADE;


--
-- Name: bookings bookings_service_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.bookings
    ADD CONSTRAINT bookings_service_id_fkey FOREIGN KEY (service_id) REFERENCES public.services(id) ON DELETE CASCADE;


--
-- Name: broadcast_logs broadcast_logs_master_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.broadcast_logs
    ADD CONSTRAINT broadcast_logs_master_id_fkey FOREIGN KEY (master_id) REFERENCES public.masters(id) ON DELETE CASCADE;


--
-- Name: clients clients_master_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.clients
    ADD CONSTRAINT clients_master_id_fkey FOREIGN KEY (master_id) REFERENCES public.masters(id) ON DELETE CASCADE;


--
-- Name: reviews reviews_booking_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.reviews
    ADD CONSTRAINT reviews_booking_id_fkey FOREIGN KEY (booking_id) REFERENCES public.bookings(id) ON DELETE CASCADE;


--
-- Name: reviews reviews_client_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.reviews
    ADD CONSTRAINT reviews_client_id_fkey FOREIGN KEY (client_id) REFERENCES public.clients(id) ON DELETE CASCADE;


--
-- Name: reviews reviews_master_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.reviews
    ADD CONSTRAINT reviews_master_id_fkey FOREIGN KEY (master_id) REFERENCES public.masters(id) ON DELETE CASCADE;


--
-- Name: service_categories service_categories_master_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.service_categories
    ADD CONSTRAINT service_categories_master_id_fkey FOREIGN KEY (master_id) REFERENCES public.masters(id) ON DELETE CASCADE;


--
-- Name: services services_category_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.services
    ADD CONSTRAINT services_category_id_fkey FOREIGN KEY (category_id) REFERENCES public.service_categories(id) ON DELETE SET NULL;


--
-- Name: services services_master_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: beauty
--

ALTER TABLE ONLY public.services
    ADD CONSTRAINT services_master_id_fkey FOREIGN KEY (master_id) REFERENCES public.masters(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

\unrestrict sWSwTKf5QdUe66K0yqskB7LGOl52t0QgqHJA2PMLOJeFHjn1MzqvllLqbFjC97G

