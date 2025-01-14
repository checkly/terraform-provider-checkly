package provider

type Options struct {
	UseCredentialsFromEnvironment bool
}

func DefaultOptions() *Options {
	return &Options{
		UseCredentialsFromEnvironment: true,
	}
}

type Option interface {
	Apply(opts *Options)
}

type OptionFunc func(opts *Options)

func (f OptionFunc) Apply(opts *Options) {
	f(opts)
}

func WithUseCredentialsFromEnvironment(enabled bool) Option {
	return OptionFunc(func(opts *Options) {
		opts.UseCredentialsFromEnvironment = enabled
	})
}
