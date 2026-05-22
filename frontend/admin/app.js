// ============================================================
// BOX·MAGIC 管理端
// ============================================================

const state = {
  apiBase: 'http://localhost:18100',
  token: '',
  campaigns: [],
  editCampaignId: null,
  editPrizeCampaignId: null,
  editPrizeId: null,
};

// ============================================================
// API 请求
// ============================================================

async function api(path, options = {}) {
  const headers = { 'Content-Type': 'application/json', ...(options.headers || {}) };
  if (state.token) headers['Authorization'] = `Bearer ${state.token}`;
  try {
    const res = await fetch(state.apiBase + path, { ...options, headers });
    const data = await res.json();
    if (!res.ok) throw new Error(data.message || `HTTP ${res.status}`);
    return data;
  } catch (e) {
    throw e;
  }
}

function showMsg(el, msg, isError = false) {
  el.textContent = msg;
  el.style.color = isError ? '#f87171' : '#34d399';
}

// ============================================================
// 登录
// ============================================================

async function adminLogin() {
  state.apiBase = document.getElementById('apiBaseInput').value.trim();
  const username = document.getElementById('usernameInput').value.trim();
  const password = document.getElementById('passwordInput').value;
  const msg = document.getElementById('loginMsg');
  try {
    const res = await api('/api/v1/admin/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    });
    state.token = res.data.token;
    document.getElementById('loginPanel').classList.add('hidden');
    document.getElementById('adminPanel').classList.remove('hidden');
    document.getElementById('loginStatus').textContent = '已登录';
    document.getElementById('loginStatus').style.background = 'rgba(52,211,153,0.2)';
    loadOverview();
    loadCampaigns();
  } catch (e) {
    showMsg(msg, '❌ ' + e.message, true);
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
  switch (tab) {
    case 'overview': loadOverview(); break;
    case 'series': loadCampaigns(); break;
    case 'records': loadRecords(); break;
    case 'fulfill': loadFulfillment(); break;
  }
}

// ============================================================
// 总览
// ============================================================

async function loadOverview() {
  const el = document.getElementById('overviewContent');
  try {
    const [overviewRes, statsRes] = await Promise.all([
      api('/api/v1/admin/overview'),
      api('/api/v1/admin/statistics'),
    ]);
    const ov = overviewRes.data;
    const st = statsRes.data;

    el.innerHTML = `
      <div class="stats-grid">
        <div class="stat-card"><div class="num">${ov.total_users || 0}</div><div class="label">用户数</div></div>
        <div class="stat-card"><div class="num">${ov.total_draws || 0}</div><div class="label">总抽奖</div></div>
        <div class="stat-card"><div class="num">${ov.total_wins || 0}</div><div class="label">中奖数</div></div>
        <div class="stat-card"><div class="num">${st ? st.win_rate?.toFixed(1) : '0'}%</div><div class="label">中奖率</div></div>
      </div>
      <div class="panel">
        <h2>🎯 奖品出货分布</h2>
        ${st?.prize_breakdown?.length ? `
          <table><tr><th>奖品</th><th>级别</th><th>出货</th><th>占比</th></tr>
          ${st.prize_breakdown.map(p => `
            <tr><td>${p.prize_name}</td><td><span class="prize-tag prize-${p.level}">${p.level}</span></td>
            <td>${p.count}</td><td>${p.percent?.toFixed(1)}%</td></tr>
          `).join('')}</table>
        ` : '<div class="loading">暂无数据</div>'}
      </div>
      <div class="panel">
        <h2>📈 最新抽奖记录</h2>
        ${ov.recent_draws?.length ? `
          <table><tr><th>用户</th><th>系列</th><th>结果</th><th>奖品</th><th>时间</th></tr>
          ${ov.recent_draws.slice(0, 8).map(r => `
            <tr><td>${(r.user_id || '').slice(0, 10)}...</td><td>${r.campaign_id}</td>
            <td class="${r.result}">${r.result === 'win' ? '✅中奖' : '❌未中'}</td>
            <td>${r.prize_name}</td>
            <td style="font-size:11px;color:var(--muted)">${new Date(r.drawn_at).toLocaleString()}</td></tr>
          `).join('')}</table>
        ` : '<div class="loading">暂无记录</div>'}
      </div>
    `;
  } catch (e) {
    el.innerHTML = `<div class="loading">❌ ${e.message}</div>`;
  }
}

// ============================================================
// 系列管理
// ============================================================

async function loadCampaigns() {
  const el = document.getElementById('seriesContent');
  try {
    const res = await api('/api/v1/admin/campaigns');
    state.campaigns = res.data || [];
    renderCampaigns();
  } catch (e) {
    el.innerHTML = `<div class="loading">❌ ${e.message}</div>`;
  }
}

function renderCampaigns() {
  const el = document.getElementById('seriesContent');
  if (!state.campaigns.length) {
    el.innerHTML = '<div class="loading">暂无系列</div>';
    return;
  }
  el.innerHTML = state.campaigns.map(c => {
    const pity = c.pity_config;
    const pityInfo = pity?.enabled ? `<span class="pity-badge">保底: ${pity.soft_pity_n || '-'}/${pity.hard_pity_n || '-'}</span>` : '';
    return `
      <div class="series-card">
        <div class="head">
          <h3>${c.name} ${pityInfo}</h3>
          <span style="font-size:12px;color:var(--muted)">${c.status}</span>
        </div>
        <div class="meta">
          <span>Miss权重: ${c.miss_weight}</span>
          <span>每日次数: ${c.daily_draw_limit}</span>
          <span>${c.campaign_summary || ''}</span>
        </div>
        <div class="actions">
          <button class="btn-sm" onclick="editCampaign('${c.id}')">✏️ 编辑</button>
          <button class="btn-sm" onclick="togglePrizes('${c.id}')">🎁 奖品</button>
          <button class="btn-sm" onclick="editPity('${c.id}')">🎯 保底</button>
          <button class="btn-danger" onclick="deleteCampaign('${c.id}')">删除</button>
        </div>
        <div id="prizes-${c.id}" class="hidden" style="margin-top:8px;border-top:1px solid rgba(255,255,255,0.06);padding-top:8px;">
          <div id="prizeList-${c.id}"></div>
          <button class="btn-sm" onclick="showCreatePrize('${c.id}')" style="margin-top:6px;">+ 添加奖品</button>
        </div>
      </div>`;
  }).join('');
}

async function showCreateCampaign() {
  state.editCampaignId = null;
  document.getElementById('seriesModalTitle').textContent = '新建系列';
  document.getElementById('sName').value = '';
  document.getElementById('sSlug').value = '';
  document.getElementById('sStatus').value = 'online';
  document.getElementById('sLimit').value = '10';
  document.getElementById('sMissW').value = '72';
  document.getElementById('sSummary').value = '';
  document.getElementById('pEnabled').value = 'false';
  document.getElementById('pSoft').value = '60';
  document.getElementById('pHard').value = '90';
  document.getElementById('pFactor').value = '0.015';
  document.getElementById('pTarget').value = '';
  document.getElementById('seriesModal').classList.remove('hidden');
}

async function editCampaign(id) {
  const c = state.campaigns.find(x => x.id === id);
  if (!c) return;
  state.editCampaignId = id;
  document.getElementById('seriesModalTitle').textContent = '编辑系列';
  document.getElementById('sName').value = c.name;
  document.getElementById('sSlug').value = c.slug;
  document.getElementById('sStatus').value = c.status;
  document.getElementById('sLimit').value = c.daily_draw_limit;
  document.getElementById('sMissW').value = c.miss_weight;
  document.getElementById('sSummary').value = c.campaign_summary || '';
  const pity = c.pity_config || {};
  document.getElementById('pEnabled').value = pity.enabled ? 'true' : 'false';
  document.getElementById('pSoft').value = pity.soft_pity_n || 60;
  document.getElementById('pHard').value = pity.hard_pity_n || 90;
  document.getElementById('pFactor').value = pity.pity_factor || 0.015;
  document.getElementById('pTarget').value = pity.target_prize || '';
  document.getElementById('seriesModal').classList.remove('hidden');
}

async function saveCampaign() {
  const data = {
    name: document.getElementById('sName').value,
    slug: document.getElementById('sSlug').value,
    status: document.getElementById('sStatus').value,
    daily_draw_limit: parseInt(document.getElementById('sLimit').value) || 0,
    miss_weight: parseInt(document.getElementById('sMissW').value) || 0,
    campaign_summary: document.getElementById('sSummary').value,
    pity_config: {
      enabled: document.getElementById('pEnabled').value === 'true',
      soft_pity_n: parseInt(document.getElementById('pSoft').value) || 0,
      hard_pity_n: parseInt(document.getElementById('pHard').value) || 0,
      pity_factor: parseFloat(document.getElementById('pFactor').value) || 0,
      target_prize: document.getElementById('pTarget').value,
    },
  };
  // 加上时间
  data.starts_at = new Date(Date.now() - 86400000).toISOString();
  data.ends_at = new Date(Date.now() + 30 * 86400000).toISOString();

  try {
    if (state.editCampaignId) {
      await api(`/api/v1/admin/campaigns/${state.editCampaignId}`, {
        method: 'PUT', body: JSON.stringify(data),
      });
    } else {
      await api('/api/v1/admin/campaigns', {
        method: 'POST', body: JSON.stringify(data),
      });
    }
    closeSeriesModal();
    loadCampaigns();
  } catch (e) {
    alert('❌ ' + e.message);
  }
}

function closeSeriesModal() {
  document.getElementById('seriesModal').classList.add('hidden');
}

async function deleteCampaign(id) {
  if (!confirm('确定删除此系列？')) return;
  try {
    await api(`/api/v1/admin/campaigns/${id}`, { method: 'DELETE' });
    loadCampaigns();
  } catch (e) {
    alert('❌ ' + e.message);
  }
}

// ============================================================
// 奖品管理
// ============================================================

async function togglePrizes(campaignId) {
  const el = document.getElementById('prizes-' + campaignId);
  if (el.classList.contains('hidden')) {
    el.classList.remove('hidden');
    try {
      const res = await api(`/api/v1/admin/campaigns/${campaignId}/prizes`);
      const prizes = res.data || [];
      document.getElementById('prizeList-' + campaignId).innerHTML = prizes.map(p => `
        <span class="prize-tag prize-${p.level}">
          ${p.name} (${p.level}) 库存${p.stock} 权重${p.probability_weight}
          <button class="btn-sm" style="margin-left:4px;padding:2px 6px;font-size:10px;" onclick="editPrize('${campaignId}','${p.id}')">✏️</button>
          <button style="background:var(--red);border:0;border-radius:4px;padding:2px 6px;color:white;font-size:10px;cursor:pointer;" onclick="deletePrize('${p.id}')">×</button>
        </span>
      `).join('');
    } catch (e) {
      document.getElementById('prizeList-' + campaignId).innerHTML = '❌ ' + e.message;
    }
  } else {
    el.classList.add('hidden');
  }
}

function showCreatePrize(campaignId) {
  state.editPrizeCampaignId = campaignId;
  state.editPrizeId = null;
  document.getElementById('prizeModalTitle').textContent = '新建奖品';
  document.getElementById('pName').value = '';
  document.getElementById('pLevel').value = 'common';
  document.getElementById('pStock').value = '100';
  document.getElementById('pWeight').value = '10';
  document.getElementById('prizeModal').classList.remove('hidden');
}

async function editPrize(campaignId, prizeId) {
  state.editPrizeCampaignId = campaignId;
  state.editPrizeId = prizeId;
  document.getElementById('prizeModalTitle').textContent = '编辑奖品';
  // 通过 API 获取当前奖品信息
  try {
    const res = await api(`/api/v1/admin/campaigns/${campaignId}/prizes`);
    const prize = (res.data || []).find(p => p.id === prizeId);
    if (prize) {
      document.getElementById('pName').value = prize.name;
      document.getElementById('pLevel').value = prize.level;
      document.getElementById('pStock').value = prize.stock;
      document.getElementById('pWeight').value = prize.probability_weight;
      document.getElementById('prizeModal').classList.remove('hidden');
    }
  } catch (e) {
    alert('❌ ' + e.message);
  }
}

async function savePrize() {
  const data = {
    name: document.getElementById('pName').value,
    level: document.getElementById('pLevel').value,
    stock: parseInt(document.getElementById('pStock').value) || 0,
    probability_weight: parseInt(document.getElementById('pWeight').value) || 0,
    status: 'active',
  };
  const cid = state.editPrizeCampaignId;
  try {
    if (state.editPrizeId) {
      await api(`/api/v1/admin/prizes/${state.editPrizeId}`, {
        method: 'PUT', body: JSON.stringify(data),
      });
    } else {
      await api(`/api/v1/admin/campaigns/${cid}/prizes`, {
        method: 'POST', body: JSON.stringify(data),
      });
    }
    closePrizeModal();
    loadCampaigns();
  } catch (e) {
    alert('❌ ' + e.message);
  }
}

async function deletePrize(prizeId) {
  if (!confirm('确定删除？')) return;
  try {
    await api(`/api/v1/admin/prizes/${prizeId}`, { method: 'DELETE' });
    loadCampaigns();
  } catch (e) {
    alert('❌ ' + e.message);
  }
}

function closePrizeModal() {
  document.getElementById('prizeModal').classList.add('hidden');
}

// ============================================================
// 保底配置快捷编辑
// ============================================================

async function editPity(campaignId) {
  const enabled = prompt('启用保底? (true/false)', 'true');
  if (enabled === null) return;
  const soft = parseInt(prompt('软保底次数 (默认60)', '60')) || 60;
  const hard = parseInt(prompt('硬保底次数 (默认90)', '90')) || 90;
  const factor = parseFloat(prompt('递增因子 (默认0.015)', '0.015')) || 0.015;

  try {
    await api(`/api/v1/admin/campaigns/${campaignId}/pity-config`, {
      method: 'PUT',
      body: JSON.stringify({
        enabled: enabled === 'true',
        soft_pity_n: soft,
        hard_pity_n: hard,
        pity_factor: factor,
      }),
    });
    loadCampaigns();
  } catch (e) {
    alert('❌ ' + e.message);
  }
}

// ============================================================
// 抽奖记录
// ============================================================

async function loadRecords() {
  const el = document.getElementById('recordsContent');
  try {
    const res = await api('/api/v1/admin/draw-records');
    const records = res.data || [];
    el.innerHTML = records.length ? `
      <table><tr><th>用户</th><th>系列</th><th>结果</th><th>奖品</th><th>剩余次数</th><th>时间</th></tr>
      ${records.slice(0, 50).map(r => `
        <tr><td style="font-size:11px;">${(r.user_id || '').slice(0, 12)}..</td>
        <td style="font-size:11px;">${(r.campaign_id || '').slice(0, 16)}</td>
        <td class="${r.result}">${r.result === 'win' ? '✅' : '❌'}</td>
        <td>${r.prize_name}</td>
        <td>${r.chance_after !== undefined ? r.chance_after : '-'}</td>
        <td style="font-size:11px;color:var(--muted)">${new Date(r.drawn_at).toLocaleString()}</td></tr>
      `).join('')}</table>
    ` : '<div class="loading">暂无记录</div>';
  } catch (e) {
    el.innerHTML = `<div class="loading">❌ ${e.message}</div>`;
  }
}

// ============================================================
// 发奖管理
// ============================================================

async function loadFulfillment() {
  const el = document.getElementById('fulfillContent');
  try {
    const res = await api('/api/v1/admin/fulfillment-tasks');
    const tasks = res.data || [];
    el.innerHTML = tasks.length ? `
      <table><tr><th>ID</th><th>用户</th><th>奖品</th><th>状态</th><th>备注</th><th>操作</th></tr>
      ${tasks.map(t => `
        <tr><td style="font-size:11px;">${t.draw_record_id?.slice(0, 12) || t.id}</td>
        <td style="font-size:11px;">${(t.user_id || '').slice(0, 12)}..</td>
        <td>${t.prize_id?.slice(0, 10) || '-'}</td>
        <td><span class="${t.status === 'fulfilled' ? 'win' : ''}">${t.status}</span></td>
        <td style="font-size:11px;color:var(--muted);max-width:100px;overflow:hidden;white-space:nowrap;text-overflow:ellipsis">${t.operator_note || '-'}</td>
        <td>
          <button class="btn-sm" onclick="fulfillTask(${t.id}, 'fulfilled')">✅ 完成</button>
          <button class="btn-danger" onclick="fulfillTask(${t.id}, 'rejected')">驳回</button>
        </td></tr>
      `).join('')}</table>
    ` : '<div class="loading">暂无发奖任务</div>';
  } catch (e) {
    el.innerHTML = `<div class="loading">❌ ${e.message}</div>`;
  }
}

async function fulfillTask(taskId, status) {
  const note = prompt(status === 'rejected' ? '驳回原因：' : '备注（可选）：', '');
  try {
    await api(`/api/v1/admin/fulfillment-tasks/${taskId}`, {
      method: 'PATCH',
      body: JSON.stringify({ status, operator_note: note || '' }),
    });
    loadFulfillment();
  } catch (e) {
    alert('❌ ' + e.message);
  }
}
