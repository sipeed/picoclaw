// Package bus 是 PicoClaw 的消息总线，负责连接 Agent 循环（AgentLoop）与渠道管理器（Manager）。
//
// 消息总线维护三条带缓冲的 Go channel：
//   - inbound：用户发送的消息，流向 Agent 循环进行处理。
//   - outbound：Agent 生成的文本回复，流向渠道管理器再发送给用户。
//   - outboundMedia：Agent 生成的媒体消息（图片、文件等），流向渠道管理器。
//
// 消息流向：渠道 → inbound → AgentLoop → outbound/outboundMedia → Manager → 渠道
//
// 内部机制：
//   - 所有 channel 使用 defaultBusBufferSize 大小的缓冲区，削峰填谷。
//   - publish 使用 context 感知的三步安全检查，避免向已关闭 channel 发送数据。
//   - Close 方法按照 安全关闭顺序（close done → set closed → wait wg → close channels → drain）
//     实现优雅关闭，确保不丢失已在缓冲区中的消息。
package bus

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// ErrBusClosed 当向已关闭的 MessageBus 发布消息时返回此错误。
var ErrBusClosed = errors.New("message bus closed")

// defaultBusBufferSize 是每条消息通道的默认缓冲区大小，用于削峰填谷。
const defaultBusBufferSize = 64

// StreamDelegate 由渠道管理器（Manager）实现，为 Agent 循环提供流式输出能力。
// 通过该接口将流式处理逻辑与 Agent 循环解耦，避免直接依赖具体渠道实现。
type StreamDelegate interface {
	// GetStreamer 根据渠道名称和聊天 ID 返回对应的 Streamer 实例。
	// 如果该渠道不支持流式输出，返回 nil, false。
	GetStreamer(ctx context.Context, channel, chatID string) (Streamer, bool)
}

// Streamer 负责将增量内容推送到支持流式输出的渠道。
// 定义在此包中，使 Agent 循环可以使用流式输出而无需导入 pkg/channels。
type Streamer interface {
	// Update 推送增量内容片段（流式中间结果）。
	Update(ctx context.Context, content string) error
	// Finalize 推送最终完整内容，标识流式输出结束。
	Finalize(ctx context.Context, content string) error
	// Cancel 取消当前流式输出会话。
	Cancel(ctx context.Context)
}

// MessageBus 是消息总线的核心结构，管理三条消息通道及生命周期控制。
type MessageBus struct {
	// inbound 接收用户发来的消息，由渠道侧写入，Agent 循环侧读取。
	inbound chan InboundMessage
	// outbound 承载 Agent 生成的文本回复，由 Agent 循环写入，渠道管理器读取。
	outbound chan OutboundMessage
	// outboundMedia 承载 Agent 生成的媒体消息（图片、文件等）。
	outboundMedia chan OutboundMediaMessage

	// closeOnce 确保 Close 操作只执行一次。
	closeOnce sync.Once
	// done 关闭后通知所有阻塞中的发布者退出。
	done chan struct{}
	// closed 原子布尔标记，快速判断总线是否已关闭。
	closed atomic.Bool
	// wg 跟踪正在进行中的 publish 调用，关闭时等待它们完成。
	wg sync.WaitGroup
	// streamDelegate 以原子方式存储 StreamDelegate 实现（通常为渠道 Manager）。
	streamDelegate atomic.Value
}

// NewMessageBus 创建并返回一个新的 MessageBus 实例，
// 初始化三条带缓冲的消息通道和关闭信号通道。
func NewMessageBus() *MessageBus {
	return &MessageBus{
		inbound:       make(chan InboundMessage, defaultBusBufferSize),
		outbound:      make(chan OutboundMessage, defaultBusBufferSize),
		outboundMedia: make(chan OutboundMediaMessage, defaultBusBufferSize),
		done:          make(chan struct{}),
	}
}

