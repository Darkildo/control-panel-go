package converter

import (
	pb "control-panel-go/gen/pb"
	"control-panel-go/internal/models"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func DeviceToProto(d *models.Device) *pb.DeviceResponse {
	return &pb.DeviceResponse{
		Id:        d.ID,
		Hostname:  d.Hostname,
		Ip:        d.IP,
		Location:  d.Location,
		IsActive:  d.IsActive,
		CreatedAt: timestamppb.New(d.CreatedAt),
	}
}

func DevicesToProto(devices []*models.Device) []*pb.DeviceResponse {
	result := make([]*pb.DeviceResponse, 0, len(devices))
	for _, d := range devices {
		result = append(result, DeviceToProto(d))
	}
	return result
}

func ConfigToProto(c *models.Config) *pb.ConfigResponse {
	resp := &pb.ConfigResponse{
		Id:        c.ID,
		DeviceId:  c.DeviceID,
		Version:   c.Version,
		Content:   c.Content,
		CreatedAt: timestamppb.New(c.CreatedAt),
	}
	if c.AppliedAt.Valid {
		resp.AppliedAt = timestamppb.New(c.AppliedAt.Time)
	}
	return resp
}

func ConfigsToProto(configs []*models.Config) []*pb.ConfigResponse {
	result := make([]*pb.ConfigResponse, 0, len(configs))
	for _, c := range configs {
		result = append(result, ConfigToProto(c))
	}
	return result
}
