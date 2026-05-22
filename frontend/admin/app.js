// API 基础地址 & Token
let API = 'http://localhost:18100';
let AdminToken = '';

// ======================== 工具函数 ========================
async function api(path, opts = {}) {
  const url = API + path;
  const headers = { 'Content-Type': 'application/json' };
  if (AdminToken) headers['Authorization'] = 'Bearer ' + AdminToken;
  const res = await fetch(url, { ...opts, headers });
  const json = await res.json();
  if (json.code !== 'ok' && json.code !== 0) throw new Error(json.message || '请求失败');
  return json.data;
}

function showMsg(id, text, type) {
  const el = document.getElementById(id);
  if (!el) return;
  el.textContent = text;
  el.className = 'msg ' + type;
}
function clearMsg(id) {
  const el = document.getElementById(id);
  if (el) el.className = 'msg';
}
function closeModal(id) { document.getElementById(id).classList.add('hidden'); }

// ======================== 登录 ========================
async function adminLogin() {
  API = document.getElementById('apiBaseInput').value.replace(/\/+$/, '');
  const username = document.getElementById('usernameInput').value || 'admin';
  const password = document.getElementById('passwordInput').value || 'admin123';
  clearMsg('loginMsg');
  try {
    const data = await api('/api/v1/admin/login', {
      method: 'POST', body: JSON.stringify({ username, password })
    });
    AdminToken = data.token;
    document.getElementById('loginPanel').classList.add('hidden');
    document.getElementById('adminPanel').classList.remove('hidden');
    document.getElementById('loginStatus').textContent = '已登录';
    document.getElementById('logoutBtn').classList.remove('hidden');
    switchTab('overview');
  } catch (e) {
    showMsg('loginMsg', '登录失败: ' + e.message, 'error');
  }
}
function adminLogout() {
  AdminToken = '';
  document.getElementById('adminPanel').classList.add('hidden');
  document.getElementById('loginPanel').classList.remove('hidden');
  document.getElementById('loginStatus').textContent = '未登录';
  document.getElementById('logoutBtn').classList.add('hidden');
}

// ======================== 标签页切换 ========================
function switchTab(name) {
  document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
  document.querySelectorAll('.tab-panel').forEach(p => p.classList.add('hidden'));
  const tabBtn = [...document.querySelectorAll('.tab')].find(b => b.textContent.includes(
    { overview: '总览', campaigns: '活动', prizes: '礼品', pity: '概率', delivery: '发奖', records: '记录', monthcard: '月卡' }[name]
  ));
  if (tabBtn) tabBtn.classList.add('active');
  const panel = document.getElementById('tab' + name.charAt(0).toUpperCase() + name.slice(1));
  if (panel) panel.classList.remove('hidden');
  
  // 加载数据
  const loaders = {
    overview: loadOverview,
    campaigns: loadCampaigns,
    prizes: () => { populateCampaignSelect('prizeCampaignSelect'); loadPrizes(); },
    pity: () => { populateCampaignSelect('pityCampaignSelect'); },
    delivery: loadFulfillment,
    records: loadDrawRecords,
    monthcard: loadBattlePassStatus,
  };
  if (loaders[name]) loaders[name]();
}

// ======================== 总览 ========================
async function loadOverview() {
  const el = document.getElementById('overviewStats');
  try {
    const data = await api('/api/v1/admin/overview');
    el.innerHTML = `
      <div class="stat-card"><div class="stat-value">${data.total_users}</div><div class="stat-label">用户数</div></div>
      <div class="stat-card"><div class="stat-value">${data.total_draws}</div><div class="stat-label">抽奖次数</div></div>
      <div class="stat-card"><div class="stat-value">${data.total_wins}</div><div class="stat-label">中奖次数</div></div>
      <div class="stat-card"><div class="stat-value">${data.campaigns?.length || 0}</div><div class="stat-label">活动数</div></div>
    `;
    document.getElementById('overviewCampaigns').innerHTML = '<h3>活动列表</h3>' +
      (data.campaigns || []).map(c => `<div class="data-card"><div class="data-row"><div class="data-info">
        <span class="data-title">${c.name}</span>
        <span class="tag ${c.status === 'online' ? 'tag-online' : 'tag-draft'}">${c.status}</span>
        <div class="data-sub">${c.id} · 每日上限 ${c.daily_draw_limit} · 未中权重 ${c.miss_weight}</div>
      </div></div></div>`).join('');
    document.getElementById('overviewRecent').innerHTML = '<h3>最近抽奖</h3>' +
      ((data.recent_draws || []).slice(0, 10).map(r => `<div class="data-card"><div class="data-row">
        <div class="data-info"><span class="data-title">${r.prize_name}</span><span class="data-sub">${r.user_id?.slice(0,12)} · ${r.drawn_at?.slice(0,16) || ''}</span></div>
        <span class="tag ${r.result === 'win' ? 'tag-online' : ''}">${r.result}</span>
      </div></div>`).join('')) || '<p>暂无记录</p>';
  } catch (e) { el.textContent = '加载失败: ' + e.message; }
}

