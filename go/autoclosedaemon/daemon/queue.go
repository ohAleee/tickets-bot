package daemon

import (
	"go.uber.org/zap"
	"time"
)

type Queue[T any] struct {
	logger    *zap.Logger
	ratelimit time.Duration
	ch        chan T
	processor func(T) error
}

func NewQueue[T any](logger *zap.Logger, ratelimit time.Duration, processor func(T) error) *Queue[T] {
	return &Queue[T]{
		logger:    logger,
		ratelimit: ratelimit,
		ch:        make(chan T),
		processor: processor,
	}
}

func (q *Queue[T]) Push(el T) {
	q.ch <- el
}

func (q *Queue[T]) Listen() {
	for el := range q.ch {
		if err := q.processor(el); err != nil {
			q.logger.Error("Error thrown by queued task", zap.Error(err), zap.Any("element", el))
		}

		time.Sleep(q.ratelimit)
	}
}