// publish 是泛型消息发布函数，所有 PublishXxx 方法均委托给它。
// 执行三步安全检查以避免向已关闭 channel 发送数据：
//  1. 通过 atomic.Bool 快速检查 closed 标记，避免不必要的 wg.Add 和潜在死锁。
//  2. 通过 select 非阻塞检查 done 和 ctx.Done()，在真正发送前再次确认状态。
//  3. 在 wg 保护下执行实际的 channel 发送，确保 Close 会等待发送完成。
func publish[T any](ctx context.Context, mb *MessageBus, ch chan T, msg T) error {
	// 第一步：快速检查总线是否已关闭，避免不必要的 wg.Add 及潜在死锁
	if mb.closed.Load() {
		return ErrBusClosed
	}

	// 第二步：在发送消息前再次检查，避免向已关闭 channel 发送数据
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-mb.done:
		return ErrBusClosed
	default:
	}

	// 第三步：在 WaitGroup 保护下执行发送，Close 方法会等待所有进行中的发送完成
	mb.wg.Add(1)
	defer mb.wg.Done()

	select {
	case ch <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-mb.done:
		return ErrBusClosed
	}
}

// PublishInbound 将用户消息发布到 inbound 通道，供 Agent 循环消费。
func (mb *MessageBus) PublishInbound(ctx context.Context, msg InboundMessage) error {
	return publish(ctx, mb, mb.inbound, msg)
}

// InboundChan 返回 inbound 通道的只读端，供 Agent 循环接收用户消息。
func (mb *MessageBus) InboundChan() <-chan InboundMessage {
	return mb.inbound
}

// PublishOutbound 将 Agent 生成的文本回复发布到 outbound 通道，供渠道管理器消费。
func (mb *MessageBus) PublishOutbound(ctx context.Context, msg OutboundMessage) error {
	return publish(ctx, mb, mb.outbound, msg)
}

// OutboundChan 返回 outbound 通道的只读端，供渠道管理器接收文本回复。
func (mb *MessageBus) OutboundChan() <-chan OutboundMessage {
	return mb.outbound
}

// PublishOutboundMedia 将 Agent 生成的媒体消息发布到 outboundMedia 通道。
func (mb *MessageBus) PublishOutboundMedia(ctx context.Context, msg OutboundMediaMessage) error {
	return publish(ctx, mb, mb.outboundMedia, msg)
}

// OutboundMediaChan 返回 outboundMedia 通道的只读端，供渠道管理器接收媒体消息。
func (mb *MessageBus) OutboundMediaChan() <-chan OutboundMediaMessage {
	return mb.outboundMedia
}

// SetStreamDelegate 注册流式代理（通常为渠道管理器 Manager），
// 使 Agent 循环可以通过消息总线获取 Streamer。
func (mb *MessageBus) SetStreamDelegate(d StreamDelegate) {
	mb.streamDelegate.Store(d)
}

// GetStreamer 通过已注册的 StreamDelegate 获取指定渠道和聊天 ID 的 Streamer。
// 如果未注册代理或该渠道不支持流式输出，返回 nil, false。
func (mb *MessageBus) GetStreamer(ctx context.Context, channel, chatID string) (Streamer, bool) {
	if d, ok := mb.streamDelegate.Load().(StreamDelegate); ok && d != nil {
		return d.GetStreamer(ctx, channel, chatID)
	}
	return nil, false
}

// Close 优雅关闭消息总线，确保不丢失已缓冲的消息。
// 关闭顺序：
//  1. 关闭 done 通道 → 通知所有阻塞中的发布者退出。
//  2. 设置 closed 标记 → 阻止新的发布者进入。
//  3. 等待 wg → 确保所有进行中的 publish 调用完成。
//  4. 关闭三条消息通道 → 释放资源。
//  5. 排空（drain）通道中残留的消息 → 防止消息丢失。
func (mb *MessageBus) Close() {
	mb.closeOnce.Do(func() {
		// 第一步：关闭 done 通道，通知所有阻塞在 select 上的发布者退出
		close(mb.done)

		// 第二步：设置 closed 原子标记，确保新的发布者在 wg.Add 之前就能感知关闭状态
		mb.closed.Store(true)

		// 第三步：等待所有正在进行中的 publish 调用完成，确保消息已写入通道或已退出
		mb.wg.Wait()

		// 第四步：安全关闭三条消息通道
		close(mb.inbound)
		close(mb.outbound)
		close(mb.outboundMedia)

		// 第五步：排空通道中残留的消息，防止消息丢失
		drained := 0
		for range mb.inbound {
			drained++
		}
		for range mb.outbound {
			drained++
		}
		for range mb.outboundMedia {
			drained++
		}

		if drained > 0 {
			logger.DebugCF("bus", "Drained buffered messages during close", map[string]any{
				"count": drained,
			})
		}
	})
}
