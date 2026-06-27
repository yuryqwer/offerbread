package audio

import (
	"context"
)

// AudioCapturer 从系统音频采集 PCM 数据
type AudioCapturer interface {
	// Capture 开始采集，将音频块（16kHz, 16bit, mono PCM）写入 channel
	Capture(ctx context.Context, out chan<- []byte) error
}
