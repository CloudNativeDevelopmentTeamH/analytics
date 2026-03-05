package analytics

import (
	"errors"
	"sort"
	"sync"
)

const Uncategorized = "uncategorized"

type CategoryAgg struct {
	Count      int64
	SumSeconds int64
}

type UserAgg struct {
	OverallCount      int64
	OverallSumSeconds int64

	// Ringbuffer für die letzten 10 Session-Dauern (in Sekunden)
	Last10     [10]int64
	Last10Size int // <= 10
	Last10Idx  int // next write index
	Last10Sum  int64

	ByCategory map[string]*CategoryAgg

	// Für Kafka später (at-least-once): optional Idempotency
	// SeenSessionIDs map[string]struct{}
}

type Store struct {
	mu    sync.RWMutex
	users map[string]*UserAgg
}

func NewStore() *Store {
	return &Store{
		users: make(map[string]*UserAgg),
	}
}

// IngestSessionEnded updated aggregates for a finished focus session.
// categoryID may be empty -> Uncategorized.
// durationSeconds must be > 0.
func (s *Store) IngestSessionEnded(userID string, categoryID string, durationSeconds int64) error {
	if userID == "" {
		return errors.New("userID must not be empty")
	}
	if durationSeconds <= 0 {
		return errors.New("durationSeconds must be > 0")
	}
	if categoryID == "" {
		categoryID = Uncategorized
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ua := s.users[userID]
	if ua == nil {
		ua = &UserAgg{
			ByCategory: make(map[string]*CategoryAgg),
		}
		s.users[userID] = ua
	}

	// overall
	ua.OverallCount++
	ua.OverallSumSeconds += durationSeconds

	// last 10 ringbuffer
	if ua.Last10Size < 10 {
		ua.Last10[ua.Last10Idx] = durationSeconds
		ua.Last10Sum += durationSeconds
		ua.Last10Size++
		ua.Last10Idx = (ua.Last10Idx + 1) % 10
	} else {
		old := ua.Last10[ua.Last10Idx]
		ua.Last10[ua.Last10Idx] = durationSeconds
		ua.Last10Sum += durationSeconds - old
		ua.Last10Idx = (ua.Last10Idx + 1) % 10
	}

	// per category
	ca := ua.ByCategory[categoryID]
	if ca == nil {
		ca = &CategoryAgg{}
		ua.ByCategory[categoryID] = ca
	}
	ca.Count++
	ca.SumSeconds += durationSeconds

	return nil
}

// GetGeneralStats returns:
// - average length overall
// - average length last 10 (or fewer if <10 sessions exist)
// - category shares (time-based), sorted by TotalSeconds desc
func (s *Store) GetGeneralStats(userID string) GeneralStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ua := s.users[userID]
	if ua == nil || ua.OverallCount == 0 {
		return GeneralStats{
			AvgOverallSeconds: 0,
			AvgLast10Seconds:  0,
			CategoryShares:    []CategoryShare{},
		}
	}

	avgOverall := float64(ua.OverallSumSeconds) / float64(ua.OverallCount)

	var avgLast10 float64
	if ua.Last10Size > 0 {
		avgLast10 = float64(ua.Last10Sum) / float64(ua.Last10Size)
	}

	total := ua.OverallSumSeconds
	shares := make([]CategoryShare, 0, len(ua.ByCategory))
	if total > 0 {
		for cid, ca := range ua.ByCategory {
			shares = append(shares, CategoryShare{
				CategoryID:   cid,
				Share:        float64(ca.SumSeconds) / float64(total),
				TotalSeconds: ca.SumSeconds,
			})
		}
	}

	// stable output ordering
	sort.Slice(shares, func(i, j int) bool {
		if shares[i].TotalSeconds == shares[j].TotalSeconds {
			return shares[i].CategoryID < shares[j].CategoryID
		}
		return shares[i].TotalSeconds > shares[j].TotalSeconds
	})

	return GeneralStats{
		AvgOverallSeconds: avgOverall,
		AvgLast10Seconds:  avgLast10,
		CategoryShares:    shares,
	}
}

// GetAverageLengthByCategory returns average duration for given category.
// If categoryID empty -> Uncategorized.
// If no data -> zeros.
func (s *Store) GetAverageLengthByCategory(userID, categoryID string) AvgByCategory {
	if categoryID == "" {
		categoryID = Uncategorized
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	ua := s.users[userID]
	if ua == nil {
		return AvgByCategory{CategoryID: categoryID}
	}

	ca := ua.ByCategory[categoryID]
	if ca == nil || ca.Count == 0 {
		return AvgByCategory{CategoryID: categoryID}
	}

	return AvgByCategory{
		CategoryID: categoryID,
		AvgSeconds: float64(ca.SumSeconds) / float64(ca.Count),
		Count:      ca.Count,
		SumSeconds: ca.SumSeconds,
	}
}
