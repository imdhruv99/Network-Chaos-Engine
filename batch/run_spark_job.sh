#!/bin/bash

echo "Submitting Medallion Pipeline to Spark Container..."

docker exec -i spark-jupyter spark-submit \
  --packages org.apache.hadoop:hadoop-aws:3.3.4,com.amazonaws:aws-java-sdk-bundle:1.12.262,io.delta:delta-spark_2.12:3.1.0 \
  --master local[*] \
  /home/jovyan/work/batch/medallion_lakehouse_pipeline.py

echo "Job execution finished at $(date)"
