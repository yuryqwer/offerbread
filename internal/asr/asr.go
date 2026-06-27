package asr

import "context"

// ASRService 将音频流转为文本流
type ASRService interface {
	// Transcribe 从 in 读取音频块，将识别结果（完整句子）写入 out
	// 支持流式，当识别出一个完整句子时就发送
	Transcribe(ctx context.Context, in <-chan []byte, out chan<- string) error
}
