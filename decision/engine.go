package decision

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"nofx/market"
	"nofx/mcp"
	"nofx/pool"
	"strings"
	"time"
)

// PositionInfo 持仓信息
type PositionInfo struct {
	Symbol           string  `json:"symbol"`
	Side             string  `json:"side"` // "long" or "short"
	EntryPrice       float64 `json:"entry_price"`
	MarkPrice        float64 `json:"mark_price"`
	Quantity         float64 `json:"quantity"`
	Leverage         int     `json:"leverage"`
	UnrealizedPnL    float64 `json:"unrealized_pnl"`
	UnrealizedPnLPct float64 `json:"unrealized_pnl_pct"`
	LiquidationPrice float64 `json:"liquidation_price"`
	MarginUsed       float64 `json:"margin_used"`
	UpdateTime       int64   `json:"update_time"` // 持仓更新时间戳（毫秒）
}

// AccountInfo 账户信息
type AccountInfo struct {
	TotalEquity      float64 `json:"total_equity"`      // 账户净值
	AvailableBalance float64 `json:"available_balance"` // 可用余额
	TotalPnL         float64 `json:"total_pnl"`         // 总盈亏
	TotalPnLPct      float64 `json:"total_pnl_pct"`     // 总盈亏百分比
	MarginUsed       float64 `json:"margin_used"`       // 已用保证金
	MarginUsedPct    float64 `json:"margin_used_pct"`   // 保证金使用率
	PositionCount    int     `json:"position_count"`    // 持仓数量
}

// CandidateCoin 候选币种（来自币种池）
type CandidateCoin struct {
	Symbol  string   `json:"symbol"`
	Sources []string `json:"sources"` // 来源: "ai500" 和/或 "oi_top"
}

// OITopData 持仓量增长Top数据（用于AI决策参考）
type OITopData struct {
	Rank              int     // OI Top排名
	OIDeltaPercent    float64 // 持仓量变化百分比（1小时）
	OIDeltaValue      float64 // 持仓量变化价值
	PriceDeltaPercent float64 // 价格变化百分比
	NetLong           float64 // 净多仓
	NetShort          float64 // 净空仓
}

// Context 交易上下文（传递给AI的完整信息）
type Context struct {
	CurrentTime     string                  `json:"current_time"`
	RuntimeMinutes  int                     `json:"runtime_minutes"`
	CallCount       int                     `json:"call_count"`
	Account         AccountInfo             `json:"account"`
	Positions       []PositionInfo          `json:"positions"`
	CandidateCoins  []CandidateCoin         `json:"candidate_coins"`
	MarketDataMap   map[string]*market.Data `json:"-"` // 不序列化，但内部使用
	OITopDataMap    map[string]*OITopData   `json:"-"` // OI Top数据映射
	Performance     interface{}             `json:"-"` // 历史表现分析（logger.PerformanceAnalysis）
	BTCETHLeverage  int                     `json:"-"` // BTC/ETH杠杆倍数（从配置读取）
	AltcoinLeverage int                     `json:"-"` // 山寨币杠杆倍数（从配置读取）
}

// Decision AI的交易决策
type Decision struct {
	Symbol          string  `json:"symbol"`
	Action          string  `json:"action"` // "open_long", "open_short", "close_long", "close_short", "hold", "wait"
	Leverage        int     `json:"leverage,omitempty"`
	PositionSizeUSD float64 `json:"position_size_usd,omitempty"`
	StopLoss        float64 `json:"stop_loss,omitempty"`
	TakeProfit      float64 `json:"take_profit,omitempty"`
	Confidence      int     `json:"confidence,omitempty"` // 信心度 (0-100)
	RiskUSD         float64 `json:"risk_usd,omitempty"`   // 最大美元风险
	Reasoning       string  `json:"reasoning"`
	ClosePercentage float64 `json:"close_percentage,omitempty"` // 部分平仓百分比 (0-100)
	NewStopLoss     float64 `json:"new_stop_loss,omitempty"`    // 新止损价
	NewTakeProfit   float64 `json:"new_take_profit,omitempty"`  // 新止盈价
}

// FullDecision AI的完整决策（包含思维链）
type FullDecision struct {
	SystemPrompt string     `json:"system_prompt"` // 系统提示词（发送给AI的系统prompt）
	UserPrompt   string     `json:"user_prompt"`   // 发送给AI的输入prompt
	CoTTrace     string     `json:"cot_trace"`     // 思维链分析（AI输出）
	Decisions    []Decision `json:"decisions"`     // 具体决策列表
	Timestamp    time.Time  `json:"timestamp"`
}

// GetFullDecision 获取AI的完整交易决策（批量分析所有币种和持仓）
func GetFullDecision(ctx *Context, mcpClient *mcp.Client) (*FullDecision, error) {
	return GetFullDecisionWithCustomPrompt(ctx, mcpClient, "", false, "")
}

// GetFullDecisionWithCustomPrompt 获取AI的完整交易决策（支持自定义prompt和模板选择）
func GetFullDecisionWithCustomPrompt(ctx *Context, mcpClient *mcp.Client, customPrompt string, overrideBase bool, templateName string) (*FullDecision, error) {
	// 1. 为所有币种获取市场数据
	if err := fetchMarketDataForContext(ctx); err != nil {
		return nil, fmt.Errorf("获取市场数据失败: %w", err)
	}

	// 2. 构建 System Prompt（固定规则）和 User Prompt（动态数据）
	systemPrompt := buildSystemPromptWithCustom(ctx.Account.TotalEquity, ctx.BTCETHLeverage, ctx.AltcoinLeverage, customPrompt, overrideBase, templateName)
	userPrompt := buildUserPrompt(ctx)

	// 3. 调用AI API（使用 system + user prompt）
	aiResponse, err := mcpClient.CallWithMessages(systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("调用AI API失败: %w", err)
	}

	// 4. 解析AI响应
	decision, err := parseFullDecisionResponse(ctx, aiResponse)
	if err != nil {
		return decision, fmt.Errorf("解析AI响应失败: %w", err)
	}

	decision.Timestamp = time.Now()
	decision.SystemPrompt = systemPrompt // 保存系统prompt
	decision.UserPrompt = userPrompt     // 保存输入prompt
	return decision, nil
}

