package repository

import "testing"

// Internal test: buildListQuery is unexported, and the point of these cases is that
// adding the undated filter did not move a single byte of the existing query.

func TestBuildListQuery_UnchangedWhenUndatedAbsent(t *testing.T) {
	cases := []struct {
		name  string
		f     DocumentListFilter
		query string
		args  []any
	}{
		{
			name:  "no filters",
			f:     DocumentListFilter{Limit: 50},
			query: "SELECT d.id FROM documents d LEFT JOIN extractions e ON e.document_id = d.id WHERE 1=1 ORDER BY d.created_at DESC LIMIT ? OFFSET ?",
			args:  []any{50, 0},
		},
		{
			name: "every existing filter",
			f: DocumentListFilter{
				Q:                "rent",
				Tag:              "bank,tax",
				Year:             "2024",
				DocumentDateFrom: "2024-01-01",
				DocumentDateTo:   "2024-12-31",
				CreatedFrom:      "2024-01-01",
				CreatedTo:        "2024-12-31",
				Status:           "pending, failed",
				Sort:             "date_desc",
				Limit:            20,
				Offset:           40,
			},
			query: "SELECT d.id FROM documents d INNER JOIN (SELECT document_id FROM extractions_fts WHERE extractions_fts MATCH ? UNION SELECT document_id FROM extractions WHERE tags LIKE ?) f ON f.document_id = d.id LEFT JOIN extractions e ON e.document_id = d.id" +
				" WHERE 1=1 AND d.status IN (?,?) AND strftime('%Y', d.created_at) IN (?)" +
				" AND d.created_at >= ? AND d.created_at <= ?" +
				" AND json_valid(e.tags) AND EXISTS (SELECT 1 FROM json_each(e.tags) WHERE value IN (?,?))" +
				" AND e.document_date >= ? AND e.document_date <= ?" +
				" ORDER BY CASE WHEN e.document_date IS NULL OR e.document_date = '' THEN 1 ELSE 0 END, e.document_date DESC, d.created_at DESC LIMIT ? OFFSET ?",
			args: []any{`"rent"`, "%rent%", "pending", "failed", "2024", "2024-01-01", "2024-12-31", "bank", "tax", "2024-01-01", "2024-12-31", 20, 40},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			query, args := buildListQuery(tc.f, true, "d.id", true, false)
			if query != tc.query {
				t.Fatalf("query =\n%s\nwant\n%s", query, tc.query)
			}
			if len(args) != len(tc.args) {
				t.Fatalf("args = %#v want %#v", args, tc.args)
			}
			for i := range args {
				if args[i] != tc.args[i] {
					t.Fatalf("arg %d = %#v want %#v", i, args[i], tc.args[i])
				}
			}
			// Explicitly-false undated must be byte-identical to the absent case.
			off := tc.f
			off.Undated = false
			offQuery, _ := buildListQuery(off, true, "d.id", true, false)
			if offQuery != query {
				t.Fatalf("undated=false changed the query:\n%s", offQuery)
			}
		})
	}
}

func TestBuildListQuery_UndatedReplacesDateRange(t *testing.T) {
	query, args := buildListQuery(DocumentListFilter{
		Undated:          true,
		DocumentDateFrom: "2024-01-01",
		DocumentDateTo:   "2024-12-31",
		Limit:            50,
	}, true, "d.id", true, false)

	const want = "SELECT d.id FROM documents d LEFT JOIN extractions e ON e.document_id = d.id" +
		" WHERE 1=1 AND (e.document_date IS NULL OR e.document_date = '')" +
		" ORDER BY d.created_at DESC LIMIT ? OFFSET ?"
	if query != want {
		t.Fatalf("query =\n%s\nwant\n%s", query, want)
	}
	// The ignored range contributes no placeholders.
	if len(args) != 2 {
		t.Fatalf("args = %#v", args)
	}
}

func TestFTSMatchArg(t *testing.T) {
	if got := ftsMatchArg("rent"); got != `"rent"` {
		t.Fatalf("got %q", got)
	}
	if got := ftsMatchArg("hello world"); got != `"hello" AND "world"` {
		t.Fatalf("got %q", got)
	}
	if got := ftsMatchArg(`AND OR NOT ( * ^`); got != `"AND" AND "OR" AND "NOT"` {
		t.Fatalf("got %q", got)
	}
	if got := ftsMatchArg("!!!"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildListQuery_UndatedJoinsExtractionsForExport(t *testing.T) {
	// The export path does not join extractions unconditionally, so undated must ask for it.
	query, _ := buildListQuery(DocumentListFilter{Undated: true}, false, "d.id, d.created_at", false, false)
	const want = "SELECT d.id, d.created_at FROM documents d LEFT JOIN extractions e ON e.document_id = d.id" +
		" WHERE 1=1 AND (e.document_date IS NULL OR e.document_date = '') ORDER BY d.created_at"
	if query != want {
		t.Fatalf("query =\n%s\nwant\n%s", query, want)
	}
}
