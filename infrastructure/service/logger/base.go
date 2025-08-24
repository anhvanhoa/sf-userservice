package logger

import (
	"auth-service/domain/service/logger"
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/natefinch/lumberjack.v2"
)

type log struct {
	Logger *zap.Logger
}

func NewConfig() *lumberjack.Logger {
	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		os.Mkdir("logs", os.ModePerm)
	}
	date := time.Now().Format("2006-01-02")
	return &lumberjack.Logger{
		Filename:   fmt.Sprintf("logs/%s.log", date), // Tạo file theo ngày
		MaxSize:    10,                               // MB, giới hạn file log
		MaxBackups: 7,                                // Giữ lại 7 file log cũ
		MaxAge:     30,                               // Giữ log trong 30 ngày
		Compress:   true,                             // Nén log cũ
	}
}

// InitLogger thiết lập Logger với Lumberjack và Zap
func InitLogger(config *lumberjack.Logger, logLevel zapcore.Level, logFile bool) logger.Log {
	encoderConfig := zapcore.EncoderConfig{
		LevelKey:         "level",
		MessageKey:       "message",
		CallerKey:        "caller",
		TimeKey:          "time",
		LineEnding:       zapcore.DefaultLineEnding,
		EncodeTime:       zapcore.ISO8601TimeEncoder, // Format thời gian
		EncodeCaller:     zapcore.ShortCallerEncoder, // Hiển thị file.go:line
		EncodeLevel:      zapcore.CapitalLevelEncoder,
		ConsoleSeparator: " | ",
	}

	var coreLogs []zapcore.Core

	// Nếu log ra file thì thêm vào coreLogs
	if logFile {
		configFile := encoderConfig
		fileEncoder := zapcore.NewJSONEncoder(configFile)
		fileWriter := zapcore.AddSync(config)
		coreLogs = append(coreLogs, zapcore.NewCore(fileEncoder, fileWriter, logLevel))
	}

	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // Hiển thị màu
	encoderConfig.StacktraceKey = "stack"
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	consoleWriter := zapcore.Lock(os.Stdout)
	coreLogs = append(coreLogs, zapcore.NewCore(consoleEncoder, consoleWriter, logLevel))

	// Kết hợp nhiều writer
	core := zapcore.NewTee(coreLogs...)

	return &log{
		Logger: zap.New(
			core,
			zap.AddCaller(),
			zap.AddCallerSkip(2),
			zap.AddStacktrace(zapcore.ErrorLevel),
		),
	}
}

func convertToZapFields(fields ...any) []zap.Field {
	var zapFields []zap.Field
	for i := 0; i < len(fields)-1; i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			continue // bỏ qua nếu key không phải string
		}
		value := fields[i+1]
		zapFields = append(zapFields, zap.Any(key, value))
	}
	return zapFields
}

// Các hàm tiện ích
func (l *log) Info(msg string, fields ...any) {
	zapFields := convertToZapFields(fields...)
	if len(zapFields) > 0 {
		zapFields = append(zapFields, zap.String("info", msg))
	}
	l.Logger.Info(msg, zapFields...)
}

func (l *log) Debug(msg string, fields ...any) {
	zapFields := convertToZapFields(fields...)
	if len(zapFields) > 0 {
		zapFields = append(zapFields, zap.String("debug", msg))
	}
	l.Logger.Debug(msg, zapFields...)
}

func (l *log) Warn(msg string, fields ...any) {
	zapFields := convertToZapFields(fields...)
	if len(zapFields) > 0 {
		zapFields = append(zapFields, zap.String("warning", msg))
	}
	l.Logger.Warn(msg, zapFields...)
}

func (l *log) Error(msg string, fields ...any) {
	zapFields := convertToZapFields(fields...)
	if len(zapFields) > 0 {
		zapFields = append(zapFields, zap.String("error", msg))
	}
	l.Logger.Error(msg, zapFields...)
}

func (l *log) Fatal(msg string, fields ...any) {
	zapFields := convertToZapFields(fields...)
	if len(zapFields) > 0 {
		zapFields = append(zapFields, zap.String("fatal", msg))
	}
	l.Logger.Fatal(msg, zapFields...)
}

func (l *log) LogGRPC(ctx context.Context, method string, req any, resp any, err error, duration time.Duration) {
	statusCode := codes.OK
	if err != nil {
		if st, ok := status.FromError(err); ok {
			statusCode = st.Code()
		} else {
			statusCode = codes.Internal
		}
	}

	maskedReq := maskSensitiveData(req)
	maskedResp := maskSensitiveData(resp)

	fields := []zap.Field{
		zap.String("method", method),
		zap.String("status", statusCode.String()),
		zap.Duration("duration", duration),
		zap.Any("request", maskedReq),
		zap.Any("response", maskedResp),
	}

	if deadline, ok := ctx.Deadline(); ok {
		fields = append(fields, zap.Time("deadline", deadline))
	}

	if err != nil {
		fields = append(fields, zap.Error(err))
		l.Logger.Error("gRPC call completed with error", fields...)
	} else {
		l.Logger.Info("gRPC call completed successfully", fields...)
	}
}

func (l *log) LogGRPCRequest(ctx context.Context, method string, req any) {
	maskedReq := maskSensitiveData(req)

	fields := []zap.Field{
		zap.String("method", method),
		zap.Any("request", maskedReq),
	}

	if deadline, ok := ctx.Deadline(); ok {
		fields = append(fields, zap.Time("deadline", deadline))
	}

	l.Logger.Info("gRPC request received", fields...)
}

func (l *log) LogGRPCResponse(ctx context.Context, method string, resp any, err error, duration time.Duration) {
	statusCode := codes.OK
	if err != nil {
		if st, ok := status.FromError(err); ok {
			statusCode = st.Code()
		} else {
			statusCode = codes.Internal
		}
	}

	maskedResp := maskSensitiveData(resp)

	fields := []zap.Field{
		zap.String("method", method),
		zap.String("status", statusCode.String()),
		zap.Duration("duration", duration),
		zap.Any("response", maskedResp),
	}

	if err != nil {
		fields = append(fields, zap.Error(err))
		l.Logger.Error("gRPC response sent with error", fields...)
	} else {
		l.Logger.Info("gRPC response sent successfully", fields...)
	}
}

func maskSensitiveData(data any) any {
	if data == nil {
		return nil
	}
	return "[MASKED]"
}
