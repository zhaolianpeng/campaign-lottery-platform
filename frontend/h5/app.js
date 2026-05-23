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
// 🆕 骨架屏 + 确认弹窗 + 动画 辅助
// ============================================================

// 确认弹窗（替换浏览器confirm）
function showConfirm(title, desc, callback) {
  document.getElementById('confirmTitle').textContent = title;
  document.getElementById('confirmDesc').textContent = desc;
  document.getElementById('confirmBtn').onclick = function() {
    closeConfirmModal();
    callback();
  };
  document.getElementById('confirmModal').classList.remove('hidden');
}
function closeConfirmModal() {
  document.getElementById('confirmModal').classList.add('hidden');
}

// 积分飞入动画
function showPointsAnim(text) {
  var el = document.createElement('div');
  el.className = 'points-anim';
  el.textContent = text;
  document.body.appendChild(el);
  setTimeout(function() { el.remove(); }, 1000);
}

// 错误状态HTML
function errorHTML(msg, retryFn) {
  return '<div class="error-state"><div class="error-icon">❌</div><div class="error-msg">' + msg + '</div><button class="btn-sm" onclick="' + retryFn + '">🔄 重试</button></div>';
}

// 空状态HTML
function emptyHTML(icon, title, desc, btn) {
  return '<div class="empty-state"><div class="empty-icon">' + icon + '</div><div class="empty-title">' + title + '</div><div class="empty-desc">' + desc + '</div>' + (btn || '') + '</div>';
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
    if (state.user) {
      const ud = document.getElementById('userDisplay');
      if (ud) {
        ud.textContent = '👤 ' + (state.user.nickname || '用户');
        ud.classList.remove('hidden');
      }
    }
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
    loadActivityBanners();
  } catch (e) {
    document.getElementById('seriesList').innerHTML = errorHTML(e.message, 'loadSeries()');
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
  document.getElementById('openBoxModal').querySelector('.modal-content').className = 'modal-content';
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
  var mc = document.getElementById('openBoxModal').querySelector('.modal-content');
  mc.className = 'modal-content box-result ' + ('rarity-' + show.prize_level + '-bg');
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
    el.innerHTML = errorHTML(e.message, 'loadInventory()');
  }
}

function renderInventory() {
  const el = document.getElementById('inventoryContent');
  if (!state.inventory.length) {
    el.innerHTML = emptyHTML('📦', '暂无收藏', '还没有收集到任何盲盒款式，去抽盒吧！', '<button class="btn-sm" onclick="switchTab(\'series\')">🎲 去抽盒</button>');
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
            ${canBlend ? `<button class="btn-sm" style="margin-top:6px;font-size:11px;" onclick="blendPrize('${it.prize_id}','${campId}')\">🔬 合成</button>` : ''}
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
  if (!confirm(`确定合成吗？\\n${recipe[prize?.prize_level] || ''}`)) return;
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
  showConfirm('确认购买', '确定购买' + (names[cardType] || cardType) + '吗？', function() {
  try {
    var res = await api('/api/v1/blindbox/buy-card', {
      method: 'POST',
      body: JSON.stringify({ card_type: cardType }),
    });
    var data = res.data;
    showToast('🎫 购买成功！有效期至 ' + data.expires_at);
    showPointsAnim('🎫 +1');
    loadMember();
  } catch (e) {
    showToast('❌ ' + e.message, true);
  }
});

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
    document.getElementById('shopItems').innerHTML = errorHTML(e.message, 'loadShop()');
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
    showPointsAnim('💎 -' + res.data.points_cost);
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
    el.innerHTML = errorHTML(e.message, 'loadInventory()');
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
    showPointsAnim('🎉 首充礼包');
    state.member.points = res.data.new_points;
    updatePoints();
    loadFirstRechargePacks();
    renderFirstRechargeBanner({ claimed: ['dummy'] });
  } catch (e) {
    showToast('❌ ' + e.message, true);
  }
}

