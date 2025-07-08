#!/bin/bash

# Load environment variables
set -a
[ -f .env ] && source .env
[ -f .env.local ] && source .env.local
set +a

# Start air with environment variables
air