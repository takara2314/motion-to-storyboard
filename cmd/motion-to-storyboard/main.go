package main

import (
	"context"
	"fmt"
	"os"

	"github.com/takara2314/motion-to-storyboard/internal/cli"
	media "github.com/takara2314/motion-to-storyboard/internal/ffmpeg"
	"github.com/takara2314/motion-to-storyboard/internal/storyboard"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	options, err := cli.Parse(args, os.Stderr)
	if err != nil {
		return err
	}
	service := media.NewService(media.ExecRunner{})
	metadata, err := service.Probe(ctx, options.InputPath)
	if err != nil {
		return err
	}
	plan, err := storyboard.NewPlan(options, metadata)
	if err != nil {
		return err
	}
	plan.FrameCount, err = service.CountFrames(ctx, plan)
	if err != nil {
		return err
	}
	plan.Rows = storyboard.RowCount(plan.FrameCount, options.Columns)
	return service.Render(ctx, plan)
}
