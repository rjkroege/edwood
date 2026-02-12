package file

import (
	"testing"
)

func TestBufferWriter(t *testing.T) {
	// Create a buffer to use as our underlying storage
	buf := NewBufferNoNr(nil)

	// Create the WriterAtWriter
	writer := buf.NewWriter(buf.End(), 0)

	// Test 1: Write some data
	data1 := []byte("Hello, ")
	n, err := writer.Write(data1)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data1) {
		t.Errorf("Write returned %d bytes, expected %d", n, len(data1))
	}

	// Verify position
	if pos := buf.End(); pos.B != len(data1) {
		t.Errorf("Position is %d, expected %d", pos, len(data1))
	}

	// Test 2: Write more data
	data2 := []byte("World!")
	n, err = writer.Write(data2)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data2) {
		t.Errorf("Write returned %d bytes, expected %d", n, len(data2))
	}

	// Verify final position
	expectedPos := len(data1) + len(data2)
	if pos := buf.End(); pos.B != expectedPos {
		t.Errorf("Position is %d, expected %d", pos, expectedPos)
	}

	// Verify the complete data
	expected := "Hello, World!"
	if result := buf.String(); result != expected {
		t.Errorf("Buffer contains '%s', expected '%s'", result, expected)
	}

	writer = buf.NewWriter(buf.ByteTuple(len(data1)), 0)
	data3 := []byte("日本語 ")
	n, err = writer.Write(data3)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data3) {
		t.Errorf("Write returned %d bytes, expected %d", n, len(data3))
	}

	// Verify the complete data again.
	expected = "Hello, 日本語 World!"
	if result := buf.String(); result != expected {
		t.Errorf("Buffer contains '%s', expected '%s'", result, expected)
	}
}

func TestBufferWriterNulls(t *testing.T) {
	// Create a buffer to use as our underlying storage
	buf := NewBufferNoNr(nil)

	// Create the WriterAtWriter
	writer := buf.NewWriter(buf.End(), 0)

	// Test 1: Write some data
	data1 := []byte("Hel\000lo")
	n, err := writer.Write(data1)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data1)-1 {
		t.Errorf("Write returned %d bytes, expected %d", n, len(data1)-1)
	}

	// Verify the complete data
	expected := "Hello"
	if result := buf.String(); result != expected {
		t.Errorf("Buffer contains '%s', expected '%s'", result, expected)
	}

	if writer.HadNull() != true {
		t.Errorf("data1 had nulls but hasnull not set")
	}
}
