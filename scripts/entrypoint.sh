#!/bin/sh
# entrypoint.sh

SENDPULSE_BIN=${SENDPULSE_BIN:-/bin/sendpulse}

echo "Initializing the database..."
$SENDPULSE_BIN db init

echo "Apply the migrations..."
$SENDPULSE_BIN db migrate
$SENDPULSE_BIN db status

if [ $? -eq 0 ]; then
    echo "Database initialized successfully."

    echo "Starting SendPulse server..."
    exec $SENDPULSE_BIN serve  # run HTTP server in the foreground
else
    echo "Database initialization failed. Exiting..."
    exit 1
fi