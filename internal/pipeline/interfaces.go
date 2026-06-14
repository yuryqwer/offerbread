package pipeline

import "context"

// AudioCapturer 从系统音频（BlackHole）采集 PCM 数据
type AudioCapturer interface {
	// Start 开始采集，将音频块（16kHz, 16bit, mono PCM）写入 channel
	Start(ctx context.Context, out chan<- []byte) error
}

// ASRService 将音频流转为文本流
type ASRService interface {
	// Transcribe 从 in 读取音频块，将识别结果（完整句子）写入 out
	// 支持流式，当识别出一个完整句子时就发送
	Transcribe(ctx context.Context, in <-chan []byte, out chan<- string) error
}

// QuestionExtractor 从连续的识别文本中提取面试官问题
type QuestionExtractor interface {
	// Extract 从识别文本流中提取完整问题，写入 out
	Extract(ctx context.Context, in <-chan string, out chan<- string) error
}

// AnswerGenerator 根据问题生成面试答案
type AnswerGenerator interface {
	// Generate 从问题流中取出问题，生成答案写入 out
	Generate(ctx context.Context, in <-chan string, out chan<- string) error
}
