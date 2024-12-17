package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
)

// Helper function to calculate moving average
func movingAverage(values []float64, window int) []float64 {
	result := make([]float64, len(values))
	for i := range values {
		if i < window-1 {
			result[i] = math.NaN()
		} else {
			sum := 0.0
			for j := 0; j < window; j++ {
				sum += values[i-j]
			}
			result[i] = sum / float64(window)
		}
	}
	return result
}

// Helper function to calculate volatility (standard deviation)
func volatility(values []float64, window int) []float64 {
	result := make([]float64, len(values))
	for i := range values {
		if i < window-1 {
			result[i] = math.NaN()
		} else {
			mean := 0.0
			for j := 0; j < window; j++ {
				mean += values[i-j]
			}
			mean /= float64(window)

			variance := 0.0
			for j := 0; j < window; j++ {
				diff := values[i-j] - mean
				variance += diff * diff
			}
			result[i] = math.Sqrt(variance / float64(window))
		}
	}
	return result
}

// Helper function to calculate MAD rate
func madRate(values, movingAvg []float64) []float64 {
	result := make([]float64, len(values))
	for i := range values {
		if math.IsNaN(movingAvg[i]) {
			result[i] = math.NaN()
		} else {
			result[i] = (values[i] - movingAvg[i]) / movingAvg[i] * 100
		}
	}
	return result
}

// Helper function to calculate RSI
func rsi(values []float64, window int) []float64 {
	result := make([]float64, len(values))
	gain := make([]float64, len(values))
	loss := make([]float64, len(values))

	for i := 1; i < len(values); i++ {
		change := values[i] - values[i-1]
		if change > 0 {
			gain[i] = change
			loss[i] = 0
		} else {
			gain[i] = 0
			loss[i] = -change
		}
	}

	for i := range values {
		if i < window {
			result[i] = math.NaN()
		} else {
			avgGain := 0.0
			avgLoss := 0.0
			for j := 0; j < window; j++ {
				avgGain += gain[i-j]
				avgLoss += loss[i-j]
			}
			avgGain /= float64(window)
			avgLoss /= float64(window)

			if avgLoss == 0 {
				result[i] = 100
			} else {
				rs := avgGain / avgLoss
				result[i] = 100 - (100 / (1 + rs))
			}
		}
	}
	return result
}

