#!/bin/bash
set -e

APP_CONTAINER="test-readthenburn"
TEST_MESSAGE="Hello, this is a test message!"
APP_IP="localhost"
AUTH_HEADER="${AUTHHEADER_PASSWORD:-testauth123}"

# Wait for the application to be ready with connection check
echo "Waiting for application to be ready..."
MAX_RETRIES=30
RETRY_COUNT=0
while ! curl -s "http://$APP_IP:8080/ready" > /dev/null 2>&1; do
    if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
        fail "Timeout waiting for application to start"
    fi
    echo "Waiting... ($(($RETRY_COUNT + 1))/$MAX_RETRIES)"
    sleep 2
    RETRY_COUNT=$((RETRY_COUNT + 1))
done

echo "✅ Application is ready"

# Function to check if test failed
fail() {
    echo "❌ Test failed: $1"
    exit 1
}

# Create a burn message
echo "Creating burn message..."
RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -H "Authorization: ${AUTH_HEADER}" \
    -d "{\"message\":\"$TEST_MESSAGE\"}" \
    "http://$APP_IP:8080/")

echo "Response received: $RESPONSE"

MSG_ID=$(echo $RESPONSE | jq -r .msgId)

if [ -z "$MSG_ID" ]; then
    echo "Raw response: $RESPONSE"
    fail "Failed to create message"
fi

echo "✅ Successfully created message with ID: $MSG_ID"

# Read the message
echo "Reading message..."
RESPONSE=$(curl -s \
    -H "Authorization: ${AUTH_HEADER}" \
    "http://$APP_IP:8080/$MSG_ID")

RECEIVED_MSG=$(echo $RESPONSE | jq -r .burnMsg)

if [ "$RECEIVED_MSG" != "$TEST_MESSAGE" ]; then
    fail "Message content doesn't match. Expected '$TEST_MESSAGE', got '$RECEIVED_MSG'"
fi

echo "✅ Successfully read message"

# Try to read the message again (should be burned)
echo "Attempting to read burned message..."
RESPONSE=$(curl -s \
    -H "Authorization: ${AUTH_HEADER}" \
    "http://$APP_IP:8080/$MSG_ID")

BURNED_MSG=$(echo $RESPONSE | jq -r .burnMsg)
EXPECTED_BURNED_MSG="Message does not exist or has been burned already"

if [ "$BURNED_MSG" != "$EXPECTED_BURNED_MSG" ]; then
    fail "Message wasn't properly burned. Expected '$EXPECTED_BURNED_MSG', got '$BURNED_MSG'"
fi

echo "✅ Message was properly burned"
echo "✅ All tests passed successfully!"