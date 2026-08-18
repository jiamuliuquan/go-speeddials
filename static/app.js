const state = {
  dials: [],
  groups: [],
  settings: {},
  authed: false,
  currentFolder: null,
  editingImage: '',
};

const grid = document.getElementById('grid');
const footer = document.getElementById('footer');

// ---------- 工具 ----------
async function api(url, method = 'GET', body) {
  const opts = { method, headers: {} };
  if (body !== undefined) {
    opts.headers['Content-Type'] = 'application/json';
    opts.body = JSON.stringify(body);
  }
  const res = await fetch(url, opts);
  if (!res.ok) {
    const data = await res.json().catch(() => ({}));
    const err = new Error(data.error || '请求失败');
    err.status = res.status;
    throw err;
  }
  return res.json();
}

function escapeHtml(s) {
  return String(s ?? '').replace(/[&<>"']/g, (c) => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
  }[c]));
}

function letterIcon(title) {
  return (title.trim()[0] || '?').toUpperCase();
}

function groupOptions(selectedId) {
  let html = `<option value="0">（根目录）</option>`;
  const groups = [...state.groups].sort((a, b) => a.sort - b.sort || a.id - b.id);
  for (const g of groups) {
    const sel = Number(selectedId) === g.id ? ' selected' : '';
    html += `<option value="${g.id}"${sel}>${escapeHtml(g.name)}</option>`;
  }
  return html;
}

// ---------- 首页渲染 ----------
function render() {
  grid.innerHTML = '';

  const folders = [...state.groups].sort((a, b) => a.sort - b.sort || a.id - b.id);
  const dials = state.dials
    .filter((d) => d.group_id === 0)
    .sort((a, b) => a.sort - b.sort || a.id - b.id);

  for (const g of folders) grid.appendChild(renderFolderCard(g));
  for (const d of dials) grid.appendChild(renderDialCard(d, 0));

  if (folders.length === 0 && dials.length === 0) {
    const empty = document.createElement('div');
    empty.className = 'grid-empty';
    empty.textContent = '暂无内容';
    grid.appendChild(empty);
  }
}

function renderDialCard(d, scope) {
  const item = document.createElement('a');
  item.className = 'dial-item';
  item.href = d.url;
  item.target = '_blank';
  item.rel = 'noopener noreferrer';

  const icon = document.createElement('div');
  icon.className = 'dial-icon';
  if (d.image) {
    const img = document.createElement('img');
    img.src = d.image;
    img.alt = d.title;
    img.onerror = () => { img.remove(); icon.textContent = letterIcon(d.title); };
    icon.appendChild(img);
  } else {
    icon.textContent = letterIcon(d.title);
  }

  const name = document.createElement('span');
  name.className = 'dial-name';
  name.textContent = d.title;

  item.appendChild(icon);
  item.appendChild(name);

  if (state.authed) attachTools(item, d);
  attachDrag(item, 'dial', d.id, scope);
  return item;
}

function renderFolderCard(g) {
  const item = document.createElement('div');
  item.className = 'dial-item folder';
  const icon = document.createElement('div');
  icon.className = 'dial-icon';
  icon.textContent = '📁';
  const name = document.createElement('span');
  name.className = 'dial-name';
  name.textContent = g.name;
  item.onclick = () => openFolderModal(g);
  item.appendChild(icon);
  item.appendChild(name);

  if (state.authed) attachTools(item, g);
  attachDrag(item, 'group', g.id, 0);
  return item;
}

// 编辑/删除图标（仅登录后）
function attachTools(item, obj) {
  const isDial = 'url' in obj;
  const tools = document.createElement('div');
  tools.className = 'tools';
  tools.innerHTML = `
    <button type="button" class="edit" title="编辑">✎</button>
    <button type="button" class="del" title="删除">×</button>
  `;
  tools.querySelector('.edit').onclick = (e) => {
    e.preventDefault(); e.stopPropagation();
    if (isDial) openDialModal(obj); else openGroupModal(obj);
  };
  tools.querySelector('.del').onclick = (e) => {
    e.preventDefault(); e.stopPropagation();
    if (isDial) {
      if (confirm(`确定删除「${obj.title}」？`)) deleteDial(obj.id);
    } else {
      if (confirm(`确定删除目录「${obj.name}」及其所有内容？`)) deleteGroup(obj.id);
    }
  };
  item.appendChild(tools);
}

