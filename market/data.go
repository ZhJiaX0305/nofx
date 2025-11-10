package market

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"strings"
)

// Get 获取指定代币的市场数据
func Get(symbol string) (*Data, error) {
	var klines3m, klines15m, klines1h, klines4h []Kline
	var err error
	// 标准化symbol
	symbol = Normalize(symbol)

	// 获取3分钟K线数据 (最近10个)
	klines3m, err = WSMonitorCli.GetCurrentKlines(symbol, "3m")
	if err != nil {
		return nil, fmt.Errorf("获取3分钟K线失败: %v", err)
	}

	// 获取15分钟K线数据
	klines15m, err = WSMonitorCli.GetCurrentKlines(symbol, "15m")
	if err != nil {
		return nil, fmt.Errorf("获取15分钟K线失败: %v", err)
	}

	// 获取1小时K线数据
	klines1h, err = WSMonitorCli.GetCurrentKlines(symbol, "1h")
	if err != nil {
		return nil, fmt.Errorf("获取1小时K线失败: %v", err)
	}

	// 获取4小时K线数据
	klines4h, err = WSMonitorCli.GetCurrentKlines(symbol, "4h")
	if err != nil {
		return nil, fmt.Errorf("获取4小时K线失败: %v", err)
	}

	// 计算当前价格
	currentPrice := klines3m[len(klines3m)-1].Close

	// 计算各时间框架的EMA20
	ema20_3m := calculateEMA(klines3m, 20)
	ema20_15m := calculateEMA(klines15m, 20)
	ema20_1h := calculateEMA(klines1h, 20)
	ema20_4h := calculateEMA(klines4h, 20)

	// 计算各时间框架的MACD
	macd_3m := calculateMACD(klines3m)
	macd_15m := calculateMACD(klines15m)
	macd_1h := calculateMACD(klines1h)
	macd_4h := calculateMACD(klines4h)

	// 计算各时间框架的RSI7
	rsi7_3m := calculateRSI(klines3m, 7)
	rsi7_15m := calculateRSI(klines15m, 7)
	rsi7_1h := calculateRSI(klines1h, 7)

	// 计算买卖比率（基于最新K线）
	buySellRatio := 0.0
	if len(klines3m) > 0 {
		latestKline := klines3m[len(klines3m)-1]
		if latestKline.Volume > 0 {
			buySellRatio = latestKline.TakerBuyBaseVolume / latestKline.Volume
		}
	}

	// 计算价格变化百分比（使用对应时间框架的数据）
	priceChange15m := calculatePriceChange(klines15m, currentPrice, 1)
	priceChange1h := calculatePriceChange(klines1h, currentPrice, 1)
	priceChange4h := calculatePriceChange(klines4h, currentPrice, 1)

	// 获取OI数据
	oiData, err := getOpenInterestData(symbol)
	if err != nil {
		// OI失败不影响整体,使用默认值
		oiData = &OIData{Latest: 0, Average: 0}
	}

	// 获取Funding Rate
	fundingRate, _ := getFundingRate(symbol)

	// 计算日内系列数据（使用多个时间框架）
	intradayData := calculateIntradaySeries(klines3m, klines15m, klines1h)

	// 计算长期数据（使用1小时和4小时数据）
	longerTermData := calculateLongerTermData(klines1h, klines4h)

	return &Data{
		Symbol:            symbol,
		CurrentPrice:      currentPrice,
		PriceChange15m:    priceChange15m,
		PriceChange1h:     priceChange1h,
		PriceChange4h:     priceChange4h,
		EMA20_3m:          ema20_3m,
		EMA20_15m:         ema20_15m,
		EMA20_1h:          ema20_1h,
		EMA20_4h:          ema20_4h,
		MACD_3m:           macd_3m,
		MACD_15m:          macd_15m,
		MACD_1h:           macd_1h,
		MACD_4h:           macd_4h,
		RSI7_3m:           rsi7_3m,
		RSI7_15m:          rsi7_15m,
		RSI7_1h:           rsi7_1h,
		BuySellRatio:      buySellRatio,
		OpenInterest:      oiData,
		FundingRate:       fundingRate,
		IntradaySeries:    intradayData,
		LongerTermContext: longerTermData,
	}, nil
}