// ======================== 活动管理 ========================
async function loadCampaigns() {
  const el = document.getElementById('campaignList');
  try {
    const data = await api('/api/v1/admin/campaigns');
    el.innerHTML = data.map(c => `<div class="data-card">
      <div class="data-row">
        <div class="data-info">
          <div class="data-title">${c.name} <span class="tag ${c.status === 'online' ? 'tag-online' : c.status === 'draft' ? 'tag-draft' : 'tag-soldout'}">${c.status}</span></div>
          <div class="data-sub">ID: ${c.id} · 每日上限 ${c.daily_draw_limit} · 未中权重 ${c.miss_weight}</div>
          <div class="data-sub">${c.campaign_summary?.slice(0,60) || ''}</div>
        </div>
        <div class="data-actions">
          <button class="btn-outline" onclick="switchTab('pity'); document.getElementById('pityCampaignSelect').value='${c.id}'; loadPityConfig();">⚙️概率</button>
          <button class="btn-outline" onclick="switchTab('prizes'); document.getElementById('prizeCampaignSelect').value='${c.id}'; loadPrizes();">🏆礼品</button>
          <button class="btn-danger" onclick="deleteCampaign('${c.id}')">删除</button>
        </div>
      </div>
    </div>`).join('');
  } catch (e) { el.textContent = '加载失败: ' + e.message; }
}

function showCreateCampaign() {
  document.getElementById('createCampaignModal').classList.remove('hidden');
  clearMsg('createCampaignMsg');
}
async function createCampaign() {
  const body = {
    name: document.getElementById('newCampaignName').value,
    slug: document.getElementById('newCampaignSlug').value || document.getElementById('newCampaignName').value.toLowerCase().replace(/\s+/g, '-'),
    status: 'online',
    starts_at: new Date(Date.now() - 86400000).toISOString(),
    ends_at: new Date(Date.now() + 30*86400000).toISOString(),
    daily_draw_limit: parseInt(document.getElementById('newCampaignDailyLimit').value) || 10,
    miss_weight: parseInt(document.getElementById('newCampaignMissWeight').value) || 30,
    campaign_summary: document.getElementById('newCampaignSummary').value || '',
  };
  try {
    await api('/api/v1/admin/campaigns', { method: 'POST', body: JSON.stringify(body) });
    showMsg('createCampaignMsg', '创建成功', 'success');
    setTimeout(() => { closeModal('createCampaignModal'); loadCampaigns(); }, 800);
  } catch (e) { showMsg('createCampaignMsg', '创建失败: ' + e.message, 'error'); }
}
async function deleteCampaign(id) {
  if (!confirm('确定删除活动 ' + id + ' 吗？')) return;
  try {
    await api('/api/v1/admin/campaigns/' + id, { method: 'DELETE' });
    loadCampaigns();
  } catch (e) { alert('删除失败: ' + e.message); }
}

// ======================== 礼品管理 ========================
async function populateCampaignSelect(elId) {
  try {
    const data = await api('/api/v1/admin/campaigns');
    const el = document.getElementById(elId);
    el.innerHTML = '<option value="">-- 选择活动 --</option>' +
      data.map(c => `<option value="${c.id}">${c.name}</option>`).join('');
  } catch (e) {}
}