// ---------- 拖拽排序（仅登录后） ----------
function attachDrag(item, type, id, scope) {
  item.draggable = state.authed;
  item.addEventListener('dragstart', (e) => {
    e.dataTransfer.setData('text/plain', JSON.stringify({ type, id }));
    e.dataTransfer.effectAllowed = 'move';
    item.classList.add('dragging');
  });
  item.addEventListener('dragend', () => {
    item.classList.remove('dragging');
  });
  item.addEventListener('dragover', (e) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
  });
  item.addEventListener('dragenter', (e) => {
    e.preventDefault();
    item.classList.add('drag-over');
  });
  item.addEventListener('dragleave', () => {
    item.classList.remove('drag-over');
  });
  item.addEventListener('drop', (e) => {
    e.preventDefault();
    item.classList.remove('drag-over');
    let src;
    try { src = JSON.parse(e.dataTransfer.getData('text/plain')); } catch (_) { return; }
    if (!src || src.type !== type || src.id === id) return;
    reorderItem(type, src.id, id, scope);
  });
}

async function reorderItem(type, srcId, targetId, scope) {
  let visible;
  if (type === 'dial') {
    visible = state.dials.filter((d) => d.group_id === scope);
  } else {
    visible = state.groups;
  }
  visible.sort((a, b) => a.sort - b.sort || a.id - b.id);

  const from = visible.findIndex((x) => x.id === srcId);
  const to = visible.findIndex((x) => x.id === targetId);
  if (from < 0 || to < 0 || from === to) return;

  const [moved] = visible.splice(from, 1);
  visible.splice(to, 0, moved);
  visible.forEach((x, i) => { x.sort = i; });

  try {
    await api('/api/reorder', 'PUT', { type, ids: visible.map((x) => x.id) });
    render();
    if (state.currentFolder) renderFolderDials();
  } catch (e) {
    alert(e.message);
  }
}

function renderFooter() {
  footer.innerHTML = state.settings.footer_html || '';
}

function applySiteTitle() {
  document.title = state.settings.site_title || '快速拨号';
  document.getElementById('site-title').textContent = state.settings.site_title || '快速拨号';
  document.getElementById('site-subtitle').textContent = state.settings.site_subtitle || '';
}

function applyColumns() {
  const n = parseInt(state.settings.columns, 10);
  const cols = Number.isFinite(n) && n >= 1 && n <= 12 ? n : 6;
  grid.style.setProperty('--cols', cols);
}

async function applyDailyImage() {
  if (state.settings.daily_image !== '1') {
    document.body.classList.remove('has-background');
    document.body.style.backgroundImage = '';
    return;
  }
  try {
    const data = await api('/api/daily');
    if (data.url) {
      document.body.classList.add('has-background');
      document.body.style.backgroundImage = `url("${data.url}")`;
    } else {
      document.body.classList.remove('has-background');
      document.body.style.backgroundImage = '';
    }
  } catch (e) {
    document.body.classList.remove('has-background');
    document.body.style.backgroundImage = '';
  }
}

// ---------- 头部按钮 ----------
function renderHeader() {
  document.getElementById('btn-login').classList.toggle('hidden', state.authed);
  document.getElementById('btn-settings').classList.toggle('hidden', !state.authed);
  document.getElementById('btn-logout').classList.toggle('hidden', !state.authed);
  document.getElementById('new-dropdown').classList.toggle('hidden', !state.authed);
  document.getElementById('new-menu').classList.add('hidden');
}

// ---------- 新建下拉 ----------
const newMenu = document.getElementById('new-menu');

document.getElementById('btn-new').onclick = (e) => {
  e.stopPropagation();
  newMenu.classList.toggle('hidden');
};

document.querySelectorAll('#new-menu [data-new]').forEach((btn) => {
  btn.onclick = () => {
    newMenu.classList.add('hidden');
    if (btn.dataset.new === 'dial') {
      openDialModal(null);
    } else {
      openGroupModal(null);
    }
  };
});

document.addEventListener('click', () => newMenu.classList.add('hidden'));

// ---------- 登录 ----------
const loginModal = document.getElementById('login-modal');

function openLogin() {
  document.getElementById('login-password').value = '';
  loginModal.classList.remove('hidden');
  document.getElementById('login-password').focus();
}

document.getElementById('login-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const username = document.getElementById('login-username').value.trim();
  const password = document.getElementById('login-password').value;
  try {
    await api('/api/login', 'POST', { username, password });
    state.authed = true;
    loginModal.classList.add('hidden');
    renderHeader();
    render();
  } catch (err) {
    alert(err.message);
  }
});

document.getElementById('btn-login-cancel').onclick = () => loginModal.classList.add('hidden');
loginModal.addEventListener('click', (e) => {
  if (e.target === loginModal) loginModal.classList.add('hidden');
});

async function doLogout() {
  try {
    await api('/api/logout', 'POST');
  } catch (e) { /* 忽略 */ }
  state.authed = false;
  closeFolderModal();
  settingsModal.classList.add('hidden');
  renderHeader();
  render();
}

