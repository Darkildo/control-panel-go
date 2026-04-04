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
	"google.golang.org/protobuf/types/known/emptypb"

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

func (s *DeviceService) GetDevice(ctx context.Context, req *pb.GetDeviceRequest) (*pb.DeviceResponse, error) {
	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "device id is required")
	}

	device, err := s.deviceRepo.GetByID(ctx, req.Id)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get device")
		return nil, status.Error(codes.Internal, "internal error")
	}
	if device == nil {
		return nil, status.Errorf(codes.NotFound, "device with id %d not found", req.Id)
	}

	return converter.DeviceToProto(device), nil
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

func (s *DeviceService) UpdateDevice(ctx context.Context, req *pb.UpdateDeviceRequest) (*pb.DeviceResponse, error) {
	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "device id is required")
	}

	if req.Ip != nil {
		if ip := net.ParseIP(*req.Ip); ip == nil {
			return nil, status.Error(codes.InvalidArgument, "invalid IP address")
		}
	}

	updated, err := s.deviceRepo.Update(ctx, req.Id, req.Hostname, req.Ip, req.Location, req.IsActive)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to update device")
		return nil, status.Error(codes.Internal, "failed to update device")
	}
	if updated == nil {
		return nil, status.Errorf(codes.NotFound, "device with id %d not found", req.Id)
	}

	return converter.DeviceToProto(updated), nil
}

func (s *DeviceService) DeleteDevice(ctx context.Context, req *pb.DeleteDeviceRequest) (*emptypb.Empty, error) {
	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "device id is required")
	}

	err := s.deviceRepo.Delete(ctx, req.Id)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to delete device")
		return nil, status.Errorf(codes.NotFound, "device with id %d not found", req.Id)
	}

	return &emptypb.Empty{}, nil
}
