package consumer

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"runtime"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/kds"
)

const (
	maxRetries        = 5
	retryBaseDelay    = 200 * time.Millisecond
	maxJitter         = 200 * time.Millisecond
	failureRetryDelay = 1 * time.Second
	successCycleDelay = 500 * time.Millisecond
)

// ConsumerHandler handles KDS consumer operations
type ConsumerHandler struct {
	kdsService *kds.KDSService
	logger     infrastructure.Logger
}

// NewConsumerHandler creates a new consumer handler
func NewConsumerHandler(
	kdsService *kds.KDSService,
	logger infrastructure.Logger,
) *ConsumerHandler {
	return &ConsumerHandler{
		kdsService: kdsService,
		logger:     logger,
	}
}

// RunConsumerLoop starts the main consumer loop
func (h *ConsumerHandler) RunConsumerLoop(rootCtx context.Context) {
	h.logger.InfoWithContext(rootCtx, "Starting Consumer for all event listening")

	for {
		if rootCtx.Err() != nil {
			h.logger.WarnWithContext(
				rootCtx,
				"All events consumer stopping due to rootCtx cancellation",
			)
			return
		}

		consumerCtx, consumerCancel := context.WithCancel(rootCtx)
		success := h.runConsumerWithRetry(consumerCtx)
		consumerCancel()

		if success {
			time.Sleep(successCycleDelay)
		} else {
			time.Sleep(failureRetryDelay)
		}
	}
}

// Close performs cleanup for the consumer handler
func (h *ConsumerHandler) Close() error {
	h.logger.InfoLog("Consumer handler is closing")
	return nil
}

// runConsumerWithRetry implements retry logic with backoff
func (h *ConsumerHandler) runConsumerWithRetry(consumerCtx context.Context) bool {
	var lastError error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if consumerCtx.Err() != nil {
			h.logger.WarnWithContext(
				consumerCtx,
				"All events consumer stopping due to context cancellation during retry",
			)
			return false
		}

		if attempt > 0 {
			h.logger.InfoWithContext(consumerCtx, "Retrying to restart all events consumer...",
				h.logger.Int("attempt", attempt+1),
				h.logger.Int("max_retries", maxRetries))
			time.Sleep(h.backoffDelay(attempt))
		}

		success, err := h.executeConsumerWithRecovery(consumerCtx, attempt)
		if success {
			return true
		}
		lastError = err
	}

	h.logger.ErrorWithContext(consumerCtx, "All events consumer failed after max retries",
		h.logger.Error("err", lastError),
		h.logger.Int("max_retries", maxRetries))
	return false
}

// executeConsumerWithRecovery executes consumer with panic recovery
func (h *ConsumerHandler) executeConsumerWithRecovery(
	consumerCtx context.Context,
	attempt int,
) (bool, error) {
	var lastError error
	var success bool

	func() {
		defer func() {
			if r := recover(); r != nil {
				const size = 64 << 10
				buf := make([]byte, size)
				buf = buf[:runtime.Stack(buf, false)]

				if err, ok := r.(error); ok {
					lastError = fmt.Errorf("panic recovered: %w\n%s", err, buf)
				} else {
					lastError = fmt.Errorf("panic recovered: %v\n%s", r, buf)
				}

				h.logger.ErrorWithContext(consumerCtx, "All events consumer panicked",
					h.logger.Error("err", lastError))
			}
		}()

		err := h.kdsService.ConsumeAllEvents(consumerCtx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(consumerCtx.Err(), context.Canceled) {
				h.logger.WarnWithContext(
					consumerCtx,
					"All events consumer stopped due to context cancellation during consume",
				)
				return
			}

			if errors.Is(consumerCtx.Err(), context.DeadlineExceeded) {
				lastError = fmt.Errorf("consumer timed out: %w", consumerCtx.Err())
				h.logger.ErrorWithContext(consumerCtx, "All events consumer timed out",
					h.logger.Error("err", lastError),
					h.logger.Int("attempt", attempt+1),
					h.logger.Int("max_retries", maxRetries))
				return
			}

			lastError = err
			h.logger.ErrorWithContext(consumerCtx, "All events consumer failed",
				h.logger.Error("error", err),
				h.logger.Int("attempt", attempt+1),
				h.logger.Int("max_retries", maxRetries))
			return
		}

		h.logger.InfoWithContext(consumerCtx, "All events consumer completed successfully")
		success = true
	}()

	return success, lastError
}

// backoffDelay calculates backoff delay with jitter
func (h *ConsumerHandler) backoffDelay(attempt int) time.Duration {
	backoffDuration := retryBaseDelay * time.Duration(1+attempt/2)
	jitter := time.Duration(rand.Int63n(int64(maxJitter)))
	return backoffDuration + jitter
}
