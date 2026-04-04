package service

import (
	"context"

	"control-panel-go/internal/auth"
	"control-panel-go/internal/converter"
	"control-panel-go/internal/repository"
	"control-panel-go/pkg/jwt"

	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "control-panel-go/gen/pb"
)

type AuthService struct {
	pb.UnimplementedAuthServiceServer
	userRepo   *repository.UserRepository
	jwtManager *jwt.Manager
	logger     zerolog.Logger
}

func NewAuthService(userRepo *repository.UserRepository, jwtManager *jwt.Manager, logger zerolog.Logger) *AuthService {
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

func (s *AuthService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}

	user, err := s.userRepo.GetByID(ctx, req.Id)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get user")
		return nil, status.Error(codes.Internal, "internal error")
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user with id %d not found", req.Id)
	}

	return converter.UserToProto(user), nil
}

func (s *AuthService) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	users, err := s.userRepo.List(ctx, req.LoginSearch)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to list users")
		return nil, status.Error(codes.Internal, "failed to list users")
	}

	return &pb.ListUsersResponse{
		Users: converter.UsersToProto(users),
	}, nil
}

func (s *AuthService) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UserResponse, error) {
	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}

	var passwordHash *string
	if req.Password != nil {
		if len(*req.Password) < 6 {
			return nil, status.Error(codes.InvalidArgument, "password must be at least 6 characters")
		}
		hash, err := auth.HashPassword(*req.Password)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to hash password")
			return nil, status.Error(codes.Internal, "internal error")
		}
		passwordHash = &hash
	}

	// Check login uniqueness if changing login
	if req.Login != nil {
		existing, err := s.userRepo.GetByLogin(ctx, *req.Login)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to check existing user")
			return nil, status.Error(codes.Internal, "internal error")
		}
		if existing != nil && existing.ID != req.Id {
			return nil, status.Error(codes.AlreadyExists, "user with this login already exists")
		}
	}

	updated, err := s.userRepo.Update(ctx, req.Id, req.Login, passwordHash)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to update user")
		return nil, status.Error(codes.Internal, "failed to update user")
	}
	if updated == nil {
		return nil, status.Errorf(codes.NotFound, "user with id %d not found", req.Id)
	}

	return converter.UserToProto(updated), nil
}

func (s *AuthService) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*emptypb.Empty, error) {
	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}

	err := s.userRepo.Delete(ctx, req.Id)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to delete user")
		return nil, status.Errorf(codes.NotFound, "user with id %d not found", req.Id)
	}

	return &emptypb.Empty{}, nil
}
