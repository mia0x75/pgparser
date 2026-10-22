package parser_test

import (
	"strings"
	"testing"

	pgparser "github.com/mia0x75/pgparser"
	"github.com/mia0x75/pgparser/ast"
)

// Ported from parser_ir_advanced_test.go: advanced SQL constructs.
func TestParseSQLAdvancedConstructsAST(t *testing.T) {
	ins, ok := mustParseOne(t, `INSERT INTO users (id, name, email) VALUES (1, 'John Doe', 'john.doe@example.com') ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, email = EXCLUDED.email;`).(*ast.InsertStmt)
	if !ok || ins.OnConflict == nil {
		t.Fatalf("expected INSERT ... ON CONFLICT, got %#v", ins)
	}
	if len(ins.OnConflict.IndexCols) != 1 || len(ins.OnConflict.UpdateList) != 2 {
		t.Fatalf("unexpected ON CONFLICT: %#v", ins.OnConflict)
	}

	setOp, ok := mustParseOne(t, `SELECT id, name FROM active_users UNION SELECT id, name FROM inactive_users;`).(*ast.SetOprStmt)
	if !ok || setOp.SelectList == nil {
		t.Fatalf("expected UNION set operation, got %#v", setOp)
	}
	if tables := collectTables(setOp); !hasTable(tables, "", "inactive_users") {
		t.Fatalf("tables = %#v, want inactive_users", tables)
	}

	for _, sql := range []string{
		`SELECT product_id FROM online_orders INTERSECT SELECT product_id FROM in_store_purchases;`,
		`SELECT customer_id FROM all_customers EXCEPT SELECT customer_id FROM recent_buyers;`,
		`SELECT name FROM customers c WHERE EXISTS (SELECT 1 FROM orders o WHERE o.customer_id = c.id);`,
	} {
		mustParseOne(t, sql)
	}

	insSelect, ok := mustParseOne(t, `INSERT INTO high_value_orders (order_id, customer_id, total) SELECT id, customer_id, order_total FROM orders WHERE order_total > 1000;`).(*ast.InsertStmt)
	if !ok || insSelect.Select == nil || len(insSelect.Columns) != 3 {
		t.Fatalf("unexpected INSERT ... SELECT: %#v", insSelect)
	}
	if tables := collectTables(insSelect); !hasTable(tables, "", "high_value_orders") || !hasTable(tables, "", "orders") {
		t.Fatalf("tables = %#v", tables)
	}

	for _, sql := range []string{
		`SELECT u.name, p.product_name FROM users u RIGHT JOIN purchases p ON u.id = p.user_id;`,
		`SELECT d.name, e.name FROM departments d FULL OUTER JOIN employees e ON d.id = e.department_id;`,
	} {
		st := mustParseSelect(t, sql)
		if got := len(collectTables(st)); got < 2 {
			t.Fatalf("tables = %d, want >= 2 for %q", got, sql)
		}
	}
}

