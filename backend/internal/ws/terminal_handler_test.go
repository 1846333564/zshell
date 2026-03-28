package ws

import "testing"

func TestUTF8StreamDecoderHandlesSplitChinese(t *testing.T) {
	decoder := &utf8StreamDecoder{}
	data := []byte("你好abc")

	chunk1 := data[:4] // includes one full rune + partial second rune
	chunk2 := data[4:]

	out1 := decoder.Decode(chunk1)
	if out1 != "你" {
		t.Fatalf("unexpected first output: %q", out1)
	}

	out2 := decoder.Decode(chunk2)
	if out2 != "好abc" {
		t.Fatalf("unexpected second output: %q", out2)
	}
}

func TestSplitValidUTF8ReturnsTailForIncompleteRune(t *testing.T) {
	data := []byte("你好")
	partial := data[:5]

	valid, tail := splitValidUTF8(partial)
	if string(valid) != "你" {
		t.Fatalf("unexpected valid segment: %q", string(valid))
	}
	if len(tail) != 2 {
		t.Fatalf("unexpected tail length: %d", len(tail))
	}
}

func TestUTF8StreamDecoderFlushesPendingBytes(t *testing.T) {
	decoder := &utf8StreamDecoder{}
	_ = decoder.Decode([]byte{0xE4, 0xBD}) // incomplete rune

	flushed := decoder.Flush()
	if flushed == "" {
		t.Fatal("expected flush output for pending bytes")
	}
}