// calculatePriceChange 计算价格变化百分比
func calculatePriceChange(klines []Kline, currentPrice float64, periodsBack int) float64 {
	if len(klines) <= periodsBack {
		return 0.0
	}
	previousPrice := klines[len(klines)-1-periodsBack].Close
	if previousPrice > 0 {
		return ((currentPrice - previousPrice) / previousPrice) * 100
	}
	return 0.0
}

// calculateEMA 计算EMA
func calculateEMA(klines []Kline, period int) float64 {
	if len(klines) < period {
		return 0
	}

	// 计算SMA作为初始EMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += klines[i].Close
	}
	ema := sum / float64(period)

	// 计算EMA
	multiplier := 2.0 / float64(period+1)
	for i := period; i < len(klines); i++ {
		ema = (klines[i].Close-ema)*multiplier + ema
	}

	return ema
}

// calculateMACD 计算MACD
func calculateMACD(klines []Kline) float64 {
	if len(klines) < 26 {
		return 0
	}

	// 计算12期和26期EMA
	ema12 := calculateEMA(klines, 12)
	ema26 := calculateEMA(klines, 26)

	// MACD = EMA12 - EMA26
	return ema12 - ema26
}

// calculateRSI 计算RSI
func calculateRSI(klines []Kline, period int) float64 {
	if len(klines) <= period {
		return 0
	}

	gains := 0.0
	losses := 0.0

	// 计算初始平均涨跌幅
	for i := 1; i <= period; i++ {
		change := klines[i].Close - klines[i-1].Close
		if change > 0 {
			gains += change
		} else {
			losses += -change
		}
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	// 使用Wilder平滑方法计算后续RSI
	for i := period + 1; i < len(klines); i++ {
		change := klines[i].Close - klines[i-1].Close
		if change > 0 {
			avgGain = (avgGain*float64(period-1) + change) / float64(period)
			avgLoss = (avgLoss * float64(period-1)) / float64(period)
		} else {
			avgGain = (avgGain * float64(period-1)) / float64(period)
			avgLoss = (avgLoss*float64(period-1) + (-change)) / float64(period)
		}
	}

	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))

	return rsi
}

// calculateATR 计算ATR
func calculateATR(klines []Kline, period int) float64 {
	if len(klines) <= period {
		return 0
	}

	trs := make([]float64, len(klines))
	for i := 1; i < len(klines); i++ {
		high := klines[i].High
		low := klines[i].Low
		prevClose := klines[i-1].Close

		tr1 := high - low
		tr2 := math.Abs(high - prevClose)
		tr3 := math.Abs(low - prevClose)

		trs[i] = math.Max(tr1, math.Max(tr2, tr3))
	}

	// 计算初始ATR
	sum := 0.0
	for i := 1; i <= period; i++ {
		sum += trs[i]
	}
	atr := sum / float64(period)

	// Wilder平滑
	for i := period + 1; i < len(klines); i++ {
		atr = (atr*float64(period-1) + trs[i]) / float64(period)
	}

	return atr
}