// Ported from parser_ir_advanced_test.go: function wrapper and cast shapes are
// accepted across WHERE, JOIN, HAVING, windows and DML.
func TestParseSQLFunctionWrapperShapes(t *testing.T) {
	queries := []string{
		"SELECT * FROM t WHERE length(notes) < 100;",
		"SELECT * FROM users u WHERE lower(u.email) = 'a@b.com';",
		"SELECT * FROM t WHERE char_length(name) > 3;",
		"SELECT * FROM t WHERE coalesce(name, '') <> '';",
		"SELECT * FROM t WHERE coalesce(name, NULL) <> 'x';",
		"SELECT * FROM t WHERE coalesce(name, $1) <> '';",
		"SELECT * FROM t WHERE extract(year FROM created_at) = 2025;",
		"SELECT * FROM t WHERE extract(year FROM created_at) > 2024;",
		"SELECT * FROM t WHERE date_trunc('month', created_at) = '2025-03-01';",
		"SELECT * FROM t WHERE lower(lower(name)) = 'foo';",
		"SELECT * FROM t WHERE length(lower(name)) > 5;",
		"SELECT * FROM t WHERE pg_catalog.length(name) > 5;",
		`SELECT * FROM t WHERE length("Customer (legacy)") < 50;`,
		"SELECT * FROM t WHERE length(col)::int::bigint > 10;",
		"SELECT * FROM t WHERE CAST(length(col) AS bigint) > 10;",
		"SELECT * FROM t WHERE length(name) = 5;",
		"SELECT * FROM t WHERE length(name) <> 5;",
		"SELECT * FROM t WHERE length(name) <= 5;",
		"SELECT * FROM t WHERE lower(name) NOT LIKE '%a%';",
		"SELECT * FROM t WHERE length(name) NOT IN (1, 2);",
		"SELECT * FROM t WHERE length(name) BETWEEN 1 AND 5;",
		"SELECT * FROM t WHERE lower(name) IS NULL;",
		"SELECT * FROM t WHERE 100 > length(notes);",
		"SELECT * FROM t WHERE length(col)::int BETWEEN 1 AND 5;",
		"UPDATE t SET status = 'x' WHERE length(name) < 5;",
		"DELETE FROM t WHERE length(name) < 5;",
		"SELECT * FROM users WHERE id IN (SELECT id FROM t WHERE length(name) < 5);",
		"WITH x AS (SELECT id FROM t WHERE length(name) < 5) SELECT * FROM x;",
		"SELECT id FROM a WHERE length(name) < 5 UNION SELECT id FROM b;",
		"SELECT id FROM a UNION ALL SELECT id FROM b WHERE length(name) < 5;",
		"SELECT id FROM a WHERE length(name) < 5 INTERSECT SELECT id FROM b;",
		"SELECT id FROM a EXCEPT SELECT id FROM b WHERE length(name) < 5;",
		"SELECT id FROM t GROUP BY id HAVING length(name) < 5;",
		"SELECT * FROM a JOIN b ON lower(a.x) = lower(b.y) WHERE a.x = 1;",
		"SELECT * FROM a FULL OUTER JOIN b ON lower(a.x) = lower(b.y);",
		"SELECT * FROM a JOIN b USING (id);",
		"SELECT * FROM t ORDER BY lower(name);",
		"SELECT length(name), COUNT(*) FROM t GROUP BY length(name);",
		"SELECT row_number() OVER (PARTITION BY lower(name)) FROM t;",
		"UPDATE t SET status = 'x' WHERE id = 1 RETURNING length(name);",
		"SELECT length(name) FROM t WHERE id = 1;",
		"SELECT * FROM t WHERE length(name || ' suffix') < 50;",
		"SELECT * FROM t WHERE md5(name) = 'abc';",
		"SELECT * FROM t WHERE public.my_length(col) > 5;",
		"SELECT * FROM t WHERE col::int < 5;",
		"SELECT * FROM t WHERE length(md5(col)) > 5;",
	}
	for _, sql := range queries {
		if _, err := pgparser.New().ParseOneStmt(sql); err != nil {
			t.Errorf("ParseOneStmt(%q): %v", sql, err)
		}
	}
}

func TestParseSQLGenericUtilityStatementsAST(t *testing.T) {
	for _, tc := range []struct {
		sql  string
		kind string
	}{
		{`CREATE EXTENSION IF NOT EXISTS hstore;`, "CREATE"},
	} {
		st, ok := mustParseOne(t, tc.sql).(*ast.GenericStmt)
		if !ok {
			t.Fatalf("%q: expected GenericStmt, got %T", tc.sql, mustParseOne(t, tc.sql))
		}
		if st.Kind != tc.kind {
			t.Errorf("%q: kind = %q, want %q", tc.sql, st.Kind, tc.kind)
		}
		if st.Raw == "" {
			t.Errorf("%q: empty raw statement", tc.sql)
		}
	}
}

func TestParseSQLCreateSchemaAST(t *testing.T) {
	schema := mustParseOne(t, `CREATE SCHEMA IF NOT EXISTS app AUTHORIZATION alice;`).(*ast.CreateSchemaStmt)
	if !schema.IfNotExists || schema.Name != "app" || schema.Authorization != "alice" {
		t.Fatalf("CreateSchemaStmt = %#v", schema)
	}
	if got := compactSQL(restoreNode(t, schema)); got != "CREATE SCHEMA IF NOT EXISTS app AUTHORIZATION alice" {
		t.Fatalf("CreateSchemaStmt restore = %q", got)
	}
	authOnly := mustParseOne(t, `CREATE SCHEMA AUTHORIZATION alice;`).(*ast.CreateSchemaStmt)
	if authOnly.Name != "" || authOnly.Authorization != "alice" {
		t.Fatalf("CreateSchemaStmt = %#v", authOnly)
	}
}

