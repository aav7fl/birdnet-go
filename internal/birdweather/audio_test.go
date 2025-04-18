package birdweather

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

func TestEncodePCMtoWAV_EmptyInput(t *testing.T) {
	// Test with empty PCM data
	emptyData := []byte{}
	_, err := encodePCMtoWAV(emptyData)

	if err == nil {
		t.Error("encodePCMtoWAV should return an error with empty PCM data")
	}

	if err != nil && err.Error() != "pcmData is empty" {
		t.Errorf("Expected error message 'pcmData is empty', got: %v", err)
	}
}

func TestEncodePCMtoWAV_ValidInput(t *testing.T) {
	// Create valid PCM data (simple sine wave or just zeros)
	sampleCount := 48000                   // 1 second of audio at 48kHz
	pcmData := make([]byte, sampleCount*2) // 16-bit samples = 2 bytes per sample

	// Fill with some non-zero values (could be a simple pattern)
	for i := 0; i < sampleCount; i++ {
		// Write a simple sawtooth pattern
		value := uint16(i % 32768) // 16-bit value range
		binary.LittleEndian.PutUint16(pcmData[i*2:], value)
	}

	// Encode to WAV
	wavBuffer, err := encodePCMtoWAV(pcmData)

	// Check for errors
	if err != nil {
		t.Errorf("encodePCMtoWAV failed with valid input: %v", err)
		return
	}

	// Basic validation of WAV header
	if wavBuffer == nil {
		t.Fatal("encodePCMtoWAV returned nil buffer")
	}

	// Extract header components
	header := make([]byte, 44) // Standard WAV header size
	_, err = io.ReadFull(wavBuffer, header)
	if err != nil {
		t.Fatalf("Failed to read WAV header: %v", err)
	}

	// Reset buffer position
	wavBuffer.Reset()
	io.ReadFull(wavBuffer, header)

	// Check RIFF header
	if string(header[0:4]) != "RIFF" {
		t.Errorf("WAV header missing RIFF marker, got: %s", string(header[0:4]))
	}

	// Check WAVE format
	if string(header[8:12]) != "WAVE" {
		t.Errorf("WAV header missing WAVE format, got: %s", string(header[8:12]))
	}

	// Check fmt chunk
	if string(header[12:16]) != "fmt " {
		t.Errorf("WAV header missing fmt chunk, got: %s", string(header[12:16]))
	}

	// Check PCM format (should be 1)
	format := binary.LittleEndian.Uint16(header[20:22])
	if format != 1 {
		t.Errorf("WAV format should be 1 (PCM), got: %d", format)
	}

	// Check channels (should be 1 - mono)
	channels := binary.LittleEndian.Uint16(header[22:24])
	if channels != 1 {
		t.Errorf("WAV channels should be 1 (mono), got: %d", channels)
	}

	// Check sample rate (should be 48000)
	sampleRate := binary.LittleEndian.Uint32(header[24:28])
	if sampleRate != 48000 {
		t.Errorf("WAV sample rate should be 48000, got: %d", sampleRate)
	}

	// Check bit depth (should be 16)
	bitDepth := binary.LittleEndian.Uint16(header[34:36])
	if bitDepth != 16 {
		t.Errorf("WAV bit depth should be 16, got: %d", bitDepth)
	}

	// Check data chunk
	if string(header[36:40]) != "data" {
		t.Errorf("WAV header missing data chunk, got: %s", string(header[36:40]))
	}

	// Check data size (should match input size)
	dataSize := binary.LittleEndian.Uint32(header[40:44])
	if int(dataSize) != len(pcmData) {
		t.Errorf("WAV data size should be %d, got: %d", len(pcmData), dataSize)
	}
}

func TestEncodePCMtoWAV_SmallInput(t *testing.T) {
	// Test with very small PCM data (smaller than WAV header)
	smallData := []byte{0x01, 0x02, 0x03, 0x04} // Just 4 bytes

	wavBuffer, err := encodePCMtoWAV(smallData)

	if err != nil {
		t.Errorf("encodePCMtoWAV failed with small input: %v", err)
		return
	}

	// The WAV file should still be valid, just with a very small data chunk
	wavData, err := io.ReadAll(wavBuffer)
	if err != nil {
		t.Fatalf("Failed to read WAV data: %v", err)
	}

	// Expected size: 44 byte header + 4 bytes of data
	expectedSize := 44 + 4
	if len(wavData) != expectedSize {
		t.Errorf("Expected WAV file size to be %d bytes, got %d bytes", expectedSize, len(wavData))
	}
}

func TestEncodePCMtoWAV_RecreateOriginalPCM(t *testing.T) {
	// Create test PCM data with a known pattern
	sampleCount := 1000
	pcmData := make([]byte, sampleCount*2)

	// Fill with an easily recognizable pattern
	for i := 0; i < sampleCount; i++ {
		value := uint16(i % 256)
		binary.LittleEndian.PutUint16(pcmData[i*2:], value)
	}

	// Encode to WAV
	wavBuffer, err := encodePCMtoWAV(pcmData)
	if err != nil {
		t.Fatalf("encodePCMtoWAV failed: %v", err)
	}

	// Read the WAV file data
	wavData, err := io.ReadAll(wavBuffer)
	if err != nil {
		t.Fatalf("Failed to read WAV data: %v", err)
	}

	// Extract just the PCM portion (skip 44 byte header)
	extractedPCM := wavData[44:]

	// Verify the extracted PCM matches the original
	if !bytes.Equal(extractedPCM, pcmData) {
		t.Error("Extracted PCM data does not match the original PCM data")

		// Find the first mismatch for better diagnostics
		for i := 0; i < len(pcmData) && i < len(extractedPCM); i++ {
			if pcmData[i] != extractedPCM[i] {
				t.Errorf("First mismatch at byte %d: original=0x%02x, extracted=0x%02x",
					i, pcmData[i], extractedPCM[i])
				break
			}
		}
	}
}

func TestEncodePCMtoWAV_LargeInput(t *testing.T) {
	// Test with a larger PCM data (simulate 5 seconds of audio)
	sampleRate := 48000
	seconds := 5
	sampleCount := sampleRate * seconds
	largeData := make([]byte, sampleCount*2) // 16-bit samples

	// Fill with some pattern
	for i := 0; i < sampleCount; i++ {
		value := uint16(i % 32768)
		binary.LittleEndian.PutUint16(largeData[i*2:], value)
	}

	wavBuffer, err := encodePCMtoWAV(largeData)
	if err != nil {
		t.Errorf("encodePCMtoWAV failed with large input: %v", err)
		return
	}

	// Check that the returned buffer size is correct (header + data)
	wavData, err := io.ReadAll(wavBuffer)
	if err != nil {
		t.Fatalf("Failed to read WAV data: %v", err)
	}

	expectedSize := 44 + len(largeData) // 44 byte header + PCM data
	if len(wavData) != expectedSize {
		t.Errorf("Expected WAV size to be %d bytes, got %d bytes", expectedSize, len(wavData))
	}
}
