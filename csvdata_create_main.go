package main

import (
	"fmt"
	"log/slog"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"

	"sv_stockcheck/convert"
	"sv_stockcheck/fileio"
)

// ---- const
const StockCode = "7779"
const ModelOutputDir = "ModelData/"
const RawOutputDir = "RawData/"

type ObtainType int

const (
	Stock ObtainType = iota // 株価の取得
	Forex                   // 為替の取得
)

type TermEnum int

const (
	Term5 TermEnum = iota
	Term14
	Term30

	TermNum
)

// ---- struct

// 銘柄情報構造体
type StockBrandInformation struct {
	ParseDate         time.Time        `json:"parsedate"`         //	日付
	Opening           float64          `json:"opening"`           //	始値
	High              float64          `json:"high"`              //	高値
	Low               float64          `json:"low"`               //	安値
	Closing           float64          `json:"closing"`           //	終値
	Volume            float64          `json:"volume"`            //	出来高
	MovingAve         [TermNum]float64 `json:"movingave"`         // 移動平均配列
	Volatility        [TermNum]float64 `json:"volatility"`        // ボラティリティ(標準偏差)
	HighLowVolatility [TermNum]float64 `json:"highlowvolatility"` // ボラティリティ(高値と安値)
	ATR               [TermNum]float64 `json:"atr"`               // ATR
	UpperBBand        [TermNum]float64 `json:"upperbband"`        // ボリンジャーバンド(上)
	UnderBBand        [TermNum]float64 `json:"underbband"`        // ボリンジャーバンド(下)
	MADRate           [TermNum]float64 `json:"madrate"`           // 移動平均線乖離率
	RSI               [TermNum]float64 `json:"rsi"`               // Relative Strength Index
	ShortMacdVal      float64          `json:"shortmacdval"`      // MACD値(短期間)
	ShortMacdSmaSig   float64          `json:"shortmacdsmasig"`   // SMA-MACDシグナルライン(短期間)
	ShortMacdSmaHisto float64          `json:"shortmacdsmahisto"` // SMA-MACDヒストグラム(短期間)
	ShortMacdEmaSig   float64          `json:"shortmacdemasig"`   // EMA-MACDシグナルライン(短期間)
	ShortMacdEmaHisto float64          `json:"shortmacdemahisto"` // EMA-MACDヒストグラム(短期間)
	LongMacdVal       float64          `json:"longmacdval"`       // MACD値(長期間)
	LongMacdSmaSig    float64          `json:"longmacdsmasig"`    // SMA-MACDシグナルライン(長期間)
	LongMacdSmaHisto  float64          `json:"longmacdsmahisto"`  // SMA-MACDヒストグラム(長期間)
	LongMacdEmaSig    float64          `json:"longmacdemasig"`    // EMA-MACDシグナルライン(長期間)
	LongMacdEmaHisto  float64          `json:"longmacdemahisto"`  // EMA-MACDヒストグラム(長期間)
}

// ---- Global Variable

// ---- Package Global Variable

var termDay = []int{
	5,  // Term5
	14, // Term14
	30, // Term30
}

var nowObtain = Stock

// ---- public function ----

// ---- private function

// 移動平均を計算する
func calcMovingAverage(data []float64) float64 {
	dataLength := len(data)
	if dataLength == 0 {
		return 0 // またはエラー処理
	}
	sumPastData := 0.0
	for _, c := range data {
		sumPastData += c
	}
	return sumPastData / float64(dataLength)
}

// calculateEMA calculates the Exponential Moving Average (EMA : 指数移動平均) for a given data series
func calculateEMA(data []float64, window int) []float64 {
	if len(data) < window {
		return nil // Not enough data to calculate EMA
	}

	alpha := 2.0 / float64(window+1) // Smoothing factor
	ema := make([]float64, len(data))

	// 有効なデータがある一番古い箇所を検索
	var startIdx int
	for startIdx = len(data) - 1; startIdx >= 0; startIdx-- {
		if !math.IsNaN(data[startIdx]) {
			break
		}
	}
	if startIdx < window {
		return nil // Not enough data to calculate EMA
	}

	// Initialize the first EMA value as the average of the first 'window' data points
	sum := 0.0
	for i := startIdx - 1; i > startIdx-1-window; i-- {
		sum += data[i]

	}
	ema[startIdx-window] = sum / float64(window)

	// Calculate EMA for the rest of the data
	for i := startIdx - 1 - window; i >= 0; i-- {
		ema[i] = (data[i] * alpha) + (ema[i+1] * (1 - alpha))
	}

	// Set EMA values before the initial window to NaN (not enough data)
	for i := startIdx - window + 1; i < len(data); i++ {
		ema[i] = math.NaN()
	}

	return ema
}