// fetchMarketDataForContext 为上下文中的所有币种获取市场数据和OI数据
func fetchMarketDataForContext(ctx *Context) error {
	ctx.MarketDataMap = make(map[string]*market.Data)
	ctx.OITopDataMap = make(map[string]*OITopData)

	// 收集所有需要获取数据的币种
	symbolSet := make(map[string]bool)

	// 1. 优先获取持仓币种的数据（这是必须的）
	for _, pos := range ctx.Positions {
		symbolSet[pos.Symbol] = true
	}

	// 2. 候选币种数量根据账户状态动态调整
	maxCandidates := calculateMaxCandidates(ctx)
	candidateSymbols := []string{}
	for i, coin := range ctx.CandidateCoins {
		if i >= maxCandidates {
			break
		}
		symbolSet[coin.Symbol] = true
		candidateSymbols = append(candidateSymbols, coin.Symbol)
	}
	log.Printf("📊 准备获取 %d 个币种的市场数据: %v", len(symbolSet), candidateSymbols)

	// 并发获取市场数据
	// 持仓币种集合（用于判断是否跳过OI检查）
	positionSymbols := make(map[string]bool)
	for _, pos := range ctx.Positions {
		positionSymbols[pos.Symbol] = true
	}

	for symbol := range symbolSet {
		data, err := market.Get(symbol)
		if err != nil {
			// 单个币种失败不影响整体，只记录错误
			log.Printf("⚠️  获取 %s 市场数据失败: %v", symbol, err)
			continue
		}

		// ⚠️ 流动性过滤：持仓价值低于15M USD的币种不做（多空都不做）
		// 持仓价值 = 持仓量 × 当前价格
		// 但现有持仓必须保留（需要决策是否平仓）
		isExistingPosition := positionSymbols[symbol]
		if !isExistingPosition && data.OpenInterest != nil && data.CurrentPrice > 0 {
			// 计算持仓价值（USD）= 持仓量 × 当前价格
			oiValue := data.OpenInterest.Latest * data.CurrentPrice
			oiValueInMillions := oiValue / 1_000_000 // 转换为百万美元单位
			if oiValueInMillions < 15 {
				log.Printf("⚠️  %s 持仓价值过低(%.2fM USD < 15M)，跳过此币种 [持仓量:%.0f × 价格:%.4f]",
					symbol, oiValueInMillions, data.OpenInterest.Latest, data.CurrentPrice)
				continue
			}
		}

		ctx.MarketDataMap[symbol] = data
	}

	// 输出成功获取的市场数据
	successSymbols := []string{}
	for symbol := range ctx.MarketDataMap {
		successSymbols = append(successSymbols, symbol)
	}
	log.Printf("✅ 成功获取 %d 个币种的市场数据: %v", len(ctx.MarketDataMap), successSymbols)

	// 检查是否有 BTC 数据
	if _, hasBTC := ctx.MarketDataMap["BTCUSDT"]; !hasBTC {
		log.Printf("⚠️  警告: 缺少 BTCUSDT 市场数据，AI 可能无法确认市场方向")
	}

	// 加载OI Top数据（不影响主流程）
	oiPositions, err := pool.GetOITopPositions()
	if err == nil {
		for _, pos := range oiPositions {
			// 标准化符号匹配
			symbol := pos.Symbol
			ctx.OITopDataMap[symbol] = &OITopData{
				Rank:              pos.Rank,
				OIDeltaPercent:    pos.OIDeltaPercent,
				OIDeltaValue:      pos.OIDeltaValue,
				PriceDeltaPercent: pos.PriceDeltaPercent,
				NetLong:           pos.NetLong,
				NetShort:          pos.NetShort,
			}
		}
	}

	return nil
}

// calculateMaxCandidates 根据账户状态计算需要分析的候选币种数量
func calculateMaxCandidates(ctx *Context) int {
	// 直接返回候选池的全部币种数量
	// 因为候选池已经在 auto_trader.go 中筛选过了
	// 固定分析前20个评分最高的币种（来自AI500）
	return len(ctx.CandidateCoins)
}

// buildSystemPromptWithCustom 构建包含自定义内容的 System Prompt
func buildSystemPromptWithCustom(accountEquity float64, btcEthLeverage, altcoinLeverage int, customPrompt string, overrideBase bool, templateName string) string {
	// 如果覆盖基础prompt且有自定义prompt，只使用自定义prompt
	if overrideBase && customPrompt != "" {
		return customPrompt
	}

	// 获取基础prompt（使用指定的模板）
	basePrompt := buildSystemPrompt(accountEquity, btcEthLeverage, altcoinLeverage, templateName)

	// 如果没有自定义prompt，直接返回基础prompt
	if customPrompt == "" {
		return basePrompt
	}

	// 添加自定义prompt部分到基础prompt
	var sb strings.Builder
	sb.WriteString(basePrompt)
	sb.WriteString("\n\n")
	sb.WriteString("# 📌 个性化交易策略\n\n")
	sb.WriteString(customPrompt)
	sb.WriteString("\n\n")
	sb.WriteString("注意: 以上个性化策略是对基础规则的补充，不能违背基础风险控制原则。\n")

	return sb.String()
}