// switchTab: 覆盖原始函数以支持平滑过渡
const origSwitchTab = switchTab;
switchTab = function(tab) {
  document.querySelectorAll('.tab-content').forEach(el => {
    el.classList.remove('active');
  });
  document.querySelectorAll('.tab').forEach(el => el.classList.remove('active'));

  const target = document.getElementById('tab-' + tab);
  if (target) {
    target.classList.remove('tab-leaf-enter');
    // force reflow then add
    void target.offsetWidth;
    target.classList.add('active', 'tab-leaf-enter');
  }

  document.querySelector(`.tab[data-tab="${tab}"]`)?.classList.add('active');
  if (!state.token) return;

  // 延迟加载，让过渡动画先播放
  setTimeout(() => {
    switch (tab) {
      case 'series': loadSeries(); break;
      case 'inventory': loadInventory(); break;
      case 'exchange': loadExchange(); break;
      case 'rank': loadLeaderboard(); break;
      case 'member': loadMember(); break;
      case 'shop': loadShop(); break;
      case 'social': loadSocial(); break;
      case 'puzzle': loadPuzzle(); break;
    }
  }, 50);
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
  } catch(e) { document.getElementById('assistProgress').innerHTML = errorHTML(e.message, 'loadAssistProgress()'); }
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


// ============================================================
// 🧩 拼图碎片功能
// ============================================================

async function loadPuzzle() {
  loadPuzzleTemplates();
  loadFlashSales();
}

async function loadPuzzleTemplates() {
  const el = document.getElementById('puzzleTemplates');
  if (!el) return;
  try {
    const res = await api('/api/v1/puzzle/templates');
    const templates = res.data || [];
    if (!templates.length) {
      el.innerHTML = '<div class="loading">暂无拼图活动</div>';
      return;
    }
    el.innerHTML = templates.map(t => `
      <div class="puzzle-card" onclick="showPuzzleDetail('${t.id}')">
        <div class="puzzle-header">
          <span class="puzzle-name">🧩 ${t.name}</span>
        </div>
        <div class="puzzle-meta">
          <span>碎片 ${t.total_pieces}片</span>
          <span>🏆 ${t.reward_name || '神秘奖励'}</span>
        </div>
        ${t.user_progress ? `
          <div class="puzzle-progress">
            <div class="progress-bar"><div class="progress-fill" style="width:${Math.min(100, (t.user_progress.collected / t.user_progress.total) * 100)}%"></div></div>
            <span>${t.user_progress.collected}/${t.user_progress.total}</span>
          </div>
        ` : ''}
        <div class="puzzle-actions">
          <button class="btn-sm" onclick="event.stopPropagation();showPuzzleDetail('${t.id}')">查看详情</button>
          <button class="btn-sm" onclick="event.stopPropagation();composePuzzle('${t.id}')">🔨 拼合</button>
        </div>
      </div>
    `).join('');
  } catch (e) {
    el.innerHTML = errorHTML(e.message, 'loadInventory()');
  }
}

async function loadPuzzleProgress(templateId) {
  try {
    const res = await api('/api/v1/puzzle/progress/' + templateId);
    return res.data;
  } catch (e) {
    return null;
  }
}

async function showPuzzleDetail(templateId) {
  try {
    const [templatesRes, progress] = await Promise.all([
      api('/api/v1/puzzle/templates'),
      loadPuzzleProgress(templateId),
    ]);
    const templates = templatesRes.data || [];
    const tmpl = templates.find(t => t.id === templateId);
    if (!tmpl) { showToast('拼图不存在', true); return; }

    const total = tmpl.total_pieces || 1;
    const collected = progress ? (progress.collected_pieces || []) : [];
    const piecesHtml = Array.from({ length: total }, (_, i) => {
      const idx = i + 1;
      const isCollected = collected.includes(idx);
      return `<div class="puzzle-piece ${isCollected ? 'collected' : 'missing'}">${isCollected ? '✅' : idx}</div>`;
    }).join('');

    document.getElementById('puzzleModalTitle').textContent = '🧩 ' + tmpl.name;
    document.getElementById('puzzleModalBody').innerHTML = `
      <p style="color:var(--muted);font-size:13px;margin-bottom:12px;">🏆 奖励: ${tmpl.reward_name || '神秘奖励'}</p>
      <div class="puzzle-grid-visual">${piecesHtml}</div>
      <p style="font-size:12px;color:var(--muted);margin-top:8px;">已收集 ${collected.length}/${total} 片</p>
      ${collected.length >= total ? `
        <div style="margin-top:12px;">
          <button class="btn-primary" onclick="composePuzzle('${templateId}')">🔨 拼合领取奖励</button>
        </div>
      ` : ''}

      <!-- 拼图队伍 -->
      <div class="puzzle-team-section">
        <h3 style="margin-top:16px;">👥 拼图队伍</h3>
        <div id="puzzleTeamList-${templateId}">
          <button class="btn-sm" onclick="loadMyPuzzleTeams('${templateId}')">查看我的队伍</button>
          <button class="btn-sm" onclick="createPuzzleTeam('${templateId}')">创建队伍</button>
        </div>
      </div>
    `;
    document.getElementById('puzzleDetailModal').classList.remove('hidden');
  } catch (e) {
    showToast('❌ ' + e.message, true);
  }
}