// calculateIntradaySeries 计算日内系列数据（多时间框架分析）
func calculateIntradaySeries(klines3m, klines15m, klines1h []Kline) *IntradayData {
	data := &IntradayData{
		MidPrices:   make([]float64, 0, 10),
		EMA20Values: make([]float64, 0, 10),
		MACDValues:  make([]float64, 0, 10),
		RSI7Values:  make([]float64, 0, 10),
		RSI14Values: make([]float64, 0, 10),
	}

	// 计算3分钟级别的详细分析
	data.Trend3m = analyzeTrend(klines3m)
	data.Volatility3m = calculateVolatility(klines3m)
	data.VolumeProfile3m = calculateVolumeProfile(klines3m)
	data.SupportResistance3m = findSupportResistance(klines3m, 5) // 最近5根K线

	// 计算15分钟级别的详细分析
	data.Trend15m = analyzeTrend(klines15m)
	data.Volatility15m = calculateVolatility(klines15m)
	data.VolumeProfile15m = calculateVolumeProfile(klines15m)
	data.SupportResistance15m = findSupportResistance(klines15m, 10) // 最近10根K线
	data.Momentum15m = calculateMomentum(klines15m)

	// 计算1小时级别的详细分析
	data.Trend1h = analyzeTrend(klines1h)
	data.Volatility1h = calculateVolatility(klines1h)
	data.VolumeProfile1h = calculateVolumeProfile(klines1h)
	data.SupportResistance1h = findSupportResistance(klines1h, 20) // 最近20根K线
	data.MarketStructure1h = analyzeMarketStructure(klines1h)

	// 计算多时间框架协同分析
	data.TimeframeAlignment = checkTimeframeAlignment(
		data.Trend3m, data.Trend15m, data.Trend1h,
	)
	data.TrendStrength = calculateTrendStrength(
		klines3m, klines15m, klines1h,
	)

	// 保留原有的3分钟序列数据计算
	start := len(klines3m) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines3m); i++ {
		data.MidPrices = append(data.MidPrices, klines3m[i].Close)

		// 计算每个点的EMA20
		if i >= 19 {
			ema20 := calculateEMA(klines3m[:i+1], 20)
			data.EMA20Values = append(data.EMA20Values, ema20)
		}

		// 计算每个点的MACD
		if i >= 25 {
			macd := calculateMACD(klines3m[:i+1])
			data.MACDValues = append(data.MACDValues, macd)
		}

		// 计算每个点的RSI
		if i >= 7 {
			rsi7 := calculateRSI(klines3m[:i+1], 7)
			data.RSI7Values = append(data.RSI7Values, rsi7)
		}
		if i >= 14 {
			rsi14 := calculateRSI(klines3m[:i+1], 14)
			data.RSI14Values = append(data.RSI14Values, rsi14)
		}
	}

	return data
}

// calculateVolatility 计算波动率（基于ATR）
func calculateVolatility(klines []Kline) float64 {
	if len(klines) < 14 {
		return 0.0
	}
	return calculateATR(klines, 14)
}

// calculateVolumeProfile 计算成交量分布
func calculateVolumeProfile(klines []Kline) *VolumeProfile {
	if len(klines) == 0 {
		return &VolumeProfile{}
	}

	profile := &VolumeProfile{}
	latest := klines[len(klines)-1]

	profile.VolumeTotal = latest.Volume
	profile.VolumeBuy = latest.TakerBuyBaseVolume
	profile.VolumeSell = latest.Volume - latest.TakerBuyBaseVolume

	if latest.Volume > 0 {
		profile.VolumeRatio = latest.TakerBuyBaseVolume / latest.Volume
	}

	// 计算平均成交量
	totalVolume := 0.0
	for _, k := range klines {
		totalVolume += k.Volume
	}
	profile.VolumeAvg = totalVolume / float64(len(klines))

	// 判断成交量是否异常（超过平均2倍）
	profile.VolumeSpike = latest.Volume > (profile.VolumeAvg * 2)

	return profile
}

// calculateMomentum 计算动量指标
func calculateMomentum(klines []Kline) float64 {
	if len(klines) < 10 {
		return 0.0
	}

	// 使用RSI和价格变化率的组合作为动量指标
	rsi := calculateRSI(klines, 14)

	// 计算价格变化率
	priceChange := 0.0
	if len(klines) >= 10 {
		oldPrice := klines[len(klines)-10].Close
		currentPrice := klines[len(klines)-1].Close
		if oldPrice > 0 {
			priceChange = ((currentPrice - oldPrice) / oldPrice) * 100
		}
	}

	// 综合动量指标 (RSI标准化到-100到100 + 价格变化率)
	momentum := (rsi-50)*2 + priceChange
	return momentum
}

