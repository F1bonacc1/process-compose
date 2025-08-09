package app

import (
	"bufio"
	"io"
	"strings"
	"testing"
)

// Original function using ReadString
func handleOutputOriginal(pipe io.ReadCloser, handler func(message string), done chan struct{}) {
	reader := bufio.NewReader(pipe)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			break
		}
		handler(strings.TrimSuffix(line, "\n"))
	}
	close(done)
}

// New function using ReadByte
func handleOutputNew(pipe io.ReadCloser, handler func(message string), done chan struct{}) {
	reader := bufio.NewReader(pipe)
	var buffer strings.Builder

	for {
		b, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				// Handle any remaining content in buffer
				if buffer.Len() > 0 {
					handler(buffer.String())
				}
				break
			}
			break
		}

		if b == '\n' || b == '\r' {
			line := buffer.String()
			if line != "" || b == '\n' {
				handler(line)
			}
			buffer.Reset()
		} else {
			buffer.WriteByte(b)
		}
	}
	close(done)
}

// Test data generators
func generateRegularLines(count int, lineLength int) string {
	var builder strings.Builder
	line := strings.Repeat("a", lineLength)
	for i := 0; i < count; i++ {
		builder.WriteString(line)
		builder.WriteByte('\n')
	}
	return builder.String()
}

func generateProgressBarOutput(count int) string {
	var builder strings.Builder
	for i := 0; i < count; i++ {
		progress := strings.Repeat("=", i%50)
		spaces := strings.Repeat(" ", 50-(i%50))
		builder.WriteString("Progress: [" + progress + spaces + "] " + string(rune('0'+(i%10))))
		if i < count-1 {
			builder.WriteByte('\r') // Carriage return for progress bars
		} else {
			builder.WriteByte('\n') // Final newline
		}
	}
	return builder.String()
}

func generateMixedOutput(lines int) string {
	var builder strings.Builder
	for i := 0; i < lines; i++ {
		if i%10 == 0 {
			// Every 10th line is a progress update with \r
			progress := strings.Repeat("=", (i/10)%20)
			spaces := strings.Repeat(" ", 20-((i/10)%20))
			builder.WriteString("Loading [" + progress + spaces + "]")
			builder.WriteByte('\r')
		} else {
			// Regular log lines with \n
			builder.WriteString("Log line " + string(rune('0'+(i%10))) + " with some content")
			builder.WriteByte('\n')
		}
	}
	return builder.String()
}

// Benchmark helper
func benchmarkHandler(b *testing.B, data string, handlerFunc func(io.ReadCloser, func(string), chan struct{})) {
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(data)
		pipe := io.NopCloser(reader)
		done := make(chan struct{})
		messageCount := 0

		handler := func(message string) {
			messageCount++
		}

		handlerFunc(pipe, handler, done)
		<-done
	}
}

// Benchmarks for regular lines (newline terminated)
func BenchmarkOriginal_RegularLines_Short(b *testing.B) {
	data := generateRegularLines(100, 50)
	benchmarkHandler(b, data, handleOutputOriginal)
}

func BenchmarkNew_RegularLines_Short(b *testing.B) {
	data := generateRegularLines(100, 50)
	benchmarkHandler(b, data, handleOutputNew)
}

func BenchmarkOriginal_RegularLines_Long(b *testing.B) {
	data := generateRegularLines(100, 500)
	benchmarkHandler(b, data, handleOutputOriginal)
}

func BenchmarkNew_RegularLines_Long(b *testing.B) {
	data := generateRegularLines(100, 500)
	benchmarkHandler(b, data, handleOutputNew)
}

// Benchmarks for progress bar output (carriage return terminated)
func BenchmarkOriginal_ProgressBar(b *testing.B) {
	data := generateProgressBarOutput(100)
	benchmarkHandler(b, data, handleOutputOriginal)
}

func BenchmarkNew_ProgressBar(b *testing.B) {
	data := generateProgressBarOutput(100)
	benchmarkHandler(b, data, handleOutputNew)
}

// Benchmarks for mixed output
func BenchmarkOriginal_Mixed(b *testing.B) {
	data := generateMixedOutput(1000)
	benchmarkHandler(b, data, handleOutputOriginal)
}

func BenchmarkNew_Mixed(b *testing.B) {
	data := generateMixedOutput(1000)
	benchmarkHandler(b, data, handleOutputNew)
}

// Benchmarks for high volume
func BenchmarkOriginal_HighVolume(b *testing.B) {
	data := generateRegularLines(10000, 100)
	benchmarkHandler(b, data, handleOutputOriginal)
}

func BenchmarkNew_HighVolume(b *testing.B) {
	data := generateRegularLines(10000, 100)
	benchmarkHandler(b, data, handleOutputNew)
}

// Benchmarks for very short lines (common in progress updates)
func BenchmarkOriginal_ShortLines(b *testing.B) {
	data := generateRegularLines(1000, 10)
	benchmarkHandler(b, data, handleOutputOriginal)
}

func BenchmarkNew_ShortLines(b *testing.B) {
	data := generateRegularLines(1000, 10)
	benchmarkHandler(b, data, handleOutputNew)
}

// Memory allocation benchmarks
func BenchmarkOriginal_Memory(b *testing.B) {
	data := generateRegularLines(1000, 100)
	b.ResetTimer()
	b.ReportAllocs()
	benchmarkHandler(b, data, handleOutputOriginal)
}

func BenchmarkNew_Memory(b *testing.B) {
	data := generateRegularLines(1000, 100)
	b.ResetTimer()
	b.ReportAllocs()
	benchmarkHandler(b, data, handleOutputNew)
}

// Run benchmarks with: go test -bench=. -benchmem
//
// Example usage:
// go test -bench=BenchmarkOriginal -benchmem
// go test -bench=BenchmarkNew -benchmem
// go test -bench=. -benchmem
