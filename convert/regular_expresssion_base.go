// convert コンバートパッケージ
package convert // パッケージ名はディレクトリ名と同じにする

import (
	"log/slog"
	"regexp"
	"strconv"
)

// ---- Global Variable

// ---- Package Global Variable

//---- public function ----

// 文字列から数字(int64)を抜き出す
func ExtractInt64(str string) int64 {
	rex := regexp.MustCompile("[0-9]+")
	str = rex.FindString(str)
	intVal, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		slog.String("error", err.Error())
		return 0
	}
	return intVal
}

// 文字列から数字(int32)を抜き出す
// int64 を int32 に型落ちしているので使用時は注意。
func ExtractInt32(str string) int {
	rex := regexp.MustCompile("[0-9]+")
	str = rex.FindString(str)
	intVal, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		slog.String("error", err.Error())
		return 0
	}
	return int(intVal)
}

//---- private function ----
