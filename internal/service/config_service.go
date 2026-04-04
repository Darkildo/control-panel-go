package service

import (
	"context"

	"control-panel-go/internal/converter"
	"control-panel-go/internal/models"
	"control-panel-go/internal/repository"

	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "control-panel-go/gen/pb"
)

type ConfigService struct {
	pb.UnimplementedConfigServiceServer
	configRepo *repository.ConfigRepository
	deviceRepo *repository.DeviceRepository
	logger     zerolog.Logger
}

func NewConfigService(configRepo *repository.ConfigRepository, deviceRepo *repository.DeviceRepository, logger zerolog.Logger) *ConfigService {
	return &ConfigService{
		configRepo: configRepo,
		deviceRepo: deviceRepo,
		logger:     logger,
	}
}

func (s *ConfigService) CreateConfig(ctx context.Context, req *pb.CreateConfigRequest) (*pb.ConfigResponse, error) {
	if req.DeviceId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "device_id is required")
	}
	if req.Version == "" {
		return nil, status.Error(codes.InvalidArgument, "version is required")
	}
	if req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}

	device, err := s.deviceRepo.GetByID(ctx, req.DeviceId)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get device")
		return nil, status.Error(codes.Internal, "internal error")
	}
	if device == nil {
		return nil, status.Errorf(codes.NotFound, "device with id %d not found", req.DeviceId)
	}

	config := &models.Config{
		DeviceID: req.DeviceId,
		Version:  req.Version,
		Content:  req.Content,
	}

	created, err := s.configRepo.Create(ctx, config)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to create config")
		return nil, status.Error(codes.Internal, "failed to create config")
	}

	return converter.ConfigToProto(created), nil
}

func (s *ConfigService) GetConfig(ctx context.Context, req *pb.GetConfigRequest) (*pb.ConfigResponse, error) {
	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "config id is required")
	}

	config, err := s.configRepo.GetByID(ctx, req.Id)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get config")
		return nil, status.Error(codes.Internal, "internal error")
	}
	if config == nil {
		return nil, status.Errorf(codes.NotFound, "config with id %d not found", req.Id)
	}

	return converter.ConfigToProto(config), nil
}

func (s *ConfigService) ListConfigs(ctx context.Context, req *pb.ListConfigsRequest) (*pb.ListConfigsResponse, error) {
	if req.DeviceId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "device_id is required")
	}

	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	configs, total, err := s.configRepo.List(ctx, req.DeviceId, page, pageSize)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to list configs")
		return nil, status.Error(codes.Internal, "failed to list configs")
	}

	return &pb.ListConfigsResponse{
		Configs:  converter.ConfigsToProto(configs),
		Total:    int32(total),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}, nil
}

func (s *ConfigService) UpdateConfig(ctx context.Context, req *pb.UpdateConfigRequest) (*pb.ConfigResponse, error) {
	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "config id is required")
	}

	updated, err := s.configRepo.Update(ctx, req.Id, req.Version, req.Content)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to update config")
		return nil, status.Error(codes.Internal, "failed to update config")
	}
	if updated == nil {
		return nil, status.Errorf(codes.NotFound, "config with id %d not found", req.Id)
	}

	return converter.ConfigToProto(updated), nil
}

func (s *ConfigService) DeleteConfig(ctx context.Context, req *pb.DeleteConfigRequest) (*emptypb.Empty, error) {
	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "config id is required")
	}

	err := s.configRepo.Delete(ctx, req.Id)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to delete config")
		return nil, status.Errorf(codes.NotFound, "config with id %d not found", req.Id)
	}

	return &emptypb.Empty{}, nil
}

func (s *ConfigService) ApplyConfig(ctx context.Context, req *pb.ApplyConfigRequest) (*pb.ApplyConfigResponse, error) {
	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "config id is required")
	}

	config, err := s.configRepo.GetByID(ctx, req.Id)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get config")
		return nil, status.Error(codes.Internal, "internal error")
	}
	if config == nil {
		return nil, status.Errorf(codes.NotFound, "config with id %d not found", req.Id)
	}

	appliedAt, err := s.configRepo.Apply(ctx, req.Id)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to apply config")
		return nil, status.Error(codes.Internal, "failed to apply config")
	}

	return &pb.ApplyConfigResponse{
		Success:   true,
		AppliedAt: timestamppb.New(appliedAt),
	}, nil
}
