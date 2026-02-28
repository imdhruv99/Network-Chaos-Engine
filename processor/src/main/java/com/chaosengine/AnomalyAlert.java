package com.chaosengine;

public class AnomalyAlert {
    public String srcIp;
    public long totalBytes;
    public String message;

    public AnomalyAlert(String srcIp, long totalBytes, String message) {
        this.srcIp = srcIp;
        this.totalBytes = totalBytes;
        this.message = message;
    }
}
