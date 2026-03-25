import pandas as pd
from datetime import datetime, timedelta
from feast import FeatureStore

def fetch_historical_training_data():
    print("Connecting to Feast Feature Store...")
    store = FeatureStore(repo_path="./feature_repo")

    # Table of Historical events
    # We set the timestamps slightly in the future to ensure we capture the recent Parquet data you just generated.
    entity_df = pd.DataFrame(
        {
            "event_timestamp": [
                pd.Timestamp.utcnow() - timedelta(minutes=15),
                pd.Timestamp.utcnow() - timedelta(minutes=30),
            ],
            "src_ip": ["10.0.0.5", "192.168.1.100"],
            "label_is_anomaly": [1, 0] # 1 = Attack, 0 = Normal
        }
    )

    print("Executing Time-Travel Join for historical features...")
    # Fetch the features for exactly those timestamps
    training_df = store.get_historical_features(
        entity_df=entity_df,
        features=[
            "daily_ip_traffic:bytes_sent",
            "daily_ip_traffic:latency_ms",
        ],
    ).to_df()

    print("\n--- Training Dataset Generated ---")
    print(training_df.to_markdown())

if __name__ == "__main__":
    fetch_historical_training_data()
