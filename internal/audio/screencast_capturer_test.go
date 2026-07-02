package audio

import (
	"context"
	"encoding/binary"
	"os"
	"testing"
	"time"
)

func TestScreencastCapturer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	capturer := NewScreencastCapturer()
	out := make(chan []byte, 10)

	err := capturer.Capture(ctx, out)
	if err != nil {
		t.Fatalf("Capture failed: %v", err)
	}

	file, err := os.Create("test_audio.wav")
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	// 先写入 WAV 文件头（44 字节，占位，等录制完成后更新长度字段）
	sampleRate := 48000
	numChannels := 2
	bitsPerSample := 16
	dataSize := 0

	header := createWAVHeader(sampleRate, numChannels, bitsPerSample, dataSize)
	_, err = file.Write(header)
	if err != nil {
		panic(err)
	}

	// 启动写入 goroutine，累加写入的数据长度
	var totalBytes int64
	go func() {
		for data := range out {
			n, err := file.Write(data)
			if err != nil {
				t.Errorf("Failed to write audio data: %v", err)
				return
			}
			totalBytes += int64(n)
		}
	}()

	// 等待一段时间，确保有数据被采集
	time.Sleep(15 * time.Second)
	cancel() // 停止采集
	close(out)

	// 更新文件头中的 dataSize 和文件总大小
	updateWAVHeader(file, totalBytes)
}

// 创建 WAV 头（44 字节）
func createWAVHeader(sampleRate, numChannels, bitsPerSample int, dataSize int) []byte {
	// 字节数计算
	bytesPerSample := bitsPerSample / 8
	byteRate := sampleRate * numChannels * bytesPerSample
	blockAlign := numChannels * bytesPerSample

	header := make([]byte, 44)
	// ChunkID: "RIFF"
	copy(header[0:4], []byte("RIFF"))
	// ChunkSize: 文件总大小 - 8 (将在之后填充)
	binary.LittleEndian.PutUint32(header[4:8], uint32(36+dataSize))
	// Format: "WAVE"
	copy(header[8:12], []byte("WAVE"))
	// Subchunk1ID: "fmt "
	copy(header[12:16], []byte("fmt "))
	// Subchunk1Size: 16 (PCM)
	binary.LittleEndian.PutUint32(header[16:20], 16)
	// AudioFormat: 1 (PCM)
	binary.LittleEndian.PutUint16(header[20:22], 1)
	// NumChannels
	binary.LittleEndian.PutUint16(header[22:24], uint16(numChannels))
	// SampleRate
	binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRate))
	// ByteRate
	binary.LittleEndian.PutUint32(header[28:32], uint32(byteRate))
	// BlockAlign
	binary.LittleEndian.PutUint16(header[32:34], uint16(blockAlign))
	// BitsPerSample
	binary.LittleEndian.PutUint16(header[34:36], uint16(bitsPerSample))
	// Subchunk2ID: "data"
	copy(header[36:40], []byte("data"))
	// Subchunk2Size: data size
	binary.LittleEndian.PutUint32(header[40:44], uint32(dataSize))
	return header
}

// 更新 WAV 文件头中的长度字段
func updateWAVHeader(file *os.File, dataSize int64) error {
	// 定位到文件开头
	if _, err := file.Seek(0, 0); err != nil {
		return err
	}

	// 读取现有的 44 字节头部
	header := make([]byte, 44)
	if _, err := file.Read(header); err != nil {
		return err
	}

	// 计算新的 chunk size 和 data size
	totalSize := 44 + dataSize
	riffSize := uint32(totalSize - 8)
	binary.LittleEndian.PutUint32(header[4:8], riffSize)
	binary.LittleEndian.PutUint32(header[40:44], uint32(dataSize))

	// 写回文件开头
	if _, err := file.Seek(0, 0); err != nil {
		return err
	}
	if _, err := file.Write(header); err != nil {
		return err
	}

	// 将文件指针移到末尾（以免干扰后续操作）
	file.Seek(0, 2)
	return nil
}