// analyzeMarketStructure 分析市场结构
func analyzeMarketStructure(klines []Kline) string {
	if len(klines) < 20 {
		return "unknown"
	}

	// 分析高低点序列判断市场结构
	highs := make([]float64, len(klines))
	lows := make([]float64, len(klines))
	for i, k := range klines {
		highs[i] = k.High
		lows[i] = k.Low
	}

	// 简单的市场结构判断
	higherHighs := 0
	higherLows := 0
	lowerHighs := 0
	lowerLows := 0

	for i := 1; i < len(klines); i++ {
		if highs[i] > highs[i-1] {
			higherHighs++
		} else if highs[i] < highs[i-1] {
			lowerHighs++
		}

		if lows[i] > lows[i-1] {
			higherLows++
		} else if lows[i] < lows[i-1] {
			lowerLows++
		}
	}

	total := float64(len(klines) - 1)
	hhRatio := float64(higherHighs) / total
	hlRatio := float64(higherLows) / total
	lhRatio := float64(lowerHighs) / total
	llRatio := float64(lowerLows) / total

	if hhRatio > 0.6 && hlRatio > 0.6 {
		return "uptrend"
	} else if lhRatio > 0.6 && llRatio > 0.6 {
		return "downtrend"
	} else if (hhRatio + llRatio) > (hlRatio + lhRatio) {
		return "consolidation"
	} else {
		return "ranging"
	}
}

// findSupportResistance 寻找支撑阻力位
func findSupportResistance(klines []Kline, lookback int) []float64 {
	if len(klines) == 0 {
		return nil
	}

	// 限制回溯期
	start := len(klines) - lookback
	if start < 0 {
		start = 0
	}
	recent := klines[start:]

	levels := make([]float64, 0)

	// 寻找明显的高点和低点作为支撑阻力
	for i := 1; i < len(recent)-1; i++ {
		// 阻力位：当前高点高于前后K线的高点
		if recent[i].High > recent[i-1].High && recent[i].High > recent[i+1].High {
			levels = append(levels, recent[i].High)
		}
		// 支撑位：当前低点低于前后K线的低点
		if recent[i].Low < recent[i-1].Low && recent[i].Low < recent[i+1].Low {
			levels = append(levels, recent[i].Low)
		}
	}

	// 添加最近的重要高低点
	if len(recent) > 0 {
		levels = append(levels, recent[0].High, recent[0].Low)
		levels = append(levels, recent[len(recent)-1].High, recent[len(recent)-1].Low)
	}

	return removeDuplicates(levels)
}

// removeDuplicates 去除重复的价格水平
func removeDuplicates(levels []float64) []float64 {
	seen := make(map[float64]bool)
	result := make([]float64, 0)

	for _, level := range levels {
		if !seen[level] {
			seen[level] = true
			result = append(result, level)
		}
	}

	return result
}

// calculateTrendStrength 计算趋势强度
func calculateTrendStrength(klines3m, klines15m, klines1h []Kline) float64 {
	// 基于多个时间框架的指标计算综合趋势强度
	strength3m := calculateSingleTimeframeStrength(klines3m)
	strength15m := calculateSingleTimeframeStrength(klines15m)
	strength1h := calculateSingleTimeframeStrength(klines1h)

	// 权重分配：较长的时间框架权重更高
	return (strength3m*0.2 + strength15m*0.3 + strength1h*0.5)
}

// calculateSingleTimeframeStrength 计算单一时间框架的趋势强度
func calculateSingleTimeframeStrength(klines []Kline) float64 {
	if len(klines) < 20 {
		return 0.0
	}

	// 基于多个指标计算趋势强度
	trend := analyzeTrend(klines)
	rsi := calculateRSI(klines, 14)
	ema20 := calculateEMA(klines, 20)
	ema50 := calculateEMA(klines, 50)
	currentPrice := klines[len(klines)-1].Close

	strength := 0.0

	// RSI强度
	if rsi > 70 || rsi < 30 {
		strength += 25 // 超买超卖区域，趋势可能强劲
	} else if rsi > 60 || rsi < 40 {
		strength += 15
	}

	// EMA排列强度
	if (trend == "bullish" && ema20 > ema50) || (trend == "bearish" && ema20 < ema50) {
		strength += 25
	}

	// 价格相对于EMA的位置
	emaDistance := math.Abs(currentPrice-ema20) / ema20 * 100
	if emaDistance > 2.0 {
		strength += 25 // 价格远离EMA，趋势可能强劲
	} else if emaDistance > 1.0 {
		strength += 15
	}

	// 波动率贡献
	volatility := calculateVolatility(klines)
	avgPrice := (klines[0].Close + klines[len(klines)-1].Close) / 2
	volatilityRatio := volatility / avgPrice * 100
	if volatilityRatio > 3.0 {
		strength += 25 // 高波动率可能伴随强趋势
	} else if volatilityRatio > 1.5 {
		strength += 15
	}

	return math.Min(strength, 100.0)
}

