FROM python:3.11-slim

WORKDIR /app

COPY cmd/inference-server/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Pre-download the model at build time so pods start fast
RUN python -c "from transformers import pipeline; pipeline('sentiment-analysis', model='distilbert-base-uncased-finetuned-sst-2-english')"

COPY cmd/inference-server/app.py .

EXPOSE 8080

CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8080"]