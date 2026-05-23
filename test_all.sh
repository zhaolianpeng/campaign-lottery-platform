#!/bin/bash
# ============================================================
# BOX·MAGIC 一键联调测试脚本
# 在 Mac 上运行: bash test_all.sh
# ============================================================
set -e

PROJECT_DIR="/Users/zhaolianpeng/code/Goproject/src/gitlab.aiforward.cn/campaign-lottery-platform"
BACKEND_DIR="$PROJECT_DIR/backend"
BIN_DIR="$PROJECT_DIR/bin"
SERVER_PORT=18100

echo "=========================================="
echo "  BOX·MAGIC 全系统联调测试"
echo "=========================================="

# Step 1: Pull latest code
echo ""
echo "📥 [1/6] 拉取最新代码..."
cd "$PROJECT_DIR" 2>/dev/null || { echo "❌ 项目目录不存在: $PROJECT_DIR"; exit 1; }
git pull origin main 2>&1 || echo "⚠️ git pull 失败，继续使用本地代码"

# Step 2: Build
echo ""
echo "🔨 [2/6] 编译后端..."
cd "$BACKEND_DIR"
mkdir -p "$BIN_DIR"
go build -o "$BIN_DIR/campaign-lottery" ./cmd/server 2>&1
echo "✅ 编译成功！"

# Step 3: Check memory files split
echo ""
echo "📁 [3/6] 检查代码结构..."
MEMORY_FILES=$(ls -1 "$BACKEND_DIR/internal/store/memory_"*.go 2>/dev/null | wc -l)
echo "  MemoryStore: $MEMORY_FILES 个领域文件"
if [ "$MEMORY_FILES" -lt 10 ]; then
  echo "  ⚠️  domain files less than expected"
fi

# Step 4: Start server in background
echo ""
echo "🚀 [4/6] 启动后端服务 (端口 $SERVER_PORT)..."
# Kill any existing process on the port
lsof -ti:$SERVER_PORT 2>/dev/null | xargs kill -9 2>/dev/null || true
sleep 1
"$BIN_DIR/campaign-lottery" -port $SERVER_PORT &
SERVER_PID=$!
sleep 2

# Check if server started
if kill -0 $SERVER_PID 2>/dev/null; then
  echo "✅ 服务已启动 (PID: $SERVER_PID)"
else
  echo "❌ 服务启动失败，检查输出"
  "$BIN_DIR/campaign-lottery" -port $SERVER_PORT 2>&1 | head -20
  exit 1
fi

# Step 5: API 测试
echo ""
echo "🧪 [5/6] API 功能测试..."

BASE="http://localhost:$SERVER_PORT"
PASS=0
FAIL=0

test_api() {
  local desc="$1"
  local method="$2"
  local url="$3"
  local data="$4"
  local expect="${5:-200}"
  
  if [ -n "$data" ]; then
    response=$(curl -s -o /dev/null -w "%{http_code}" -X "$method" "$BASE$url" -H "Content-Type: application/json" -d "$data" 2>/dev/null || echo "000")
  else
    response=$(curl -s -o /dev/null -w "%{http_code}" -X "$method" "$BASE$url" 2>/dev/null || echo "000")
  fi
  
  if [ "$response" = "$expect" ] || [ "${response:0:1}" = "${expect:0:1}" ]; then
    echo "  ✅ $desc → $response"
    PASS=$((PASS+1))
  else
    echo "  ❌ $desc → 期望 $expect 实际 $response"
    FAIL=$((FAIL+1))
  fi
}

echo "--- 基础功能 ---"
test_api "登录" "POST" "/api/v1/auth/guest-login" '{"nickname":"测试用户"}'
# Extract token
TOKEN=$(curl -s -X POST "$BASE/api/v1/auth/guest-login" -H "Content-Type: application/json" -d '{"nickname":"测试用户"}' 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('token',''))" 2>/dev/null || echo "")
echo "  Token: ${TOKEN:0:20}..."

if [ -n "$TOKEN" ]; then
  AUTH="Authorization: Bearer $TOKEN"
  
  test_api "系列列表" "GET" "/api/v1/blindbox/campaigns" ""
  test_api "单抽" "POST" "/api/v1/blindbox/draw" '{"campaign_id":"camp_launch_001","draw_count":1}'
  test_api "我的库存" "GET" "/api/v1/blindbox/inventory" ""
  test_api "签到" "POST" "/api/v1/blindbox/checkin" ""
  test_api "摇盒提示" "GET" "/api/v1/blindbox/hint/camp_launch_001" ""
  test_api "排行榜" "GET" "/api/v1/blindbox/leaderboard" ""
  
  echo "--- 会员/月卡 ---"
  test_api "会员信息" "GET" "/api/v1/blindbox/member" ""
  test_api "月卡状态" "GET" "/api/v1/month-card/status" ""
  
  echo "--- 商店 ---"
  test_api "商店列表" "GET" "/api/v1/shop/items" ""
  test_api "首充礼包" "GET" "/api/v1/first-recharge/packs" ""
  test_api "首充状态" "GET" "/api/v1/first-recharge/status" ""
  
  echo "--- 合成 ---"
  test_api "合成" "POST" "/api/v1/blindbox/blend" '{"source_prize_id":"","campaign_id":"camp_launch_001"}'
  
  echo "--- 社交裂变 ---"
  test_api "分享奖励" "POST" "/api/v1/blindbox/share-reward" ""
  test_api "邀请链接" "POST" "/api/v1/share/invite" ""
  test_api "邀请记录" "GET" "/api/v1/share/invitees" ""
  test_api "助力进度" "GET" "/api/v1/share/assist-progress" ""
  test_api "用户队伍" "GET" "/api/v1/team/my" ""
  test_api "待收礼物" "GET" "/api/v1/share/gifts/incoming" ""
  test_api "已送礼物" "GET" "/api/v1/share/gifts/sent" ""
  
  echo "--- v1.6 碎片拼图 ---"
  test_api "拼图模板" "GET" "/api/v1/puzzle/templates" ""
  test_api "我的拼图" "GET" "/api/v1/puzzle/my" ""
  test_api "抢购列表" "GET" "/api/v1/flash/list" ""
  
  echo "--- v2.0 活动系统 ---"
  test_api "活动列表" "GET" "/api/v1/activities" ""
  # Try to get activities detail
  ACTIVITY_ID=$(curl -s "$BASE/api/v1/activities" -H "$AUTH" 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('data',[]); print(items[0]['activity']['id'] if items else '')" 2>/dev/null || echo "")
  if [ -n "$ACTIVITY_ID" ]; then
    test_api "活动详情" "GET" "/api/v1/activities/$ACTIVITY_ID" ""
    test_api "参与活动" "POST" "/api/v1/activities/$ACTIVITY_ID/join" ""
  fi
fi

# Step 6: Summary
echo ""
echo "📊 [6/6] 测试报告"
echo "=========================================="
echo "  通过: $PASS  |  失败: $FAIL  |  总计: $((PASS+FAIL))"
echo "=========================================="

# Cleanup
kill $SERVER_PID 2>/dev/null || true
echo ""
echo "✨ 测试完成！服务已停止。"
echo ""
if [ $FAIL -gt 0 ]; then
  echo "⚠️  $FAIL 个测试失败，需要检查！"
  exit 1
else
  echo "🎉 全部测试通过！"
fi
