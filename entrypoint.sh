#!/bin/sh
set -e

echo "Running seeder..."
./seed

echo "Starting API server..."
exec ./server
