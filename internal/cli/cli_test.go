package cli

import (
	"io"
	"testing"
	"time"

	"github.com/takara2314/motion-to-storyboard/internal/storyboard"
)

func TestParseDefaults(t *testing.T) {
	options, err := Parse([]string{"input.mp4", "output.png"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if options.Interval != storyboard.DefaultInterval || options.Scale != 1.0 || options.Columns != storyboard.DefaultColumns {
		t.Fatalf("unexpected defaults: %+v", options)
	}
}

func TestParseTimeRangeAndCrop(t *testing.T) {
	options, err := Parse([]string{"-start", "00:00.100", "-end", "00:00.400", "-x", "0", "-y", "0", "-w", "80", "-h", "40", "input.mp4", "output.png"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if options.Start != 100*time.Millisecond || options.End == nil || *options.End != 400*time.Millisecond || !options.Crop.IsComplete() {
		t.Fatalf("unexpected parsed options: %+v", options)
	}
}