async function loadPrizes() {
  const campaignId = document.getElementById('prizeCampaignSelect').value;
  const el = document.getElementById('prizeList');
  if (!campaignId) { el.textContent = '请先选择一个活动'; return; }
  try {
    const data = await api(`/api/v1/admin/campaigns/${campaignId}/prizes`);
    el.innerHTML = data.map(p => `<div class="data-card">
      <div class="data-row">
        <div class="data-info">
          <div class="data-title">${p.name} <span class="tag tag-${p.level}">${p.level}</span></div>
          <div class="data-sub">ID: ${p.id} · 库存 ${p.stock} · 权重 ${p.probability_weight} · ${p.status}</div>
        </div>
        <div class="data-actions">
          <button class="btn-danger" onclick="deletePrize('${p.id}', '${campaignId}')">删除</button>
        </div>
      </div>
    </div>`).join('');
  } catch (e) { el.textContent = '加载失败: ' + e.message; }
}

function showCreatePrize() {
  const campaignId = document.getElementById('prizeCampaignSelect').value;
  if (!campaignId) { alert('请先选择活动'); return; }
  document.getElementById('createPrizeModal').classList.remove('hidden');
  clearMsg('createPrizeMsg');
}
async function createPrize() {
  const campaignId = document.getElementById('prizeCampaignSelect').value;
  const body = {
    name: document.getElementById('newPrizeName').value,
    level: document.getElementById('newPrizeLevel').value,
    stock: parseInt(document.getElementById('newPrizeStock').value) || 100,
    probability_weight: parseInt(document.getElementById('newPrizeWeight').value) || 10,
    status: 'active',
  };
  try {
    await api(`/api/v1/admin/campaigns/${campaignId}/prizes`, { method: 'POST', body: JSON.stringify(body) });
    showMsg('createPrizeMsg', '创建成功', 'success');
    setTimeout(() => { closeModal('createPrizeModal'); loadPrizes(); }, 800);
  } catch (e) { showMsg('createPrizeMsg', '创建失败: ' + e.message, 'error'); }
}
async function deletePrize(id, campaignId) {
  if (!confirm('确定删除礼品 ' + id + '？')) return;
  try { await api(`/api/v1/admin/campaigns/${campaignId}/prizes/` + id, { method: 'DELETE' }); loadPrizes(); }
  catch (e) { alert('删除失败: ' + e.message); }
}

// ======================== 概率/UP池配置 ========================
async function loadPityConfig() {
  const campaignId = document.getElementById('pityCampaignSelect').value;
  const form = document.getElementById('pityConfigForm');
  clearMsg('pityMsg');
  if (!campaignId) { form.classList.add('hidden'); return; }
  form.classList.remove('hidden');
  try {
    const cfg = await api(`/api/v1/admin/campaigns/${campaignId}/pity-config`);
    document.getElementById('pityEnabled').checked = cfg.enabled || false;
    document.getElementById('softPityN').value = cfg.soft_pity_n || 30;
    document.getElementById('pityFactor').value = cfg.pity_factor || 0.015;
    document.getElementById('hardPityN').value = cfg.hard_pity_n || 60;
    document.getElementById('targetPrize').value = cfg.target_prize || '';
    document.getElementById('upPoolEnabled').checked = cfg.up_pool_enabled || false;
    document.getElementById('upPrizeId').value = cfg.up_prize_id || '';
    document.getElementById('upMultiplier').value = cfg.up_multiplier || 5;
    if (cfg.up_level) document.getElementById('upLevel').value = cfg.up_level;
    // 时间字段格式化
    if (cfg.up_start_at) document.getElementById('upStartAt').value = cfg.up_start_at.slice(0,16);
    if (cfg.up_end_at) document.getElementById('upEndAt').value = cfg.up_end_at.slice(0,16);
  } catch (e) {
    // 可能没有配置（新活动），使用默认值
  }
}

