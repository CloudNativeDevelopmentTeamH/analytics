package grpc

import (
	"context"

	"github.com/CloudNativeDevelopmentTeamH/analytics/backend/app/analytics"
	analyticsv1 "github.com/CloudNativeDevelopmentTeamH/analytics/backend/proto/analytics/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AnalyticsServer implements analytics.v1.AnalyticsService.
type AnalyticsServer struct {
	analyticsv1.UnimplementedAnalyticsServiceServer
	store *analytics.Store
}

func NewAnalyticsServer(store *analytics.Store) *AnalyticsServer {
	return &AnalyticsServer{store: store}
}

func (s *AnalyticsServer) GetGeneralStats(ctx context.Context, req *analyticsv1.GetGeneralStatsRequest) (*analyticsv1.GetGeneralStatsResponse, error) {
	userID := req.GetUserId()
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	gs := s.store.GetGeneralStats(userID)

	resp := &analyticsv1.GetGeneralStatsResponse{
		AverageLengthOverallSeconds: gs.AvgOverallSeconds,
		AverageLengthLast10Seconds:  gs.AvgLast10Seconds,
		CategoryShares:              make([]*analyticsv1.CategoryShare, 0, len(gs.CategoryShares)),
	}

	for _, cs := range gs.CategoryShares {
		resp.CategoryShares = append(resp.CategoryShares, &analyticsv1.CategoryShare{
			CategoryId:   cs.CategoryID,
			Share:        cs.Share,
			TotalSeconds: cs.TotalSeconds,
		})
	}

	return resp, nil
}

func (s *AnalyticsServer) GetAverageLengthByCategory(ctx context.Context, req *analyticsv1.GetAverageLengthByCategoryRequest) (*analyticsv1.GetAverageLengthByCategoryResponse, error) {
	userID := req.GetUserId()
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	categoryID := req.GetCategoryId()
	if categoryID == "" {
		// du willst category als request param required
		return nil, status.Error(codes.InvalidArgument, "category_id is required")
	}

	res := s.store.GetAverageLengthByCategory(userID, categoryID)

	return &analyticsv1.GetAverageLengthByCategoryResponse{
		CategoryId:           res.CategoryID,
		AverageLengthSeconds: res.AvgSeconds,
		Count:                res.Count,
		SumSeconds:           res.SumSeconds,
	}, nil
}
