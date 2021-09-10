package zapsentry

import (
	"fmt"
	"go.uber.org/zap"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap/zapcore"
)

func NewScope() zapcore.Field {
	return zap.Any("zapsentry_scope", sentry.NewScope())
}

func WithNewScope(s *sentry.Scope) zapcore.Field {
	return zap.Any("zapsentry_scope", s)
}

func NewCore(cfg Configuration, factory SentryClientFactory) (zapcore.Core, error) {
	client, err := factory()
	if err != nil {
		return zapcore.NewNopCore(), err
	}

	core := core{
		client:       client,
		cfg:          &cfg,
		LevelEnabler: cfg.Level,
		flushTimeout: 5 * time.Second,
		fields:       make(map[string]interface{}),
	}

	if cfg.FlushTimeout > 0 {
		core.flushTimeout = cfg.FlushTimeout
	}

	return &core, nil
}

func (c *core) With(fs []zapcore.Field) zapcore.Core {
	return c.with(fs)
}

func (c *core) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.cfg.Level.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *core) Write(ent zapcore.Entry, fs []zapcore.Field) error {
	clone := c.with(fs)

	event := sentry.NewEvent()
	event.Message = ent.Message
	event.Timestamp = ent.Time
	event.Level = sentrySeverity(ent.Level)
	event.Platform = "Golang"
	event.Extra = clone.fields
	event.Tags = c.cfg.Tags

	if !c.cfg.DisableStacktrace {
		trace := sentry.NewStacktrace()
		if trace != nil {
			trace.Frames = filterFrames(trace.Frames)
			event.Exception = []sentry.Exception{{
				Type:       ent.Message,
				Value:      ent.Caller.TrimmedPath(),
				Stacktrace: trace,
			}}
		}
	}

	_ = c.client.CaptureEvent(event, nil, c.scope())

	// We may be crashing the program, so should flush any buffered events.
	if ent.Level > zapcore.ErrorLevel {
		c.client.Flush(c.flushTimeout)
	}
	return nil
}

func (c *core) hub() *sentry.Hub {
	if c.cfg.Hub != nil {
		return c.cfg.Hub
	}
	return sentry.CurrentHub()
}

func (c *core) scope() *sentry.Scope {
	if c.sentryScope != nil {
		return c.sentryScope
	}
	return c.hub().Scope()
}

func findScope(fs []zapcore.Field) *sentry.Scope {
	for _, f := range fs {
		if scope, ok := f.Interface.(*sentry.Scope); ok {
			return scope
		}
	}
	return nil
}

func (c *core) shouldBeEncoded(field zapcore.Field) bool {
	if _, ok := field.Interface.(*sentry.Scope); ok {
		return false
	}

	return true
}

func (c *core) Sync() error {
	c.client.Flush(c.flushTimeout)
	return nil
}

func (c *core) with(fs []zapcore.Field) *core {
	// Copy our map.
	m := make(map[string]interface{}, len(c.fields))
	for k, v := range c.fields {
		m[k] = v
	}

	// Add fields to an in-memory encoder.
	enc := zapcore.NewMapObjectEncoder()
	for _, f := range fs {
		if !c.shouldBeEncoded(f) {
			continue
		}
		f.AddTo(enc)
	}

	// Merge the two maps.
	for k, v := range enc.Fields {
		m[k] = v
	}

	return &core{
		client:       c.client,
		cfg:          c.cfg,
		flushTimeout: c.flushTimeout,
		fields:       m,
		LevelEnabler: c.LevelEnabler,
		sentryScope:  findScope(fs),
	}
}

type ClientGetter interface {
	GetClient() *sentry.Client
}

func (c *core) GetClient() *sentry.Client {
	return c.client
}

type core struct {
	client *sentry.Client
	cfg    *Configuration
	zapcore.LevelEnabler
	flushTimeout time.Duration

	sentryScope *sentry.Scope

	fields map[string]interface{}
}

// follow same logic with sentry-go to filter unnecessary frames
// ref:
// https://github.com/getsentry/sentry-go/blob/362a80dcc41f9ad11c8df556104db3efa27a419e/stacktrace.go#L256-L280
func filterFrames(frames []sentry.Frame) []sentry.Frame {
	if len(frames) == 0 {
		return nil
	}
	filteredFrames := make([]sentry.Frame, 0, len(frames))

	for i := range frames {
		// Skip zapsentry and zap internal frames, except for frames in _test packages (for
		// testing).
		if (strings.HasPrefix(frames[i].Module, "github.com/TheZeroSlave/zapsentry") ||
			strings.HasPrefix(frames[i].Function, "go.uber.org/zap")) &&
			!strings.HasSuffix(frames[i].Module, "_test") {
			break
		}
		filteredFrames = append(filteredFrames, frames[i])
	}
	return filteredFrames
}
