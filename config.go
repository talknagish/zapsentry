package zapsentry

import (
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap/zapcore"
)

// Configuration is a minimal set of parameters for Sentry integration.
type Configuration struct {
	// Tags are passed as is to the corresponding sentry.Event field.
	Tags map[string]string

	// LoggerNameKey is the key for zap logger name.
	// If not empty, the name is added to the rest of zapcore.Field(s),
	// so that be careful with key duplicates.
	// Leave LoggerNameKey empty to disable the feature.
	LoggerNameKey string

	// DisableStacktrace disables adding stacktrace to sentry.Event, if set.
	DisableStacktrace bool

	// Level is the minimal level of sentry.Event(s).
	Level zapcore.Level

	// EnableBreadcrumbs enables use of sentry.Breadcrumb(s).
	// This feature works only when you explicitly passed new scope.
	EnableBreadcrumbs bool

	// BreadcrumbLevel is the minimal level of sentry.Breadcrumb(s).
	// Breadcrumb specifies an application event that occurred before a Sentry event.
	// NewCore fails if BreadcrumbLevel is greater than Level.
	// The field is ignored, if EnableBreadcrumbs is not set.
	BreadcrumbLevel zapcore.Level

	// FlushTimeout is the timeout for flushing events to Sentry.
	FlushTimeout time.Duration

	// Hub overrides the sentry.CurrentHub value.
	// See sentry.Hub docs for more detail.
	Hub         *sentry.Hub
	DynamicTags []string // tags from zap field names
}
