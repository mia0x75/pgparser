package parser_test

import (
	"testing"

	"github.com/mia0x75/pgparser/ast"
)

// Regression for shared binary set-operation nodes and their query tails.

func TestSetOpSharedOperationAndAllTypes(t *testing.T) {
	ops := []struct {
		sql string
		op  ast.SetOprType
	}{
		{"SELECT 1 UNION SELECT 2", ast.Union},
		{"SELECT 1 UNION ALL SELECT 2", ast.UnionAll},
		{"SELECT 1 INTERSECT SELECT 2", ast.Intersect},
		{"SELECT 1 INTERSECT ALL SELECT 2", ast.IntersectAll},
		{"SELECT 1 EXCEPT SELECT 2", ast.Except},
		{"SELECT 1 EXCEPT ALL SELECT 2", ast.ExceptAll},
	}
	for _, tc := range ops {
		t.Run(tc.sql, func(t *testing.T) {
			n, ok := mustParseOne(t, tc.sql).(*ast.SetOprStmt)
			if !ok {
				t.Fatalf("expected *ast.SetOprStmt, got %T", mustParseOne(t, tc.sql))
			}
			if n.Operator != tc.op {
				t.Fatalf("operator = %v, want %v", n.Operator, tc.op)
			}
			// The shared core must be reachable and point at the same data.
			core := n.SetOperationCore()
			if core == nil || core.Left != n.Left || core.Right != n.Right || core.Operator != n.Operator {
				t.Fatalf("shared core %+v diverges from statement %+v", core, n)
			}
			restoreString(t, n)
		})
	}
}

func TestSetOpAssociativityAndIntersectPrecedence(t *testing.T) {
	// "A op1 B op2 C" nests on the left regardless of which operator is
	// outer, but INTERSECT binds tighter than UNION/EXCEPT.
	st, ok := mustParseOne(t, "SELECT 1 UNION ALL SELECT 2 UNION SELECT 3;").(*ast.SetOprStmt)
	if !ok {
		t.Fatal("expected *ast.SetOprStmt")
	}
	if st.Operator != ast.Union {
		t.Fatalf("outer operator = %v, want UNION", st.Operator)
	}
	inner, ok := st.Left.(*ast.SetOprStmt)
	if !ok || inner.Operator != ast.UnionAll {
		t.Fatalf("inner = %T, want *ast.SetOprStmt with UNION ALL", st.Left)
	}

	st, ok = mustParseOne(t, "SELECT 1 UNION SELECT 2 INTERSECT SELECT 3;").(*ast.SetOprStmt)
	if !ok {
		t.Fatal("expected *ast.SetOprStmt")
	}
	if st.Operator != ast.Union {
		t.Fatalf("outer operator = %v, want UNION", st.Operator)
	}
	if grouped, ok := st.Right.(*ast.SetOprStmt); !ok || grouped.Operator != ast.Intersect {
		t.Fatalf("right = %T, want *ast.SetOprStmt with INTERSECT", st.Right)
	}
}

