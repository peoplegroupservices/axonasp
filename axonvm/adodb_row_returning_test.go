package axonvm

import "testing"

// Classic ASP against MSDASQL reads a new row's id straight back out of an
// INSERT ... RETURNING. Routing that through Exec runs the statement but throws
// the rows away, so the page sees an empty recordset and concludes the write
// failed — while the row is sitting in the table. The PGS portal does this in 16
// files, and it is what stops a user signing in locally.
func TestADODBIsQueryRecognisesRowReturningStatements(t *testing.T) {
	vm := &VM{}

	tests := []struct {
		name string
		sql  string
		want bool
	}{
		{"plain select", "select * from t", true},
		{"leading whitespace and case", "   SeLeCt 1", true},

		{"insert returning", "insert into t (a) values (1) returning id", true},
		{"insert returning, multiple columns", "INSERT INTO t (a) VALUES (1) RETURNING id, guid", true},
		{"update returning", "update t set a=1 where id=2 returning id", true},
		{"delete returning", "delete from t where id=2 returning id", true},

		{"plain insert is not a query", "insert into t (a) values (1)", false},
		{"plain update is not a query", "update t set a=1", false},
		{"plain delete is not a query", "delete from t where id=1", false},

		// A common table expression returns rows through the select it feeds.
		{"cte", "with recent as (select * from t) select * from recent", true},
		{"cte, uppercase", "WITH x AS (SELECT 1) SELECT * FROM x", true},

		// The word may appear in data. Matching it there would send a plain
		// insert down the query path.
		{"returning inside a string literal", "insert into notes (body) values ('returning next week')", false},
		{"returning inside a literal with an escaped quote",
			"insert into notes (body) values ('it''s returning soon')", false},
		{"returning as part of an identifier", "insert into t (returning_date) values (now())", false},

		{"ddl is not a query", "create table t (id int)", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := vm.adodbIsQuery(tc.sql); got != tc.want {
				t.Errorf("adodbIsQuery(%q) = %v, want %v", tc.sql, got, tc.want)
			}
		})
	}
}

func TestADODBStripStringLiterals(t *testing.T) {
	tests := []struct{ in, want string }{
		{"a 'b' c", "a ' ' c"},
		{"a 'it''s' c", "a '     ' c"},
		{"no literals here", "no literals here"},
		{"'unterminated", "'            "},
	}
	for _, tc := range tests {
		if got := adodbStripStringLiterals(tc.in); got != tc.want {
			t.Errorf("adodbStripStringLiterals(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
