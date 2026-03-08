package analytics

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
)

type PostgresStore struct {
	db *sqlx.DB
}

func NewPostgresStore(dsn string) (*PostgresStore, error) {
	if dsn == "" {
		return nil, errors.New("dsn must not be empty")
	}

	db, err := sqlx.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *PostgresStore) IngestSessionEnded(userID string, categoryID string, durationSeconds int64) error {
	if userID == "" {
		return errors.New("userID must not be empty")
	}
	if durationSeconds <= 0 {
		return errors.New("durationSeconds must be > 0")
	}
	if categoryID == "" {
		categoryID = Uncategorized
	}

	_, err := s.db.Exec(`
		INSERT INTO analytics_sessions (user_id, category_id, duration_seconds)
		VALUES ($1, $2, $3)
	`, userID, categoryID, durationSeconds)

	if err != nil {
		return fmt.Errorf("insert analytics session: %w", err)
	}

	return nil
}

func (s *PostgresStore) GetGeneralStats(userID string) GeneralStats {
	if userID == "" {
		return GeneralStats{CategoryShares: []CategoryShare{}}
	}

	var overallCount int64
	var overallSum int64
	var avgOverall sql.NullFloat64

	err := s.db.QueryRowx(`
		SELECT
			COUNT(*) AS overall_count,
			COALESCE(SUM(duration_seconds), 0) AS overall_sum,
			COALESCE(AVG(duration_seconds)::float8, 0) AS avg_overall
		FROM analytics_sessions
		WHERE user_id = $1
	`, userID).Scan(&overallCount, &overallSum, &avgOverall)
	if err != nil {
		return GeneralStats{CategoryShares: []CategoryShare{}}
	}

	if overallCount == 0 {
		return GeneralStats{CategoryShares: []CategoryShare{}}
	}

	var avgLast10 sql.NullFloat64
	err = s.db.QueryRowx(`
		SELECT COALESCE(AVG(t.duration_seconds)::float8, 0)
		FROM (
			SELECT duration_seconds
			FROM analytics_sessions
			WHERE user_id = $1
			ORDER BY id DESC
			LIMIT 10
		) t
	`, userID).Scan(&avgLast10)
	if err != nil {
		avgLast10 = sql.NullFloat64{Float64: 0, Valid: true}
	}

	rows, err := s.db.Queryx(`
		SELECT category_id, SUM(duration_seconds) AS total_seconds
		FROM analytics_sessions
		WHERE user_id = $1
		GROUP BY category_id
		ORDER BY total_seconds DESC, category_id ASC
	`, userID)
	if err != nil {
		return GeneralStats{
			AvgOverallSeconds: avgOverall.Float64,
			AvgLast10Seconds:  avgLast10.Float64,
			CategoryShares:    []CategoryShare{},
		}
	}
	defer rows.Close()

	shares := make([]CategoryShare, 0)
	for rows.Next() {
		var categoryID string
		var totalSeconds int64
		if scanErr := rows.Scan(&categoryID, &totalSeconds); scanErr != nil {
			continue
		}

		share := 0.0
		if overallSum > 0 {
			share = float64(totalSeconds) / float64(overallSum)
		}

		shares = append(shares, CategoryShare{
			CategoryID:   categoryID,
			Share:        share,
			TotalSeconds: totalSeconds,
		})
	}

	return GeneralStats{
		AvgOverallSeconds: avgOverall.Float64,
		AvgLast10Seconds:  avgLast10.Float64,
		CategoryShares:    shares,
	}
}

func (s *PostgresStore) GetAverageLengthByCategory(userID, categoryID string) AvgByCategory {
	if categoryID == "" {
		categoryID = Uncategorized
	}

	if userID == "" {
		return AvgByCategory{CategoryID: categoryID}
	}

	var count int64
	var sumSeconds int64
	var avg sql.NullFloat64

	err := s.db.QueryRowx(`
		SELECT
			COUNT(*) AS count,
			COALESCE(SUM(duration_seconds), 0) AS sum_seconds,
			COALESCE(AVG(duration_seconds)::float8, 0) AS avg_seconds
		FROM analytics_sessions
		WHERE user_id = $1 AND category_id = $2
	`, userID, categoryID).Scan(&count, &sumSeconds, &avg)
	if err != nil || count == 0 {
		return AvgByCategory{CategoryID: categoryID}
	}

	return AvgByCategory{
		CategoryID: categoryID,
		AvgSeconds: avg.Float64,
		Count:      count,
		SumSeconds: sumSeconds,
	}
}
