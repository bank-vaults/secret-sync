package sync

import "time"

var DefaultSyncPeriod = 1 * time.Minute

type options struct {
	Keys   []string
	Paths  []string
	Period time.Duration
}

type Option func(*options)

// WithKeys defines the keys that should be synchronized.
func WithKeys(keys ...string) Option {
	return func(o *options) { o.Paths = keys }
}

// WithPaths defines the paths that should be synchronized. Manager will first
// fetch the keys from the specified paths using List, and then synchronize
// fetched keys.
func WithPaths(paths ...string) Option {
	return func(o *options) { o.Paths = paths }
}

// WithPeriod defines the period at which the synchronization will be triggered.
// Default to DefaultSyncPeriod.
func WithPeriod(t time.Duration) Option {
	return func(o *options) { o.Period = t }
}

func newOptions(opts ...Option) *options {
	option := &options{
		Period: DefaultSyncPeriod,
	}
	for _, opt := range opts {
		opt(option)
	}
	return option
}
