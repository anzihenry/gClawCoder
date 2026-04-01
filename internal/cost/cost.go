package cost

import (
	"fmt"
	"sync"
)

// ModelPricing 模型定价
type ModelPricing struct {
	Model           string
	InputPrice      float64 // per 1M tokens
	OutputPrice     float64 // per 1M tokens
	CacheWritePrice float64 // per 1M tokens
	CacheReadPrice  float64 // per 1M tokens
}

// PricingData 定价数据
var PricingData = map[string]ModelPricing{
	"claude-sonnet-4-20250514": {
		Model:           "claude-sonnet-4-20250514",
		InputPrice:      3.00,
		OutputPrice:     15.00,
		CacheWritePrice: 3.75,
		CacheReadPrice:  0.30,
	},
	"claude-opus-4-20250514": {
		Model:           "claude-opus-4-20250514",
		InputPrice:      15.00,
		OutputPrice:     75.00,
		CacheWritePrice: 18.75,
		CacheReadPrice:  1.50,
	},
	"claude-haiku-4-5-20251213": {
		Model:           "claude-haiku-4-5-20251213",
		InputPrice:      1.00,
		OutputPrice:     5.00,
		CacheWritePrice: 1.25,
		CacheReadPrice:  0.10,
	},
}

// UsageTracker 使用追踪器
type UsageTracker struct {
	mu               sync.Mutex
	inputTokens      int
	outputTokens     int
	cacheWriteTokens int
	cacheReadTokens  int
	model            string
	totalCost        float64
}

// NewUsageTracker 创建追踪器
func NewUsageTracker(model string) *UsageTracker {
	return &UsageTracker{
		model: model,
	}
}

// AddUsage 添加使用量
func (u *UsageTracker) AddUsage(input, output, cacheWrite, cacheRead int) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.inputTokens += input
	u.outputTokens += output
	u.cacheWriteTokens += cacheWrite
	u.cacheReadTokens += cacheRead

	u.totalCost = u.calculateCost()
}

// calculateCost 计算成本
func (u *UsageTracker) calculateCost() float64 {
	pricing, ok := PricingData[u.model]
	if !ok {
		// 默认使用 sonnet 定价
		pricing = PricingData["claude-sonnet-4-20250514"]
	}

	cost := 0.0
	cost += float64(u.inputTokens) * pricing.InputPrice / 1_000_000
	cost += float64(u.outputTokens) * pricing.OutputPrice / 1_000_000
	cost += float64(u.cacheWriteTokens) * pricing.CacheWritePrice / 1_000_000
	cost += float64(u.cacheReadTokens) * pricing.CacheReadPrice / 1_000_000

	return cost
}

// GetUsage 获取使用量
func (u *UsageTracker) GetUsage() (input, output, cacheWrite, cacheRead, total int) {
	u.mu.Lock()
	defer u.mu.Unlock()

	return u.inputTokens, u.outputTokens, u.cacheWriteTokens, u.cacheReadTokens,
		u.inputTokens + u.outputTokens + u.cacheWriteTokens + u.cacheReadTokens
}

// GetCost 获取成本
func (u *UsageTracker) GetCost() float64 {
	u.mu.Lock()
	defer u.mu.Unlock()

	return u.totalCost
}

// Reset 重置
func (u *UsageTracker) Reset() {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.inputTokens = 0
	u.outputTokens = 0
	u.cacheWriteTokens = 0
	u.cacheReadTokens = 0
	u.totalCost = 0
}

// FormatUsage 格式化使用量显示
func (u *UsageTracker) FormatUsage() string {
	u.mu.Lock()
	defer u.mu.Unlock()

	return fmt.Sprintf("Input: %d | Output: %d | Cache Write: %d | Cache Read: %d | Total: %d tokens",
		u.inputTokens, u.outputTokens, u.cacheWriteTokens, u.cacheReadTokens,
		u.inputTokens+u.outputTokens+u.cacheWriteTokens+u.cacheReadTokens)
}

// FormatCost 格式化成本显示
func (u *UsageTracker) FormatCost() string {
	u.mu.Lock()
	defer u.mu.Unlock()

	return fmt.Sprintf("Estimated cost: $%.4f", u.totalCost)
}

// GetModelPricing 获取模型定价
func GetModelPricing(model string) *ModelPricing {
	if pricing, ok := PricingData[model]; ok {
		return &pricing
	}
	return nil
}

// EstimateCost 估算成本
func EstimateCost(inputTokens, outputTokens int, model string) float64 {
	pricing, ok := PricingData[model]
	if !ok {
		pricing = PricingData["claude-sonnet-4-20250514"]
	}

	cost := 0.0
	cost += float64(inputTokens) * pricing.InputPrice / 1_000_000
	cost += float64(outputTokens) * pricing.OutputPrice / 1_000_000

	return cost
}

// FormatTokens 格式化 token 显示
func FormatTokens(tokens int) string {
	if tokens >= 1_000_000 {
		return fmt.Sprintf("%.2fM", float64(tokens)/1_000_000)
	}
	if tokens >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(tokens)/1_000)
	}
	return fmt.Sprintf("%d", tokens)
}
