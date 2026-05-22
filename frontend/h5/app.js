const state = {
  apiBase: initialApiBase(),
  token: '',
  campaignID: '',
};

const apiBaseInput = document.querySelector('#apiBase');
const nicknameInput = document.querySelector('#nickname');
const sessionOutput = document.querySelector('#sessionOutput');
const drawOutput = document.querySelector('#drawOutput');
const recordOutput = document.querySelector('#recordOutput');
const campaignList = document.querySelector('#campaignList');

document.querySelector('#loginBtn').addEventListener('click', login);
document.querySelector('#loadCampaignsBtn').addEventListener('click', loadCampaigns);
document.querySelector('#loadRecordsBtn').addEventListener('click', loadRecords);
apiBaseInput.addEventListener('change', () => {
  state.apiBase = apiBaseInput.value.trim();
});

apiBaseInput.value = state.apiBase;

async function login() {
  state.apiBase = apiBaseInput.value.trim();
  const payload = await request('/api/v1/auth/guest-login', {
    method: 'POST',
    body: JSON.stringify({ nickname: nicknameInput.value.trim() }),
  });
  state.token = payload.data.session.token;
  sessionOutput.textContent = JSON.stringify(payload.data, null, 2);
}

async function loadCampaigns() {
  state.apiBase = apiBaseInput.value.trim();
  const payload = await request('/api/v1/campaigns');
  campaignList.innerHTML = '';
  payload.data.forEach((item) => {
    const card = document.createElement('article');
    card.className = 'campaign-card';
    state.campaignID = item.campaign.id;
    const prizes = item.prizes
      .map(
        (prize) => `
          <div class="prize-row">
            <span>${prize.level} · ${prize.name}</span>
            <span>库存 ${prize.stock} / 权重 ${prize.probability_weight}</span>
          </div>`
      )
      .join('');
    card.innerHTML = `
      <h3>${item.campaign.name}</h3>
      <p>${item.campaign.campaign_summary}</p>
      <p>每日抽奖次数：${item.campaign.daily_draw_limit}</p>
      <button data-campaign-id="${item.campaign.id}">立即抽奖</button>
      <div>${prizes}</div>`;
    card.querySelector('button').addEventListener('click', () => draw(item.campaign.id));
    campaignList.appendChild(card);
  });
}

async function draw(campaignID) {
  if (!state.token) {
    drawOutput.textContent = '请先游客登录';
    return;
  }

  const payload = await request('/api/v1/lottery/draw', {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${state.token}`,
    },
    body: JSON.stringify({ campaign_id: campaignID || state.campaignID }),
  });
  drawOutput.textContent = JSON.stringify(payload.data, null, 2);
}

async function loadRecords() {
  if (!state.token) {
    recordOutput.textContent = '请先游客登录';
    return;
  }

  const payload = await request('/api/v1/me/draw-records', {
    headers: {
      Authorization: `Bearer ${state.token}`,
    },
  });
  recordOutput.textContent = JSON.stringify(payload.data, null, 2);
}

async function request(path, options = {}) {
  const response = await fetch(`${state.apiBase}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers || {}),
    },
    ...options,
  });
  const payload = await response.json();
  if (!response.ok) {
    throw new Error(payload.message || 'request failed');
  }
  return payload;
}

window.addEventListener('error', (event) => {
  drawOutput.textContent = event.error ? event.error.message : event.message;
});

function initialApiBase() {
  if (window.location.protocol.startsWith('http')) {
    return `${window.location.origin}/api/campaign`;
  }
  return 'http://localhost:18100';
}
