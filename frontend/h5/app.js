// ============================================================
// BOX·MAGIC 盲盒前端 - 应用逻辑
// ============================================================

const state = {
  apiBase: 'http://localhost:18100',
  token: '',
  user: null,
  campaigns: [],
  inventory: [],
  offers: [],
  prizes: {},
  member: null,
  currentCampaign: null,
  drawInProgress: false,
};

// ============================================================
// API 请求封装
// ============================================================

async function api(path, options = {}) {
  const url = state.apiBase + path;
  const headers = { 'Content-Type': 'application/json', ...(options.headers || {}) };
  if (state.token) headers['Authorization'] = `Bearer ${state.token}`;
  try {
    const res = await fetch(url, { ...options, headers });
    const data = await res.json();
    if (!res.ok) throw new Error(data.message || `HTTP ${res.status}`);
    return data;
  } catch (e) {
    if (e.message.includes('Failed to fetch')) throw new Error('无法连接到服务器，请检查API地址');
    throw e;
  }
}

// ============================================================
// Toast 提示
// ============================================================

let toastTimer;

function showToast(msg, isError = false) {
  const t = document.getElementById('toast');
  t.textContent = msg;
  t.style.background = isError ? 'rgba(200,50,50,0.9)' : 'rgba(0,0,0,0.85)';
  t.classList.remove('hidden');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => t.classList.add('hidden'), 2500);
}

// ============================================================
// 登录
// ============================================================

async function guestLogin() {
  state.apiBase = document.getElementById('apiBaseInput').value.trim();
  const nickname = document.getElementById('nicknameInput').value.trim();
  const status = document.getElementById('loginStatus');
  status.textContent = '登录中...';
  try {
    const payload = await api('/api/v1/auth/guest-login', {
      method: 'POST',
      body: JSON.stringify({ nickname }),
    });
    state.token = payload.data.session.token;
    state.user = payload.data.user;
    document.getElementById('loginOverlay').style.display = 'none';
    showToast('🎉 欢迎来到盲盒世界！送你100积分体验金');
    refreshAll();
  } catch (e) {
    status.textContent = '❌ ' + e.message;
  }
}

// ============================================================
// Tab 切换
// ============================================================

function switchTab(tab) {
  document.querySelectorAll('.tab-content').forEach(el => el.classList.remove('active'));
  document.querySelectorAll('.tab').forEach(el => el.classList.remove('active'));
  document.getElementById('tab-' + tab)?.classList.add('active');
  document.querySelector(`.tab[data-tab="${tab}"]`)?.classList.add('active');
  if (!state.token) return;
  switch (tab) {
    case 'series': loadSeries(); break;
    case 'inventory': loadInventory(); break;
    case 'exchange': loadExchange(); break;
    case 'rank': loadLeaderboard(); break;
    case 'member': loadMember(); break;
  }
}

// ============================================================
// 刷新所有数据
// ============================================================

async function refreshAll() {
  if (!state.token) return;
  try {
    const [seriesData, memberData] = await Promise.all([
      api('/api/v1/blindbox/campaigns'),
      api('/api/v1/blindbox/member'),
    ]);
    state.campaigns = seriesData.data;
    state.member = memberData.data;
    updatePoints();
    renderSeries();
  } catch (e) {
    showToast('加载失败: ' + e.message, true);
  }
}

function updatePoints() {
  document.getElementById('pointsDisplay').textContent = '💎 ' + (state.member?.points || 0);
}

// ============================================================
// 每日签到
// ============================================================

async function dailyCheckIn() {
  if (!state.token) { showToast('请先登录', true); return; }
  try {
    const res = await api('/api/v1/blindbox/checkin', { method: 'POST' });
    const data = res.data;
    let msg = `📅 签到成功！+${data.points_awarded}分`;
    if (data.is_bonus) msg += ' 🎉 连续签到奖励！';
    showToast(msg);
    state.member.points = data.new_balance;
    updatePoints();
  } catch (e) {
    showToast(e.message, true);
  }
}

// ============================================================
// 系列列表
// ============================================================

function renderSeries() {
  const list = document.getElementById('seriesList');
  if (!state.campaigns.length) {
    list.innerHTML = '<div class="loading">暂无系列</div>';
    return;
  }
  list.innerHTML = state.campaigns.map(c => {
    const camp = c.campaign || c;
    const progress = c.progress || '';
    const pct = progress ? (progress.collected_items / Math.max(progress.total_items, 1) * 100) : 0;
    return `
      <div class="series-card" onclick="showDetail('${camp.id}')">
        <h3>${camp.name}</h3>
        <div class="summ">${camp.campaign_summary || ''}</div>
        <div class="series-meta">
          <span>🎯 ${c.prizes?.length || 0}款</span>
          ${progress ? `<span>📊 ${progress.collected_items}/${progress.total_items}</span>` : ''}
        </div>
        ${progress ? `<div class="progress-bar"><div class="progress-fill" style="width:${pct}%"></div></div>` : ''}
      </div>`;
  }).join('');
}

async function loadSeries() {
  try {
    const seriesData = await api('/api/v1/blindbox/campaigns');
    state.campaigns = seriesData.data;
    renderSeries();
  } catch (e) {
    document.getElementById('seriesList').innerHTML = `<div class="loading">❌ ${e.message}</div>`;
  }
}

// ============================================================
// 系列详情
// ============================================================

