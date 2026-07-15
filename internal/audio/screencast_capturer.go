package audio

import (
	"context"
	"fmt"
	"sync"

	"go2tv.app/screencast/capture"
)

// ScreencastCapturer 基于 go2tv.app/screencast 的音频采集实现
type ScreencastCapturer struct {
	stream    *capture.Stream
	running   bool
	mu        sync.Mutex
	resampler *PCMResampler // 48kHz stereo → 16kHz mono
}

// NewScreencastCapturer 创建一个新的音频采集器
func NewScreencastCapturer() *ScreencastCapturer {
	return &ScreencastCapturer{
		resampler: NewPCMResampler(),
	}
}

func (s *ScreencastCapturer) Capture(ctx context.Context, out chan<- []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("already capturing") // 已经在采集了
	}

	stream, err := capture.Open(&capture.Options{
		IncludeAudio: true,
	}) // 默认开启音频
	if err != nil {
		return fmt.Errorf("failed to open capture stream: %w", err)
	}

	if stream.Audio == nil {
		stream.Close()
		return fmt.Errorf("audio capture not available (requires macOS 13.0+)")
	}

	s.stream = stream
	s.running = true
	s.resampler.Reset() // 清除上次采集的 FIR 状态

	go func() {
		defer func() {
			s.mu.Lock()
			s.running = false
			s.stream.Close()
			s.mu.Unlock()
		}()
		s.readLoop(ctx, out)
	}()

	return nil
}

// readLoop 从 stream.Audio 读取 48kHz 立体声，重采样为 16kHz 单声道，输出到 out
func (s *ScreencastCapturer) readLoop(ctx context.Context, out chan<- []byte) {
	buf := make([]byte, 4096) // 4KB 的缓冲区
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := s.stream.Audio.Read(buf)
			if err != nil {
				fmt.Printf("audio read error: %v\n", err)
				return
			}
			if n == 0 {
				continue
			}

			resampled := s.resampler.Process(buf[:n])
			if len(resampled) == 0 {
				continue
			}

			select {
			case out <- resampled:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (s *ScreencastCapturer) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}
