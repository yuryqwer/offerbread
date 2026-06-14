package pipeline

import (
	"context"
	"sync"
)

// Pipeline 管理所有模块和数据流
type Pipeline struct {
	audioCapturer AudioCapturer
	asrService    ASRService
	extractor     QuestionExtractor
	generator     AnswerGenerator

	// 内部 channel
	audioStream chan []byte
	asrResultCh chan string
	questionCh  chan string
	answerCh    chan string

	// 控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewPipeline(
	audio AudioCapturer,
	asr ASRService,
	extractor QuestionExtractor,
	generator AnswerGenerator,
) *Pipeline {
	return &Pipeline{
		audioCapturer: audio,
		asrService:    asr,
		extractor:     extractor,
		generator:     generator,
	}
}

// Start 启动整个流水线（非阻塞）
func (p *Pipeline) Start() error {
	p.ctx, p.cancel = context.WithCancel(context.Background())

	// 初始化 channel，带缓冲以解耦各模块
	p.audioStream = make(chan []byte, 32)
	p.asrResultCh = make(chan string, 32)
	p.questionCh = make(chan string, 32)
	p.answerCh = make(chan string, 32)

	// 启动各模块 goroutine
	p.wg.Add(4)

	go func() {
		defer p.wg.Done()
		if err := p.audioCapturer.Start(p.ctx, p.audioStream); err != nil {
			// 生产环境应记录错误并优雅处理
			p.cancel()
		}
	}()

	go func() {
		defer p.wg.Done()
		if err := p.asrService.Transcribe(p.ctx, p.audioStream, p.asrResultCh); err != nil {
			p.cancel()
		}
	}()

	go func() {
		defer p.wg.Done()
		if err := p.extractor.Extract(p.ctx, p.asrResultCh, p.questionCh); err != nil {
			p.cancel()
		}
	}()

	go func() {
		defer p.wg.Done()
		if err := p.generator.Generate(p.ctx, p.questionCh, p.answerCh); err != nil {
			p.cancel()
		}
	}()

	return nil
}

// Stop 优雅停止流水线
func (p *Pipeline) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
	// 可选：关闭 channel（但由于已 cancel，没人会写，无需 close）
}