// ---------- 设置弹窗 ----------
const settingsModal = document.getElementById('settings-modal');

function openSettings() {
  document.getElementById('setting-site-title').value = state.settings.site_title || '';
  document.getElementById('setting-site-subtitle').value = state.settings.site_subtitle || '';
  document.getElementById('setting-columns').value = state.settings.columns || '6';
  document.getElementById('setting-daily-image').checked = state.settings.daily_image === '1';
  document.getElementById('setting-footer-html').value = state.settings.footer_html || '';
  switchTab('basic');
  settingsModal.classList.remove('hidden');
}

function switchTab(name) {
  document.querySelectorAll('.tab').forEach((t) => t.classList.toggle('active', t.dataset.tab === name));
  document.getElementById('tab-basic').classList.toggle('hidden', name !== 'basic');
  document.getElementById('tab-footer').classList.toggle('hidden', name !== 'footer');
}

document.querySelectorAll('.tab').forEach((tab) => {
  tab.onclick = () => switchTab(tab.dataset.tab);
});

settingsModal.addEventListener('click', (e) => {
  if (e.target === settingsModal) settingsModal.classList.add('hidden');
});
document.getElementById('btn-settings-close').onclick = () => settingsModal.classList.add('hidden');

// 基本设置
document.getElementById('btn-save-settings').onclick = async () => {
  let columns = parseInt(document.getElementById('setting-columns').value, 10);
  if (!Number.isFinite(columns)) columns = 6;
  if (columns < 1) columns = 1;
  if (columns > 12) columns = 12;
  const payload = {
    site_title: document.getElementById('setting-site-title').value.trim(),
    site_subtitle: document.getElementById('setting-site-subtitle').value.trim(),
    columns: String(columns),
    daily_image: document.getElementById('setting-daily-image').checked ? '1' : '0',
  };
  try {
    await api('/api/settings', 'PUT', payload);
    state.settings.site_title = payload.site_title;
    state.settings.site_subtitle = payload.site_subtitle;
    state.settings.columns = payload.columns;
    state.settings.daily_image = payload.daily_image;
    applySiteTitle();
    applyColumns();
    applyDailyImage();
    alert('已保存');
  } catch (e) {
    alert(e.message);
  }
};

// 页脚设置
document.getElementById('btn-save-footer').onclick = async () => {
  const html = document.getElementById('setting-footer-html').value;
  try {
    await api('/api/settings', 'PUT', { footer_html: html });
    state.settings.footer_html = html;
    renderFooter();
    alert('已保存');
  } catch (e) {
    alert(e.message);
  }
};

// ---------- 目录内容弹窗 ----------
const folderModal = document.getElementById('folder-modal');

function openFolderModal(g) {
  state.currentFolder = g;
  document.getElementById('folder-title').textContent = g.name;
  document.getElementById('btn-folder-add').classList.toggle('hidden', !state.authed);
  renderFolderDials();
  folderModal.classList.remove('hidden');
}

function closeFolderModal() {
  state.currentFolder = null;
  folderModal.classList.add('hidden');
}

function renderFolderDials() {
  const box = document.getElementById('folder-dials');
  box.innerHTML = '';
  const g = state.currentFolder;
  if (!g) return;
  const dials = state.dials
    .filter((d) => d.group_id === g.id)
    .sort((a, b) => a.sort - b.sort || a.id - b.id);
  if (dials.length === 0) {
    box.innerHTML = '<div class="list-empty">该目录暂无内容</div>';
    return;
  }
  for (const d of dials) box.appendChild(renderDialCard(d, g.id));
}

folderModal.addEventListener('click', (e) => {
  if (e.target === folderModal) closeFolderModal();
});
document.getElementById('btn-folder-close').onclick = closeFolderModal;
document.getElementById('btn-folder-add').onclick = () => openDialModal(null);

// ---------- 增删 ----------
async function deleteDial(id) {
  try {
    await api(`/api/dials/${id}`, 'DELETE');
    state.dials = state.dials.filter((d) => d.id !== id);
    render();
    if (state.currentFolder) renderFolderDials();
  } catch (e) {
    alert(e.message);
  }
}

async function deleteGroup(id) {
  try {
    await api(`/api/groups/${id}`, 'DELETE');
    await loadGroupsAndDials();
    if (state.currentFolder && state.currentFolder.id === id) closeFolderModal();
    render();
  } catch (e) {
    alert(e.message);
  }
}

// ---------- 拨号项弹窗 ----------
const dialModal = document.getElementById('dial-modal');