// calculateLongerTermData 计算长期数据（使用1小时和4小时数据，只提供客观数据）
func calculateLongerTermData(klines1h, klines4h []Kline) *LongerTermData {
	data := &LongerTermData{
		MACDValues:  make([]float64, 0, 10),
		RSI14Values: make([]float64, 0, 10),
	}

	// 使用1小时数据计算主要指标
	data.EMA20 = calculateEMA(klines1h, 20)
	data.EMA50 = calculateEMA(klines1h, 50)
	data.EMA100 = calculateEMA(klines1h, 100)

	// 计算ATR
	data.ATR3 = calculateATR(klines1h, 3)
	data.ATR14 = calculateATR(klines1h, 14)

	// 计算成交量
	if len(klines1h) > 0 {
		data.CurrentVolume = klines1h[len(klines1h)-1].Volume
		// 计算平均成交量
		sum := 0.0
		for _, k := range klines1h {
			sum += k.Volume
		}
		data.AverageVolume = sum / float64(len(klines1h))
	}

	// 计算MACD和RSI序列
	start := len(klines1h) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines1h); i++ {
		if i >= 25 {
			macd := calculateMACD(klines1h[:i+1])
			data.MACDValues = append(data.MACDValues, macd)
		}
		if i >= 14 {
			rsi14 := calculateRSI(klines1h[:i+1], 14)
			data.RSI14Values = append(data.RSI14Values, rsi14)
		}
	}

	// 计算4小时级别背景数据（只提供客观数据）
	data.HigherTimeframeContext = &HigherTimeframeContext{
		EMA20_4h:      calculateEMA(klines4h, 20),
		EMA50_4h:      calculateEMA(klines4h, 50),
		EMA100_4h:     calculateEMA(klines4h, 100),
		KeyLevels4h:   findKeyLevels(klines4h),      // 客观的关键价位计算
		PriceLevels4h: extractPriceLevels(klines4h), // 客观的价格数据
	}

	return data
}

// 辅助函数：提取价格水平数据
func extractPriceLevels(klines []Kline) []PriceLevel {
	levels := make([]PriceLevel, len(klines))
	for i, k := range klines {
		levels[i] = PriceLevel{
			High:  k.High,
			Low:   k.Low,
			Open:  k.Open,
			Close: k.Close,
			Time:  k.OpenTime,
		}
	}
	return levels
}

// 辅助函数：分析趋势方向
func analyzeTrend(klines []Kline) string {
	if len(klines) < 2 {
		return "neutral"
	}

	current := klines[len(klines)-1].Close
	previous := klines[len(klines)-2].Close

	if current > previous {
		return "bullish"
	} else if current < previous {
		return "bearish"
	}
	return "neutral"
}

// 辅助函数：检查多时间框架趋势一致性
func checkTimeframeAlignment(trend3m, trend15m, trend1h string) string {
	trends := []string{trend3m, trend15m, trend1h}

	bullishCount := 0
	bearishCount := 0

	for _, trend := range trends {
		if trend == "bullish" {
			bullishCount++
		} else if trend == "bearish" {
			bearishCount++
		}
	}

	if bullishCount == 3 {
		return "strong_bullish_alignment"
	} else if bearishCount == 3 {
		return "strong_bearish_alignment"
	} else if bullishCount >= 2 {
		return "bullish_bias"
	} else if bearishCount >= 2 {
		return "bearish_bias"
	}

	return "mixed"
}

