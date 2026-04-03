package service

import (
	"context"

	"control-panel-go/internal/auth"
	"control-panel-go/internal/repository"

	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "control-panel-go/gen/pb"
)

type AuthService struct {
	pb.UnimplementedAuthServiceServer
	userRepo   *repository.UserRepository
	jwtManager *auth.JWTManager
	logger     zerolog.Logger
}

func NewAuthService(userRepo *repository.UserRepository, jwtManager *auth.JWTManager, logger zerolog.Logger) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
		logger:     logger,
	}
}

func (s *AuthService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.AuthResponse, error) {
	if req.Login == "" {
		return nil, status.Error(codes.InvalidArgument, "login is required")
	}
	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}
	if len(req.Password) < 6 {
		return nil, status.Error(codes.InvalidArgument, "password must be at least 6 characters")
	}

	existing, err := s.userRepo.GetByLogin(ctx, req.Login)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to check existing user")
		return nil, status.Error(codes.Internal, "internal error")
	}
	if existing != nil {
		return nil, status.Error(codes.AlreadyExists, "user with this login already exists")
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to hash password")
		return nil, status.Error(codes.Internal, "internal error")
	}

	user, err := s.userRepo.Create(ctx, req.Login, hash)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to create user")
		return nil, status.Error(codes.Internal, "internal error")
	}

	token, err := s.jwtManager.Generate(user.ID, user.Login)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to generate token")
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.AuthResponse{Token: token}, nil
}

func (s *AuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.AuthResponse, error) {
	if req.Login == "" {
		return nil, status.Error(codes.InvalidArgument, "login is required")
	}
	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	user, err := s.userRepo.GetByLogin(ctx, req.Login)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get user")
		return nil, status.Error(codes.Internal, "internal error")
	}
	if user == nil {
		return nil, status.Error(codes.Unauthenticated, "invalid login or password")
	}

	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		return nil, status.Error(codes.Unauthenticated, "invalid login or password")
	}

	token, err := s.jwtManager.Generate(user.ID, user.Login)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to generate token")
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.AuthResponse{Token: token}, nil
}