func TestParseSQLMaintenanceAndSessionStatementsAST(t *testing.T) {
	for _, tc := range []struct {
		sql   string
		check func(ast.StmtNode) bool
	}{
		{`VACUUM (FULL, ANALYZE) t;`, func(n ast.StmtNode) bool { _, ok := n.(*ast.VacuumStmt); return ok }},
		{`CREATE TEMP VIEW v AS SELECT 1;`, func(n ast.StmtNode) bool { _, ok := n.(*ast.CreateViewStmt); return ok }},
		{`CREATE VIEW v AS SELECT 1 WITH LOCAL CHECK OPTION;`, func(n ast.StmtNode) bool {
			_, ok := n.(*ast.CreateViewStmt)
			return ok
		}},
		{`REINDEX (VERBOSE) TABLE CONCURRENTLY t;`, func(n ast.StmtNode) bool {
			st, ok := n.(*ast.ReindexStmt)
			return ok && st.Concurrently && st.Verbose && st.Name == "t"
		}},
		{`CLUSTER t USING ix;`, func(n ast.StmtNode) bool { _, ok := n.(*ast.ClusterStmt); return ok }},
		{`REFRESH MATERIALIZED VIEW CONCURRENTLY mv WITH NO DATA;`, func(n ast.StmtNode) bool {
			st, ok := n.(*ast.RefreshMatViewStmt)
			return ok && st.Concurrently && st.NoData
		}},
		{`CHECKPOINT;`, func(n ast.StmtNode) bool { _, ok := n.(*ast.CheckpointStmt); return ok }},
		{`DISCARD ALL;`, func(n ast.StmtNode) bool { _, ok := n.(*ast.DiscardStmt); return ok }},
		{`LOAD '/tmp/x';`, func(n ast.StmtNode) bool { _, ok := n.(*ast.LoadStmt); return ok }},
		{`SECURITY LABEL FOR selinux ON TABLE t IS 's';`, func(n ast.StmtNode) bool {
			_, ok := n.(*ast.SecurityLabelStmt)
			return ok
		}},
		{`IMPORT FOREIGN SCHEMA remote FROM SERVER srv INTO public OPTIONS (x);`, func(n ast.StmtNode) bool {
			_, ok := n.(*ast.ImportForeignSchemaStmt)
			return ok
		}},
		{`REASSIGN OWNED BY a, b TO c;`, func(n ast.StmtNode) bool {
			_, ok := n.(*ast.ReassignOwnedStmt)
			return ok
		}},
		{`DROP OWNED BY a CASCADE;`, func(n ast.StmtNode) bool { _, ok := n.(*ast.DropOwnedStmt); return ok }},
		{`CALL refresh_data(1);`, func(n ast.StmtNode) bool { _, ok := n.(*ast.CallStmt); return ok }},
		{`DECLARE c BINARY SCROLL CURSOR WITH HOLD FOR SELECT 1;`, func(n ast.StmtNode) bool {
			_, ok := n.(*ast.DeclareCursorStmt)
			return ok
		}},
		{`FETCH NEXT FROM c;`, func(n ast.StmtNode) bool { _, ok := n.(*ast.FetchCursorStmt); return ok }},
		{`MOVE ALL IN c;`, func(n ast.StmtNode) bool {
			st, ok := n.(*ast.FetchCursorStmt)
			return ok && st.IsMove
		}},
		{`CLOSE c;`, func(n ast.StmtNode) bool { _, ok := n.(*ast.CloseCursorStmt); return ok }},
		{`LISTEN chan;`, func(n ast.StmtNode) bool { _, ok := n.(*ast.ListenStmt); return ok }},
		{`NOTIFY chan, 'payload';`, func(n ast.StmtNode) bool { _, ok := n.(*ast.NotifyStmt); return ok }},
		{`UNLISTEN *;`, func(n ast.StmtNode) bool { _, ok := n.(*ast.UnlistenStmt); return ok }},
		{`COPY t (a, b) FROM STDIN WITH (FORMAT csv);`, func(n ast.StmtNode) bool {
			_, ok := n.(*ast.CopyStmt)
			return ok
		}},
	} {
		st := mustParseOne(t, tc.sql)
		if !tc.check(st) {
			t.Errorf("%q: unexpected AST %T", tc.sql, st)
		}
	}
}