async function showDetail(campaignId) {
  try {
    const [probData, progressData] = await Promise.all([
      api(`/api/v1/blindbox/campaigns/${campaignId}/probabilities`),
      api(`/api/v1/blindbox/series-progress?campaign_id=${campaignId}`),
    ]);
    state.currentCampaign = probData.data.campaign;
    const prizes = probData.data.prizes;
    const progress = progressData.data;
    const pity = probData.data.pity_config;
    state.prizes = {};

    const rarityIcons = { common: '⬜', rare: '🔵', secret: '🟣', limited: '🌟' };
    const rarityNames = { common: '普通', rare: '稀有', secret: '隐藏', limited: '限定' };

    document.getElementById('detailContent').innerHTML = `
      <div class="detail-hero">
        <h2>${state.currentCampaign.name}</h2>
        <p>${state.currentCampaign.campaign_summary || ''}</p>
      </div>
      ${progress ? `
        <div class="pity-info">
          已收集 ${progress.collected_items}/${progress.total_items} 款
          ${progress.duplicates > 0 ? ` | 重复 ×${progress.duplicates}` : ''}
          <div class="progress-bar"><div class="progress-fill" style="width:${progress.progress_percent || 0}%"></div></div>
        </div>` : ''}
      <div class="prize-grid">
        ${prizes.map(p => {
          const owned = progress?.collected_prizes?.find(cp => cp.id === p.id);
          state.prizes[p.id] = p;
          return `<div class="prize-item ${owned ? 'owned' : ''} ${p.level === 'limited' ? 'limited' : ''}">
            <div class="icon">${rarityIcons[p.level] || '🎁'}</div>
            <div class="name ${'rarity-' + p.level}">${p.name}</div>
            <div style="font-size:11px;color:var(--muted);">${rarityNames[p.level] || p.level}</div>
            <div style="font-size:11px;color:var(--muted);">${p.base_prob || '-'}</div>
            ${owned ? `<div class="count-badge">×${owned.count || 1}</div>` : ''}
          </div>`;
        }).join('')}
      </div>
      ${pity?.enabled ? `
        <div class="pity-info">
          🛡️ 软保底: ${pity.soft_pity_n}次 | 硬保底: ${pity.hard_pity_n}次
          ${progress?.pity_status ? `| 已连续 ${progress.pity_status.consecutive_misses} 次未出稀有` : ''}
        </div>` : ''}
      <div class="draw-actions">
        <button onclick="singleDraw('${campaignId}')" ${state.drawInProgress ? 'disabled' : ''}>🎲 单抽 (100分)</button>
        <button onclick="tenDraw('${campaignId}')" ${state.drawInProgress ? 'disabled' : ''}>🎲 十连 (950分)</button>
      </div>
      <div id="drawResult"></div>
    `;

    switchTab('detail');
    document.getElementById('tab-detail').classList.add('active');
    document.querySelector('.tab[data-tab="series"]')?.classList.add('active');
  } catch (e) {
    showToast('加载详情失败: ' + e.message, true);
  }
}

// ============================================================
// 抽奖逻辑
// ============================================================

async function singleDraw(campaignId) {
  await doDraw(campaignId, 1);
}

async function tenDraw(campaignId) {
  await doDraw(campaignId, 10);
}

async function doDraw(campaignId, count) {
  if (!state.token) { showToast('请先登录', true); return; }
  if (state.drawInProgress) return;
  state.drawInProgress = true;
  document.querySelectorAll('.draw-actions button').forEach(b => b.disabled = true);

  // 播放开盒动画
  showOpenBoxAnim();

  try {
    const res = await api('/api/v1/blindbox/draw', {
      method: 'POST',
      body: JSON.stringify({ campaign_id: campaignId, draw_count: count }),
    });
    const data = res.data;

    // 延迟展示结果（模拟开盒）
    setTimeout(() => {
      showDrawResults(data, count);
      state.drawInProgress = false;
      document.querySelectorAll('.draw-actions button').forEach(b => b.disabled = false);

      // 刷新积分和进度
      refreshAll();
      if (data.collection_reward) {
        setTimeout(() => showToast(`🏆 集齐奖励！解锁: ${data.collection_reward.reward_name}！+500分`), 500);
      }
    }, count === 10 ? 2000 : 1200);
  } catch (e) {
    setTimeout(() => {
      closeModal();
      showToast('❌ ' + e.message, true);
      state.drawInProgress = false;
      document.querySelectorAll('.draw-actions button').forEach(b => b.disabled = false);
    }, 500);
  }
}

// ============================================================
// 开盒动画 & 结果展示
// ============================================================

function showOpenBoxAnim() {
  document.getElementById('boxAnim').classList.remove('hidden');
  document.getElementById('boxResult').classList.add('hidden');
  document.getElementById('openBoxModal').classList.remove('hidden');
}