// 辅助函数：寻找关键价位
func findKeyLevels(klines []Kline) []float64 {
	if len(klines) == 0 {
		return nil
	}

	// 简单的支撑阻力位识别：近期高点和低点
	levels := make([]float64, 0)

	// 添加近期高点和低点作为关键价位
	recent := klines[len(klines)-5:] // 最近5根K线
	if len(recent) > 0 {
		high := recent[0].High
		low := recent[0].Low

		for _, k := range recent {
			if k.High > high {
				high = k.High
			}
			if k.Low < low {
				low = k.Low
			}
		}

		levels = append(levels, high, low)
	}

	return levels
}

// getOpenInterestData 获取OI数据
func getOpenInterestData(symbol string) (*OIData, error) {
	url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/openInterest?symbol=%s", symbol)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		OpenInterest string `json:"openInterest"`
		Symbol       string `json:"symbol"`
		Time         int64  `json:"time"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	oi, _ := strconv.ParseFloat(result.OpenInterest, 64)

	return &OIData{
		Latest:  oi,
		Average: oi * 0.999, // 近似平均值
	}, nil
}

// getFundingRate 获取资金费率
func getFundingRate(symbol string) (float64, error) {
	url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/premiumIndex?symbol=%s", symbol)

	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result struct {
		Symbol          string `json:"symbol"`
		MarkPrice       string `json:"markPrice"`
		IndexPrice      string `json:"indexPrice"`
		LastFundingRate string `json:"lastFundingRate"`
		NextFundingTime int64  `json:"nextFundingTime"`
		InterestRate    string `json:"interestRate"`
		Time            int64  `json:"time"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	rate, _ := strconv.ParseFloat(result.LastFundingRate, 64)
	return rate, nil
}

