package ffmpeg

import (
	"context"
	"image"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/takara2314/motion-to-storyboard/internal/storyboard"
)

type fakeRunner struct {
	output []byte
	name   string
	args   []string
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.name, f.args = name, args
	return f.output, nil
}

func TestProbeUsesCommandBoundary(t *testing.T) {
	runner := &fakeRunner{output: []byte(`{"streams":[{"width":640,"height":360}],"format":{"duration":"1.500000"}}`)}
	metadata, err := NewService(runner).Probe(context.Background(), "video.mp4")
	if err != nil {
		t.Fatal(err)
	}
	if runner.name != "ffprobe" || metadata.Width != 640 || metadata.Duration != 1500*time.Millisecond {
		t.Fatalf("unexpected probe result: command=%q metadata=%+v", runner.name, metadata)
	}
}

func TestBuildFilterIncludesCropAndLayout(t *testing.T) {
	options := storyboard.Options{InputPath: "in.mp4", OutputPath: "out.png", Interval: 100 * time.Millisecond, Scale: 0.4, Columns: 5, Crop: storyboard.Crop{X: "10", Y: "20", Width: "iw/2", Height: "ih/2"}}
	filter := BuildFilter(storyboard.StoryboardPlan{Options: options, FrameCount: 6, Rows: 2})
	for _, want := range []string{"crop=iw/2:ih/2:10:20", "tile=5x2:nb_frames=6", "drawbox=", "drawtext=text='>'"} {
		if !strings.Contains(filter, want) {
			t.Errorf("filter %q does not contain %q", filter, want)
		}
	}
}

func TestCountFramesUsesProgressOutput(t *testing.T) {
	runner := &fakeRunner{output: []byte("frame=2\nprogress=continue\nframe=4\nprogress=end\n")}
	plan := storyboard.StoryboardPlan{Options: storyboard.Options{InputPath: "in.mp4", Interval: 100 * time.Millisecond}}
	frames, err := NewService(runner).CountFrames(context.Background(), plan)
	if err != nil || frames != 4 {
		t.Fatalf("CountFrames = %d, %v; want 4, nil", frames, err)
	}
	if runner.name != "ffmpeg" || !strings.Contains(strings.Join(runner.args, " "), "-progress pipe:1") {
		t.Fatalf("unexpected command: %s %v", runner.name, runner.args)
	}
}

func TestRenderIntegration(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed")
	}
	temp := t.TempDir()
	input, output := filepath.Join(temp, "input.mp4"), filepath.Join(temp, "storyboard.png")
	create := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "testsrc2=size=160x90:rate=30:duration=0.45", "-pix_fmt", "yuv420p", input)
	if data, err := create.CombinedOutput(); err != nil {
		t.Fatalf("create test video: %v\n%s", err, data)
	}
	service := NewService(ExecRunner{})
	metadata, err := service.Probe(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	options := storyboard.Options{InputPath: input, OutputPath: output, Interval: 100 * time.Millisecond, Scale: 0.4, Columns: 3}
	plan, err := storyboard.NewPlan(options, metadata)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Render(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(output); err != nil || info.Size() == 0 {
		t.Fatalf("output image was not created: %v", err)
	}
	assertImageSize(t, output, 392, 192)

	croppedOutput := filepath.Join(temp, "cropped.png")
	end := 400 * time.Millisecond
	croppedOptions := storyboard.Options{
		InputPath: input, OutputPath: croppedOutput, Interval: 100 * time.Millisecond,
		Scale: 0.4, Start: 100 * time.Millisecond, End: &end, Columns: 2,
		Crop: storyboard.Crop{X: "0", Y: "0", Width: "80", Height: "40"},
	}
	croppedPlan, err := storyboard.NewPlan(croppedOptions, metadata)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Render(context.Background(), croppedPlan); err != nil {
		t.Fatal(err)
	}
	assertImageSize(t, croppedOutput, 204, 152)
}

func assertImageSize(t *testing.T, path string, width, height int) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	config, _, err := image.DecodeConfig(file)
	if err != nil {
		t.Fatal(err)
	}
	if config.Width != width || config.Height != height {
		t.Fatalf("image size = %dx%d; want %dx%d", config.Width, config.Height, width, height)
	}
}
