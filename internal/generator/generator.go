package generator

import (
	"context"
)

// AnswerGenerator 根据问题生成面试答案
type AnswerGenerator interface {
	// Generate 从问题流中取出问题，生成答案写入 out
	Generate(ctx context.Context, in <-chan string, out chan<- string) error
}