function showDrawResults(data, count) {
  document.getElementById('boxAnim').classList.add('hidden');
  document.getElementById('boxResult').classList.remove('hidden');

  const draws = data.draws || [];
  const last = draws[draws.length - 1] || {};
  const first = draws[0] || {};

  // 根据最高稀有度决定展示哪个结果
  let show = first;
  for (const d of draws) {
    const levels = { limited: 5, secret: 4, rare: 3, common: 1 };
    if ((levels[d.prize_level] || 0) > (levels[show.prize_level] || 0)) show = d;
  }

  const rarityColors = {
    common: '#94a3b8', rare: '#60a5fa', secret: '#a78bfa', limited: '#fbbf24',
  };
  const rarityLabels = { common: '普通款', rare: '稀有款 🔵', secret: '隐藏款 🟣', limited: '限定款 🌟' };

  document.getElementById('resultRarity').style.background = rarityColors[show.prize_level] || '#666';
  document.getElementById('resultRarity').textContent = rarityLabels[show.prize_level] || '普通款';

  const icons = { common: '🎁', rare: '✨', secret: '🌟', limited: '👑' };
  document.getElementById('resultIcon').textContent = icons[show.prize_level] || '🎁';
  document.getElementById('resultName').textContent = show.prize_name || '未中奖';

  const newTag = document.getElementById('resultNew');
  if (show.is_new) {
    newTag.classList.remove('hidden');
  } else {
    newTag.classList.add('hidden');
  }

  const reward = document.getElementById('resultReward');
  if (data.collection_reward) {
    reward.textContent = `🏆 集齐奖励：${data.collection_reward.description || '解锁成功！'}`;
    reward.classList.remove('hidden');
  } else {
    reward.classList.add('hidden');
  }

  // 十连抽展示汇总
  if (count >= 2 && draws.length > 1) {
    const summary = draws.map(d => {
      const icons = { common: '⬜', rare: '🔵', secret: '🟣', limited: '🌟' };
      return `${icons[d.prize_level] || '⬜'} ${d.prize_name || '未中'}`;
    }).join('  ');
    document.getElementById('resultName').innerHTML = show.prize_name + `<div style="font-size:12px;color:var(--muted);margin-top:8px">${summary}</div>`;
  }
}

function closeModal() {
  document.getElementById('openBoxModal').classList.add('hidden');
}

async function shareDraw() {
  if (!state.token) { showToast('请先登录', true); return; }
  try {
    const res = await api('/api/v1/blindbox/share-reward', { method: 'POST' });
    showToast(`📤 分享成功！+${res.data.points_awarded}分（今日还可分享${res.data.daily_left}次）`);
    state.member.points = res.data.new_balance;
    updatePoints();
  } catch (e) {
    showToast(e.message, true);
  }
}

// ============================================================
// 我的盲盒/库存
// ============================================================

async function loadInventory() {
  const el = document.getElementById('inventoryContent');
  try {
    const [invRes, memberRes] = await Promise.all([
      api('/api/v1/blindbox/inventory'),
      api('/api/v1/blindbox/member'),
    ]);
    state.inventory = invRes.data || [];
    state.member = memberRes.data;
    updatePoints();
    renderInventory();
  } catch (e) {
    el.innerHTML = `<div class="loading">❌ ${e.message}</div>`;
  }
}

function renderInventory() {
  const el = document.getElementById('inventoryContent');
  if (!state.inventory.length) {
    el.innerHTML = '<div class="loading">📭 还没有收集到任何盲盒款式，去抽盒吧！</div>';
    return;
  }

  // 按系列分组
  const groups = {};
  for (const item of state.inventory) {
    if (!groups[item.campaign_id]) groups[item.campaign_id] = [];
    groups[item.campaign_id].push(item);
  }

  const rarityIcons = { common: '⬜', rare: '🔵', secret: '🟣', limited: '🌟' };
  const rarityNames = { common: '普通', rare: '稀有', secret: '隐藏', limited: '限定' };
  const blendRecipes = { common: 3, rare: 5, secret: 3 };

  el.innerHTML = Object.entries(groups).map(([campId, items]) => {
    const camp = state.campaigns.find(c => (c.campaign || c).id === campId);
    const name = camp ? (camp.campaign || camp).name : campId;
    // 按款式去重统计
    const countMap = {};
    for (const it of items) {
      if (!countMap[it.prize_id]) countMap[it.prize_id] = { ...it, count: 0 };
      countMap[it.prize_id].count++;
    }
    const unique = Object.values(countMap);
    return `<div class="inv-group">
      <h3>${name} (${unique.length}款 / ${items.length}件)</h3>
      <div class="prize-grid">
        ${unique.map(it => {
          const need = blendRecipes[it.prize_level];
          const canBlend = need && it.count >= need;
          return `<div class="prize-item owned">
            <div class="icon">${rarityIcons[it.prize_level] || '🎁'}</div>
            <div class="name ${'rarity-' + it.prize_level}">${it.prize_name}</div>
            <div style="font-size:11px;color:var(--muted);">${rarityNames[it.prize_level] || ''}</div>
            ${it.count > 1 ? `<div class="count-badge">×${it.count}</div>` : ''}
            ${canBlend ? `<button class="btn-sm" style="margin-top:6px;font-size:11px;" onclick="blendPrize('${it.prize_id}','${campId}')">🔬 合成</button>` : ''}
          </div>`;
        }).join('')}
      </div>
    </div>`;
  }).join('');
}

// 合成
async function blendPrize(prizeId, campaignId) {
  const recipe = { common: '3普通→1稀有', rare: '5稀有→1隐藏', secret: '3隐藏→1限定' };
  const prize = state.inventory.find(i => i.prize_id === prizeId);
  if (!confirm(`确定合成吗？\n${recipe[prize?.prize_level] || ''}`)) return;
  try {
    const res = await api('/api/v1/blindbox/blend', {
      method: 'POST',
      body: JSON.stringify({ source_prize_id: prizeId, campaign_id: campaignId }),
    });
    const data = res.data;
    showToast(`🔬 合成成功！${data.source_prize_name} → ${data.result_prize_name} 🎉`);
    loadInventory();
    refreshAll();
  } catch (e) {
    showToast('❌ ' + e.message, true);
  }
}

// ============================================================
// 交换市场
// ============================================================

