#!/usr/bin/env bash
# End-to-end smoke test over the public gateway API. Requires: curl, jq, a running stack (make up).
set -euo pipefail

API=${API:-http://localhost:8080/api/v1}
GW=${GW:-http://localhost:8090}
PASS=password123
fail() { echo "FAIL: $*" >&2; exit 1; }
step() { echo "• $*"; }

login() { curl -fsS -X POST "$API/auth/login" -H 'Content-Type: application/json' -d "{\"email\":\"$1\",\"password\":\"$PASS\"}"; }
get()   { curl -fsS "$API$2" -H "Authorization: Bearer $1"; }
send()  { local tok=$1 method=$2 path=$3; shift 3; curl -fsS -X "$method" "$API$path" -H "Authorization: Bearer $tok" -H 'Content-Type: application/json' "$@"; }
code()  { local tok=$1 method=$2 path=$3; shift 3; curl -s -o /dev/null -w '%{http_code}' -X "$method" "$API$path" -H "Authorization: Bearer $tok" -H 'Content-Type: application/json' "$@"; }
expect() { # expect <http-status> <failure message> <code-args...>
  local want=$1 msg=$2; shift 2
  local got; got=$(code "$@")
  [ "$got" = "$want" ] || fail "$msg (expected $want, got $got)"
}
wait_for() { # wait_for <description> <command...>
  local d=$1; shift
  for _ in $(seq 1 30); do "$@" >/dev/null 2>&1 && return 0; sleep 1; done
  fail "timed out waiting for: $d"
}

step "wait for gateway"
wait_for "gateway" curl -fsS "${API%/api/v1}/healthz"
wait_for "seeded login" login staff@condo.local

STAFF=$(login staff@condo.local | jq -r .token)
R1_JSON=$(login resident1@condo.local);  R1=$(jq -r .token <<<"$R1_JSON");  R1_ROOM=$(jq -r .user.roomId <<<"$R1_JSON")
R4_JSON=$(login resident4@condo.local);  R4=$(jq -r .token <<<"$R4_JSON")   # building A, floor 2
R7_JSON=$(login resident7@condo.local);  R7=$(jq -r .token <<<"$R7_JSON")   # building B
BLD_A=$(jq -r .user.buildingId <<<"$R1_JSON")
[ "$BLD_A" != "$(jq -r .user.buildingId <<<"$R7_JSON")" ] || fail "residents 1 and 7 should be in different buildings"

step "auth: no token / wrong role are rejected"
[ "$(curl -s -o /dev/null -w '%{http_code}' "$API/parcels")" = 401 ] || fail "expected 401 without token"
expect 403 "resident must not list all parcels" "$R1" GET /parcels
expect 403 "resident must not publish announcements" "$R1" POST /announcements -d '{}'
expect 404 "internal route must not be exposed" "$STAFF" POST /internal/payments/callback -d '{}'

step "use case 1: parcel management"
P=$(curl -fsS -X POST "$API/parcels" -H "Authorization: Bearer $STAFF" -F "roomId=$R1_ROOM" -F carrier=Kerry -F trackingNo=TH123 | jq -r .id)
[ -n "$P" ] && [ "$P" != null ] || fail "parcel not created"
get "$STAFF" "/parcels?status=WAITING" | jq -e --arg p "$P" 'map(.id)|index($p)!=null' >/dev/null || fail "parcel not in staff pending list"
get "$R1" /parcels/me | jq -e --arg p "$P" 'map(select(.id==$p and .status=="WAITING"))|length==1' >/dev/null || fail "resident does not see parcel"
wait_for "resident notification for parcel" bash -c "curl -fsS '$API/notifications' -H 'Authorization: Bearer $R1' | jq -e 'map(select(.type==\"parcel.received\"))|length>=1'"
expect 404 "another room's resident must not pick up the parcel" "$R4" POST "/parcels/$P/pickup"
send "$R1" POST "/parcels/$P/pickup" | jq -e '.status=="PICKED_UP"' >/dev/null || fail "pickup failed"
expect 409 "second pickup should conflict" "$R1" POST "/parcels/$P/pickup"
wait_for "staff notification for pickup" bash -c "curl -fsS '$API/notifications' -H 'Authorization: Bearer $STAFF' | jq -e 'map(select(.type==\"parcel.accepted\"))|length>=1'"

step "use case 3: announcements (REST -> gRPC), targeted at building A"
A=$(send "$STAFF" POST /announcements -d "{\"title\":\"Lift maintenance\",\"body\":\"Lift A closed Sat\",\"category\":\"MAINTENANCE\",\"targetType\":\"BUILDING\",\"targetBuildingId\":\"$BLD_A\"}" | jq -r .id)
[ -n "$A" ] && [ "$A" != null ] || fail "announcement not created"
expect 400 "invalid target should be 400" "$STAFF" POST /announcements -d '{"title":"x","body":"y","targetType":"FLOOR"}'
get "$R1" /announcements | jq -e --arg a "$A" 'map(.id)|index($a)!=null' >/dev/null || fail "building A resident should see announcement"
get "$R7" /announcements | jq -e --arg a "$A" 'map(.id)|index($a)==null' >/dev/null || fail "building B resident must not see announcement"
expect 404 "building B resident must not read announcement" "$R7" GET "/announcements/$A"
send "$R1" POST "/announcements/$A/read" >/dev/null
get "$R1" "/announcements/$A" | jq -e '.read==true' >/dev/null || fail "read flag not set"
wait_for "announcement notification" bash -c "curl -fsS '$API/notifications' -H 'Authorization: Bearer $R1' | jq -e 'map(select(.type==\"announcement.published\"))|length>=1'"
[ "$(curl -fsS "$API/notifications" -H "Authorization: Bearer $R7" | jq 'map(select(.type=="announcement.published"))|length')" = 0 ] || fail "building B must not be notified"
send "$STAFF" PATCH "/announcements/$A" -d "{\"title\":\"Lift maintenance (updated)\",\"body\":\"Lift A closed Sat\",\"targetType\":\"BUILDING\",\"targetBuildingId\":\"$BLD_A\"}" | jq -e '.title|test("updated")' >/dev/null || fail "update failed"
send "$STAFF" POST "/announcements/$A/archive" | jq -e '.status=="ARCHIVED"' >/dev/null || fail "archive failed"
get "$R1" /announcements | jq -e --arg a "$A" 'map(.id)|index($a)==null' >/dev/null || fail "archived announcement still visible"

step "use case 2: utility billing & payment"
CYCLE=$(date +%Y-%m)
DUE=$(date -v+14d +%F 2>/dev/null || date -d '+14 days' +%F)
# use a unique cycle per run so the smoke test is repeatable
CYCLE=$(printf '%04d-%02d' $((2100 + RANDOM % 900)) $((1 + RANDOM % 12)))
B=$(send "$STAFF" POST /utilities/bills -d "{\"roomId\":\"$R1_ROOM\",\"cycle\":\"$CYCLE\",\"waterUnits\":10,\"electricUnits\":100,\"dueDate\":\"$DUE\"}")
BILL=$(jq -r .id <<<"$B"); jq -e '.total==880 and .status=="UNPAID"' <<<"$B" >/dev/null || fail "unexpected bill: $B"
expect 409 "duplicate bill should conflict" "$STAFF" POST /utilities/bills -d "{\"roomId\":\"$R1_ROOM\",\"cycle\":\"$CYCLE\",\"dueDate\":\"$DUE\"}"
get "$R1" "/utilities/bills/me?status=UNPAID" | jq -e --arg b "$BILL" 'map(.id)|index($b)!=null' >/dev/null || fail "resident does not see bill"
get "$R4" /utilities/bills/me | jq -e --arg b "$BILL" 'map(.id)|index($b)==null' >/dev/null || fail "other room sees bill"
expect 404 "other room must not read bill" "$R4" GET "/utilities/bills/$BILL"
wait_for "bill notification" bash -c "curl -fsS '$API/notifications' -H 'Authorization: Bearer $R1' | jq -e 'map(select(.type==\"billing.bill_issued\"))|length>=1'"
expect 404 "no receipt before payment" "$R1" GET "/payments/$BILL/receipt"

PAY=$(send "$R1" POST /payments -d "{\"billId\":\"$BILL\"}")
CHECKOUT=$(jq -r .checkoutUrl <<<"$PAY"); REF=$(jq -r .paymentReference <<<"$PAY")
get "$R1" "/utilities/bills/$BILL" | jq -e '.status=="PENDING"' >/dev/null || fail "bill should be PENDING after starting payment"

step "  forged webhook is rejected by the payment adapter"
FORGED=$(docker compose exec -T payment-adapter wget -qO- --header='X-Timestamp: 1' --header='X-Signature: bad' --post-data='{"reference":"x","status":"SUCCEEDED"}' http://localhost:8085/webhooks/gateway 2>&1 || true)
grep -q "401" <<<"$FORGED" || fail "forged webhook was not rejected: $FORGED"

step "  complete payment at the mock gateway"
curl -fsS -o /dev/null -X POST "$GW/checkout/$REF/complete" -d result=success
wait_for "bill PAID" bash -c "curl -fsS '$API/utilities/bills/$BILL' -H 'Authorization: Bearer $R1' | jq -e '.status==\"PAID\"'"
get "$R1" "/payments/$BILL/receipt" | jq -e --arg r "$REF" '.paymentReference==$r and .amount==880 and (.receiptNo|startswith("RC-"))' >/dev/null || fail "receipt invalid"
# replayed webhook must be idempotent
curl -fsS -o /dev/null -X POST "$GW/checkout/$REF/complete" -d result=success
get "$R1" "/utilities/bills/$BILL" | jq -e '.status=="PAID"' >/dev/null || fail "replay changed state"
expect 409 "paying a paid bill should conflict" "$R1" POST /payments -d "{\"billId\":\"$BILL\"}"
wait_for "payment notification" bash -c "curl -fsS '$API/notifications' -H 'Authorization: Bearer $R1' | jq -e 'map(select(.type==\"billing.payment_completed\"))|length>=1'"

step "  failed payment returns the bill to UNPAID"
CYCLE2=$(printf '%04d-%02d' $((2100 + RANDOM % 900)) $((1 + RANDOM % 12)))
BILL2=$(send "$STAFF" POST /utilities/bills -d "{\"roomId\":\"$R1_ROOM\",\"cycle\":\"$CYCLE2\",\"waterUnits\":1,\"electricUnits\":1,\"dueDate\":\"$DUE\"}" | jq -r .id)
REF2=$(send "$R1" POST /payments -d "{\"billId\":\"$BILL2\"}" | jq -r .paymentReference)
curl -fsS -o /dev/null -X POST "$GW/checkout/$REF2/complete" -d result=fail
wait_for "bill back to UNPAID" bash -c "curl -fsS '$API/utilities/bills/$BILL2' -H 'Authorization: Bearer $R1' | jq -e '.status==\"UNPAID\"'"

echo "ALL SMOKE CHECKS PASSED"
