package disk

import (
	"fmt"

	"go.opentelemetry.io/otel/trace"
)

type InvalidStoredEntryError struct {
	key   string
	cause error
}

func (i InvalidStoredEntryError) Error() string {
	return fmt.Sprintf("invalid stored entry %q: %s", i.key, i.cause)
}

func recordSkippedEntries(span trace.Span, skipped []error) {
	for _, err := range skipped {
		span.RecordError(err)
	}
}