// buildSystemPrompt 构建 System Prompt（使用模板+动态部分）
func buildSystemPrompt(accountEquity float64, btcEthLeverage, altcoinLeverage int, templateName string) string {
	var sb strings.Builder

	// 1. 加载提示词模板（核心交易策略部分）
	if templateName == "" {
		templateName = "default" // 默认使用 default 模板
	}

	template, err := GetPromptTemplate(templateName)
	if err != nil {
		// 如果模板不存在，记录错误并使用 default
		log.Printf("⚠️  提示词模板 '%s' 不存在，使用 default: %v", templateName, err)
		template, err = GetPromptTemplate("default")
		if err != nil {
			// 如果连 default 都不存在，使用内置的简化版本
			log.Printf("❌ 无法加载任何提示词模板，使用内置简化版本")
			sb.WriteString("你是专业的加密货币交易AI。请根据市场数据做出交易决策。\n\n")
		} else {
			sb.WriteString(template.Content)
			sb.WriteString("\n\n")
		}
	} else {
		sb.WriteString(template.Content)
		sb.WriteString("\n\n")
	}

	// 2. 硬约束（风险控制）- 动态生成
	sb.WriteString("# 硬约束（风险控制）\n\n")
	sb.WriteString("1. 风险回报比: 必须 ≥ 1:3（冒1%风险，赚3%+收益）\n")
	sb.WriteString("2. 最多持仓: 3个币种（质量>数量）\n")
	sb.WriteString(fmt.Sprintf("3. 单币仓位: 山寨%.0f-%.0f U(%dx杠杆) | BTC/ETH %.0f-%.0f U(%dx杠杆)\n",
		accountEquity*2, accountEquity*5, altcoinLeverage, accountEquity*5, accountEquity*10, btcEthLeverage))
	sb.WriteString("4. 保证金: 总使用率 ≤ 90%\n\n")

	// 3. 输出格式 - 动态生成
	sb.WriteString("#输出格式\n\n")
	sb.WriteString("第一步: 思维链（纯文本）\n")
	sb.WriteString("简洁分析你的思考过程\n\n")
	sb.WriteString("第二步: JSON决策数组\n\n")
	sb.WriteString("```json\n[\n")
	sb.WriteString(fmt.Sprintf("  {\"symbol\": \"BTCUSDT\", \"action\": \"open_short\", \"leverage\": %d, \"position_size_usd\": %.0f, \"stop_loss\": 97000, \"take_profit\": 91000, \"confidence\": 85, \"risk_usd\": 300, \"reasoning\": \"下跌趋势+MACD死叉\"},\n", btcEthLeverage, accountEquity*5))
	sb.WriteString("  {\"symbol\": \"ETHUSDT\", \"action\": \"close_long\", \"reasoning\": \"止盈离场\"}\n")
	sb.WriteString("]\n```\n\n")
	sb.WriteString("字段说明:\n")
	sb.WriteString("- `action`: open_long | open_short | close_long | close_short | hold | wait\n")
	sb.WriteString("- `confidence`: 0-100（开仓建议≥75）\n")
	sb.WriteString("- 开仓时必填: leverage, position_size_usd, stop_loss, take_profit, confidence, risk_usd, reasoning\n\n")

	return sb.String()
}

