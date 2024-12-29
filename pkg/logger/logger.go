package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"slices"
)

func JSONFactory(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	return slog.NewJSONHandler(w, opts)
}

func WithHandlerFactory(handlerFactory func(w io.Writer, opts *slog.HandlerOptions) slog.Handler) OptionFunc {
	return func(o *options) {
		o.handlerFactory = handlerFactory
	}
}
func WithWriter(w io.Writer) OptionFunc {
	return func(o *options) {
		o.writer = w
	}
}

func WithAddSource(addSource bool) OptionFunc {
	return func(o *options) {
		o.handlerOptions.AddSource = addSource
	}
}
func WithLevel(l slog.Leveler) OptionFunc {
	return func(o *options) {
		o.handlerOptions.Level = l
	}
}
func WithReplaceAttr(replaceAttr func(groups []string, a slog.Attr) slog.Attr) OptionFunc {
	return func(o *options) {
		o.handlerOptions.ReplaceAttr = replaceAttr
	}
}

type OptionFunc func(*options)

type options struct {
	writer         io.Writer
	handlerOptions *slog.HandlerOptions
	handlerFactory func(w io.Writer, opts *slog.HandlerOptions) slog.Handler
}

func defaultOptions() *options {
	return &options{
		writer:         os.Stdout,
		handlerOptions: new(slog.HandlerOptions),
		handlerFactory: JSONFactory,
	}
}

func NewLogger(options ...OptionFunc) *slog.Logger {
	opts := defaultOptions()
	for _, opt := range options {
		opt(opts)
	}
	handler := opts.handlerFactory(opts.writer, opts.handlerOptions)

	return slog.New(handler)
}

type Attrs struct {
	attrs []any
}

type attrsKey struct{}

func AttrsFromCtx(ctx context.Context) *Attrs {
	v := ctx.Value(attrsKey{})
	if v == nil {
		return &Attrs{}
	}

	return v.(*Attrs)
}

func NewAttrContext(ctx context.Context) context.Context {
	return new(Attrs).ToCtx(ctx)
}

func (l *Attrs) Copy() *Attrs {
	return &Attrs{
		attrs: slices.Clone(l.attrs),
	}
}
func (l *Attrs) reset() {
	l.attrs = l.attrs[:0]
}
func (l *Attrs) ToCtx(ctx context.Context) context.Context {
	return context.WithValue(ctx, attrsKey{}, l)
}

func (l *Attrs) PutAttrs(attrs ...any) {
	l.attrs = append(l.attrs, attrs...)
}

func NewAttrLogger(logger *slog.Logger) *Logger {
	return &Logger{
		logger: logger,
	}
}

type Logger struct {
	logger *slog.Logger
}

func (l *Logger) Error(ctx context.Context, msg string, attrs ...any) {
	attributes := AttrsFromCtx(ctx)
	l.logger.ErrorContext(ctx, msg, append(attributes.attrs, attrs...)...)
	attributes.reset()
}

func (l *Logger) Info(ctx context.Context, msg string, attrs ...any) {
	attributes := AttrsFromCtx(ctx)
	l.logger.InfoContext(ctx, msg, append(attributes.attrs, attrs...)...)
	attributes.reset()
}

func (l *Logger) Panic(ctx context.Context, msg string, attrs ...any) {
	attributes := AttrsFromCtx(ctx)
	attributes.reset()

	attrs = append(attributes.attrs, attrs)
	l.logger.ErrorContext(ctx, msg, attrs...)

	panic(msg)
}