// 標準偏差を計算する関数
func calcStandardDeviation(data []float64, mean float64) float64 {
	if len(data) == 0 {
		return 0 // またはエラー処理
	}
	sumOfSquares := 0.0
	for _, value := range data {
		deviation := value - mean
		sumOfSquares += deviation * deviation
	}
	return math.Sqrt(sumOfSquares / float64(len(data))) // 母集団標準偏差
}

// 移動平均線の乖離率(MovingAverageDeviationRate)を計算する関数
func calcMADRate(closing float64, movingAve float64) float64 {
	if movingAve == 0 {
		return 0 // またはエラー処理
	}
	rate := ((closing - movingAve) / movingAve) * 100
	return rate
}

// RSI(Relative Strength Index)を計算する関数
// 引数pricesの日にちそれぞれに対して、期間periodのRSIを計算して、結果を配列で返す
func calcRSI(prices []float64, period int) ([]float64, error) {
	// Check if there is enough data
	if len(prices) < period {
		return nil, fmt.Errorf("not enough data to calculate RSI")
	}

	// Initialize RSI array
	rsis := make([]float64, len(prices))
	// Fill the first `period-1` elements with NaN
	for i := 0; i < period-1; i++ {
		rsis[i] = math.NaN()
	}

	// Reverse the prices to handle descending order
	reversedPrices := make([]float64, len(prices))
	for i := 0; i < len(prices); i++ {
		reversedPrices[i] = prices[len(prices)-1-i]
	}

	// Calculate the each gain and loss
	gain := make([]float64, len(prices))
	loss := make([]float64, len(prices))
	for i := 1; i < len(prices); i++ {
		change := reversedPrices[i] - reversedPrices[i-1]
		if change > 0 {
			gain[i] = change
			loss[i] = 0
		} else {
			gain[i] = 0
			loss[i] = -change
		}
	}

	// Calculate RSI
	for i := range reversedPrices {
		rsisIndex := len(prices) - 1 - i
		if i < period {
			rsis[rsisIndex] = math.NaN()
		} else {
			avgGain := 0.0
			avgLoss := 0.0
			for j := 0; j < period; j++ {
				avgGain += gain[i-j]
				avgLoss += loss[i-j]
			}
			avgGain /= float64(period)
			avgLoss /= float64(period)

			if avgLoss == 0 {
				rsis[rsisIndex] = 100
			} else {
				rs := avgGain / avgLoss
				rsis[rsisIndex] = 100 - (100 / (1 + rs))
			}
		}
	}

	return rsis, nil
}

// MACD(Moving Average Convergence Divergence)を計算する関数
func calcMACD(shortEma []float64, longEma []float64) ([]float64, []float64, []float64, []float64, []float64, error) {

	if len(shortEma) != len(longEma) {
		return nil, nil, nil, nil, nil, fmt.Errorf("not enough data to calculate MACD")
	}

	macdValue := make([]float64, len(shortEma))
	macdSignal := make([]float64, len(shortEma))
	macdHisto := make([]float64, len(shortEma))
	macdEmaSignal := make([]float64, len(shortEma))
	macdEmaHisto := make([]float64, len(shortEma))
	for i := 0; i < len(shortEma); i++ {
		macdValue[i] = shortEma[i] - longEma[i]
	}

	// シグナルラインの計算を行う(SMA)
	window := 9
	for i := 0; i < len(macdValue); i++ {

		sum := 0.0
		isOufOfTarget := false
		for j := 0; j < window; j++ {
			idx := i + j
			if idx >= len(macdValue) || macdValue[idx] == math.NaN() {
				isOufOfTarget = true
				break
			}
			sum += macdValue[idx]
		}
		if isOufOfTarget == true {
			macdSignal[i] = math.NaN()
			macdHisto[i] = math.NaN()
		} else {
			macdSignal[i] = sum / float64(window)
			macdHisto[i] = macdValue[i] - macdSignal[i]
		}
	}

	// シグナルラインの計算を行う(EMA)
	macdEmaSignal = calculateEMA(macdValue, window)
	for i := 0; i < len(macdValue); i++ {
		if macdEmaSignal[i] == math.NaN() {
			macdEmaHisto[i] = math.NaN()
		} else {
			macdEmaHisto[i] = macdValue[i] - macdEmaSignal[i]
		}

	}
	return macdValue, macdSignal, macdHisto, macdEmaSignal, macdEmaHisto, nil
}

