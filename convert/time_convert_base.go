// convert コンバートパッケージ
package convert // パッケージ名はディレクトリ名と同じにする

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

// ---- Global Variable

// ---- Package Global Variable

//---- public function ----

// 文字列からDate関数を用いてtime.Time型に変換する
// str Format [yyyy/mm/dd HH:MM:SS:NN] (HH:MM:SS:NNは省略可)
func ConvertStringToTime(str string) (time.Time, error) {

	var retValue time.Time
	var err error
	var month time.Month
	yearMonthDay := []int{0, 0, 0}
	hourMinuteSecondNanosecond := []int{0, 0, 0, 0}

	dateStr := str
	if strings.Contains(str, " ") {

		splitStr := strings.Split(str, " ")
		dateStr = splitStr[0]
		// hhmmssnnStr 0:Hour文字列 / 1:minute文字列 / 2:second文字列 / 3:nanosecond文字列
		hhmmssnnStr := strings.Split(splitStr[1], ":")
		for i, v := range hhmmssnnStr {
			hourMinuteSecondNanosecond[i], err = strconv.Atoi(v)
			if err != nil {
				slog.Info("error", "hourminutesecondnanosecond", err)
				return retValue, err
			}
		}
	}

	// yymmddStr 0:year文字列 / 1:month文字列 / 2:day文字列
	yymmddStr := strings.Split(dateStr, "/")
	if len(yymmddStr) != len(yearMonthDay) {
		// 引数文字列に3要素なければエラーとする
		slog.Info("error! Date component is shortage.", "dateStr", len(yymmddStr))
		err = fmt.Errorf("Date component is shortage. dateStr Lentgh=%d(string %s)", len(yymmddStr), dateStr)
		return retValue, err
	}
	for i, v := range yymmddStr {
		yearMonthDay[i], err = strconv.Atoi(v)
		if err != nil {
			slog.Info("error", "yearmonthday", err)
			return retValue, err
		}
	}
	month = time.Month(yearMonthDay[1])

	retValue = time.Date(yearMonthDay[0], month, yearMonthDay[2],
		hourMinuteSecondNanosecond[0], hourMinuteSecondNanosecond[1], hourMinuteSecondNanosecond[2], hourMinuteSecondNanosecond[3], time.Local)
	return retValue, err
}

//---- private function ----
