package file

import "testing"

func TestTextWindowUsesUnicodeCharacterOffsets(t *testing.T) {
	reader := NewFileReader()
	for _, testCase := range []struct {
		offset int
		length int
		text   string
		more   bool
	}{
		{0, 2, "ä🙂", true},
		{2, 2, "ab", false},
		{4, 2, "", false},
		{100, 2, "", false},
	} {
		text, more, err := reader.TextWindow("ä🙂ab", testCase.offset, testCase.length)
		if err != nil || text != testCase.text || more != testCase.more {
			t.Errorf("%+v: text=%q more=%v err=%v", testCase, text, more, err)
		}
	}
	if _, _, err := reader.TextWindow("abc", -1, 2); err == nil {
		t.Fatal("negative offset accepted")
	}
}
