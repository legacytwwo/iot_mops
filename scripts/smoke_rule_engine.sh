#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if [[ -f gitops/.env ]]; then
  set -a
  # shellcheck disable=SC1091
  . gitops/.env
  set +a
fi

MONGO_USERNAME="${MONGO_USERNAME:-root}"
MONGO_PASSWORD="${MONGO_PASSWORD:-example}"
RABBITMQ_USERNAME="${RABBITMQ_USERNAME:-guest}"
RABBITMQ_PASSWORD="${RABBITMQ_PASSWORD:-guest}"
MONGO_DB="${MONGO_DB:-iot}"

RULE_ID="smoke-$(date +%s)-$$"
if command -v uuidgen >/dev/null 2>&1; then
  MESSAGE_ID="smoke-msg-$(uuidgen | tr '[:upper:]' '[:lower:]')"
else
  MESSAGE_ID="smoke-msg-$(date +%s)-$$"
fi
DEVICE_ID="smoke-device"

cleanup() {
  docker exec -i iot_mongo mongosh "mongodb://${MONGO_USERNAME}:${MONGO_PASSWORD}@localhost:27017/admin" --eval \
    "db.getSiblingDB(\"${MONGO_DB}\").rules.deleteOne({_id:\"${RULE_ID}\"}); db.getSiblingDB(\"${MONGO_DB}\").alerts.deleteMany({message_id:\"${MESSAGE_ID}\"});" >/dev/null 2>&1 || true
}
trap cleanup EXIT

make infra-up >/dev/null
make rule-up >/dev/null

for _ in $(seq 1 30); do
  if curl -fsS http://localhost:8080/healthz >/dev/null; then
    break
  fi
  sleep 1
done

if ! curl -fsS http://localhost:8080/healthz >/dev/null; then
  echo "rule_engine not healthy on :8080"
  exit 1
fi

docker exec -i iot_mongo mongosh "mongodb://${MONGO_USERNAME}:${MONGO_PASSWORD}@localhost:27017/admin" --eval \
  "db.getSiblingDB(\"${MONGO_DB}\").rules.insertOne({_id:\"${RULE_ID}\",name:\"smoke temp\",enabled:true,scope:{metrics:[\"temp\"],tags:{},device_ids:[]},condition:{metric:\"temp\",op:\">\",threshold:30},evaluation:{type:\"instant\"},action:{type:\"alert\",severity:\"high\",message:\"smoke\"},updated_at:new Date()})" >/dev/null

curl -sS -u "${RABBITMQ_USERNAME}:${RABBITMQ_PASSWORD}" -H 'content-type: application/json' \
  -X POST "http://localhost:15672/api/exchanges/%2F/telemetry/publish" \
  -d "{\"properties\":{},\"routing_key\":\"telemetry.envelopes\",\"payload\":\"{\\\"message_id\\\":\\\"${MESSAGE_ID}\\\",\\\"ts\\\":\\\"2026-01-05T12:00:00Z\\\",\\\"device\\\":{\\\"id\\\":\\\"${DEVICE_ID}\\\",\\\"type\\\":\\\"sensor\\\",\\\"location\\\":\\\"lab\\\"},\\\"metrics\\\":{\\\"temp\\\":{\\\"value\\\":35,\\\"unit\\\":\\\"C\\\"}}}\",\"payload_encoding\":\"string\"}" >/dev/null

sleep 1

COUNT="$(docker exec -i iot_mongo mongosh "mongodb://${MONGO_USERNAME}:${MONGO_PASSWORD}@localhost:27017/admin" --quiet --eval \
  "db.getSiblingDB(\"${MONGO_DB}\").alerts.countDocuments({message_id:\"${MESSAGE_ID}\"})")"

if [[ "${COUNT}" != "1" ]]; then
  echo "smoke failed: expected 1 alert, got ${COUNT}"
  exit 1
fi

echo "smoke ok"
