package ws

type Options struct {
	// 读队列大小
	InChanSize int
	// 写队列大小
	OutChanSize int
}

type Option func(*Options)

// type Option interface {
// 	apply(*Options)
// }

// type OptionFunc func(*Options)

// func (f OptionFunc) apply(opt *Options) {
// 	f(opt)
// }

// 功能选项模式
func newOptions(opts ...Option) (opt Options) {
	opt = Options{
		InChanSize:  defaultInChanSize,
		OutChanSize: defaultOutChanSize,
	}
	for _, o := range opts {
		o(&opt)
	}
	return
}

func WithInChanSize(size int) Option {
	return func(o *Options) {
		o.InChanSize = size
	}
}

func WithOutChanSize(size int) Option {
	return func(o *Options) {
		o.OutChanSize = size
	}
}
