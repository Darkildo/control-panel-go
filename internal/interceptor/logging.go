package interceptor

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func LoggingInterceptor(logger zerolog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		st, _ := status.FromError(err)

		event := logger.Info()
		if err != nil {
			event = logger.Warn().Err(err)
		}

		event.
			Str("method", info.FullMethod).
			Dur("duration", duration).
			Str("code", st.Code().String()).
			Msg("gRPC request")

		return resp, err
	}
}
