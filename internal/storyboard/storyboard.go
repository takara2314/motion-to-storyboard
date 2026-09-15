package storyboard

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultInterval = 100 * time.Millisecond
	DefaultScale    = 1.0
	DefaultColumns  = 5
)

type Crop struct {
	X, Y, Width, Height string
}

func (c Crop) IsEmpty() bool {
	return c.X == "" && c.Y == "" && c.Width == "" && c.Height == ""
}

func (c Crop) IsComplete() bool {
	return c.X != "" && c.Y != "" && c.Width != "" && c.Height != ""
}

type Options struct {
	InputPath, OutputPath string
	Interval              time.Duration
	Scale                 float64
	Start                 time.Duration
	End                   *time.Duration
	Crop                  Crop
	Columns               int
}

func (o Options) Validate() error {
	switch {
	case o.InputPath == "":
		return fmt.Errorf("input video path is required")
	case o.OutputPath == "":
		return fmt.Errorf("output image path is required")
	case o.Interval <= 0:
		return fmt.Errorf("interval must be greater than zero")
	case o.Scale <= 0:
		return fmt.Errorf("scale must be greater than zero")
	case o.Start < 0:
		return fmt.Errorf("start must not be negative")
	case o.End != nil && *o.End <= o.Start:
		return fmt.Errorf("end must be later than start")
	case !o.Crop.IsEmpty() && !o.Crop.IsComplete():
		return fmt.Errorf("x, y, w, and h must be specified together")
	case o.Columns < 1:
		return fmt.Errorf("columns must be at least 1")
	}
	return nil
}

type VideoMetadata struct {
	Duration time.Duration
	Width    int
	Height   int
}

type StoryboardPlan struct {
	Options    Options
	Duration   time.Duration
	FrameCount int
	Rows       int
}

func NewPlan(options Options, metadata VideoMetadata) (StoryboardPlan, error) {
	if err := options.Validate(); err != nil {
		return StoryboardPlan{}, err
	}
	end := metadata.Duration
	if options.End != nil {
		end = *options.End
	}
	if options.Start >= metadata.Duration {
		return StoryboardPlan{}, fmt.Errorf("start must be earlier than the video duration")
	}
	if end > metadata.Duration {
		return StoryboardPlan{}, fmt.Errorf("end exceeds the video duration")
	}
	duration := end - options.Start
	frames := FrameCount(duration, options.Interval)
	return StoryboardPlan{Options: options, Duration: duration, FrameCount: frames, Rows: RowCount(frames, options.Columns)}, nil
}

func FrameCount(duration, interval time.Duration) int {
	return int(math.Ceil(float64(duration) / float64(interval)))
}

func RowCount(frames, columns int) int {
	return (frames + columns - 1) / columns
}

func HasFollowingFrame(frame, frameCount int) bool {
	return frame+1 < frameCount
}

func ParseTimestamp(value string) (time.Duration, error) {
	if value == "" {
		return 0, nil
	}
	if !strings.Contains(value, ":") {
		d, err := time.ParseDuration(value)
		if err != nil {
			return 0, fmt.Errorf("invalid time %q", value)
		}
		return d, nil
	}
	parts := strings.Split(value, ":")
	if len(parts) != 2 && len(parts) != 3 {
		return 0, fmt.Errorf("invalid time %q", value)
	}
	seconds, err := strconv.ParseFloat(parts[len(parts)-1], 64)
	if err != nil || seconds < 0 || seconds >= 60 {
		return 0, fmt.Errorf("invalid time %q", value)
	}
	minutes, err := strconv.Atoi(parts[len(parts)-2])
	if err != nil || minutes < 0 || (len(parts) == 3 && minutes >= 60) {
		return 0, fmt.Errorf("invalid time %q", value)
	}
	hours := 0
	if len(parts) == 3 {
		hours, err = strconv.Atoi(parts[0])
		if err != nil || hours < 0 {
			return 0, fmt.Errorf("invalid time %q", value)
		}
	}
	return time.Duration((float64(hours*3600+minutes*60) + seconds) * float64(time.Second)), nil
}

func Seconds(duration time.Duration) string {
	return strconv.FormatFloat(duration.Seconds(), 'f', 3, 64)
}
