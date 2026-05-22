const state = {
  apiBase: initialApiBase(),
  token: '',
};

const apiBaseInput = document.querySelector('#apiBase');
const usernameInput = document.querySelector('#username');
const passwordInput = document.querySelector('#password');
const authOutput = document.querySelector('#authOutput');
const overviewOutput = document.querySelector('#overviewOutput');
const recordOutput = document.querySelector('#recordOutput');

document.querySelector('#loginBtn').addEventListener('click', login);
document.querySelector('#overviewBtn').addEventListener('click', loadOverview);
document.querySelector('#recordsBtn').addEventListener('click', loadRecords);
apiBaseInput.value = state.apiBase;

async function login() {
  state.apiBase = apiBaseInput.value.trim();
  const payload = await request('/api/v1/admin/login', {
    method: 'POST',
    body: JSON.stringify({
      username: usernameInput.value.trim(),
      password: passwordInput.value,
    }),
  });
  state.token = payload.data.token;
  authOutput.textContent = JSON.stringify(payload.data, null, 2);
}

async function loadOverview() {
  if (!state.token) {
    overviewOutput.textContent = '请先登录后台';
    return;
  }

  const payload = await request('/api/v1/admin/overview', {
    headers: {
      Authorization: `Bearer ${state.token}`,
    },
  });
  overviewOutput.textContent = JSON.stringify(payload.data, null, 2);
}

async function loadRecords() {
  if (!state.token) {
    recordOutput.textContent = '请先登录后台';
    return;
  }

  const payload = await request('/api/v1/admin/draw-records', {
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
  authOutput.textContent = event.error ? event.error.message : event.message;
});

function initialApiBase() {
  if (window.location.protocol.startsWith('http')) {
    return `${window.location.origin}/api/campaign`;
  }
  return 'http://localhost:18100';
}
