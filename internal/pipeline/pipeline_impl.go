package pipeline

import (
	"context"
	"sync"
)

type InterviewPipeline struct {
	// 模块实例（接口）
	audio     AudioCapturer
	asr       ASRService
	extractor QuestionExtractor
	generator AnswerGenerator

	// 模块直连 channel 以及扇出 channel
	audioRawCh chan []byte // 音频采集产出
	audioCh    chan []byte // ASR 消费音频
	audioUICh  chan []byte // UI 波形消费音频

	textRawCh chan string // ASR 产出文本
	textCh    chan string // 问题提取消费文本
	textUICh  chan string // UI 打字机消费文本

	questionRawCh chan string // 问题提取产出
	questionCh    chan string // 答案生成消费问题
	questionUICh  chan string // UI 显示问题

	answerCh chan string // 答案生成产出

	// 控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// fanOut 将 src 中的值复制到所有的 dsts 中
func fanOut[T any](ctx context.Context, wg *sync.WaitGroup, src <-chan T, dsts ...chan<- T) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case val := <-src:
			for _, dst := range dsts {
				select {
				case dst <- val:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

func NewInterviewPipeline(
	audio AudioCapturer,
	asr ASRService,
	extractor QuestionExtractor,
	generator AnswerGenerator,
) *InterviewPipeline {
	p := &InterviewPipeline{
		audio:     audio,
		asr:       asr,
		extractor: extractor,
		generator: generator,
	}

	p.ctx, p.cancel = context.WithCancel(context.Background())

	// 初始化 channel，带缓冲以解耦各模块
	p.audioRawCh = make(chan []byte, 32)
	p.audioCh = make(chan []byte, 32)
	p.audioUICh = make(chan []byte, 32)

	p.textRawCh = make(chan string, 32)
	p.textCh = make(chan string, 32)
	p.textUICh = make(chan string, 32)

	p.questionRawCh = make(chan string, 32)
	p.questionCh = make(chan string, 32)
	p.questionUICh = make(chan string, 32)

	p.answerCh = make(chan string, 32)

	return p
}

// Start 启动整个流水线（非阻塞）
func (p *InterviewPipeline) Start() error {
	// 音频采集 -> audioRawCh
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		if err := p.audio.Capture(p.ctx, p.audioRawCh); err != nil {
			// 生产环境应记录错误并优雅处理
			p.cancel()
		}
	}()

	// 多播: audioRawCh → audioCh + audioUICh
	p.wg.Add(1)
	go fanOut(p.ctx, &p.wg, p.audioRawCh, p.audioCh, p.audioUICh)

	// ASR: audioCh -> textRawCh
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		if err := p.asr.Transcribe(p.ctx, p.audioCh, p.textRawCh); err != nil {
			p.cancel()
		}
	}()

	// 多播: textRawCh → textCh + textUICh
	p.wg.Add(1)
	go fanOut(p.ctx, &p.wg, p.textRawCh, p.textCh, p.textUICh)

	// 问题提取: textCh -> questionRawCh
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		if err := p.extractor.Extract(p.ctx, p.textCh, p.questionRawCh); err != nil {
			p.cancel()
		}
	}()

	// 多播: questionRawCh → questionCh + questionUICh
	p.wg.Add(1)
	go fanOut(p.ctx, &p.wg, p.questionRawCh, p.questionCh, p.questionUICh)

	// 答案生成: textCh -> questionRawCh
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		if err := p.generator.Generate(p.ctx, p.questionCh, p.answerCh); err != nil {
			p.cancel()
		}
	}()

	return nil
}

// Stop 优雅停止流水线
func (p *InterviewPipeline) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
	// 可选：关闭 channel（但由于已 cancel，没人会写，无需 close）
}
