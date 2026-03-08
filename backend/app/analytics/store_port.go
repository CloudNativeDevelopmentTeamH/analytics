package analytics

type StatsStore interface {
	IngestSessionEnded(userID string, categoryID string, durationSeconds int64) error
	GetGeneralStats(userID string) GeneralStats
	GetAverageLengthByCategory(userID, categoryID string) AvgByCategory
}
