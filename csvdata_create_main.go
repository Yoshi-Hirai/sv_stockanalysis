package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os/exec"
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
const StockCode = "2586"
const ResourceDir = "Resource/"
const S3BucketName = "for-stock-fx-analysis"
const RawDataFileName = "RawData.csv"
const ModelDataFileName = "ModelData.csv"

var nowObtain = Stock

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

// 共通情報構造体
type CommonInformation struct {
	ParseDate        time.Time `json:"parsedate"`        //	日付
	InterestRateJpn  float64   `json:"InterestRatejpn"`  //	金利日本
	InterestRateUsa  float64   `json:"InterestRateusa"`  //	金利アメリカ
	InterestRateEuro float64   `json:"InterestRateeuro"` //	金利ユーロ
	InterestRateUk   float64   `json:"InterestRateuk"`   //	金利イギリス
	UnemployRateJpn  float64   `json:"unemployratejpn"`  //	失業率日本
	UnemployRateUsa  float64   `json:"unemployrateusa"`  //	失業率アメリカ
	UnemployRateEuro float64   `json:"unemployrateeuro"` //	失業率ユーロ
	UnemployRateUk   float64   `json:"unemployrateuk"`   //	失業率イギリス
	CpiJpn           float64   `json:"cpijpn"`           //	CPI日本
	CpiUsa           float64   `json:"cpiusa"`           //	CPIアメリカ
	CpiEuro          float64   `json:"cpieuro"`          //	CPIユーロ
	CpiUk            float64   `json:"cpiuk"`            //	CPIイギリス
	GdpJpn           float64   `json:"gdpjpn"`           //	GDP日本
	GdpUsa           float64   `json:"gdpusa"`           //	GDPアメリカ
	GdpEuro          float64   `json:"gdpeuro"`          //	GDPユーロ
	GdpUk            float64   `json:"gdpuk"`            //	GDPイギリス
	Tankan           float64   `json:"tankan"`           // 日銀短観
}

// 銘柄情報構造体
type StockBrandInformation struct {
	ParseDate         time.Time        `json:"parsedate"`         //	日付
	Opening           float64          `json:"opening"`           //	始値
	High              float64          `json:"high"`              //	高値
	Low               float64          `json:"low"`               //	安値
	Closing           float64          `json:"closing"`           //	終値
	Volume            float64          `json:"volume"`            //	出来高
	MovingAve         [TermNum]float64 `json:"movingave"`         // 移動平均配列
	EMA               [TermNum]float64 `json:"ema"`               // 指数移動平均
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
	VolumeChangeRate  float64          `json:"volumechangerate"`  // 出来高変化率
	VolumeMovingAve   [TermNum]float64 `json:"volumemovingave"`   // 出来高移動平均配列
	VolumeMADRate     [TermNum]float64 `json:"volumemadrate"`     // 出来高移動平均線乖離率
	VolumeRatio       [TermNum]float64 `json:"volumeratio"`       // 出来高移動平均比率
	VolumeEMA         [TermNum]float64 `json:"volumeema"`         // 出来高指数移動平均
}

// ARIMA予測結果構造体
type ArimaPredictionResultInformation struct {
	Date                    string    `json:"date"`
	ParseDate               time.Time `json:"parsedate"`
	Opening                 float64   `json:"opening"` //	始値
	High                    float64   `json:"high"`    //	高値
	Low                     float64   `json:"low"`     //	安値
	Closing                 float64   `json:"closing"` //	終値
	Volume                  float64   `json:"volume"`  //	出来高
	Arima_Diff_Prediction   float64   `json:"arima_diff_prediction`
	Arima_Actual_Prediction float64   `json:"arima_actual_prediction"`
	Prediction_Difference   float64   `json:"prediction_difference"`
}

// ---- Global Variable

// ---- Package Global Variable

var termDay = []int{
	5,  // Term5
	14, // Term14
	30, // Term30
}
var windowMacdSignal = 9

// ---- public function ----

// ---- private function