async function loadExchange() {
  try {
    const res = await api('/api/v1/blindbox/exchange-offers');
    state.offers = res.data || [];
    renderExchange();
  } catch (e) {
    document.getElementById('exchangeContent').innerHTML = `<div class="loading">❌ ${e.message}</div>`;
  }
}

function renderExchange() {
  const el = document.getElementById('exchangeContent');
  if (!state.offers.length) {
    el.innerHTML = '<div class="loading">📭 暂无交换挂单，快去发布吧！</div>';
    return;
  }
  el.innerHTML = state.offers.map(o => `
    <div class="exchange-item">
      <div class="exchange-body">
        <strong>${o.user_nickname || '匿名'}</strong><br/>
        🎁 ${o.have_prize_name} <span class="arrow">→</span> 🎯 ${o.want_prize_name}
      </div>
      <div class="exchange-actions">
        <button onclick="acceptExchange('${o.id}')">接受</button>
      </div>
    </div>
  `).join('');
}

async function showExchangeForm() {
  // 加载库存和所有奖品列表
  try {
    const [invRes, seriesRes] = await Promise.all([
      api('/api/v1/blindbox/inventory'),
      api('/api/v1/campaigns'),
    ]);
    const inventory = invRes.data || [];
    const campaigns = seriesRes.data || [];

    // 我有（重复款）
    const duplicates = inventory.filter((item, idx) =>
      inventory.findIndex(i => i.prize_id === item.prize_id) !== idx
    );
    const uniqueDups = [...new Map(duplicates.map(d => [d.prize_id, d])).values()];

    // 所有奖品
    const allPrizes = [];
    for (const c of campaigns) {
      const camp = c.campaign || c;
      if (c.prizes) allPrizes.push(...c.prizes.map(p => ({ ...p, campaign_name: camp.name })));
    }

    const haveEl = document.getElementById('havePrizeSelect');
    const wantEl = document.getElementById('wantPrizeSelect');

    haveEl.innerHTML = uniqueDups.map(d => `<option value="${d.prize_id}">${d.prize_name}</option>`).join('') || '<option value="">暂无重复款</option>';
    wantEl.innerHTML = allPrizes.map(p => `<option value="${p.id}">[${p.campaign_name || ''}] ${p.name}</option>`).join('');

    document.getElementById('exchangeModal').classList.remove('hidden');
  } catch (e) {
    showToast('加载失败: ' + e.message, true);
  }
}

function closeExchangeModal() {
  document.getElementById('exchangeModal').classList.add('hidden');
}

async function publishExchange() {
  const haveId = document.getElementById('havePrizeSelect').value;
  const wantId = document.getElementById('wantPrizeSelect').value;
  if (!haveId || !wantId) { showToast('请选择要交换的款式', true); return; }
  try {
    await api('/api/v1/blindbox/exchange-offers', {
      method: 'POST',
      body: JSON.stringify({ have_prize_id: haveId, want_prize_id: wantId }),
    });
    showToast('✅ 交换挂单已发布！');
    closeExchangeModal();
    loadExchange();
  } catch (e) {
    showToast('❌ ' + e.message, true);
  }
}

async function acceptExchange(offerId) {
  try {
    await api(`/api/v1/blindbox/exchange-offers/${offerId}/accept`, { method: 'POST' });
    showToast('✅ 交换成功！');
    loadExchange();
    loadInventory();
    refreshAll();
  } catch (e) {
    showToast('❌ ' + e.message, true);
  }
}

// ============================================================
// 排行榜
// ============================================================

