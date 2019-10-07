package errors

import (
	"log"
	"os"
	"strings"

	"context"
)

var (
	handler Handler = defaultHandler
)

// Handler is a function do the actual error handling.
type Handler func(ctx context.Context, err interface{})

// Handle use handler to handle non-nil err value. Use SetHandler() to switch
// handler, default handler is a plain log.Print(), if ctx is nil, pass
// context.Background() to error handler.
func Handle(ctx context.Context, err interface{}) {
	log.Print(ForLog(err))

	if ctx == nil {
		ctx = context.Background()
	}
	handler(ctx, err)
}

// SetHandler switch error handler, NOTE: no sync lock to internal handler
// variable, only call SetHandler in application initialization code, to
// prevent data race.
// If h is nil, reset to default handler, this feature only available in test
// mode for unit tests to override error handler.
func SetHandler(h Handler) {
	if h == nil {
		if !inTestMode() {
			log.Panicf("[errors] Handler can not be nil")
		}
		handler = defaultHandler
		return
	}

	handler = h
}

func defaultHandler(ctx context.Context, err interface{}) {
}

func inTestMode() bool {
	return strings.HasSuffix(os.Args[0], ".test")
}
