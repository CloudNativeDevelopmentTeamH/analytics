package analytics

import (
	"math"
	"testing"
)

func TestStore_IngestSessionEnded_ValidatesInput(t *testing.T) {
	store := NewStore()

	if err := store.IngestSessionEnded("", "cat", 100); err == nil {
		t.Fatalf("expected error for empty user id")
	}

	if err := store.IngestSessionEnded("u1", "cat", 0); err == nil {
		t.Fatalf("expected error for non-positive duration")
	}
}

func TestStore_GetGeneralStats_EmptyUser(t *testing.T) {
	store := NewStore()
	stats := store.GetGeneralStats("unknown")

	if stats.AvgOverallSeconds != 0 {
		t.Fatalf("expected avg overall 0, got %v", stats.AvgOverallSeconds)
	}
	if stats.AvgLast10Seconds != 0 {
		t.Fatalf("expected avg last10 0, got %v", stats.AvgLast10Seconds)
	}
	if len(stats.CategoryShares) != 0 {
		t.Fatalf("expected empty shares, got %+v", stats.CategoryShares)
	}
}

func TestStore_GetGeneralStats_AggregatesAndSortsShares(t *testing.T) {
	store := NewStore()

	mustIngest(t, store, "u1", "cat-a", 100)
	mustIngest(t, store, "u1", "cat-b", 200)
	mustIngest(t, store, "u1", "cat-a", 300)

	stats := store.GetGeneralStats("u1")

	// overall: (100+200+300)/3 = 200
	assertFloatApprox(t, stats.AvgOverallSeconds, 200)
	// last10: all 3 values -> same average
	assertFloatApprox(t, stats.AvgLast10Seconds, 200)

	if len(stats.CategoryShares) != 2 {
		t.Fatalf("expected 2 category shares, got %d", len(stats.CategoryShares))
	}

	// cat-a has 400, cat-b has 200 -> cat-a must come first
	if stats.CategoryShares[0].CategoryID != "cat-a" || stats.CategoryShares[0].TotalSeconds != 400 {
		t.Fatalf("unexpected first share: %+v", stats.CategoryShares[0])
	}
	if stats.CategoryShares[1].CategoryID != "cat-b" || stats.CategoryShares[1].TotalSeconds != 200 {
		t.Fatalf("unexpected second share: %+v", stats.CategoryShares[1])
	}

	assertFloatApprox(t, stats.CategoryShares[0].Share, 400.0/600.0)
	assertFloatApprox(t, stats.CategoryShares[1].Share, 200.0/600.0)
}

func TestStore_GetGeneralStats_Last10RingBuffer(t *testing.T) {
	store := NewStore()

	var total int64
	for i := int64(1); i <= 12; i++ {
		total += i
		mustIngest(t, store, "u1", "cat", i)
	}

	stats := store.GetGeneralStats("u1")

	// overall average for 1..12 = 78/12 = 6.5
	assertFloatApprox(t, stats.AvgOverallSeconds, float64(total)/12.0)

	// last10 are 3..12 => sum 75 => avg 7.5
	assertFloatApprox(t, stats.AvgLast10Seconds, 7.5)
}

func TestStore_GetAverageLengthByCategory_UncategorizedFallback(t *testing.T) {
	store := NewStore()

	// empty category -> Uncategorized
	mustIngest(t, store, "u1", "", 120)
	mustIngest(t, store, "u1", "", 180)

	avg := store.GetAverageLengthByCategory("u1", "")
	if avg.CategoryID != Uncategorized {
		t.Fatalf("expected category %s, got %s", Uncategorized, avg.CategoryID)
	}
	if avg.Count != 2 {
		t.Fatalf("expected count 2, got %d", avg.Count)
	}
	if avg.SumSeconds != 300 {
		t.Fatalf("expected sum 300, got %d", avg.SumSeconds)
	}
	assertFloatApprox(t, avg.AvgSeconds, 150)
}

func TestStore_GetAverageLengthByCategory_NoData(t *testing.T) {
	store := NewStore()
	avg := store.GetAverageLengthByCategory("u1", "cat-x")

	if avg.CategoryID != "cat-x" {
		t.Fatalf("expected category cat-x, got %s", avg.CategoryID)
	}
	if avg.Count != 0 || avg.SumSeconds != 0 || avg.AvgSeconds != 0 {
		t.Fatalf("expected zero values, got %+v", avg)
	}
}

func mustIngest(t *testing.T, store *Store, userID string, categoryID string, duration int64) {
	t.Helper()
	if err := store.IngestSessionEnded(userID, categoryID, duration); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
}

func assertFloatApprox(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("expected %.10f, got %.10f", want, got)
	}
}
