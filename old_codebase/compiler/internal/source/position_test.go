package source

import "testing"

func TestPositionAdvance(t *testing.T) {
	tests := []struct {
		name     string
		initial  Position
		toSkip   string
		expected Position
	}{
		{
			name: "advance single character",
			initial: Position{
				Line:   1,
				Column: 1,
				Index:  0,
			},
			toSkip: "a",
			expected: Position{
				Line:   1,
				Column: 2,
				Index:  1,
			},
		},
		{
			name: "advance multiple characters",
			initial: Position{
				Line:   1,
				Column: 1,
				Index:  0,
			},
			toSkip: "abc",
			expected: Position{
				Line:   1,
				Column: 4,
				Index:  3,
			},
		},
		{
			name: "advance with newline",
			initial: Position{
				Line:   1,
				Column: 1,
				Index:  0,
			},
			toSkip: "a\nb",
			expected: Position{
				Line:   2,
				Column: 2,
				Index:  3,
			},
		},
		{
			name: "advance with tab",
			initial: Position{
				Line:   1,
				Column: 1,
				Index:  0,
			},
			toSkip: "a\tb",
			expected: Position{
				Line:   1,
				Column: 6, // 1 + 4 (tab width) + 1
				Index:  3,
			},
		},
		{
			name: "advance with mixed whitespace",
			initial: Position{
				Line:   1,
				Column: 1,
				Index:  0,
			},
			toSkip: "a\n\tb",
			expected: Position{
				Line:   2,
				Column: 5, // 4 (tab width) + 1
				Index:  4,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := tt.initial
			result := pos.Advance(tt.toSkip)

			if result.Line != tt.expected.Line {
				t.Errorf("Line = %v, want %v", result.Line, tt.expected.Line)
			}
			if result.Column != tt.expected.Column {
				t.Errorf("Column = %v, want %v", result.Column, tt.expected.Column)
			}
			if result.Index != tt.expected.Index {
				t.Errorf("Index = %v, want %v", result.Index, tt.expected.Index)
			}
		})
	}
}
