// FetchVid Frontend - Connected to Go backend
// @ts-nocheck

import * as App from '../wailsjs/go/app/App.js';
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime.js';

// DOM refs
const $ = (s) => document.querySelector(s);
const urlInput = $('#urlInput');
const videoList = $('#videoList');
const count = $('#count');
const concurrent = $('#concurrent');
const btnDownload = $('#btnDownload');
const btnExtract = $('#btnExtract');
const btnScripts = $('#btnScripts');
const btnPaste = $('#btnPaste');
const btnClear = $('#btnClear');
const btnSelectAll = $('#btnSelectAll');
const btnDeselect = $('#btnDeselect');
const btnFolder = $('#btnFolder');
const btnCookies = $('#btnCookies');
const tabs = document.querySelectorAll('.tab');
const log = $('#log');
const progressContainer = $('#progressContainer');
const progressBar = $('#progressBar');
const progressText = $('#progressText');
const progressPercent = $('#progressPercent');

let videoEntries = [];
let isRunning = false;

// ============================================================
//  Logging
// ============================================================

function logMsg(msg, type) {
  const div = document.createElement('div');
  div.className = 'log-line' + (type ? ' ' + type : '');
  div.textContent = msg;
  log.appendChild(div);
  log.scrollTop = log.scrollHeight;
}

// ============================================================
//  Platform Tabs
// ============================================================

tabs.forEach(tab => {
  tab.addEventListener('click', () => {
    tabs.forEach(t => t.classList.remove('active'));
    tab.classList.add('active');
  });
});

// ============================================================
//  Video list
// ============================================================

function renderList(entries) {
  videoList.innerHTML = '';
  videoEntries = entries;
  count.textContent = entries.length;

  if (entries.length === 0) {
    videoList.innerHTML = '<div class="empty-state">Belum ada video.</div>';
    btnDownload.disabled = true;
    return;
  }

  entries.forEach((entry, i) => {
    const div = document.createElement('div');
    div.className = 'video-item';
    div.dataset.index = i;

    const cb = document.createElement('input');
    cb.type = 'checkbox';
    cb.checked = true;

    const label = document.createElement('span');
    label.className = 'title';
    label.textContent = (entry.title || entry.url || entry.Title || entry.URL || '').slice(0, 80);

    const badge = document.createElement('span');
    badge.className = 'badge';
    const src = entry.source || entry.Source || '';
    const colors = { facebook: '#1877f2', instagram: '#e4405f', tiktok: '#00f2ea' };
    badge.style.background = (colors[src] || '#555') + '33';
    badge.style.color = colors[src] || '#aaa';
    badge.textContent = (src || '?').slice(0, 2).toUpperCase();

    div.appendChild(cb);
    div.appendChild(label);
    div.appendChild(badge);
    videoList.appendChild(div);
  });

  btnDownload.disabled = false;
}

function getSelected() {
  const items = videoList.querySelectorAll('.video-item');
  const selected = [];
  items.forEach(item => {
    const cb = item.querySelector('input[type="checkbox"]');
    const idx = parseInt(item.dataset.index);
    if (cb && cb.checked && idx >= 0) selected.push(videoEntries[idx]);
  });
  return selected;
}

btnSelectAll.addEventListener('click', () => {
  videoList.querySelectorAll('input[type="checkbox"]').forEach(cb => cb.checked = true);
});

btnDeselect.addEventListener('click', () => {
  videoList.querySelectorAll('input[type="checkbox"]').forEach(cb => cb.checked = false);
});

btnClear.addEventListener('click', () => {
  renderList([]);
  logMsg('List dibersihkan', 'warning');
});

// ============================================================
//  Paste URLs (manual paste)
// ============================================================

btnPaste.addEventListener('click', () => {
  const text = prompt('Paste URL video (1 per baris):\n\nFacebook: /reel/ID\nInstagram: /reel/XXXX\nTikTok: /video/XXXX');
  if (!text) return;

  const urls = text.split('\n').map(s => s.trim()).filter(s => s && !s.startsWith('#') && !s.startsWith('copy('));
  const entries = [];
  urls.forEach(url => {
    const src = detectPlatform(url);
    if (src !== 'unknown') {
      entries.push({ URL: url, Title: url.split('/').pop() || url.slice(0, 50), Source: src });
    }
  });

  if (entries.length > 0) {
    renderList([...videoEntries, ...entries]);
    logMsg('Ditambahkan ' + entries.length + ' URL manual', 'success');
  }
});

function detectPlatform(url) {
  if (/facebook\.com|fb\.com|fb\.watch|web\.facebook/i.test(url)) return 'facebook';
  if (/instagram\.com|instagr\.am/i.test(url)) return 'instagram';
  if (/tiktok\.com|vm\.tiktok/i.test(url)) return 'tiktok';
  return 'unknown';
}

