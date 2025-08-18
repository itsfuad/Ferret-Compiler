package modules

import (
	"reflect"
	"strings"
	"testing"
)

// Test constants
const (
	testTagV100      = "refs/tags/v1.0.0"
	testTagV150      = "refs/tags/v1.5.0"
	testTagV200      = "refs/tags/v2.0.0"
	testHeadsMain    = "refs/heads/main"
	testHeadsDevelop = "refs/heads/develop"
	testOwner        = "test-owner"
	testRepo         = "test-repo"
	testHashABC      = "abc123def456"
	testHashDEF      = "def456ghi789"
	testHashGHI      = "ghi789jkl012"
	testHashJKL      = "jkl012"
	testHashMNO      = "mno345"
	versionV100      = "v1.0.0"
	versionV150      = "v1.5.0"
	versionV200      = "v2.0.0"
	testDataSuffix   = "data"
	expectedErrMsg   = "input too short"
)

func TestParsePacketLengthValid(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected int
	}{
		{"standard length", "0010", 16},
		{"zero length", "0000", 0},
		{"larger length", "004f", 79},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parsePacketLength([]byte(tc.input + testDataSuffix))
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tc.expected {
				t.Errorf("got %d, want %d", result, tc.expected)
			}
		})
	}
}

func TestParsePacketLengthInvalid(t *testing.T) {
	t.Run("invalid hex", func(t *testing.T) {
		_, err := parsePacketLength([]byte("00xg"))
		if err == nil {
			t.Error("expected error but got none")
		}
	})

	t.Run("too short input returns error", func(t *testing.T) {
		_, err := parsePacketLength([]byte("00"))
		if err == nil {
			t.Error("expected error but got none")
		}
		if !strings.Contains(err.Error(), expectedErrMsg) {
			t.Errorf("expected '%s' error, got: %v", expectedErrMsg, err)
		}
	})
}

func TestParseRefLineValid(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected Ref
	}{
		{"tag ref", testHashABC + " " + testTagV100, Ref{Hash: testHashABC, Name: testTagV100}},
		{"branch ref", testHashDEF + " " + testHeadsMain, Ref{Hash: testHashDEF, Name: testHeadsMain}},
		{"ref with null", testHashABC + " " + testTagV200 + "\x00 extra", Ref{Hash: testHashABC, Name: testTagV200}},
		{"ref with newline", testHashABC + " " + testTagV100 + "\n", Ref{Hash: testHashABC, Name: testTagV100}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, valid := parseRefLine(tc.input)
			if !valid {
				t.Error("expected valid ref but got invalid")
			}
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("got %+v, want %+v", result, tc.expected)
			}
		})
	}
}

func TestParseRefLineInvalid(t *testing.T) {
	testCases := []string{
		"# service=git-upload-pack",
		"",
		"   \t  \n",
		"abc123def456refs/tags/v1.0.0",
		"abc123def456",
	}

	for _, input := range testCases {
		t.Run("invalid: "+input, func(t *testing.T) {
			_, valid := parseRefLine(input)
			if valid {
				t.Error("expected invalid ref but got valid")
			}
		})
	}
}

// Helper function to check if slice contains expected strings
func assertContainsAllTags(t *testing.T, result, expected []string) {
	if len(result) != len(expected) {
		t.Errorf("length got %d, want %d", len(result), len(expected))
		return
	}

	for _, expectedTag := range expected {
		found := false
		for _, resultTag := range result {
			if resultTag == expectedTag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected tag %s not found in result", expectedTag)
		}
	}
}

func TestGetTagsFromRefs(t *testing.T) {
	t.Run("mixed refs", func(t *testing.T) {
		refs := []Ref{
			{Hash: testHashABC, Name: testTagV100},
			{Hash: testHashDEF, Name: testHeadsMain},
			{Hash: testHashGHI, Name: testTagV200},
			{Hash: testHashJKL, Name: testHeadsDevelop},
			{Hash: testHashMNO, Name: testTagV150},
		}
		expected := []string{versionV100, versionV200, versionV150}
		result := GetTagsFromRefs(refs)
		assertContainsAllTags(t, result, expected)
	})

	t.Run("only branches", func(t *testing.T) {
		refs := []Ref{
			{Hash: testHashABC, Name: testHeadsMain},
			{Hash: testHashDEF, Name: testHeadsDevelop},
		}
		result := GetTagsFromRefs(refs)
		if len(result) != 0 {
			t.Errorf("expected empty result, got %v", result)
		}
	})

	t.Run("empty refs", func(t *testing.T) {
		result := GetTagsFromRefs([]Ref{})
		if len(result) != 0 {
			t.Errorf("expected empty result, got %v", result)
		}
	})
}

// Benchmark tests
func BenchmarkParsePacketLength(b *testing.B) {
	input := []byte("0010some test data here")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parsePacketLength(input)
	}
}

func BenchmarkParseRefLine(b *testing.B) {
	input := testHashABC + " " + testTagV100 + "\x00 side-band-64k multi_ack"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseRefLine(input)
	}
}

func BenchmarkGetTagsFromRefs(b *testing.B) {
	refs := []Ref{
		{Hash: testHashABC, Name: testTagV100},
		{Hash: testHashDEF, Name: testHeadsMain},
		{Hash: testHashGHI, Name: testTagV200},
		{Hash: testHashJKL, Name: testTagV150},
		{Hash: testHashMNO, Name: testHeadsDevelop},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetTagsFromRefs(refs)
	}
}