func TestSetOpParenthesizedScopeAndTail(t *testing.T) {
	t.Run("parenthesized select scope", func(t *testing.T) {
		// Parens around a plain SELECT stay as IsInBraces on the
		// SelectStmt, so ORDER BY / LIMIT are fields of the branch, not
		// of the outer set operation.
		st, ok := mustParseOne(t, "(SELECT a FROM t ORDER BY a LIMIT 3) UNION (SELECT b FROM u LIMIT 2);").(*ast.SetOprStmt)
		if !ok {
			t.Fatal("expected *ast.SetOprStmt")
		}
		left, ok := st.Left.(*ast.SelectStmt)
		if !ok || !left.IsInBraces {
			t.Fatalf("left = %T, want a parenthesized *ast.SelectStmt", st.Left)
		}
		if left.OrderBy == nil || limitClauseNode(t, left.Limit).Count == nil {
			t.Fatalf("left branch missing tail: %+v", left)
		}
		if st.Tail.OrderBy != nil || st.Tail.Limit != nil {
			t.Fatalf("outer tail should be empty, got %+v", st.Tail)
		}
		if right, ok := st.Right.(*ast.SelectStmt); !ok || !right.IsInBraces {
			t.Fatalf("right = %T, want a parenthesized *ast.SelectStmt", st.Right)
		}
		restoreString(t, st)
	})

	t.Run("with clause on set operation", func(t *testing.T) {
		// WITH applies to the whole set operation, not its first branch.
		st, ok := mustParseOne(t, "WITH c AS (SELECT 1) SELECT 1 UNION SELECT 2;").(*ast.SetOprStmt)
		if !ok {
			t.Fatal("expected *ast.SetOprStmt")
		}
		if st.With == nil || len(st.With.CTEs) != 1 {
			t.Fatalf("expected one CTE on the set operation, got %#v", st.With)
		}
		if branch, ok := st.Left.(*ast.SelectStmt); ok && branch.With != nil {
			t.Fatal("WITH should live on the set-op, not the branch")
		}
		restoreString(t, st)
	})

	t.Run("values and table tails", func(t *testing.T) {
		// VALUES and TABLE have no ORDER BY / LIMIT fields of their own;
		// those clauses are carried by a surrounding QueryTailStmt or
		// kept on the set-op's BaseQueryTail.
		for _, tc := range []struct{ sql, want string }{
			{"VALUES (1) UNION VALUES (2) LIMIT 1", "VALUES (1) UNION VALUES (2) LIMIT 1"},
			{"TABLE t LIMIT 2", `TABLE "t" LIMIT 2`},
			{"TABLE t ORDER BY 1", `TABLE "t" ORDER BY 1`},
		} {
			t.Run(tc.sql, func(t *testing.T) {
				n := mustParseOne(t, tc.sql)
				// Confirm the tail node is a shared wrapper (QueryTailStmt)
				// or the set-op's own BaseQueryTail.
				if v, ok := n.(*ast.QueryTailStmt); ok {
					if v.QueryTailStmtCore() == nil {
						t.Fatal("tail core is nil")
					}
				} else if v, ok := n.(*ast.SetOprStmt); ok {
					if v.Tail.QueryTailCore() == nil {
						t.Fatal("set-op tail core is nil")
					}
				} else {
					t.Fatalf("unexpected statement type %T for %q", n, tc.sql)
				}
				if got := restoreString(t, n); got != compactSQL(tc.want) {
					t.Fatalf("restore = %q, want %q", got, compactSQL(tc.want))
				}
			})
		}
	})
}

func TestQueryTailStmtVisitorRewrite(t *testing.T) {
	t.Run("rewrite inner expression and tail values", func(t *testing.T) {
		stmt := mustParseOne(t, "VALUES (1) LIMIT 1")
		visitor := pgRewriteVisitor{rewrite: func(n ast.Node) ast.Node {
			if child, ok := n.(ast.ValueExpr); ok && child.GetValue() == int64(1) {
				return ast.NewValueExpr(int64(2))
			}
			return n
		}}
		if _, ok := stmt.Accept(visitor); !ok {
			t.Fatal("visitor failed")
		}
		wrapper, ok := stmt.(*ast.QueryTailStmt)
		if !ok {
			t.Fatalf("expected *ast.QueryTailStmt, got %T", stmt)
		}
		if limitClauseNode(t, wrapper.Tail.Limit).Count.(ast.ValueExpr).GetValue() != int64(2) {
			t.Fatalf("tail limit not rewritten: %#v", wrapper.Tail.Limit)
		}
		if got := restoreString(t, stmt); got != "VALUES (2) LIMIT 2" {
			t.Fatalf("rewritten = %q, want VALUES (2) LIMIT 2", got)
		}
	})

	// The visitor rewrites both branch expressions and the outer set-op
	// tail in a single pass.
	stmt, ok := mustParseOne(t, "(SELECT 1) UNION (SELECT 2) LIMIT 1;").(*ast.SetOprStmt)
	if !ok {
		t.Fatal("expected *ast.SetOprStmt")
	}
	visitor := pgRewriteVisitor{rewrite: func(n ast.Node) ast.Node {
		if child, ok := n.(ast.ValueExpr); ok {
			if v, ok := child.GetValue().(int64); ok {
				if v == 1 {
					return ast.NewValueExpr(int64(3))
				}
				if v == 2 {
					return ast.NewValueExpr(int64(4))
				}
			}
		}
		return n
	}}
	if _, ok := stmt.Accept(visitor); !ok {
		t.Fatal("visitor failed")
	}
	left, ok := stmt.Left.(*ast.SelectStmt)
	if !ok || !left.IsInBraces {
		t.Fatalf("left = %T, want a parenthesized *ast.SelectStmt", stmt.Left)
	}
	if left.Fields.Items[0].Expr.(ast.ValueExpr).GetValue() != int64(3) {
		t.Fatalf("left branch not rewritten: %#v", left.Fields.Items[0])
	}
	if got := limitClauseNode(t, stmt.Tail.Limit).Count.(ast.ValueExpr).GetValue(); got != int64(3) {
		t.Fatalf("outer tail limit = %v, want 3", got)
	}
	if got := restoreString(t, stmt); got != "SELECT 3 UNION SELECT 4 LIMIT 3" {
		t.Fatalf("rewritten = %q, want %q", got, "SELECT 3 UNION SELECT 4 LIMIT 3")
	}
}