function closePuzzleModal() {
  document.getElementById('puzzleDetailModal').classList.add('hidden');
}

async function composePuzzle(templateId) {
  if (!state.token) { showToast('请先登录', true); return; }
  try {
    const res = await api('/api/v1/puzzle/compose', {
      method: 'POST',
      body: JSON.stringify({ template_id: templateId }),
    });
    showToast('🎉 拼图完成！获得: ' + (res.data.reward_name || '奖励'));
    loadPuzzleTemplates();
    closePuzzleModal();
  } catch (e) {
    showToast('❌ ' + e.message, true);
  }
}

async function loadMyPuzzleTeams(templateId) {
  const el = document.getElementById('puzzleTeamList-' + templateId);
  if (!el) return;
  try {
    const res = await api('/api/v1/puzzle/team/my');
    const teams = res.data || [];
    if (!teams.length) {
      el.innerHTML = '<p class="hint">暂无队伍</p>';
      return;
    }
    el.innerHTML = teams.map(t => `
      <div class="team-card" style="margin:4px 0;">
        <div class="team-header"><span class="team-name">👥 ${t.name}</span></div>
        <div class="team-info"><span>👤 ${t.member_count || '?'}人</span></div>
        <button class="btn-sm" onclick="joinPuzzleTeam('${t.id}')">加入</button>
      </div>
    `).join('');
  } catch (e) {
    el.innerHTML = '<p class="hint">❌ ' + e.message + '</p>';
  }
}

async function createPuzzleTeam(templateId) {
  if (!state.token) { showToast('请先登录', true); return; }
  try {
    const res = await api('/api/v1/puzzle/team/create', {
      method: 'POST',
      body: JSON.stringify({ template_id: templateId, name: '拼图队' }),
    });
    showToast('✅ 队伍创建成功！');
    loadMyPuzzleTeams(templateId);
  } catch (e) {
    showToast('❌ ' + e.message, true);
  }
}

async function joinPuzzleTeam(teamId) {
  if (!state.token) { showToast('请先登录', true); return; }
  try {
    await api('/api/v1/puzzle/team/join', {
      method: 'POST',
      body: JSON.stringify({ team_id: teamId }),
    });
    showToast('✅ 加入队伍成功！');
  } catch (e) {
    showToast('❌ ' + e.message, true);
  }
}

// ============================================================
// ⚡ 限时抢购
// ============================================================

async function loadFlashSales() {
  const el = document.getElementById('flashList');
  if (!el) return;
  try {
    const res = await api('/api/v1/flash/list');
    const items = res.data || [];
    if (!items.length) {
      el.innerHTML = '<div class="loading">暂无抢购活动</div>';
      return;
    }
    el.innerHTML = items.map(item => {
      const remaining = item.remaining_stock || 0;
      const totalStock = item.total_stock || 0;
      const isExpired = item.expires_at && new Date(item.expires_at) < new Date();
      const timeText = item.expires_at ? formatCountdown(item.expires_at) : '';
      return `
        <div class="flash-card ${isExpired ? 'expired' : ''}">
          <div class="flash-header">
            <span class="flash-name">⚡ ${item.name}</span>
            <span class="flash-stock">${isExpired ? '已结束' : (remaining > 0 ? '剩余 ' + remaining + '/' + totalStock : '已售罄')}</span>
          </div>
          <div class="flash-time">${timeText}</div>
          ${item.min_level ? `<div class="flash-eligibility">👑 等级要求: ${item.min_level}</div>` : ''}
          <div class="flash-actions">
            ${!isExpired && remaining > 0 ? `
              <button class="btn-sm" onclick="subscribeFlash('${item.id}')">🔔 订阅</button>
              <button class="btn-sm" onclick="purchaseFlash('${item.id}')">💰 购买</button>
            ` : '<span style="color:var(--muted);font-size:12px;">已结束</span>'}
          </div>
        </div>
      `;
    }).join('');
  } catch (e) {
    el.innerHTML = errorHTML(e.message, 'loadInventory()');
  }
}