async function savePityConfig() {
  const campaignId = document.getElementById('pityCampaignSelect').value;
  const toTime = (val) => val ? new Date(val).toISOString() : '';
  const body = {
    enabled: document.getElementById('pityEnabled').checked,
    soft_pity_n: parseInt(document.getElementById('softPityN').value) || 30,
    pity_factor: parseFloat(document.getElementById('pityFactor').value) || 0.015,
    hard_pity_n: parseInt(document.getElementById('hardPityN').value) || 60,
    target_prize: document.getElementById('targetPrize').value || '',
    up_pool_enabled: document.getElementById('upPoolEnabled').checked,
    up_prize_id: document.getElementById('upPrizeId').value || '',
    up_multiplier: parseFloat(document.getElementById('upMultiplier').value) || 5,
    up_level: document.getElementById('upLevel').value || 'secret',
    up_start_at: toTime(document.getElementById('upStartAt').value),
    up_end_at: toTime(document.getElementById('upEndAt').value),
  };
  try {
    await api(`/api/v1/admin/campaigns/${campaignId}/pity-config`, { method: 'PUT', body: JSON.stringify(body) });
    showMsg('pityMsg', '配置保存成功', 'success');
    setTimeout(() => clearMsg('pityMsg'), 2000);
  } catch (e) { showMsg('pityMsg', '保存失败: ' + e.message, 'error'); }
}

// ======================== 发奖管理 ========================
async function loadFulfillment() {
  const el = document.getElementById('fulfillmentList');
  try {
    const data = await api('/api/v1/admin/delivery/pending');
    el.innerHTML = data.length === 0 ? '<p>暂无待发奖记录</p>' :
      data.map(t => `<div class="data-card"><div class="data-row">
        <div class="data-info">
          <div class="data-title">任务 #${t.id}</div>
          <div class="data-sub">用户 ${t.user_id?.slice(0,12)} · 奖品 ${t.prize_id} · ${t.status}</div>
        </div>
        <div class="data-actions">
          <button class="btn-primary" onclick="approveFulfillment(${t.id})">审核通过</button>
        </div>
      </div></div>`).join('');
  } catch (e) { el.textContent = '加载失败: ' + e.message; }
}

async function approveFulfillment(id) {
  try {
    await api('/api/v1/admin/delivery/approve', { method: 'POST', body: JSON.stringify({ task_ids: [id] }) });
    loadFulfillment();
  } catch (e) { alert('操作失败: ' + e.message); }
}

// ======================== 抽奖记录 ========================
async function loadDrawRecords() {
  const el = document.getElementById('drawRecordsList');
  try {
    const data = await api('/api/v1/admin/lottery-logs');
    el.innerHTML = data.length === 0 ? '<p>暂无记录</p>' :
      '<div style="overflow-x:auto"><table><thead><tr><th>ID</th><th>用户</th><th>奖品</th><th>结果</th><th>时间</th></tr></thead><tbody>' +
      data.slice(0, 50).map(r => `<tr><td>${r.id?.slice(0,12)}</td><td>${r.user_id?.slice(0,12)}</td><td>${r.prize_name}</td><td><span class="tag ${r.result === 'win' ? 'tag-online' : ''}">${r.result}</span></td>
        <td>${r.drawn_at?.slice(0,16) || ''}</td></tr>`).join('') +
      '</tbody></table></div>';
  } catch (e) { el.textContent = '加载失败: ' + e.message; }
}

// ======================== 月卡/战令 ========================
async function loadBattlePassStatus() {
  const el = document.getElementById('bpSeasonStatus');
  el.textContent = '加载中...';
  try {
    const data = await api('/api/v1/battle-pass/info');
    const season = data.season;
    el.innerHTML = `
      <p><strong>${season.name}</strong> (赛季 #${season.id})</p>
      <p>等级上限: ${season.max_level} · 每级经验: ${season.xp_per_level}</p>
      <p>状态: ${season.status} · ${season.start_at?.slice(0,10) || ''} ~ ${season.end_at?.slice(0,10) || ''}</p>
      <p>任务数: ${(data.tasks || []).length} · 奖励数: ${(data.rewards || []).length}</p>
      ${data.user_pass ? `<p>你的进度: Lv.${data.user_pass.level} (${data.user_pass.xp}/${season.xp_per_level} XP) · ${data.user_pass.pass_type}</p>` : ''}
    `;
  } catch (e) { el.textContent = '获取失败: ' + e.message; }
}

// ======================== 初始化 ========================
// 回车登录
document.addEventListener('keydown', e => {
  if (e.key === 'Enter' && !document.getElementById('adminPanel').classList.contains('hidden')) return;
  if (e.key === 'Enter') adminLogin();
});
