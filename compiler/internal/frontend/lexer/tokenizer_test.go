package lexer

import (
	"compiler/internal/testUtils"
	"testing"
)

func TestNumberTokenization(t *testing.T) {
	tests := []struct {
		input string
		desc  string
	}{
		{"1234", "Integer"},
		{"-1234", "Negative integer"},
		{"1234.567", "Float"},
		{"-1234.567", "Negative float"},
		{"1_234.567_89", "Float with underscores"},
		{"1_234", "Integer with underscores"},
		{"0xDEAD_BEEF", "Hex with underscores"},
		{"0o1_234_567", "Octal with underscores"},
		{"0b1010_1010", "Binary with underscores"},
		{"1_234.567_89e-10", "Scientific notation with underscores"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			filePath := testUtils.CreateTestFileWithContent(t, tt.input)
			tokens := Tokenize(filePath, false)

			if len(tokens) < 1 {
				t.Errorf("%s: got no tokens", tt.desc)
				return
			}

			if tokens[0].Kind != NUMBER_TOKEN {
				t.Errorf("%s: expected NUMBER_TOKEN, got %v", tt.desc, tokens[0].Kind)
			}
		})
	}
}
