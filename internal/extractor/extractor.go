package extractor

import (
	"context"
)

// QuestionExtractor 从连续的识别文本中提取面试官问题
type QuestionExtractor interface {
	// Extract 从识别文本流中提取完整问题，写入 out
	Extract(ctx context.Context, in <-chan string, out chan<- string) error
}
