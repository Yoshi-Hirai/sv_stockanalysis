# sv_stockanalysis

株価予測モデルを形成するためのソース群

## 手順

- go run csvdata_create_main.go で予測モデルを生成するのに必要な CSV データを作成する
- 日本国内のみ。
  - 株探(https://kabutan.jp/)からデータをスクレイピング
  - 1 実行で単一銘柄のみ。銘柄コードは実行 go ファイルを直接編集する
  - RawData 以下に csv が出力される。出力場所も実行 go ファイルを直接編集する

## go ファイル説明

- verification_accounts.go
  - 移動平均、ボラティリティ、MADRate、RSI の値を検証する