async function loadLeaderboard() {
  try {
    const res = await api('/api/v1/blindbox/leaderboard');
    const entries = res.data || [];
    const el = document.getElementById('rankContent');
    const medals = ['🥇', '🥈', '🥉'];
    if (!entries.length) {
      el.innerHTML = '<div class="loading">暂无排行数据</div>';
      return;
    }
    el.innerHTML = entries.map((e, i) => `
      <div class="rank-item ${i === 0 ? 'rank-1' : i === 1 ? 'rank-2' : i === 2 ? 'rank-3' : ''}">
        <div class="rank-num">${medals[i] || `#${i+1}`}</div>
        <div class="nick">${e.nickname || '匿名'}</div>
        <div class="progress">🎯 ${e.collected_count}款 / ${e.series_completed}系列集齐</div>
      </div>
    `).join('');
  } catch (e) {
    document.getElementById('rankContent').innerHTML = `<div class="loading">❌ ${e.message}</div>`;
  }
}

// ============================================================
// 会员信息
// ============================================================
// 会员信息
async function loadMember() {
  try {
    const [memberRes, pointsRes, cardRes] = await Promise.all([
      api('/api/v1/blindbox/member'),
      api('/api/v1/blindbox/points-log'),
      api('/api/v1/blindbox/my-card'),
    ]);
    const member = memberRes.data;
    const logs = pointsRes.data || [];
    const card = cardRes.data;
    state.member = member;
    updatePoints();

    const levelIcons = { normal: '🥉', silver: '🥈', gold: '🥇', diamond: '👑' };
    const levelNames = {
      normal: '青铜', silver: '白银', gold: '黄金', diamond: '铂金/钻石',
    };
    const levelColors = {
      normal: '#94a3b8', silver: '#c0c0c0', gold: '#fbbf24', diamond: '#a78bfa',
    };

    // 优化月卡显示
    let cardHtml = '';
    if (card) {
      const cardNames = { weekly: '周卡', monthly: '月卡', season: '季卡' };
      cardHtml = '<div class="member-card" style="border-color:var(--accent2);">' +
        '<div style="font-size:14px;color:var(--accent2);font-weight:600;">🎫 ' + (cardNames[card.card_type] || card.card_type) + '</div>' +
        '<div style="font-size:12px;color:var(--muted);">有效期至: ' + (card.expires_at ? card.expires_at.slice(0,10) : '-') + '</div>' +
        '</div>';
    } else {
      cardHtml = '<div class="member-card">' +
        '<div style="font-size:14px;color:var(--muted);">暂无月卡</div>' +
        '<div style="display:flex;gap:8px;justify-content:center;margin-top:10px;flex-wrap:wrap;">' +
        '<button class="btn-sm" onclick="buyCard(\'weekly\')">周卡 9.9</button>' +
        '<button class="btn-sm" style="background:rgba(167,139,250,0.3);" onclick="buyCard(\'monthly\')">🔥 月卡 28</button>' +
        '<button class="btn-sm" onclick="buyCard(\'season\')">季卡 68</button>' +
        '</div></div>';
    }

    document.getElementById('memberContent').innerHTML =
      '<div class="member-card">' +
        '<div class="level-icon">' + (levelIcons[member.level] || '🥉') + '</div>' +
        '<div class="level-name" style="color:' + (levelColors[member.level] || '#94a3b8') + '">' + (levelNames[member.level] || '青铜') + '会员</div>' +
        '<div class="points">💎 ' + member.points + ' 分</div>' +
        '<div style="font-size:12px;color:var(--muted);margin-top:6px;">累计抽奖 ' + member.total_draws + ' 次 | 累计消费 ' + member.total_spent + ' 分</div>' +
      '</div>' +
      cardHtml +
      '<div class="panel-head"><h2>💳 积分变动</h2></div>' +
      logs.slice(0, 20).map(function(log) {
        return '<div style="display:flex;justify-content:space-between;padding:6px 0;font-size:13px;border-bottom:1px solid rgba(255,255,255,0.05)">' +
          '<span style="color:var(--muted);font-size:12px">' + formatTime(log.created_at) + '</span>' +
          '<span style="flex:1;margin-left:8px">' + (log.remark || log.reason) + '</span>' +
          '<span style="color:' + (log.points >= 0 ? 'var(--green)' : 'var(--accent)') + ';font-weight:600">' + (log.points >= 0 ? '+' : '') + log.points + '</span>' +
        '</div>';
      }).join('');

    `;
  } catch (e) {
    document.getElementById('memberContent').innerHTML = '<div class="loading">❌ ' + e.message + '</div>';
  }
}

function formatTime(t) {
  if (!t) return '';
  var d = new Date(t);
  return (d.getMonth()+1) + '/' + d.getDate() + ' ' +
    d.getHours().toString().padStart(2,'0') + ':' +
    d.getMinutes().toString().padStart(2,'0');
}

// 购买月卡
async function buyCard(cardType) {
  if (!state.token) { showToast('请先登录', true); return; }
  var names = { weekly: '周卡(990分)', monthly: '月卡(2800分)', season: '季卡(6800分)' };
  if (!confirm('确定购买' + (names[cardType] || cardType) + '吗？')) return;
  try {
    var res = await api('/api/v1/blindbox/buy-card', {
      method: 'POST',
      body: JSON.stringify({ card_type: cardType }),
    });
    var data = res.data;
    showToast('🎫 购买成功！有效期至 ' + data.expires_at);
    loadMember();
  } catch (e) {
    showToast('❌ ' + e.message, true);
  }
}

// ============================================================
// 初始化 - 检查是否已登录
// ============================================================

// 为空闲时自动加载
let autoRefreshTimer;

document.addEventListener('visibilitychange', () => {
  if (!document.hidden && state.token) refreshAll();
});

// 自动每30秒刷新积分
setInterval(() => {
  if (state.token) {
    api('/api/v1/blindbox/member').then(r => {
      state.member = r.data;
      updatePoints();
    }).catch(() => {});
  }
}, 30000);

// ============================================================
// 🆕 商店
// ============================================================

async function loadShop() {
  if (!state.token) return;
  try {
    const [shopRes, itemsRes, frRes] = await Promise.all([
      api('/api/v1/shop/items'),
      api('/api/v1/shop/items/inventory'),
      api('/api/v1/first-recharge/status'),
    ]);
    renderShopItems(shopRes.data || []);
    renderUserItems(itemsRes.data || []);
    renderFirstRechargeBanner(frRes.data);
  } catch (e) {
    document.getElementById('shopItems').innerHTML = `<div class="loading">❌ ${e.message}</div>`;
  }
}

function renderShopItems(items) {
  const el = document.getElementById('shopItems');
  if (!items.length) {
    el.innerHTML = '<div class="loading">暂无商品</div>';
    return;
  }
  const categoryLabels = { daily: '每日特价', weekly: '周礼包', festival: '节日礼包', item: '道具' };
  el.innerHTML = items.map(item => `
    <div class="shop-item">
      <span class="item-category">${categoryLabels[item.category] || item.category}</span>
      <div class="item-name">${item.name}</div>
      <div class="item-desc">${item.description || ''}</div>
      <div class="item-price">💎 ${item.price_points} 积分${item.price_cash ? ` / 💰¥${(item.price_cash/100).toFixed(0)}` : ''}</div>
      <button class="item-buy-btn" onclick="buyShopItem('${item.id}')">购买</button>
    </div>
  `).join('');
}

function renderUserItems(items) {
  const map = {};
  for (const item of items) map[item.item_type] = item.quantity;
  const types = ['hint_card', 'see_through', 'ten_draw_ticket'];
  for (const t of types) {
    const el = document.getElementById('item-' + t);
    if (el) el.textContent = {
      hint_card: '💡 提示卡: ' + (map[t] || 0),
      see_through: '👁️ 透卡: ' + (map[t] || 0),
      ten_draw_ticket: '🎟️ 十连券: ' + (map[t] || 0),
    }[t];
  }
}

async function buyShopItem(itemId) {
  if (!state.token) { showToast('请先登录', true); return; }
  try {
    const res = await api('/api/v1/shop/buy', {
      method: 'POST',
      body: JSON.stringify({ shop_item_id: itemId, quantity: 1 }),
    });
    showToast('✅ 购买成功！获得 ' + res.data.item_name);
    state.member.points = res.data.new_points;
    updatePoints();
    loadShop();
  } catch (e) {
    showToast('❌ ' + e.message, true);
  }
}

// ============================================================
// 🆕 首充礼包
// ============================================================

function renderFirstRechargeBanner(status) {
  const banner = document.getElementById('firstRechargeBanner');
  if (!status || !status.claimed) { banner.classList.add('hidden'); return; }
  const unclaimed = status.claimed.length < 3;
  banner.classList.toggle('hidden', !unclaimed);
}

function showFirstRecharge() {
  document.getElementById('firstRechargeModal').classList.remove('hidden');
  loadFirstRechargePacks();
}

function closeFirstRechargeModal() {
  document.getElementById('firstRechargeModal').classList.add('hidden');
}

async function loadFirstRechargePacks() {
  const el = document.getElementById('firstRechargePacks');
  try {
    const [packsRes, statusRes] = await Promise.all([
      api('/api/v1/first-recharge/packs'),
      api('/api/v1/first-recharge/status'),
    ]);
    const packs = packsRes.data || [];
    const claimed = (statusRes.data && statusRes.data.claimed) || [];
    const itemTypeLabels = { points: '💎积分', draw_ticket: '🎟️抽奖券', prize: '🎁盲盒', month_card: '💳月卡', hint_card: '💡提示卡', see_through: '👁️透卡', ten_draw_ticket: '🎟️十连券', free_draw: '🎲免费抽' };
    el.innerHTML = packs.sort((a,b) => a.sort_order - b.sort_order).map(pack => {
      const isClaimed = claimed.includes(pack.id);
      return `
        <div class="fr-pack-card">
          <div class="pack-header">
            <span class="pack-name">${pack.name}</span>
            <span class="pack-price">💰 ¥${(pack.cash_price/100).toFixed(0)}</span>
          </div>
          <div class="pack-desc">${pack.description}</div>
          <div class="pack-items">
            ${(pack.items || []).map(item => `<span class="pack-item-tag">${itemTypeLabels[item.type] || item.type} ×${item.qty}</span>`).join('')}
          </div>
          <button class="pack-claim-btn" ${isClaimed ? 'disabled' : ''} onclick="claimFirstRecharge('${pack.id}')">
            ${isClaimed ? '✅ 已领取' : '🔥 立即领取'}
          </button>
        </div>
      `;
    }).join('');
  } catch (e) {
    el.innerHTML = `<div class="loading">❌ ${e.message}</div>`;
  }
}

async function claimFirstRecharge(packId) {
  if (!state.token) { showToast('请先登录', true); return; }
  try {
    const res = await api('/api/v1/first-recharge/claim', {
      method: 'POST',
      body: JSON.stringify({ pack_id: packId }),
    });
    showToast('🎉 领取成功！获得 ' + (res.data.pack_name || '礼包'));
    state.member.points = res.data.new_points;
    updatePoints();
    loadFirstRechargePacks();
    renderFirstRechargeBanner({ claimed: ['dummy'] });
  } catch (e) {
    showToast('❌ ' + e.message, true);
  }
}

// Extend switchTab to include shop
const origSwitchTab = switchTab;
switchTab = function(tab) {
  document.querySelectorAll('.tab-content').forEach(el => el.classList.remove('active'));
  document.querySelectorAll('.tab').forEach(el => el.classList.remove('active'));
  document.getElementById('tab-' + tab)?.classList.add('active');
  document.querySelector(`.tab[data-tab="${tab}"]`)?.classList.add('active');
  if (!state.token) return;
  switch (tab) {
    case 'series': loadSeries(); break;
    case 'inventory': loadInventory(); break;
    case 'exchange': loadExchange(); break;
    case 'rank': loadLeaderboard(); break;
    case 'member': loadMember(); break;
    case 'shop': loadShop(); break;
    case 'social': loadSocial(); break;
  }
};

// Extend switchTab to include shop AND social
const origSwitchTab = switchTab;
switchTab = function(tab) {
  document.querySelectorAll('.tab-content').forEach(el => el.classList.remove('active'));
  document.querySelectorAll('.tab').forEach(el => el.classList.remove('active'));
  document.getElementById('tab-' + tab)?.classList.add('active');
  document.querySelector(`.tab[data-tab="${tab}"]`)?.classList.add('active');
  if (!state.token) return;
  switch (tab) {
    case 'series': loadSeries(); break;
    case 'inventory': loadInventory(); break;
    case 'exchange': loadExchange(); break;
    case 'rank': loadLeaderboard(); break;
    case 'member': loadMember(); break;
    case 'shop': loadShop(); break;
    case 'social': loadSocial(); break;
  }
};

function switchSocialTab(sub) {
  document.querySelectorAll('.social-panel').forEach(el => el.classList.remove('active'));
  document.querySelectorAll('.sub-tab').forEach(el => el.classList.remove('active'));
  document.getElementById('social-' + sub)?.classList.add('active');
  document.querySelector(`.sub-tab[onclick*="'${sub}'"]`)?.classList.add('active');
  if (sub === 'invite') { loadAssistProgress(); loadInviteRecords(); }
  else if (sub === 'team') loadTeamSection();
  else if (sub === 'gift') { loadIncomingGifts(); loadSentGifts(); }
}

async function loadSocial() {
  loadAssistProgress();
  loadInviteRecords();
  loadTeamSection();
  loadIncomingGifts();
  loadSentGifts();
  loadInviteStats();
}

async function loadInviteStats() {
  try {
    const res = await api('/api/v1/share/invite-stats');
    const stats = res.data;
    document.getElementById('inviteStats').innerHTML = `
      <div class="stat-row">
        <span>📋 已邀请 <strong>${stats.total_invites || 0}</strong> 人</span>
        <span>✋ 收到助力 <strong>${stats.total_assists || 0}</strong> 次</span>
        <span>🎯 已完成 <strong>${stats.completed_assists || 0}</strong> 项</span>
      </div>
    `;
  } catch(e) { /* ignore */ }
}

async function generateInviteLink() {
  if (!state.token) { showToast('请先登录', true); return; }
  try {
    const res = await api('/api/v1/share/invite', { method: 'POST' });
    const link = res.data?.invite_link || '链接生成失败';
    await navigator.clipboard.writeText(link).catch(() => {});
    showToast('📤 邀请链接已复制: ' + link);
    loadInviteStats();
  } catch(e) { showToast('❌ ' + e.message, true); }
}

async function loadAssistProgress() {
  try {
    const res = await api('/api/v1/share/assist-progress');
    const data = res.data || {};
    const types = [
      { key: 'free_draw', emoji: '🎲', name: '免费抽', target: 3, desc: '邀请3人助力→免费抽1次' },
      { key: 'pity_reduce', emoji: '⚡', name: '保底缩短', target: 5, desc: '邀请5人助力→保底-10' },
      { key: 'craft_boost', emoji: '🔮', name: '合成加成', target: 2, desc: '邀请2人助力→合成+20%' },
    ];
    document.getElementById('assistProgress').innerHTML = types.map(t => {
      const p = data[t.key];
      const current = p?.current || 0;
      const claimed = p?.claimed || false;
      const target = p?.target_count || t.target;
      const pct = Math.min(100, (current / target) * 100);
      return `
        <div class="assist-card ${claimed ? 'claimed' : ''}">
          <div class="assist-header">
            <span class="assist-emoji">${t.emoji}</span>
            <span class="assist-name">${t.name}</span>
            <span class="assist-count">${current}/${target}</span>
          </div>
          <div class="progress-bar"><div class="progress-fill" style="width:${pct}%"></div></div>
          <p class="assist-desc">${t.desc}</p>
          ${claimed ? '<span class="assist-claimed">✅ 已领取</span>' :
            current >= target ? `<button class="btn-sm" onclick="claimAssistReward('${t.key}')">🎁 领取奖励</button>` :
            ''}
        </div>
      `;
    }).join('');
  } catch(e) { document.getElementById('assistProgress').innerHTML = `<div class="loading">❌ ${e.message}</div>`; }
}

async function claimAssistReward(type) {
  try {
    const res = await api('/api/v1/share/assist-claim', {
      method: 'POST', body: JSON.stringify({ assist_type: type })
    });
    showToast('🎉 ' + (res.data?.description || '领取成功！'));
    loadAssistProgress();
    updatePoints();
  } catch(e) { showToast('❌ ' + e.message, true); }
}

async function loadInviteRecords() {
  try {
    const res = await api('/api/v1/share/invitees');
    const records = res.data || [];
    document.getElementById('inviteRecords').innerHTML = records.length ?
      '<h4>📋 邀请记录</h4>' + records.map(r =>
        `<div class="invite-record"><span>👤 ${r.invitee_id?.substring(0,8)}...</span><span class="time">${new Date(r.created_at).toLocaleDateString()}</span></div>`
      ).join('') : '<p class="hint">还没有邀请记录</p>';
  } catch(e) { /* ignore */ }
}

// ---- 组队 ----

async function loadTeamSection() {
  try {
    const res = await api('/api/v1/team/my');
    const info = res.data;
    const el = document.getElementById('teamSection');
    if (!info || !info.team) {
      el.innerHTML = `
        <div class="team-empty">
          <p>👥 你还没有队伍</p>
          <p class="hint">组队开盒，全员达标平分奖励！</p>
          <button class="btn-primary" onclick="showCreateTeam()">🚀 创建队伍</button>
        </div>
      `;
      return;
    }
    const t = info.team;
    const members = info.members || [];
    const remaining = info.remaining_hours || 0;
    const pct = Math.min(100, (t.current_draws / t.goal_draws) * 100);
    el.innerHTML = `
      <div class="team-card">
        <div class="team-header"><span class="team-name">👥 ${t.name}</span><span class="team-status">${t.status}</span></div>
        <div class="team-info"><span>⏱️ 剩余 ${remaining}h</span><span>👤 ${members.length}/${t.max_members}</span></div>
        <div class="team-progress">
          <div class="progress-bar"><div class="progress-fill" style="width:${pct}%"></div></div>
          <span>🎯 开盒 ${t.current_draws}/${t.goal_draws}</span>
        </div>
        <div class="team-members">
          <h4>队员 (${members.length})</h4>
          ${members.map(m => `<div class="member-row"><span>👤 ${m.nickname || m.user_id?.substring(0,8)}</span><span>🎲 ${m.draws}次</span></div>`).join('')}
        </div>
        <button class="btn-sm" onclick="leaveTeam()">🚪 离开队伍</button>
      </div>
    `;
  } catch(e) { document.getElementById('teamSection').innerHTML = `<div class="loading">❌ ${e.message}</div>`; }
}

function showCreateTeam() { document.getElementById('createTeamModal').classList.remove('hidden'); }
function closeCreateTeamModal() { document.getElementById('createTeamModal').classList.add('hidden'); }

async function createTeam() {
  const name = document.getElementById('teamNameInput').value || '我的队伍';
  const maxMembers = parseInt(document.getElementById('teamMaxMembers').value);
  const goalDraws = parseInt(document.getElementById('teamGoalDraws').value);
  try {
    const res = await api('/api/v1/team/create', {
      method: 'POST', body: JSON.stringify({ name, max_members: maxMembers, goal_draws: goalDraws })
    });
    showToast('🎉 队伍创建成功！');
    closeCreateTeamModal();
    loadTeamSection();
  } catch(e) { showToast('❌ ' + e.message, true); }
}

async function joinTeam(teamId) {
  try {
    await api('/api/v1/team/join', { method: 'POST', body: JSON.stringify({ team_id: teamId }) });
    showToast('🎉 加入队伍成功！');
    loadTeamSection();
  } catch(e) { showToast('❌ ' + e.message, true); }
}

async function leaveTeam() {
  if (!confirm('确定离开队伍？')) return;
  try {
    await api('/api/v1/team/leave', { method: 'POST' });
    showToast('已离开队伍');
    loadTeamSection();
  } catch(e) { showToast('❌ ' + e.message, true); }
}

// ---- 礼物 ----

async function loadIncomingGifts() {
  try {
    const res = await api('/api/v1/share/gifts/incoming');
    const gifts = res.data || [];
    document.getElementById('incomingGifts').innerHTML = gifts.length ?
      gifts.map(g => `
        <div class="gift-card">
          <span>🎁 ${g.prize_name || '盲盒'}</span>
          <span class="gift-from">来自 ${g.giver_id?.substring(0,8)}</span>
          <button class="btn-sm" onclick="receiveGift('${g.id}')">📥 领取</button>
        </div>
      `).join('') : '<p class="hint">暂无待收礼物</p>';
  } catch(e) { document.getElementById('incomingGifts').innerHTML = `<div class="loading">❌ ${e.message}</div>`; }
}

async function loadSentGifts() {
  try {
    const res = await api('/api/v1/share/gifts/sent');
    const gifts = res.data || [];
    document.getElementById('sentGifts').innerHTML = gifts.length ?
      gifts.map(g => `
        <div class="gift-card">
          <span>🎁 ${g.prize_name || '盲盒'}</span>
          <span>→ ${g.receiver_id?.substring(0,8)}</span>
          <span class="gift-status">${g.status}</span>
        </div>
      `).join('') : '<p class="hint">还没有送出的礼物</p>';
  } catch(e) { /* ignore */ }
}

function showSendGift() {
  document.getElementById('sendGiftModal').classList.remove('hidden');
  loadGiftPrizeSelect();
}
function closeSendGiftModal() { document.getElementById('sendGiftModal').classList.add('hidden'); }

async function loadGiftPrizeSelect() {
  try {
    const res = await api('/api/v1/inventory');
    const inv = res.data || [];
    const select = document.getElementById('giftPrizeSelect');
    select.innerHTML = inv.map(i =>
      `<option value="${i.prize_id}">${i.prize_name} (${i.prize_level || 'common'})</option>`
    ).join('') || '<option value="">没有可赠送的盲盒</option>';
  } catch(e) { /* ignore */ }
}

async function sendGift() {
  const prizeId = document.getElementById('giftPrizeSelect').value;
  const receiverId = document.getElementById('giftReceiverInput').value.trim();
  if (!prizeId || !receiverId) { showToast('请选择盲盒和输入接收者ID', true); return; }
  try {
    const res = await api('/api/v1/share/gift', {
      method: 'POST', body: JSON.stringify({ receiver_id: receiverId, prize_id: prizeId, campaign_id: '' })
    });
    showToast('🎁 赠送成功！');
    closeSendGiftModal();
    loadSentGifts();
  } catch(e) { showToast('❌ ' + e.message, true); }
}

async function receiveGift(giftId) {
  try {
    const res = await api('/api/v1/share/gift/receive', {
      method: 'POST', body: JSON.stringify({ gift_id: giftId })
    });
    showToast('🎉 收到 ' + (res.data?.prize_name || '礼物') + '！');
    loadIncomingGifts();
    updatePoints();
  } catch(e) { showToast('❌ ' + e.message, true); }
}

// ---- 分享抽卡结果 ----

async function shareDraw() {
  if (!state.token) { showToast('请先登录', true); return; }
  try {
    const res = await api('/api/v1/share/card', {
      method: 'POST', body: JSON.stringify({ card_type: 'draw_win', prize_name: '', prize_level: '' })
    });
    const card = res.data;
    const shareText = card?.title + ' ' + card?.description + ' ' + (card?.invite_link || '');
    await navigator.clipboard.writeText(shareText).catch(() => {});
    showToast('📤 分享文案已复制！');
  } catch(e) { showToast('❌ ' + e.message, true); }
}
