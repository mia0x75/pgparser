package parser_test

import (
	"strings"
	"testing"

	pgparser "github.com/mia0x75/pgparser"
)

// Ported from parser_ir_edge_cases_test.go: expression/query shapes.
var edgeCaseSQLs = []string{
	`DELETE FROM line_items li
USING orders o, customers c
WHERE li.order_id = o.id
  AND o.customer_id = c.id
  AND c.deleted_at IS NOT NULL`,
	`DELETE FROM orders o
USING customers c
WHERE o.customer_id = c.id AND c.status = 'inactive'
RETURNING o.id`,
	`INSERT INTO accounts (id, name)
VALUES ($1, $2)
ON CONFLICT ON CONSTRAINT accounts_pkey DO NOTHING`,
	`INSERT INTO counters DEFAULT VALUES RETURNING id`,
	`INSERT INTO inventory (product_id, warehouse_id, quantity)
VALUES ($1, $2, $3)
ON CONFLICT (product_id, warehouse_id) DO UPDATE
SET quantity = inventory.quantity + EXCLUDED.quantity,
    updated_at = NOW()
RETURNING product_id, warehouse_id, quantity`,
	`INSERT INTO users (id, email, login_count)
VALUES ($1, $2, 1)
ON CONFLICT (id) DO UPDATE
SET login_count = users.login_count + 1,
    last_login = NOW()
WHERE users.active = true`,
	`INSERT INTO users (name, email)
VALUES ('Alice', 'alice@example.com'),
       ('Bob', 'bob@example.com'),
       ('Charlie', 'charlie@example.com')
RETURNING id`,
	`SELECT
  CASE
    WHEN status = 'active' THEN
      CASE
        WHEN priority > 5 THEN 'high'
        ELSE 'normal'
      END
    WHEN status = 'inactive' THEN 'disabled'
    ELSE 'unknown'
  END AS label
FROM tasks`,
	`SELECT
  GREATEST(a, b, c) AS max_val,
  LEAST(a, b, c) AS min_val
FROM measurements`,
	`SELECT
  id,
  RANK() OVER (ORDER BY score DESC) AS rnk,
  DENSE_RANK() OVER (PARTITION BY category ORDER BY score DESC) AS dense_rnk
FROM leaderboard`,
	`SELECT
  id,
  ROW_NUMBER() OVER (ORDER BY id) AS rn,
  LAG(amount) OVER (ORDER BY created_at) AS prev_amount,
  LEAD(amount) OVER (ORDER BY created_at) AS next_amount,
  SUM(amount) OVER (ORDER BY created_at ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS running_total
FROM transactions`,
	`SELECT
  u.id,
  u.name,
  (SELECT COUNT(*) FROM orders o WHERE o.user_id = u.id) AS order_count
FROM users u`,
	`SELECT "user"."order" FROM "user" WHERE "user"."group" = 'admin'`,
	`SELECT *
FROM (
  SELECT id FROM users ORDER BY id LIMIT 10
) sub`,
	`SELECT *
FROM departments d
JOIN employees e USING (department_id)`,
	`SELECT * FROM (SELECT id FROM t1) sub WHERE sub.id > 1`,
	`SELECT * FROM departments NATURAL JOIN employees`,
	`SELECT * FROM events WHERE created_at > NOW() - INTERVAL '30 days'`,
	`SELECT * FROM orders WHERE created_at BETWEEN '2024-01-01' AND '2024-12-31'`,
	`SELECT * FROM orders WHERE shipped_at IS NULL AND cancelled_at IS NOT NULL`,
	`SELECT * FROM t0 WHERE EXISTS (SELECT 1 FROM t1 WHERE t1.x = t0.id)`,
	`SELECT * FROM users WHERE email LIKE '%@example.com'`,
	`SELECT * FROM users WHERE id = $1 AND name = $2 AND age > $3`,
	`SELECT * FROM users WHERE id = $1 AND status = $2`,
	`SELECT * FROM users WHERE name ILIKE $1`,
	`SELECT * FROM users WHERE status IN ('active', 'pending', 'trial')`,
	`SELECT * FROM users u, orders o WHERE u.id = o.user_id`,
	`SELECT ARRAY[1, 2, 3] AS nums FROM generate_series(1,1)`,
	`SELECT AVG(salary)`,
	`SELECT CAST(price AS integer) AS int_price FROM products`,
	`SELECT COALESCE(nickname, first_name, 'Anonymous') AS display_name FROM users`,
	`SELECT COUNT(*)`,
	`SELECT COUNT(*) AS total, MAX(price) AS highest FROM orders`,
	`SELECT DATE_TRUNC('month', created_at) AS m, COUNT(*)
FROM orders
GROUP BY DATE_TRUNC('month', created_at)`,
	`SELECT DATE_TRUNC('month', created_at) AS month, COUNT(*) AS cnt
FROM orders
GROUP BY DATE_TRUNC('month', created_at)`,
	`SELECT DISTINCT ON (user_id) user_id, created_at FROM events ORDER BY user_id, created_at DESC`,
	`SELECT DISTINCT category FROM products`,
	`SELECT EXTRACT(YEAR FROM created_at) AS yr FROM events`,
	`SELECT NULLIF(status, '') AS clean_status FROM orders`,
	`SELECT a.id, b.name
FROM schema_one.table_a a
JOIN schema_two.table_b b ON a.id = b.a_id`,
	`SELECT a.id, b.name
FROM table_a a
JOIN table_b b ON a.col1 = b.col1 AND a.col2 = b.col2 AND a.col3 = b.col3`,
	`SELECT c, count(*) FROM t0 GROUP BY c HAVING count(*) > (SELECT avg(n) FROM t1)`,
	`SELECT category, COUNT(*) AS cnt, AVG(price) AS avg_price
FROM products
GROUP BY category
HAVING COUNT(*) >= 10 AND AVG(price) < 100`,
	`SELECT created_at AT TIME ZONE 'UTC' AS utc_time FROM events`,
	`SELECT department_id, AVG(salary) AS avg_sal
FROM employees
GROUP BY department_id
HAVING AVG(salary) > (SELECT AVG(salary) FROM employees)`,
	`SELECT e.name AS employee, m.name AS manager
FROM employees e
LEFT JOIN employees m ON e.manager_id = m.id`,
	`SELECT first_name || ' ' || last_name AS full_name FROM users`,
	`SELECT id FROM events WHERE created_at::date = '2024-01-01'::date`,
	`SELECT id FROM products p
WHERE NOT EXISTS (
  SELECT 1 FROM order_items oi WHERE oi.product_id = p.id
)`,
	`SELECT id FROM public.users WHERE active = true`,
	`SELECT id FROM scores WHERE score > ALL(SELECT min_score FROM thresholds)`,
	`SELECT id FROM users
WHERE (status = 'active' OR status = 'trial')
  AND (age >= 18 AND age <= 65)
  AND NOT deleted`,
	`SELECT id FROM users
WHERE active = true
  AND deleted IS NOT TRUE
  AND verified IS TRUE`,
	`SELECT id FROM users WHERE id = ANY($1)`,
	`SELECT id FROM users WHERE name = ''`,
	`SELECT id, (SELECT max(v) FROM t1) AS m FROM t0`,
	`SELECT id, name
FROM users
WHERE CASE WHEN role = 'admin' THEN true ELSE false END = true`,
	`SELECT id, name
FROM users
WHERE id IN (
  SELECT user_id
  FROM orders
  WHERE total > (
    SELECT AVG(total) FROM orders
  )
)`,
	`SELECT id, name FROM users ORDER BY name ASC`,
	`SELECT id, name FROM users WHERE age > 18 AND status = 'active'`,
	`SELECT id, name, age
FROM users
ORDER BY age DESC, name ASC NULLS LAST, id`,
	`SELECT id, status
FROM orders
ORDER BY
  CASE status
    WHEN 'urgent' THEN 1
    WHEN 'normal' THEN 2
    ELSE 3
  END ASC`,
	`SELECT id::text, amount::numeric(10,2) AS amt FROM orders`,
	`SELECT o.id, totals.sum_amount
FROM orders o
JOIN (
  SELECT order_id, SUM(amount) AS sum_amount
  FROM line_items
  GROUP BY order_id
) totals ON totals.order_id = o.id
WHERE totals.sum_amount > 500`,
	`SELECT p.id, tag.value
FROM products p
CROSS JOIN LATERAL unnest(p.tags) AS tag(value)`,
	`SELECT region, COUNT(*) FROM metrics GROUP BY region`,
	`SELECT region, category, SUM(sales)
FROM metrics
GROUP BY region, category`,
	`SELECT sub.total_amount, sub.user_id
FROM (
  SELECT user_id, SUM(amount) AS total_amount
  FROM payments
  GROUP BY user_id
  HAVING SUM(amount) > 100
) sub
WHERE sub.total_amount < 10000`,
	`SELECT u.id FROM users u JOIN orders o ON u.id = o.user_id`,
	`SELECT u.id, o.id, p.name
FROM users u
INNER JOIN orders o ON u.id = o.user_id
LEFT JOIN products p ON o.product_id = p.id
CROSS JOIN settings s`,
	`SELECT u.id, recent.order_id
FROM users u
CROSS JOIN LATERAL (
  SELECT o.id AS order_id
  FROM orders o
  WHERE o.user_id = u.id
  ORDER BY o.created_at DESC
  LIMIT 3
) recent`,
	`SELECT u.name, u.email FROM users u`,
	`UPDATE products p
SET price = p.price * (1 - d.rate),
    updated_at = NOW()
FROM discounts d
WHERE p.category_id = d.category_id
  AND d.active = true
RETURNING p.id, p.price`,
	`UPDATE users
SET total_orders = (SELECT COUNT(*) FROM orders WHERE orders.user_id = users.id)
WHERE active = true`,
	`WITH active AS MATERIALIZED (
  SELECT id FROM users WHERE active = true
)
SELECT * FROM active`,
	`WITH base AS (
  SELECT id, amount, category FROM transactions WHERE amount > 0
),
summarized AS (
  SELECT category, SUM(amount) AS total, COUNT(*) AS cnt
  FROM base
  GROUP BY category
),
filtered AS (
  SELECT category, total
  FROM summarized
  WHERE cnt >= 5
)
SELECT * FROM filtered ORDER BY total DESC`,
	`WITH expired AS (
  SELECT id FROM sessions WHERE expires_at < NOW()
)
UPDATE sessions
SET active = false
WHERE id IN (SELECT id FROM expired)`,
	`WITH monthly_sales AS (
  SELECT
    DATE_TRUNC('month', o.created_at) AS month,
    p.category,
    SUM(oi.quantity * oi.unit_price) AS revenue
  FROM orders o
  JOIN order_items oi ON o.id = oi.order_id
  JOIN products p ON oi.product_id = p.id
  WHERE o.status = 'completed'
    AND o.created_at >= '2024-01-01'
  GROUP BY DATE_TRUNC('month', o.created_at), p.category
),
ranked AS (
  SELECT
    month,
    category,
    revenue,
    RANK() OVER (PARTITION BY month ORDER BY revenue DESC) AS rnk
  FROM monthly_sales
)
SELECT
  r.month,
  r.category,
  r.revenue,
  r.rnk,
  COALESCE(r.revenue / NULLIF(SUM(r.revenue) OVER (PARTITION BY r.month), 0), 0) AS pct_of_month
FROM ranked r
WHERE r.rnk <= 5
ORDER BY r.month DESC, r.rnk ASC
LIMIT 50`,
	`WITH old AS (
  SELECT id FROM logs WHERE created_at < NOW() - INTERVAL '90 days'
)
DELETE FROM logs WHERE id IN (SELECT id FROM old)`,
	`WITH source AS (
  SELECT id, name FROM staging_users WHERE verified = true
)
INSERT INTO users (id, name)
SELECT id, name FROM source`,
}

func TestParseSQLEdgeCaseShapes(t *testing.T) {
	for _, sql := range edgeCaseSQLs {
		s := strings.TrimSpace(sql)
		if !strings.HasSuffix(s, ";") {
			s += ";"
		}
		if _, err := pgparser.New().ParseOneStmt(s); err != nil {
			t.Errorf("ParseOneStmt(%q): %v", s, err)
		}
	}
}
