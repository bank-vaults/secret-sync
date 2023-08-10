package kv

import "context"

type Reader interface {
	Type() string
	Get(ctx context.Context, key string) (interface{}, error)
	List(ctx context.Context, path string) ([]string, error)
}

type Writer interface {
	Type() string
	Set(ctx context.Context, key string, value interface{}) error
}

type Store interface {
	Type() string
	Reader
	Writer
}
