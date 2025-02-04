package interceptor

import (
	"context"
	"log/slog"
	"os"
	"runtime/debug"
	"snitch/internal/shared/ctxutil"
	"snitch/internal/shared/trace"

	"connectrpc.com/connect"
	"github.com/google/uuid"
)

func NewTraceInterceptor() connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			traceID, err := uuid.Parse(req.Header().Get("X-Trace-ID"))
			if err != nil {
				traceID = uuid.New()
			}

			reqID, err := uuid.Parse(req.Header().Get("X-Request-ID"))
			if err != nil {
				reqID = uuid.New()
			}

			trace := trace.Trace{TraceID: traceID, RequestID: reqID}
			ctx = ctxutil.WithValue(ctx, trace)

			req.Header().Set("X-Trace-ID", trace.TraceID.String())
			req.Header().Set("X-Request-ID", trace.RequestID.String())

			return next(ctx, req)
		})
	}
	return connect.UnaryInterceptorFunc(interceptor)
}

func NewLogInterceptor() connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			var slogger = slog.New(slog.NewTextHandler(os.Stdout, nil)).With(
				slog.String("HTTP Method", req.HTTPMethod()),
				slog.Any("Procedure", req.Spec().Procedure),
			)

			trace, ok := ctxutil.Value[trace.Trace](ctx)
			if ok {
				slogger = slogger.With(
					slog.Any("TraceID", trace.TraceID),
					slog.Any("RequestID", trace.RequestID),
				)
			}

			ctx = ctxutil.WithValue(ctx, slogger)

			return next(ctx, req)
		})
	}
	return connect.UnaryInterceptorFunc(interceptor)
}

func NewRecoveryInterceptor() connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			slogger, ok := ctxutil.Value[*slog.Logger](ctx)
			if !ok {
				slogger = slog.Default()
			}

			defer func() {
				if err := recover(); err != nil {
					stack := debug.Stack()
					slogger.Error("Panic", "Error", err, "Stack", stack)
				}
			}()
			return next(ctx, req)
		})
	}
	return connect.UnaryInterceptorFunc(interceptor)
}