func TestParseSQLModeledPostgresClausesAST(t *testing.T) {
	sel := mustParseOne(t, `SELECT a COLLATE "C", count(*) FILTER (WHERE a > 0), percentile_cont(0.5) WITHIN GROUP (ORDER BY a), sum(a) OVER w FROM t TABLESAMPLE system (10) REPEATABLE (5) WINDOW w AS (PARTITION BY b ORDER BY c) FOR NO KEY UPDATE OF t SKIP LOCKED;`).(*ast.SelectStmt)
	if len(sel.WindowSpecs) != 1 || sel.WindowSpecs[0].Name.L != "w" {
		t.Fatalf("WindowSpecs = %#v", sel.WindowSpecs)
	}
	if sel.LockTp != ast.SelectLockNoKeyUpdate || sel.LockWaitPolicy != ast.SelectLockWaitSkipLocked || len(sel.LockedTables) != 1 {
		t.Fatalf("unexpected lock info: %#v %#v %#v", sel.LockTp, sel.LockWaitPolicy, sel.LockedTables)
	}
	from := sel.From.TableRefs.Left.(*ast.TableSource)
	if from.TableSample == nil || from.TableSample.Method != "system" || len(from.TableSample.Args) != 1 || from.TableSample.Repeatable == nil {
		t.Fatalf("TableSample = %#v", from.TableSample)
	}
	if _, ok := sel.Fields.Fields[0].Expr.(*ast.CollateExpr); !ok {
		t.Fatalf("COLLATE AST = %T", sel.Fields.Fields[0].Expr)
	}
	if f, ok := sel.Fields.Fields[1].Expr.(*ast.AggregateFuncExpr); !ok || f.Filter == nil {
		t.Fatalf("FILTER AST = %#v", sel.Fields.Fields[1].Expr)
	}
	switch f := sel.Fields.Fields[2].Expr.(type) {
	case *ast.FuncCallExpr:
		if f.WithinGroup == nil {
			t.Fatalf("WITHIN GROUP AST = %#v", f)
		}
	case *ast.AggregateFuncExpr:
		if f.WithinGroup == nil {
			t.Fatalf("WITHIN GROUP AST = %#v", f)
		}
	default:
		t.Fatalf("WITHIN GROUP AST = %T", f)
	}

	grouped := mustParseOne(t, `SELECT a FROM t GROUP BY GROUPING SETS ((a, b), ()), ROLLUP (a, b), CUBE (a, b);`).(*ast.SelectStmt)
	if len(grouped.GroupBy.Items) != 3 || grouped.GroupBy.Items[0].GroupingSet == nil || grouped.GroupBy.Items[1].GroupingSet.Type != ast.GroupingSetRollup {
		t.Fatalf("GroupBy = %#v", grouped.GroupBy)
	}

	ct := mustParseOne(t, `CREATE TABLE t (id int) WITH (fillfactor=70) TABLESPACE ts INHERITS (parent) PARTITION BY RANGE (id);`).(*ast.CreateTableStmt)
	if ct.WithOptions != "fillfactor = ?" || ct.Tablespace != "ts" || len(ct.Inherits) != 1 || ct.PartitionBy != "range" {
		t.Fatalf("CreateTableStmt = %#v", ct)
	}

	idx := mustParseOne(t, `CREATE INDEX CONCURRENTLY IF NOT EXISTS ix ON t USING btree (a text_pattern_ops DESC NULLS LAST) WITH (fillfactor=70) TABLESPACE ts;`).(*ast.CreateIndexStmt)
	col := idx.IndexColNames[0]
	if !idx.Concurrently || !idx.IfNotExists || idx.Using != "btree" || idx.WithOptions != "fillfactor = ?" || idx.Tablespace != "ts" || !col.Desc || col.NullsOrder != 2 || col.Opclass != "text_pattern_ops" {
		t.Fatalf("CreateIndexStmt = %#v col=%#v", idx, col)
	}
	if got := compactSQL(restoreNode(t, idx)); got != "CREATE INDEX CONCURRENTLY IF NOT EXISTS ix ON t USING BTREE (a DESC NULLS LAST text_pattern_ops) WITH (fillfactor = ?) TABLESPACE ts" {
		t.Fatalf("CreateIndexStmt restore = %q", got)
	}

	view := mustParseOne(t, `CREATE TEMP VIEW v WITH (security_barrier) AS SELECT 1 WITH LOCAL CHECK OPTION;`).(*ast.CreateViewStmt)
	if !view.Temporary || !view.HasCheckOption || view.CheckOption != 0 || view.WithOptions != "security_barrier" {
		t.Fatalf("CreateViewStmt = %#v", view)
	}
	if got := compactSQL(restoreNode(t, view)); !strings.Contains(got, "CREATE TEMPORARY VIEW v WITH (security_barrier) AS ") || !strings.HasSuffix(got, "WITH LOCAL CHECK OPTION") {
		t.Fatalf("CreateViewStmt restore = %q", got)
	}

	drop := mustParseOne(t, `DROP INDEX CONCURRENTLY IF EXISTS ix;`).(*ast.DropIndexStmt)
	if !drop.Concurrently || !drop.IfExists {
		t.Fatalf("DropIndexStmt = %#v", drop)
	}
	if got := compactSQL(restoreNode(t, drop)); got != "DROP INDEX CONCURRENTLY IF EXISTS ix" {
		t.Fatalf("DropIndexStmt restore = %q", got)
	}

	seq := mustParseOne(t, `CREATE SEQUENCE s INCREMENT BY 2 MINVALUE 1 NO MAXVALUE START WITH 10 CACHE 5 CYCLE;`).(*ast.CreateSequenceStmt)
	if len(seq.Options) != 6 || seq.Options[0].Tp != ast.SequenceOptionIncrement || seq.Options[0].Val != 2 {
		t.Fatalf("Sequence options = %#v", seq.Options)
	}

	begin := mustParseOne(t, `BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY DEFERRABLE;`).(*ast.BeginStmt)
	if begin.IsolationLevel != "SERIALIZABLE" || begin.ReadOnly == nil || !*begin.ReadOnly || begin.Deferrable == nil || !*begin.Deferrable {
		t.Fatalf("BeginStmt = %#v", begin)
	}

	explain := mustParseOne(t, `EXPLAIN (ANALYZE, VERBOSE, FORMAT JSON) SELECT 1;`).(*ast.ExplainStmt)
	if len(explain.Options) != 3 || explain.Format != "json" {
		t.Fatalf("ExplainStmt = %#v", explain)
	}
}

