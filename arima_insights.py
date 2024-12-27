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
    df["Closing"] = df["Closing"].interpolate()  # 線形補間で欠損値を埋める

    # 必要な列があるか確認
    if "Closing" not in df.columns:
        raise ValueError("Error: 'Closing' column not found in the data.")

    # ARIMA予測値と差分を格納する列を作成
    df["ARIMA_Prediction"] = None
    df["Prediction_Difference"] = None

    # 各日付に対して1日先の予測値を計算
    for i in range(5, len(df) - 1):  # 最後の日付は予測できないので -1
        # トレーニングデータ（現時点までのデータを使用）
        training_data = df.iloc[:i + 1]["Closing"]

        # ARIMAモデルの適用
        model = ARIMA(training_data, order=(1, 1, 1))
        fitted_model = model.fit()

        # 次の1日先の予測値を計算
        forecast = fitted_model.forecast(steps=1)
        next_day_forecast = forecast.iloc[0]

        # 現在の日付に予測値を格納（次の日の実際の値との比較に使う）
        df.iloc[i + 1, df.columns.get_loc("ARIMA_Prediction")] = next_day_forecast

    # 実測値との差分を計算
    df["Prediction_Difference"] = df["Closing"] - df["ARIMA_Prediction"]

    return df

# 実行例
file_path = "Resource/0970/RawData.csv"  # 該当CSVファイルのパス
result_df = add_arima_predictions(file_path)

# 結果を表示
print(result_df)
