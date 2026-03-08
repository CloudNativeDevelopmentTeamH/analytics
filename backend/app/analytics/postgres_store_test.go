package analytics

import "testing"

func TestNewPostgresStore_EmptyDSN(t *testing.T) {
	store, err := NewPostgresStore("")
	if err == nil {
		t.Fatalf("expected error for empty dsn")
	}
	if store != nil {
		t.Fatalf("expected nil store for empty dsn")
	}
}

func TestPostgresStore_IngestSessionEnded_ValidatesInput(t *testing.T) {
	store := &PostgresStore{}

	if err := store.IngestSessionEnded("", "category", 100); err == nil {
		t.Fatalf("expected error for empty userID")
	}

	if err := store.IngestSessionEnded("user", "category", 0); err == nil {
		t.Fatalf("expected error for non-positive duration")
	}
}

func TestPostgresStore_Getters_WithEmptyUser(t *testing.T) {
	store := &PostgresStore{}

	stats := store.GetGeneralStats("")
	if stats.AvgOverallSeconds != 0 || stats.AvgLast10Seconds != 0 {
		t.Fatalf("expected zero stats for empty user, got %+v", stats)
	}
	if len(stats.CategoryShares) != 0 {
		t.Fatalf("expected no category shares, got %+v", stats.CategoryShares)
	}

	avg := store.GetAverageLengthByCategory("", "")
	if avg.CategoryID != Uncategorized {
		t.Fatalf("expected category %s, got %s", Uncategorized, avg.CategoryID)
	}
	if avg.Count != 0 || avg.SumSeconds != 0 || avg.AvgSeconds != 0 {
		t.Fatalf("expected zero values, got %+v", avg)
	}
}
