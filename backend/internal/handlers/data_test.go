package handlers

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestConvertValueBytes(t *testing.T) {
	got := convertValue([]byte("hello"))
	if s, ok := got.(string); !ok || s != "hello" {
		t.Errorf("convertValue([]byte) = %v (%T), want string \"hello\"", got, got)
	}
}

func TestConvertValueTimeBasic(t *testing.T) {
	// 13:45:30 in microseconds = (13*3600 + 45*60 + 30) * 1_000_000.
	const us = (13*3600 + 45*60 + 30) * int64(1_000_000)
	got := convertValue(pgtype.Time{Microseconds: us, Valid: true})
	if got != "13:45:30" {
		t.Errorf("convertValue(pgtype.Time HH:MM:SS) = %v, want \"13:45:30\"", got)
	}
}

func TestConvertValueTimeWithFraction(t *testing.T) {
	const us = (1*3600+2*60+3)*int64(1_000_000) + 456789
	got := convertValue(pgtype.Time{Microseconds: us, Valid: true})
	if got != "01:02:03.456789" {
		t.Errorf("convertValue(pgtype.Time w/ fraction) = %v, want \"01:02:03.456789\"", got)
	}
}

func TestConvertValueTimeInvalidIsNil(t *testing.T) {
	got := convertValue(pgtype.Time{Valid: false})
	if got != nil {
		t.Errorf("convertValue(invalid pgtype.Time) = %v, want nil", got)
	}
}

func TestConvertValueIntervalPureTime(t *testing.T) {
	const us = (2*3600 + 30*60 + 15) * int64(1_000_000)
	got := convertValue(pgtype.Interval{Microseconds: us, Valid: true})
	if got != "02:30:15" {
		t.Errorf("convertValue(pgtype.Interval pure time) = %v, want \"02:30:15\"", got)
	}
}

func TestConvertValueIntervalWithMonthsDays(t *testing.T) {
	got := convertValue(pgtype.Interval{Months: 2, Days: 5, Microseconds: 0, Valid: true})
	if got != "2 months 5 days 00:00:00" {
		t.Errorf("convertValue(pgtype.Interval w/ months+days) = %v", got)
	}
}

func TestConvertValuePassthrough(t *testing.T) {
	// Anything not specifically handled returns as-is so pgx + JSON
	// marshalling produces the same output as before.
	got := convertValue(42)
	if got != 42 {
		t.Errorf("convertValue(42) = %v, want 42", got)
	}
}

// TestExplainMultiStatementGuard verifies that splitStatements, which backs the
// ExplainQuery multi-statement guard, correctly distinguishes single from
// multi-statement input. The guard itself (in ExplainQuery) rejects len>1 to
// prevent simple-query-protocol injection like "SELECT 1; DROP TABLE t".
func TestExplainMultiStatementGuard(t *testing.T) {
	cases := []struct {
		name     string
		sql      string
		wantMany bool
	}{
		{"single select", "SELECT 1", false},
		{"single select with semicolon", "SELECT 1;", false},
		{"multi-statement injection", "SELECT 1; DROP TABLE t", true},
		{"multi-statement with comment suffix", "SELECT 1; DROP TABLE t;--", true},
		{"double semi in string literal", "SELECT ';' AS x", false},
		{"two real statements", "SELECT 1; SELECT 2", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stmts := splitStatements(tc.sql)
			gotMany := len(stmts) > 1
			if gotMany != tc.wantMany {
				t.Errorf("splitStatements(%q) returned %d stmts (wantMany=%v)",
					tc.sql, len(stmts), tc.wantMany)
			}
		})
	}
}
