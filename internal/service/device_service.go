package service

import (
	"context"
	"net"

	"control-panel-go/internal/converter"
	"control-panel-go/internal/models"
	"control-panel-go/internal/repository"

	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "control-panel-go/gen/pb"
)

type DeviceService struct {
	pb.UnimplementedDeviceServiceServer
	deviceRepo *repository.DeviceRepository
	logger     zerolog.Logger
}

func NewDeviceService(deviceRepo *repository.DeviceRepository, logger zerolog.Logger) *DeviceService {
	return &DeviceService{
		deviceRepo: deviceRepo,
		logger:     logger,
	}
}

func (s *DeviceService) CreateDevice(ctx context.Context, req *pb.CreateDeviceRequest) (*pb.DeviceResponse, error) {
	if req.Hostname == "" {
		return nil, status.Error(codes.InvalidArgument, "hostname is required")
	}
	if req.Ip == "" {
		return nil, status.Error(codes.InvalidArgument, "ip is required")
	}
	if ip := net.ParseIP(req.Ip); ip == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid IP address")
	}

	device := &models.Device{
		Hostname: req.Hostname,
		IP:       req.Ip,
		Location: req.Location,
		IsActive: req.IsActive,
	}

	created, err := s.deviceRepo.Create(ctx, device)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to create device")
		return nil, status.Error(codes.Internal, "failed to create device")
	}

	return converter.DeviceToProto(created), nil
}

func (s *DeviceService) ListDevices(ctx context.Context, req *pb.ListDevicesRequest) (*pb.ListDevicesResponse, error) {
	var isActive *bool
	if req.IsActive != nil {
		v := *req.IsActive
		isActive = &v
	}

	devices, err := s.deviceRepo.List(ctx, isActive, req.HostnameSearch)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to list devices")
		return nil, status.Error(codes.Internal, "failed to list devices")
	}

	return &pb.ListDevicesResponse{
		Devices: converter.DevicesToProto(devices),
	}, nil
}