// 1銘柄のurlを引数として、該当した銘柄の情報を返す
func checkOneStockBrand(url string) []StockBrandInformation {

	// Instantiate default collector
	c := colly.NewCollector()

	// データの取得 - テーブル
	var retValue []StockBrandInformation
	// <table class="stock_kabuka_dwm">
	c.OnHTML(".stock_kabuka_dwm > tbody", func(e *colly.HTMLElement) {
		e.ForEach("tr", func(_ int, el *colly.HTMLElement) {
			var single StockBrandInformation
			var err error

			getStr := "20" + el.ChildText("th:nth-child(1)")
			single.ParseDate, err = convert.ConvertStringToTime(getStr)
			if err != nil {
				slog.Info("err", "err", err)
			}

			getStr = strings.ReplaceAll(el.ChildText("td:nth-child(2)"), ",", "")
			single.Opening, _ = strconv.ParseFloat(getStr, 64)
			getStr = strings.ReplaceAll(el.ChildText("td:nth-child(3)"), ",", "")
			single.High, _ = strconv.ParseFloat(getStr, 64)
			getStr = strings.ReplaceAll(el.ChildText("td:nth-child(4)"), ",", "")
			single.Low, _ = strconv.ParseFloat(getStr, 64)
			getStr = strings.ReplaceAll(el.ChildText("td:nth-child(5)"), ",", "")
			single.Closing, _ = strconv.ParseFloat(getStr, 64)
			getStr = strings.ReplaceAll(el.ChildText("td:nth-child(8)"), ",", "")
			single.Volume, _ = strconv.ParseFloat(getStr, 64)

			retValue = append(retValue, single)
		})
	})

	// Start scraping on https://XXX
	c.Visit(url)

	return retValue
}

// csvファイル、スクレイピングした該当銘柄の情報をマージする
func csvMergeOneStockBrand(stockData []StockBrandInformation, csvContents []StockBrandInformation) []StockBrandInformation {

	for _, v := range stockData {

		// 重複チェック
		isRegistered := false
		for _, vv := range csvContents {
			isRegistered = v.ParseDate.Equal(vv.ParseDate)
			if isRegistered == true {
				break
			}
		}
		if isRegistered != false {
			//slog.Info("Duplicate SKip", "v", v.ParseDate)
			continue
		}

		csvContents = append(csvContents, v)
	}
	return csvContents
}

// csvファイルを読みStockBrandInformationへデータをインサートする
func readCSVInsertData(csvName string) ([]StockBrandInformation, bool) {

	var retData []StockBrandInformation
	retInitialFlag := false

	fileContents, err := fileio.FileIoCsvRead(csvName)
	if err != nil {
		retInitialFlag = true
		slog.Info("FileReadError", "err", err)
	} else {

		for i, v := range fileContents {
			// 先頭はタイトル行なのでSkip
			if i == 0 {
				continue
			}

			var single StockBrandInformation
			/*
				v[0] = v[0] + " 00:00:00"
				single.ParseDate, _ = time.Parse(time.DateTime, v[0])
			*/
			single.ParseDate, err = convert.ConvertStringToTime(v[0])
			if err != nil {
				slog.Info("err", "err", err)
			}
			single.Opening, _ = strconv.ParseFloat(v[1], 64)
			single.High, _ = strconv.ParseFloat(v[2], 64)
			single.Low, _ = strconv.ParseFloat(v[3], 64)
			single.Closing, _ = strconv.ParseFloat(v[4], 64)
			single.Volume, _ = strconv.ParseFloat(v[5], 64)
			retData = append(retData, single)
		}
		sort.Slice(retData, func(i, j int) bool {
			return retData[i].ParseDate.After(retData[j].ParseDate)
		})

		// 読み込んだCSVを1バージョン前のモノとしてcsvに出力する
		csvNameSlice := strings.Split(csvName, ".")
		csvNameSlice[0] = csvNameSlice[0] + "_bef.csv"
		_ = fileio.FileIoCsvWrite(csvNameSlice[0], fileContents, false)
	}
	return retData, retInitialFlag
}

