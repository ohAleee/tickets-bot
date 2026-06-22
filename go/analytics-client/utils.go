package analytics

import "time"

func mapNotNull[T any, U any](v *T, f func(T) U) *U {
	if v == nil {
		return nil
	}

	mapped := f(*v)
	return &mapped
}

func mapNullableSecondsToDuration[T int64 | float64](seconds *T) *time.Duration {
	return mapNotNull(seconds, func(secs T) time.Duration {
		return time.Duration(secs) * time.Second
	})
}

func ptr[T any](v T) *T {
	return &v
}
