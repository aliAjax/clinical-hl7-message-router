#!/bin/sh
set -eu
base_url="${BASE_URL:-http://127.0.0.1:8084}"
curl -fsS "$base_url/healthz"
curl -fsS "$base_url/readyz"
curl -fsS -H 'Content-Type: application/json' -d '{"ID":"demo","Name":"Demo target","Address":"memory://demo"}' "$base_url/v1/targets"
curl -fsS -H 'Content-Type: application/json' -d '{"ID":"adt","Name":"ADT route","MessageType":"ADT","Trigger":"A01","TargetIDs":["demo"]}' "$base_url/v1/routes"
curl -fsS -H 'Content-Type: application/json' -d '{}' "$base_url/v1/routes/adt/validate"
curl -fsS -H 'Content-Type: application/json' -d '{}' "$base_url/v1/routes/adt/publish"
curl -fsS -H 'Content-Type: application/json' -d '{"raw":"MSH|^~\\\\&|LAB|HOSP|EHR|HOSP|20260821150000||ADT^A01|VERIFY-001|P|2.5\rPID|1||12345||DOE^JANE"}' "$base_url/v1/messages"