// 共通データのCSVを読み込む
// 値がないデータは線形回帰を行う
func readCommonCsv() []CommonInformation {

	var retData []CommonInformation

	fileContents, err := fileio.FileIoCsvRead("Resource/CommonData.csv")
	if err != nil {
		slog.Info("FileReadError", "err", err)
	} else {

		const nMetirc = 17 // 共通CSV(CommonData.csv)にあるメトリック種類数
		const nData = 50   // 共通CSV(CommonData.csv)にあるメトリックデータ数(多めに)
		fileLen := len(fileContents)
		dataLen := fileLen - 1              // 実際のデータ数
		var parseDate [nData]time.Time      // 時系列昇順(csvの並びと反対)
		var metricV [nMetirc][nData]float64 // 時系列昇順(csvの並びと反対)
		var isLinearRegressio [nMetirc]bool

		// CSVを読み線形回帰が必要かを判定
		for i, v := range fileContents {
			// 先頭はタイトル行なのでSkip
			if i == 0 {
				continue
			}

			idx := fileLen - i - 1
			parseDate[idx], _ = convert.ConvertStringToTime(v[0])
			for j := 0; j < nMetirc; j++ {
				// j番目のメトリックスを取得(csvは0カラム目が日付なのでj+1を取得)
				if len(v[j+1]) != 0 {
					metricV[j][idx], _ = strconv.ParseFloat(v[j+1], 64)
				} else {
					isLinearRegressio[j] = true
					metricV[j][idx] = math.Inf(1)
				}
			}
		}

		// メトリック毎にチェックして必要であれば線形回帰を実行
		for i := 0; i < nMetirc; i++ {

			if isLinearRegressio[i] == true {

				var x, y, predictX []float64
				for idx := 0; idx < dataLen; idx++ {
					if math.IsInf(metricV[i][idx], 1) != true {
						x = append(x, float64(idx))
						y = append(y, metricV[i][idx])
					} else {
						predictX = append(predictX, float64(idx))
					}
				}
				predictY := linearRegression(x, y, predictX)
				for j, v := range predictY {
					metricV[i][int(predictX[j])] = v
					//slog.Info("predict", "type", i, "idx", predictX[j], "v", v)
				}
			}
		}

		// CommonInformationに代入
		for i := 0; i < dataLen; i++ {
			var single CommonInformation
			idx := fileLen - i - 1
			single.ParseDate = parseDate[idx]
			single.InterestRateJpn = metricV[0][idx]
			single.InterestRateUsa = metricV[1][idx]
			single.InterestRateEuro = metricV[2][idx]
			single.InterestRateUk = metricV[3][idx]
			single.UnemployRateJpn = metricV[4][idx]
			single.UnemployRateUsa = metricV[5][idx]
			single.UnemployRateEuro = metricV[6][idx]
			single.UnemployRateUk = metricV[7][idx]
			single.CpiJpn = metricV[8][idx]
			single.CpiUsa = metricV[9][idx]
			single.CpiEuro = metricV[10][idx]
			single.CpiUk = metricV[11][idx]
			single.GdpJpn = metricV[12][idx]
			single.GdpUsa = metricV[13][idx]
			single.GdpEuro = metricV[14][idx]
			single.GdpUk = metricV[15][idx]
			single.Tankan = metricV[16][idx]
			retData = append(retData, single)
		}
	}
	return retData
}

// 適正な共通データを返す
func getCommonInformation(cData []CommonInformation, date time.Time) CommonInformation {
	for _, v := range cData {

		if date.Year() == v.ParseDate.Year() && date.Month() == v.ParseDate.Month() {
			return v
		}
	}
	var ret CommonInformation
	return ret
}