func TestParseSQLAlterTableSpecsAST(t *testing.T) {
	for _, sql := range []string{
		`ALTER TABLE t ALTER COLUMN a TYPE bigint;`,
		`ALTER TABLE t ALTER COLUMN a SET DATA TYPE text;`,
		`ALTER TABLE t ALTER COLUMN a SET DEFAULT 1;`,
		`ALTER TABLE t ALTER COLUMN a DROP DEFAULT;`,
		`ALTER TABLE t ALTER COLUMN a SET NOT NULL;`,
		`ALTER TABLE t ALTER COLUMN a DROP NOT NULL;`,
		`ALTER TABLE t ALTER COLUMN a SET STATISTICS 100;`,
		`ALTER TABLE t ALTER COLUMN a SET STORAGE external;`,
		`ALTER TABLE t ADD COLUMN IF NOT EXISTS c int;`,
		`ALTER TABLE t DROP COLUMN IF EXISTS c CASCADE;`,
		`ALTER TABLE t OWNER TO u;`,
		`ALTER TABLE t SET TABLESPACE ts;`,
		`ALTER TABLE t SET LOGGED;`,
		`ALTER TABLE t SET UNLOGGED;`,
		`ALTER TABLE t ENABLE TRIGGER ALL;`,
		`ALTER TABLE t DISABLE TRIGGER USER;`,
		`ALTER TABLE t SET (fillfactor = 70);`,
		`ALTER TABLE t RESET (fillfactor);`,
		`ALTER TABLE t ADD CONSTRAINT c CHECK (a > 0);`,
		`ALTER TABLE t ATTACH PARTITION p FOR VALUES FROM (1) TO (10);`,
		`ALTER TABLE t DETACH PARTITION p;`,
		`ALTER TABLE t VALIDATE CONSTRAINT c;`,
		`ALTER TABLE t DROP CONSTRAINT IF EXISTS c CASCADE;`,
		`ALTER TABLE t SET WITHOUT CLUSTER;`,
		`ALTER TABLE t SET WITHOUT OIDS;`,
		`ALTER TABLE t CLUSTER ON ix;`,
		`ALTER TABLE t INHERIT p;`,
		`ALTER TABLE t NO INHERIT p;`,
		`ALTER TABLE t OF typ;`,
		`ALTER TABLE t NOT OF;`,
		`ALTER TABLE t REPLICA IDENTITY NOTHING;`,
		`ALTER TABLE t REPLICA IDENTITY FULL;`,
		`ALTER TABLE t REPLICA IDENTITY DEFAULT;`,
		`ALTER TABLE t REPLICA IDENTITY USING INDEX ix;`,
		`ALTER TABLE t ENABLE ROW LEVEL SECURITY;`,
		`ALTER TABLE t DISABLE ROW LEVEL SECURITY;`,
		`ALTER TABLE t FORCE ROW LEVEL SECURITY;`,
		`ALTER TABLE t NO FORCE ROW LEVEL SECURITY;`,
		`ALTER TABLE t RENAME CONSTRAINT a TO b;`,
		`ALTER TABLE t ALTER CONSTRAINT c DEFERRABLE INITIALLY DEFERRED;`,
		`ALTER TABLE t ALTER COLUMN a TYPE bigint USING a::bigint;`,
		`ALTER TABLE t ALTER COLUMN a SET (n_distinct = 100);`,
		`ALTER TABLE t ALTER COLUMN a RESET (n_distinct);`,
		`ALTER TABLE t ENABLE ALWAYS TRIGGER tr;`,
		`ALTER TABLE t ENABLE REPLICA TRIGGER tr;`,
		`ALTER TABLE t DISABLE RULE r;`,
	} {
		if _, ok := mustParseOne(t, sql).(*ast.AlterTableStmt); !ok {
			t.Fatalf("%q: expected AlterTableStmt", sql)
		}
	}
}