function formatCountdown(dateStr) {
  if (!dateStr) return '';
  const diff = new Date(dateStr) - new Date();
  if (diff <= 0) return '⏰ 已结束';
  const days = Math.floor(diff / 86400000);
  const hours = Math.floor((diff % 86400000) / 3600000);
  const mins = Math.floor((diff % 3600000) / 60000);
  const secs = Math.floor((diff % 60000) / 1000);
  if (days > 0) return `⏱️ 剩余 ${days}天${hours}小时`;
  if (hours > 0) return `⏱️ 剩余 ${hours}时${mins}分`;
  return `⏱️ 剩余 ${mins}分${secs}秒`;
}

async function subscribeFlash(flashId) {
  if (!state.token) { showToast('请先登录', true); return; }
  try {
    await api('/api/v1/flash/' + flashId + '/subscribe', { method: 'POST' });
    showToast('🔔 订阅成功！开售时将通知您');
  } catch (e) {
    showToast('❌ ' + e.message, true);
  }
}

async function purchaseFlash(flashId) {
  if (!state.token) { showToast('请先登录', true); return; }
  try {
    const res = await api('/api/v1/flash/' + flashId + '/purchase', { method: 'POST' });
    showToast('🎉 抢购成功！' + (res.data?.item_name || ''));
    loadFlashSales();
  } catch (e) {
    showToast('❌ ' + e.message, true);
  }
}

// ============================================================
// 活动系统
// ============================================================

async function loadActivityBanners() {
  const banner = document.getElementById('activityBanner');
  const list = document.getElementById('activityBannerList');
  if (!banner || !list) return;
  try {
    const res = await api('/api/v1/activities');
    const activities = res.data || [];
    if (!activities.length) {
      banner.style.display = 'none';
      return;
    }
    banner.style.display = '';
    list.innerHTML = activities.map(a => {
      const statusClass = getActivityStatusClass(a);
      const timeText = formatActivityTime(a);
      return `
        <div class="activity-banner-card ${statusClass}" onclick="event.stopPropagation();showActivityDetail('${a.id}')">
          ${a.type ? `<span class="act-badge">${a.type}</span>` : ''}
          <div class="act-name">${a.name}</div>
          <div class="act-desc">${a.description || ''}</div>
          <div class="act-time">${timeText}</div>
        </div>
      `;
    }).join('');
  } catch (e) {
    banner.style.display = 'none';
  }
}

function getActivityStatusClass(a) {
  const now = Date.now();
  const start = a.start_time ? new Date(a.start_time).getTime() : 0;
  const end = a.end_time ? new Date(a.end_time).getTime() : Infinity;
  if (end < now) return 'act-ended';
  if (start > now) return 'act-upcoming';
  return 'act-ongoing';
}

function formatActivityTime(a) {
  const now = Date.now();
  const start = a.start_time ? new Date(a.start_time).getTime() : 0;
  const end = a.end_time ? new Date(a.end_time).getTime() : Infinity;
  if (end < now) return '⏰ 已结束';
  if (start > now) {
    const diff = start - now;
    const days = Math.floor(diff / 86400000);
    const hours = Math.floor((diff % 86400000) / 3600000);
    return days > 0 ? `📅 ${days}天后开始` : `📅 ${hours}小时后开始`;
  }
  const diff = end - now;
  if (diff <= 0) return '⏰ 即将结束';
  const days = Math.floor(diff / 86400000);
  const hours = Math.floor((diff % 86400000) / 3600000);
  const mins = Math.floor((diff % 3600000) / 60000);
  if (days > 0) return `⏱️ 剩余 ${days}天${hours}小时`;
  if (hours > 0) return `⏱️ 剩余 ${hours}时${mins}分`;
  return `⏱️ 剩余 ${mins}分`;
}

