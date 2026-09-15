package cli

import (
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/takara2314/motion-to-storyboard/internal/storyboard"
)

func Parse(args []string, output io.Writer) (storyboard.Options, error) {
	flags := flag.NewFlagSet("motion-to-storyboard", flag.ContinueOnError)
	flags.SetOutput(output)
	interval := flags.String("interval", "100ms", "frame sampling interval")
	scale := flags.Float64("scale", storyboard.DefaultScale, "frame scale relative to the source")
	start := flags.String("start", "0s", "start time")
	end := flags.String("end", "", "end time")
	x := flags.String("x", "", "crop x expression")
	y := flags.String("y", "", "crop y expression")
	w := flags.String("w", "", "crop width expression")
	h := flags.String("h", "", "crop height expression")
	columns := flags.Int("columns", storyboard.DefaultColumns, "maximum frames per row")
	flags.Usage = func() {
		fmt.Fprintln(output, "Usage: motion-to-storyboard [options] <input-video> <output-image>")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return storyboard.Options{}, err
	}
	if flags.NArg() != 2 {
		flags.Usage()
		return storyboard.Options{}, fmt.Errorf("input video and output image paths are required")
	}
	parsedInterval, err := storyboard.ParseTimestamp(*interval)
	if err != nil {
		return storyboard.Options{}, fmt.Errorf("interval: %w", err)
	}
	parsedStart, err := storyboard.ParseTimestamp(*start)
	if err != nil {
		return storyboard.Options{}, fmt.Errorf("start: %w", err)
	}
	var parsedEnd *time.Duration
	if *end != "" {
		value, parseErr := storyboard.ParseTimestamp(*end)
		if parseErr != nil {
			return storyboard.Options{}, fmt.Errorf("end: %w", parseErr)
		}
		parsedEnd = &value
	}
	options := storyboard.Options{
		InputPath: flags.Arg(0), OutputPath: flags.Arg(1), Interval: parsedInterval,
		Scale: *scale, Start: parsedStart, Crop: storyboard.Crop{X: *x, Y: *y, Width: *w, Height: *h}, Columns: *columns,
	}
	if parsedEnd != nil {
		options.End = parsedEnd
	}
	return options, options.Validate()
}