func TestParseSQLAlterTableStructuredSpecsAST(t *testing.T) {
	cases := []struct {
		sql  string
		want ast.AlterTableType
		chk  func(*ast.AlterTableSpec) bool
	}{
		{`ALTER TABLE t ALTER COLUMN a DROP DEFAULT;`, ast.AlterTableDropDefault, func(s *ast.AlterTableSpec) bool { return s.OldColumnName.Name.L == "a" }},
		{`ALTER TABLE t ALTER COLUMN a SET NOT NULL;`, ast.AlterTableSetNotNull, func(s *ast.AlterTableSpec) bool { return s.OldColumnName.Name.L == "a" }},
		{`ALTER TABLE t ALTER COLUMN a DROP NOT NULL;`, ast.AlterTableDropNotNull, func(s *ast.AlterTableSpec) bool { return s.OldColumnName.Name.L == "a" }},
		{`ALTER TABLE t ALTER COLUMN a SET STATISTICS 100;`, ast.AlterTableSetStatistics, func(s *ast.AlterTableSpec) bool { return s.Num == 100 }},
		{`ALTER TABLE t ALTER COLUMN a SET STORAGE external;`, ast.AlterTableSetStorage, func(s *ast.AlterTableSpec) bool { return s.Storage == "external" }},
		{`ALTER TABLE t OWNER TO u;`, ast.AlterTableOwner, func(s *ast.AlterTableSpec) bool { return s.Owner == "u" }},
		{`ALTER TABLE t SET TABLESPACE ts;`, ast.AlterTableSetTablespace, func(s *ast.AlterTableSpec) bool { return s.Name == "ts" }},
		{`ALTER TABLE t ENABLE ALWAYS TRIGGER tr;`, ast.AlterTableEnableTrigger, func(s *ast.AlterTableSpec) bool { return s.TriggerMode == "ALWAYS" && s.Name == "tr" }},
		{`ALTER TABLE t DISABLE TRIGGER ALL;`, ast.AlterTableDisableTrigger, func(s *ast.AlterTableSpec) bool { return s.TriggerMode == "ALL" }},
		{`ALTER TABLE t SET (fillfactor = 70);`, ast.AlterTableSetRelOptions, func(s *ast.AlterTableSpec) bool { return s.RelOptions == "fillfactor = ?" }},
		{`ALTER TABLE t ALTER COLUMN a RESET (n_distinct);`, ast.AlterTableResetRelOptions, func(s *ast.AlterTableSpec) bool { return s.OldColumnName.Name.L == "a" && s.RelOptions == "n_distinct" }},
		{`ALTER TABLE t CLUSTER ON ix;`, ast.AlterTableSetCluster, func(s *ast.AlterTableSpec) bool { return s.Name == "ix" }},
		{`ALTER TABLE t SET WITHOUT CLUSTER;`, ast.AlterTableSetWithoutCluster, func(s *ast.AlterTableSpec) bool { return true }},
		{`ALTER TABLE t INHERIT p;`, ast.AlterTableInherit, func(s *ast.AlterTableSpec) bool { return s.NewTable.Name.L == "p" }},
		{`ALTER TABLE t OF typ;`, ast.AlterTableOf, func(s *ast.AlterTableSpec) bool { return s.NewTable.Name.L == "typ" }},
		{`ALTER TABLE t REPLICA IDENTITY USING INDEX ix;`, ast.AlterTableReplicaIdentity, func(s *ast.AlterTableSpec) bool { return s.Name == "ix" }},
		{`ALTER TABLE t FORCE ROW LEVEL SECURITY;`, ast.AlterTableRowLevelSecurity, func(s *ast.AlterTableSpec) bool { return s.RowLevelSecurity == "FORCE" }},
		{`ALTER TABLE t RENAME CONSTRAINT a TO b;`, ast.AlterTableRenameConstraint, func(s *ast.AlterTableSpec) bool { return s.Name == "a" && s.NewColumnName.Name.L == "b" }},
		{`ALTER TABLE t ALTER CONSTRAINT c DEFERRABLE INITIALLY DEFERRED;`, ast.AlterTableAlterConstraint, func(s *ast.AlterTableSpec) bool { return s.ConstraintOptions == "DEFERRABLE initially deferred" }},
		{`ALTER TABLE t ATTACH PARTITION p FOR VALUES FROM (1) TO (10);`, ast.AlterTableAttachPartition, func(s *ast.AlterTableSpec) bool {
			return s.NewTable.Name.L == "p" && s.PartitionBound.BoundType == "FROM" && len(s.PartitionBound.From) == 1 && len(s.PartitionBound.To) == 1
		}},
		{`ALTER TABLE t DETACH PARTITION p;`, ast.AlterTableDetachPartition, func(s *ast.AlterTableSpec) bool { return s.NewTable.Name.L == "p" }},
		{`ALTER TABLE t DISABLE RULE r;`, ast.AlterTableDisableRule, func(s *ast.AlterTableSpec) bool { return s.Name == "r" && s.TriggerMode == "RULE" }},
		{`ALTER TABLE t SET WITHOUT OIDS;`, ast.AlterTableSetOption, func(s *ast.AlterTableSpec) bool { return s.Name == "without" && s.SetValue == "oids" }},
	}
	for _, c := range cases {
		stmt := mustParseOne(t, c.sql).(*ast.AlterTableStmt)
		spec := stmt.Specs[0]
		if spec.Tp != c.want || !c.chk(spec) {
			t.Fatalf("%q: spec = %#v", c.sql, spec)
		}
	}
}

