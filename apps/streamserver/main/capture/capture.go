package capture

import (
	"context"

	"github.com/tinyzimmer/go-gst/gst"
)

type MediaChannel struct {
	ID      string
	Channel chan *gst.Buffer
	Context context.Context
	Cancel  context.CancelFunc
	IsDone  bool
	Writing bool
}