// buildUserPrompt 构建 User Prompt（动态数据）
func buildUserPrompt(ctx *Context) string {
	var sb strings.Builder

	// 系统状态
	sb.WriteString(fmt.Sprintf("时间: %s | 周期: #%d | 运行: %d分钟\n\n",
		ctx.CurrentTime, ctx.CallCount, ctx.RuntimeMinutes))

	// 账户
	sb.WriteString(fmt.Sprintf("账户: 净值%.2f | 余额%.2f (%.1f%%) | 盈亏%+.2f%% | 保证金%.1f%% | 持仓%d个\n\n",
		ctx.Account.TotalEquity,
		ctx.Account.AvailableBalance,
		(ctx.Account.AvailableBalance/ctx.Account.TotalEquity)*100,
		ctx.Account.TotalPnLPct,
		ctx.Account.MarginUsedPct,
		ctx.Account.PositionCount))

	// 持仓（完整市场数据）
	if len(ctx.Positions) > 0 {
		sb.WriteString("## 当前持仓\n")
		for i, pos := range ctx.Positions {
			// 计算持仓时长
			holdingDuration := ""
			if pos.UpdateTime > 0 {
				durationMs := time.Now().UnixMilli() - pos.UpdateTime
				durationMin := durationMs / (1000 * 60) // 转换为分钟
				if durationMin < 60 {
					holdingDuration = fmt.Sprintf(" | 持仓时长%d分钟", durationMin)
				} else {
					durationHour := durationMin / 60
					durationMinRemainder := durationMin % 60
					holdingDuration = fmt.Sprintf(" | 持仓时长%d小时%d分钟", durationHour, durationMinRemainder)
				}
			}

			sb.WriteString(fmt.Sprintf("%d. %s %s | 入场价%.4f 当前价%.4f | 数量%.4f | 盈亏%+.2f(%.2f%%) | 杠杆%dx | 保证金%.0f | 强平价%.4f%s\n\n",
				i+1, pos.Symbol, strings.ToUpper(pos.Side),
				pos.EntryPrice, pos.MarkPrice, pos.Quantity,
				pos.UnrealizedPnL, pos.UnrealizedPnLPct,
				pos.Leverage, pos.MarginUsed, pos.LiquidationPrice, holdingDuration))

			// 使用FormatMarketData输出完整市场数据
			if marketData, ok := ctx.MarketDataMap[pos.Symbol]; ok {
				sb.WriteString(market.Format(marketData))

				// 添加持仓相关的技术分析
				sb.WriteString(analyzePositionTechnical(marketData, &pos))
				sb.WriteString("\n")
			}
		}
	} else {
		sb.WriteString("当前持仓: 无\n\n")
	}

	// 候选币种（完整市场数据）
	sb.WriteString(fmt.Sprintf("## 候选币种 (%d个)\n\n", len(ctx.MarketDataMap)))
	displayedCount := 0
	for _, coin := range ctx.CandidateCoins {
		marketData, hasData := ctx.MarketDataMap[coin.Symbol]
		if !hasData {
			continue
		}
		displayedCount++

		sourceTags := ""
		if len(coin.Sources) > 1 {
			sourceTags = " (AI500+OI_Top双重信号)"
		} else if len(coin.Sources) == 1 && coin.Sources[0] == "oi_top" {
			sourceTags = " (OI_Top持仓增长)"
		}

		// 使用FormatMarketData输出完整市场数据
		sb.WriteString(fmt.Sprintf("### %d. %s%s\n\n", displayedCount, coin.Symbol, sourceTags))
		sb.WriteString(market.Format(marketData))

		// 添加交易信号分析
		sb.WriteString(analyzeTradingSignals(marketData))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// 性能指标
	if ctx.Performance != nil {
		type PerformanceData struct {
			SharpeRatio float64 `json:"sharpe_ratio"`
			WinRate     float64 `json:"win_rate"`
			TotalTrades int     `json:"total_trades"`
			MaxDrawdown float64 `json:"max_drawdown"`
		}
		var perfData PerformanceData
		if jsonData, err := json.Marshal(ctx.Performance); err == nil {
			if err := json.Unmarshal(jsonData, &perfData); err == nil {
				sb.WriteString("## 📊 性能指标\n")
				sb.WriteString(fmt.Sprintf("夏普比率: %.2f | 胜率: %.1f%% | 总交易: %d | 最大回撤: %.2f%%\n\n",
					perfData.SharpeRatio, perfData.WinRate*100, perfData.TotalTrades, perfData.MaxDrawdown*100))
			}
		}
	}

	// 市场状态摘要
	sb.WriteString("## 📈 市场状态摘要\n")
	sb.WriteString(generateMarketSummary(ctx.MarketDataMap))
	sb.WriteString("\n")

	sb.WriteString("---\n\n")
	sb.WriteString("现在请基于多时间框架分析并输出决策（思维链 + JSON）\n")

	return sb.String()
}

// 辅助函数：分析持仓技术面
func analyzePositionTechnical(data *market.Data, position *PositionInfo) string {
	var sb strings.Builder
	sb.WriteString("持仓技术分析:\n")

	// 多时间框架趋势一致性
	if data.IntradaySeries != nil {
		sb.WriteString(fmt.Sprintf("时间框架对齐: %s | 趋势强度: %.1f\n",
			data.IntradaySeries.TimeframeAlignment, data.IntradaySeries.TrendStrength))
	}

	// 关键支撑阻力事实
	if data.LongerTermContext != nil && data.LongerTermContext.HigherTimeframeContext != nil {
		htf := data.LongerTermContext.HigherTimeframeContext
		if len(htf.KeyLevels4h) > 0 {
			nearestLevel := findNearestLevel(data.CurrentPrice, htf.KeyLevels4h)
			distancePct := (data.CurrentPrice - nearestLevel) / nearestLevel * 100
			sb.WriteString(fmt.Sprintf("最近关键位: %.4f (距离: %.2f%%)\n", nearestLevel, distancePct))
		}
	}

	// 当前风险评估（基于事实数据）
	riskLevel := assessCurrentRisk(data, position)
	sb.WriteString(fmt.Sprintf("当前风险等级: %s\n", riskLevel))

	// 持仓盈亏分析（纯事实）
	sb.WriteString(analyzePositionPnL(data, position))

	return sb.String()
}

// 辅助函数：评估当前风险（基于事实数据）
func assessCurrentRisk(data *market.Data, position *PositionInfo) string {
	// 计算距离强平价格的距离
	distanceToLiquidation := math.Abs(position.MarkPrice-position.LiquidationPrice) / position.MarkPrice * 100

	// 计算波动率风险
	volatilityRisk := 0.0
	if data.LongerTermContext != nil {
		atrPct := data.LongerTermContext.ATR14 / data.CurrentPrice * 100
		volatilityRisk = atrPct * float64(position.Leverage)
	}

	// 未实现盈亏风险
	pnlRisk := 0.0
	if math.Abs(position.UnrealizedPnLPct) > 10 {
		pnlRisk = 30
	} else if math.Abs(position.UnrealizedPnLPct) > 5 {
		pnlRisk = 20
	}

	// 综合风险评估（仅描述当前状态）
	totalRisk := distanceToLiquidation*0.4 + volatilityRisk*0.4 + pnlRisk*0.2

	if totalRisk > 15 || distanceToLiquidation < 3 {
		return "高风险"
	} else if totalRisk > 8 || distanceToLiquidation < 6 {
		return "中风险"
	} else {
		return "低风险"
	}
}

// 辅助函数：评估持仓风险
func assessPositionRisk(data *market.Data, position *PositionInfo) string {
	// 计算距离强平价格的距离
	distanceToLiquidation := math.Abs(position.MarkPrice-position.LiquidationPrice) / position.MarkPrice * 100

	// 计算波动率风险
	volatilityRisk := 0.0
	if data.LongerTermContext != nil {
		atrPct := data.LongerTermContext.ATR14 / data.CurrentPrice * 100
		volatilityRisk = atrPct * float64(position.Leverage)
	}

	// 未实现盈亏风险
	pnlRisk := 0.0
	if math.Abs(position.UnrealizedPnLPct) > 10 {
		pnlRisk = 30
	} else if math.Abs(position.UnrealizedPnLPct) > 5 {
		pnlRisk = 20
	}

	// 综合风险评估
	totalRisk := distanceToLiquidation*0.4 + volatilityRisk*0.4 + pnlRisk*0.2

	if totalRisk > 15 || distanceToLiquidation < 3 {
		return "🔴 高风险"
	} else if totalRisk > 8 || distanceToLiquidation < 6 {
		return "🟡 中风险"
	} else {
		return "🟢 低风险"
	}
}

// 辅助函数：分析持仓盈亏情况
func analyzePositionPnL(data *market.Data, position *PositionInfo) string {
	var sb strings.Builder

	// 计算相对于入场价的位置（考虑持仓方向）
	var priceChangeSinceEntry float64
	if position.Side == "long" {
		priceChangeSinceEntry = (position.MarkPrice - position.EntryPrice) / position.EntryPrice * 100
	} else { // short position
		priceChangeSinceEntry = (position.EntryPrice - position.MarkPrice) / position.EntryPrice * 100
	}

	// 当前持仓表现状态
	if position.UnrealizedPnLPct > 0 {
		sb.WriteString(fmt.Sprintf("持仓状态: 盈利%.2f%% (金额: %.2f)\n", position.UnrealizedPnLPct, position.UnrealizedPnL))
	} else {
		sb.WriteString(fmt.Sprintf("持仓状态: 亏损%.2f%% (金额: %.2f)\n", position.UnrealizedPnLPct, position.UnrealizedPnL))
	}

	// 价格变化事实
	sb.WriteString(fmt.Sprintf("价格变化: %.2f%% (入场: %.4f → 当前: %.4f)\n",
		priceChangeSinceEntry, position.EntryPrice, position.MarkPrice))

	// 相对于市场的表现
	if data.PriceChange1h > 0 && priceChangeSinceEntry < 0 {
		sb.WriteString("表现对比: 1小时市场上涨但持仓下跌\n")
	} else if data.PriceChange1h < 0 && priceChangeSinceEntry > 0 {
		sb.WriteString("表现对比: 1小时市场下跌但持仓上涨\n")
	} else if data.PriceChange1h > 0 && priceChangeSinceEntry > 0 {
		sb.WriteString("表现对比: 持仓与1小时市场同向上涨\n")
	} else if data.PriceChange1h < 0 && priceChangeSinceEntry < 0 {
		sb.WriteString("表现对比: 持仓与1小时市场同向下跌\n")
	}

	// 相对于关键价位的位置
	if data.LongerTermContext != nil && data.LongerTermContext.HigherTimeframeContext != nil {
		htf := data.LongerTermContext.HigherTimeframeContext
		if len(htf.KeyLevels4h) > 0 {
			nearestLevel := findNearestLevel(position.MarkPrice, htf.KeyLevels4h)
			levelType := "阻力位"
			if nearestLevel < position.MarkPrice {
				levelType = "支撑位"
			}
			distancePct := math.Abs(position.MarkPrice-nearestLevel) / position.MarkPrice * 100
			sb.WriteString(fmt.Sprintf("关键价位: 最近%s在%.4f (距离: %.2f%%)\n", levelType, nearestLevel, distancePct))
		}
	}

	// 持仓价值事实
	positionValue := position.Quantity * position.MarkPrice
	sb.WriteString(fmt.Sprintf("持仓价值: %.2f | 保证金比例: %.1f%%\n",
		positionValue, (position.MarginUsed/positionValue)*100))

	return sb.String()
}

// 辅助函数：计算整体趋势强度
func calculateOverallTrendStrength(data *market.Data) float64 {
	// 基于多时间框架指标计算综合趋势强度
	strength := 0.0

	// EMA趋势强度 (30%)
	emaStrength := 0.0
	if data.EMA20_1h > data.EMA20_4h {
		emaStrength += 15
	}
	if data.EMA20_15m > data.EMA20_1h {
		emaStrength += 15
	}

	// MACD趋势强度 (30%)
	macdStrength := 0.0
	if data.MACD_1h > 0 && data.MACD_4h > 0 {
		macdStrength += 15
	}
	if data.MACD_15m > 0 {
		macdStrength += 15
	}

	// RSI趋势强度 (20%)
	rsiStrength := 0.0
	if data.RSI7_1h > 50 {
		rsiStrength += 10
	}
	if data.RSI7_15m > 50 {
		rsiStrength += 10
	}

	// 价格变化强度 (20%)
	priceStrength := 0.0
	if data.PriceChange1h > 0 && data.PriceChange4h > 0 {
		priceStrength += 10
	}
	if data.PriceChange15m > 0 {
		priceStrength += 10
	}

	strength = emaStrength + macdStrength + rsiStrength + priceStrength
	return strength
}

// 辅助函数：分析交易信号
func analyzeTradingSignals(data *market.Data) string {
	var sb strings.Builder
	sb.WriteString("交易信号分析:\n")

	signals := make([]string, 0)

	// MACD信号状态
	if data.MACD_1h > 0 && data.MACD_15m > 0 {
		signals = append(signals, "MACD双时间框架正值")
	} else if data.MACD_1h < 0 && data.MACD_15m < 0 {
		signals = append(signals, "MACD双时间框架负值")
	}

	// RSI信号状态
	if data.RSI7_1h < 30 && data.RSI7_15m < 30 {
		signals = append(signals, "RSI双时间框架超卖")
	} else if data.RSI7_1h > 70 && data.RSI7_15m > 70 {
		signals = append(signals, "RSI双时间框架超买")
	}

	// 趋势信号状态
	if data.IntradaySeries != nil {
		alignment := data.IntradaySeries.TimeframeAlignment
		if strings.Contains(alignment, "bullish") {
			signals = append(signals, "多时间框架看涨对齐")
		} else if strings.Contains(alignment, "bearish") {
			signals = append(signals, "多时间框架看跌对齐")
		}
	}

	// 成交量信号状态
	if data.IntradaySeries != nil && data.IntradaySeries.VolumeProfile1h != nil {
		if data.IntradaySeries.VolumeProfile1h.VolumeSpike {
			signals = append(signals, "1小时成交量异动")
		}
		if data.IntradaySeries.VolumeProfile1h.VolumeRatio > 0.6 {
			signals = append(signals, "买方成交量主导")
		} else if data.IntradaySeries.VolumeProfile1h.VolumeRatio < 0.4 {
			signals = append(signals, "卖方成交量主导")
		}
	}

	if len(signals) > 0 {
		sb.WriteString("当前信号: " + strings.Join(signals, " | ") + "\n")
	} else {
		sb.WriteString("当前信号: 无显著信号\n")
	}

	// 信号强度事实
	strength := calculateSignalStrength(data)
	sb.WriteString(fmt.Sprintf("信号强度: %d/5\n", strength))

	// 综合评分事实
	score := calculateTradingScore(data)
	sb.WriteString(fmt.Sprintf("综合评分: %.1f/100\n", score))

	return sb.String()
}

// 辅助函数：计算信号强度（纯事实计算）
func calculateSignalStrength(data *market.Data) int {
	strength := 0

	// MACD信号强度
	if data.MACD_1h > 0 && data.MACD_15m > 0 {
		strength += 2
	} else if data.MACD_1h < 0 && data.MACD_15m < 0 {
		strength += 2
	}

	// RSI信号强度
	if data.RSI7_1h < 30 && data.RSI7_15m < 30 {
		strength += 1
	} else if data.RSI7_1h > 70 && data.RSI7_15m > 70 {
		strength += 1
	}

	// 趋势信号强度
	if data.IntradaySeries != nil {
		alignment := data.IntradaySeries.TimeframeAlignment
		if alignment == "strong_bullish_alignment" || alignment == "strong_bearish_alignment" {
			strength += 2
		} else if alignment == "bullish_bias" || alignment == "bearish_bias" {
			strength += 1
		}
	}

	// 成交量信号强度
	if data.IntradaySeries != nil && data.IntradaySeries.VolumeProfile1h != nil {
		if data.IntradaySeries.VolumeProfile1h.VolumeSpike {
			strength += 1
		}
	}

	return strength
}

// 辅助函数：生成市场摘要
func generateMarketSummary(marketDataMap map[string]*market.Data) string {
	var sb strings.Builder

	bullishCount := 0
	bearishCount := 0
	strongSignals := 0

	for _, data := range marketDataMap {
		score := calculateTradingScore(data)
		if score > 70 {
			strongSignals++
		}

		if data.IntradaySeries != nil {
			if strings.Contains(data.IntradaySeries.TimeframeAlignment, "bullish") {
				bullishCount++
			} else if strings.Contains(data.IntradaySeries.TimeframeAlignment, "bearish") {
				bearishCount++
			}
		}
	}

	total := len(marketDataMap)
	if total > 0 {
		sb.WriteString(fmt.Sprintf("看涨币种: %d (%.1f%%) | 看跌币种: %d (%.1f%%) | 强信号: %d\n",
			bullishCount, float64(bullishCount)/float64(total)*100,
			bearishCount, float64(bearishCount)/float64(total)*100,
			strongSignals))
	}

	return sb.String()
}

// 辅助函数：计算交易评分
func calculateTradingScore(data *market.Data) float64 {
	score := 0.0

	// 趋势评分 (30%)
	if data.IntradaySeries != nil {
		switch data.IntradaySeries.TimeframeAlignment {
		case "strong_bullish_alignment", "strong_bearish_alignment":
			score += 30
		case "bullish_bias", "bearish_bias":
			score += 20
		default:
			score += 10
		}
	}

	// 动量评分 (25%)
	if data.MACD_1h > 0 && data.MACD_15m > 0 {
		score += 25
	} else if data.MACD_1h < 0 && data.MACD_15m < 0 {
		score += 25
	} else {
		score += 10
	}

	// RSI评分 (20%)
	if (data.RSI7_1h > 50 && data.RSI7_15m > 50) || (data.RSI7_1h < 50 && data.RSI7_15m < 50) {
		score += 20
	} else {
		score += 5
	}

	// 成交量评分 (15%)
	if data.IntradaySeries != nil && data.IntradaySeries.VolumeProfile1h != nil {
		if data.IntradaySeries.VolumeProfile1h.VolumeSpike {
			score += 15
		} else {
			score += 5
		}
	}

	// 波动率评分 (10%)
	if data.LongerTermContext != nil {
		atrRatio := data.LongerTermContext.ATR14 / data.CurrentPrice
		if atrRatio > 0.02 { // 2% ATR，适中的波动率
			score += 10
		} else {
			score += 5
		}
	}

	return score
}

// 辅助函数：寻找最近的关键价位
func findNearestLevel(price float64, levels []float64) float64 {
	if len(levels) == 0 {
		return price
	}

	nearest := levels[0]
	minDiff := math.Abs(price - nearest)

	for _, level := range levels[1:] {
		diff := math.Abs(price - level)
		if diff < minDiff {
			minDiff = diff
			nearest = level
		}
	}

	return nearest
}

// parseFullDecisionResponse 解析AI的完整决策响应
func parseFullDecisionResponse(ctx *Context, aiResponse string) (*FullDecision, error) {
	// 1. 提取思维链
	cotTrace := extractCoTTrace(aiResponse)

	// 2. 提取JSON决策列表
	decisions, err := extractDecisions(aiResponse)
	if err != nil {
		return &FullDecision{
			CoTTrace:  cotTrace,
			Decisions: []Decision{},
		}, fmt.Errorf("提取决策失败: %w", err)
	}

	// 3. 验证决策
	if err := validateDecisions(ctx, decisions); err != nil {
		return &FullDecision{
			CoTTrace:  cotTrace,
			Decisions: decisions,
		}, fmt.Errorf("决策验证失败: %w", err)
	}

	return &FullDecision{
		CoTTrace:  cotTrace,
		Decisions: decisions,
	}, nil
}

// extractCoTTrace 提取思维链分析
func extractCoTTrace(response string) string {
	// 查找JSON数组的开始位置
	jsonStart := strings.Index(response, "[")

	if jsonStart > 0 {
		// 思维链是JSON数组之前的内容
		return strings.TrimSpace(response[:jsonStart])
	}

	// 如果找不到JSON，整个响应都是思维链
	return strings.TrimSpace(response)
}

// extractDecisions 提取JSON决策列表
func extractDecisions(response string) ([]Decision, error) {
	// 直接查找JSON数组 - 找第一个完整的JSON数组
	arrayStart := strings.Index(response, "[")
	if arrayStart == -1 {
		return nil, fmt.Errorf("无法找到JSON数组起始")
	}

	// 从 [ 开始，匹配括号找到对应的 ]
	arrayEnd := findMatchingBracket(response, arrayStart)
	if arrayEnd == -1 {
		return nil, fmt.Errorf("无法找到JSON数组结束")
	}

	jsonContent := strings.TrimSpace(response[arrayStart : arrayEnd+1])

	// 🔧 修复常见的JSON格式错误：缺少引号的字段值
	// 匹配: "reasoning": 内容"}  或  "reasoning": 内容}  (没有引号)
	// 修复为: "reasoning": "内容"}
	// 使用简单的字符串扫描而不是正则表达式
	jsonContent = fixMissingQuotes(jsonContent)

	// 解析JSON
	var decisions []Decision
	if err := json.Unmarshal([]byte(jsonContent), &decisions); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w\nJSON内容: %s", err, jsonContent)
	}

	return decisions, nil
}