async function showActivityDetail(activityId) {
  try {
    const res = await api('/api/v1/activities/' + activityId);
    const act = res.data;
    if (!act) { showToast('活动不存在', true); return; }

    document.getElementById('activityModalTitle').textContent = act.name || '活动详情';

    const now = Date.now();
    const start = act.start_time ? new Date(act.start_time).getTime() : 0;
    const end = act.end_time ? new Date(act.end_time).getTime() : Infinity;
    let statusText = '进行中', statusClass = 'ongoing';
    if (end < now) { statusText = '已结束'; statusClass = 'ended'; }
    else if (start > now) { statusText = '即将开始'; statusClass = 'upcoming'; }

    const rewards = act.rewards || [];
    const userJoined = act.user_joined || false;
    const rewardsHtml = rewards.length ? `
      <h3 style="margin-top:12px;margin-bottom:8px;">🎁 奖励列表</h3>
      <div class="activity-reward-list">
        ${rewards.map(r => {
          const claimed = r.claimed || false;
          return `
            <div class="activity-reward-item">
              <span class="reward-icon">${r.icon || '🎁'}</span>
              <div class="reward-info">
                <div class="reward-name">${r.name}</div>
                <div class="reward-desc">${r.description || ''}</div>
              </div>
              <button class="reward-claim-btn" ${claimed ? 'disabled' : ''} onclick="event.stopPropagation();claimActivityReward('${act.id}','${r.id}')">
                ${claimed ? '✅ 已领取' : '领取'}
              </button>
            </div>
          `;
        }).join('')}
      </div>
    ` : '';

    document.getElementById('activityModalBody').innerHTML = `
      <div class="activity-modal-body">
        <div class="act-info">
          <span class="act-status ${statusClass}">${statusText}</span>
          <p>${act.description || ''}</p>
          ${act.start_time ? `<p>📅 开始: ${new Date(act.start_time).toLocaleDateString()}</p>` : ''}
          ${act.end_time ? `<p>📅 结束: ${new Date(act.end_time).toLocaleDateString()}</p>` : ''}
        </div>
        ${rewardsHtml}
        <button class="activity-join-btn ${userJoined ? 'joined' : ''}" onclick="joinActivity('${act.id}')" ${statusClass === 'ended' ? 'disabled' : ''}>
          ${userJoined ? '✅ 已参与' : (statusClass === 'ended' ? '已结束' : '🎯 立即参与')}
        </button>
      </div>
    `;
    document.getElementById('activityModal').classList.remove('hidden');
  } catch (e) {
    showToast('❌ ' + e.message, true);
  }
}

function closeActivityModal() {
  document.getElementById('activityModal').classList.add('hidden');
}

async function joinActivity(activityId) {
  if (!state.token) { showToast('请先登录', true); return; }
  try {
    const res = await api('/api/v1/activities/' + activityId + '/join', { method: 'POST' });
    showToast('🎉 ' + (res.data?.message || '参与成功！'));
    showActivityDetail(activityId);
    loadActivityBanners();
  } catch (e) {
    showToast('❌ ' + e.message, true);
  }
}

async function claimActivityReward(activityId, rewardId) {
  if (!state.token) { showToast('请先登录', true); return; }
  try {
    const res = await api('/api/v1/activities/claim', {
      method: 'POST',
      body: JSON.stringify({ activity_id: activityId, reward_id: rewardId }),
    });
    showToast('🎉 领取成功！' + (res.data?.reward_name || ''));
    if (res.data?.new_points) {
      state.member.points = res.data.new_points;
      updatePoints();
    }
    showActivityDetail(activityId);
    loadActivityBanners();
  } catch (e) {
    showToast('❌ ' + e.message, true);
  }
}

// ============================================================
// AR 开盒动画增强
// ============================================================

// AR模式切换
document.addEventListener('DOMContentLoaded', function() {
  var arBtn = document.getElementById('arToggle');
  if (arBtn) {
    arBtn.onclick = function() {
      var boxAnim = document.getElementById('boxAnim');
      if (boxAnim) {
        boxAnim.classList.toggle('ar-mode');
        arBtn.textContent = boxAnim.classList.contains('ar-mode') ? '🔮 AR开启' : '🔮 AR模式';
      }
    };
  }
});

// ---- confetti 彩带特效 ----
const canvas = document.getElementById('confettiCanvas');
let ctx, animId, particles = [];