// スクレイピングし、csvファイルから読みこんだデータとマージしたStockBrandInformationを作成する
func getWebIntegrateData(code string, isInitialCreate bool, csvData []StockBrandInformation) []StockBrandInformation {

	// スクレイピング
	const maxPage = 10
	// Obtain = Stock URL https://kabutan.jp/stock/kabuka?code=147A&ashi=day&page=1
	// Obtain = Forex URL https://kabutan.jp/stock/kabuka?code=0970&ashi=day&page=4
	baseUrl := "https://kabutan.jp/stock/kabuka?code="
	subUrl := "&ashi=day&page="
	for i := 1; i <= maxPage; i++ {
		scrapeUrl := fmt.Sprintf("%s%s%s%d", baseUrl, code, subUrl, i)
		slog.Info("url", "url", scrapeUrl)

		retInformation := checkOneStockBrand(scrapeUrl)
		slog.Info("Web Component", "len", len(retInformation))

		if len(retInformation) <= 0 {
			break
		}
		if isInitialCreate == false && !retInformation[0].ParseDate.After(csvData[0].ParseDate) {
			slog.Info("Already", "information", retInformation[0].ParseDate, "lasttime", csvData[0].ParseDate)
			break
		}

		// CSVからのデータとマージ
		csvData = csvMergeOneStockBrand(retInformation, csvData)
		// 降順でソート
		sort.Slice(csvData, func(i, j int) bool {
			return csvData[i].ParseDate.After(csvData[j].ParseDate)
		})
	}
	return csvData
}

// 取得した該当データに対する移動平均、ボラティリティ(標準偏差)などのテクニカル指標を計算する
func calculateTechnicalIndex(stockData []StockBrandInformation) []StockBrandInformation {

	dataLen := len(stockData)
	var closingPrices []float64
	for _, c := range stockData {
		closingPrices = append(closingPrices, c.Closing)
	}
	var movingAveValue [TermNum][]float64

	for mAveType := Term5; mAveType < TermNum; mAveType++ {
		for i := 0; i < dataLen; i++ {

			const bollingerBandK = 2
			var price, priceDiff, trueRange []float64
			for idx := i; idx < i+termDay[mAveType]; idx++ {
				if idx < dataLen {
					price = append(price, stockData[idx].Closing)
					priceDiff = append(priceDiff, stockData[idx].High-stockData[idx].Low)
					if (idx + 1) < dataLen {
						candidate := []float64{stockData[idx].High - stockData[idx].Low,
							math.Abs(stockData[idx].High - stockData[idx+1].Closing), math.Abs(stockData[idx].Low - stockData[idx+1].Closing)}
						trueRange = append(trueRange, slices.Max(candidate))
					}
				}
			}
			if len(price) == termDay[mAveType] {
				stockData[i].MovingAve[mAveType] = calcMovingAverage(price)
				stockData[i].Volatility[mAveType] = calcStandardDeviation(price, stockData[i].MovingAve[mAveType])
				stockData[i].MADRate[mAveType] = calcMADRate(stockData[i].Closing, stockData[i].MovingAve[mAveType])
				stockData[i].UpperBBand[mAveType] = stockData[i].MovingAve[mAveType] + (bollingerBandK * stockData[i].Volatility[mAveType])
				stockData[i].UnderBBand[mAveType] = stockData[i].MovingAve[mAveType] - (bollingerBandK * stockData[i].Volatility[mAveType])
			} else {
				stockData[i].MovingAve[mAveType] = math.NaN()
				stockData[i].Volatility[mAveType] = math.NaN()
				stockData[i].MADRate[mAveType] = math.NaN()
				stockData[i].UpperBBand[mAveType] = math.NaN()
				stockData[i].UnderBBand[mAveType] = math.NaN()
			}
			if len(priceDiff) == termDay[mAveType] {
				stockData[i].HighLowVolatility[mAveType] = calcMovingAverage(priceDiff)
			} else {
				stockData[i].HighLowVolatility[mAveType] = math.NaN()
			}
			if len(trueRange) == termDay[mAveType] {
				stockData[i].ATR[mAveType] = calcMovingAverage(trueRange)
			} else {
				stockData[i].ATR[mAveType] = math.NaN()
			}
			// MACD計算用に移動平均を配列化
			movingAveValue[mAveType] = append(movingAveValue[mAveType], stockData[i].MovingAve[mAveType])
		}

		// RSIの計算
		resultRSI, errRSI := calcRSI(closingPrices, termDay[mAveType])
		if errRSI != nil {
			for i, _ := range resultRSI {
				stockData[i].RSI[mAveType] = math.NaN()
			}
		} else {
			for i, c := range resultRSI {
				stockData[i].RSI[mAveType] = c
			}
		}
	}

	// MACDの計算
	// 5日移動平均と30日移動平均のMACD
	tmpMACDVal, tmpMACDSignal, tmpMACDHisto, tmpMACDEMASignal, tmpMACDEMAHisto, _ := calcMACD(movingAveValue[Term5], movingAveValue[Term30])
	for i := 0; i < dataLen; i++ {
		stockData[i].ShortMacdVal = tmpMACDVal[i]
		stockData[i].ShortMacdSmaSig = tmpMACDSignal[i]
		stockData[i].ShortMacdSmaHisto = tmpMACDHisto[i]
		stockData[i].ShortMacdEmaSig = tmpMACDEMASignal[i]
		stockData[i].ShortMacdEmaHisto = tmpMACDEMAHisto[i]
	}
	// 14日移動平均と30日移動平均のMACD
	tmpMACDVal, tmpMACDSignal, tmpMACDHisto, tmpMACDEMASignal, tmpMACDEMAHisto, _ = calcMACD(movingAveValue[Term14], movingAveValue[Term30])
	for i := 0; i < dataLen; i++ {
		stockData[i].LongMacdVal = tmpMACDVal[i]
		stockData[i].LongMacdSmaSig = tmpMACDSignal[i]
		stockData[i].LongMacdSmaHisto = tmpMACDHisto[i]
		stockData[i].LongMacdEmaSig = tmpMACDEMASignal[i]
		stockData[i].LongMacdEmaHisto = tmpMACDEMAHisto[i]
	}

	return stockData
}

