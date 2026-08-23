#!/bin/sh
set -eu

base="${STORE_URL:-http://localhost:8080}"
cookie="$(mktemp)"
trap 'rm -f "$cookie"' EXIT

curl -fsS -X POST "$base/signup" -H 'Content-Type: application/json' -d '{"email":"buyer@example.com","password":"correct-horse-battery-staple","name":"Buyer","captcha_token":"replace-with-browser-token","request_id":"signup-demo-1"}'
curl -fsS -X POST "$base/login" -H 'Content-Type: application/json' -c "$cookie" -d '{"email":"buyer@example.com","request_id":"login-demo-1"}'
curl -fsS -X POST "$base/checkout" -H 'Content-Type: application/json' -b "$cookie" -d '{"order_id":"order-demo-1","sku":"mug-black","quantity":2}'
curl -fsS -X POST "$base/orders/order-demo-1/fulfill" -b "$cookie"
curl -fsS -X GET "$base/orders" -b "$cookie"
