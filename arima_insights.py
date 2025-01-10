import sys
import json
import pandas as pd
from statsmodels.tsa.arima.model import ARIMA

def add_arima_predictions(file_path):
    # CSVファイルの読み込み
    df = pd.read_csv(file_path)

    # データの前処理
    df["date"] = pd.to_datetime(df["date"])  # 日付列をdatetime型に変換
    df.set_index("date", inplace=True)       # 日付をインデックスに設定
    df = df.sort_index()                     # 日付昇順にソート

    # インデックスに頻度を設定
    df = df.asfreq('D')  # 日次データの場合

    # 欠損値の補間 (必要に応じて)
    df["closing"] = df["closing"].interpolate()  # 線形補間で欠損値を埋める

    # 必要な列があるか確認
    if "closing" not in df.columns:
        raise ValueError("Error: 'closing' column not found in the data.")

    # ARIMA予測値と差分を格納する列を作成
    df["arima_prediction"] = None
    df["prediction_difference"] = None

    # 各日付に対して1日先の予測値を計算
    for i in range(5, len(df) - 1):  # 最後の日付は予測できないので -1
        # トレーニングデータ（現時点までのデータを使用）
        training_data = df.iloc[:i + 1]["closing"]

        # ARIMAモデルの適用
        model = ARIMA(training_data, order=(1, 1, 1))
        fitted_model = model.fit()

        # 次の1日先の予測値を計算
        forecast = fitted_model.forecast(steps=1)
        next_day_forecast = forecast.iloc[0]

        # 現在の日付に予測値を格納（次の日の実際の値との比較に使う）
        df.iloc[i + 1, df.columns.get_loc("arima_prediction")] = next_day_forecast

    # 実測値との差分を計算
    df["prediction_difference"] = df["closing"] - df["arima_prediction"]

    # 欠損値を含む行を削除
    # print(df.isna())  // 欠損値の真偽確認
    dfNonNA = df.dropna()

    return dfNonNA

# 実行(csvファイル名を引数として渡す)
if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python arima_insights.py <file_path>")
        sys.exit(1)

    file_path = sys.argv[1]
    result_df = add_arima_predictions(file_path)

    # JSON文字列に変換
    result_df = result_df.reset_index()  # インデックスをリセットして 'date' 列に戻す
    json_string = result_df.to_json(orient="records", date_format="iso")

    # 結果を表示
    print(json_string)