func TestParseSQLPostgresSyntaxExtensionsAST(t *testing.T) {
	queries := []string{
		`CREATE TABLE t (id int) WITH (fillfactor=70);`,
		`CREATE TABLE t (id int) TABLESPACE ts;`,
		`CREATE TABLE t (id int) INHERITS (parent);`,
		`CREATE TABLE t (LIKE other INCLUDING ALL);`,
		`CREATE TABLE t (id int) PARTITION BY RANGE (id);`,
		`CREATE TABLE t OF typ;`,
		`CREATE TABLE t AS SELECT 1 WITH NO DATA;`,
		`CREATE INDEX ix ON t (a DESC NULLS LAST, lower(b)) WHERE a > 1;`,
		`CREATE INDEX ix ON t USING gin (a jsonb_path_ops);`,
		`CREATE VIEW v WITH (security_barrier) AS SELECT 1;`,
		`CREATE VIEW v AS SELECT 1 WITH CHECK OPTION;`,
		`CREATE SEQUENCE s START 10 INCREMENT 2 MINVALUE 1 MAXVALUE 100 CACHE 5 CYCLE;`,
		`TRUNCATE TABLE a, b RESTART IDENTITY CASCADE;`,
		`INSERT INTO t (a) OVERRIDING SYSTEM VALUE VALUES (1);`,
		`UPDATE t SET a = DEFAULT, b = 1 FROM u WHERE t.id = u.id RETURNING t.*;`,
		`MERGE INTO t USING s ON t.id = s.id WHEN MATCHED AND s.x > 0 THEN UPDATE SET a = s.a WHEN NOT MATCHED THEN INSERT (id,a) VALUES (s.id,s.a) WHEN NOT MATCHED BY SOURCE THEN DELETE;`,
		`SELECT a, count(*) FILTER (WHERE b > 0) FROM t GROUP BY a;`,
		`SELECT a FROM t GROUP BY GROUPING SETS ((a), (b), ());`,
		`SELECT a FROM t GROUP BY ROLLUP (a,b);`,
		`SELECT a FROM t GROUP BY CUBE (a,b);`,
		`SELECT * FROM t TABLESAMPLE BERNOULLI (10);`,
		`SELECT * FROM t FOR UPDATE OF t NOWAIT;`,
		`SELECT * FROM t FETCH FIRST 10 ROWS ONLY;`,
		`SELECT * FROM t FETCH FIRST ROW ONLY;`,
		`SELECT * FROM t ORDER BY a OFFSET 5 ROWS FETCH NEXT 10 ROWS ONLY;`,
		`SELECT date '2020-01-01', timestamp '2020-01-01 00:00:00', time '12:00';`,
		`SELECT a COLLATE "C" FROM t;`,
		`SELECT x IS UNKNOWN, y IS NOT UNKNOWN FROM t;`,
		`SELECT a FROM t ORDER BY a USING <;`,
		`SELECT * FROM t FOR NO KEY UPDATE;`,
		`SELECT * FROM t FOR KEY SHARE;`,
		`SELECT ARRAY(SELECT 1);`,
		`SELECT (t).a FROM t;`,
		`SELECT * FROM ONLY t;`,
		`SELECT * FROM t*;`,
		`SELECT percentile_cont(0.5) WITHIN GROUP (ORDER BY x) FROM t;`,
		`SELECT array_agg(x ORDER BY x) FROM t;`,
		`SELECT sum(x) OVER w FROM t WINDOW w AS (PARTITION BY y);`,
		`SELECT sum(x) OVER w FROM t WINDOW w AS (ORDER BY y);`,
		`SET TIME ZONE 'UTC';`,
		`SET LOCAL TIME ZONE 'UTC';`,
		`SET SESSION TIME ZONE 'UTC';`,
		`SET TIME ZONE LOCAL;`,
		`SET TIME ZONE DEFAULT;`,
		`SELECT ARRAY[1,2,3], ROW(1,2), (1,2);`,
		`SELECT * FROM t WHERE (a,b) IN ((1,2),(3,4));`,
		`SELECT * FROM t WHERE a IS DISTINCT FROM b;`,
		`SELECT * FROM t WHERE a IS NOT DISTINCT FROM b;`,
		`SELECT * FROM t WHERE a SIMILAR TO 'x%';`,
		`BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY DEFERRABLE;`,
		`COMMIT AND CHAIN;`,
		`ROLLBACK AND NO CHAIN;`,
		`SAVEPOINT s;`,
		`RELEASE SAVEPOINT s;`,
		`ROLLBACK TO SAVEPOINT s;`,
		`EXPLAIN (ANALYZE, VERBOSE, FORMAT JSON) SELECT 1;`,
		`EXPLAIN ANALYZE SELECT 1;`,
		`COMMENT ON COLUMN t.a IS 'x';`,
		`COMMENT ON FUNCTION f(int) IS 'x';`,
		`COMMENT ON CONSTRAINT c ON t IS 'x';`,
		`COMMENT ON SCHEMA s IS 'x';`,
	}
	for _, sql := range queries {
		if _, err := pgparser.New().ParseOneStmt(sql); err != nil {
			t.Errorf("ParseOneStmt(%q): %v", sql, err)
		}
	}
}
