package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAttrLogger(t *testing.T) {
	var (
		buf        = bytes.NewBuffer(nil)
		logger     = NewLogger(WithWriter(buf))
		attrLogger = NewAttrLogger(logger)
		ctx        = NewAttrContext(context.Background())
	)
	AttrsFromCtx(ctx).PutAttrs(slog.String("MY_ATTR", "TEST"))

	attrLogger.Error(ctx, "MSG", slog.String("CORE_ATTR", "TEST"))

	require.True(t, strings.Contains(buf.String(), "MY_ATTR"))
	require.True(t, strings.Contains(buf.String(), "CORE_ATTR"))
	require.True(t, strings.Contains(buf.String(), "MSG"))
}
