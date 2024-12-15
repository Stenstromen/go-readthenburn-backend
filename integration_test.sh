#!/bin/bash
set -e

APP_CONTAINER="test-readthenburn"
TEST_MESSAGE="Hello, this is a test message!"
APP_IP="localhost"

# Wait for the application to be ready
echo "ℹ️ Waiting for application to be ready..."
MAX_RETRIES=30
RETRY_COUNT=0
while ! curl -s "http://$APP_IP:8080/ready" > /dev/null 2>&1; do
    if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
        fail "Timeout waiting for application to start"
    fi
    echo "ℹ️ Waiting... ($(($RETRY_COUNT + 1))/$MAX_RETRIES)"
    sleep 2
    RETRY_COUNT=$((RETRY_COUNT + 1))
done

echo "✅ Application is ready"

# Function to check if test failed
fail() {
    echo "❌ Test failed: $1"
    exit 1
}

# Create and burn first message
echo "ℹ️ Creating first burn message..."
RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d "{\"message\":\"$TEST_MESSAGE\"}" \
    "http://$APP_IP:8080/")

MSG_ID=$(echo $RESPONSE | jq -r .msgId)

echo "ℹ️ Reading and burning first message..."
curl -s "http://$APP_IP:8080/$MSG_ID"

echo "ℹ️ Checking stats after first message..."
STATS_RESPONSE=$(curl -s "http://$APP_IP:8080/stats")
EXPECTED_STATS='{"totalMessages":1,"history":[{"date":"2024-03-01","totalMessages":1}]}'

if [ "$(echo $STATS_RESPONSE | jq -c .)" != "$(echo $EXPECTED_STATS | jq -c .)" ]; then
    echo "Expected: $EXPECTED_STATS"
    echo "Got: $STATS_RESPONSE"
    fail "Stats don't match after first message"
fi

echo "✅ First message stats verified"

# Create and burn second message
echo "ℹ️ Creating second burn message..."
RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d "{\"message\":\"$TEST_MESSAGE\"}" \
    "http://$APP_IP:8080/")

MSG_ID=$(echo $RESPONSE | jq -r .msgId)

echo "ℹ️ Reading and burning second message..."
curl -s "http://$APP_IP:8080/$MSG_ID"

echo "ℹ️ Checking stats after second message..."
STATS_RESPONSE=$(curl -s "http://$APP_IP:8080/stats")
EXPECTED_STATS='{"totalMessages":2,"history":[{"date":"2024-03-01","totalMessages":2}]}'

if [ "$(echo $STATS_RESPONSE | jq -c .)" != "$(echo $EXPECTED_STATS | jq -c .)" ]; then
    echo "Expected: $EXPECTED_STATS"
    echo "Got: $STATS_RESPONSE"
    fail "Stats don't match after second message"
fi

echo "✅ Second message stats verified"

# Simulate date change
echo "ℹ️ Simulating date change to April..."
podman stop "${APP_CONTAINER}"
podman rm "${APP_CONTAINER}"
podman run -d --name "${APP_CONTAINER}" \
    --network "testnetwork" \
    -p 8080:8080 \
    -e MYSQL_HOSTNAME="test-mariadb" \
    -e MYSQL_DATABASE=burndb \
    -e MYSQL_USERNAME=burnuser \
    -e MYSQL_PASSWORD=testpass123 \
    -e SECRET_KEY="7AE49A19B3C844BDB68E460D9224A5D0" \
    -e CURRENT_DATE="2024-04-01" \
    readthenburn

# Wait for app to be ready again
RETRY_COUNT=0
while ! curl -s "http://$APP_IP:8080/ready" > /dev/null 2>&1; do
    if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
        fail "Timeout waiting for application to restart"
    fi
    echo "ℹ️ Waiting for app restart... ($(($RETRY_COUNT + 1))/$MAX_RETRIES)"
    sleep 2
    RETRY_COUNT=$((RETRY_COUNT + 1))
done

# Create and burn third message
echo "ℹ️ Creating third burn message..."
RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d "{\"message\":\"$TEST_MESSAGE\"}" \
    "http://$APP_IP:8080/")

MSG_ID=$(echo $RESPONSE | jq -r .msgId)

echo "ℹ️ Reading and burning third message..."
curl -s "http://$APP_IP:8080/$MSG_ID"

echo "ℹ️ Checking stats after third message..."
STATS_RESPONSE=$(curl -s "http://$APP_IP:8080/stats")
EXPECTED_STATS='{"totalMessages":3,"history":[{"date":"2024-04-01","totalMessages":1},{"date":"2024-03-01","totalMessages":2}]}'

if [ "$(echo $STATS_RESPONSE | jq -c .)" != "$(echo $EXPECTED_STATS | jq -c .)" ]; then
    echo "Expected: $EXPECTED_STATS"
    echo "Got: $STATS_RESPONSE"
    fail "Stats don't match after third message"
fi

echo "✅ Third message stats verified"
echo "✅ All tests passed successfully!"