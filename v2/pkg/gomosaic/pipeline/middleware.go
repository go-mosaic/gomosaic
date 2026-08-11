package pipeline

import (
	"context"
	"log"
	"time"
)

// Middleware представляет промежуточный обработчик, оборачивающий этап генерации.
type Middleware func(next StageFunc) StageFunc

// StageFunc функция обработки этапа.
type StageFunc func(ctx context.Context, data *StageData) (*StageData, error)

// Chain объединяет несколько middleware в цепочку.
func Chain(middlewares ...Middleware) Middleware {
	return func(next StageFunc) StageFunc {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}

		return next
	}
}

func LoggingMiddleware(logger *log.Logger) Middleware {
	return func(next StageFunc) StageFunc {
		return func(ctx context.Context, data *StageData) (*StageData, error) {
			logger.Printf("[gomosaic] начало генерации, типов: %d", len(data.Types))

			start := time.Now()
			result, err := next(ctx, data)
			elapsed := time.Since(start)

			if err != nil {
				logger.Printf("[gomosaic] ошибка генерации: %v (за %s)", err, elapsed)
			} else {
				logger.Printf("[gomosaic] генерация завершена, файлов: %d (за %s)", len(result.Files), elapsed)
			}

			return result, err
		}
	}
}

func RecoveryMiddleware() Middleware {
	return func(next StageFunc) StageFunc {
		return func(ctx context.Context, data *StageData) (result *StageData, err error) {
			defer func() {
				if r := recover(); r != nil {
					err = &PanicError{Value: r}
					result = data
				}
			}()

			return next(ctx, data)
		}
	}
}

type PanicError struct {
	Value any
}

func (e *PanicError) Error() string {
	return "паника в процессе генерации: " + formatValue(e.Value)
}

func formatValue(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case error:
		return val.Error()
	default:
		return "(неизвестная паника)"
	}
}
