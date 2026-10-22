package parser_test

import (
	"strings"
	"testing"

	pgparser "github.com/mia0x75/pgparser"
)

// Ported from parser_ir_ddl_test.go: supported DDL shapes.
var ddlSQLs = []string{
	`ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);`,
	`ALTER TABLE ONLY public.store_settings ADD CONSTRAINT check_name CHECK (name <> '');`,
	`ALTER TABLE ONLY schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);`,
	`ALTER TABLE orders ADD CONSTRAINT valid_qty CHECK (quantity > 0 AND quantity < 10000);`,
	`ALTER TABLE public.products ADD CONSTRAINT positive_price CHECK (price > 0);`,
	`ALTER TABLE public.users
    ADD CONSTRAINT users_org_fk FOREIGN KEY (org_id) REFERENCES public.organizations(id);`,
	`ALTER TABLE public.users ADD COLUMN status text`,
	`ALTER TABLE public.users ADD CONSTRAINT users_email_uniq UNIQUE (email);`,
	`ALTER TABLE users ADD COLUMN status text`,
	`ALTER TABLE users ADD COLUMN status text, DROP COLUMN legacy`,
	`ALTER TABLE users DROP COLUMN IF EXISTS email`,
	`ALTER TABLE users DROP COLUMN email`,
	`ALTER TABLE users DROP COLUMN email CASCADE`,
	`COMMENT ON COLUMN public."my.table"."my.col" IS 'x';`,
	`COMMENT ON COLUMN public.my.table.my.col IS 'x';`,
	`COMMENT ON COLUMN public.users.email IS 'User email address, must be unique';`,
	`COMMENT ON COLUMN users.email IS 'x';`,
	`COMMENT ON FOREIGN TABLE public.remote_users IS 'Foreign mirror';`,
	`COMMENT ON FUNCTION public.my_func(integer) IS 'Does something';`,
	`COMMENT ON INDEX public.idx_bookings_dates IS 'Composite index for efficient date range queries on bookings';`,
	`COMMENT ON SCHEMA public IS 'Application schema';`,
	`COMMENT ON TABLE public.users IS $tag$Stores "quoted" data$tag$;`,
	`COMMENT ON TABLE public.users IS 'Stores user account information';`,
	`COMMENT ON TABLE public.users IS E'line1\nline2';`,
	`COMMENT ON TABLE public.users IS NULL;`,
	`COMMENT ON TABLE users IS '';`,
	`COMMENT ON TABLE users IS 'it''s a test';`,
	`COMMENT ON TYPE public.email_address IS 'Type used for email addresses';`,
	`CREATE INDEX CONCURRENTLY idx_email ON users (email)`,
	`CREATE INDEX IF NOT EXISTS idx_email ON users (email)`,
	`CREATE INDEX IF NOT EXISTS public.idx_users_email ON public.users (email)`,
	`CREATE INDEX analytics.idx_users_email ON public.users (email)`,
	`CREATE INDEX analytics.idx_users_email ON users (email)`,
	`CREATE INDEX idx_compound ON users (last_name, first_name)`,
	`CREATE INDEX idx_email ON public.users (email)`,
	`CREATE INDEX idx_email ON users (email)`,
	`CREATE INDEX idx_name ON users USING btree (name)`,
	`CREATE INDEX idx_tags ON posts USING gin (tags)`,
	`CREATE INDEX idx_users_active_email
ON users (email)
INCLUDE (last_login_at)
WHERE active = true`,
	`CREATE INDEX idx_users_email ON ONLY public.users (email)`,
	`CREATE INDEX idx_users_email ON users (email)`,
	`CREATE INDEX idx_users_email ON users (email) INCLUDE (name, last_login_at)`,
	`CREATE TABLE IF NOT EXISTS users (id integer)`,
	`CREATE TABLE accounts (
    id integer,
    tenant_id integer,
    payload text,
    PRIMARY KEY (id, tenant_id)
);`,
	`CREATE TABLE products (
				id serial PRIMARY KEY,
				price numeric CHECK (price > 0)
			);`,
	`CREATE TABLE products (
				id serial PRIMARY KEY,
				price numeric CONSTRAINT positive_price CHECK (price > 0)
			);`,
	`CREATE TABLE products (
				id serial PRIMARY KEY,
				price numeric,
				CONSTRAINT positive_price CHECK (price > 0)
			);`,
	`CREATE TABLE products (
				id serial PRIMARY KEY,
				price numeric,
				quantity integer,
				CONSTRAINT positive_price CHECK (price > 0),
				CONSTRAINT valid_qty CHECK (quantity >= 0)
			);`,
	`CREATE TABLE public.accounts (
    id integer,
    tenant_id integer,
    payload text,
    CONSTRAINT accounts_pk PRIMARY KEY (id, tenant_id)
);`,
	`CREATE TABLE public.events (
				start_date date NOT NULL,
				end_date date NOT NULL,
				CONSTRAINT valid_dates CHECK ((end_date > start_date))
			);`,
	`CREATE TABLE public.memberships (
    id integer PRIMARY KEY,
    org_id integer CONSTRAINT memberships_org_fk REFERENCES public.organizations(id),
    branch_id integer REFERENCES branches(id)
);`,
	`CREATE TABLE public.orders (
    id integer PRIMARY KEY,
    user_id integer REFERENCES public.users(id) ON DELETE CASCADE ON UPDATE SET NULL,
    product_id integer,
    CONSTRAINT orders_product_fk FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE SET DEFAULT ON UPDATE RESTRICT
);`,
	`CREATE TABLE public.store_settings (
				id boolean DEFAULT true NOT NULL,
				active_languages text[] NOT NULL,
				default_language text NOT NULL,
				max_players_per_team integer NOT NULL,
				timezone text NOT NULL,
				CONSTRAINT store_settings_check CHECK ((default_language = ANY (active_languages))),
				CONSTRAINT store_settings_id_check CHECK ((id = true))
			);`,
	`CREATE TABLE public.store_settings (
    id boolean DEFAULT true NOT NULL,
    active_languages text[] NOT NULL,
    default_language text NOT NULL,
    max_players_per_team integer NOT NULL,
    timezone text NOT NULL,
    CONSTRAINT store_settings_check CHECK ((default_language = ANY (active_languages))),
    CONSTRAINT store_settings_id_check CHECK ((id = true))
);`,
	`CREATE TABLE public.type_matrix (
    c_smallint smallint,
    c_integer integer,
    c_bigint bigint,
    c_numeric numeric(10,2),
    c_real real,
    c_double double precision,
    c_money money,
    c_bool boolean,
    c_char char(3),
    c_varchar varchar(50),
    c_text text,
    c_bytea bytea,
    c_date date,
    c_time time without time zone,
    c_timetz time with time zone,
    c_timestamp timestamp without time zone,
    c_timestamptz timestamp with time zone,
    c_interval interval year to month,
    c_uuid uuid,
    c_json json,
    c_jsonb jsonb,
    c_xml xml,
    c_inet inet,
    c_cidr cidr,
    c_macaddr macaddr,
    c_macaddr8 macaddr8,
    c_point point,
    c_line line,
    c_lseg lseg,
    c_box box,
    c_path path,
    c_polygon polygon,
    c_circle circle,
    c_int4range int4range,
    c_numrange numrange,
    c_tstzrange tstzrange,
    c_int_array integer[],
    c_text_array text[]
);`,
	`CREATE TABLE public.users (
				email text NOT NULL,
				CONSTRAINT valid_email CHECK ((email ~~ '%@%'::text) OR (length(email) > 5))
			);`,
	`CREATE TABLE public.users (
    --    starts with spaces
    --no-space-prefix
    -- $tag marker
    name text
);`,
	`CREATE TABLE public.users (
    -- [Attribute("Just an example")]
    -- required, min 5, max 55
    name        text,

    -- single-column FK, inline
    org_id      integer     REFERENCES public.organizations(id)
);`,
	`CREATE TABLE public.users (
    -- case-sensitive display email
    "Email" text,
    -- lowercase fallback
    email text
);`,
	`CREATE TABLE public.users (
    -- should not be extracted by default
    name text
);`,
	`CREATE TABLE public.users (
    -- user id
    id integer,
    -- should not attach to any column
    CONSTRAINT users_pk PRIMARY KEY (id),
    -- user email
    email text
);`,
	`CREATE TABLE public.users (
    /* ignored */
    -- picked up
    name text
);`,
	`CREATE TABLE public.users (
    id integer PRIMARY KEY,
    email text UNIQUE,
    code text,
    region text,
    CONSTRAINT users_code_region_uniq UNIQUE (code, region)
);`,
	`CREATE TABLE public.users (
    id integer,
    org_id integer,
    region text NOT NULL,
    branch_id integer NOT NULL,
    CONSTRAINT users_pk PRIMARY KEY (id),
    CONSTRAINT users_org_fk FOREIGN KEY (org_id) REFERENCES public.organizations(id),
    CONSTRAINT users_branch_fk FOREIGN KEY (region, branch_id) REFERENCES public.branches(region, branch_id)
);`,
	`CREATE TABLE public.users (
    id integer, -- not attached
    -- attached to email
    email text
);`,
	`CREATE TABLE t (
    --    starts with spaces
    --no-space-prefix
    -- $tag marker
    name text
);`,
	`CREATE TABLE t (
    -- payload docs
    payload text DEFAULT $tag$a,b,c$tag$,
    -- tags docs
    tags text[] DEFAULT ARRAY['x','y']
);`,
	`CREATE TABLE t (
    -- quoted
    "Email" text,
    -- unquoted
    email text
);`,
	`CREATE TABLE t (
    -- uppercase spelling
    NAME text
);`,
	`CREATE TABLE t (
    id integer PRIMARY KEY,
    ref_id integer REFERENCES other(id) ON DELETE NO ACTION
);`,
	`CREATE UNIQUE INDEX "analytics"."IdxUsersEmail" ON "public"."users" ("email")`,
	`CREATE UNIQUE INDEX idx_email ON users (email)`,
	`CREATE UNIQUE INDEX public.idx_users_email ON public.users (email)`,
	`DROP INDEX CONCURRENTLY idx_users_email`,
	`DROP INDEX IF EXISTS idx_users_email`,
	`DROP INDEX idx_users_email`,
	`DROP INDEX public.idx_users_email`,
	`DROP TABLE IF EXISTS users`,
	`DROP TABLE myschema.users`,
	`DROP TABLE users`,
	`DROP TABLE users CASCADE`,
	`DROP TABLE users, orders, products`,
	`TRUNCATE ONLY public.users`,
	`TRUNCATE TABLE users`,
	`TRUNCATE TABLE users CASCADE`,
	`TRUNCATE public.users`,
	`TRUNCATE users`,
	`TRUNCATE users, orders`,
}

func TestParseSQLDDLShapes(t *testing.T) {
	for _, sql := range ddlSQLs {
		s := strings.TrimSpace(sql)
		if !strings.HasSuffix(s, ";") {
			s += ";"
		}
		if _, err := pgparser.New().ParseOneStmt(s); err != nil {
			t.Errorf("ParseOneStmt(%q): %v", s, err)
		}
	}
}
