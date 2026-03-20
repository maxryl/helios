package db

import (
	"fmt"
	"testing"
)

func TestNewSchemaCache_Empty(t *testing.T) {
	sc := NewSchemaCache()
	if len(sc.Tables) != 0 {
		t.Errorf("expected empty Tables, got %d", len(sc.Tables))
	}
	if len(sc.Columns) != 0 {
		t.Errorf("expected empty Columns, got %d", len(sc.Columns))
	}
	if len(sc.Functions) != 0 {
		t.Errorf("expected empty Functions, got %d", len(sc.Functions))
	}
}

func TestSuggest_EmptyPrefix(t *testing.T) {
	sc := NewSchemaCache()
	got := sc.Suggest("")
	if got != nil {
		t.Errorf("expected nil for empty prefix, got %v", got)
	}
}

func TestSuggest_CaseInsensitive(t *testing.T) {
	sc := NewSchemaCache()
	sc.Tables = []string{"users", "user_roles"}

	got := sc.Suggest("us")
	if len(got) == 0 {
		t.Fatal("expected results for prefix 'us'")
	}
	foundUsers := false
	for _, s := range got {
		if s == "users" {
			foundUsers = true
		}
	}
	if !foundUsers {
		t.Errorf("expected 'users' in results, got %v", got)
	}
}

func TestSuggest_NoKeywords(t *testing.T) {
	sc := NewSchemaCache()
	// With no tables/functions/columns, nothing should match.
	got := sc.Suggest("SEL")
	if len(got) != 0 {
		t.Errorf("expected no results (keywords removed), got %v", got)
	}
}

func TestSuggest_Limit20(t *testing.T) {
	sc := NewSchemaCache()
	for i := 0; i < 25; i++ {
		sc.Tables = append(sc.Tables, fmt.Sprintf("t_%02d", i))
	}

	got := sc.Suggest("t_")
	if len(got) > 20 {
		t.Errorf("expected at most 20 results, got %d", len(got))
	}
}

func TestSuggest_ExactCaseFirst(t *testing.T) {
	sc := NewSchemaCache()
	sc.Tables = []string{"Select_log", "settings"}

	got := sc.Suggest("S")
	if len(got) == 0 {
		t.Fatal("expected results")
	}
	// First result should have exact case match (starts with "S").
	if got[0] != "Select_log" {
		t.Errorf("expected exact-case match first, got %q", got[0])
	}
}

func TestSuggest_IncludesFunctionsAndColumns(t *testing.T) {
	sc := NewSchemaCache()
	sc.Functions = []string{"my_func"}
	sc.Columns = map[string][]string{
		"orders": {"my_column"},
	}

	got := sc.Suggest("my")
	foundFunc, foundCol := false, false
	for _, s := range got {
		if s == "my_func" {
			foundFunc = true
		}
		if s == "my_column" {
			foundCol = true
		}
	}
	if !foundFunc {
		t.Error("expected my_func in results")
	}
	if !foundCol {
		t.Error("expected my_column in results")
	}
}

func TestSuggestColumns_Found(t *testing.T) {
	sc := NewSchemaCache()
	sc.Columns = map[string][]string{
		"users": {"id", "name", "email"},
	}

	got := sc.SuggestColumns("users")
	if len(got) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(got))
	}
	if got[0] != "id" || got[1] != "name" || got[2] != "email" {
		t.Errorf("unexpected columns: %v", got)
	}
}

func TestSuggestColumns_CaseInsensitive(t *testing.T) {
	sc := NewSchemaCache()
	sc.Columns = map[string][]string{
		"Users": {"id", "name"},
	}

	got := sc.SuggestColumns("users")
	if got == nil {
		t.Fatal("expected columns for case-insensitive lookup")
	}
	if len(got) != 2 {
		t.Errorf("expected 2 columns, got %d", len(got))
	}
}

func TestSuggestColumns_Unknown(t *testing.T) {
	sc := NewSchemaCache()
	got := sc.SuggestColumns("nonexistent")
	if got != nil {
		t.Errorf("expected nil for unknown table, got %v", got)
	}
}

func TestSuggestColumns_ReturnsCopy(t *testing.T) {
	sc := NewSchemaCache()
	sc.Columns = map[string][]string{
		"users": {"id", "name"},
	}

	got := sc.SuggestColumns("users")
	got[0] = "modified"
	if sc.Columns["users"][0] == "modified" {
		t.Error("SuggestColumns should return a copy, not the original slice")
	}
}