// ============================================================
//  Scripts dialog (get from Go backend)
// ============================================================

btnScripts.addEventListener('click', async () => {
  try {
    const scripts = await App.GetScripts();
    let msg = 'SCRIPT CONSOLE - Copy & paste di browser (F12 > Console):\n\n';
    if (scripts && scripts.length > 0) {
      scripts.forEach(s => {
        msg += '[' + s.label + ']\n' + s.desc + '\n' + s.script + '\n\n';
      });
    } else {
      msg += 'Tidak ada script tersedia.';
    }
    prompt(msg);
  } catch (e) {
    logMsg('Gagal load scripts: ' + e, 'error');
  }
});

// ============================================================
//  Extract URLs (Go backend)
// ============================================================

btnExtract.addEventListener('click', async () => {
  const url = urlInput.value.trim();
  if (!url) { logMsg('Masukkan URL profile!', 'error'); return; }

  logMsg('Analisa: ' + url, 'info');
  btnExtract.disabled = true;
  btnExtract.textContent = 'Loading...';

  try {
    const resp = await App.ExtractURLs(url);
    if (!resp.success) {
      logMsg(resp.message, 'error');
      return;
    }

    const data = resp.data || [];
    if (data.length === 0) {
      logMsg(resp.message || 'Tidak ada video ditemukan', 'warning');
      return;
    }

    renderList(data);
    logMsg('Ditemukan ' + data.length + ' video', 'success');
  } catch (e) {
    logMsg('Error: ' + e, 'error');
  } finally {
    btnExtract.disabled = false;
    btnExtract.textContent = 'Ambil Daftar Video';
  }
});

// ============================================================
//  Download (Go backend)
// ============================================================

btnDownload.addEventListener('click', async () => {
  if (isRunning) return;

  const selected = getSelected();
  if (selected.length === 0) { logMsg('Pilih video yang akan di-download!', 'warning'); return; }

  isRunning = true;
  btnDownload.textContent = 'DOWNLOADING...';
  btnDownload.disabled = true;
  progressContainer.classList.add('show');

  try {
    const queueResp = await App.QueueDownload(selected);
    if (!queueResp.success) {
      logMsg('Gagal queue: ' + queueResp.message, 'error');
      resetDownloadUI();
      return;
    }

    logMsg('Mulai download ' + selected.length + ' video (bareng ' + concurrent.value + ')...', 'info');

    const startResp = await App.StartDownload(parseInt(concurrent.value) || 3);
    if (!startResp.success) {
      logMsg('Gagal mulai: ' + startResp.message, 'error');
      resetDownloadUI();
      return;
    }
  } catch (e) {
    logMsg('Error: ' + e, 'error');
    resetDownloadUI();
  }
});

function resetDownloadUI() {
  isRunning = false;
  btnDownload.textContent = 'DOWNLOAD VIDEO';
  btnDownload.disabled = false;
}

// ============================================================
//  Wails Events
// ============================================================

EventsOn('download-progress', (p) => {
  const pct = p.Percent || 0;
  progressBar.style.width = pct + '%';
  progressText.textContent = (p.Completed || 0) + ' / ' + (p.Total || 0);
  progressPercent.textContent = Math.round(pct) + '%';
});

EventsOn('download-job-done', (job) => {
  if (job.Error) {
    logMsg('[' + job.Index + '] GAGAL: ' + job.Title + ' - ' + job.Error, 'error');
  } else {
    logMsg('[' + job.Index + '] SELESAI: ' + job.Title, 'success');
  }
});

EventsOn('download-complete', (p) => {
  logMsg('Selesai! ' + (p.Success || 0) + ' berhasil, ' + (p.Failed || 0) + ' gagal', 'success');
  resetDownloadUI();
  progressContainer.classList.remove('show');
});

// ============================================================
//  Folder picker (Go backend)
// ============================================================

btnFolder.addEventListener('click', async () => {
  try {
    const dir = await App.SelectFolder();
    if (dir) logMsg('Folder: ' + dir, 'info');
  } catch (e) {
    logMsg('Error: ' + e, 'error');
  }
});

// ============================================================
//  Cookies picker (Go backend)
// ============================================================

btnCookies.addEventListener('click', async () => {
  try {
    const file = await App.SelectCookiesFile();
    if (file) logMsg('Cookies: ' + file, 'info');
  } catch (e) {
    logMsg('Error: ' + e, 'error');
  }
});

// ============================================================
//  Init
// ============================================================

logMsg('FetchVid siap. Masukkan URL profile untuk memulai.');