function initConfetti() {
  if (!canvas) return;
  canvas.width = window.innerWidth;
  canvas.height = window.innerHeight;
  ctx = canvas.getContext('2d');
}

function fireConfetti(count = 60) {
  initConfetti();
  if (!ctx) return;
  const colors = ['#f472b6', '#a78bfa', '#fbbf24', '#34d399', '#60a5fa', '#fb923c', '#fff'];
  for (let i = 0; i < count; i++) {
    particles.push({
      x: canvas.width / 2 + (Math.random() - 0.5) * 200,
      y: canvas.height / 2,
      vx: (Math.random() - 0.5) * 12,
      vy: -Math.random() * 14 - 4,
      size: Math.random() * 8 + 4,
      color: colors[Math.floor(Math.random() * colors.length)],
      rotation: Math.random() * 360,
      rotSpeed: (Math.random() - 0.5) * 10,
      gravity: 0.3,
      opacity: 1,
      decay: 0.008 + Math.random() * 0.01,
      shape: Math.random() > 0.5 ? 'rect' : 'circle',
    });
  }
  if (animId) cancelAnimationFrame(animId);
  animateConfetti();
}

function animateConfetti() {
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  let alive = false;
  for (const p of particles) {
    p.x += p.vx;
    p.vy += p.gravity;
    p.y += p.vy;
    p.rotation += p.rotSpeed;
    p.opacity -= p.decay;
    if (p.opacity <= 0) continue;
    alive = true;
    ctx.save();
    ctx.translate(p.x, p.y);
    ctx.rotate(p.rotation * Math.PI / 180);
    ctx.globalAlpha = Math.max(0, p.opacity);
    ctx.fillStyle = p.color;
    if (p.shape === 'rect') {
      ctx.fillRect(-p.size/2, -p.size/4, p.size, p.size/2);
    } else {
      ctx.beginPath();
      ctx.arc(0, 0, p.size/2, 0, Math.PI * 2);
      ctx.fill();
    }
    ctx.restore();
  }
  if (alive) {
    animId = requestAnimationFrame(animateConfetti);
  } else {
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    particles = [];
  }
}

window.addEventListener('resize', () => {
  if (canvas) { canvas.width = window.innerWidth; canvas.height = window.innerHeight; }
});

// showDrawResults 增强版
const originalShowDrawResults = window.showDrawResults;
showDrawResults = function(data, count) {
  const draws = data.draws || [];
  let show = draws[0] || {};
  const levels = { limited: 5, secret: 4, rare: 3, common: 1 };
  for (const d of draws) {
    if ((levels[d.prize_level] || 0) > (levels[show.prize_level] || 0)) show = d;
  }

  const level = show.prize_level || 'common';
  const name = show.prize_name || '神秘盲盒';

  // 先调用原始逻辑
  if (typeof originalShowDrawResults === 'function') {
    originalShowDrawResults(data, count);
  } else if (origShowDrawResults) {
    origShowDrawResults(data, count);
  }

  // 稀有度标签
  const labels = { common: '普通款', rare: '稀有款', secret: '隐藏款', limited: '限定款' };
  const emojis = { common: '🎁', rare: '🌟', secret: '💜', limited: '👑' };

  document.getElementById('resultRarity').textContent = labels[level] || '普通款';
  document.getElementById('resultRarity').className = 'rarity-badge ' + level;
  document.getElementById('resultIcon').textContent = emojis[level] || '🎁';
  document.getElementById('resultName').textContent = name;
  document.getElementById('resultName').className = 'result-name' + (level !== 'common' ? ' ' + level + '-name' : '');

  // 稀有度光效背景
  document.getElementById('boxResult').className = 'box-result ' + level + '-bg';

  // 稀有+彩带
  if (level === 'rare') fireConfetti(30);
  else if (level === 'secret') fireConfetti(80);
  else if (level === 'limited') fireConfetti(120);

  // 隐藏/限定光效文字
  if (level === 'secret' || level === 'limited') {
    document.getElementById('resultName').classList.add('rarity-text-glow');
  }

  // 动画揭晓（延迟触发）
  document.getElementById('resultIcon').style.transform = 'scale(0)';
  setTimeout(() => {
    document.getElementById('resultIcon').classList.add('result-reveal');
  }, 100);
};