// fixMissingQuotes 替换中文引号为英文引号（避免输入法自动转换）
func fixMissingQuotes(jsonStr string) string {
	jsonStr = strings.ReplaceAll(jsonStr, "\u201c", "\"") // "
	jsonStr = strings.ReplaceAll(jsonStr, "\u201d", "\"") // "
	jsonStr = strings.ReplaceAll(jsonStr, "\u2018", "'")  // '
	jsonStr = strings.ReplaceAll(jsonStr, "\u2019", "'")  // '
	return jsonStr
}

// validateDecisions 验证所有决策（需要账户信息和杠杆配置）
func validateDecisions(ctx *Context, decisions []Decision) error {
	for i, decision := range decisions {
		if err := validateDecision(ctx, &decision); err != nil {
			return fmt.Errorf("决策 #%d 验证失败: %w", i+1, err)
		}
	}
	return nil
}

// findMatchingBracket 查找匹配的右括号
func findMatchingBracket(s string, start int) int {
	if start >= len(s) || s[start] != '[' {
		return -1
	}

	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i
			}
		}
	}

	return -1
}

// validateDecision 验证单个决策的有效性
func validateDecision(ctx *Context, d *Decision) error {
	// 验证action
	validActions := map[string]bool{
		"open_long":          true,
		"open_short":         true,
		"close_long":         true,
		"close_short":        true,
		"hold":               true,
		"wait":               true,
		"update_stop_loss":   true,
		"update_take_profit": true,
		"partial_close":      true,
	}

	if !validActions[d.Action] {
		return fmt.Errorf("无效的action: %s", d.Action)
	}

	accountEquity := ctx.Account.TotalEquity
	availableBalance := ctx.Account.AvailableBalance
	btcEthLeverage := ctx.BTCETHLeverage
	altcoinLeverage := ctx.AltcoinLeverage

	// 开仓操作必须提供完整参数
	if d.Action == "open_long" || d.Action == "open_short" {
		// 根据币种使用配置的杠杆上限
		maxLeverage := altcoinLeverage        // 山寨币使用配置的杠杆
		maxPositionValue := accountEquity * 5 // 山寨币最多5倍账户净值
		positionType := "山寨币"
		if d.Symbol == "BTCUSDT" || d.Symbol == "ETHUSDT" {
			maxLeverage = btcEthLeverage          // BTC和ETH使用配置的杠杆
			maxPositionValue = accountEquity * 10 // BTC/ETH最多10倍账户净值
			positionType = "BTC/ETH"
		}

		if d.Leverage <= 0 || d.Leverage > maxLeverage {
			return fmt.Errorf("杠杆必须在1-%d之间（%s，当前配置上限%d倍）: %d", maxLeverage, d.Symbol, maxLeverage, d.Leverage)
		}
		if d.PositionSizeUSD <= 0 {
			return fmt.Errorf("仓位大小必须大于0: %.2f", d.PositionSizeUSD)
		}

		// 验证可用余额是否足够支付保证金（优先检查，更实际）
		requiredMargin := d.PositionSizeUSD / float64(d.Leverage)
		if requiredMargin > availableBalance {
			return fmt.Errorf("可用余额不足: 开仓需要保证金 %.2f USDT (仓位%.0f ÷ %dx杠杆)，但可用余额仅 %.2f USDT",
				requiredMargin, d.PositionSizeUSD, d.Leverage, availableBalance)
		}

		// 验证仓位价值上限（加1%容差以避免浮点数精度问题）
		// 注意：这个检查在余额检查之后，因为余额是更硬的约束
		tolerance := maxPositionValue * 0.01 // 1%容差
		if d.PositionSizeUSD > maxPositionValue+tolerance {
			return fmt.Errorf("%s单币种仓位价值不能超过%.0f USDT（%.0f倍账户净值），实际: %.0f",
				positionType, maxPositionValue, maxPositionValue/accountEquity, d.PositionSizeUSD)
		}

		if d.StopLoss <= 0 || d.TakeProfit <= 0 {
			return fmt.Errorf("止损和止盈必须大于0")
		}

		currentMarketPrice := ctx.MarketDataMap[d.Symbol].CurrentPrice

		// 验证止损止盈的合理性
		if d.Action == "open_long" {
			if d.StopLoss >= currentMarketPrice {
				return fmt.Errorf("做多时止损价必须低于当前市价%.2f", currentMarketPrice)
			}
			if d.TakeProfit <= currentMarketPrice {
				return fmt.Errorf("做多时止盈价必须高于当前市价%.2f", currentMarketPrice)
			}
			if d.StopLoss >= d.TakeProfit {
				return fmt.Errorf("做多时止损价必须小于止盈价")
			}
		} else {
			if d.StopLoss <= currentMarketPrice {
				return fmt.Errorf("做空时止损价必须高于当前市价%.2f", currentMarketPrice)
			}
			if d.TakeProfit >= currentMarketPrice {
				return fmt.Errorf("做空时止盈价必须低于当前市价%.2f", currentMarketPrice)
			}
			if d.StopLoss <= d.TakeProfit {
				return fmt.Errorf("做空时止损价必须大于止盈价")
			}
		}

		// 计算入场价和爆仓价（假设当前市价）
		var entryPrice, liquidationPrice float64

		maintenanceMargin := 0.005 // 维持保证金率0.5%
		if d.Action == "open_long" {
			// 做多：入场价应该低于当前市价（假设限价单）
			entryPrice = currentMarketPrice * 0.998 // 比市价低0.2%

			// 检查杠杆
			if 1/float64(d.Leverage) <= maintenanceMargin {
				return fmt.Errorf("杠杆倍数%d过高，初始保证金率（1/杠杆）必须大于维持保证金率%.3f", d.Leverage, maintenanceMargin)
			}

			// 做多爆仓价格计算（逐仓模式）
			// 爆仓价 = 入场价 * (1 - 初始保证金率 + 维持保证金率)
			liquidationPrice = entryPrice * (1 - 1/float64(d.Leverage) + maintenanceMargin)

			// 验证止损价必须高于爆仓价（留安全距离）
			safetyMargin := entryPrice * (0.01 + 0.005*float64(d.Leverage)/10) // 1%基础 + 杠杆调整
			if d.StopLoss <= liquidationPrice+safetyMargin {
				return fmt.Errorf("止损价过低，会在触及前爆仓: 止损%.2f ≤ 爆仓价%.2f+安全距离%.2f (杠杆%dx)，建议止损设在%.2f以上",
					d.StopLoss, liquidationPrice, safetyMargin, d.Leverage, liquidationPrice+safetyMargin)
			}
		} else {
			// 做空逻辑类似...
			entryPrice = currentMarketPrice * 1.002 // 比市价高0.2%

			// 检查杠杆
			if 1/float64(d.Leverage) <= maintenanceMargin {
				return fmt.Errorf("杠杆倍数%d过高，初始保证金率（1/杠杆）必须大于维持保证金率%.3f", d.Leverage, maintenanceMargin)
			}

			// 做空爆仓价格计算（逐仓模式）
			// 爆仓价 = 入场价 * (1 + 初始保证金率 - 维持保证金率)
			liquidationPrice = entryPrice * (1 + (1/float64(d.Leverage) - maintenanceMargin))

			// 验证止损价必须低于爆仓价（留安全距离）
			safetyMargin := entryPrice * (0.01 + 0.005*float64(d.Leverage)/10)

			if d.StopLoss >= liquidationPrice-safetyMargin {
				return fmt.Errorf("止损价过高，会在触及前爆仓: 止损%.2f ≥ 爆仓价%.2f-安全距离%.2f (杠杆%dx)，建议止损设在%.2f以下",
					d.StopLoss, liquidationPrice, safetyMargin, d.Leverage, liquidationPrice-safetyMargin)
			}
		}

		var riskPercent, rewardPercent, riskRewardRatio float64
		if d.Action == "open_long" {
			riskPercent = (entryPrice - d.StopLoss) / entryPrice * 100
			rewardPercent = (d.TakeProfit - entryPrice) / entryPrice * 100
			if riskPercent > 0 {
				riskRewardRatio = rewardPercent / riskPercent
			}
		} else {
			riskPercent = (d.StopLoss - entryPrice) / entryPrice * 100
			rewardPercent = (entryPrice - d.TakeProfit) / entryPrice * 100
			if riskPercent > 0 {
				riskRewardRatio = rewardPercent / riskPercent
			}
		}

		// 主流币可以适当放宽要求
		var minRiskRewardRatio float64
		if d.Symbol == "SOLUSDT" || d.Symbol == "BTCUSDT" || d.Symbol == "ETHUSDT" {
			minRiskRewardRatio = 2.5 // 主流币2.5:1即可
		} else {
			minRiskRewardRatio = 3.0 // 山寨币保持3.0
		}
		// 硬约束：风险回报比必须≥minRiskRewardRatio
		if riskRewardRatio < minRiskRewardRatio {
			return fmt.Errorf("风险回报比过低(%.2f:1)，必须≥%.2f:1 [%s %s 市价:%.4f] [风险:%.2f%% 收益:%.2f%%] [止损:%.4f 止盈:%.4f]",
				riskRewardRatio, minRiskRewardRatio, d.Symbol, d.Action, entryPrice, riskPercent, rewardPercent, d.StopLoss, d.TakeProfit)
		}
	}

	return nil
}
