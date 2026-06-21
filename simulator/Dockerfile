FROM python:3.11-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY censer_simulator.py .

ENV CENSER_CODE=CENSER-001
ENV MOTION_PROFILE=walking
ENV INTERVAL=60
ENV MQTT_BROKER=
ENV MQTT_TOPIC=censer/sensor
ENV API_BASE=

ENTRYPOINT ["python", "censer_simulator.py"]
CMD ["--fast"]
