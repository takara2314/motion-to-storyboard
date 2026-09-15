package ffmpeg

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/takara2314/motion-to-storyboard/internal/storyboard"
)

type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, name, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s failed: %w\n%s", name, err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

type Service struct {
	runner CommandRunner
}

func NewService(runner CommandRunner) Service {
	return Service{runner: runner}
}

func (s Service) Probe(ctx context.Context, inputPath string) (storyboard.VideoMetadata, error) {
	output, err := s.runner.Run(ctx, "ffprobe", ProbeArgs(inputPath)...)
	if err != nil {
		return storyboard.VideoMetadata{}, err
	}
	return ParseProbeOutput(output)
}

func (s Service) Render(ctx context.Context, plan storyboard.StoryboardPlan) error {
	_, err := s.runner.Run(ctx, "ffmpeg", RenderArgs(plan)...)
	return err
}

func (s Service) CountFrames(ctx context.Context, plan storyboard.StoryboardPlan) (int, error) {
	output, err := s.runner.Run(ctx, "ffmpeg", CountFramesArgs(plan)...)
	if err != nil {
		return 0, err
	}
	frames := 0
	for _, line := range strings.Split(string(output), "\n") {
		if value, found := strings.CutPrefix(line, "frame="); found {
			frames, err = strconv.Atoi(value)
			if err != nil {
				return 0, fmt.Errorf("parse frame count: %w", err)
			}
		}
	}
	if frames < 1 {
		return 0, fmt.Errorf("input produced no frames")
	}
	return frames, nil
}

func ProbeArgs(inputPath string) []string {
	return []string{"-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height:format=duration", "-of", "json", inputPath}
}

func ParseProbeOutput(output []byte) (storyboard.VideoMetadata, error) {
	var result struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return storyboard.VideoMetadata{}, fmt.Errorf("parse ffprobe output: %w", err)
	}
	if len(result.Streams) == 0 || result.Streams[0].Width < 1 || result.Streams[0].Height < 1 {
		return storyboard.VideoMetadata{}, fmt.Errorf("input has no video stream")
	}
	seconds, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil || seconds <= 0 {
		return storyboard.VideoMetadata{}, fmt.Errorf("ffprobe returned an invalid duration %q", result.Format.Duration)
	}
	return storyboard.VideoMetadata{Duration: time.Duration(seconds * float64(time.Second)), Width: result.Streams[0].Width, Height: result.Streams[0].Height}, nil
}

func RenderArgs(plan storyboard.StoryboardPlan) []string {
	options := plan.Options
	args := []string{"-hide_banner", "-loglevel", "error", "-y", "-ss", storyboard.Seconds(options.Start), "-i", options.InputPath, "-t", storyboard.Seconds(plan.Duration)}
	filter := BuildFilter(plan)
	return append(args, "-vf", filter, "-frames:v", "1", options.OutputPath)
}

func CountFramesArgs(plan storyboard.StoryboardPlan) []string {
	options := plan.Options
	return []string{"-hide_banner", "-loglevel", "error", "-nostats", "-progress", "pipe:1", "-ss", storyboard.Seconds(options.Start), "-i", options.InputPath, "-t", storyboard.Seconds(plan.Duration), "-vf", samplingFilter(options), "-f", "null", "-"}
}

func BuildFilter(plan storyboard.StoryboardPlan) string {
	options := plan.Options
	filters := []string{
		samplingFilter(options),
	}
	if options.Crop.IsComplete() {
		filters = append(filters, fmt.Sprintf("crop=%s:%s:%s:%s", options.Crop.Width, options.Crop.Height, options.Crop.X, options.Crop.Y))
	}
	filters = append(filters,
		fmt.Sprintf("scale=trunc(iw*%s/2)*2:trunc(ih*%s/2)*2", decimal(options.Scale), decimal(options.Scale)),
		`pad=iw+max(40\,iw*0.18):ih+max(30\,ih*0.14):0:max(30\,ih*0.14):white`,
		timestampFilter(),
	)
	filters = append(filters, arrowFilters(plan)...)
	filters = append(filters, fmt.Sprintf("tile=%dx%d:nb_frames=%d:padding=20:margin=20:color=white", options.Columns, plan.Rows, plan.FrameCount))
	return strings.Join(filters, ",")
}

func samplingFilter(options storyboard.Options) string {
	return "setpts=PTS-STARTPTS," + fmt.Sprintf("fps=1/%s", storyboard.Seconds(options.Interval))
}

func timestampFilter() string {
	text := `%{eif\:trunc(t/60)\:d\:2}\:%{eif\:mod(trunc(t)\,60)\:d\:2}.%{eif\:mod(trunc(t*1000)\,1000)\:d\:3}`
	return "drawtext=text='" + text + "':x=0:y=4:fontsize=max(16\\,h*0.08):fontcolor=black"
}

func arrowFilters(plan storyboard.StoryboardPlan) []string {
	// A frame at the end of a row has no visual successor beside it.
	condition := arrowCondition(plan)
	return []string{
		"drawbox=x=iw-max(40\\,iw*0.18)*0.88:y=(ih-4)/2:w=max(40\\,iw*0.18)*0.58:h=4:color=black:t=fill:enable='" + condition + "'",
		"drawtext=text='>':x=w-max(40\\,w*0.18)*0.38:y=(h-text_h)/2:fontsize=max(24\\,h*0.15):fontcolor=black:enable='" + condition + "'",
	}
}

func arrowCondition(plan storyboard.StoryboardPlan) string {
	return fmt.Sprintf("lt(mod(n\\,%d)\\,%d)*lt(n\\,%d)", plan.Options.Columns, plan.Options.Columns-1, plan.FrameCount-1)
}

func decimal(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