// Format 格式化输出市场数据
func Format(data *Data) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Symbol: %s\n", data.Symbol))
	sb.WriteString(fmt.Sprintf("Current Price: %.2f\n", data.CurrentPrice))
	sb.WriteString(fmt.Sprintf("Price Changes - 15m: %.2f%%, 1h: %.2f%%, 4h: %.2f%%\n\n",
		data.PriceChange15m, data.PriceChange1h, data.PriceChange4h))

	// 多时间框架EMA
	sb.WriteString("EMA20 across timeframes:\n")
	sb.WriteString(fmt.Sprintf("  3m: %.3f, 15m: %.3f, 1h: %.3f, 4h: %.3f\n\n",
		data.EMA20_3m, data.EMA20_15m, data.EMA20_1h, data.EMA20_4h))

	// 多时间框架MACD
	sb.WriteString("MACD across timeframes:\n")
	sb.WriteString(fmt.Sprintf("  3m: %.3f, 15m: %.3f, 1h: %.3f, 4h: %.3f\n\n",
		data.MACD_3m, data.MACD_15m, data.MACD_1h, data.MACD_4h))

	// 多时间框架RSI
	sb.WriteString("RSI7 across timeframes:\n")
	sb.WriteString(fmt.Sprintf("  3m: %.3f, 15m: %.3f, 1h: %.3f\n\n",
		data.RSI7_3m, data.RSI7_15m, data.RSI7_1h))

	sb.WriteString(fmt.Sprintf("Buy/Sell Ratio: %.3f\n\n", data.BuySellRatio))

	sb.WriteString("Additional Data:\n")
	if data.OpenInterest != nil {
		sb.WriteString(fmt.Sprintf("Open Interest: Latest: %.2f Average: %.2f\n",
			data.OpenInterest.Latest, data.OpenInterest.Average))
	}
	sb.WriteString(fmt.Sprintf("Funding Rate: %.2e\n\n", data.FundingRate))

	if data.IntradaySeries != nil {
		sb.WriteString("Intraday series (3-minute intervals, oldest → latest):\n")

		if len(data.IntradaySeries.MidPrices) > 0 {
			sb.WriteString(fmt.Sprintf("Mid prices: %s\n", formatFloatSlice(data.IntradaySeries.MidPrices)))
		}

		if len(data.IntradaySeries.EMA20Values) > 0 {
			sb.WriteString(fmt.Sprintf("EMA indicators (20-period): %s\n", formatFloatSlice(data.IntradaySeries.EMA20Values)))
		}

		if len(data.IntradaySeries.MACDValues) > 0 {
			sb.WriteString(fmt.Sprintf("MACD indicators: %s\n", formatFloatSlice(data.IntradaySeries.MACDValues)))
		}

		if len(data.IntradaySeries.RSI7Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (7-Period): %s\n", formatFloatSlice(data.IntradaySeries.RSI7Values)))
		}

		if len(data.IntradaySeries.RSI14Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (14-Period): %s\n", formatFloatSlice(data.IntradaySeries.RSI14Values)))
		}

		if data.IntradaySeries.TimeframeAlignment != "" {
			sb.WriteString(fmt.Sprintf("Timeframe Alignment: %s\n", data.IntradaySeries.TimeframeAlignment))
		}
		sb.WriteString("\n")
	}

	if data.LongerTermContext != nil {
		sb.WriteString("Longer-term context (1-hour + 4-hour timeframes):\n")

		sb.WriteString(fmt.Sprintf("1h EMAs - 20: %.3f, 50: %.3f, 100: %.3f\n",
			data.LongerTermContext.EMA20, data.LongerTermContext.EMA50, data.LongerTermContext.EMA100))

		sb.WriteString(fmt.Sprintf("ATR - 3-period: %.3f, 14-period: %.3f\n",
			data.LongerTermContext.ATR3, data.LongerTermContext.ATR14))

		sb.WriteString(fmt.Sprintf("Volume - Current: %.3f, Average: %.3f\n",
			data.LongerTermContext.CurrentVolume, data.LongerTermContext.AverageVolume))

		if len(data.LongerTermContext.MACDValues) > 0 {
			sb.WriteString(fmt.Sprintf("MACD indicators: %s\n", formatFloatSlice(data.LongerTermContext.MACDValues)))
		}

		if len(data.LongerTermContext.RSI14Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (14-Period): %s\n", formatFloatSlice(data.LongerTermContext.RSI14Values)))
		}

		// 4小时数据
		if data.LongerTermContext.HigherTimeframeContext != nil {
			htf := data.LongerTermContext.HigherTimeframeContext
			sb.WriteString(fmt.Sprintf("4h EMA: 20=%.3f, 50=%.3f, 100=%.3f\n",
				htf.EMA20_4h, htf.EMA50_4h, htf.EMA100_4h))

			if len(htf.KeyLevels4h) > 0 {
				sb.WriteString(fmt.Sprintf("4h Price Levels: %s\n", formatFloatSlice(htf.KeyLevels4h)))
			}

			if len(htf.PriceLevels4h) > 0 {
				latest := htf.PriceLevels4h[len(htf.PriceLevels4h)-1]
				sb.WriteString(fmt.Sprintf("Latest 4h Kline: Open=%.4f, High=%.4f, Low=%.4f, Close=%.4f\n",
					latest.Open, latest.High, latest.Low, latest.Close))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatFloatSlice 格式化float64切片为字符串
func formatFloatSlice(values []float64) string {
	strValues := make([]string, len(values))
	for i, v := range values {
		strValues[i] = fmt.Sprintf("%.3f", v)
	}
	return "[" + strings.Join(strValues, ", ") + "]"
}

// Normalize 标准化symbol,确保是USDT交易对
func Normalize(symbol string) string {
	symbol = strings.ToUpper(symbol)
	if strings.HasSuffix(symbol, "USDT") {
		return symbol
	}
	return symbol + "USDT"
}

// parseFloat 解析float值
func parseFloat(v interface{}) (float64, error) {
	switch val := v.(type) {
	case string:
		return strconv.ParseFloat(val, 64)
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	default:
		return 0, fmt.Errorf("unsupported type: %T", v)
	}
}