// 該当銘柄のcsvデータを作成する
func csvCreationOneStockBrand(code string) {

	// RawDataのcsvファイルを読み込んでStockBrandInformationに展開
	// モデル用にテクニカル指標を付加したファイルをModelDataに出力
	rawCsvFileName := fmt.Sprintf("%s%s.csv", RawOutputDir, code)
	modelCsvFileName := fmt.Sprintf("%s%s.csv", ModelOutputDir, code)
	synthesisStockData, isInitialCreation := readCSVInsertData(rawCsvFileName)
	slog.Info("File Component", "len", len(synthesisStockData))

	// スクレイピングし、csvファイルから読みこんだデータとマージしたStockBrandInformationを作成
	synthesisStockData = getWebIntegrateData(code, isInitialCreation, synthesisStockData)

	// 移動平均、ボラティリティの計算
	synthesisStockData = calculateTechnicalIndex(synthesisStockData)

	// CSVに出力するように文字列に変換。日付フォーマットを time.DateTime から　yyyy/mm/dd へ変更する。
	var outputStr [][]string
	var lineStr []string = []string{"date", "Opening", "High", "Low", "Closing"}
	var lineSubStr []string = []string{"MovingAve5", "MovingAve14", "MovingAve30", "Volatility5", "Volatility14", "Volatility30",
		"HighLowVolatility5", "HighLowVolatility14", "HighLowVolatility30",
		"ATR5", "ATR14", "ATR30", "MADRate5", "MADRate14", "MADRate30", "RSI5", "RSI14", "RSI30",
		"shortMACD", "shortMACDSignalSMA", "shortMACDHistoSMA", "shortMACDSignalEMA", "shortMACDHistoEMA", "longMACD", "longMACDSignalSMA", "longMACDHistoSMA", "longMACDSignalEMA", "longMACDHistoEMA",
		"upperBBand5", "upperBBand14", "upperBBand30", "underBBand5", "underBBand14", "underBBand30"}
	if nowObtain != Forex {
		lineStr = append(lineStr, "Volume")
	}
	lineStr = append(lineStr, lineSubStr...)
	//		var nowObtain = Forex
	outputStr = append(outputStr, lineStr)
	for i, c := range synthesisStockData {

		// Nanが発生してしまうデータを出力しない
		// 30日間移動平均でデータ数が30未満だとNaNが発生してしまう
		if i >= len(synthesisStockData)-termDay[Term30] {
			break
		}

		lineStr = nil
		dateStr := c.ParseDate.Format(time.DateTime)
		dateSlice := strings.Split(dateStr, " ")
		dateSlice[0] = strings.ReplaceAll(dateSlice[0], "-", "/")

		lineStr = append(lineStr, dateSlice[0], strconv.FormatFloat(c.Opening, 'f', 5, 64), strconv.FormatFloat(c.High, 'f', 5, 64), strconv.FormatFloat(c.Low, 'f', 5, 64), strconv.FormatFloat(c.Closing, 'f', 5, 64))
		if nowObtain != Forex {
			lineStr = append(lineStr, strconv.FormatFloat(c.Volume, 'f', 5, 64))
		}
		lineStr = append(lineStr, strconv.FormatFloat(c.MovingAve[Term5], 'f', 5, 64), strconv.FormatFloat(c.MovingAve[Term14], 'f', 5, 64), strconv.FormatFloat(c.MovingAve[Term30], 'f', 5, 64),
			strconv.FormatFloat(c.Volatility[Term5], 'f', 5, 64), strconv.FormatFloat(c.Volatility[Term14], 'f', 5, 64), strconv.FormatFloat(c.Volatility[Term30], 'f', 5, 64),
			strconv.FormatFloat(c.HighLowVolatility[Term5], 'f', 5, 64), strconv.FormatFloat(c.HighLowVolatility[Term14], 'f', 5, 64), strconv.FormatFloat(c.HighLowVolatility[Term30], 'f', 5, 64),
			strconv.FormatFloat(c.ATR[Term5], 'f', 5, 64), strconv.FormatFloat(c.ATR[Term14], 'f', 5, 64), strconv.FormatFloat(c.ATR[Term30], 'f', 5, 64),
			strconv.FormatFloat(c.MADRate[Term5], 'f', 5, 64), strconv.FormatFloat(c.MADRate[Term14], 'f', 5, 64), strconv.FormatFloat(c.MADRate[Term30], 'f', 5, 64),
			strconv.FormatFloat(c.RSI[Term5], 'f', 5, 64), strconv.FormatFloat(c.RSI[Term14], 'f', 5, 64), strconv.FormatFloat(c.RSI[Term30], 'f', 5, 64),
			strconv.FormatFloat(c.ShortMacdVal, 'f', 5, 64), strconv.FormatFloat(c.ShortMacdSmaSig, 'f', 5, 64), strconv.FormatFloat(c.ShortMacdSmaHisto, 'f', 5, 64),
			strconv.FormatFloat(c.ShortMacdEmaSig, 'f', 5, 64), strconv.FormatFloat(c.ShortMacdEmaHisto, 'f', 5, 64),
			strconv.FormatFloat(c.LongMacdVal, 'f', 5, 64), strconv.FormatFloat(c.LongMacdSmaSig, 'f', 5, 64), strconv.FormatFloat(c.LongMacdSmaHisto, 'f', 5, 64),
			strconv.FormatFloat(c.LongMacdEmaSig, 'f', 5, 64), strconv.FormatFloat(c.LongMacdEmaHisto, 'f', 5, 64),
			strconv.FormatFloat(c.UpperBBand[Term5], 'f', 5, 64), strconv.FormatFloat(c.UpperBBand[Term14], 'f', 5, 64), strconv.FormatFloat(c.UpperBBand[Term30], 'f', 5, 64),
			strconv.FormatFloat(c.UnderBBand[Term5], 'f', 5, 64), strconv.FormatFloat(c.UnderBBand[Term14], 'f', 5, 64), strconv.FormatFloat(c.UnderBBand[Term30], 'f', 5, 64),
		)
		outputStr = append(outputStr, lineStr)
	}
	_ = fileio.FileIoCsvWrite(modelCsvFileName, outputStr, false)
	slog.Info("Final Component", "Data", len(synthesisStockData), "output", len(outputStr))

	// 基本データをRawDataディレクトリに出力
	var rawLineStr []string = []string{"date", "Opening", "High", "Low", "Closing", "Volume"}
	outputStr = nil
	outputStr = append(outputStr, rawLineStr)
	for _, c := range synthesisStockData {
		lineStr = nil
		dateStr := c.ParseDate.Format(time.DateTime)
		dateSlice := strings.Split(dateStr, " ")
		dateSlice[0] = strings.ReplaceAll(dateSlice[0], "-", "/")

		lineStr = append(lineStr, dateSlice[0], strconv.FormatFloat(c.Opening, 'f', 5, 64), strconv.FormatFloat(c.High, 'f', 5, 64), strconv.FormatFloat(c.Low, 'f', 5, 64),
			strconv.FormatFloat(c.Closing, 'f', 5, 64), strconv.FormatFloat(c.Volume, 'f', 5, 64),
		)
		outputStr = append(outputStr, lineStr)
	}
	_ = fileio.FileIoCsvWrite(rawCsvFileName, outputStr, false)

}

// ---- main
func main() {
	//lambda.Start(checkJraEntries)
	csvCreationOneStockBrand(StockCode)
}
