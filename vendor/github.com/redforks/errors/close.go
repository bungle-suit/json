package errors

import (
	"context"
	"io"
)

// Close close an IO closer, handle by errors.Handle() on failed.
//
// It is very often to forget to close a closer, such as file, net connection,
// and very annoying to handle this kind normally won't failed error.
func Close(c io.Closer) {
	err := c.Close()
	if err != nil {
		Handle(context.Background(), err)
	}
}