function openDialModal(dial) {
  document.getElementById('dial-modal-title').textContent = dial ? '编辑拨号项' : '新增拨号项';
  document.getElementById('dial-id').value = dial ? dial.id : '';
  document.getElementById('dial-title').value = dial ? dial.title : '';
  document.getElementById('dial-url').value = dial ? dial.url : '';
  document.getElementById('dial-image').value = dial ? dial.image : '';
  const defaultGroup = dial ? dial.group_id : (state.currentFolder ? state.currentFolder.id : 0);
  document.getElementById('dial-group').innerHTML = groupOptions(defaultGroup);
  state.editingImage = dial ? dial.image : '';
  updateImagePreview();
  dialModal.classList.remove('hidden');
}

function closeDialModal() {
  dialModal.classList.add('hidden');
}

function updateImagePreview() {
  const url = document.getElementById('dial-image').value.trim() || state.editingImage;
  const box = document.getElementById('image-preview');
  box.innerHTML = url ? `<img src="${escapeHtml(url)}" alt="预览" />` : '';
}

// ---------- 目录弹窗 ----------
const groupModal = document.getElementById('group-modal');

function openGroupModal(group) {
  document.getElementById('group-modal-title').textContent = group ? '编辑目录' : '新增目录';
  document.getElementById('group-id').value = group ? group.id : '';
  document.getElementById('group-name').value = group ? group.name : '';
  groupModal.classList.remove('hidden');
}

function closeGroupModal() {
  groupModal.classList.add('hidden');
}

// ---------- 数据加载 ----------
async function loadGroupsAndDials() {
  const [dials, groups] = await Promise.all([api('/api/dials'), api('/api/groups')]);
  state.dials = dials;
  state.groups = groups;
}

async function loadAll() {
  const [dials, groups, settings, auth] = await Promise.all([
    api('/api/dials'),
    api('/api/groups'),
    api('/api/settings'),
    api('/api/auth'),
  ]);
  state.dials = dials;
  state.groups = groups;
  state.settings = settings;
  state.authed = !!auth.authenticated;
  render();
  renderFooter();
  applySiteTitle();
  applyColumns();
  applyDailyImage();
  renderHeader();
}

// ---------- 事件绑定 ----------
document.getElementById('btn-login').onclick = openLogin;
document.getElementById('btn-settings').onclick = openSettings;
document.getElementById('btn-logout').onclick = doLogout;

document.getElementById('btn-cancel').onclick = closeDialModal;
dialModal.addEventListener('click', (e) => {
  if (e.target === dialModal) closeDialModal();
});

document.getElementById('dial-image').addEventListener('input', updateImagePreview);

document.getElementById('dial-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const id = document.getElementById('dial-id').value;
  const payload = {
    title: document.getElementById('dial-title').value.trim(),
    url: document.getElementById('dial-url').value.trim(),
    image: document.getElementById('dial-image').value.trim(),
    group_id: Number(document.getElementById('dial-group').value),
  };
  try {
    if (id) {
      await api(`/api/dials/${id}`, 'PUT', payload);
    } else {
      payload.sort = state.dials.filter((d) => d.group_id === payload.group_id).length;
      await api('/api/dials', 'POST', payload);
    }
    closeDialModal();
    await loadGroupsAndDials();
    render();
    if (state.currentFolder) renderFolderDials();
  } catch (err) {
    alert(err.message);
  }
});

document.getElementById('btn-group-cancel').onclick = closeGroupModal;
groupModal.addEventListener('click', (e) => {
  if (e.target === groupModal) closeGroupModal();
});

document.getElementById('group-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const id = document.getElementById('group-id').value;
  const payload = {
    name: document.getElementById('group-name').value.trim(),
    parent_id: 0,
  };
  try {
    if (id) {
      await api(`/api/groups/${id}`, 'PUT', payload);
    } else {
      payload.sort = state.groups.length;
      await api('/api/groups', 'POST', payload);
    }
    closeGroupModal();
    await loadGroupsAndDials();
    render();
  } catch (err) {
    alert(err.message);
  }
});

// 图片上传
const fileInput = document.getElementById('file-input');
document.getElementById('btn-upload').onclick = () => fileInput.click();
fileInput.onchange = async () => {
  const file = fileInput.files[0];
  if (!file) return;
  const form = new FormData();
  form.append('file', file);
  try {
    const res = await fetch('/api/upload', { method: 'POST', body: form });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || '上传失败');
    document.getElementById('dial-image').value = data.url;
    state.editingImage = data.url;
    updateImagePreview();
  } catch (err) {
    alert(err.message);
  }
  fileInput.value = '';
};

// ---------- 启动 ----------
loadAll().catch((e) => alert('加载失败：' + e.message));
