package source

import "testing"

func TestLocationContains(t *testing.T) {
	tests := []struct {
		name     string
		location *Location
		pos      *Position
		want     bool
	}{
		{
			name: "position at start",
			location: NewLocation(
				&Position{Line: 1, Column: 1},
				&Position{Line: 1, Column: 10},
			),
			pos:  &Position{Line: 1, Column: 1},
			want: true,
		},
		{
			name: "position at end",
			location: NewLocation(
				&Position{Line: 1, Column: 1},
				&Position{Line: 1, Column: 10},
			),
			pos:  &Position{Line: 1, Column: 10},
			want: true,
		},
		{
			name: "position in middle",
			location: NewLocation(
				&Position{Line: 1, Column: 1},
				&Position{Line: 1, Column: 10},
			),
			pos:  &Position{Line: 1, Column: 5},
			want: true,
		},
		{
			name: "position before start",
			location: NewLocation(
				&Position{Line: 1, Column: 5},
				&Position{Line: 1, Column: 10},
			),
			pos:  &Position{Line: 1, Column: 1},
			want: false,
		},
		{
			name: "position after end",
			location: NewLocation(
				&Position{Line: 1, Column: 1},
				&Position{Line: 1, Column: 5},
			),
			pos:  &Position{Line: 1, Column: 10},
			want: false,
		},
		{
			name: "position on different line",
			location: NewLocation(
				&Position{Line: 1, Column: 1},
				&Position{Line: 1, Column: 10},
			),
			pos:  &Position{Line: 2, Column: 1},
			want: false,
		},
		{
			name: "multi-line location",
			location: NewLocation(
				&Position{Line: 1, Column: 1},
				&Position{Line: 3, Column: 10},
			),
			pos:  &Position{Line: 2, Column: 5},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.location.Contains(tt.pos); got != tt.want {
				t.Errorf("Location.Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewLocation(t *testing.T) {
	start := &Position{Line: 1, Column: 1}
	end := &Position{Line: 1, Column: 10}

	loc := NewLocation(start, end)

	if loc.Start != start {
		t.Errorf("NewLocation().Start = %v, want %v", loc.Start, start)
	}
	if loc.End != end {
		t.Errorf("NewLocation().End = %v, want %v", loc.End, end)
	}
}