func main() {
	// Open the CSV file
	file, err := os.Open("RawData/8113.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Read the CSV data
	reader := csv.NewReader(file)
	data, err := reader.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	// Extract header and rows
	header := data[0]
	rows := data[1:]

	// データを昇順に並べ替え（date列を基準）
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i][0] < rows[j][0] // 日付が昇順になるよう比較
	})

	// Extract Closing prices
	var closingPrices []float64
	var originalMovingAve5, originalMovingAve14, originalMovingAve30 []float64
	var originalVolatility5, originalVolatility14, originalVolatility30 []float64
	var originalMADRate5, originalMADRate14, originalMADRate30 []float64
	var originalRSI5, originalRSI14, originalRSI30 []float64
	for _, row := range rows {
		price, err := strconv.ParseFloat(row[4], 64) // Assuming "Closing" is the 5th column
		if err != nil {
			closingPrices = append(closingPrices, math.NaN())
		} else {
			closingPrices = append(closingPrices, price)
		}
		// originalMovingAve
		price, err = strconv.ParseFloat(row[6], 64)
		if err != nil {
			originalMovingAve5 = append(originalMovingAve5, math.NaN())
		} else {
			originalMovingAve5 = append(originalMovingAve5, price)
		}
		price, err = strconv.ParseFloat(row[7], 64)
		if err != nil {
			originalMovingAve14 = append(originalMovingAve14, math.NaN())
		} else {
			originalMovingAve14 = append(originalMovingAve14, price)
		}
		price, err = strconv.ParseFloat(row[8], 64)
		if err != nil {
			originalMovingAve30 = append(originalMovingAve30, math.NaN())
		} else {
			originalMovingAve30 = append(originalMovingAve30, price)
		}
		// volatility
		price, err = strconv.ParseFloat(row[9], 64)
		if err != nil {
			originalVolatility5 = append(originalVolatility5, math.NaN())
		} else {
			originalVolatility5 = append(originalVolatility5, price)
		}
		price, err = strconv.ParseFloat(row[10], 64)
		if err != nil {
			originalVolatility14 = append(originalVolatility14, math.NaN())
		} else {
			originalVolatility14 = append(originalVolatility14, price)
		}
		price, err = strconv.ParseFloat(row[11], 64)
		if err != nil {
			originalVolatility30 = append(originalVolatility30, math.NaN())
		} else {
			originalVolatility30 = append(originalVolatility30, price)
		}
		// MADRate
		price, err = strconv.ParseFloat(row[12], 64)
		if err != nil {
			originalMADRate5 = append(originalMADRate5, math.NaN())
		} else {
			originalMADRate5 = append(originalMADRate5, price)
		}
		price, err = strconv.ParseFloat(row[13], 64)
		if err != nil {
			originalMADRate14 = append(originalMADRate14, math.NaN())
		} else {
			originalMADRate14 = append(originalMADRate14, price)
		}
		price, err = strconv.ParseFloat(row[14], 64)
		if err != nil {
			originalMADRate30 = append(originalMADRate30, math.NaN())
		} else {
			originalMADRate30 = append(originalMADRate30, price)
		}
		// RSI
		price, err = strconv.ParseFloat(row[15], 64)
		if err != nil {
			originalRSI5 = append(originalRSI5, math.NaN())
		} else {
			originalRSI5 = append(originalRSI5, price)
		}
		price, err = strconv.ParseFloat(row[16], 64)
		if err != nil {
			originalRSI14 = append(originalRSI14, math.NaN())
		} else {
			originalRSI14 = append(originalRSI14, price)
		}
		price, err = strconv.ParseFloat(row[17], 64)
		if err != nil {
			originalRSI30 = append(originalRSI30, math.NaN())
		} else {
			originalRSI30 = append(originalRSI30, price)
		}
	}

	/*
		// Closing 確認
		for i, c := range closingPrices {
			fmt.Println("price ", i, " ", c)
		}
	*/

	// Calculate metrics
	movingAve5 := movingAverage(closingPrices, 5)
	movingAve14 := movingAverage(closingPrices, 14)
	movingAve30 := movingAverage(closingPrices, 30)
	volatility5 := volatility(closingPrices, 5)
	volatility14 := volatility(closingPrices, 14)
	volatility30 := volatility(closingPrices, 30)
	madRate5 := madRate(closingPrices, movingAve5)
	madRate14 := madRate(closingPrices, movingAve14)
	madRate30 := madRate(closingPrices, movingAve30)
	rsi5 := rsi(closingPrices, 5)
	rsi14 := rsi(closingPrices, 14)
	rsi30 := rsi(closingPrices, 30)

	// 差分を計算
	diffMovingAve5 := make([]float64, len(movingAve5))
	diffMovingAve14 := make([]float64, len(movingAve14))
	diffMovingAve30 := make([]float64, len(movingAve30))
	diffVolatility5 := make([]float64, len(volatility5))
	diffVolatility14 := make([]float64, len(volatility14))
	diffVolatility30 := make([]float64, len(volatility30))
	diffMADRate5 := make([]float64, len(madRate5))
	diffMADRate14 := make([]float64, len(madRate14))
	diffMADRate30 := make([]float64, len(madRate30))
	diffRSI5 := make([]float64, len(rsi5))
	diffRSI14 := make([]float64, len(rsi14))
	diffRSI30 := make([]float64, len(rsi30))
	// movingave
	for i := range movingAve5 {
		if math.IsNaN(movingAve5[i]) || math.IsNaN(originalMovingAve5[i]) {
			diffMovingAve5[i] = math.NaN()
		} else {
			diffMovingAve5[i] = movingAve5[i] - originalMovingAve5[i]
		}
	}
	for i := range movingAve14 {
		if math.IsNaN(movingAve14[i]) || math.IsNaN(originalMovingAve14[i]) {
			diffMovingAve14[i] = math.NaN()
		} else {
			diffMovingAve14[i] = movingAve14[i] - originalMovingAve14[i]
		}
	}
	for i := range movingAve30 {
		if math.IsNaN(movingAve30[i]) || math.IsNaN(originalMovingAve30[i]) {
			diffMovingAve30[i] = math.NaN()
		} else {
			diffMovingAve30[i] = movingAve30[i] - originalMovingAve30[i]
		}
	}
	// volatility
	for i := range volatility5 {
		if math.IsNaN(volatility5[i]) || math.IsNaN(originalVolatility5[i]) {
			diffVolatility5[i] = math.NaN()
		} else {
			diffVolatility5[i] = volatility5[i] - originalVolatility5[i]
		}
	}
	for i := range volatility14 {
		if math.IsNaN(volatility14[i]) || math.IsNaN(originalVolatility14[i]) {
			diffVolatility14[i] = math.NaN()
		} else {
			diffVolatility14[i] = volatility14[i] - originalVolatility14[i]
		}
	}
	for i := range volatility30 {
		if math.IsNaN(volatility30[i]) || math.IsNaN(originalVolatility30[i]) {
			diffVolatility30[i] = math.NaN()
		} else {
			diffVolatility30[i] = volatility30[i] - originalVolatility30[i]
		}
	}
	// MADRate
	for i := range madRate5 {
		if math.IsNaN(madRate5[i]) || math.IsNaN(originalMADRate5[i]) {
			diffMADRate5[i] = math.NaN()
		} else {
			diffMADRate5[i] = madRate5[i] - originalMADRate5[i]
		}
	}
	for i := range madRate14 {
		if math.IsNaN(madRate14[i]) || math.IsNaN(originalMADRate14[i]) {
			diffMADRate14[i] = math.NaN()
		} else {
			diffMADRate14[i] = madRate14[i] - originalMADRate14[i]
		}
	}
	for i := range madRate30 {
		if math.IsNaN(madRate30[i]) || math.IsNaN(originalMADRate30[i]) {
			diffMADRate30[i] = math.NaN()
		} else {
			diffMADRate30[i] = madRate30[i] - originalMADRate30[i]
		}
	}
	// RSI
	for i := range rsi5 {
		if math.IsNaN(rsi5[i]) || math.IsNaN(originalRSI5[i]) {
			diffRSI5[i] = math.NaN()
		} else {
			diffRSI5[i] = rsi5[i] - originalRSI5[i]
		}
	}
	for i := range rsi14 {
		if math.IsNaN(rsi14[i]) || math.IsNaN(originalRSI14[i]) {
			diffRSI14[i] = math.NaN()
		} else {
			diffRSI14[i] = rsi14[i] - originalRSI14[i]
		}
	}
	for i := range rsi30 {
		if math.IsNaN(rsi30[i]) || math.IsNaN(originalRSI30[i]) {
			diffRSI30[i] = math.NaN()
		} else {
			diffRSI30[i] = rsi30[i] - originalRSI30[i]
		}
	}

	diffHeaders := []string{"Diff_MovingAve5", "Diff_MovingAve14", "Diff_MovingAve30",
		"Diff_Volatility5", "Diff_Volatility14", "Diff_Volatility30",
		"Diff_MADRate5", "Diff_MADRate14", "Diff_MADRate30",
		"Diff_RSI5", "Diff_RSI14", "Diff_RSI30"}

	// Create output CSV
	outputFile, err := os.Create("RawData/verify.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer outputFile.Close()

	writer := csv.NewWriter(outputFile)
	defer writer.Flush()

	// Write header
	newHeader := append(header, "Recalc_MovingAve5", "Recalc_MovingAve14", "Recalc_MovingAve30",
		"Recalc_Volatility5", "Recalc_Volatility14", "Recalc_Volatility30",
		"Recalc_MADRate5", "Recalc_MADRate14", "Recalc_MADRate30",
		"Recalc_RSI5", "Recalc_RSI14", "Recalc_RSI30")
	newHeader = append(newHeader, diffHeaders...)
	writer.Write(newHeader)

	// Write rows
	for i, row := range rows {
		newRow := append(row,
			fmt.Sprintf("%.2f", movingAve5[i]),
			fmt.Sprintf("%.2f", movingAve14[i]),
			fmt.Sprintf("%.2f", movingAve30[i]),
			fmt.Sprintf("%.2f", volatility5[i]),
			fmt.Sprintf("%.2f", volatility14[i]),
			fmt.Sprintf("%.2f", volatility30[i]),
			fmt.Sprintf("%.2f", madRate5[i]),
			fmt.Sprintf("%.2f", madRate14[i]),
			fmt.Sprintf("%.2f", madRate30[i]),
			fmt.Sprintf("%.2f", rsi5[i]),
			fmt.Sprintf("%.2f", rsi14[i]),
			fmt.Sprintf("%.2f", rsi30[i]),
		)
		// 差分の列を追加
		// movingave
		newRow = append(newRow, fmt.Sprintf("%.2f", diffMovingAve5[i]))
		newRow = append(newRow, fmt.Sprintf("%.2f", diffMovingAve14[i]))
		newRow = append(newRow, fmt.Sprintf("%.2f", diffMovingAve30[i]))
		// volatility
		newRow = append(newRow, fmt.Sprintf("%.2f", diffVolatility5[i]))
		newRow = append(newRow, fmt.Sprintf("%.2f", diffVolatility14[i]))
		newRow = append(newRow, fmt.Sprintf("%.2f", diffVolatility30[i]))
		// MADRate
		newRow = append(newRow, fmt.Sprintf("%.2f", diffMADRate5[i]))
		newRow = append(newRow, fmt.Sprintf("%.2f", diffMADRate14[i]))
		newRow = append(newRow, fmt.Sprintf("%.2f", diffMADRate30[i]))
		// RSI
		newRow = append(newRow, fmt.Sprintf("%.2f", diffRSI5[i]))
		newRow = append(newRow, fmt.Sprintf("%.2f", diffRSI14[i]))
		newRow = append(newRow, fmt.Sprintf("%.2f", diffRSI30[i]))
		writer.Write(newRow)
	}

	fmt.Println("Recalculation complete. Output saved to output.csv.")
}
