package analytics

type CategoryShare struct {
	// category_id can be "uncategorized"
	CategoryID string `json:"categoryId"`
	// Share 0..1 (time-based: totalSeconds / overallSumSeconds)
	Share float64 `json:"share"`
	// Total time in this category
	TotalSeconds int64 `json:"totalSeconds"`
}

type GeneralStats struct {
	AvgOverallSeconds float64         `json:"averageLengthOverallSeconds"`
	AvgLast10Seconds  float64         `json:"averageLengthLast10Seconds"`
	CategoryShares    []CategoryShare `json:"categoryShares"`
}

type AvgByCategory struct {
	CategoryID string  `json:"categoryId"`
	AvgSeconds float64 `json:"averageLengthSeconds"`
	Count      int64   `json:"count"`
	SumSeconds int64   `json:"sumSeconds"`
}
