#!/bin/sh
# Run this once to generate go.sum before first docker-compose up
# or let the Dockerfile handle it
go mod tidy
