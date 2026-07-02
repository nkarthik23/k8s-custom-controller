import time
import threading
from fastapi import FastAPI
from pydantic import BaseModel
from transformers import pipeline
from prometheus_client import Gauge, generate_latest, CONTENT_TYPE_LATEST
from starlette.responses import Response

app = FastAPI()

# Load model once at startup
print("Loading model...")
classifier = pipeline("sentiment-analysis", model="distilbert-base-uncased-finetuned-sst-2-english")
print("Model loaded.")

# Prometheus metric
queue_depth = Gauge("queue_depth", "Number of requests currently being processed")

# Track active requests
active_requests = 0
lock = threading.Lock()

class PredictRequest(BaseModel):
    text: str

@app.post("/predict")
def predict(request: PredictRequest):
    global active_requests

    with lock:
        active_requests += 1
        queue_depth.set(active_requests)

    try:
        # Simulate slight processing time so queue builds up under load
        time.sleep(2)
        result = classifier(request.text)
        return {"result": result}
    finally:
        with lock:
            active_requests -= 1
            queue_depth.set(active_requests)

@app.get("/metrics")
def metrics():
    return Response(generate_latest(), media_type=CONTENT_TYPE_LATEST)

@app.get("/healthz")
def health():
    return {"status": "ok"}