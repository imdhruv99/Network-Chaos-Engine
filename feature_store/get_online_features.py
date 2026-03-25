from feast import FeatureStore

def fetch_realtime_inference_data():
    print("Connecting to Feast Feature Store...")
    store = FeatureStore(repo_path="./feature_repo")

    # The IP addresses currently hitting your application right now
    target_ips = [{"src_ip": "10.0.0.1"}, {"src_ip": "192.168.1.100"}]

    print(f"Fetching sub-millisecond features for {len(target_ips)} IPs...")

    # Query Redis for the absolute latest materialized features
    features = store.get_online_features(
        features=[
            "daily_ip_traffic:bytes_sent",
            "daily_ip_traffic:latency_ms",
        ],
        entity_rows=target_ips,
    ).to_dict()

    print("\n--- Real-Time Feature Vector for Inference ---")
    for i in range(len(target_ips)):
        print(f"IP: {target_ips[i]['src_ip']}")
        print(f" - Bytes Sent: {features['bytes_sent'][i]}")
        print(f" - Latency MS: {features['latency_ms'][i]}\n")

if __name__ == "__main__":
    fetch_realtime_inference_data()