// 線形回帰の関数
func linearRegression(x []float64, y []float64, predictX []float64) []float64 {
	// 平均を計算
	mean := func(values []float64) float64 {
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum / float64(len(values))
	}

	meanX := mean(x)
	meanY := mean(y)

	// 傾き (a) を計算
	numerator := 0.0
	denominator := 0.0
	for i := range x {
		numerator += (x[i] - meanX) * (y[i] - meanY)
		denominator += (x[i] - meanX) * (x[i] - meanX)
	}
	a := numerator / denominator

	// 切片 (b) を計算
	b := meanY - a*meanX

	// 予測値を計算
	var predictions []float64
	for _, px := range predictX {
		predictions = append(predictions, a*px+b)
	}

	return predictions
}

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
	for i := 0; i < len(macdValue); i++ {

		sum := 0.0
		isOufOfTarget := false
		for j := 0; j < windowMacdSignal; j++ {
			idx := i + j
			if idx >= len(macdValue) || math.IsNaN(macdValue[idx]) {
				isOufOfTarget = true
				break
			}
			sum += macdValue[idx]
		}
		if isOufOfTarget == true {
			macdSignal[i] = math.NaN()
			macdHisto[i] = math.NaN()
		} else {
			macdSignal[i] = sum / float64(windowMacdSignal)
			macdHisto[i] = macdValue[i] - macdSignal[i]
		}
	}

	// シグナルラインの計算を行う(EMA)
	macdEmaSignal = calculateEMA(macdValue, windowMacdSignal)
	for i := 0; i < len(macdValue); i++ {
		if math.IsNaN(macdEmaSignal[i]) {
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
	var closingPrices, volumeValues []float64
	for _, c := range stockData {
		closingPrices = append(closingPrices, c.Closing)
		volumeValues = append(volumeValues, c.Volume)
	}
	var movingAveValue [TermNum][]float64

	for mAveType := Term5; mAveType < TermNum; mAveType++ {
		for i := 0; i < dataLen; i++ {

			const bollingerBandK = 2
			var price, priceDiff, trueRange, volume []float64
			for idx := i; idx < i+termDay[mAveType]; idx++ {
				if idx < dataLen {
					price = append(price, stockData[idx].Closing)
					priceDiff = append(priceDiff, stockData[idx].High-stockData[idx].Low)
					if (idx + 1) < dataLen {
						candidate := []float64{stockData[idx].High - stockData[idx].Low,
							math.Abs(stockData[idx].High - stockData[idx+1].Closing), math.Abs(stockData[idx].Low - stockData[idx+1].Closing)}
						trueRange = append(trueRange, slices.Max(candidate))
					}
					volume = append(volume, stockData[idx].Volume)
				}
			}
			if len(price) == termDay[mAveType] {
				stockData[i].MovingAve[mAveType] = calcMovingAverage(price)
				stockData[i].Volatility[mAveType] = calcStandardDeviation(price, stockData[i].MovingAve[mAveType])
				stockData[i].MADRate[mAveType] = calcMADRate(stockData[i].Closing, stockData[i].MovingAve[mAveType])
				stockData[i].UpperBBand[mAveType] = stockData[i].MovingAve[mAveType] + (bollingerBandK * stockData[i].Volatility[mAveType])
				stockData[i].UnderBBand[mAveType] = stockData[i].MovingAve[mAveType] - (bollingerBandK * stockData[i].Volatility[mAveType])
				stockData[i].VolumeMovingAve[mAveType] = calcMovingAverage(volume)
				stockData[i].VolumeRatio[mAveType] = stockData[i].Volume / stockData[i].VolumeMovingAve[mAveType]
				stockData[i].VolumeMADRate[mAveType] = calcMADRate(stockData[i].Volume, stockData[i].VolumeMovingAve[mAveType])
			} else {
				stockData[i].MovingAve[mAveType] = math.NaN()
				stockData[i].Volatility[mAveType] = math.NaN()
				stockData[i].MADRate[mAveType] = math.NaN()
				stockData[i].UpperBBand[mAveType] = math.NaN()
				stockData[i].UnderBBand[mAveType] = math.NaN()
				stockData[i].VolumeMovingAve[mAveType] = math.NaN()
				stockData[i].VolumeRatio[mAveType] = math.NaN()
				stockData[i].VolumeMADRate[mAveType] = math.NaN()
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

	//	指数移動平均(終値、出来高)
	tempEma5 := calculateEMA(closingPrices, termDay[Term5])
	tempEma14 := calculateEMA(closingPrices, termDay[Term14])
	tempEma30 := calculateEMA(closingPrices, termDay[Term30])
	tempVolumeEma5 := calculateEMA(volumeValues, termDay[Term5])
	tempVolumeEma14 := calculateEMA(volumeValues, termDay[Term14])
	tempVolumeEma30 := calculateEMA(volumeValues, termDay[Term30])
	//	出来高変化率
	for i := 0; i < dataLen; i++ {
		if i+1 < dataLen && stockData[i+1].Volume > 0 {
			stockData[i].VolumeChangeRate = (stockData[i].Volume - stockData[i+1].Volume) / stockData[i+1].Volume
		}
		stockData[i].EMA[Term5] = tempEma5[i]
		stockData[i].EMA[Term14] = tempEma14[i]
		stockData[i].EMA[Term30] = tempEma30[i]
		stockData[i].VolumeEMA[Term5] = tempVolumeEma5[i]
		stockData[i].VolumeEMA[Term14] = tempVolumeEma14[i]
		stockData[i].VolumeEMA[Term30] = tempVolumeEma30[i]
	}

	return stockData
}

// ARIMA予測モデルのpythonファイルを実行し予測結果を取得する
func arimaPrediction(csvfile string) ([]ArimaPredictionResultInformation, error) {

	// Pythonスクリプトのコマンドを構築
	cmd := exec.Command("python", "arima_insights.py", csvfile)
	// コマンド実行と結果取得
	var stdOutBuff, stdErrBuff bytes.Buffer
	cmd.Stdout = &stdOutBuff
	cmd.Stderr = &stdErrBuff
	err := cmd.Run()
	if err != nil {
		slog.Info("err cmd Run.", "err", err)
		return nil, err
	}
	stdOutStr := stdOutBuff.String()

	// JSON -> 構造体へ
	var arimaPredictResult []ArimaPredictionResultInformation
	err = json.Unmarshal([]byte(stdOutStr), &arimaPredictResult)
	if err != nil {
		slog.Info("err Json Convert.", "err", err)
		return nil, err
	}

	// time.Time を求める
	for i := 0; i < len(arimaPredictResult); i++ {

		// フォーマットを指定
		layout := "2006-01-02T15:04:05.000"
		// パース
		arimaPredictResult[i].ParseDate, err = time.Parse(layout, arimaPredictResult[i].Date)
		if err != nil {
			slog.Info("Error parsing time", "err", err)
			return nil, err
		}
	}
	return arimaPredictResult, nil
}

// 該当銘柄のcsvデータを作成する
func csvCreationOneStockBrand(code string, cData []CommonInformation) {

	// RawDataのcsvファイルを読み込んでStockBrandInformationに展開
	// モデル用にテクニカル指標を付加したファイルをModelDataに出力
	rawCsvFileName := fmt.Sprintf("%s%s/%s", ResourceDir, code, RawDataFileName)
	modelCsvFileName := fmt.Sprintf("%s%s/%s", ResourceDir, code, ModelDataFileName)
	synthesisStockData, isInitialCreation := readCSVInsertData(rawCsvFileName)
	slog.Info("File Component", "len", len(synthesisStockData))

	// スクレイピングし、csvファイルから読みこんだデータとマージしたStockBrandInformationを作成
	synthesisStockData = getWebIntegrateData(code, isInitialCreation, synthesisStockData)

	// 移動平均、ボラティリティの計算
	synthesisStockData = calculateTechnicalIndex(synthesisStockData)

	// ARIMA予測モデル計算
	var arimaPredictionResult []ArimaPredictionResultInformation
	var errArima error
	if isInitialCreation == false {
		arimaPredictionResult, errArima = arimaPrediction(rawCsvFileName)
		if errArima != nil {
			slog.Info("ARIMA Prediction Err.", "error", errArima)
			return
		}
	}
	/*
		for _, c := range arimaPredictionResult {
			slog.Info("Arima", "date", c.Date, "predict", c.Arima_Prediction, "Diff", c.Prediction_Difference)
		}
	*/

	// CSVに出力するように文字列に変換。日付フォーマットを time.DateTime から　yyyy/mm/dd へ変更する。
	var outputStr [][]string
	var lineStr []string = []string{"date", "DayOfWeek", "opening", "high", "low", "closing"}
	var lineSubStr []string = []string{"MovingAve5", "MovingAve14", "MovingAve30", "EMA5", "EMA14", "EMA30", "Volatility5", "Volatility14", "Volatility30",
		"HighLowVolatility5", "HighLowVolatility14", "HighLowVolatility30",
		"ATR5", "ATR14", "ATR30", "MADRate5", "MADRate14", "MADRate30", "RSI5", "RSI14", "RSI30",
		"shortMACD", "shortMACDSignalSMA", "shortMACDHistoSMA", "shortMACDSignalEMA", "shortMACDHistoEMA", "longMACD", "longMACDSignalSMA", "longMACDHistoSMA", "longMACDSignalEMA", "longMACDHistoEMA",
		"upperBBand5", "upperBBand14", "upperBBand30", "underBBand5", "underBBand14", "underBBand30",
		"ARIMAPredict", "ARIMAPredictDiff"}
	if nowObtain != Forex {
		lineStr = append(lineStr, "volume", "VCR", "VMovingAve5", "VMovingAve14", "VMovingAve30", "VolumeRatio5", "VolumeRatio14", "VolumeRatio30",
			"VolumeEMA5", "VolumeEMA14", "VolumeEMA30", "VolumeMADRate5", "VolumeMADRate14", "VolumeMADRate30",
			"InterestRateate", "UnemployRateate", "CPI", "GDP", "Tankan")
	} else {
		if code == "0970" {
			// ユーロドル
			lineStr = append(lineStr, "InterestRateate(USA)", "InterestRateate(EURO)", "UnemployRateate(USA)", "UnemployRateate(EURO)",
				"CPI(USA)", "CPI(EURO)", "GDP(USA)", "GDP(EURO)")
		} else if code == "0952" {
			// ポンド円
			lineStr = append(lineStr, "InterestRateate(UK)", "InterestRateate(JPN)", "UnemployRateate(UK)", "UnemployRateate(JPN)",
				"CPI(UK)", "CPI(JPN)", "GDP(UK)", "GDP(JPN)")
		}
	}
	lineStr = append(lineStr, lineSubStr...)
	//		var nowObtain = Forex
	outputStr = append(outputStr, lineStr)
	for i, c := range synthesisStockData {

		// Nanが発生してしまうデータを出力しない
		// 30日間移動平均でデータ数が30未満だとNaNが発生してしまう
		// MACDシグナルを計算するために、さらにwindowMacdSignal-1(8)日間のデータがないとNanが発生する
		if i >= len(synthesisStockData)-termDay[Term30]-windowMacdSignal+1 {
			break
		}

		lineStr = nil
		commonInfo := getCommonInformation(cData, c.ParseDate)
		var arimaC ArimaPredictionResultInformation
		for arimaI, _ := range arimaPredictionResult {
			if c.ParseDate.Year() == arimaPredictionResult[arimaI].ParseDate.Year() &&
				c.ParseDate.Month() == arimaPredictionResult[arimaI].ParseDate.Month() &&
				c.ParseDate.Day() == arimaPredictionResult[arimaI].ParseDate.Day() {
				arimaC = arimaPredictionResult[arimaI]
				break
			}
		}
		dateStr := c.ParseDate.Format(time.DateTime)
		dateSlice := strings.Split(dateStr, " ")
		dateSlice[0] = strings.ReplaceAll(dateSlice[0], "-", "/")

		lineStr = append(lineStr, dateSlice[0], strconv.Itoa(int(c.ParseDate.Weekday())), strconv.FormatFloat(c.Opening, 'f', 5, 64), strconv.FormatFloat(c.High, 'f', 5, 64), strconv.FormatFloat(c.Low, 'f', 5, 64), strconv.FormatFloat(c.Closing, 'f', 5, 64))
		if nowObtain != Forex {
			lineStr = append(lineStr, strconv.FormatFloat(c.Volume, 'f', 5, 64),
				strconv.FormatFloat(c.VolumeChangeRate, 'f', 5, 64),
				strconv.FormatFloat(c.VolumeMovingAve[Term5], 'f', 5, 64), strconv.FormatFloat(c.VolumeMovingAve[Term14], 'f', 5, 64), strconv.FormatFloat(c.VolumeMovingAve[Term30], 'f', 5, 64),
				strconv.FormatFloat(c.VolumeRatio[Term5], 'f', 5, 64), strconv.FormatFloat(c.VolumeRatio[Term14], 'f', 5, 64), strconv.FormatFloat(c.VolumeRatio[Term30], 'f', 5, 64),
				strconv.FormatFloat(c.VolumeEMA[Term5], 'f', 5, 64), strconv.FormatFloat(c.VolumeEMA[Term14], 'f', 5, 64), strconv.FormatFloat(c.VolumeEMA[Term30], 'f', 5, 64),
				strconv.FormatFloat(c.VolumeMADRate[Term5], 'f', 5, 64), strconv.FormatFloat(c.VolumeMADRate[Term14], 'f', 5, 64), strconv.FormatFloat(c.VolumeMADRate[Term30], 'f', 5, 64),
				strconv.FormatFloat(commonInfo.InterestRateJpn, 'f', 5, 64), strconv.FormatFloat(commonInfo.UnemployRateJpn, 'f', 5, 64),
				strconv.FormatFloat(commonInfo.CpiJpn, 'f', 5, 64), strconv.FormatFloat(commonInfo.GdpJpn, 'f', 5, 64), strconv.FormatFloat(commonInfo.Tankan, 'f', 5, 64))
		} else {
			if code == "0970" {
				// ユーロドル
				lineStr = append(lineStr, strconv.FormatFloat(commonInfo.InterestRateUsa, 'f', 5, 64), strconv.FormatFloat(commonInfo.InterestRateEuro, 'f', 5, 64),
					strconv.FormatFloat(commonInfo.UnemployRateUsa, 'f', 5, 64), strconv.FormatFloat(commonInfo.UnemployRateEuro, 'f', 5, 64),
					strconv.FormatFloat(commonInfo.CpiUsa, 'f', 5, 64), strconv.FormatFloat(commonInfo.CpiEuro, 'f', 5, 64),
					strconv.FormatFloat(commonInfo.GdpUsa, 'f', 5, 64), strconv.FormatFloat(commonInfo.GdpEuro, 'f', 5, 64),
				)
			} else if code == "0952" {
				// ポンド円
				lineStr = append(lineStr, strconv.FormatFloat(commonInfo.InterestRateUk, 'f', 5, 64), strconv.FormatFloat(commonInfo.InterestRateJpn, 'f', 5, 64),
					strconv.FormatFloat(commonInfo.UnemployRateUk, 'f', 5, 64), strconv.FormatFloat(commonInfo.UnemployRateJpn, 'f', 5, 64),
					strconv.FormatFloat(commonInfo.CpiUk, 'f', 5, 64), strconv.FormatFloat(commonInfo.CpiJpn, 'f', 5, 64),
					strconv.FormatFloat(commonInfo.GdpUk, 'f', 5, 64), strconv.FormatFloat(commonInfo.GdpJpn, 'f', 5, 64),
				)
			}
		}
		lineStr = append(lineStr, strconv.FormatFloat(c.MovingAve[Term5], 'f', 5, 64), strconv.FormatFloat(c.MovingAve[Term14], 'f', 5, 64), strconv.FormatFloat(c.MovingAve[Term30], 'f', 5, 64),
			strconv.FormatFloat(c.EMA[Term5], 'f', 5, 64), strconv.FormatFloat(c.EMA[Term14], 'f', 5, 64), strconv.FormatFloat(c.EMA[Term30], 'f', 5, 64),
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
			strconv.FormatFloat(arimaC.Arima_Actual_Prediction, 'f', 7, 64), strconv.FormatFloat(arimaC.Prediction_Difference, 'f', 7, 64),
		)
		outputStr = append(outputStr, lineStr)
	}
	_ = fileio.FileIoCsvWrite(modelCsvFileName, outputStr, false)
	s3Key := fmt.Sprintf("%s/%s", code, ModelDataFileName)
	_ = fileio.UploadFileToS3(S3BucketName, modelCsvFileName, s3Key)
	slog.Info("Final Component", "Data", len(synthesisStockData), "output", len(outputStr))

	// 基本データをRawDataディレクトリに出力
	var rawLineStr []string = []string{"date", "opening", "high", "low", "closing", "volume"}
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
	commonData := readCommonCsv()
	csvCreationOneStockBrand(StockCode, commonData)

}
