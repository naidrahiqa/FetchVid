// FetchVid Frontend
// ============================================================
// Import Wails runtime (will be injected by Wails build)
// @ts-nocheck

// DOM refs
const $ = (s) => document.querySelector(s);
const $$ = (s) => document.querySelectorAll(s);

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
const tabs = $$('.tab');
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

function logMsg(msg, color) {
  const div = document.createElement('div');
  div.textContent = msg;
  if (color) div.style.color = `var(--${color})`;
  log.appendChild(div);
  log.scrollTop = log.scrollHeight;
}

// ============================================================
//  Platform detection
// ============================================================

function detectPlatform(url) {
  if (/facebook\.com|fb\.com|fb\.watch|web\.facebook/i.test(url)) return 'facebook';
  if (/instagram\.com|instagr\.am/i.test(url)) return 'instagram';
  if (/tiktok\.com|vm\.tiktok/i.test(url)) return 'tiktok';
  return 'unknown';
}

tabs.forEach(tab => {
  tab.addEventListener('click', () => {
    tabs.forEach(t => {
      t.style.background = 'var(--bg-secondary)';
      t.style.color = 'var(--text-secondary)';
    });
    tab.style.background = 'var(--accent)';
    tab.style.color = 'white';
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
    videoList.innerHTML = '<div class="text-center text-[var(--text-secondary)] text-xs py-8">Belum ada video.</div>';
    btnDownload.disabled = true;
    return;
  }

  entries.forEach((entry, i) => {
    const div = document.createElement('div');
    div.className = 'flex items-center gap-2 px-2 py-1 rounded hover:bg-[var(--bg-tertiary)] cursor-pointer selected';
    div.dataset.index = i;

    const cb = document.createElement('input');
    cb.type = 'checkbox';
    cb.checked = true;
    cb.className = 'accent-[var(--accent)]';

    const label = document.createElement('span');
    label.className = 'text-xs truncate flex-1';
    label.textContent = (entry.title || entry.url).slice(0, 70);

    const badge = document.createElement('span');
    badge.className = 'text-[10px] px-1.5 py-0.5 rounded font-medium';
    const colors = { facebook: '#1877f2', instagram: '#e4405f', tiktok: '#00f2ea' };
    badge.style.background = (colors[entry.source] || '#555') + '33';
    badge.style.color = colors[entry.source] || '#aaa';
    badge.textContent = (entry.source || '?').slice(0, 2).toUpperCase();

    div.prepend(badge);
    div.prepend(label);
    div.prepend(cb);
    videoList.appendChild(div);
  });

  btnDownload.disabled = false;
}

function getSelected() {
  const items = videoList.querySelectorAll('.selected');
  const selected = [];
  items.forEach(item => {
    const cb = item.querySelector('input[type="checkbox"]');
    const idx = parseInt(item.dataset.index);
    if (cb.checked && idx >= 0) selected.push(videoEntries[idx]);
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
//  Paste URLs
// ============================================================

btnPaste.addEventListener('click', () => {
  const text = prompt('Paste URL video (1 per baris):\n\nFacebook: /reel/ID\nInstagram: /reel/XXXX\nTikTok: /video/XXXX\n\nPisahkan dengan baris baru.');
  if (!text) return;

  const urls = text.split('\n').map(s => s.trim()).filter(s => s && !s.startsWith('#') && !s.startsWith('copy('));
  const entries = [];
  urls.forEach(url => {
    if (/facebook|fb\.com|instagram|tiktok/i.test(url)) {
      const source = detectPlatform(url);
      entries.push({ url, title: url.split('/').pop() || url.slice(0, 50), source });
    }
  });

  if (entries.length > 0) {
    renderList([...videoEntries, ...entries]);
    logMsg(`Ditambahkan ${entries.length} URL`, 'success');
  }
});

// ============================================================
//  Scripts dialog
// ============================================================

btnScripts.addEventListener('click', () => {
  const scripts = [
    { label: 'Profile - Reels', desc: 'Untuk akun personal Facebook', code: 'copy([...document.querySelectorAll(\'a[href*="/reel/"]\')].map(a=>a.href.split("?")[0]).filter((v,i,a)=>a.indexOf(v)===i).join(\'\\n\'))' },
    { label: 'Fanpage - Reels + Video', desc: 'Untuk fanpage Facebook', code: 'copy([...document.querySelectorAll(\'a[href*="/reel/"], a[href*="/video/"]\')].map(a=>a.href.split("?")[0]).filter((v,i,a)=>a.indexOf(v)===i).join(\'\\n\'))' },
    { label: 'Instagram - Reels', desc: 'Profile Instagram (semua reels)', code: 'copy([...document.querySelectorAll(\'a[href*="/reel/"], a[href*="/p/"]\')].map(a=>a.href.split("?")[0].split("/").slice(0,7).join("/")+"/").filter((v,i,a)=>a.indexOf(v)===i).join(\'\\n\'))' },
    { label: 'TikTok - Videos', desc: 'Profile TikTok (semua video)', code: 'copy([...document.querySelectorAll(\'a[href*="/video/"]\')].map(a=>a.href.split("?")[0]).filter((v,i,a)=>a.indexOf(v)===i).join(\'\\n\'))' },
    { label: 'Semua Video (FB)', desc: 'Ambil semua link video di halaman Facebook', code: 'copy([...document.querySelectorAll(\'a[href*="/reel/"], a[href*="/video/"], a[href*="/watch"]\')].map(a=>a.href.split("?")[0]).filter((v,i,a)=>a.indexOf(v)===i).join(\'\\n\'))' },
  ];

  let msg = 'SCRIPT CONSOLE - Copy & paste di browser (F12 > Console):\n\n';
  scripts.forEach(s => {
    msg += `[${s.label}]\n${s.desc}\n${s.code}\n\n`;
  });

  prompt(msg, scripts[0].code);
});

// ============================================================
//  Extract URLs (placeholder - will call Go)
// ============================================================

btnExtract.addEventListener('click', async () => {
  const url = urlInput.value.trim();
  if (!url) { logMsg('Masukkan URL profile!', 'error'); return; }

  logMsg(`Analisa: ${url}`, 'cyan');

  // Will call Go backend
  logMsg('Fitur scraping akan diimplementasikan di Go backend (platform-scraper)', 'warning');
});

// ============================================================
//  Download (placeholder)
// ============================================================

btnDownload.addEventListener('click', () => {
  if (isRunning) return;

  const selected = getSelected();
  if (selected.length === 0) { logMsg('Pilih video yang akan di-download!', 'warning'); return; }

  isRunning = true;
  btnDownload.textContent = 'DOWNLOADING...';
  btnDownload.disabled = true;
  progressContainer.style.display = 'block';

  logMsg(`Mulai download ${selected.length} video (bareng ${concurrent.value})...`, 'cyan');

  // Will call Go backend
  // For now, simulate
  let i = 0;
  const total = selected.length;
  const interval = setInterval(() => {
    i++;
    const pct = (i / total) * 100;
    progressBar.style.width = pct + '%';
    progressText.textContent = `${i} / ${total}`;
    progressPercent.textContent = `${Math.round(pct)}%`;
    logMsg(`[${i}/${total}] SELESAI (simulasi)`, 'success');

    if (i >= total) {
      clearInterval(interval);
      isRunning = false;
      btnDownload.textContent = 'DOWNLOAD VIDEO';
      btnDownload.disabled = false;
      logMsg('Selesai! (simulasi - implementasi Go backend menyusul)', 'green');
    }
  }, 500);
});

// ============================================================
//  Folder picker (placeholder)
// ============================================================

btnFolder.addEventListener('click', () => {
  logMsg('Folder picker akan diimplementasikan di Go backend (Wails runtime)', 'warning');
});

btnCookies.addEventListener('click', () => {
  logMsg('Cookies picker akan diimplementasikan di Go backend (Wails runtime)', 'warning');
});

// ============================================================
//  Init
// ============================================================

logMsg('FetchVid siap. Masukkan URL profile untuk memulai.');
