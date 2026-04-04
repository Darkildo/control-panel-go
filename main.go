package main

import (
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"control-panel-go/internal/database"
	"control-panel-go/internal/interceptor"
	"control-panel-go/internal/repository"
	"control-panel-go/internal/service"
	"control-panel-go/pkg/jwt"

	pb "control-panel-go/gen/pb"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	grpcPort := getEnv("GRPC_PORT", ":50051")
	grpcWebPort := getEnv("GRPC_WEB_PORT", ":8080")
	dbPath := getEnv("DB_PATH", "control_panel.db")
	jwtSecret := getEnv("JWT_SECRET", "pupa-i-lupa")
	jwtDuration := 24 * time.Hour

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	db, err := database.New(dbPath, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to init database")
	}
	defer db.Close()

	jwtManager := jwt.NewManager(jwtSecret, jwtDuration)

	userRepo := repository.NewUserRepository(db)
	deviceRepo := repository.NewDeviceRepository(db)
	configRepo := repository.NewConfigRepository(db)

	authSvc := service.NewAuthService(userRepo, jwtManager, logger)
	deviceSvc := service.NewDeviceService(deviceRepo, logger)
	configSvc := service.NewConfigService(configRepo, deviceRepo, logger)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptor.LoggingInterceptor(logger),
			interceptor.AuthInterceptor(jwtManager, logger),
		),
	)

	pb.RegisterAuthServiceServer(grpcServer, authSvc)
	pb.RegisterDeviceServiceServer(grpcServer, deviceSvc)
	pb.RegisterConfigServiceServer(grpcServer, configSvc)

	reflection.Register(grpcServer) // grpcurl/etc

	listener, err := net.Listen("tcp", grpcPort)
	if err != nil {
		logger.Fatal().Str("port", grpcPort).Err(err).Msg("failed to listen")
	}

	go func() {
		logger.Info().Str("port", grpcPort).Msg("gRPC server starting")
		if err := grpcServer.Serve(listener); err != nil {
			logger.Fatal().Err(err).Msg("gRPC server failed")
		}
	}()

	var httpServer *http.Server
	if grpcWebPort != "" && grpcWebPort != "off" {
		wrappedGrpc := grpcweb.WrapServer(grpcServer,
			grpcweb.WithOriginFunc(func(origin string) bool {
				return true
			}),
			grpcweb.WithAllowedRequestHeaders([]string{"*"}),
		)

		httpServer = &http.Server{
			Addr: grpcWebPort,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if wrappedGrpc.IsGrpcWebRequest(r) || wrappedGrpc.IsAcceptableGrpcCorsRequest(r) {
					wrappedGrpc.ServeHTTP(w, r)
					return
				}
				http.NotFound(w, r)
			}),
		}

		go func() {
			logger.Info().Str("port", grpcWebPort).Msg("gRPC-Web server starting")
			if err = httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Fatal().Err(err).Msg("gRPC-Web server failed")
			}
		}()
	} else {
		logger.Info().Msg("gRPC-Web server disabled (GRPC_WEB_PORT=off)")
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("shutting down servers...")
	grpcServer.GracefulStop()
	if httpServer != nil {
		err = httpServer.Close()
		if err != nil {
			logger.Error().Err(err).Msg("failed to close http server(gracefully)")
		}
	}
	logger.Info().Msg("servers stopped")
}
