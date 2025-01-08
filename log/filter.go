package log

import (
	"github.com/kochabonline/kit/log/level"
)

const fuzzer = "***"

type Filter struct {
	logger Logger
	fuzzer string
	level  level.Level
	key    map[any]struct{}
	value  map[any]struct{}
	filter func(level.Level, ...any) bool
}

type FilterOption func(*Filter)

func WithFilterFuzzer(fuzzer string) FilterOption {
	return func(f *Filter) {
		f.fuzzer = fuzzer
	}
}

func WithFilterLevel(level level.Level) FilterOption {
	return func(f *Filter) {
		f.level = level
	}
}

func WithFilterKey(key ...any) FilterOption {
	return func(f *Filter) {
		for _, v := range key {
			f.key[v] = struct{}{}
		}
	}
}

func WithFilterValue(value ...any) FilterOption {
	return func(f *Filter) {
		for _, v := range value {
			f.value[v] = struct{}{}
		}
	}
}

func WithFilterFunc(filter func(level.Level, ...any) bool) FilterOption {
	return func(f *Filter) {
		f.filter = filter
	}
}

func NewFilter(logger Logger, opts ...FilterOption) Filter {
	f := Filter{
		logger: logger,
		fuzzer: fuzzer,
		key:    make(map[any]struct{}),
		value:  make(map[any]struct{}),
	}

	for _, opt := range opts {
		opt(&f)
	}

	return f
}

func (f *Filter) Log(l level.Level, args ...any) {
	if l < f.level {
		return
	}

	if f.filter != nil && f.filter(l, args...) {
		return
	}

	length := len(args)
	if len(f.key) > 0 || len(f.value) > 0 {
		for i := 0; i < length; i += 1 {
			v := i + 1
			if v >= length {
				continue
			}
			if _, ok := f.key[args[i]]; ok {
				args[v] = f.fuzzer
			}
			if _, ok := f.value[args[i+1]]; ok {
				args[v] = f.fuzzer
			}
		}
	}

	f.logger.Log(l, args...)
}
