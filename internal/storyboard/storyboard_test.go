package storyboard

import (
	"testing"
	"time"
)

func TestParseTimestamp(t *testing.T) {
	tests := map[string]time.Duration{
		"100ms":        100 * time.Millisecond,
		"0.1s":         100 * time.Millisecond,
		"00:00.100":    100 * time.Millisecond,
		"01:02:03.456": time.Hour + 2*time.Minute + 3456*time.Millisecond,
	}
	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			got, err := ParseTimestamp(input)
			if err != nil || got != want {
				t.Fatalf("ParseTimestamp(%q) = %v, %v; want %v", input, got, err, want)
			}
		})
	}
	for _, input := range []string{"nope", "1:60", "1:2:60"} {
		if _, err := ParseTimestamp(input); err == nil {
			t.Errorf("ParseTimestamp(%q) accepted an invalid value", input)
		}
	}
}

func TestLayoutRules(t *testing.T) {
	if got := FrameCount(401*time.Millisecond, 100*time.Millisecond); got != 5 {
		t.Fatalf("FrameCount = %d; want 5", got)
	}
	if got := RowCount(11, 5); got != 3 {
		t.Fatalf("RowCount = %d; want 3", got)
	}
	if !HasFollowingFrame(3, 5) || HasFollowingFrame(4, 5) {
		t.Fatal("HasFollowingFrame returned the wrong boundary result")
	}
}

func TestOptionsValidation(t *testing.T) {
	valid := Options{InputPath: "in.mp4", OutputPath: "out.png", Interval: DefaultInterval, Scale: DefaultScale, Columns: DefaultColumns}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	partialCrop := valid
	partialCrop.Crop.X = "0"
	if err := partialCrop.Validate(); err == nil {
		t.Fatal("partial crop must be rejected")
	}
	end := time.Second
	invalidRange := valid
	invalidRange.Start, invalidRange.End = time.Second, &end
	if err := invalidRange.Validate(); err == nil {
		t.Fatal("end equal to start must be rejected")
	}
}
