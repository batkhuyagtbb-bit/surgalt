/* Surgalt — интерактив давхарга (build алхамгүй, цэвэр JS) */
(() => {
"use strict";
const $ = (s, r = document) => r.querySelector(s);
const $$ = (s, r = document) => [...r.querySelectorAll(s)];
const reduce = matchMedia("(prefers-reduced-motion: reduce)").matches;
const page = document.body.dataset.page;
const esc = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
const money = (n) => (n ? Number(n).toLocaleString("en-US") + "₮" : "Үнэгүй");
// Бүтцийн дүрс (самбар, гарчиг, жагсаалт): нэг хэв маягийн шугаман SVG — emoji шиг төхөөрөмжөөс хамаарч өөрчлөгдөхгүй.
const UI_ICONS = {
  courses: '<path d="m2 9 10-5 10 5-10 5z"/><path d="M6 11.5V16c0 1.2 2.7 3 6 3s6-1.8 6-3v-4.5"/>',
  book: '<path d="M4 5a2 2 0 0 1 2-2h14v16H6a2 2 0 0 0-2 2z"/><path d="M4 21V5M9 7h7"/>',
  live: '<rect x="3" y="6" width="12" height="12" rx="2"/><path d="m15 10 6-3v10l-6-3z"/>',
  chat: '<path d="M21 12a8 8 0 0 1-11.6 7.1L4 20l1-4.6A8 8 0 1 1 21 12Z"/>',
  users: '<circle cx="9" cy="8" r="3.5"/><path d="M2.5 20c0-3.5 3-5.5 6.5-5.5s6.5 2 6.5 5.5"/><path d="M16 4.6a3.5 3.5 0 0 1 0 6.8M18 14.7c2 .7 3.5 2.4 3.5 5.3"/>',
  teacher: '<circle cx="12" cy="7" r="3.5"/><path d="M5 21c0-4 3.1-6.5 7-6.5s7 2.5 7 6.5"/><path d="M15 3.5h5v3"/>',
  play: '<circle cx="12" cy="12" r="9"/><path d="m10 8.5 5.5 3.5-5.5 3.5z"/>',
  clip: '<path d="m20 11-8.5 8.5a5 5 0 0 1-7-7L13 4a3.3 3.3 0 0 1 4.7 4.7l-8.4 8.4a1.7 1.7 0 0 1-2.4-2.4L14.5 7"/>',
  exam: '<rect x="5" y="3" width="14" height="18" rx="2"/><path d="M9 3.5V5h6V3.5M9 10h6M9 14h6M9 18h3"/>',
  medal: '<circle cx="12" cy="15" r="5"/><path d="M8.5 11 6 3h4l2 5 2-5h4l-2.5 8"/><path d="m12 13 .9 1.8 2 .3-1.4 1.4.3 2-1.8-1-1.8 1 .3-2-1.4-1.4 2-.3z"/>',
  chart: '<path d="M4 20V10M10 20V4M16 20v-7M22 20H2"/>',
  money: '<rect x="3" y="6" width="18" height="12" rx="2"/><circle cx="12" cy="12" r="2.5"/><path d="M7 9v.01M17 15v.01"/>',
  gear: '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1Z"/>',
  eye: '<path d="M2 12s3.6-7 10-7 10 7 10 7-3.6 7-10 7S2 12 2 12Z"/><circle cx="12" cy="12" r="3"/>',
  lock: '<rect x="5" y="11" width="14" height="9" rx="2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/>',
  check: '<path d="m5 12.5 4.5 4.5L19 7.5"/>',
  clock: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
  slides: '<rect x="3" y="4" width="18" height="12" rx="2"/><path d="M12 16v4M8 20h8M8 9h8M8 12h5"/>',
  doc: '<path d="M6 3h8l4 4v14H6z"/><path d="M14 3v4h4M9 12h6M9 16h6"/>',
  sheet: '<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M3 10h18M3 15h18M9 4v16M15 4v16"/>',
  down: '<path d="M12 4v11M7 10l5 5 5-5M5 20h14"/>',
  chev: '<path d="M6 9l6 6 6-6"/>',
  next: '<path d="M9 6l6 6-6 6"/>',
  expand: '<path d="M4 9V4h5M20 9V4h-5M4 15v5h5M20 15v5h-5"/>',
};
const icon = (n, size = 18) => `<svg viewBox="0 0 24 24" width="${size}" height="${size}" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${UI_ICONS[n] || ""}</svg>`;
// Огноог өөрсдөө хэлбэржүүлнэ: олон браузерт mn-MN локаль байхгүй тул англиар гардаг.
const WEEKDAYS = ["Ням", "Даваа", "Мягмар", "Лхагва", "Пүрэв", "Баасан", "Бямба"];
const WEEKDAYS_SHORT = ["Ня", "Да", "Мя", "Лх", "Пү", "Ба", "Бя"];
const pad2 = (n) => String(n).padStart(2, "0");
const fmtTime = (t) => { const d = new Date(t); return pad2(d.getHours()) + ":" + pad2(d.getMinutes()); };
// Секундийг хүний хэлээр: 45 сек, 12 мин, 1 ц 05 мин
const dur = (s) => { s = Math.max(0, Math.round(+s || 0)); if (s < 60) return s + " сек"; const h = Math.floor(s / 3600), m = Math.round((s % 3600) / 60); return h ? `${h} ц ${String(m).padStart(2, "0")} мин` : `${m} мин`; };
const fmtDay = (t, year) => { const d = new Date(t); return (year ? d.getFullYear() + " оны " : "") + (d.getMonth() + 1) + "-р сарын " + d.getDate(); };
const fmtDate = (t) => {
  const d = new Date(t), now = new Date();
  const days = Math.round((new Date(d).setHours(0, 0, 0, 0) - new Date(now).setHours(0, 0, 0, 0)) / 864e5);
  const day = days === 0 ? "Өнөөдөр" : days === 1 ? "Маргааш" : days === -1 ? "Өчигдөр" : fmtDay(d, d.getFullYear() !== now.getFullYear());
  return day + " " + fmtTime(d);
};
const store = {
  get(k) { try { return JSON.parse(localStorage.getItem(k)); } catch { return null; } },
  set(k, v) { try { v == null ? localStorage.removeItem(k) : localStorage.setItem(k, JSON.stringify(v)); } catch {} },
};

/* ---------- auth + API ---------- */
const Auth = {
  get token() { return store.get("sg_token"); },
  get user() { return store.get("sg_user"); },
  get guest() { return store.get("sg_guest"); },
  set(token, user) { store.set("sg_token", token); store.set("sg_user", user); },
  clear() { store.set("sg_token", null); store.set("sg_user", null); },
  // Чатад: нэвтэрсэн бол хэрэглэгчийн, үгүй бол зочны токен
  get chatToken() { return this.token || this.guest?.token || null; },
};

async function api(path, { method = "GET", body, token = Auth.token, raw } = {}) {
  const headers = {};
  if (token) headers.Authorization = "Bearer " + token;
  if (body !== undefined && !(body instanceof FormData)) headers["Content-Type"] = "application/json";
  const res = await fetch(path, { method, headers, body: body instanceof FormData ? body : body !== undefined ? JSON.stringify(body) : undefined });
  if (raw) return res;
  const data = res.status === 204 ? null : await res.json().catch(() => null);
  if (!res.ok) {
    if (res.status === 401 && token && token === Auth.token) { Auth.clear(); }
    const e = new Error(data?.error || data?.message || "Алдаа гарлаа (" + res.status + ")");
    e.status = res.status; e.data = data; throw e;
  }
  return data;
}

function toast(msg, err) {
  const t = document.createElement("div");
  t.className = "toast" + (err ? " err" : "");
  t.textContent = msg;
  $("#toasts")?.append(t);
  setTimeout(() => { t.style.transition = ".4s"; t.style.opacity = "0"; t.style.transform = "translateY(10px)"; }, 3200);
  setTimeout(() => t.remove(), 3700);
}

/* ---------- кино маягийн хөдөлгөөн ---------- */
function splitText() {
  $$(".split").forEach((el) => {
    if (el.dataset.split) return;
    el.dataset.split = 1;
    const words = el.textContent.trim().split(/\s+/);
    el.innerHTML = words.map((w, i) => `<span class="w"><span style="--i:${i}">${esc(w)}</span></span>`).join(" ");
  });
}

function reveals() {
  const io = new IntersectionObserver((es) => es.forEach((e) => {
    if (e.isIntersecting) { e.target.classList.add("in"); io.unobserve(e.target); }
  }), { threshold: 0.12, rootMargin: "0px 0px -40px" });
  $$(".reveal:not(.in), .split:not(.in)").forEach((el) => io.observe(el));
}

function counters() {
  const io = new IntersectionObserver((es) => es.forEach((e) => {
    if (!e.isIntersecting) return;
    io.unobserve(e.target);
    const end = +e.target.dataset.count || 0, dur = reduce ? 0 : 1600, t0 = performance.now();
    const step = (t) => {
      const p = dur ? Math.min(1, (t - t0) / dur) : 1;
      e.target.textContent = Math.round(end * (1 - Math.pow(1 - p, 4))).toLocaleString("en-US");
      if (p < 1) requestAnimationFrame(step);
    };
    requestAnimationFrame(step);
  }), { threshold: 0.5 });
  $$("[data-count]").forEach((el) => io.observe(el));
}

function tilt() {
  if (reduce || matchMedia("(hover: none)").matches) return;
  document.addEventListener("pointermove", (e) => {
    const el = e.target.closest?.(".tilt");
    $$(".tilt.tilting").forEach((t) => { if (t !== el) { t.classList.remove("tilting"); t.style.setProperty("--rx", "0deg"); t.style.setProperty("--ry", "0deg"); } });
    if (!el) return;
    const r = el.getBoundingClientRect(), x = (e.clientX - r.left) / r.width, y = (e.clientY - r.top) / r.height;
    el.classList.add("tilting");
    el.style.setProperty("--ry", (x - 0.5) * 12 + "deg");
    el.style.setProperty("--rx", (0.5 - y) * 10 + "deg");
    el.style.setProperty("--gx", x * 100 + "%");
    el.style.setProperty("--gy", y * 100 + "%");
  }, { passive: true });
}

function navScroll() {
  const nav = $(".nav");
  const onScroll = () => nav?.classList.toggle("scrolled", scrollY > 20);
  addEventListener("scroll", onScroll, { passive: true }); onScroll();
}
function ambient() { // (хуучин: parallax + соронзон товч) — ашиглахаа больсон
  if (reduce) return;
  const para = $$("[data-parallax]");
  let raf = 0, mx = 0.5, my = 0.3;
  addEventListener("pointermove", (e) => {
    mx = e.clientX / innerWidth; my = e.clientY / innerHeight;
    if (!raf) raf = requestAnimationFrame(() => {
      raf = 0;
      document.documentElement.style.setProperty("--mx", mx * 100 + "%");
      document.documentElement.style.setProperty("--my", my * 100 + "%");
      para.forEach((el) => { const k = +el.dataset.parallax; el.style.translate = `${(mx - 0.5) * k}px ${(my - 0.5) * k}px`; });
    });
  }, { passive: true });
  // Соронзон товч
  $$(".magnetic").forEach((b) => {
    b.addEventListener("pointermove", (e) => { const r = b.getBoundingClientRect(); b.style.translate = `${(e.clientX - r.left - r.width / 2) * 0.18}px ${(e.clientY - r.top - r.height / 2) * 0.25}px`; });
    b.addEventListener("pointerleave", () => { b.style.translate = ""; });
  });
}

function curtain() {
  const c = $("#curtain");
  if (!c) return;
  const key = "sg_curtain:" + location.pathname;
  if (reduce || sessionStorage.getItem(key)) { c.classList.add("skip"); $$(".split").forEach((s) => s.style.setProperty("--base", "0ms")); return; }
  sessionStorage.setItem(key, 1);
  $$(".hero .split").forEach((s) => s.style.setProperty("--base", "1200ms"));
  $$(".hero .reveal").forEach((s) => s.style.transitionDelay = "1.35s");
  addEventListener("keydown", () => c.classList.add("skip"), { once: true });
  c.addEventListener("click", () => c.classList.add("skip"));
}

/* ---------- Material ripple: дарсан цэгээс долгион тархана ---------- */
const RIPPLE = ".btn,.pf-tab,.pill,.tile,.stat-tile,.tab,.icon-btn,.rail-item,.side-nav button,.side-foot a,.side-foot button,.item,.person,.group,.idcard-actions a,.idcard-actions button";
function ripples() {
  if (reduce) return;
  document.addEventListener("pointerdown", (e) => {
    const el = e.target.closest?.(RIPPLE);
    if (!el || el.disabled) return;
    const r = el.getBoundingClientRect(), d = Math.max(r.width, r.height) * 2.2;
    if (getComputedStyle(el).position === "static") el.style.position = "relative";
    el.style.overflow = "hidden";
    const s = document.createElement("span");
    s.className = "ripple";
    s.style.cssText = `width:${d}px;height:${d}px;left:${e.clientX - r.left - d / 2}px;top:${e.clientY - r.top - d / 2}px`;
    el.append(s);
    s.addEventListener("animationend", () => s.remove());
  }, { passive: true });
}

/* ---------- modal ---------- */
function openModal(el) { if (!el) return; el.classList.add("open"); el.setAttribute("aria-hidden", "false"); }
function closeModal(el) {
  if (!el) return;
  el.classList.remove("open"); el.setAttribute("aria-hidden", "true");
  $$("video,audio", el).forEach((v) => v.pause());
  const p = $(".player", el); if (p) setTimeout(() => { if (!el.classList.contains("open")) p.innerHTML = ""; }, 400);
}
document.addEventListener("click", (e) => {
  const m = e.target.closest(".modal");
  if (m && (e.target === m || e.target.closest("[data-close]"))) closeModal(m);
});
addEventListener("keydown", (e) => { if (e.key === "Escape") $$(".modal.open").forEach(closeModal); });

/* ---------- WebSocket: чат + мэдэгдэл (нэг холболт) ---------- */
const Live = {
  ws: null, token: null, tries: 0, handlers: new Set(),
  on(fn) { this.handlers.add(fn); },
  // Бүлэгт шинээр орсны дараа (сувгууд холбогдох үед тооцогддог) дахин холбогдоно.
  reconnect() { const t = this.token || Auth.chatToken; const ws = this.ws; this.ws = null; try { ws?.close(); } catch {} this.token = null; this.connect(t); },
  connect(token) {
    if (!token || (this.ws && this.token === token && this.ws.readyState <= 1)) return;
    this.token = token;
    try { this.ws?.close(); } catch {}
    const ws = new WebSocket((location.protocol === "https:" ? "wss://" : "ws://") + location.host + "/api/chat/ws?token=" + encodeURIComponent(token));
    this.ws = ws;
    ws.onopen = () => { this.tries = 0; };
    ws.onmessage = (e) => { let d; try { d = JSON.parse(e.data); } catch { return; } this.handlers.forEach((h) => h(d)); };
    ws.onclose = () => {
      if (this.ws !== ws) return;
      // Экспоненциал backoff + jitter: сервер дахин эхлэхэд мянга мянган клиент нэг зэрэг давхцахгүй.
      const delay = Math.min(30000, 1000 * 2 ** this.tries++) * (0.5 + Math.random());
      setTimeout(() => { if (this.ws === ws) { this.ws = null; this.connect(this.token); } }, delay);
    };
  },
};

/* ---------- Чатын самбар: нэвтэрсэн хүн бүрт баруун талд байнга ----------
   Багш: суралцагчид, зочид, бүлгүүд. Суралцагч: багш нартайгаа хийсэн яриа, элссэн сургалтын бүлгүүд.
   Өргөн дэлгэцэд хуудасны баруун талд наалдсан; нарийн дэлгэцэд дээд цэсний товчоор гулсаж гарна. */
function chatRail() {
  const u = Auth.user;
  if (!u || !Auth.token || !$(".nav-links") || !["home", "profile", "course", "me"].includes(page)) return;
  const teacher = u.role === "teacher";
  const I = {
    chat: `<svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a8 8 0 0 1-11.6 7.1L4 20l1-4.6A8 8 0 1 1 21 12Z"/></svg>`,
    x: `<svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M6 6l12 12M18 6 6 18"/></svg>`,
    back: `<svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 6l-6 6 6 6"/></svg>`,
    search: `<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/></svg>`,
    live: `<svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linejoin="round"><rect x="3" y="6" width="12" height="12" rx="2"/><path d="m15 10 6-3v10l-6-3z"/></svg>`,
    group: `<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><circle cx="9" cy="8" r="3.2"/><path d="M3 19c0-3.2 2.7-5 6-5s6 1.8 6 5"/><circle cx="17" cy="9" r="2.4"/><path d="M15.5 14.3c3-.3 5.5 1.5 5.5 4.2"/></svg>`,
    people: `<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><circle cx="8" cy="8" r="3"/><circle cx="16" cy="8" r="3"/><path d="M2.5 19c0-3 2.5-5 5.5-5s5.5 2 5.5 5M10.5 19c0-3 2.5-5 5.5-5s5.5 2 5.5 5"/></svg>`,
    plus: `<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"><path d="M12 5v14M5 12h14"/></svg>`,
    collapse: `<svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m7 6 6 6-6 6M13 6l6 6-6 6"/></svg>`,
  };
  const sameDay = (t) => new Date(t).toDateString() === new Date().toDateString();
  // Хумьсан хавтас "би энд байна": шинэ мессеж ирэхэд нэг удаа дохино.
  const nudgeTab = () => { const tab = $("#railTab"); if (!tab) return; tab.classList.remove("nudge"); void tab.offsetWidth; tab.classList.add("nudge"); };
  document.documentElement.classList.add("has-rail");
  document.body.insertAdjacentHTML("beforeend", `<aside class="rail" id="studioRail" aria-label="Чат">
    <div class="rail-view" id="railHome">
      <header class="rail-head"><strong><i></i>Чат</strong><button class="btn btn-sm rail-group-btn" id="railNewGroup" title="${teacher ? "Сургалтын бүлэг чат нээх" : "Ангийн найзуудтайгаа бүлэг үүсгэх"}">${I.plus}Бүлэг чат</button><button class="icon-btn rail-close" data-rail-close aria-label="Чат хумих" title="Хумих">${I.collapse}</button></header>
      <label class="rail-search">${I.search}<input type="search" id="railSearch" placeholder="Нэрээр хайх…" aria-label="Чат хайх"></label>
      <div class="rail-list" id="railList">${[1, 2, 3, 4].map(() => `<div class="rail-skel"><i></i><span><b></b><b></b></span></div>`).join("")}</div></div>
    <div class="rail-view" id="railThread" hidden>
      <header class="rail-head"><button class="icon-btn" id="railBack" aria-label="Жагсаалт руу буцах">${I.back}</button><div class="grow" id="railWho"></div><button class="icon-btn rail-close" data-rail-close aria-label="Чат хаах">${I.x}</button></header>
      <div class="chat-body" id="railBody"><ol class="chat-msgs" id="railMsgs"></ol></div>
      <form class="chat-input" id="railForm">${teacher ? `<button type="button" id="railMeet" data-plus>${I.live} Google Meet үүсгээд илгээх</button>` : ""}</form></div>
  </aside><div class="rail-scrim" id="railScrim"></div>`);
  $(".nav-links").insertAdjacentHTML("afterbegin", `<button class="icon-btn rail-toggle" id="railToggle" aria-label="Чат" aria-controls="studioRail" aria-expanded="false">${I.chat}<span class="bell-badge" id="railBadge" hidden></span></button>`);
  const rail = $("#studioRail"), home = $("#railHome"), thread = $("#railThread"), msgs = $("#railMsgs"), body = $("#railBody"), form = $("#railForm");
  let convs = [], unread = new Map(), timer = 0, cur = null, curGroup = false, myRole = "visitor";
  // Өргөн дэлгэцэд чат баруун талд наалддаг; хумих (⟩) товчоор нуугдаж, баруун ирмэгийн хавтсаар дахин нээгдэнэ.
  const wide = () => matchMedia("(min-width:1280px)").matches;
  document.body.insertAdjacentHTML("beforeend", `<button class="rail-tab" id="railTab" aria-label="Чат нээх" title="Чат нээх"><span class="rail-tab-ico">${I.chat}<i class="rail-tab-dot"></i></span><span class="rail-tab-txt">Чат</span><span class="bell-badge" id="railTabBadge" hidden></span></button>`);
  const setCollapsed = (c) => { document.documentElement.classList.toggle("rail-collapsed", c); try { localStorage.setItem("sg_rail_collapsed", c ? "1" : "0"); } catch {} };
  try { if (localStorage.getItem("sg_rail_collapsed") === "1") setCollapsed(true); } catch {}
  const setRail = (open) => {
    if (wide()) { setCollapsed(!open); }
    document.body.classList.toggle("rail-open", open && !wide()); $("#railToggle").setAttribute("aria-expanded", String(open));
  };
  const badge = () => { let n = 0; unread.forEach((v) => (n += v)); for (const id of ["railBadge", "railTabBadge"]) { const b = $("#" + id); if (b) { b.hidden = !n; b.textContent = n > 99 ? "99+" : n; } } document.documentElement.classList.toggle("rail-has-unread", n > 0); };
  $("#railToggle").onclick = () => setRail(wide() ? document.documentElement.classList.contains("rail-collapsed") : !document.body.classList.contains("rail-open"));
  $("#railTab").onclick = () => setRail(true);
  $("#railScrim").onclick = () => setRail(false);
  rail.addEventListener("click", (e) => { if (e.target.closest("[data-rail-close]")) setRail(false); });
  addEventListener("keydown", (e) => { if (e.key === "Escape") setRail(false); });

  // Нэг хэлбэрт оруулна: {id, title, sub, kind, avatar, last, at}
  const norm = async () => {
    const out = new Map();
    if (teacher) {
      const list = await api("/api/me/conversations").catch(() => []);
      list.forEach((c) => out.set(c.id, { id: c.id, kind: c.kind || "", title: c.visitor_name, sub: c.kind === "group" ? "Бүлэг · сургалтын суралцагчид" : c.user_id ? "Суралцагч" : "Зочин", last: c.last_message, at: c.last_message_at }));
    }
    const h = await api("/api/me/home").catch(() => null);
    (h?.chats || []).forEach((c) => { if (!out.has(c.id)) out.set(c.id, { id: c.id, kind: c.kind || "", title: c.kind === "group" || c.kind === "team" ? c.title : c.teacher?.display_name || "", sub: c.kind === "group" ? "Бүлэг · " + (c.teacher?.display_name || "") : c.kind === "team" ? "Сурагчдын бүлэг" : c.kind === "dm" ? "Ангийн найз" : "Багш", user: c.kind === "team" ? null : c.teacher, last: c.last_message, at: c.last_message_at }); });
    return [...out.values()].sort((a, b) => new Date(b.at) - new Date(a.at));
  };
  // Хүмүүс: багшид — суралцагчид нь, суралцагчид — багш нар нь (яриа эхлээгүй байсан ч нэг дарж эхэлнэ).
  let people = [];
  const loadPeople = async () => {
    try {
      if (teacher) {
        const rows = await api("/api/me/students");
        people = rows.map((r) => ({ id: r.user.id, user: r.user, name: r.user.display_name, sub: "Суралцагч" + (r.courses?.[0]?.title ? " · " + r.courses[0].title : ""), convId: r.conv_id || "", start: () => api(`/api/me/students/${r.user.id}/chat`, { method: "POST" }) }));
      } else {
        const [h, mates] = await Promise.all([api("/api/me/home"), api("/api/me/classmates").catch(() => [])]);
        people = (h.teachers || []).map((t) => ({ id: t.id, user: t, name: t.display_name, sub: "Багш" + (t.headline ? " · " + t.headline : ""), convId: "", start: () => api(`/api/teachers/${t.username}/chat`, { method: "POST" }) }))
          .concat(mates.map((m) => ({ id: m.user.id, user: m.user, name: m.user.display_name, sub: "Ангийн найз · " + (m.via || m.courses?.[0] || ""), convId: m.conv_id || "", mate: true, start: () => api(`/api/me/dm/${m.user.id}`, { method: "POST" }) })));
      }
    } catch { people = []; }
  };
  let firstDraw = true;
  const draw = () => {
    const anim = firstDraw ? "rail-in" : ""; firstDraw = false;
    const q = $("#railSearch").value.trim().toLowerCase(), hit = (s) => !q || s.toLowerCase().includes(q);
    const shown = convs.filter((c) => hit(c.title));
    const convUsers = new Set(convs.map((c) => c.user?.id).filter(Boolean));
    const ppl = people.filter((p) => hit(p.name) && !(p.convId && convs.some((c) => c.id === p.convId)) && !convUsers.has(p.id));
    let i = 0;
    const item = (c) => { const n = unread.get(c.id) || 0; return `<button class="rail-item ${anim} ${n ? "unread" : ""} ${c.id === cur ? "active" : ""}" style="--i:${i++}" data-id="${esc(c.id)}">${c.kind === "group" || c.kind === "team" ? `<span class="avatar avatar-sm group ${c.kind}">${c.kind === "team" ? I.people : I.group}</span>` : c.user ? avatarHTML(c.user) : `<span class="avatar avatar-sm" style="--h:${hueOfName(c.title)}">${esc(c.title.slice(0, 1).toUpperCase())}</span>`}
      <span class="grow"><span class="rail-row1"><strong>${esc(c.title)}</strong>${c.at ? `<time class="rail-time">${sameDay(c.at) ? fmtTime(c.at) : fmtDay(c.at)}</time>` : ""}</span><small>${c.last ? esc(c.last) : `<i>${esc(c.sub)}</i>`}</small></span>${n ? `<span class="rail-unread">${n > 9 ? "9+" : n}</span>` : ""}</button>`; };
    const person = (p) => `<button class="rail-item ${anim} rail-person" style="--i:${Math.min(i++, 12)}" data-person="${esc(p.id)}">${avatarHTML(p.user)}<span class="grow"><strong>${esc(p.name)}</strong><small><b>${esc(p.sub)}</b></small></span><span class="rail-new" aria-hidden="true">${I.chat}</span></button>`;
    const sec = (title, n) => `<div class="rail-sec"><span>${title}</span><em>${n}</em></div>`;
    $("#railList").innerHTML = (shown.length ? sec("Сүүлийн яриа", shown.length) + shown.map(item).join("") : "") + (ppl.length ? sec("Хүмүүс", ppl.length) + ppl.map(person).join("") : "") ||
      `<p class="muted small" style="padding:16px">${q ? "Илэрц алга" : teacher ? "Одоогоор чат алга. Профайлаа түгээгээрэй!" : "Багшийн профайл дээрх «Чатлах» эсвэл сургалтын «Бүлэг чат»-аар яриа эхэлнэ."}</p>`;
  };
  const load = async () => { convs = await norm(); draw(); };
  const startWith = async (pid) => {
    const p = people.find((x) => x.id === pid); if (!p) return;
    if (p.convId) return open(p.convId);
    try { const d = await p.start(); const id = d.conversation?.id || d.id; p.convId = id; await load(); open(id); } catch (e) { toast(e.message, true); }
  };
  $("#railNewGroup")?.addEventListener("click", async (e) => {
    $(".rail-pop")?.remove();
    const pop = document.createElement("div"); pop.className = "rail-pop";
    e.currentTarget.insertAdjacentElement("afterend", pop);
    if (teacher) {
      let courses = []; try { courses = await api("/api/me/courses"); } catch {}
      pop.innerHTML = `<b>Сургалтын бүлэг чат</b>${courses.map((c) => `<button type="button" data-course="${esc(c.id)}">👥 ${esc(c.title)}</button>`).join("") || `<p class="muted small">Сургалт алга</p>`}`;
      pop.onclick = async (ev) => { const b = ev.target.closest("[data-course]"); if (!b) return; pop.remove(); try { const d = await api(`/api/courses/${b.dataset.course}/chat`, { method: "POST" }); Live.reconnect(); await load(); open(d.conversation.id); } catch (x) { toast(x.message, true); } };
    } else {
      const mates = people.filter((p) => p.mate);
      pop.innerHTML = `<b>Ангийн найзуудтайгаа бүлэг үүсгэх</b><form class="team-form"><input name="title" maxlength="60" required placeholder="Бүлгийн нэр (ж: Математикийн баг)">
        <div class="team-list">${mates.map((m) => `<label><input type="checkbox" name="m" value="${esc(m.id)}">${avatarHTML(m.user)}<span>${esc(m.name)}<small>${esc(m.sub)}</small></span></label>`).join("") || `<p class="muted small">Ангийн найз хараахан алга — нэг багшид дагасан сурагчид энд гарна.</p>`}</div>
        <button class="btn btn-gold btn-sm btn-block">Бүлэг үүсгэх</button></form>`;
      pop.querySelector("form").onsubmit = async (ev) => {
        ev.preventDefault();
        const f = ev.target, members = [...f.querySelectorAll("input[name=m]:checked")].map((x) => x.value);
        if (!members.length) { toast("Дор хаяж нэг найзаа сонгоно уу", true); return; }
        try { const d = await api("/api/me/teams", { method: "POST", body: { title: f.title.value.trim(), members } }); pop.remove(); Live.reconnect(); await load(); open(d.conversation.id); toast("👥 Бүлэг үүслээ"); }
        catch (x) { toast(x.message, true); }
      };
      setTimeout(() => pop.querySelector("input[name=title]")?.focus(), 50);
    }
    setTimeout(() => document.addEventListener("click", (ev) => { if (!pop.contains(ev.target)) pop.remove(); }, { once: true }), 0);
  });
  const th = new ChatThread({ ol: msgs, body, form, token: () => Auth.token, convId: () => cur, role: () => myRole, group: () => curGroup });
  const append = (m) => th.append(m);
  const open = async (id) => {
    cur = id; unread.delete(id); badge(); setRail(true); draw();
    home.hidden = true; thread.hidden = false;
    $("#railWho").innerHTML = `<span class="muted small">Ачаалж байна…</span>`; msgs.innerHTML = "";
    try {
      const d = await api(`/api/chat/${id}/messages`);
      if (cur !== id) return;
      const c = d.conversation; curGroup = c.kind === "group" || c.kind === "team"; myRole = d.me;
      const known = convs.find((x) => x.id === id);
      $("#railWho").innerHTML = `<strong>${esc(known?.title || c.visitor_name)}</strong><small class="muted">${c.kind === "team" ? "Сурагчдын бүлэг" : c.kind === "dm" ? "Ангийн найз" : curGroup ? "Бүлэг чат · сургалтын суралцагчид" : known?.sub || (c.user_id ? "Суралцагч" : "Зочин")}</small>`;
      th.set(d); form.body.focus();
    } catch (e) { $("#railWho").innerHTML = `<span class="form-error" style="margin:0">${esc(e.message)}</span>`; }
  };
  $("#railBack").onclick = () => { cur = null; thread.hidden = true; home.hidden = false; draw(); };
  $("#railSearch").addEventListener("input", draw);
  $("#railList").addEventListener("click", (e) => { const p = e.target.closest("[data-person]"); if (p) return startWith(p.dataset.person); const it = e.target.closest(".rail-item"); if (it) open(it.dataset.id); });
  form.addEventListener("click", async (e) => {
    if (!e.target.closest("#railMeet") || !cur) return;
    try { const d = await api(`/api/chat/${cur}/meet`, { method: "POST" }); append(d.message); toast("Meet холбоос илгээгдлээ"); }
    catch (x) { toast(x.message, true); }
  });
  Live.on((d) => {
    if (["reaction", "read", "typing"].includes(d.type)) { th.event(d); return; }
    if (d.type !== "message") return;
    const m = d.message;
    if (m.conversation_id === cur) append(m);
    else if (m.sender_id !== u.id) { unread.set(m.conversation_id, (unread.get(m.conversation_id) || 0) + 1); badge(); if (window.innerWidth >= 1280 && document.documentElement.classList.contains("rail-collapsed")) nudgeTab(); }
    clearTimeout(timer); timer = setTimeout(load, 400);
  });
  Live.connect(Auth.token);
  loadPeople().then(load);
  window.openRailChat = (id) => (id ? open(id) : setRail(true));
  // Сургалтын хуудасны товчнууд: нэвтэрсэн хүнд яриа энэ самбарт нээгдэнэ.
  window.railStartDirect = async (username) => { const d = await api(`/api/teachers/${username}/chat`, { method: "POST" }); await load(); open(d.conversation.id); };
  window.railStartGroup = async (courseId) => { const d = await api(`/api/courses/${courseId}/chat`, { method: "POST" }); Live.reconnect(); await load(); open(d.conversation.id); };
}

/* ---------- навигаци, мэдэгдлийн хонх ---------- */
function authNav() {
  const u = Auth.user;
  $$("[data-auth-show]").forEach((el) => (el.hidden = !u || (el.dataset.role && el.dataset.role !== u.role)));
  document.documentElement.classList.toggle("authed", !!u);
  $$("[data-auth-hide]").forEach((el) => (el.hidden = !!u));
  $$("[data-logout]").forEach((b) => b.addEventListener("click", () => { Auth.clear(); location.href = "/"; }));
  if (u && Auth.token) bell();
}

function bell() {
  const links = $(".nav-links");
  if (!links || $("#bell")) return;
  links.insertAdjacentHTML("afterbegin", `
    <div class="bell-wrap"><button class="icon-btn bell" id="bell" aria-label="Мэдэгдэл"><svg viewBox="0 0 24 24" width="21" height="21" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6 9a6 6 0 0 1 12 0c0 6 2.5 7.5 2.5 7.5h-17S6 15 6 9"/><path d="M10 20a2.2 2.2 0 0 0 4 0"/></svg><span class="bell-badge" hidden></span></button>
    <div class="bell-panel glass" id="bellPanel" hidden><div class="bell-head"><strong>Мэдэгдэл</strong><button class="btn btn-ghost btn-sm" id="bellPerm" hidden>🔔 Браузерт мэдэгдэх</button></div><ol class="bell-list" id="bellList"><li class="muted small">Ачаалж байна…</li></ol></div></div>`);
  const badge = $(".bell-badge"), panel = $("#bellPanel"), list = $("#bellList");
  let unread = 0;
  const setBadge = (n) => { unread = n; badge.hidden = !n; badge.textContent = n > 99 ? "99+" : n; };
  const item = (n) => {
    const ub = n.type === "blocked" && /^unblock:([^:]+):(.+)$/.exec(n.link || "");
    if (ub) return `<li class="bell-item bell-blocked ${n.read ? "" : "unread"}"><div><strong>${esc(n.title)}</strong>${n.body ? `<span>${esc(n.body)}</span>` : ""}<time>${fmtDate(n.created_at)}</time>
      <button type="button" class="btn btn-sm bell-unblock" data-unblock-user="${esc(ub[1])}" data-unblock-lesson="${esc(ub[2])}">⛔ Хаагдсан — 🔓 Дахин нээх</button></div></li>`;
    return `<li class="bell-item ${n.read ? "" : "unread"}"><a href="${esc(n.Link || n.link || "#")}"><strong>${esc(n.title)}</strong>${n.body ? `<span>${esc(n.body)}</span>` : ""}<time>${fmtDate(n.created_at)}</time></a></li>`;
  };
  list.addEventListener("click", async (e) => {
    const b = e.target.closest("[data-unblock-user]"); if (!b) return;
    e.preventDefault(); e.stopPropagation(); b.disabled = true;
    try {
      const r = await api(`/api/me/students/${b.dataset.unblockUser}/unblock`, { method: "POST", body: { lesson_id: b.dataset.unblockLesson } });
      b.className = "btn btn-sm bell-unblock done"; b.textContent = r.exam ? "✓ Шалгалт дахин нээгдлээ (+1 оролдлого)" : r.unblocked ? "✓ Дахин нээгдлээ — суралцагчид мэдэгдлээ" : "✓ Аль хэдийн нээлттэй";
    } catch (x) { toast(x.message, true); b.disabled = false; }
  });
  const load = async () => {
    try {
      const d = await api("/api/me/notifications");
      list.innerHTML = d.items.length ? d.items.map(item).join("") : `<li class="muted small">Одоогоор мэдэгдэл алга</li>`;
      setBadge(d.unread);
    } catch {}
  };
  load();
  const perm = $("#bellPerm");
  if ("Notification" in window && Notification.permission === "default") perm.hidden = false;
  perm.addEventListener("click", async () => { await Notification.requestPermission(); perm.hidden = true; });
  $("#bell").addEventListener("click", async (e) => {
    e.stopPropagation();
    panel.hidden = !panel.hidden;
    if (!panel.hidden && unread) { setBadge(0); api("/api/me/notifications/read", { method: "POST" }).catch(() => {}); }
  });
  document.addEventListener("click", (e) => { if (!e.target.closest(".bell-wrap")) panel.hidden = true; });
  Live.on((d) => {
    if (d.type !== "notification") return;
    const n = d.notification;
    list.querySelector(".muted")?.remove();
    list.insertAdjacentHTML("afterbegin", item(n));
    setBadge(unread + 1);
    $("#bell").animate([{ transform: "rotate(0)" }, { transform: "rotate(18deg)" }, { transform: "rotate(-14deg)" }, { transform: "rotate(0)" }], { duration: 600 });
    toast(n.title);
    if (document.hidden && "Notification" in window && Notification.permission === "granted") {
      const bn = new Notification(n.title, { body: n.body || "", tag: n.id });
      bn.onclick = () => { focus(); if (n.link && !n.link.startsWith("unblock:")) location.href = n.link; else panel.hidden = false; };
    }
  });
  Live.connect(Auth.token);
}

/* ---------- медиа: видео, аудио, PDF (3D ном), зураг ---------- */
function extOf(url) {
  try { const p = new URL(url, location.href).pathname.toLowerCase(); const i = p.lastIndexOf("."); return i < 0 ? "" : p.slice(i); } catch { return ""; }
}
// Гадаад видеоны кодолсон ID ("yt:"/"vm:" + урвуу base64url) — жинхэнэ холбоосыг суралцагчид харуулахгүй.
const unob = (s) => { try { return atob(s.replace(/-/g, "+").replace(/_/g, "/")).split("").reverse().join(""); } catch { return ""; } };
// YouTube: controls=0 + тунгалаг хамгаалах давхарга + өөрийн удирдлага → лого, гарчиг, "YouTube-д үзэх", хуваалцах дарагдахгүй.
const ytShieldHTML = (id) => `<div class="yt-wrap" oncontextmenu="return false"><iframe data-yt="${esc(id)}" src="https://www.youtube-nocookie.com/embed/${esc(id)}?rel=0&controls=0&disablekb=1&modestbranding=1&iv_load_policy=3&fs=0&playsinline=1&cc_load_policy=0&enablejsapi=1&origin=${encodeURIComponent(location.origin)}" allow="autoplay; encrypted-media" referrerpolicy="strict-origin" tabindex="-1"></iframe>
  <div class="yt-cover" style="background-image:url('https://i.ytimg.com/vi/${esc(id)}/hqdefault.jpg')"></div>
  <div class="yt-shield" data-yt-toggle title="Тоглуулах / зогсоох"><span class="yt-big">▶</span></div>
  <div class="vg-bar yt-bar"><button type="button" class="vg-play" data-yt-toggle aria-label="Тоглуулах">▶</button><span class="vg-time">0:00 / 0:00</span><span class="vg-prog"><b class="vg-seen"></b><i></i></span><button type="button" class="vg-mute" aria-label="Дуу">🔊</button><button type="button" class="vg-full" aria-label="Бүтэн дэлгэц">⛶</button></div></div>`;
function mediaHTML(url, title) {
  if (!url) return "";
  if (url.startsWith("yt:")) return ytShieldHTML(unob(url.slice(3)));
  if (url.startsWith("vm:")) { const id = unob(url.slice(3)); return `<div class="yt-wrap vm" oncontextmenu="return false"><iframe src="https://player.vimeo.com/video/${esc(id)}?title=0&byline=0&portrait=0&dnt=1&pip=0" allow="autoplay; fullscreen" allowfullscreen referrerpolicy="strict-origin"></iframe><div class="vm-shield-top"></div><div class="vm-shield-logo"></div></div>`; }
  const yt = url.match(/(?:youtube\.com\/(?:watch\?v=|embed\/|shorts\/)|youtu\.be\/)([\w-]{11})/);
  if (yt) return `<iframe data-yt="${yt[1]}" src="https://www.youtube-nocookie.com/embed/${yt[1]}?rel=0&enablejsapi=1&origin=${encodeURIComponent(location.origin)}" allow="accelerometer; autoplay; encrypted-media; picture-in-picture" allowfullscreen></iframe>`;
  const vm = url.match(/vimeo\.com\/(\d+)/);
  if (vm) return `<iframe src="https://player.vimeo.com/video/${vm[1]}" allow="autoplay; fullscreen; picture-in-picture" allowfullscreen></iframe>`;
  const ext = extOf(url);
  if ([".webm", ".mp4", ".mov", ".m4v"].includes(ext)) return `<video src="${esc(url)}" controls playsinline preload="metadata" controlslist="nodownload" disablepictureinpicture oncontextmenu="return false"></video>`;
  if ([".mp3", ".m4a", ".wav"].includes(ext)) return `<audio src="${esc(url)}" controls controlslist="nodownload" oncontextmenu="return false" style="width:100%"></audio>`;
  if ([".webp", ".jpg", ".jpeg", ".png", ".gif"].includes(ext)) return `<img src="${esc(url)}" alt="">`;
  if (docKind(url)) return docViewHTML(url, title); // PDF, PowerPoint, Word, Excel — хичээл дотор шууд, хамгаалалттай
  return `<div style="padding:24px;text-align:center"><a class="btn btn-gold" href="${esc(url)}" target="_blank" rel="noopener">⬇ Файл татах</a></div>`;
}

/* ---------- Хичээлийн дэлгэрэнгүй агуулга (блокууд) ---------- */
// Хэлбэржүүлсэн текст: аюулгүй энгийн тэмдэглэгээг HTML болгоно (эхлээд escape хийдэг тул XSS боломжгүй).
function richHTML(src) {
  const inline = (t) => {
    const links = [];
    let h = esc(t).replace(/\[([^\]\n]{1,300})\]\((https?:\/\/[^\s)]{1,800})\)/g, (_, a, u) => { links.push([a, u]); return "\u0000" + (links.length - 1) + "\u0000"; });
    h = h.replace(/\*\*(.+?)\*\*/g, "<b>$1</b>").replace(/__(.+?)__/g, "<u>$1</u>").replace(/==(.+?)==/g, "<mark>$1</mark>").replace(/(^|[^*])\*([^*\n]+?)\*/g, "$1<i>$2</i>");
    return h.replace(/\u0000(\d+)\u0000/g, (_, i) => `<a href="${links[i][1]}" target="_blank" rel="noopener nofollow">${links[i][0]}</a>`);
  };
  const out = []; let para = [], list = null, quote = [];
  const flushPara = () => { if (para.length) out.push(`<p>${para.map(inline).join("<br>")}</p>`); para = []; };
  const flushList = () => { if (list) out.push(`<${list.tag}>${list.items.map((x) => `<li>${inline(x)}</li>`).join("")}</${list.tag}>`); list = null; };
  const flushQuote = () => { if (quote.length) out.push(`<blockquote>${quote.map(inline).join("<br>")}</blockquote>`); quote = []; };
  const flush = () => { flushPara(); flushList(); flushQuote(); };
  for (const line of String(src || "").split("\n")) {
    let m;
    if ((m = line.match(/^\s*[-•]\s+(.*)$/)) || (m = line.match(/^\s*\d+[.)]\s+(.*)$/))) {
      const tag = /^\s*\d/.test(line) ? "ol" : "ul";
      flushPara(); flushQuote(); if (list && list.tag !== tag) flushList();
      (list ||= { tag, items: [] }).items.push(m[1]);
    } else if ((m = line.match(/^>\s?(.*)$/))) { flushPara(); flushList(); quote.push(m[1]); }
    else if (!line.trim()) flush();
    else { flushList(); flushQuote(); para.push(line); }
  }
  flush();
  return out.join("");
}
const fmtBytes = (b) => !b ? "" : b >= 1 << 20 ? (b / (1 << 20)).toFixed(1) + " MB" : Math.max(1, Math.round(b / 1024)) + " KB";
// Embed-ийн HTML-ийг бүтэн баримт болгоно (хоосон зай, хэмжээ тохируулна).
const embedDoc = (html) => `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><style>html,body{margin:0;padding:0;font-family:system-ui,sans-serif}iframe,video,img{max-width:100%}body>iframe:only-child{width:100%;height:100vh;border:0;display:block}</style></head><body>${html}</body></html>`;
function blockHTML(b) {
  const cap = b.text && b.type !== "text" && b.type !== "heading" ? `<figcaption>${esc(b.text)}</figcaption>` : "";
  switch (b.type) {
    case "heading": return `<h3 class="rb-h">${esc(b.text)}</h3>`;
    // HTML embed: sandbox (allow-same-origin-гүй) — скрипт ажиллана, гэхдээ сайтын нэвтрэлт, өгөгдөлд хүрэхгүй.
    case "embed": return `<figure class="rb-embed"><iframe sandbox="allow-scripts allow-popups allow-forms allow-presentation allow-popups-to-escape-sandbox" allow="fullscreen; autoplay; encrypted-media; picture-in-picture" allowfullscreen loading="lazy" referrerpolicy="no-referrer" style="height:${Math.max(80, Math.min(2000, +b.height || 420))}px" srcdoc="${esc(embedDoc(b.text))}" title="Embed"></iframe></figure>`;
    case "text": return `<div class="rb-text">${richHTML(b.text)}</div>`;
    case "image": return `<figure class="rb-fig"><img src="${esc(b.url)}" alt="${esc(b.text || "")}" loading="lazy">${cap}</figure>`;
    case "audio": return `<figure class="rb-audio"><span class="rb-audio-ico">🎧</span><div><b>${esc(b.name || "Дуу бичлэг")}</b><audio src="${esc(b.url)}" controls preload="metadata" controlslist="nodownload" oncontextmenu="return false"></audio></div>${cap}</figure>`;
    case "video": return `<figure class="rb-video" data-bid="${esc(b.id)}" ${b.parts?.length > 1 ? `data-parts="${esc(JSON.stringify(b.parts))}"` : ""}><div class="player">${mediaHTML(b.url, b.name)}</div>${b.parts?.length > 1 ? `<div class="vparts" aria-label="Видеоны хэсгүүд">${b.parts.map((_, i) => `<button type="button" data-vpart="${i}" class="${i ? "" : "on"}">${i + 1}-р хэсэг</button>`).join("")}<span class="muted small">6 минутын ${b.parts.length} хэсэг · дараалан тоглоно</span></div>` : ""}${cap}</figure>`;
    case "file": {
      const meta = esc([extOf(b.url).slice(1).toUpperCase(), fmtBytes(b.size)].filter(Boolean).join(" · ")) + (b.text ? " · " + esc(b.text) : "");
      const kind = docKind(b.url);
      // PDF, PowerPoint, Word, Excel: хичээл дотор шууд (попапгүй), татах боломжгүй — багш зөвшөөрсөн бол л татна.
      if (kind && kind !== "legacy") return `<figure class="rb-doc" ${b.download ? "data-dl" : "data-nodl"}>${docViewHTML(b.url, b.name || DOC_LABEL[kind])}${cap}</figure>`;
      return b.download ? `<a class="rb-file" href="${esc(b.url)}" target="_blank" rel="noopener" download><span class="rb-file-ico">📄</span><span><b>${esc(b.name || "Файл")}</b><small>${meta}</small></span><span class="btn btn-sm btn-glass">⬇ Татах</span></a>`
        : `<div class="rb-file"><span class="rb-file-ico">🔒</span><span><b>${esc(b.name || "Файл")}</b><small>${meta} · ${kind === "legacy" ? "Хуучин форматыг шууд үзүүлэх боломжгүй — багш PDF, .pptx, .docx болгож оруулна" : "Зөвхөн хичээл дотор үзнэ"}</small></span></div>`;
    }
    case "quiz": return quizFormHTML(b, false);
  }
  return "";
}
const blocksHTML = (bs) => `<div class="rb">${(bs || []).map(blockHTML).join("")}</div>`;
// Асуулт: 5 төрөл. exam=true бол "Шалгах" товчгүй (шалгалтыг нэг дор илгээнэ).
const QKIND_HINT = { single: "Нэг зөв хариулттай", multi: "Хэд хэдэн зөв хариулттай", text: "Хариултаа бичнэ", match: "Зүүн талыг баруун талтай нь харгалзуулна", image: "Зураг дээр дарж хариулна" };
function quizFormHTML(b, exam, n) {
  const q = b.quiz || {}, kind = q.kind || (q.multi ? "multi" : "single");
  let body = "";
  if (kind === "single" || kind === "multi") body = `<div class="rq-opts">${(q.options || []).map((o, i) => `<label class="rq-opt"><input type="${kind === "multi" ? "checkbox" : "radio"}" name="a" value="${i}"><span>${esc(o)}</span></label>`).join("")}</div>`;
  else if (kind === "text") body = `<input class="rq-text" name="text" maxlength="200" autocomplete="off" placeholder="Хариултаа бичнэ үү">`;
  else if (kind === "match") body = `<div class="rq-match">${(q.left || []).map((l, i) => `<div class="rq-pair"><span>${esc(l)}</span><span class="rq-arrow">⇄</span><select name="m${i}" aria-label="${esc(l)}-ийн хос"><option value="">— сонгох —</option>${(q.right || []).map((r, j) => `<option value="${j}">${esc(r)}</option>`).join("")}</select></div>`).join("")}</div>`;
  else if (kind === "image") body = `<div class="rq-spotwrap" data-pick><img src="${esc(q.image)}" alt="" draggable="false"><i class="rq-pt" hidden></i><i class="rq-spot" hidden></i></div>`;
  return `<form class="rb-quiz ${exam ? "exam-q" : ""}" data-quiz="${esc(b.id)}" data-kind="${kind}">
    <div class="rq-head"><span class="rq-badge">${exam ? `${n}.` : "❓ Асуулт"}</span><small>${QKIND_HINT[kind]}</small>${exam && q.points > 1 ? `<small class="rq-points">${q.points} оноо</small>` : ""}<span class="rq-prev" hidden></span></div>
    <p class="rq-q">${esc(q.question)}</p>${q.image && kind !== "image" ? `<img class="rq-img" src="${esc(q.image)}" alt="" draggable="false">` : ""}
    ${body}
    ${exam ? "" : `<div class="rq-foot"><button class="btn btn-gold btn-sm">Шалгах</button><span class="rq-res" role="status"></span></div>`}
    <p class="rq-explain" hidden></p></form>`;
}
// quizAnswer: маягтаас серверт илгээх хариулт; хоосон бол null.
function quizAnswer(f) {
  const k = f.dataset.kind;
  if (k === "single" || k === "multi") { const a = $$("input:checked", f).map((i) => +i.value); return a.length ? { answer: a } : null; }
  if (k === "text") { const t = $(".rq-text", f).value.trim(); return t ? { text: t } : null; }
  if (k === "match") { const m = $$("select", f).map((x) => (x.value === "" ? -1 : +x.value)); return m.some((x) => x >= 0) ? { match: m } : null; }
  if (k === "image") return f.dataset.px ? { point: [+f.dataset.px, +f.dataset.py] } : null;
  return null;
}
function quizReveal(f, r, correct) {
  const k = f.dataset.kind;
  f.classList.add("answered", correct ? "is-ok" : "is-bad");
  if ((k === "single" || k === "multi") && r.answer) $$(".rq-opt", f).forEach((o, i) => { const right = r.answer.includes(i), picked = $("input", o).checked; o.classList.toggle("ok", right); o.classList.toggle("bad", picked && !right); o.classList.toggle("miss", right && !picked); });
  if (k === "text" && r.answers) { $(".rq-text", f).classList.add(correct ? "ok" : "bad"); if (!correct) $(".rq-text", f).insertAdjacentHTML("afterend", `<p class="rq-correct">Зөв хариулт: <b>${r.answers.map(esc).join(" / ")}</b></p>`); }
  if (k === "match" && r.match) $$(".rq-pair", f).forEach((p, i) => { const sel = $("select", p), ok = +sel.value === r.match[i]; p.classList.add(ok ? "ok" : "bad"); if (!ok) p.insertAdjacentHTML("beforeend", `<small class="rq-correct">→ ${esc(sel.options[r.match[i] + 1]?.textContent || "")}</small>`); });
  if (k === "image" && r.spot) { const sp = $(".rq-spot", f); Object.assign(sp.style, { left: r.spot.x + "%", top: r.spot.y + "%", width: r.spot.r * 2 + "%", height: r.spot.r * 2 + "%" }); sp.hidden = false; }
  if (r.explain) { const ex = $(".rq-explain", f); ex.hidden = false; ex.textContent = "💡 " + r.explain; }
}
function quizReset(f) {
  f.classList.remove("answered", "is-ok", "is-bad");
  $$(".ok,.bad,.miss", f).forEach((o) => o.classList.remove("ok", "bad", "miss"));
  $$(".rq-correct", f).forEach((x) => x.remove());
  $(".rq-res", f) && ($(".rq-res", f).textContent = ""); $(".rq-explain", f).hidden = true; const sp = $(".rq-spot", f); if (sp) sp.hidden = true;
  const b = $(".rq-foot button", f); if (b) b.textContent = "Шалгах";
}
// Зурган дээр дарж хариулах (хичээл ба шалгалтад).
function bindQuizPicks(box) {
  if (box._picks) return; box._picks = true;
  box.addEventListener("click", (e) => {
    const w = e.target.closest(".rq-spotwrap[data-pick]"); if (!w) return;
    const f = w.closest(".rb-quiz"); if (f.classList.contains("answered") || f.classList.contains("locked")) return;
    const r = w.getBoundingClientRect(), x = (e.clientX - r.left) / r.width * 100, y = (e.clientY - r.top) / r.height * 100;
    f.dataset.px = x.toFixed(1); f.dataset.py = y.toFixed(1);
    const pt = $(".rq-pt", w); pt.hidden = false; pt.style.left = x + "%"; pt.style.top = y + "%";
    f.dispatchEvent(new Event("change", { bubbles: true }));
  });
}
// Хичээл доторх асуултууд: сервер шалгаж зөв хариулт, тайлбарыг буцаана. results — өмнөх хариултууд.
function mountQuizzes(box, base, results = {}) {
  bindQuizPicks(box);
  // Асуулт хэзээ нүдэнд харагдсаныг тэмдэглэнэ — хариулах хугацааг (ms) үүнээс тоолж, хэт хурдан
  // (таамагласан) хариултыг багшийн самбарт тусад нь харуулна.
  const io = "IntersectionObserver" in window ? new IntersectionObserver((es) => {
    for (const e of es) if (e.isIntersecting) { e.target.dataset.shown ||= Date.now(); io.unobserve(e.target); }
  }, { threshold: 0.4 }) : null;
  $$(".rb-quiz", box).forEach((f) => {
    const prev = results[f.dataset.quiz], chip = $(".rq-prev", f);
    if (prev !== undefined) { chip.hidden = false; chip.textContent = prev ? "✓ Өмнө нь зөв хариулсан" : "Өмнө нь буруу хариулсан"; chip.className = "rq-prev " + (prev ? "ok" : "bad"); }
    f.addEventListener("change", () => { if (f.classList.contains("answered")) quizReset(f); });
    if (io) io.observe(f); else f.dataset.shown = Date.now();
  });
  box.onsubmit = async (e) => { // onsubmit: хичээлийг дахин нээхэд давхардахгүй
    const f = e.target.closest(".rb-quiz"); if (!f) return;
    e.preventDefault();
    if (f.classList.contains("answered")) { quizReset(f); f.reset(); delete f.dataset.px; $$(".rq-pt", f).forEach((p) => (p.hidden = true)); return; }
    const ans = quizAnswer(f);
    if (!ans) { toast("Хариултаа сонгоно уу"); return; }
    ans.ms = f.dataset.shown ? Date.now() - +f.dataset.shown : 0;
    const btn = $(".rq-foot button", f); btn.disabled = true;
    try {
      const r = await api(`${base}/quiz/${encodeURIComponent(f.dataset.quiz)}`, { method: "POST", body: ans });
      quizReveal(f, r, r.correct);
      // Хичээлийн асуулгын явц: бүгдэд нь зөв хариулбал дараагийн хичээл нээгдэнэ.
      const prog = r.quiz_total ? ` · Асуулга ${r.quiz_correct}/${r.quiz_total}` : "";
      $(".rq-res", f).textContent = (r.correct ? "✓ Зөв! Сайн байна" : "✗ Буруу байна — ногоон нь зөв хариулт") + prog;
      btn.textContent = "Дахин оролдох";
      if (r.correct) celebrate?.();
      if (r.mastered && r.quiz_total) {
        if (!box.dataset.masteredShown) { box.dataset.masteredShown = "1"; toast("🎉 Бүх асуултад зөв хариуллаа — дараагийн хичээл нээгдлээ"); }
        document.dispatchEvent(new CustomEvent("sg:quiz-mastered"));
      }
    } catch (err) { toast(err.message, true); } finally { btn.disabled = false; }
  };
}

/* ---------- Хэлэлцүүлэг: лайк, сэтгэгдэл, асуулт, хариу (Facebook маягийн) ---------- */
function timeAgo(t) {
  const m = Math.round((Date.now() - new Date(t)) / 60000);
  if (m < 1) return "саяхан";
  if (m < 60) return m + " мин";
  if (m < 1440) return Math.floor(m / 60) + " ц";
  if (m < 10080) return Math.floor(m / 1440) + " өдөр";
  return fmtDate(t);
}
async function discussionPanel(box, { courseId, lessonId }) {
  const base = `/api/courses/${courseId}/lessons/${lessonId}`;
  box.hidden = false;
  if (!Auth.token) { box.innerHTML = `<div class="dc-head"><b>💬 Хэлэлцүүлэг</b></div><p class="muted small">Сэтгэгдэл бичих, лайк дарахын тулд <a href="/login?next=${encodeURIComponent(location.pathname)}">нэвтэрнэ үү</a>.</p>`; return; }
  const me = Auth.user || {};
  let d, openReply = null;
  const cm = (c, reply = false) => `<article class="dc-item ${reply ? "dc-reply" : ""}" data-cid="${esc(c.id)}">
      ${avatarHTML({ display_name: c.user_name, avatar_url: c.avatar_url, username: c.user_name }, "avatar-sm")}
      <div class="dc-main"><div class="dc-bubble"><b>${esc(c.user_name)}${c.teacher ? ` <span class="dc-badge">Багш</span>` : ""}</b><p>${linkify(c.body)}</p></div>
        <div class="dc-acts"><button type="button" class="dc-like ${c.liked ? "on" : ""}" data-like-c="${esc(c.id)}">${c.liked ? "👍 Таалагдсан" : "Таалагдлаа"}${c.likes ? ` · ${c.likes}` : ""}</button>
          ${reply ? "" : `<button type="button" data-reply="${esc(c.id)}">Хариулах</button>`}<span class="muted">${timeAgo(c.created_at)}</span>
          ${c.can_delete ? `<button type="button" class="dc-del" data-del-c="${esc(c.id)}">Устгах</button>` : ""}</div>
        ${(c.replies || []).map((r) => cm(r, true)).join("")}
        ${openReply === c.id ? composer(c.id) : ""}</div></article>`;
  const composer = (parent) => `<form class="dc-form ${parent ? "dc-form-reply" : ""}" data-parent="${esc(parent || "")}">${avatarHTML(me, "avatar-sm")}
      <div class="dc-input"><textarea name="body" rows="1" maxlength="2000" placeholder="${parent ? "Хариу бичих…" : "Асуулт, сэтгэгдлээ бичнэ үү…"}" required></textarea><button class="btn btn-gold btn-sm" aria-label="Илгээх">➤</button></div></form>`;
  const render = () => {
    const n = d.count;
    box.innerHTML = `<div class="dc-head"><button type="button" class="dc-like dc-like-lesson ${d.liked ? "on" : ""}" data-like-lesson>👍 ${d.liked ? "Таалагдсан" : "Таалагдлаа"}${d.likes ? ` · ${d.likes}` : ""}</button>
        <span class="muted small">💬 ${n ? n + " сэтгэгдэл" : "Сэтгэгдэл алга — эхнийхийг нь та бичээрэй"}</span></div>
      ${composer("")}
      <div class="dc-list">${d.comments.map((c) => cm(c)).join("")}</div>`;
    $$(".dc-form textarea", box).forEach((t) => t.addEventListener("input", () => { t.style.height = "auto"; t.style.height = Math.min(160, t.scrollHeight) + "px"; }));
    $(".dc-form-reply textarea", box)?.focus();
  };
  const load = async () => { try { d = await api(`${base}/discussion`); render(); } catch (e) { box.innerHTML = e.status === 409 ? "" : `<p class="form-error">${esc(e.message)}</p>`; if (e.status === 409) box.hidden = true; } };
  box.onclick = async (e) => {
    const t = e.target;
    if (t.closest("[data-like-lesson]")) { try { const r = await api(`${base}/like`, { method: "POST", body: { target: "lesson" } }); d.liked = r.liked; d.likes = r.likes; render(); } catch (x) { toast(x.message, true); } return; }
    const lc = t.closest("[data-like-c]");
    if (lc) { try { const r = await api(`${base}/like`, { method: "POST", body: { target: "comment", id: lc.dataset.likeC } }); const upd = (c) => { if (c.id === lc.dataset.likeC) { c.liked = r.liked; c.likes = r.likes; } (c.replies || []).forEach(upd); }; d.comments.forEach(upd); render(); } catch (x) { toast(x.message, true); } return; }
    const rp = t.closest("[data-reply]");
    if (rp) { openReply = openReply === rp.dataset.reply ? null : rp.dataset.reply; render(); return; }
    const del = t.closest("[data-del-c]");
    if (del && confirm("Сэтгэгдлийг устгах уу?")) { try { await api(`${base}/discussion/${del.dataset.delC}`, { method: "DELETE" }); await load(); } catch (x) { toast(x.message, true); } }
  };
  box.onsubmit = async (e) => {
    const f = e.target.closest(".dc-form"); if (!f) return; e.preventDefault();
    const body = f.body.value.trim(); if (!body) return;
    const b = $("button", f); b.disabled = true;
    try { await api(`${base}/discussion`, { method: "POST", body: { body, parent_id: f.dataset.parent || "" } }); openReply = null; await load(); }
    catch (x) { toast(x.message, true); } finally { b.disabled = false; }
  };
  box.addEventListener("keydown", (e) => { if (e.key === "Enter" && !e.shiftKey && e.target.matches(".dc-form textarea")) { e.preventDefault(); e.target.closest("form").requestSubmit(); } });
  await load();
}

/* ---------- Цол олгох ёслол: өнгөлөг салют + баяр хүргэх карт (систем өөрөө) ---------- */
function fireworks(canvas, ms = 5000) {
  const ctx = canvas.getContext("2d"), dpr = Math.min(2, devicePixelRatio || 1);
  const fit = () => { canvas.width = innerWidth * dpr; canvas.height = innerHeight * dpr; };
  fit(); addEventListener("resize", fit);
  const colors = ["#eaa02e", "#f6c56f", "#ffd79a", "#ffffff", "#9fb4ef", "#c9d6ff"];
  const parts = [];
  const burst = () => {
    const x = (0.15 + Math.random() * 0.7) * canvas.width, y = (0.15 + Math.random() * 0.45) * canvas.height, col = colors[Math.floor(Math.random() * colors.length)], n = 70 + Math.random() * 50;
    for (let i = 0; i < n; i++) { const a = Math.random() * Math.PI * 2, v = (2 + Math.random() * 6) * dpr; parts.push({ x, y, vx: Math.cos(a) * v, vy: Math.sin(a) * v, life: 1, decay: 0.008 + Math.random() * 0.012, col, r: (1.2 + Math.random() * 1.8) * dpr }); }
  };
  const start = performance.now(); let last = 0, raf;
  const tick = (t) => {
    if (t - last > 420 && t - start < ms - 1200) { last = t; burst(); if (Math.random() < 0.5) burst(); }
    ctx.globalCompositeOperation = "destination-out"; ctx.fillStyle = "rgba(0,0,0,.18)"; ctx.fillRect(0, 0, canvas.width, canvas.height);
    ctx.globalCompositeOperation = "lighter";
    for (let i = parts.length - 1; i >= 0; i--) {
      const p = parts[i]; p.x += p.vx; p.y += p.vy; p.vy += 0.05 * dpr; p.vx *= 0.985; p.vy *= 0.985; p.life -= p.decay;
      if (p.life <= 0) { parts.splice(i, 1); continue; }
      ctx.globalAlpha = Math.max(0, p.life); ctx.fillStyle = p.col; ctx.beginPath(); ctx.arc(p.x, p.y, p.r * (0.6 + p.life * 0.6), 0, Math.PI * 2); ctx.fill();
    }
    ctx.globalAlpha = 1;
    if (t - start < ms || parts.length) raf = requestAnimationFrame(tick); else removeEventListener("resize", fit);
  };
  raf = requestAnimationFrame(tick);
  return () => { cancelAnimationFrame(raf); removeEventListener("resize", fit); };
}
function rankSalute(rank) {
  if (!rank?.name || $(".salute")) return;
  const reduce = matchMedia("(prefers-reduced-motion: reduce)").matches;
  const el = document.createElement("div"); el.className = "salute"; el.setAttribute("role", "dialog"); el.setAttribute("aria-label", "Шинэ цол");
  el.innerHTML = `<canvas class="salute-sky" aria-hidden="true"></canvas>
    <div class="salute-card"><span class="eyebrow">Систем баяр хүргэж байна</span>
      <div class="salute-sign rank-shine" data-level="${rank.level}">${esc(rank.insignia || "★")}</div>
      <h2>🎖 Шинэ цол: <em class="rank-name">${esc(rank.name)}</em></h2>
      <p>Идэвхтэй, шударга суралцсан тань үнэлэгдэж <b>${rank.points}</b> оноонд хүрлээ.${rank.next ? ` Дараагийн «${esc(rank.next_name)}» цол ${rank.next} оноонд.` : " Энэ бол дээд цол!"}</p>
      <button class="btn btn-gold btn-lg">Баярлалаа 🎉</button></div>`;
  document.body.append(el);
  const stop = reduce ? () => {} : fireworks($(".salute-sky", el), 6000);
  celebrate?.();
  const close = () => { stop(); el.classList.add("out"); setTimeout(() => el.remove(), 400); };
  $("button", el).onclick = close; $("button", el).focus();
  el.addEventListener("click", (e) => { if (e.target === el) close(); });
  setTimeout(close, 12000);
}

// lockedControls: анх үзэж байхад хөтчийн ердийн удирдлагыг нууж, өөрийн удирдлага тавина —
// явцын мөр дарагдахгүй (гүйлгэх, үсрэх боломжгүй), зөвхөн тоглуулах/зогсоох, дуу, бүтэн дэлгэц.
// Бүрэн үзсэний дараа release() → ердийн удирдлага (гүйлгэж болно).
// maxFn() — энэ видеон дээр үзсэн хамгийн хол цэг (сек): түүнээс өмнө чөлөөтэй гүйлгэнэ, цааш нь үгүй.
function lockedControls(v, isFree, note, maxFn = () => 0) {
  if (isFree()) return () => {};
  const wrap = v.parentElement; if (!wrap) return () => {};
  v.controls = false; v.removeAttribute("controls");
  wrap.classList.add("vg-locked");
  const bar = document.createElement("div");
  bar.className = "vg-bar";
  bar.innerHTML = `<button type="button" class="vg-play" aria-label="Тоглуулах">▶</button><span class="vg-time">0:00 / 0:00</span>
    <span class="vg-prog" title="Үзсэн хэсэг рүүгээ буцаж гүйлгэнэ — үзээгүй хэсэг рүү үгүй"><b class="vg-seen"></b><i></i></span><span class="vg-lock" aria-hidden="true">🔒</span>
    <button type="button" class="vg-mute" aria-label="Дуу">🔊</button><button type="button" class="vg-full" aria-label="Бүтэн дэлгэц">⛶</button>`;
  wrap.append(bar);
  const fmt = (t) => `${Math.floor(t / 60)}:${String(Math.floor(t % 60)).padStart(2, "0")}`;
  const play = $(".vg-play", bar), time = $(".vg-time", bar), prog = $(".vg-prog i", bar), mute = $(".vg-mute", bar), full = $(".vg-full", bar);
  const seen = $(".vg-seen", bar);
  const sync = () => { play.textContent = v.paused ? "▶" : "❚❚"; time.textContent = `${fmt(v.currentTime)} / ${fmt(v.duration || 0)}`; prog.style.width = v.duration ? v.currentTime / v.duration * 100 + "%" : "0"; seen.style.width = v.duration ? Math.min(1, Math.max(maxFn(), v.currentTime) / v.duration) * 100 + "%" : "0"; mute.textContent = v.muted ? "🔇" : "🔊"; };
  // Үзсэн хүртэлх хэсэгт л үсэрнэ; цааш дарвал үзсэн хамгийн хол цэг дээр зогсооно.
  const seekTo = (t) => { const lim = Math.max(maxFn(), 0); if (t > lim + 0.5) { v.currentTime = lim; note?.(); } else v.currentTime = Math.max(0, t); sync(); };
  const toggle = () => { if (v.paused) v.play().catch(() => {}); else v.pause(); };
  play.onclick = toggle;
  v.addEventListener("click", toggle);
  mute.onclick = () => { v.muted = !v.muted; sync(); };
  full.onclick = () => { if (document.fullscreenElement) document.exitFullscreen?.(); else wrap.requestFullscreen?.(); };
  $(".vg-prog", bar).onclick = (e) => { if (!v.duration) return; const r = e.currentTarget.getBoundingClientRect(); seekTo(v.duration * (e.clientX - r.left) / r.width); };
  ["timeupdate", "play", "pause", "loadedmetadata", "volumechange", "durationchange"].forEach((n) => v.addEventListener(n, sync));
  const onKey = (e) => { // сум, PageUp/Down, тоонуудаар үсрэхийг хаана
    if (!wrap.contains(document.activeElement) && document.fullscreenElement !== wrap) return;
    if (e.key === "ArrowLeft" || e.key === "j") { e.preventDefault(); seekTo(v.currentTime - 5); return; }
    if (e.key === "ArrowRight" || e.key === "l") { e.preventDefault(); seekTo(v.currentTime + 5); return; }
    if (e.key === "Home") { e.preventDefault(); seekTo(0); return; }
    if (["End", "PageUp", "PageDown"].includes(e.key) || /^[0-9]$/.test(e.key)) { e.preventDefault(); note?.(); }
    if (e.key === " " || e.key === "k") { e.preventDefault(); toggle(); }
  };
  document.addEventListener("keydown", onKey);
  sync();
  let released = false;
  return () => {
    if (released) return; released = true;
    document.removeEventListener("keydown", onKey);
    bar.remove(); wrap.classList.remove("vg-locked");
    v.controls = true; v.setAttribute("controls", "");
    v.removeEventListener("click", toggle);
  };
}

/* ---------- Унших туслах: курсор байгаа мөрийг тодруулна (хичээлийн текст дээр, hover үед) ---------- */
function readingRuler(box) {
  if (!box || box._ruler || !matchMedia("(hover: hover) and (pointer: fine)").matches) return;
  const band = document.createElement("div"); band.className = "read-line"; band.setAttribute("aria-hidden", "true");
  box._ruler = band; box.classList.add("has-read-line");
  let raf = 0, px = 0, py = 0, inside = false;
  const caretAt = (x, y) => {
    if (document.caretPositionFromPoint) { const c = document.caretPositionFromPoint(x, y); return c && [c.offsetNode, c.offset]; }
    const r = document.caretRangeFromPoint?.(x, y); return r && [r.startContainer, r.startOffset];
  };
  const hide = () => band.classList.remove("on");
  const update = () => {
    raf = 0;
    if (!inside) return hide();
    const c = caretAt(px, py), t = c?.[0];
    if (!t || t.nodeType !== 3 || !t.length || !box.contains(t) || t.parentElement.closest("button, input, textarea, select, .doc-view, .rb-embed, .player")) return hide();
    const i = Math.min(c[1], t.length - 1), rg = document.createRange();
    rg.setStart(t, i); rg.setEnd(t, i + 1);
    const r = rg.getBoundingClientRect();
    if (!r.height || py < r.top - 3 || py > r.bottom + 3) return hide(); // мөрийн хоорондох зай, хоосон хэсэг
    const blk = t.parentElement.closest("p, li, h1, h2, h3, h4, h5, h6, blockquote, td, th, pre, figcaption, .rb-text, .lesson-content") || box;
    const host = box.getBoundingClientRect(), b = blk.getBoundingClientRect();
    if (!band.isConnected) box.prepend(band);
    band.style.cssText = `top:${r.top - host.top - box.clientTop - 3}px;height:${r.height + 6}px;left:${Math.max(0, b.left - host.left - box.clientLeft - 8)}px;width:${Math.min(host.width, b.width + 16)}px`;
    band.classList.add("on");
  };
  const queue = () => { if (!raf) raf = requestAnimationFrame(update); };
  box.addEventListener("pointermove", (e) => { if (e.pointerType === "touch") return; px = e.clientX; py = e.clientY; inside = true; queue(); }, { passive: true });
  box.addEventListener("pointerleave", () => { inside = false; hide(); });
  addEventListener("scroll", () => { if (inside) queue(); }, { passive: true, capture: true });
}

/* ---------- Хичээл үзэх үеийн анхаарал: өөр таб, цонх руу шилжвэл сануулга, 3 дахь удаад зогсоно ---------- */
const WATCH_MAX_WARN = 3;
// tone: "danger" (таб солих — улаан, сэгсрэлт, дохио), "stop" (хичээл зогссон), "info" (идэвхийн шалгалт).
function watchAlert(html, btn, onClose, tone = "info") {
  $(".watch-alert")?.remove();
  const el = document.createElement("div");
  el.className = `watch-alert wa-${tone}`; el.setAttribute("role", "alertdialog"); el.setAttribute("aria-modal", "true");
  el.innerHTML = `<div class="watch-card">${html}<button class="btn ${tone === "info" ? "btn-gold" : "wa-btn"} btn-block">${btn}</button></div>`;
  document.body.append(el);
  if (tone !== "info") alarm(tone);
  const b = $("button", el); b.focus();
  b.onclick = () => { el.remove(); onClose?.(); };
  return el;
}
// Анхааруулгын дохио: богино хоёр аялгуу (WebAudio) + утсан дээр чичиргээ.
let alarmCtx = null;
function alarm(kind) {
  navigator.vibrate?.(kind === "stop" ? [260, 90, 260, 90, 260] : [140, 70, 140]);
  try {
    alarmCtx ||= new (window.AudioContext || window.webkitAudioContext)();
    const ctx = alarmCtx, t0 = ctx.currentTime + 0.02;
    ctx.resume?.();
    const notes = kind === "stop" ? [[523, 0], [392, 0.2], [262, 0.4]] : [[988, 0], [740, 0.16], [988, 0.32]];
    for (const [f, dt] of notes) {
      const o = ctx.createOscillator(), g = ctx.createGain();
      o.type = "triangle"; o.frequency.value = f;
      g.gain.setValueAtTime(0.0001, t0 + dt);
      g.gain.exponentialRampToValueAtTime(0.22, t0 + dt + 0.02);
      g.gain.exponentialRampToValueAtTime(0.0001, t0 + dt + 0.15);
      o.connect(g).connect(ctx.destination); o.start(t0 + dt); o.stop(t0 + dt + 0.17);
    }
  } catch {}
}
// Курсор хичээлийн цонхноос гарахад: тусдаа шар сануулга (хичээл зогсохгүй, багшид тэмдэглэгдэнэ).
function cursorAlert(n) {
  $(".cursor-alert")?.remove();
  const el = document.createElement("div");
  el.className = "cursor-alert"; el.setAttribute("role", "status");
  el.innerHTML = `<span class="ca-ico" aria-hidden="true"><svg viewBox="0 0 24 24" width="26" height="26"><path d="M5 3l14 7.5-6.2 1.6L9.6 18z" fill="currentColor" stroke="#fff" stroke-width="1.4" stroke-linejoin="round"/></svg></span>
    <span class="ca-txt"><b>Курсор хичээлийн цонхноос гарлаа</b><small>Курсороо хичээл дээрээ байлгаарай · ${n} дахь удаа — багшид тэмдэглэгдэнэ</small></span>`;
  document.body.append(el);
  return el;
}
// watchLesson нь хичээлийн цонх нээлттэй байх хугацааг хянаж, хаагдахад зогсоох функц буцаана.
// Өөр таб/цонх руу шилжвэл сануулга (3 дахь удаад зогсоно), 5 минут тутам "Та үзэж байна уу?" (30 сек),
// хөдөлгөөнт усан тэмдэг (нэр, ID, IP), идэвхтэй хугацааны тоолуур.
const WATCH_PING_SEC = 300, WATCH_PING_ANSWER = 30, WATCH_IDLE_MS = 120000;
function watchLesson({ courseId, lessonId, modal, onStop, onActive, maxWarn }) {
  const WATCH_MAX_WARN = Math.max(1, +maxWarn || 3); // багш сургалтын тохиргоонд тоогоор оруулна
  if (!Auth.token) return () => {};
  let sid = null, owner = false, warns = 0, stopped = false, away = false, awayAt = 0, last = Date.now(), lastInput = Date.now(), idleLogged = false;
  let acc = { active: 0, idle: 0, away: 0 }, events = [], pingAt = Date.now() + WATCH_PING_SEC * 1000, pingTimer = null, wmTimer = null;
  // Идэвхийн оноо E = идэвхтэй / нийт (сүүлийн 5 минут). 3+ минут хэмжээд E < 0.1 бол хичээл зогсоно.
  const win = [];
  const engagement = () => { const a = win.reduce((n, x) => n + x[0], 0), t = win.reduce((n, x) => n + x[1], 0); return { e: t ? a / t : 1, t }; };
  const card = $(".modal-card", modal);
  const badge = document.createElement("div");
  badge.className = "watch-badge";
  const paint = () => { badge.innerHTML = `👁 Анхаарлын горим · өөр таб руу шилжвэл сануулга <b>${warns}/${WATCH_MAX_WARN}</b>`; badge.classList.toggle("warned", warns > 0); };
  paint(); card.prepend(badge);
  const wm = document.createElement("div"); wm.className = "wm"; wm.setAttribute("aria-hidden", "true");
  const playing = () => $$("video, audio", modal).some((v) => !v.paused && !v.ended) || !!modal._ytPlaying;
  const tick = () => {
    const now = Date.now(), dt = Math.round((now - last) / 1000); last = now;
    if (dt <= 0) return;
    if (away) acc.away += dt;
    else if (now - lastInput > WATCH_IDLE_MS && !playing()) { acc.idle += dt; if (!idleLogged) { idleLogged = true; events.push({ type: "idle", detail: "2 минутаас дээш хөдөлгөөнгүй" }); } }
    else acc.active += dt;
  };
  const send = (end) => {
    tick();
    win.push([acc.active, acc.active + acc.idle + acc.away]); while (win.length > 20) win.shift();
    if (!sid) return;
    const body = JSON.stringify({ sessionId: sid, ...acc, events, end: end || "" });
    acc = { active: 0, idle: 0, away: 0 }; events = [];
    // Protobuf API (Connect-JSON): BeatOnce — gRPC Beat урсгалтай ижил логик.
    fetch("/surgalt.v1.Surgalt/BeatOnce", { method: "POST", keepalive: true, headers: { "Content-Type": "application/json", Authorization: "Bearer " + Auth.token }, body })
      .then((r) => r.json()).then((d) => { if (d?.activeDoneSec != null || d?.ok) onActive?.(d.activeDoneSec || 0); }).catch(() => {});
  };
  api("/surgalt.v1.Surgalt/StartSession", { method: "POST", body: { courseId, lessonId } })
    .then((d) => {
      sid = d.sessionId; owner = !!d.policy?.owner;
      onActive?.(d.policy?.activeDoneSec || 0);
      if (owner) { badge.remove(); return; }
      // Усан тэмдэг: хуудсыг зураг авбал хэн болох нь харагдана.
      const w = d.watermark || {};
      wm.textContent = `${w.name || ""} · #${w.id || ""} · ${w.ip || ""}`;
      card.append(wm);
      const move = () => { wm.style.left = 5 + Math.random() * 55 + "%"; wm.style.top = 12 + Math.random() * 70 + "%"; };
      move(); wmTimer = setInterval(move, 6000);
    }).catch(() => badge.remove());
  const pauseMedia = () => { $$("video, audio", modal).forEach((v) => v.pause()); modal._ytPause?.(); };
  // Өөр таб руу гарсан үед хуудасны гарчиг анивчина: «⚠️ Хичээл рүүгээ буцна уу!»
  const title0 = document.title; let flashT = 0;
  const flashOn = () => { if (owner || flashT) return; let f = false; flashT = setInterval(() => { document.title = (f = !f) ? "⚠️ Хичээл рүүгээ буцна уу!" : title0; }, 900); };
  const flashOff = () => { clearInterval(flashT); flashT = 0; document.title = title0; };
  const leave = () => { if (away || stopped) return; tick(); away = true; awayAt = Date.now(); pauseMedia(); flashOn(); };
  const back = () => {
    if (!away || stopped) return;
    tick(); away = false; flashOff();
    const sec = Math.max(1, Math.round((Date.now() - awayAt) / 1000));
    if (owner) return; // багш өөрийн хичээлийг шалгаж байна
    if (Date.now() - awayAt < 3000) { // санамсаргүй богино шилжилт (мэдэгдэл, дуудлага) — тоолохгүй, зөөлөн сануулна
      toast("👀 Хичээлдээ анхаараарай — дахин гарвал сануулга тоологдоно"); return;
    }
    warns++; paint();
    events.push({ type: "tab_switch", detail: `${sec} сек өөр цонхонд байсан (сануулга ${warns}/${WATCH_MAX_WARN})` });
    if (warns >= WATCH_MAX_WARN) return stop(`${WATCH_MAX_WARN} удаа хичээлээс гарсан`, `Та ${WATCH_MAX_WARN} удаа хичээлээс гарсан тул хичээл зогслоо.`);
    send();
    const left = WATCH_MAX_WARN - warns;
    watchAlert(`<span class="wa-ico" aria-hidden="true"><i>!</i></span>
      <span class="wa-eyebrow">Анхааруулга ${warns}/${WATCH_MAX_WARN}</span>
      <h3>Та хичээлээс гарлаа!</h3>
      <p>Хичээл үзэж байхдаа өөр таб, цонх руу шилжихгүй байна уу. Видео түр зогслоо.</p>
      <div class="wa-meter" aria-label="Сануулга ${warns}/${WATCH_MAX_WARN}">${Array.from({ length: WATCH_MAX_WARN }, (_, i) => `<i class="${i < warns ? "used" : ""} ${i === WATCH_MAX_WARN - 1 ? "last" : ""}">${i === WATCH_MAX_WARN - 1 ? "⛔" : i + 1}</i>`).join("")}</div>
      <p class="wa-left">${left === 1 ? "Дахиад гарвал хичээл <b>хаагдаж</b>, багшид мэдэгдэнэ!" : `Дахиад <b>${left}</b> удаа гарвал хичээл хаагдана`}</p>
      <p class="wa-away">⏱ ${sec} секунд өөр цонхонд байсан · багшид тэмдэглэгдлээ</p>`, "Ойлголоо, хичээл рүүгээ буцах", null, "danger");
  };
  const stop = (detail, msg) => {
    stopped = true;
    events.push({ type: "auto_block", detail });
    send("auto_block"); cleanup(true);
    onStop?.();
    watchAlert(`<span class="wa-ico" aria-hidden="true"><i>⛔</i></span>
      <span class="wa-eyebrow">Хичээл хаагдлаа</span>
      <h3>Анхаарал идэвхгүй тул хичээл зогслоо</h3>
      <p>${msg} Энэ тухай багшид мэдэгдсэн. Идэвхгүй байсан хугацаа суралцсан цагт тооцогдохгүй.</p>
      <p class="wa-away">Дахин үзэхдээ бусад цонхоо хаагаад, анхааралтай үзээрэй.</p>`, "Ойлголоо", null, "stop");
  };
  // 5 минут тутам идэвхийн шалгалт: 30 секундэд хариулахгүй бол хичээл зогсоно.
  const ping = () => {
    if (stopped || owner || !sid) return;
    pauseMedia();
    let left = WATCH_PING_ANSWER, answered = false;
    watchAlert(`<span class="watch-ico">🙋</span><h3>Та хичээлээ үргэлжлүүлэн үзэж байна уу?</h3>
      <p>Идэвхийн шалгалт. <b class="ping-left">${left}</b> секундэд хариулахгүй бол хичээл зогсоно.</p>`, "Тийм, үзэж байна", () => { answered = true; lastInput = Date.now(); });
    const t = setInterval(() => {
      if (answered || stopped) { clearInterval(t); return; }
      left--; const el = $(".watch-alert .ping-left"); if (el) el.textContent = left;
      if (left <= 0) { clearInterval(t); $(".watch-alert")?.remove(); events.push({ type: "ping_missed", detail: "30 секундэд хариулаагүй" }); stop("Идэвхийн шалгалтад хариулаагүй", "Та идэвхийн шалгалтад 30 секундэд хариулаагүй тул хичээл зогслоо."); }
    }, 1000);
  };
  pingTimer = setInterval(() => { if (!away && Date.now() >= pingAt) { pingAt = Date.now() + WATCH_PING_SEC * 1000; ping(); } }, 1000);
  // Таб нуугдах: найдвартай. Цонхны blur: видео тоглуулагч (iframe) дээр дарахад ч гардаг тул шалгана.
  const onVis = () => (document.hidden ? leave() : back());
  const onBlur = () => setTimeout(() => { if (document.activeElement?.tagName !== "IFRAME" && !document.hasFocus()) leave(); }, 0);
  const onFocus = () => { if (!document.hidden) back(); };
  const onInput = () => { lastInput = Date.now(); idleLogged = false; };
  const inputs = ["pointermove", "pointerdown", "keydown", "wheel", "touchstart"];
  // Курсор хичээлийн цонхноос 2.5 сек+ гарвал тусдаа (шар) сануулга; буцаж ороход алга болж, хугацааг нь тэмдэглэнэ.
  const fine = matchMedia("(hover: hover) and (pointer: fine)").matches;
  let outT = 0, outAt = 0, outs = 0, ca = null;
  const overlayOpen = () => $(".watch-alert") || $(".dv-full") || $$(".modal.open").some((m) => m !== modal);
  const cursorBack = () => {
    clearTimeout(outT); if (!ca) return;
    events.push({ type: "cursor_out", detail: `${Math.max(1, Math.round((Date.now() - outAt) / 1000))} сек курсор хичээлийн цонхноос гадуур (${outs} дахь удаа)` });
    const el = ca; ca = null; el.classList.add("out"); setTimeout(() => el.remove(), 250); card.classList.remove("cursor-out");
  };
  const onLeaveCard = (e) => {
    if (!fine || owner || stopped || e.pointerType === "touch" || (e.relatedTarget && card.contains(e.relatedTarget))) return;
    clearTimeout(outT); outAt = Date.now();
    outT = setTimeout(() => { if (stopped || away || owner || ca || overlayOpen() || card.matches(":hover")) return; outs++; ca = cursorAlert(outs); card.classList.add("cursor-out"); }, 2500);
  };
  card.addEventListener("pointerleave", onLeaveCard); card.addEventListener("pointerenter", cursorBack);
  document.addEventListener("visibilitychange", onVis);
  addEventListener("blur", onBlur); addEventListener("focus", onFocus);
  inputs.forEach((n) => addEventListener(n, onInput, { passive: true }));
  const beat = setInterval(() => {
    send();
    const { e, t } = engagement();
    if (!stopped && !owner && t >= 180 && e < 0.1) stop(`Идэвхийн оноо ${e.toFixed(2)} (0.1-ээс бага)`, "Сүүлийн хэдэн минутад та хичээлд бараг идэвхгүй байсан тул хичээл зогслоо.");
  }, 15000);
  // Агуулгыг хуулахаас хамгаална: хуулах, тайрах, баруун товч, сонгох, чирэх. Хуулах оролдлогыг багшид бүртгэнэ.
  const editable = (t) => t.closest?.("input, textarea, select, [contenteditable]");
  const noCopy = (e) => { if (editable(e.target) || owner) return; e.preventDefault(); events.push({ type: "copy", detail: "Хичээлийн агуулгыг хуулах гэсэн" }); toast("Хичээлийн агуулгыг хуулах боломжгүй"); };
  const block = (e) => { if (!editable(e.target) && !owner) e.preventDefault(); };
  modal.addEventListener("copy", noCopy); modal.addEventListener("cut", noCopy);
  ["contextmenu", "selectstart", "dragstart"].forEach((n) => modal.addEventListener(n, block));
  modal.classList.add("protected");
  let done = false;
  const cleanup = (fromStop) => {
    if (done) return; done = true;
    clearInterval(beat); clearInterval(pingTimer); clearInterval(wmTimer);
    modal.removeEventListener("copy", noCopy); modal.removeEventListener("cut", noCopy);
    ["contextmenu", "selectstart", "dragstart"].forEach((n) => modal.removeEventListener(n, block));
    modal.classList.remove("protected");
    document.removeEventListener("visibilitychange", onVis);
    removeEventListener("blur", onBlur); removeEventListener("focus", onFocus);
    inputs.forEach((n) => removeEventListener(n, onInput));
    card.removeEventListener("pointerleave", onLeaveCard); card.removeEventListener("pointerenter", cursorBack);
    clearTimeout(outT); ca?.remove(); ca = null; card.classList.remove("cursor-out"); flashOff();
    badge.remove(); wm.remove();
    if (!fromStop) send("closed");
  };
  return () => cleanup(false);
}

// Видеог урагш гүйлгэхгүй: үзсэн хэсэг рүүгээ буцаж болно; нэг удаа бүрэн үзсэний дараа чөлөөтэй.
// watched — {bid: true}; onWatched(bid) — бүрэн үзэхэд.
let ytApi = null;
function loadYT() {
  ytApi ||= new Promise((res) => {
    if (window.YT?.Player) return res(window.YT);
    window.onYouTubeIframeAPIReady = () => res(window.YT);
    const sc = document.createElement("script"); sc.src = "https://www.youtube.com/iframe_api"; document.head.append(sc);
  });
  return ytApi;
}
// progressBase: "/api/courses/{id}/lessons/{lid}" — өгөгдсөн бол үзэлтийн зураглалыг 10 секундийн
// хэсгүүдээр бичиж, багшид аль хэсгийг давтаж, аль хэсгийг алгассаныг харуулна.
function guardVideos(box, { watched = {}, owner = false, onWatched, modal, progressBase }) {
  const note = () => toast("Үзээгүй хэсэг рүү гүйлгэх боломжгүй — үзсэн хэсэгтээ буцаж болно. Бүрэн үзсэний дараа чөлөөтэй.");
  const BUCKET = 10;
  const stoppers = [];
  const blocks = [...$$("[data-bid] video, [data-bid] iframe[data-yt]", box), ...$$("#player video, #player iframe[data-yt]", modal)];
  for (const v of blocks) {
    const bid = v.closest("[data-bid]")?.dataset.bid || "main";
    let free = owner || !!watched[bid], max = 0;
    const buckets = {}, mark = (t) => { if (progressBase && !owner) buckets[Math.floor(t / BUCKET)] = (buckets[Math.floor(t / BUCKET)] || 0) + 1; };
    const flush = () => {
      if (!progressBase || owner || !Object.keys(buckets).length) return;
      const body = { duration: Math.round(max), buckets: { ...buckets } };
      for (const k in buckets) delete buckets[k];
      fetch(`${progressBase}/progress/${bid}`, { method: "POST", keepalive: true, headers: { "Content-Type": "application/json", Authorization: "Bearer " + Auth.token }, body: JSON.stringify(body) }).catch(() => {});
    };
    const flushTimer = progressBase ? setInterval(flush, 20000) : null;
    stoppers.push(() => { clearInterval(flushTimer); flush(); });
    if (v.tagName === "VIDEO") {
      // 6 минутын хэсгүүдтэй бол дараалан тасралтгүй тоглоно; явц, гүйлгэх хориг нь нийт хугацаагаар.
      const fig = v.closest("[data-parts]"), parts = fig ? JSON.parse(fig.dataset.parts || "[]") : [];
      let lastBucket = -1, idx = 0, offset = 0, maxPart = 0, unlockedPart = 0;
      const btns = fig ? $$("[data-vpart]", fig) : [];
      const paintParts = () => btns.forEach((b, i) => { b.classList.toggle("on", i === idx); b.disabled = !free && i > unlockedPart; });
      const loadPart = (i, autoplay) => { idx = i; offset = 0; for (let k = 0; k < i; k++) offset += SEG; maxPart = 0; v.src = parts[i]; if (autoplay) v.play().catch(() => {}); paintParts(); };
      const SEG = 360;
      paintParts();
      btns.forEach((b) => b.addEventListener("click", () => { const i = +b.dataset.vpart; if (!free && i > unlockedPart) { note(); return; } loadPart(i, true); }));
      v.addEventListener("timeupdate", () => {
        if (!v.seeking && v.currentTime > maxPart && v.currentTime - maxPart < 3) maxPart = v.currentTime;
        const g = offset + v.currentTime; if (!v.seeking && g > max && g - max < 3) max = g;
        const b = Math.floor(g / BUCKET); if (b !== lastBucket) { lastBucket = b; mark(g); }
      });
      v.addEventListener("seeking", () => { if (!free && idx >= unlockedPart && v.currentTime > maxPart + 1) { v.currentTime = maxPart; note(); } });
      v.addEventListener("ratechange", () => { if (!free && v.playbackRate > 2) v.playbackRate = 2; });
      const release = lockedControls(v, () => free, note, () => (idx < unlockedPart ? Infinity : maxPart));
      v.addEventListener("ended", () => {
        if (parts.length > 1 && idx < parts.length - 1) { unlockedPart = Math.max(unlockedPart, idx + 1); loadPart(idx + 1, true); return; } // дараагийн хэсэг
        if (!free) { free = true; onWatched?.(bid); release(); paintParts(); } flush();
      });
      if (free) release();
      continue;
    }
    // YouTube: IFrame API-аар байрлалыг хянана.
    loadYT().then((YT) => {
      let poll = null, lastBucket = -1;
      const p = new YT.Player(v, { events: {
        onStateChange: (e) => {
          modal._ytPlaying = e.data === YT.PlayerState.PLAYING;
          if (e.data === YT.PlayerState.ENDED && !free) { free = true; onWatched?.(bid); flush(); }
        },
      } });
      modal._ytPause = () => { try { p.pauseVideo(); } catch {} };
      // Хамгаалах давхарга ба өөрийн удирдлага (YouTube-ийн холбоос, лого дарагдахгүй).
      const wrap = v.closest(".yt-wrap");
      if (wrap) {
        const fmt = (t) => `${Math.floor(t / 60)}:${String(Math.floor(t % 60)).padStart(2, "0")}`;
        const toggle = () => { try { p.getPlayerState() === YT.PlayerState.PLAYING ? p.pauseVideo() : p.playVideo(); } catch {} };
        $$("[data-yt-toggle]", wrap).forEach((b) => (b.onclick = toggle));
        $(".vg-mute", wrap).onclick = () => { try { p.isMuted() ? p.unMute() : p.mute(); } catch {} };
        $(".vg-full", wrap).onclick = () => (document.fullscreenElement ? document.exitFullscreen?.() : wrap.requestFullscreen?.());
        $(".vg-prog", wrap).onclick = (e) => { // үзсэн хэсэгтээ буцаж гүйлгэнэ; бүрэн үзсэний дараа чөлөөтэй
          const r = e.currentTarget.getBoundingClientRect(), d = p.getDuration?.() || 0, t = d * (e.clientX - r.left) / r.width;
          if (!free && t > max + 0.5) { p.seekTo(max, true); return note(); }
          p.seekTo(Math.max(0, t), true);
        };
        wrap.addEventListener("keydown", (e) => { if (e.key === " ") { e.preventDefault(); toggle(); } });
        setInterval(() => {
          if (typeof p.getCurrentTime !== "function") return;
          const t = p.getCurrentTime() || 0, d = p.getDuration?.() || 0, st = p.getPlayerState?.();
          $(".vg-time", wrap).textContent = `${fmt(t)} / ${fmt(d)}`; $(".vg-prog i", wrap).style.width = d ? (t / d) * 100 + "%" : "0";
          const playing = st === YT.PlayerState.PLAYING; $(".vg-play", wrap).textContent = playing ? "❚❚" : "▶"; wrap.classList.toggle("playing", playing);
          $(".vg-mute", wrap).textContent = p.isMuted?.() ? "🔇" : "🔊"; $(".vg-prog", wrap).classList.toggle("seekable", true); $(".vg-seen", wrap).style.width = d ? Math.min(1, (free ? d : Math.max(max, t)) / d) * 100 + "%" : "0";
        }, 400);
      }
      poll = setInterval(() => {
        if (!document.body.contains(v) && !p.getIframe?.()?.isConnected) return clearInterval(poll);
        if (typeof p.getCurrentTime !== "function") return;
        const t = p.getCurrentTime();
        if (!free && t > max + 2.5) { p.seekTo(max, true); note(); return; }
        if (t > max) max = t;
        const b = Math.floor(t / BUCKET); if (b !== lastBucket) { lastBucket = b; mark(t); }
      }, 500);
    }).catch(() => {});
  }
  return () => stoppers.forEach((f) => f());
}

/* ---------- Шалгалт: нэг дор өгнө, хугацаатай, хатуу хамгаалалттай ----------
   Шалгалтын үеэр өөр таб/цонх руу шилжих, хуулах, тайрах үйлдэл хийвэл шалгалт шууд хаагдаж,
   тэр хүртэлх хариултаар дүгнэгдэнэ. Бүгд багшийн логт бичигдэнэ. */
async function examCard(box, { courseId, lessonId, title, onFinish }) {
  const base = `/api/courses/${courseId}/lessons/${lessonId}`;
  const card = document.createElement("section"); card.className = "exam-card"; box.append(card);
  if (!Auth.token) { card.innerHTML = `<h3>${icon("exam", 20)} Шалгалт</h3><p>Шалгалт өгөхийн тулд нэвтэрнэ үү.</p><a class="btn btn-gold" href="/login?next=${encodeURIComponent(location.pathname)}">Нэвтрэх</a>`; return; }
  const draw = async () => {
    let info;
    try { info = await api(`${base}/exam`); } catch (e) { card.innerHTML = `<p class="form-error">${esc(e.message)}</p>`; return; }
    const ex = info.exam, best = info.attempts.filter((a) => a.status !== "active").reduce((m, a) => Math.max(m, a.pct), -1);
    const active = info.attempts.find((a) => a.status === "active"), due = info.due || { open: true };
    card.innerHTML = `<h3>${icon("exam", 20)} Шалгалт</h3>
      <ul class="exam-facts"><li>❓ <b>${info.questions}</b> асуулт</li><li>⏱ ${ex.time_min ? `<b>${ex.time_min}</b> минут` : "Хугацаа хязгааргүй"}</li>${due.start_at ? `<li>▶ Эхлэх: <b>${fmtDate(due.start_at)}</b></li>` : ""}${due.at ? `<li>📅 ${due.late ? `<span class="an-bad">Хугацаа дууссан</span> (${fmtDate(due.at)})` : `<b>${fmtDate(due.at)}</b> хүртэл`}</li>` : ""}${due.entry_fee ? `<li>💳 Оролцооны төлбөр: <b>${money(due.entry_fee)}</b>${due.paid ? " ✓" : ""}</li>` : ""}
        <li>🎯 Тэнцэх: <b>${ex.pass_pct}%</b></li><li>🔁 ${info.left < 0 ? "Оролдлого хязгааргүй" : `Үлдсэн оролдлого: <b>${info.left}</b>`}</li>${best >= 0 ? `<li>🏆 Таны шилдэг: <b>${best}%</b></li>` : ""}</ul>
      <div class="exam-rules"><b>⚠️ Дүрэм:</b> Шалгалтын үеэр өөр таб, цонх руу шилжих эсвэл текст хуулах үед шалгалт <b>шууд хаагдаж</b>, тэр хүртэлх хариултаар дүгнэгдэнэ. Энэ тухай багшид мэдэгдэнэ.</div>
      ${info.attempts.length ? `<table class="tbl"><thead><tr><th>Огноо</th><th>Оноо</th><th>Төлөв</th></tr></thead><tbody>${info.attempts.map((a) => `<tr><td>${fmtDate(a.started_at)}</td><td>${a.status === "active" ? "—" : a.pct + "%" + (a.passed ? " ✓" : "")}</td><td>${a.status === "terminated" ? `⛔ Хаагдсан · ${esc(a.reason || "")}` : a.status === "submitted" ? (a.passed ? "Тэнцсэн" : "Тэнцээгүй") : a.status === "expired" ? "Хугацаа хэтэрсэн" : "Үргэлжилж байна"}</td></tr>`).join("")}</tbody></table>` : ""}
      ${dueNotice(due)}
      ${due.closed || due.not_started ? "" : due.need_pay ? `<button class="btn btn-gold btn-lg" data-late-pay>💳 ${money(due.fee)} төлж шалгалтаа нээх</button>` : active || info.left !== 0 ? `<button class="btn btn-gold btn-lg" data-exam-start>${active ? "▶ Шалгалтаа үргэлжлүүлэх" : "▶ Шалгалт эхлүүлэх"}</button>` : `<p class="muted">Оролдлогын тоо дууссан.</p>`}`;
    bindLatePay(card, base, title ? `«${title}» — шалгалтын төлбөр` : "Шалгалтын төлбөр", draw);
    const b = $("[data-exam-start]", card);
    if (b) b.onclick = async () => {
      if (!active && !confirm("Шалгалт эхлүүлэх үү? Эхэлсний дараа өөр цонх руу шилжвэл шалгалт хаагдана.")) return;
      b.disabled = true;
      try { const d = await api(`${base}/exam/start`, { method: "POST" }); runExam({ base, courseId, lessonId, title, data: d, onClose: () => { draw(); onFinish?.(); } }); }
      catch (e) { toast(e.message, true); } finally { b.disabled = false; }
    };
  };
  await draw();
}

// Хугацааны мэдэгдэл (шалгалт, даалгаварт ижил): хоцорсон → төлбөргүй / төлбөртэй / хаалттай.
function dueNotice(due) {
  if (!due) return "";
  if (due.not_started) return `<div class="exam-rules"><b>🕒 Эхлээгүй.</b> ${fmtDate(due.start_at)}-д нээгдэнэ.</div>`;
  if (due.closed) return `<div class="exam-rules"><b>⛔ Хугацаа дууссан.</b> Хаалттай — багштайгаа холбогдоно уу.</div>`;
  if (due.need_pay) {
    const parts = [];
    if (due.need_entry) parts.push(`оролцооны төлбөр ${money(due.entry_fee)}`);
    if (due.need_late) parts.push(`хоцролтын төлбөр ${money(due.late_fee)}`);
    return `<div class="exam-rules"><b>${due.need_late ? "⏰ Хугацаа хоцорсон." : "💳 Төлбөртэй."}</b> Үргэлжлүүлэхийн тулд ${parts.join(" + ")} = <b>${money(due.fee)}</b> төлнө.</div>`;
  }
  if (due.paid) return `<div class="exam-rules">✓ Төлбөр төлсөн — үргэлжлүүлж болно.</div>`;
  if (due.late) return `<div class="exam-rules">⏰ Хугацаа өнгөрсөн ч төлбөргүй үргэлжлүүлж болно (хоцорсон гэж тэмдэглэгдэнэ).</div>`;
  return "";
}
// Төлбөртэй шалгалт/даалгавар: хичээлийг нээмэгц «Та төлбөрөө төлнө үү» цонх (QR) нэг удаа өөрөө гарна; хаасан ч товч нь үлдэнэ.
function bindLatePay(card, base, label, after) {
  const b = $("[data-late-pay]", card); if (!b) return;
  b.onclick = async () => {
    b.disabled = true;
    try { const d = await api(`${base}/late-pay`, { method: "POST" }); if (d.unlocked) return after(); window.SG_pay?.(d.order, d.payment, label, () => { toast("✓ Нээгдлээ"); after(); }, { name: label }); }
    catch (e) { toast(e.message, true); } finally { b.disabled = false; }
  };
  if (!card.dataset.payShown) { card.dataset.payShown = "1"; b.click(); }
}

// Даалгавар: нөхцөл (блокууд дээр), хугацаа, хариу илгээх (текст + файл), багшийн дүн.
async function assignmentCard(box, { courseId, lessonId }) {
  const base = `/api/courses/${courseId}/lessons/${lessonId}`;
  const card = document.createElement("section"); card.className = "exam-card asg-card"; box.append(card);
  if (!Auth.token) { card.innerHTML = `<h3>${icon("clip", 20)} Даалгавар</h3><p>Хариу илгээхийн тулд нэвтэрнэ үү.</p><a class="btn btn-gold" href="/login?next=${encodeURIComponent(location.pathname)}">Нэвтрэх</a>`; return; }
  const draw = async () => {
    let info;
    try { info = await api(`${base}/assignment`); } catch (e) { card.innerHTML = `<p class="form-error">${esc(e.message)}</p>`; return; }
    const a = info.assignment, due = info.due || { open: true }, sub = info.submission;
    card.innerHTML = `<h3>${icon("clip", 20)} Даалгавар</h3>
      <ul class="exam-facts">${due.start_at ? `<li>▶ Эхлэх: <b>${fmtDate(due.start_at)}</b></li>` : ""}${due.at ? `<li>📅 ${due.late ? `<span class="an-bad">Хугацаа дууссан</span> (${fmtDate(due.at)})` : `<b>${fmtDate(due.at)}</b> хүртэл`}</li>` : "<li>📅 Хугацаагүй</li>"}<li>🎯 Дээд оноо: <b>${a.max_score || 100}</b></li>${due.entry_fee ? `<li>💳 Төлбөр: <b>${money(due.entry_fee)}</b>${due.paid ? " ✓" : ""}</li>` : ""}<li>✍️ Хариу: текст ба холбоос</li></ul>
      ${dueNotice(due)}
      ${sub ? `<div class="sub-mine"><b>Таны хариу</b> <small class="muted">${fmtDate(sub.submitted_at)}${sub.late ? " · хоцорсон" : ""}</small>${sub.text ? `<p>${esc(sub.text)}</p>` : ""}${sub.links?.length ? `<p>${sub.links.map((u) => `<a href="${esc(u)}" target="_blank" rel="noopener noreferrer">🔗 ${esc(u.replace(/^https?:\/\//, "").slice(0, 60))}</a>`).join("<br>")}</p>` : ""}${sub.files?.length ? `<p>${sub.files.map((f) => `<a href="${esc(f.url)}" target="_blank" rel="noopener">📄 ${esc(f.name)}</a>`).join(" · ")}</p>` : ""}
        ${sub.score !== undefined ? `<div class="sub-score">✅ Дүн: <b>${sub.score}/${a.max_score || 100}</b>${sub.feedback ? `<p>${esc(sub.feedback)}</p>` : ""}</div>` : `<p class="muted small">Багш хараахан дүгнээгүй байна.</p>`}</div>` : ""}
      ${due.closed || due.not_started ? "" : due.need_pay ? `<button class="btn btn-gold btn-lg" data-late-pay>💳 ${money(due.fee)} төлж даалгавраа илгээх</button>`
        : `<form class="sub-form"><label>${sub ? "Хариугаа шинэчлэх" : "Хариу"}<textarea name="text" rows="5" maxlength="20000" placeholder="Хариугаа энд бичнэ үү…">${esc(sub?.text || "")}</textarea></label>
          <label>Холбоос (Google Docs, GitHub, видео… мөр тус бүрд нэг, 5 хүртэл)<textarea name="links" rows="2" placeholder="https://…">${esc((sub?.links || []).join("\n"))}</textarea></label>
          <p class="form-error" role="alert"></p><button class="btn btn-gold">${sub ? "Дахин илгээх" : "Илгээх"}</button></form>`}`;
    bindLatePay(card, base, "Даалгаврын төлбөр", draw);
    const f = $(".sub-form", card);
    if (f) f.onsubmit = async (e) => {
      e.preventDefault();
      const err = $(".form-error", f), b = $("button", f); err.textContent = ""; b.disabled = true;
      try {
        await api(`${base}/submit`, { method: "POST", body: { text: f.text.value, links: f.links.value.split(/[\n,\s]+/).filter(Boolean) } });
        toast("✓ Даалгавар илгээгдлээ"); celebrate?.(); await draw(); document.dispatchEvent(new CustomEvent("sg:quiz-mastered"));
      } catch (x) { err.textContent = x.message; } finally { b.disabled = false; }
    };
  };
  await draw();
}

function runExam({ base, courseId, lessonId, title, data, onClose }) {
  const att = data.attempt, key = "sg_exam_" + att.id;
  let saved = {}; try { saved = JSON.parse(sessionStorage.getItem(key) || "{}"); } catch {}
  let finished = false, sid = null, tick = null, beat = null;
  const lessonModal = $("#lessonModal"); if (lessonModal?.classList.contains("open")) closeModal(lessonModal);
  const el = document.createElement("div"); el.className = "exam"; el.setAttribute("role", "dialog"); el.setAttribute("aria-label", "Шалгалт");
  el.innerHTML = `<header class="exam-bar"><strong>📝 ${esc(title)}</strong><span class="exam-prog"></span><span class="exam-timer" aria-live="polite"></span><button class="btn btn-gold btn-sm" data-exam-submit>Илгээх</button></header>
    <nav class="exam-nav" aria-label="Асуултаар шилжих">${data.questions.map((_, i) => `<button type="button" data-goq="${i}" aria-label="${i + 1}-р асуулт руу очих">${i + 1}</button>`).join("")}
      <span class="exam-nav-legend" aria-hidden="true"><i class="lg-todo"></i>хариулаагүй<i class="lg-done"></i>хариулсан</span></nav>
    <div class="exam-body"><div class="exam-qs">${data.questions.map((b, i) => quizFormHTML(b, true, i + 1)).join("")}</div>
      <div class="exam-submit"><button class="btn btn-gold btn-lg" data-exam-submit>Шалгалтаа илгээх</button></div></div>`;
  document.body.append(el); document.documentElement.classList.add("reader-open");
  const body = $(".exam-body", el); bindQuizPicks(body);
  // Хадгалсан хариултыг сэргээнэ (хуудсыг дахин ачаалсан бол).
  $$(".rb-quiz", el).forEach((f) => {
    const a = saved[f.dataset.quiz]; if (!a) return;
    if (a.answer) $$("input", f).forEach((i) => (i.checked = a.answer.includes(+i.value)));
    if (a.text) $(".rq-text", f).value = a.text;
    if (a.match) $$("select", f).forEach((x, i) => (x.value = a.match[i] >= 0 ? a.match[i] : ""));
    if (a.point) { f.dataset.px = a.point[0]; f.dataset.py = a.point[1]; const pt = $(".rq-pt", f); if (pt) { pt.hidden = false; pt.style.left = a.point[0] + "%"; pt.style.top = a.point[1] + "%"; } }
  });
  const answers = () => { const out = {}; $$(".rb-quiz", el).forEach((f) => { const a = quizAnswer(f); if (a) out[f.dataset.quiz] = a; }); return out; };
  // Асуултын дугаарын мөр: хариулаагүй (улбар шар) / хариулсан (хар хөх), одоо харж буй нь тодорно; дарахад шилжинэ.
  const forms = $$(".exam-qs > .rb-quiz", el), navBtns = $$("[data-goq]", el);
  const todoNums = (a = answers()) => forms.map((f, i) => (a[f.dataset.quiz] ? 0 : i + 1)).filter(Boolean);
  const progress = () => {
    const a = answers(), n = Object.keys(a).length;
    $(".exam-prog", el).textContent = `${n}/${data.questions.length} хариулсан`;
    forms.forEach((f, i) => navBtns[i]?.classList.toggle("done", !!a[f.dataset.quiz]));
    try { sessionStorage.setItem(key, JSON.stringify(a)); } catch {}
  };
  el.addEventListener("change", progress); el.addEventListener("input", progress); progress();
  const goQ = (i) => {
    const f = forms[i]; if (!f) return;
    f.scrollIntoView({ behavior: reduce ? "auto" : "smooth", block: "start" });
    f.classList.remove("q-flash"); void f.offsetWidth; f.classList.add("q-flash");
  };
  if ("IntersectionObserver" in window) { // гүйлгэхэд одоо харж буй асуултыг тодруулна
    const seen = new IntersectionObserver((es) => es.forEach((e) => {
      if (!e.isIntersecting) return;
      const i = forms.indexOf(e.target);
      navBtns.forEach((b, k) => b.classList.toggle("cur", k === i));
      navBtns[i]?.scrollIntoView({ block: "nearest", inline: "nearest" });
    }), { root: body, rootMargin: "-20% 0px -70% 0px" });
    forms.forEach((f) => seen.observe(f));
  }
  // Цаг
  const deadline = att.deadline_at ? new Date(att.deadline_at).getTime() - (new Date(data.now).getTime() - Date.now()) : 0;
  if (deadline) tick = setInterval(() => {
    const left = Math.max(0, Math.round((deadline - Date.now()) / 1000));
    const t = $(".exam-timer", el); t.textContent = `⏱ ${Math.floor(left / 60)}:${String(left % 60).padStart(2, "0")}`; t.classList.toggle("low", left < 60);
    if (left <= 0) submit("", "Хугацаа дууслаа — хариултууд автоматаар илгээгдлээ.");
  }, 500);
  // Идэвхийн сесс (багшийн самбарт), зөрчлийг оролдлоготой холбоно.
  api("/surgalt.v1.Surgalt/StartSession", { method: "POST", body: { courseId, lessonId, kind: "exam" } }).then((d) => { sid = d.sessionId; }).catch(() => {});
  let last = Date.now();
  const beatNow = (events = [], end = "") => { if (!sid) return; const dt = Math.round((Date.now() - last) / 1000); last = Date.now();
    fetch("/surgalt.v1.Surgalt/BeatOnce", { method: "POST", keepalive: true, headers: { "Content-Type": "application/json", Authorization: "Bearer " + Auth.token },
      body: JSON.stringify({ sessionId: sid, active: dt, events, end, attemptId: att.id }) }).catch(() => {}); };
  beat = setInterval(() => beatNow(), 15000);
  // Хатуу хамгаалалт
  // Зөрчил: шалгалт шууд хаагдана; суралцагч буцаж ирэхэд анхаарал татахуйц (улаан, дохиотой) цонх гарна.
  const stopAlert = (label) => {
    const show = () => watchAlert(`<span class="wa-ico" aria-hidden="true"><i>⛔</i></span><span class="wa-eyebrow">Шалгалт хаагдлаа</span>
      <h3>Шалгалтын үеэр ${esc(label)}!</h3><p>Дүрмийн дагуу шалгалт тань тэр мөчийн хариултаар дүгнэгдэж хаагдлаа. Энэ тухай багшид мэдэгдсэн — багш дахин нээж өгч болно.</p>`, "Ойлголоо", null, "stop");
    if (!document.hidden) return show();
    const once = () => { if (document.hidden) return; document.removeEventListener("visibilitychange", once); show(); };
    document.addEventListener("visibilitychange", once);
  };
  const violate = (type, label) => { if (finished) return; beatNow([{ type, detail: "Шалгалтын үеэр: " + label }]); submit(type, `⛔ Шалгалт хаагдлаа: ${label}.`); stopAlert(label); };
  const onVis = () => { if (document.hidden) violate("tab_switch", "өөр таб руу шилжсэн"); };
  const onBlur = () => setTimeout(() => { if (!finished && !document.hasFocus() && document.activeElement?.tagName !== "IFRAME") violate("tab_switch", "өөр цонх руу шилжсэн"); }, 0);
  const onCopy = (e) => { e.preventDefault(); violate("copy", "хуулах оролдлого хийсэн"); };
  const onMenu = (e) => e.preventDefault();
  const onLeave = (e) => { if (!finished) { e.preventDefault(); e.returnValue = ""; } };
  document.addEventListener("visibilitychange", onVis); addEventListener("blur", onBlur);
  el.addEventListener("copy", onCopy); el.addEventListener("cut", onCopy); el.addEventListener("contextmenu", onMenu);
  addEventListener("beforeunload", onLeave);
  const cleanup = () => {
    clearInterval(tick); clearInterval(beat);
    document.removeEventListener("visibilitychange", onVis); removeEventListener("blur", onBlur); removeEventListener("beforeunload", onLeave);
  };
  async function submit(terminate, msg) {
    if (finished) return; finished = true; cleanup();
    $$("[data-exam-submit]", el).forEach((b) => (b.disabled = true));
    let res;
    try { res = await api(`${base}/exam/submit`, { method: "POST", body: { attempt_id: att.id, answers: answers(), terminate } }); }
    catch (e) { res = null; toast(e.message, true); }
    try { sessionStorage.removeItem(key); } catch {}
    beatNow([], terminate ? "exam_terminated" : "closed");
    const a = res?.attempt;
    $$(".rb-quiz", el).forEach((f) => { f.classList.add("locked"); $$("input,select", f).forEach((x) => (x.disabled = true)); if (a?.results) { const ok = a.results[f.dataset.quiz]; f.classList.add(ok ? "is-ok" : "is-bad"); } if (res?.reveal?.[f.dataset.quiz]) quizReveal(f, res.reveal[f.dataset.quiz], a.results[f.dataset.quiz]); });
    $(".exam-bar [data-exam-submit]", el).remove();
    $(".exam-submit", el)?.remove();
    if (a?.results) forms.forEach((f, i) => { navBtns[i]?.classList.remove("done"); navBtns[i]?.classList.add(a.results[f.dataset.quiz] ? "ok" : "bad"); });
    const pass = a?.passed;
    $(".exam-body", el).insertAdjacentHTML("afterbegin", `<div class="exam-result ${a?.status === "terminated" ? "bad" : pass ? "ok" : ""}">
      <span class="exam-score">${a ? a.pct + "%" : "—"}</span><div><h3>${a?.status === "terminated" ? "Шалгалт хаагдлаа" : pass ? "🎉 Тэнцлээ!" : "Тэнцсэнгүй"}</h3>
      <p>${esc(msg || (a ? `${a.score}/${a.max} оноо` : ""))}${a?.status === "terminated" ? " Энэ тухай багшид мэдэгдсэн." : ""}</p></div>
      <button class="btn btn-gold" data-exam-close>Хаах</button></div>`);
    $(".exam-body", el).scrollTop = 0;
    if (pass) celebrate();
    $("[data-exam-close]", el).onclick = () => { el.remove(); document.documentElement.classList.remove("reader-open"); onClose?.(); };
  }
  el.addEventListener("click", (e) => {
    const g = e.target.closest("[data-goq]"); if (g) return goQ(+g.dataset.goq);
    if (!e.target.closest("[data-exam-submit]")) return;
    const todo = todoNums();
    if (todo.length && !confirm(`${todo.join(", ")}-р асуултад хариулаагүй байна (${todo.length}). Илгээх үү?`)) return goQ(todo[0] - 1);
    submit("", "");
  });
}

/* ---------- 3D ном ---------- */
const PDFJS = "https://cdnjs.cloudflare.com/ajax/libs/pdf.js/4.10.38/";
let pdfjs;
function pdfLib() {
  pdfjs ||= import(PDFJS + "pdf.min.mjs").then((m) => { m.GlobalWorkerOptions.workerSrc = PDFJS + "pdf.worker.min.mjs"; return m; });
  return pdfjs;
}
// Логоны өнгөнд зохицсон өнгөнүүд (templates.go-гийн brandHues-тэй ижил).
const BRAND_HUES = [222, 216, 228, 212, 232, 219, 225, 214];
function hueOf(s) { let h = 0; for (const c of String(s)) h = (h * 31 + c.charCodeAt(0)) >>> 0; return BRAND_HUES[h % BRAND_HUES.length]; }
function book3dHTML(url, title) {
  return `<div class="book3d" data-pdf="${esc(url)}" data-title="${esc(title || "")}" style="--hue:${hueOf(title || url)}" title="${esc(title || "Ном")}">
    <div class="bk"><div class="back"></div><div class="spine"></div><div class="pages"></div>
    <div class="cover"><span class="cover-title">${esc(title || "")}</span></div></div></div>`;
}
// Хавтсыг PDF-ийн эхний хуудаснаас зурна (харагдах үед л ачаална).
const coverIO = "IntersectionObserver" in window ? new IntersectionObserver((es) => es.forEach(async (e) => {
  if (!e.isIntersecting) return;
  coverIO.unobserve(e.target);
  try {
    const lib = await pdfLib();
    const doc = await lib.getDocument(e.target.dataset.pdf).promise;
    const pg = await doc.getPage(1);
    const vp0 = pg.getViewport({ scale: 1 }), scale = (240 / vp0.width);
    const vp = pg.getViewport({ scale });
    const cv = document.createElement("canvas"); cv.width = vp.width; cv.height = vp.height;
    await pg.render({ canvasContext: cv.getContext("2d"), viewport: vp }).promise;
    const cover = $(".cover", e.target); cover.prepend(cv); $(".cover-title", cover)?.remove();
    doc.destroy();
  } catch {}
}), { rootMargin: "200px" }) : null;
function hydrateBooks(root = document) { $$(".book3d:not([data-h])", root).forEach((b) => { b.dataset.h = 1; coverIO?.observe(b); }); hydrateDocs(root); }
document.addEventListener("click", (e) => {
  const b = e.target.closest("[data-book], .book3d");
  if (!b) return;
  e.preventDefault();
  // Хичээл доторх PDF: багш зөвшөөрөөгүй бол татах холбоосгүй нээнэ.
  const nodl = !!b.closest("[data-nodl]") || (!!b.closest("#lessonModal") && !b.closest("[data-dl]"));
  Flipbook.open(b.dataset.book || b.dataset.pdf, b.dataset.title, { nodl });
});

// FlipCore — 3D хуудас эргүүлдэг PDF харагч (томруулах, чирэх, тод дахин зурах). Попап (Flipbook.open) болон
// хичээл доторх (DocView) хувилбар хоёулаа үүнийг прототип болгоно: this.el дотор .fb-stage, .fb-nav байна.
const FlipCore = {
  el: null,
  async open(url, title, opts = {}) {
    this.close();
    const el = document.createElement("div");
    el.className = "flipbook";
    el.innerHTML = `<div class="fb-bar"><strong>${esc(title || "Баримт")}</strong>
        <div class="fb-zoom" role="group" aria-label="Томруулах"><button class="icon-btn" data-z="-" title="Жижигрүүлэх (−)" aria-label="Жижигрүүлэх">−</button><button class="fb-zv" data-z="0" title="Анхны хэмжээ (0)">100%</button><button class="icon-btn" data-z="+" title="Томруулах (+)" aria-label="Томруулах">+</button></div>
        ${opts.nodl ? `<span class="chip">🔒 Зөвхөн унших</span>` : `<a class="btn btn-ghost btn-sm" href="${esc(url)}" target="_blank" rel="noopener">⬇</a>`}<button class="icon-btn" data-fb-close aria-label="Хаах">✕</button></div>
      <div class="fb-stage"><div class="loader"></div></div>
      <div class="fb-nav"><button class="btn btn-glass btn-sm" data-fb-prev>←</button><input type="range" min="0" value="0"><span class="fb-count"></span><button class="btn btn-glass btn-sm" data-fb-next>→</button><span class="fb-hint">Давхар дарах, Ctrl + дугуй эсвэл хоёр хуруугаар томруулна</span></div>`;
    document.body.append(el);
    this.el = el; this.zoom = 1; this.pan = { x: 0, y: 0 };
    requestAnimationFrame(() => el.classList.add("open"));
    el.querySelector("[data-fb-close]").onclick = () => this.close();
    el.addEventListener("click", (e) => { const z = e.target.closest("[data-z]"); if (z) this.setZoom(z.dataset.z === "+" ? this.zoom * 1.25 : z.dataset.z === "-" ? this.zoom / 1.25 : 1); });
    this.key = (e) => {
      if (e.key === "ArrowRight") this.go(this.cur + 1);
      else if (e.key === "ArrowLeft") this.go(this.cur - 1);
      else if (e.key === "+" || e.key === "=") this.setZoom(this.zoom * 1.25);
      else if (e.key === "-") this.setZoom(this.zoom / 1.25);
      else if (e.key === "0") this.setZoom(1);
      else if (e.key === "Escape") this.close();
    };
    addEventListener("keydown", this.key);
    try {
      const lib = await pdfLib();
      this.doc = await lib.getDocument(url).promise;
      const first = await this.doc.getPage(1);
      this.ratio = first.getViewport({ scale: 1 }).height / first.getViewport({ scale: 1 }).width;
      this.build();
    } catch (err) {
      $(".fb-stage", el).innerHTML = opts.nodl ? `<p class="muted">Баримтыг нээж чадсангүй.</p>` : `<p class="muted">Баримтыг нээж чадсангүй. <a href="${esc(url)}" target="_blank" rel="noopener">Татаж үзэх</a></p>`;
    }
  },
  build() {
    const el = this.el, stage = $(".fb-stage", el), n = this.doc.numPages;
    const sw = stage.clientWidth * 0.92, sh = stage.clientHeight * 0.9;
    const keepPage = this.leafEls ? (this.single ? this.cur + 1 : Math.max(1, this.cur * 2)) : 1; // дахин барихад хуудсаа хадгална
    this.single = this.forceSingle || sw < 760;
    const cols = this.single ? 1 : 2;
    let pw = Math.min(sw / cols, sh / this.ratio), ph = pw * this.ratio;
    // Хуудас бүр нэг "навч"; хоёр талтай горимд навч нэг бүр 2 хуудас (урд/ард).
    const per = this.single ? 1 : 2;
    this.leaves = Math.ceil(n / per);
    stage.innerHTML = `<div class="fb-pan"><div class="fb-book ${this.single ? "single" : ""}" style="width:${pw * cols}px;height:${ph}px"></div></div>`;
    const book = $(".fb-book", stage);
    this.pw = pw; this.ph = ph; this.book = book; this.rq = new Map(); this.busy = new Set(); this.again = new Set();
    for (let i = 0; i < this.leaves; i++) {
      const f = i * per + 1, b = per === 2 ? f + 1 : 0;
      const leaf = document.createElement("div");
      leaf.className = "leaf";
      leaf.innerHTML = `<div class="face front" data-p="${f}"><div class="page-loading">${f}</div><span class="pnum">${f}</span></div>` +
        (b && b <= n ? `<div class="face back" data-p="${b}"><div class="page-loading">${b}</div><span class="pnum">${b}</span></div>` : `<div class="face back"></div>`);
      leaf.addEventListener("click", (e) => {
        e.stopPropagation();
        if (this.zoom > 1 || this.dragged) return; // томруулсан эсвэл чирсэн бол хуудас эргүүлэхгүй
        if (this.clickT) { clearTimeout(this.clickT); this.clickT = 0; return; } // давхар дарах → томруулна
        const to = leaf.classList.contains("flipped") ? i : i + 1;
        this.clickT = setTimeout(() => { this.clickT = 0; this.go(to); }, 240);
      });
      book.append(leaf);
    }
    this.leafEls = $$(".leaf", book);
    const range = $("input[type=range]", el);
    range.max = this.leaves; range.oninput = () => this.go(+range.value, true);
    $("[data-fb-prev]", el).onclick = () => this.go(this.cur - 1);
    $("[data-fb-next]", el).onclick = () => this.go(this.cur + 1);
    if (!stage.dataset.g) { stage.dataset.g = 1; this.bindGestures(stage); }
    this.cur = 0;
    this.go(this.single ? keepPage - 1 : Math.floor(keepPage / 2), true);
    this.applyZoom();
  },
  // Томруулах: Ctrl/⌘ + дугуй (trackpad чимхэлт), давхар дарах, хоёр хуруугаар чимхэх, товч; томруулсан үед чирж/гүйлгэж харна.
  // Томруулж дуусахад харагдаж буй хуудсыг өндөр нягтралтайгаар дахин зурна (бичиг бүдгэрэхгүй).
  setZoom(z, cx, cy) {
    if (!this.el) return;
    const old = this.zoom; z = Math.min(4, Math.max(1, z));
    if (cx != null) { // хулгана/хурууны цэг дээр төвлөрч томруулна
      const r = $(".fb-stage", this.el).getBoundingClientRect(), ox = cx - r.left - r.width / 2, oy = cy - r.top - r.height / 2;
      this.pan.x = ox - (ox - this.pan.x) * (z / old); this.pan.y = oy - (oy - this.pan.y) * (z / old);
    }
    if (z <= 1) this.pan = { x: 0, y: 0 };
    this.zoom = z; this.applyZoom();
    clearTimeout(this.zt); this.zt = setTimeout(() => this.sharpen(), 250);
  },
  applyZoom() {
    const pan = this.el && $(".fb-pan", this.el); if (!pan) return;
    const st = $(".fb-stage", this.el), bk = this.book; // хуудсыг дэлгэцнээс бүрэн гаргахгүй
    if (st && bk) {
      const mx = Math.max(0, (bk.offsetWidth * this.zoom - st.clientWidth) / 2 + 40), my = Math.max(0, (bk.offsetHeight * this.zoom - st.clientHeight) / 2 + 40);
      this.pan.x = Math.max(-mx, Math.min(mx, this.pan.x)); this.pan.y = Math.max(-my, Math.min(my, this.pan.y));
    }
    pan.style.transform = `translate(${this.pan.x}px, ${this.pan.y}px) scale(${this.zoom})`;
    this.el.classList.toggle("zoomed", this.zoom > 1);
    const zv = $(".fb-zv", this.el); if (zv) zv.textContent = Math.round(this.zoom * 100) + "%";
  },
  bindGestures(stage) {
    const pts = new Map();
    let start = null, pinch = null, gz = 1;
    stage.addEventListener("wheel", (e) => {
      if (e.ctrlKey || e.metaKey) { e.preventDefault(); this.setZoom(this.zoom * Math.exp(-e.deltaY / 300), e.clientX, e.clientY); }
      else if (this.zoom > 1) { e.preventDefault(); this.pan.x -= e.deltaX; this.pan.y -= e.deltaY; this.applyZoom(); }
    }, { passive: false });
    // Safari (macOS) trackpad-ийн чимхэлт
    stage.addEventListener("gesturestart", (e) => { e.preventDefault(); gz = this.zoom; });
    stage.addEventListener("gesturechange", (e) => { e.preventDefault(); this.setZoom(gz * e.scale, e.clientX, e.clientY); });
    stage.addEventListener("dblclick", (e) => {
      e.preventDefault(); clearTimeout(this.clickT); this.clickT = 0;
      this.setZoom(this.zoom > 1 ? 1 : 2, e.clientX, e.clientY);
    });
    stage.addEventListener("pointerdown", (e) => {
      pts.set(e.pointerId, { x: e.clientX, y: e.clientY });
      if (pts.size === 2) { const [a, b] = [...pts.values()]; pinch = { d: Math.hypot(a.x - b.x, a.y - b.y) || 1, z: this.zoom }; start = null; return; }
      start = { x: e.clientX, y: e.clientY, px: this.pan.x, py: this.pan.y, moved: false };
      this.dragged = false;
    });
    stage.addEventListener("pointermove", (e) => {
      if (!pts.has(e.pointerId)) return;
      pts.set(e.pointerId, { x: e.clientX, y: e.clientY });
      if (pinch && pts.size === 2) {
        const [a, b] = [...pts.values()];
        this.dragged = true;
        this.setZoom(pinch.z * Math.hypot(a.x - b.x, a.y - b.y) / pinch.d, (a.x + b.x) / 2, (a.y + b.y) / 2);
        return;
      }
      if (!start) return;
      const dx = e.clientX - start.x, dy = e.clientY - start.y;
      if (Math.abs(dx) + Math.abs(dy) > 6) { start.moved = true; this.dragged = true; }
      if (this.zoom > 1 && start.moved) { this.pan.x = start.px + dx; this.pan.y = start.py + dy; this.applyZoom(); }
    });
    const end = (e) => {
      pts.delete(e.pointerId);
      if (pts.size < 2) pinch = null;
      if (!start) return;
      const dx = e.clientX - start.x;
      if (this.zoom <= 1 && start.moved && Math.abs(dx) > 40) this.go(this.cur + (dx < 0 ? 1 : -1)); // шудрах
      start = null;
    };
    stage.addEventListener("pointerup", end); stage.addEventListener("pointercancel", end);
  },
  // Харагдаж буй хуудсууд (нэг эсвэл хоёр) — томруулсан хэмжээнд тохирох нягтралаар.
  sharpen() {
    if (!this.el || !this.doc) return;
    const k = this.cur, n = this.doc.numPages;
    (this.single ? [k + 1] : [k * 2, k * 2 + 1]).filter((p) => p >= 1 && p <= n).forEach((p) => this.render(p, true));
  },
  quality(sharp) {
    const dpr = Math.min(2, devicePixelRatio || 1);
    return sharp ? Math.min(dpr * Math.max(1, this.zoom), 4096 / this.pw) : dpr; // зураг хэт томрохгүй (≤4096px)
  },
  go(k, instant) {
    if (!this.leafEls) return;
    k = Math.max(0, Math.min(this.leaves, k));
    const L = this.leafEls;
    L.forEach((leaf, i) => {
      const flip = i < k;
      if (instant) leaf.style.transition = "none";
      if (flip !== leaf.classList.contains("flipped")) {
        leaf.classList.add("turning");
        leaf.style.zIndex = 1000;
        setTimeout(() => leaf.classList.remove("turning"), 1000);
      }
      leaf.classList.toggle("flipped", flip);
      // Эргэлт дууссаны дараа давхаргын дарааллыг тогтооно.
      setTimeout(() => { leaf.style.zIndex = flip ? i + 1 : L.length - i; }, instant ? 0 : 500);
      if (instant) requestAnimationFrame(() => (leaf.style.transition = ""));
    });
    this.cur = k;
    if (!this.single) this.book.classList.toggle("closed-front", k === 0), this.book.classList.toggle("closed-back", k === this.leaves);
    $("input[type=range]", this.el).value = k;
    const per = this.single ? 1 : 2, n = this.doc.numPages;
    const from = this.single ? k + 1 : Math.max(1, k * 2), to = Math.min(n, this.single ? k + 1 : k * 2 + 1);
    $(".fb-count", this.el).textContent = `${from}${to !== from ? "–" + to : ""} / ${n}`;
    // Ойролцоох хуудсуудыг л зурна (том PDF ч хурдан); томруулсан бол харагдаж буйг тодруулна.
    for (let p = Math.max(1, k * per - 3); p <= Math.min(n, k * per + 5); p++) this.render(p);
    if (this.zoom > 1) { clearTimeout(this.zt); this.zt = setTimeout(() => this.sharpen(), 250); }
  },
  async render(p, sharp) {
    const need = this.quality(sharp);
    if ((this.rq.get(p) || 0) >= need - 0.01) return;
    if (this.busy.has(p)) { if (sharp) this.again.add(p); return; } // зурж байгаа бол дараа нь дахин шалгана
    this.busy.add(p);
    try {
      const face = this.el && $(`.face[data-p="${p}"]`, this.el);
      if (!face || !this.doc) return;
      const pg = await this.doc.getPage(p);
      const base = pg.getViewport({ scale: 1 });
      const vp = pg.getViewport({ scale: (this.pw / base.width) * need });
      const cv = document.createElement("canvas"); cv.width = Math.round(vp.width); cv.height = Math.round(vp.height);
      await pg.render({ canvasContext: cv.getContext("2d"), viewport: vp }).promise;
      if (!this.el) return;
      const old = face.querySelector("canvas, .page-loading");
      old ? old.replaceWith(cv) : face.prepend(cv);
      this.rq.set(p, need);
    } catch {} finally { this.busy.delete(p); if (this.again.delete(p)) this.render(p, true); }
  },
  close() {
    if (!this.el) return;
    const el = this.el; this.el = null; this.leafEls = null;
    removeEventListener("keydown", this.key);
    clearTimeout(this.zt); clearTimeout(this.clickT); this.clickT = 0;
    el.classList.remove("open");
    setTimeout(() => el.remove(), 400);
    this.doc?.destroy(); this.doc = null;
  },
};

const Flipbook = Object.create(FlipCore); // попап уншигч (номын сан, багшийн файл)

/* ---------- Хичээлийн баримтыг хичээл дотор шууд, хамгаалалттай үзүүлэх ----------
   Попап цонхгүй — хичээлийн агуулга дотор:
   • PDF: хэвтээ хуудас (слайд) → слайд тоглуулагч; босоо → ном шиг эргэдэг хуудас (3D, томруулна).
   • PowerPoint (.pptx) → слайд тоглуулагч («▶ Тоглуулах» = бүтэн дэлгэцээр презентаци).
   • Word (.docx) → хуудаслагдсан баримт (томруулна).
   • Excel (.xlsx/.xls/.ods/.csv) → хүснэгт: хуудаснууд, томьёоны мөр; томьёо бодогдоно, утгыг өөрчилж туршина.
   Хамгаалалт: холбоос нь хугацаатай тасалбар (хичээл дотроос л уншигдана, шинэ таб/татахад 403), хуулах,
   баруун товч, хэвлэх хаалттай, үзэж буй хүний нэрээр усан тэмдэг. Багш тухайн файлд зөвшөөрсөн үед л татах товч. */
const DOC_LIBS = {
  xlsx: "https://cdnjs.cloudflare.com/ajax/libs/xlsx/0.18.5/xlsx.core.min.js", // xlsx/ods/csv (хөнгөн)
  xlsxFull: "https://cdnjs.cloudflare.com/ajax/libs/xlsx/0.18.5/xlsx.full.min.js", // хуучин .xls (кодын хуудастай)
  jszip: "https://cdnjs.cloudflare.com/ajax/libs/jszip/3.10.1/jszip.min.js",
  docx: "https://cdn.jsdelivr.net/npm/docx-preview@0.3.5/dist/docx-preview.min.js",
  pptx: "https://cdn.jsdelivr.net/npm/pptx-preview@1.0.7/dist/pptx-preview.umd.js",
  formula: "https://cdn.jsdelivr.net/npm/hot-formula-parser@4.0.0/dist/formula-parser.min.js",
};
const scriptP = new Map();
function loadScript(src) {
  if (!scriptP.has(src)) scriptP.set(src, new Promise((res, rej) => {
    const s = document.createElement("script"); s.src = src; s.async = true;
    s.onload = res;
    s.onerror = () => { scriptP.delete(src); s.remove(); rej(new Error("Харагч ачаалагдсангүй — интернэт холболтоо шалгана уу")); };
    document.head.append(s);
  }));
  return scriptP.get(src);
}
const DOC_KIND = { ".pdf": "pdf", ".pptx": "pptx", ".docx": "docx", ".xlsx": "sheet", ".xls": "sheet", ".ods": "sheet", ".csv": "sheet", ".ppt": "legacy", ".doc": "legacy", ".odt": "legacy", ".odp": "legacy", ".rtf": "legacy" };
const docKind = (url) => DOC_KIND[extOf(url)] || "";
const DOC_LABEL = { pdf: "PDF баримт", pptx: "Презентаци", docx: "Word баримт", sheet: "Excel хүснэгт", legacy: "Баримт" };
const DOC_ICO = { pdf: "book", pptx: "slides", docx: "doc", sheet: "sheet", legacy: "doc" };
const docViewHTML = (url, title) => `<div class="doc-view" data-doc="${esc(url)}" data-kind="${docKind(url)}" data-title="${esc(title || "")}"><div class="dv-skel"><span class="loader"></span></div></div>`;
// Усан тэмдэг: үзэж буй хүний нэр + ID-ийн сүүл (дэлгэцийн зураг аваад тараахаас сэргийлнэ).
const wmText = () => { const u = Auth.user; return u ? `${u.display_name || u.username || ""} · #${String(u.id || "").slice(-6)}` : "surgalt.mn"; };
const wmBG = (t) => `url("data:image/svg+xml,${encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" width="360" height="210"><text x="180" y="105" text-anchor="middle" transform="rotate(-24 180 105)" font-family="Manrope,Arial,sans-serif" font-size="15" font-weight="700" fill="rgba(120,134,170,0.2)">${t.replace(/[<>&"']/g, "")}</text></svg>`)}")`;
const docIO = "IntersectionObserver" in window ? new IntersectionObserver((es) => es.forEach((e) => { if (e.isIntersecting) { docIO.unobserve(e.target); DocView.mount(e.target); } }), { rootMargin: "400px" }) : null;
function hydrateDocs(root = document) { $$(".doc-view:not([data-h])", root).forEach((d) => { d.dataset.h = 1; docIO ? docIO.observe(d) : DocView.mount(d); }); }
const zoomGroupHTML = () => `<div class="fb-zoom" role="group" aria-label="Томруулах"><button class="icon-btn" data-z="-" title="Жижигрүүлэх (−)" aria-label="Жижигрүүлэх">−</button><button class="fb-zv" data-z="0" title="Анхны хэмжээ (0)">100%</button><button class="icon-btn" data-z="+" title="Томруулах (+)" aria-label="Томруулах">+</button></div>`;

const DocView = {
  async mount(el) {
    const url = el.dataset.doc, kind = el.dataset.kind || "legacy", title = el.dataset.title || DOC_LABEL[kind];
    if (!url || el.dataset.m) return;
    el.dataset.m = 1;
    el.removeAttribute("data-doc"); // холбоосыг DOM-д ил үлдээхгүй
    const canDl = !!el.closest("[data-dl]"); // зөвхөн багш энэ файлд зөвшөөрсөн бол
    el.tabIndex = 0; el.classList.add("dv-" + kind);
    el.innerHTML = `<div class="dv-bar"><span class="dv-ico" aria-hidden="true">${icon(DOC_ICO[kind], 18)}</span>
        <span class="dv-title"><b>${esc(title)}</b><small>${DOC_LABEL[kind]}${canDl ? "" : " · зөвхөн үзнэ"}</small></span><span class="dv-tools"></span>
        ${canDl ? `<button class="icon-btn" data-dv-dl title="Татах" aria-label="Татах">${icon("down", 18)}</button>` : `<span class="dv-lock" title="Зөвхөн хичээл дотор үзнэ — татах боломжгүй">${icon("lock", 16)}</span>`}
        <button class="icon-btn dv-fs" data-dv-fs title="Бүтэн дэлгэц (F)" aria-label="Бүтэн дэлгэц">${icon("expand", 18)}</button></div>
      <div class="dv-body"><div class="dv-load"><span class="loader"></span><small>${esc(DOC_LABEL[kind])} ачаалж байна…</small></div></div><div class="dv-wm" aria-hidden="true"></div>`;
    $(".dv-wm", el).style.backgroundImage = wmBG(wmText());
    // Хамгаалалт: хуулах, сонгох, чирэх, баруун товч (оролтын талбараас бусад). copy үйл явдал хичээлийн
    // хяналт руу дамжина (зөрчил гэж бүртгэгдэнэ).
    const guard = (e) => { if (!e.target.closest?.("input, textarea")) e.preventDefault(); };
    ["contextmenu", "dragstart", "selectstart", "copy", "cut"].forEach((t) => el.addEventListener(t, guard));
    el.addEventListener("click", (e) => {
      if (e.target.closest("[data-dv-fs]")) DocView.fullscreen(el);
      else if (e.target.closest("[data-dv-dl]")) DocView.download(url, title);
    });
    el.addEventListener("keydown", (e) => {
      if (e.target.closest("input, textarea")) return;
      if ((e.key === "f" || e.key === "F") && !e.ctrlKey && !e.metaKey) { e.preventDefault(); return DocView.fullscreen(el); }
      if (e.key === "Escape" && el.classList.contains("dv-max")) { e.preventDefault(); return DocView.unmax(el); }
      el._dv?.key?.(e);
    });
    const body = $(".dv-body", el);
    if (kind === "legacy") {
      body.innerHTML = `<div class="dv-msg">${icon("doc", 28)}<b>Энэ хуучин форматыг шууд үзүүлэх боломжгүй</b><span>Багш файлаа PDF, .pptx эсвэл .docx болгож оруулбал хичээл дотор шууд харагдана.</span></div>`;
      return;
    }
    try {
      const res = await fetch(url, { cache: "no-store" });
      if (!res.ok) throw new Error((await res.json().catch(() => null))?.error || "Баримтыг нээж чадсангүй");
      const buf = await res.arrayBuffer();
      el._dv = (await DOC_VIEWS[kind](el, body, buf, url)) || {};
    } catch (err) {
      body.innerHTML = `<div class="dv-msg">${icon("doc", 28)}<b>Баримтыг нээж чадсангүй</b><span>${esc(err.message || "")}</span></div>`;
    }
  },
  // Бүтэн дэлгэц: тухайн харагч өөрөө (шинэ цонх биш). iPhone гэх мэт дэмжихгүй бол дэлгэц дүүргэнэ.
  fullscreen(el) {
    if (document.fullscreenElement === el) return document.exitFullscreen?.();
    if (el.classList.contains("dv-max")) return DocView.unmax(el);
    const req = el.requestFullscreen || el.webkitRequestFullscreen;
    // Fullscreen API байхгүй (iPhone) эсвэл татгалзвал: дэлгэцийг дүүргэнэ. Эцэг элемент transform/overflow-той
    // байж болох тул түр body руу зөөж, гарахад байранд нь буцаана.
    const fallback = () => {
      el._ph = document.createComment("doc-view"); el.before(el._ph); document.body.append(el);
      el.classList.add("dv-max", "dv-full"); document.documentElement.classList.add("dv-locked"); el.focus(); el._dv?.layout?.();
    };
    if (!req) return fallback();
    try { const p = req.call(el); p?.catch?.(fallback); } catch { fallback(); }
  },
  unmax(el) {
    el.classList.remove("dv-max", "dv-full"); document.documentElement.classList.remove("dv-locked");
    if (el._ph) { el._ph.replaceWith(el); el._ph = null; }
    el._dv?.layout?.();
  },
  async download(url, title) {
    try {
      const res = await fetch(url); if (!res.ok) throw new Error("Татаж чадсангүй");
      const a = document.createElement("a"); a.href = URL.createObjectURL(await res.blob());
      a.download = (title || "file").replace(/[\\/:*?"<>|]+/g, "_") + extOf(url); a.click();
      setTimeout(() => URL.revokeObjectURL(a.href), 3000);
    } catch (e) { toast(e.message, true); }
  },
};
document.addEventListener("fullscreenchange", () => {
  $$(".doc-view.dv-full:not(.dv-max)").forEach((d) => { if (document.fullscreenElement !== d) { d.classList.remove("dv-full"); d._dv?.layout?.(); } });
  const fe = document.fullscreenElement;
  if (fe?.classList?.contains("doc-view")) { fe.classList.add("dv-full"); fe.focus(); fe._dv?.layout?.(); }
});

// Слайд тоглуулагч (PDF-ийн хэвтээ хуудас, PowerPoint): нэг слайд, сум/товч/шудрах, ▶ бүтэн дэлгэц.
function slidePlayer(el, body, { count, ratio, draw, prefetch }) {
  el.classList.add("dv-slides");
  $(".dv-tools", el).innerHTML = `<span class="sl-count" aria-live="polite"></span><button class="btn btn-gold btn-sm" data-sl-play>▶ Тоглуулах</button>`;
  body.innerHTML = `<div class="sl-stage" style="--r:${ratio}"><div class="sl-slide"></div>
      <button class="sl-nav sl-prev" aria-label="Өмнөх слайд">‹</button><button class="sl-nav sl-next" aria-label="Дараагийн слайд">›</button></div>
    <div class="sl-prog"><i></i></div>`;
  const stage = $(".sl-stage", body), box = $(".sl-slide", body);
  let cur = -1, tok = 0, lastW = 0;
  const show = async (i, dir = 0) => {
    i = Math.max(0, Math.min(count - 1, i));
    const my = ++tok, changed = i !== cur; cur = i;
    $(".sl-count", el).textContent = `${i + 1} / ${count}`;
    $(".sl-prog i", body).style.width = ((i + 1) / count) * 100 + "%";
    $(".sl-prev", body).disabled = i === 0; $(".sl-next", body).disabled = i === count - 1;
    const w = lastW = box.clientWidth || stage.clientWidth;
    const node = await draw(i, w);
    if (my !== tok) return;
    if (box.firstChild !== node) box.replaceChildren(node);
    if (changed) fly(node, dir);
    if (prefetch && i + 1 < count) setTimeout(() => { if (my === tok) draw(i + 1, w).catch(() => {}); }, 250);
  };
  // Тоглуулах үед: слайд шилжих чиглэлээсээ гулсаж орж ирээд, дээрх гарчиг, текст, зураг нэг нэгээрээ
  // доороос нисэж гарч ирнэ (PowerPoint-ийн "Fly In" шиг).
  const fly = (node, dir) => {
    if (reduce) return;
    if (dir) { box.classList.remove("fly-l", "fly-r"); void box.offsetWidth; box.classList.add(dir < 0 ? "fly-l" : "fly-r"); }
    $$(".slide-wrapper > *", node).forEach((sh, k) => {
      sh.classList.remove("fly-in"); sh.style.animationDelay = 140 + k * 120 + "ms"; void sh.offsetWidth; sh.classList.add("fly-in");
    });
  };
  const step = (d) => show(cur + d, d);
  $(".sl-prev", body).onclick = (e) => { e.stopPropagation(); step(-1); };
  $(".sl-next", body).onclick = (e) => { e.stopPropagation(); step(1); };
  $("[data-sl-play]", el).onclick = () => { DocView.fullscreen(el); };
  let x0 = null, moved = false;
  stage.addEventListener("pointerdown", (e) => { x0 = e.clientX; moved = false; });
  stage.addEventListener("pointermove", (e) => { if (x0 != null && Math.abs(e.clientX - x0) > 8) moved = true; });
  stage.addEventListener("pointerup", (e) => {
    if (x0 == null) return;
    const dx = e.clientX - x0; x0 = null;
    if (e.target.closest(".sl-nav")) return;
    if (moved && Math.abs(dx) > 40) return step(dx < 0 ? 1 : -1); // шудрах
    if (!moved) { const r = stage.getBoundingClientRect(); step(e.clientX < r.left + r.width * 0.3 ? -1 : 1); } // дарах: зүүн талаар өмнөх
  });
  let rt = 0;
  const ro = "ResizeObserver" in window ? new ResizeObserver(() => { clearTimeout(rt); rt = setTimeout(() => { if (Math.abs((box.clientWidth || 0) - lastW) > 4) show(cur); }, 120); }) : null;
  ro?.observe(stage);
  show(0);
  return {
    key(e) {
      if (["ArrowRight", "PageDown", " ", "Enter"].includes(e.key)) { e.preventDefault(); step(1); }
      else if (["ArrowLeft", "PageUp", "Backspace"].includes(e.key)) { e.preventDefault(); step(-1); }
      else if (e.key === "Home") show(0); else if (e.key === "End") show(count - 1);
    },
    layout() { setTimeout(() => show(cur), 60); },
  };
}

// Босоо PDF → хичээл доторх 3D ном (FlipCore-ийг ашиглана: томруулах, чирэх, тод дахин зурах).
function flipInline(el, body, doc, ratio, wide) {
  el.classList.add("dv-book"); if (wide) el.classList.add("dv-wide");
  $(".dv-tools", el).innerHTML = zoomGroupHTML() + (wide ? `<button class="btn btn-gold btn-sm" data-sl-play>▶ Тоглуулах</button>` : "");
  body.innerHTML = `<div class="fb-stage"></div><div class="fb-nav"><button class="btn btn-glass btn-sm" data-fb-prev aria-label="Өмнөх хуудас">←</button><input type="range" min="0" value="0" aria-label="Хуудас"><span class="fb-count"></span><button class="btn btn-glass btn-sm" data-fb-next aria-label="Дараагийн хуудас">→</button></div>`;
  const fb = Object.create(FlipCore);
  Object.assign(fb, { el, doc, ratio, zoom: 1, pan: { x: 0, y: 0 }, forceSingle: !!wide });
  fb.build();
  el.addEventListener("click", (e) => {
    const z = e.target.closest("[data-z]"); if (z) fb.setZoom(z.dataset.z === "+" ? fb.zoom * 1.25 : z.dataset.z === "-" ? fb.zoom / 1.25 : 1);
    if (e.target.closest("[data-sl-play]")) DocView.fullscreen(el);
  });
  let rt = 0;
  return {
    key(e) {
      if (["ArrowRight", "PageDown", " "].includes(e.key)) { e.preventDefault(); fb.go(fb.cur + 1); }
      else if (["ArrowLeft", "PageUp"].includes(e.key)) { e.preventDefault(); fb.go(fb.cur - 1); }
      else if (e.key === "+" || e.key === "=") fb.setZoom(fb.zoom * 1.25);
      else if (e.key === "-") fb.setZoom(fb.zoom / 1.25);
      else if (e.key === "0") fb.setZoom(1);
    },
    layout() { clearTimeout(rt); rt = setTimeout(() => { fb.zoom = 1; fb.pan = { x: 0, y: 0 }; fb.build(); }, 150); },
  };
}

const DOC_VIEWS = {
  // PDF: ном шиг (3D хуудас эргүүлнэ, томруулна). Хэвтээ хуудастай (слайд) бол нэг хуудсаар эргэдэг ном
  // бөгөөд «▶ Тоглуулах» нь бүтэн дэлгэцээр презентаци болгон тоглуулна.
  async pdf(el, body, buf) {
    const doc = await (await pdfLib()).getDocument({ data: buf }).promise;
    const v = (await doc.getPage(1)).getViewport({ scale: 1 }), ratio = v.height / v.width;
    return flipInline(el, body, doc, ratio, ratio < 0.9);
  },
  async pptx(el, body, buf) {
    await loadScript(DOC_LIBS.pptx);
    const BASE = 960, host = document.createElement("div");
    host.className = "pp-host"; body.append(host);
    let pv = window.pptxPreview.init(host, { width: BASE, height: 540, mode: "slide" });
    await pv.preview(buf);
    const ratio = pv.pptx?.width ? pv.pptx.height / pv.pptx.width : 0.5625;
    if (Math.abs(ratio - 0.5625) > 0.01) { // 4:3 гэх мэт: харьцааг нь хадгалж дахин бэлтгэнэ
      pv.destroy?.(); host.innerHTML = "";
      pv = window.pptxPreview.init(host, { width: BASE, height: Math.round(BASE * ratio), mode: "slide" });
      await pv.preview(buf);
    }
    host.style.width = BASE + "px"; host.style.height = Math.round(BASE * ratio) + "px";
    return slidePlayer(el, body, { count: pv.slideCount || 1, ratio, async draw(i, w) {
      if (pv.currentIndex !== i) { pv.renderSingleSlide(i); pv.currentIndex = i; }
      host.style.transform = `scale(${w / BASE})`;
      return host;
    } });
  },
  async docx(el, body, buf) {
    await loadScript(DOC_LIBS.jszip); await loadScript(DOC_LIBS.docx);
    $(".dv-tools", el).innerHTML = `<span class="dx-pages muted small"></span>` + zoomGroupHTML();
    body.innerHTML = `<div class="dx-scroll"><div class="dx-doc"></div></div>`;
    const sc = $(".dx-scroll", body), doc = $(".dx-doc", body);
    await window.docx.renderAsync(buf, doc, null, { className: "docx", inWrapper: true, breakPages: true, ignoreLastRenderedPageBreak: false, useBase64URL: true, renderHeaders: true, renderFooters: true, renderFootnotes: true });
    const pages = $$("section.docx", doc), PW = (pages[0]?.offsetWidth || 816) + 40;
    $(".dx-pages", el).textContent = pages.length + " хуудас";
    let z = 1;
    const apply = () => { const fit = Math.min(1, (sc.clientWidth - 4) / PW); doc.style.zoom = String(fit * z); $(".fb-zv", el).textContent = Math.round(z * 100) + "%"; };
    const setZ = (v) => { z = Math.min(3, Math.max(0.5, v)); apply(); };
    el.addEventListener("click", (e) => { const b = e.target.closest("[data-z]"); if (b) setZ(b.dataset.z === "+" ? z * 1.25 : b.dataset.z === "-" ? z / 1.25 : 1); });
    sc.addEventListener("wheel", (e) => { if (e.ctrlKey || e.metaKey) { e.preventDefault(); setZ(z * Math.exp(-e.deltaY / 300)); } }, { passive: false });
    if ("ResizeObserver" in window) new ResizeObserver(apply).observe(sc);
    apply();
    return {
      key(e) {
        if (e.key === "+" || e.key === "=") setZ(z * 1.25); else if (e.key === "-") setZ(z / 1.25); else if (e.key === "0") setZ(1);
        else if (e.key === "PageDown" || e.key === " ") { e.preventDefault(); sc.scrollBy({ top: sc.clientHeight * 0.9, behavior: "smooth" }); }
        else if (e.key === "PageUp") { e.preventDefault(); sc.scrollBy({ top: -sc.clientHeight * 0.9, behavior: "smooth" }); }
      },
      layout: apply,
    };
  },
  async sheet(el, body, buf, url) {
    await loadScript(extOf(url) === ".xls" ? DOC_LIBS.xlsxFull : DOC_LIBS.xlsx);
    // sheetStubs: томьёотой ч хадгалсан утгагүй нүд (жишээ нь программаар үүсгэсэн файл) алга болохгүй.
    const wb = XLSX.read(buf, { type: "array", cellFormula: true, cellNF: true, cellDates: true, sheetStubs: true });
    let FP = null; try { await loadScript(DOC_LIBS.formula); FP = window.formulaParser; } catch {}
    return sheetView(el, body, wb, FP);
  },
};

// Excel: хуудсууд, томьёоны мөр, нүдийг сонгох; томьёог бодно (дэмжээгүй функц бол файлд хадгалсан утга),
// оролтын нүдийг давхар дарж өөрчилбөл томьёо шууд дахин бодогдоно (хадгалагдахгүй — дадлага).
function sheetView(el, body, wb, FP) {
  const MAXR = 1000, MAXC = 50, colName = XLSX.utils.encode_col;
  $(".dv-tools", el).innerHTML = `<button class="btn btn-ghost btn-sm" data-xs-reset hidden>↺ Анхны утга</button>`;
  body.innerHTML = `<div class="xs-fx"><span class="xs-ref">A1</span><span class="xs-fn">ƒx</span><span class="xs-f"></span></div>
    <div class="xs-grid"></div><div class="xs-foot"><div class="xs-tabs" role="tablist">${wb.SheetNames.map((n, i) => `<button role="tab" data-sheet="${i}">${esc(n)}</button>`).join("")}</div>
    <span class="xs-note"></span></div>`;
  const grid = $(".xs-grid", body), parser = FP ? new FP.Parser() : null;
  let S = null, memo = new Map(), stack = new Set(), sel = "0,0";
  const valueAt = (r, c) => {
    const k = r + "," + c, cell = S.cells.get(k);
    if (!cell) return null;
    if (!cell.f) return cell.v ?? null;
    if (memo.has(k)) return memo.get(k);
    if (stack.has(k)) return "#CYCLE!";
    stack.add(k);
    let v = cell.v0 ?? "";
    if (parser && !cell.f.includes("!")) { const { result, error } = parser.parse(cell.f); v = error ? (cell.v0 ?? error) : result; } // өөр хуудас/дэмжээгүй → хадгалсан утга
    stack.delete(k); memo.set(k, v);
    return v;
  };
  if (parser) {
    parser.on("callCellValue", (cc, done) => done(valueAt(cc.row.index, cc.column.index)));
    parser.on("callRangeValue", (a, b, done) => {
      const out = [];
      for (let r = a.row.index; r <= b.row.index; r++) { const row = []; for (let c = a.column.index; c <= b.column.index; c++) row.push(valueAt(r, c)); out.push(row); }
      done(out);
    });
  }
  const recalc = () => { memo = new Map(); for (const [k, cell] of S.cells) if (cell.f) { const [r, c] = k.split(",").map(Number); cell.v = valueAt(r, c); } };
  const fmt = (cell, v) => {
    if (v == null || v === "") return "";
    if (v instanceof Date) return v.toLocaleDateString();
    if (typeof v === "boolean") return v ? "TRUE" : "FALSE";
    if (typeof v === "number") { if (cell?.z && cell.z !== "General") { try { return XLSX.SSF.format(cell.z, v); } catch {} } return String(Math.round(v * 1e10) / 1e10); }
    return String(v);
  };
  const text = (k, cell) => !cell ? "" : (!cell.f && !S.edited.has(k) && cell.w != null) ? cell.w : (cell.f && cell.v === cell.v0 && cell.w != null) ? cell.w : fmt(cell, cell.v);
  const load = (i) => {
    const ws = wb.Sheets[wb.SheetNames[i]] || {};
    const ref = ws["!ref"] ? XLSX.utils.decode_range(ws["!ref"]) : { e: { r: 0, c: 0 } };
    const rows = Math.min(ref.e.r + 1, MAXR), cols = Math.min(ref.e.c + 1, MAXC), cells = new Map();
    for (const a in ws) {
      if (a[0] === "!") continue;
      const p = XLSX.utils.decode_cell(a); if (p.r >= rows || p.c >= cols) continue;
      const c = ws[a], stub = c.t === "z"; // хоосон/утгагүй нүд
      if (stub && !c.f) continue;
      cells.set(p.r + "," + p.c, { v: stub ? null : c.v, v0: stub ? null : c.v, f: c.f, z: c.z, w: stub ? undefined : c.w });
    }
    S = { i, ws, rows, cols, cells, edited: new Set(), merges: ws["!merges"] || [], cut: ref.e.r + 1 > MAXR || ref.e.c + 1 > MAXC };
    recalc(); render(); select("0,0");
    $("[data-xs-reset]", el).hidden = true;
  };
  const render = () => {
    const covered = new Set(), span = new Map();
    for (const m of S.merges) {
      if (m.s.r >= S.rows || m.s.c >= S.cols) continue;
      const er = Math.min(m.e.r, S.rows - 1), ec = Math.min(m.e.c, S.cols - 1);
      span.set(m.s.r + "," + m.s.c, [er - m.s.r + 1, ec - m.s.c + 1]);
      for (let r = m.s.r; r <= er; r++) for (let c = m.s.c; c <= ec; c++) if (r !== m.s.r || c !== m.s.c) covered.add(r + "," + c);
    }
    const cw = (c) => { const x = (S.ws["!cols"] || [])[c]; return Math.max(40, Math.min(420, Math.round(x?.wpx || (x?.wch ? x.wch * 7.5 + 12 : 92)))); };
    let h = `<table class="xs-t"><colgroup><col style="width:46px">${Array.from({ length: S.cols }, (_, c) => `<col style="width:${cw(c)}px">`).join("")}</colgroup>
      <thead><tr><th class="xs-corner"></th>${Array.from({ length: S.cols }, (_, c) => `<th>${colName(c)}</th>`).join("")}</tr></thead><tbody>`;
    for (let r = 0; r < S.rows; r++) {
      h += `<tr><th>${r + 1}</th>`;
      for (let c = 0; c < S.cols; c++) {
        const k = r + "," + c; if (covered.has(k)) continue;
        const cell = S.cells.get(k), sp = span.get(k);
        h += `<td data-k="${k}"${sp ? ` rowspan="${sp[0]}" colspan="${sp[1]}"` : ""} class="${typeof cell?.v === "number" ? "n" : ""}${cell?.f ? " f" : ""}">${esc(text(k, cell))}</td>`;
      }
      h += "</tr>";
    }
    grid.innerHTML = h + "</tbody></table>";
    $$(".xs-tabs button", body).forEach((b) => b.setAttribute("aria-selected", String(+b.dataset.sheet === S.i)));
    $(".xs-note", body).textContent = S.cut ? `Эхний ${S.rows} мөр, ${S.cols} баганыг харууллаа` : parser ? "Нүдийг давхар дарж утгыг өөрчилбөл томьёо шууд бодогдоно (хадгалагдахгүй)" : "";
  };
  const refresh = () => {
    for (const td of $$("td[data-k]", grid)) {
      const k = td.dataset.k, cell = S.cells.get(k);
      if (cell?.f || S.edited.has(k)) { td.textContent = text(k, cell); td.classList.toggle("e", S.edited.has(k)); td.classList.toggle("n", typeof cell?.v === "number"); }
    }
    select(sel);
  };
  const select = (k) => {
    const td = grid.querySelector(`td[data-k="${k}"]`); if (!td) return;
    $(".xs-sel", grid)?.classList.remove("xs-sel"); td.classList.add("xs-sel"); sel = k;
    const [r, c] = k.split(",").map(Number), cell = S.cells.get(k);
    $(".xs-ref", body).textContent = colName(c) + (r + 1);
    $(".xs-f", body).textContent = cell?.f ? "=" + cell.f : text(k, cell);
  };
  const edit = (td) => {
    const k = td.dataset.k, cell = S.cells.get(k);
    if (!parser) return;
    if (cell?.f) return toast("Томьёотой нүд — оролтын нүдний утгыг өөрчилж туршина уу");
    const inp = document.createElement("input"); inp.className = "xs-in"; inp.value = cell?.v ?? "";
    td.textContent = ""; td.append(inp); inp.focus(); inp.select();
    let done = false;
    const finish = (save) => {
      if (done) return; done = true;
      if (save) {
        const t = inp.value.trim(), num = t !== "" && isFinite(+t.replace(",", ".")) ? +t.replace(",", ".") : null;
        let c2 = S.cells.get(k); if (!c2) { c2 = { v: null, v0: null }; S.cells.set(k, c2); }
        c2.v = num ?? (t === "" ? null : t); S.edited.add(k); recalc();
        $("[data-xs-reset]", el).hidden = false;
      }
      inp.remove(); refresh();
    };
    inp.addEventListener("keydown", (e) => { e.stopPropagation(); if (e.key === "Enter") finish(true); else if (e.key === "Escape") finish(false); });
    inp.addEventListener("blur", () => finish(true));
  };
  grid.addEventListener("click", (e) => { const td = e.target.closest("td[data-k]"); if (td && !e.target.closest("input")) select(td.dataset.k); });
  grid.addEventListener("dblclick", (e) => { const td = e.target.closest("td[data-k]"); if (td && !e.target.closest("input")) edit(td); });
  body.addEventListener("click", (e) => { const t = e.target.closest("[data-sheet]"); if (t) load(+t.dataset.sheet); });
  el.addEventListener("click", (e) => { if (e.target.closest("[data-xs-reset]")) load(S.i); });
  load(0);
  return {
    key(e) {
      const [r, c] = sel.split(",").map(Number), mv = { ArrowUp: [-1, 0], ArrowDown: [1, 0], ArrowLeft: [0, -1], ArrowRight: [0, 1] }[e.key];
      if (mv) { e.preventDefault(); const k = Math.max(0, Math.min(S.rows - 1, r + mv[0])) + "," + Math.max(0, Math.min(S.cols - 1, c + mv[1])); select(k); grid.querySelector(`td[data-k="${k}"]`)?.scrollIntoView({ block: "nearest", inline: "nearest" }); }
      else if (e.key === "Enter" || e.key === "F2") { e.preventDefault(); const td = grid.querySelector(`td[data-k="${sel}"]`); if (td) edit(td); }
    },
  };
}

/* ---------- Номын уншигч: хамгаалагдсан хуудсууд, 3D эргэлт, томруулах ----------
   Хуудас бүрийг серверээс (эрх шалгаж, уншигчийн тэмдэгтэйгээр) нэг нэгээр нь авна. Төлөөгүй бол зөвхөн
   үнэгүй хуудсууд ба төлбөрийн хуудас — үлдсэн хуудас огт байхгүй. */
const BookReader = {
  open({ id, title, pages, full, price, loggedIn, onBuy }) {
    this.close();
    Object.assign(this, { id, title, total: full ? pages : pages, readable: pages, full, price, loggedIn, onBuy });
    this.paywall = !full; // үнэгүй хэсгийн дараа төлбөрийн хуудас
    this.n = this.readable + (this.paywall ? 1 : 0);
    this.urls = new Map(); this.loading = new Set(); this.zoom = 1; this.pan = { x: 0, y: 0 }; this.fit = "page"; this.mode = "auto";
    const el = document.createElement("div");
    el.className = "reader"; el.setAttribute("role", "dialog"); el.setAttribute("aria-label", title);
    el.innerHTML = `<div class="rd-bar">
        <strong class="rd-title">${esc(title)}</strong>
        <span class="rd-count" aria-live="polite"></span>
        <div class="rd-tools">
          <button class="rd-btn" data-z="-" title="Жижигрүүлэх (−)" aria-label="Жижигрүүлэх">−</button>
          <button class="rd-btn rd-zoom" data-z="0" title="Анхны хэмжээ (0)">100%</button>
          <button class="rd-btn" data-z="+" title="Томруулах (+)" aria-label="Томруулах">+</button>
          <span class="rd-sep"></span>
          <button class="rd-btn" data-fit title="Хуудасны өргөнд / бүтэн хуудсанд тааруулах">⇔</button>
          <button class="rd-btn" data-mode title="Нэг / хоёр хуудсаар">▯▯</button>
          <button class="rd-btn" data-fs title="Бүтэн дэлгэц (F)">⛶</button>
          <button class="rd-btn" data-close title="Хаах (Esc)" aria-label="Хаах">✕</button>
        </div></div>
      <div class="rd-stage"><div class="rd-pan"><div class="fb-book"></div></div></div>
      <div class="rd-nav"><button class="rd-btn rd-arrow" data-prev aria-label="Өмнөх хуудас">‹</button>
        <input type="range" min="0" value="0" aria-label="Хуудас">
        <button class="rd-btn rd-arrow" data-next aria-label="Дараагийн хуудас">›</button></div>`;
    document.body.append(el); this.el = el;
    document.documentElement.classList.add("reader-open");
    requestAnimationFrame(() => el.classList.add("open"));
    el.addEventListener("contextmenu", (e) => e.preventDefault());
    el.addEventListener("dragstart", (e) => e.preventDefault());
    el.addEventListener("click", (e) => this.onClick(e));
    this.key = (e) => this.onKey(e); addEventListener("keydown", this.key);
    this.rs = () => { clearTimeout(this.rt); this.rt = setTimeout(() => this.layout(), 120); }; addEventListener("resize", this.rs);
    this.bindGestures();
    this.layout();
  },
  layout() {
    const el = this.el; if (!el) return;
    const stage = $(".rd-stage", el), sw = stage.clientWidth - 24, sh = stage.clientHeight - 24;
    this.single = this.mode === "single" || (this.mode === "auto" && sw < 820);
    const ratio = this.ratio || 1.414, cols = this.single ? 1 : 2;
    let pw = this.fit === "width" ? sw / cols : Math.min(sw / cols, sh / ratio);
    pw = Math.max(160, pw);
    const ph = pw * ratio, per = this.single ? 1 : 2;
    this.pw = pw; this.ph = ph;
    const book = $(".fb-book", el);
    book.className = "fb-book" + (this.single ? " single" : "");
    book.style.width = pw * cols + "px"; book.style.height = ph + "px";
    this.leaves = Math.ceil(this.n / per);
    book.innerHTML = "";
    for (let i = 0; i < this.leaves; i++) {
      const f = i * per + 1, b = per === 2 ? f + 1 : 0, leaf = document.createElement("div");
      leaf.className = "leaf";
      leaf.innerHTML = this.faceHTML(f, "front") + (b && b <= this.n ? this.faceHTML(b, "back") : `<div class="face back"></div>`);
      book.append(leaf);
    }
    this.leafEls = $$(".leaf", book);
    const range = $("input[type=range]", el); range.max = this.leaves;
    range.oninput = () => this.go(+range.value, true);
    this.applyZoom();
    this.go(Math.min(this.cur || 0, this.leaves), true);
  },
  faceHTML(p, side) {
    if (this.paywall && p === this.n) return `<div class="face ${side} rd-paywall"><div class="rd-pw">
        <span class="rd-pw-ico">🔒</span><h3>Үргэлжлүүлэн унших</h3>
        <p>Та эхний ${this.readable} хуудсыг үнэгүй уншлаа. ${this.price ? "Үлдсэн хэсгийг унших бол номыг худалдаж авна уу." : "Үлдсэн хэсгийг унших бол нэвтэрнэ үү."}</p>
        <button class="btn btn-gold" data-buy>${this.price ? "Худалдаж авах · " + money(this.price) : "Нэвтрэх"}</button></div></div>`;
    const u = this.urls.get(p);
    return `<div class="face ${side}" data-p="${p}">${u ? `<img src="${u}" alt="" draggable="false">` : `<div class="page-loading">${p}</div>`}<span class="pnum">${p}</span></div>`;
  },
  go(k, instant) {
    if (!this.leafEls) return;
    k = Math.max(0, Math.min(this.leaves, k));
    this.leafEls.forEach((leaf, i) => {
      const flip = i < k;
      if (instant) leaf.style.transition = "none";
      if (flip !== leaf.classList.contains("flipped")) { leaf.classList.add("turning"); leaf.style.zIndex = 1000; setTimeout(() => leaf.classList.remove("turning"), 900); }
      leaf.classList.toggle("flipped", flip);
      setTimeout(() => { leaf.style.zIndex = flip ? i + 1 : this.leafEls.length - i; }, instant ? 0 : 450);
      if (instant) requestAnimationFrame(() => (leaf.style.transition = ""));
    });
    this.cur = k;
    const book = $(".fb-book", this.el);
    if (!this.single) { book.classList.toggle("closed-front", k === 0); book.classList.toggle("closed-back", k === this.leaves); }
    $("input[type=range]", this.el).value = k;
    const per = this.single ? 1 : 2;
    const from = this.single ? k + 1 : Math.max(1, k * 2), to = Math.min(this.n, this.single ? k + 1 : k * 2 + 1);
    const shown = (x) => (this.paywall && x === this.n ? "🔒" : x);
    $(".rd-count", this.el).textContent = `${shown(from)}${to !== from ? "–" + shown(to) : ""} / ${this.full ? this.total : this.readable + " үнэгүй"}`;
    $("[data-prev]", this.el).disabled = k === 0; $("[data-next]", this.el).disabled = k >= this.leaves;
    for (let p = Math.max(1, k * per - 2); p <= Math.min(this.readable, k * per + 4); p++) this.load(p);
    this.trim(k * per);
  },
  async load(p) {
    if (this.urls.has(p) || this.loading.has(p) || !this.el) return;
    this.loading.add(p);
    try {
      const res = await fetch(`/api/books/${this.id}/pages/${p}`, { headers: Auth.token ? { Authorization: "Bearer " + Auth.token } : {}, cache: "no-store" });
      if (res.status === 429) { toast("Хэт хурдан эргүүлж байна — түр хүлээнэ үү"); setTimeout(() => { this.loading.delete(p); this.load(p); }, 8000); return; }
      if (!res.ok) throw new Error((await res.json().catch(() => null))?.error || "Хуудас ачаалагдсангүй");
      const url = URL.createObjectURL(await res.blob());
      if (!this.el) return URL.revokeObjectURL(url);
      this.urls.set(p, url);
      const face = $(`.face[data-p="${p}"]`, this.el);
      if (face) { const img = new Image(); img.src = url; img.alt = ""; img.draggable = false; img.onload = () => { if (!this.ratio && p === 1) { this.ratio = img.naturalHeight / img.naturalWidth; this.layout(); } }; $(".page-loading", face)?.replaceWith(img); }
    } catch (e) { toast(e.message, true); } finally { this.loading.delete(p); }
  },
  // Санах ойг хэмнэнэ: одоогийн хуудаснаас хол байгаа зургийг чөлөөлнө.
  trim(center) {
    for (const [p, u] of this.urls) if (Math.abs(p - center) > 16) { URL.revokeObjectURL(u); this.urls.delete(p); const f = $(`.face[data-p="${p}"]`, this.el); if (f) f.innerHTML = `<div class="page-loading">${p}</div><span class="pnum">${p}</span>`; }
  },
  setZoom(z, cx, cy) {
    const old = this.zoom; z = Math.min(4, Math.max(0.5, z));
    if (cx != null) { // хулганы цэг дээр төвлөрч томруулна
      const r = $(".rd-stage", this.el).getBoundingClientRect(), ox = cx - r.left - r.width / 2, oy = cy - r.top - r.height / 2;
      this.pan.x = ox - (ox - this.pan.x) * (z / old); this.pan.y = oy - (oy - this.pan.y) * (z / old);
    }
    if (z <= 1) this.pan = { x: 0, y: 0 };
    this.zoom = z; this.applyZoom();
  },
  applyZoom() {
    const pan = $(".rd-pan", this.el); if (!pan) return;
    pan.style.transform = `translate(${this.pan.x}px, ${this.pan.y}px) scale(${this.zoom})`;
    this.el.classList.toggle("zoomed", this.zoom > 1);
    $(".rd-zoom", this.el).textContent = Math.round(this.zoom * 100) + "%";
  },
  bindGestures() {
    const stage = $(".rd-stage", this.el), pts = new Map();
    let start = null, pinch = null;
    stage.addEventListener("wheel", (e) => {
      if (e.ctrlKey || e.metaKey) { e.preventDefault(); this.setZoom(this.zoom * (e.deltaY < 0 ? 1.1 : 0.9), e.clientX, e.clientY); }
      else if (this.zoom > 1) { e.preventDefault(); this.pan.x -= e.deltaX; this.pan.y -= e.deltaY; this.applyZoom(); }
    }, { passive: false });
    stage.addEventListener("dblclick", (e) => this.setZoom(this.zoom > 1 ? 1 : 2, e.clientX, e.clientY));
    stage.addEventListener("pointerdown", (e) => {
      pts.set(e.pointerId, { x: e.clientX, y: e.clientY });
      if (pts.size === 2) { const [a, b] = [...pts.values()]; pinch = { d: Math.hypot(a.x - b.x, a.y - b.y), z: this.zoom }; start = null; return; }
      start = { x: e.clientX, y: e.clientY, px: this.pan.x, py: this.pan.y, moved: false };
      stage.setPointerCapture?.(e.pointerId);
    });
    stage.addEventListener("pointermove", (e) => {
      if (!pts.has(e.pointerId)) return;
      pts.set(e.pointerId, { x: e.clientX, y: e.clientY });
      if (pinch && pts.size === 2) { const [a, b] = [...pts.values()]; this.setZoom(pinch.z * Math.hypot(a.x - b.x, a.y - b.y) / pinch.d, (a.x + b.x) / 2, (a.y + b.y) / 2); return; }
      if (!start) return;
      const dx = e.clientX - start.x, dy = e.clientY - start.y;
      if (Math.abs(dx) + Math.abs(dy) > 6) start.moved = true;
      if (this.zoom > 1) { this.pan.x = start.px + dx; this.pan.y = start.py + dy; this.applyZoom(); }
    });
    const end = (e) => {
      pts.delete(e.pointerId);
      if (pts.size < 2) pinch = null;
      if (!start) return;
      const dx = e.clientX - start.x;
      if (this.zoom <= 1) {
        if (start.moved && Math.abs(dx) > 40) this.go(this.cur + (dx < 0 ? 1 : -1)); // шудрах
        else if (!start.moved && !e.target.closest("[data-buy]")) { // хуудсан дээр дарах: баруун тал → дараах
          const r = stage.getBoundingClientRect(); this.go(this.cur + (e.clientX > r.left + r.width / 2 ? 1 : -1));
        }
      }
      start = null;
    };
    stage.addEventListener("pointerup", end); stage.addEventListener("pointercancel", end);
  },
  onClick(e) {
    const t = e.target;
    if (t.closest("[data-close]")) return this.close();
    if (t.closest("[data-prev]")) return this.go(this.cur - 1);
    if (t.closest("[data-next]")) return this.go(this.cur + 1);
    if (t.closest("[data-buy]")) { this.close(); return this.onBuy?.(); }
    const z = t.closest("[data-z]");
    if (z) return this.setZoom(z.dataset.z === "+" ? this.zoom * 1.25 : z.dataset.z === "-" ? this.zoom / 1.25 : 1);
    if (t.closest("[data-fit]")) { this.fit = this.fit === "page" ? "width" : "page"; this.zoom = 1; this.pan = { x: 0, y: 0 }; return this.layout(); }
    if (t.closest("[data-mode]")) { this.mode = this.single ? "double" : "single"; const p = this.cur * (this.single ? 1 : 2); this.cur = this.mode === "single" ? p : Math.floor(p / 2); return this.layout(); }
    if (t.closest("[data-fs]")) { document.fullscreenElement ? document.exitFullscreen?.() : this.el.requestFullscreen?.(); }
  },
  onKey(e) {
    if (e.key === "ArrowRight" || e.key === "PageDown" || e.key === " ") { e.preventDefault(); this.go(this.cur + 1); }
    else if (e.key === "ArrowLeft" || e.key === "PageUp") { e.preventDefault(); this.go(this.cur - 1); }
    else if (e.key === "Home") this.go(0); else if (e.key === "End") this.go(this.leaves);
    else if (e.key === "+" || e.key === "=") this.setZoom(this.zoom * 1.25);
    else if (e.key === "-") this.setZoom(this.zoom / 1.25);
    else if (e.key === "0") this.setZoom(1);
    else if (e.key.toLowerCase() === "f") $("[data-fs]", this.el).click();
    else if (e.key === "Escape" && !document.fullscreenElement) this.close();
  },
  close() {
    if (!this.el) return;
    const el = this.el; this.el = null; this.leafEls = null;
    removeEventListener("keydown", this.key); removeEventListener("resize", this.rs);
    if (document.fullscreenElement) document.exitFullscreen?.();
    for (const u of this.urls.values()) URL.revokeObjectURL(u);
    this.urls.clear(); this.ratio = 0; this.cur = 0;
    document.documentElement.classList.remove("reader-open");
    el.classList.remove("open"); setTimeout(() => el.remove(), 300);
  },
};

function bookPage() {
  const root = $(".book-page"); if (!root) return;
  const id = root.dataset.book, pages = +root.dataset.pages, title = root.dataset.title, note = $("#bkNote");
  let access = { full: false, logged_in: !!Auth.token, preview_pages: Math.min(+root.dataset.preview, pages) };
  const buyBtn = $("#bkBuy"), readBtn = $("#bkRead");
  const refresh = () => api(`/api/books/${id}`).then((d) => {
    access = d.access;
    if (access.full) {
      readBtn.textContent = "📖 Бүтнээр нь унших";
      if (buyBtn) buyBtn.hidden = true;
      note.textContent = access.owner ? "Энэ таны ном — уншигчид хуудас бүр дээр өөрийн нэр шингэсэн байдлаар харна." : "✓ Танд энэ ном бүтнээрээ нээлттэй. Хуудас бүрд таны нэр шингэсэн тул бусдад тараахгүй байна уу.";
    }
    return access;
  }).catch(() => access);
  refresh();
  const login = () => (location.href = "/login?next=" + encodeURIComponent(location.pathname));
  const buy = async () => {
    if (!Auth.token) return login();
    if (!+root.dataset.price) return login();
    try {
      const d = await api(`/api/books/${id}/buy`, { method: "POST" });
      if (d.unlocked) { await refresh(); return read(); }
      BookReader.close?.(); // уншигч нээлттэй бол төлбөрийн цонх дээр нь гарна
      payWindow(d.order, d.payment, `«${title}» ном`, async () => { toast("🎉 Ном нээгдлээ!"); celebrate(); await refresh(); read(); }, { name: "Энэ ном" });
    } catch (e) { toast(e.message, true); }
  };
  const read = () => BookReader.open({ id, title, pages: access.full ? pages : Math.min(access.preview_pages || 3, pages), full: access.full || pages <= (access.preview_pages || 3),
    price: +root.dataset.price, loggedIn: !!Auth.token, onBuy: +root.dataset.price ? buy : login });
  readBtn.onclick = read;
  if (buyBtn) buyBtn.onclick = buy;
  $("#bkLike").onclick = async (e) => { const b = e.currentTarget; await api(`/api/books/${id}/interest`, { method: "POST" }).catch(() => {}); b.textContent = "♥ Сонирхлоо"; b.disabled = true; toast("Багшид таны сонирхлыг мэдэгдлээ"); };
  if (new URLSearchParams(location.hash.slice(1)).has("read")) refresh().then(read);
}

/* ---------- чатын цонх (профайл, сургалт) ---------- */
function linkify(text) {
  return esc(text).replace(/https?:\/\/[^\s<]+/g, (u) => `<a href="${u}" target="_blank" rel="noopener">${u.includes("meet.google.com") ? "📹 Google Meet-д нэгдэх" : u}</a>`);
}
// group=true бол бүлэг чат: "миний" эсэхийг хэрэглэгчийн ID-аар, бусдын мессежид нэрийг нь харуулна.
function msgHTML(m, me, group) { // хуучин дуудлагад: энгийн бөмбөлөг
  const myId = Auth.user?.id;
  const mine = group && myId ? m.sender_id === myId : m.sender === me;
  const who = group && !mine && m.sender_name ? `<b class="msg-who">${esc(m.sender_name)}${m.sender === "teacher" ? " · багш" : ""}</b>` : "";
  return `<li class="msg ${mine ? "me" : "them"}" data-id="${esc(m.id)}">${who}${linkify(m.body)}<time>${fmtTime(m.created_at)}</time></li>`;
}

/* ---------- Чатын урсгал (Messenger маягийн): бүлэглэсэн бөмбөлөг, өдрийн тусгаарлагч, хариулах (quote),
   реакц (👍❤️😂😮😢🙏), "Үзсэн", "бичиж байна…", Enter-ээр илгээх ---------- */
const REACTS = ["👍", "❤️", "😂", "😮", "😢", "🙏"];
const EMOJIS = ["😀", "😂", "🥰", "😍", "😎", "🤔", "😅", "😭", "😡", "🙏", "👍", "👎", "👏", "🙌", "💪", "🔥", "🎉", "❤️", "💙", "✅", "❌", "⭐", "📚", "✍️", "🧠", "💡", "⏰", "🏆"];
const STICKERS = ["🎉", "👏", "🏆", "💯", "🔥", "🚀", "🧠", "📚", "✅", "🙏", "😎", "🥳", "🤝", "💪", "🌟", "🎯"];
const ICO_IMG = `<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linejoin="round"><rect x="3" y="4" width="18" height="16" rx="3"/><circle cx="9" cy="10" r="1.6"/><path d="m21 16-5-5-9 9"/></svg>`;
const ICO_STICKER = `<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 13V7a3 3 0 0 0-3-3H7a3 3 0 0 0-3 3v10a3 3 0 0 0 3 3h6"/><path d="M20 13h-4a3 3 0 0 0-3 3v4l7-7Z"/><path d="M9 10h.01M14 10h.01M9 14c.8.8 1.8 1.2 3 1.2"/></svg>`;
const ICO_SMILE = `<svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="12" cy="12" r="9"/><path d="M9 10h.01M15 10h.01M8.5 14.5c1 1.2 2.2 1.8 3.5 1.8s2.5-.6 3.5-1.8"/></svg>`;
function dayLabel(t) {
  const d = new Date(t), now = new Date(), one = 86400000;
  const sd = new Date(d.getFullYear(), d.getMonth(), d.getDate()), sn = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const diff = Math.round((sn - sd) / one);
  if (diff === 0) return "Өнөөдөр";
  if (diff === 1) return "Өчигдөр";
  if (diff < 7) return WEEKDAYS?.[d.getDay()] || fmtDate(t);
  return fmtDate(t).replace(/\s\d{1,2}:\d{2}$/, "");
}
class ChatThread {
  // opts: {ol, body, form, token(): string, convId(): string, role(): "teacher"|"visitor", group(): bool}
  constructor(o) {
    Object.assign(this, { list: [], reads: {}, meKey: "", typing: new Map(), replyTo: null, lastReadSent: "", typeAt: 0, editing: null }, o);
    this.ol.classList.add("thread");
    this.ol.addEventListener("click", (e) => this.click(e));
    this.ol.addEventListener("submit", async (e) => { // мессеж засах
      const f = e.target.closest(".msg-edit"); if (!f) return; e.preventDefault();
      const id = this.editing, body = f.querySelector("textarea").value.trim(); if (!body) return;
      try { const m = await api(`/api/chat/${this.convId()}/messages/${id}`, { method: "PUT", body: { body }, token: this.token() }); const cur = this.list.find((x) => x.id === id); if (cur) Object.assign(cur, m); this.editing = null; this.render(); }
      catch (x) { toast(x.message, true); }
    });
    this.ol.addEventListener("keydown", (e) => { if (e.target.matches(".msg-edit textarea")) { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); e.target.closest("form").requestSubmit(); } if (e.key === "Escape") { this.editing = null; this.render(); } } });
    document.addEventListener("click", (e) => { if (!e.target.closest(".react-pick, [data-react]")) $(".react-pick", this.ol)?.remove(); });
    if (this.form) this.bindForm();
  }
  mine(m) { const myId = Auth.user?.id; return this.group() && myId ? m.sender_id === myId : m.sender === this.role(); }
  set(d) { this.list = d.messages || []; this.reads = d.reads || {}; this.meKey = d.me_key || this.meKey; this.replyTo = null; this.render(); this.scroll(true); this.markRead(); }
  append(m) {
    if (this.list.some((x) => x.id === m.id)) return;
    this.list.push(m); this.typing.delete(m.sender_id ? "u:" + m.sender_id : m.sender_name); this._animLast = true;
    const atBottom = this.body.scrollHeight - this.body.scrollTop - this.body.clientHeight < 120;
    this.render(); if (atBottom || this.mine(m)) this.scroll(true);
    if (!this.mine(m)) this.markRead();
  }
  event(d) {
    if (d.conversation_id !== this.convId()) return;
    if (d.type === "reaction") { const m = this.list.find((x) => x.id === d.message_id); if (m) { m.reactions = d.reactions; this.render(); } }
    if (d.type === "read" && d.key !== this.meKey) { this.reads[d.key] = d.last_id; this.render(); }
    if (d.type === "edit") { const m = this.list.find((x) => x.id === d.message.id); if (m) { Object.assign(m, d.message); this.render(); } }
    if (d.type === "delete") { const m = this.list.find((x) => x.id === d.message_id); if (m) { m.deleted = true; m.body = ""; m.attachment_url = ""; m.reactions = {}; this.render(); } }
    if (d.type === "typing" && d.key !== this.meKey) { this.typing.set(d.key, { name: d.name, at: Date.now() }); this.render(); this.scroll(); clearTimeout(this._tt); this._tt = setTimeout(() => { this.sweepTyping(); this.render(); }, 4200); }
  }
  sweepTyping() { for (const [k, v] of this.typing) if (Date.now() - v.at > 4000) this.typing.delete(k); }
  scroll(force) { if (force || this.body.scrollHeight - this.body.scrollTop - this.body.clientHeight < 160) this.body.scrollTop = this.body.scrollHeight; }
  async markRead() {
    const last = [...this.list].reverse().find((m) => !this.mine(m)); if (!last || last.id === this.lastReadSent || !this.convId()) return;
    this.lastReadSent = last.id;
    try { await api(`/api/chat/${this.convId()}/read`, { method: "POST", body: { message_id: last.id }, token: this.token() }); } catch {}
  }
  seenBy() { // миний сүүлийн мессежийг хэн үзсэн бэ
    const mineLast = [...this.list].reverse().find((m) => this.mine(m)); if (!mineLast) return null;
    const who = Object.entries(this.reads).filter(([k, id]) => k !== this.meKey && id >= mineLast.id);
    if (!who.length) return null;
    return { id: mineLast.id, n: who.length };
  }
  render() {
    this.sweepTyping();
    const L = this.list, out = [], seen = this.seenBy();
    let lastDay = "";
    const GAP = 15 * 60000; // ойрхон цагт (15 мин дотор) бичсэн нь нэг бүлэг
    const who = (m) => m.sender_id || m.sender_name || m.sender;
    L.forEach((m, i) => {
      const day = dayLabel(m.created_at);
      const prev = L[i - 1], next = L[i + 1];
      if (day !== lastDay) { out.push(`<li class="msg-day"><span>${esc(day)}</span></li>`); lastDay = day; }
      else if (prev && new Date(m.created_at) - new Date(prev.created_at) > GAP) out.push(`<li class="msg-day msg-gap"><span>${fmtTime(m.created_at)}</span></li>`); // завсар их → цагийн шошго
      const same = (a, b) => a && b && who(a) === who(b) && Math.abs(new Date(a.created_at) - new Date(b.created_at)) < GAP && dayLabel(a.created_at) === dayLabel(b.created_at);
      const first = !same(prev, m), last = !same(m, next), mine = this.mine(m);
      const name = this.group() && !mine && first && m.sender_name ? `<b class="msg-who">${esc(m.sender_name)}${m.sender === "teacher" ? ` <span class="dc-badge">Багш</span>` : ""}</b>` : "";
      const quote = m.reply_to ? `<button type="button" class="msg-quote" data-goto="${esc(m.reply_to)}"><b>${esc(m.reply_name || "")}</b><span>${esc((m.reply_body || "").replace(/^::sticker::/, ""))}</span></button>` : "";
      const rx = Object.entries(m.reactions || {}).filter(([, u]) => u.length);
      const reacts = rx.length ? `<div class="msg-reacts">${rx.map(([e, u]) => `<button type="button" class="${u.some((x) => x.key === this.meKey) ? "on" : ""}" data-react-one="${e}" title="${esc(u.map((x) => x.name).join(", "))}">${e}${u.length > 1 ? ` ${u.length}` : ""}</button>`).join("")}</div>` : "";
      const av = !mine && last ? `<span class="msg-av">${avatarHTML({ display_name: m.sender_name || (m.sender === "teacher" ? "Багш" : "?") }, "avatar-sm")}</span>` : `<span class="msg-av"></span>`;
      const foot = last || rx.length || m.edited ? `<div class="msg-foot"><time>${fmtTime(m.created_at)}</time>${m.edited && !m.deleted ? `<span class="msg-edited">засварласан</span>` : ""}${mine && seen && seen.id === m.id ? `<span class="msg-seen">✓✓ Үзсэн${this.group() && seen.n > 1 ? ` · ${seen.n}` : ""}</span>` : ""}</div>` : "";
      const sticker = !m.deleted && /^::sticker::(.+)$/.exec(m.body || "");
      const imgUrl = m.deleted ? "" : m.attachment_url || (/^https?:\/\/\S+\.(gif|png|jpe?g|webp)(\?\S*)?$/i.test(m.body || "") ? m.body : "");
      const editing = this.editing === m.id;
      const content = m.deleted ? `<span class="msg-text msg-deleted">Мессеж устгагдсан</span>` : editing ? `<form class="msg-edit"><textarea rows="2" maxlength="2000">${esc(m.body)}</textarea><div><button type="submit" class="btn btn-gold btn-sm">Хадгалах</button><button type="button" class="btn btn-ghost btn-sm" data-edit-cancel>Болих</button></div></form>` : sticker ? `<span class="msg-sticker">${esc(sticker[1])}</span>` : imgUrl ? `<a class="msg-img" href="${esc(imgUrl)}" target="_blank" rel="noopener"><img src="${esc(imgUrl)}" alt="" loading="lazy"></a>${m.attachment_url && m.body && m.body !== "📷 Зураг" ? `<span class="msg-text">${linkify(m.body)}</span>` : ""}` : `<span class="msg-text">${linkify(m.body)}</span>`;
      out.push(`<li class="msg ${mine ? "me" : "them"} ${first ? "first" : ""} ${last ? "last" : ""} ${sticker ? "is-sticker" : ""} ${imgUrl ? "has-img" : ""}" data-id="${esc(m.id)}">${av}<div class="msg-col">${name}
        <div class="msg-row"><div class="bubble" title="${fmtDate(m.created_at)}">${quote}${content}</div>
          <div class="msg-tools">${m.deleted ? "" : `<button type="button" data-react="${esc(m.id)}" title="Реакц">☺</button>`}${m.deleted ? "" : `<button type="button" data-reply="${esc(m.id)}" title="Хариулах">↩</button>`}${mine && !m.deleted ? `<button type="button" data-edit="${esc(m.id)}" title="Засах">✎</button>` : ""}${(mine || this.role() === "teacher") && !m.deleted ? `<button type="button" data-del="${esc(m.id)}" title="Устгах">🗑</button>` : ""}</div></div>
        ${reacts}${foot}</div></li>`);
    });
    for (const [, v] of this.typing) out.push(`<li class="msg them typing"><span class="msg-av"></span><div class="msg-col"><div class="bubble"><span class="dots"><i></i><i></i><i></i></span></div><small class="muted">${esc(v.name)} бичиж байна…</small></div></li>`);
    if (!out.length) out.push(`<li class="muted small" style="text-align:center">Анхны мессежээ бичээрэй ✍️</li>`);
    this.ol.innerHTML = out.join("");
    if (this._animLast) { this.ol.lastElementChild?.classList.add("msg-in"); this._animLast = false; }
    this.renderReply();
  }
  async click(e) {
    const t = e.target, conv = this.convId();
    const rb = t.closest("[data-react]");
    if (rb) { $(".react-pick", this.ol)?.remove(); rb.insertAdjacentHTML("afterend", `<div class="react-pick">${REACTS.map((r) => `<button type="button" data-pick="${r}">${r}</button>`).join("")}</div>`); return; }
    const pk = t.closest("[data-pick]");
    if (pk) { const id = pk.closest(".msg").dataset.id; $(".react-pick", this.ol)?.remove(); await this.react(id, pk.dataset.pick); return; }
    const one = t.closest("[data-react-one]");
    if (one) { await this.react(one.closest(".msg").dataset.id, one.dataset.reactOne); return; }
    const rp = t.closest("[data-reply]");
    if (rp) { const m = this.list.find((x) => x.id === rp.dataset.reply); this.replyTo = m || null; this.renderReply(); this.form?.body.focus(); return; }
    const ed = t.closest("[data-edit]");
    if (ed) { this.editing = ed.dataset.edit; this.render(); const ta = $(".msg-edit textarea", this.ol); if (ta) { ta.focus(); ta.selectionStart = ta.value.length; } return; }
    if (t.closest("[data-edit-cancel]")) { this.editing = null; this.render(); return; }
    const del = t.closest("[data-del]");
    if (del && confirm("Мессежийг устгах уу?")) { try { await api(`/api/chat/${conv}/messages/${del.dataset.del}`, { method: "DELETE", token: this.token() }); const m = this.list.find((x) => x.id === del.dataset.del); if (m) { m.deleted = true; m.body = ""; m.attachment_url = ""; m.reactions = {}; } this.render(); } catch (x) { toast(x.message, true); } return; }
    const go = t.closest("[data-goto]");
    if (go) { const el = this.ol.querySelector(`[data-id="${CSS.escape(go.dataset.goto)}"]`); if (el) { el.scrollIntoView({ block: "center", behavior: "smooth" }); el.classList.add("flash"); setTimeout(() => el.classList.remove("flash"), 1200); } return; }
    if (t.closest("[data-cancel-reply]")) { this.replyTo = null; this.renderReply(); }
  }
  async react(id, emoji) {
    try { const r = await api(`/api/chat/${this.convId()}/react`, { method: "POST", body: { message_id: id, emoji }, token: this.token() }); const m = this.list.find((x) => x.id === id); if (m) { m.reactions = r.reactions; this.render(); } }
    catch (x) { toast(x.message, true); }
  }
  renderReply() {
    if (!this.form) return;
    let bar = this.form.querySelector(".reply-bar");
    if (!this.replyTo) { bar?.remove(); return; }
    if (!bar) { bar = document.createElement("div"); bar.className = "reply-bar"; this.form.prepend(bar); }
    const who = this.mine(this.replyTo) ? "Өөртөө" : `<b>${esc(this.replyTo.sender_name || (this.replyTo.sender === "teacher" ? "Багш" : "Зочин"))}</b>-д`;
    bar.innerHTML = `<span>↩ ${who} хариулж байна: <em>${esc((this.replyTo.body || "").replace(/^::sticker::/, "").slice(0, 80))}</em></span><button type="button" data-cancel-reply aria-label="Болих">✕</button>`;
    bar.querySelector("[data-cancel-reply]").onclick = () => { this.replyTo = null; this.renderReply(); };
  }
  // Бичих мөр: [📹 Meet (багш)] [Aa] [➤] — энгийн, зөвхөн уулзалтын товчтой.
  bindForm() {
    const f = this.form, extra = [...f.querySelectorAll("[data-plus]")];
    f.classList.add("composer-bar");
    f.innerHTML = `${extra.length ? `<div class="cb-left">${extra.map((b) => `<button type="button" class="cb-ic" id="${esc(b.id)}" title="${esc(b.textContent.trim())}" aria-label="${esc(b.textContent.trim())}">${b.querySelector("svg")?.outerHTML || "📹"}</button>`).join("")}</div>` : ""}
      <div class="cb-input"><textarea name="body" rows="1" maxlength="2000" placeholder="Мессеж бичих…" aria-label="Мессеж"></textarea></div>
      <button class="cb-send has-arrow" aria-label="Илгээх" title="Илгээх (Enter)"><svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor" aria-hidden="true"><path d="M3.4 20.4 21.9 12 3.4 3.6l-.1 6.6L15 12 3.3 13.8l.1 6.6Z"/></svg></button>`;
    const ta = f.body;
    const grow = () => { ta.style.height = "auto"; ta.style.height = Math.min(140, ta.scrollHeight) + "px"; f.classList.toggle("has-text", !!ta.value.trim()); };
    ta.addEventListener("input", () => { grow(); this.sendTyping(); });
    ta.addEventListener("keydown", (e) => { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); f.requestSubmit(); } if (e.key === "Escape" && this.replyTo) { this.replyTo = null; this.renderReply(); } });
    f.addEventListener("submit", async (e) => { e.preventDefault(); const text = ta.value.trim(); if (!text) return; ta.value = ""; grow(); if (!(await this.send(text))) { ta.value = text; grow(); } });
    grow();
  }
  async send(body, attachment = "") {
    const conv = this.convId(); if (!conv) return false;
    const reply = this.replyTo; this.replyTo = null; this.renderReply();
    try { const m = await api(`/api/chat/${conv}/messages`, { method: "POST", body: { body, reply_to: reply?.id || "", attachment }, token: this.token() }); this.append(m); this.onSent?.(m); return true; }
    catch (x) { toast(x.message, true); this.replyTo = reply; this.renderReply(); return false; }
  }
  async sendTyping() { if (Date.now() - this.typeAt < 3000 || !this.convId()) return; this.typeAt = Date.now(); try { await api(`/api/chat/${this.convId()}/typing`, { method: "POST", body: {}, token: this.token() }); } catch {} }
}
function chatWidget() {
  const fab = $("#chatFab"), panel = $("#chatPanel");
  if (!fab || !panel) return;
  const teacher = panel.dataset.teacher, msgs = $("#chatMsgs"), body = $("#chatBody"), form = $("#chatForm"), intro = $("#chatIntro");
  const courseId = $(".course-page")?.dataset.course;
  let conv = null, group = false;
  const headText = $(".chat-head > div:nth-child(2)", panel);
  const thread = new ChatThread({ ol: msgs, body, form, token: () => (group ? Auth.token : Auth.chatToken), convId: () => conv?.id, role: () => "visitor", group: () => group });
  const scroll = () => (body.scrollTop = body.scrollHeight);
  const open = async () => {
    panel.classList.add("open"); panel.setAttribute("aria-hidden", "false"); $(".chat-fab-dot").hidden = true;
    if (conv) return;
    if (!Auth.chatToken) { intro.hidden = false; intro.querySelector("input").focus(); return; }
    await start();
  };
  // Сургалтын бүлэг чат: зөвхөн нэвтэрсэн, элссэн суралцагч. Багш нээгээгүй бол хэлнэ.
  const openGroup = async () => {
    if (!Auth.token) { location.href = "/login?next=" + encodeURIComponent(location.pathname + "#chat=group"); return; }
    panel.classList.add("open"); panel.setAttribute("aria-hidden", "false"); $(".chat-fab-dot").hidden = true;
    if (conv && group) return;
    try {
      const d = await api(`/api/courses/${courseId}/chat`, { method: "POST" });
      conv = d.conversation; group = true;
      headText.innerHTML = `<strong>${esc(conv.visitor_name)}</strong><small class="chat-status"><i></i>Бүлэг чат · сургалтын суралцагчид</small>`;
      intro.hidden = true; msgs.hidden = false; form.hidden = false;
      thread.set(d); Live.reconnect(); form.body.focus();
    } catch (e) { intro.hidden = true; msgs.hidden = false; form.hidden = true; msgs.innerHTML = `<li class="muted" style="text-align:center;padding:12px">${esc(e.message)}</li>`; }
  };
  const start = async () => {
    try {
      const d = await api(`/api/teachers/${teacher}/chat`, { method: "POST", token: Auth.chatToken });
      conv = d.conversation; group = false;
      intro.hidden = true; msgs.hidden = false; form.hidden = false;
      thread.set(d);
      Live.connect(Auth.chatToken);
      form.body.focus();
    } catch (e) {
      if (e.status === 401) { store.set("sg_guest", null); intro.hidden = false; return; }
      intro.hidden = true; msgs.hidden = false; msgs.innerHTML = `<li class="muted">${esc(e.message)}</li>`;
    }
  };
  fab.addEventListener("click", () => (panel.classList.contains("open") ? close() : open()));
  const close = () => { panel.classList.remove("open"); panel.setAttribute("aria-hidden", "true"); };
  $("[data-chat-close]", panel).addEventListener("click", close);
  $$("[data-open-chat]").forEach((b) => b.addEventListener("click", () => {
    if (window.railStartDirect) { window.railStartDirect(teacher).catch((e) => toast(e.message, true)); return; }
    if (group) { conv = null; group = false; msgs.innerHTML = ""; } open();
  }));
  $$("[data-open-group]").forEach((b) => b.addEventListener("click", () => {
    if (window.railStartGroup && courseId) { window.railStartGroup(courseId).catch((e) => toast(e.message, true)); return; }
    openGroup();
  }));
  if (Auth.user) fab.hidden = true; // нэвтэрсэн хүнд баруун талын самбар байна
  intro.addEventListener("submit", async (e) => {
    e.preventDefault();
    try {
      const d = await api("/api/auth/guest", { method: "POST", body: { name: intro.name.value.trim() }, token: null });
      store.set("sg_guest", { token: d.token, name: d.guest.name });
      await start();
    } catch (err) { toast(err.message, true); }
  });
  Live.on((d) => {
    if (!conv) return;
    if (["reaction", "read", "typing"].includes(d.type)) { thread.event(d); return; }
    if (d.type !== "message" || d.message.conversation_id !== conv.id) return;
    thread.append(d.message);
    if (!panel.classList.contains("open")) $(".chat-fab-dot").hidden = false;
  });
  // #chat холбоосоор (нүүр хуудасны "Чат" жагсаалтаас) ирсэн бол шууд нээнэ.
  if (location.hash === "#chat=group" && courseId) { history.replaceState(null, "", location.pathname + location.search); window.railStartGroup ? window.railStartGroup(courseId).catch((e) => toast(e.message, true)) : openGroup(); }
  else if (location.hash === "#chat") { history.replaceState(null, "", location.pathname + location.search); window.railStartDirect ? window.railStartDirect(teacher).catch((e) => toast(e.message, true)) : open(); }
  // Өмнө нь чатласан бол шинэ хариуг сонсохын тулд шууд холбогдоно.
  if (Auth.chatToken && store.get("sg_chat:" + teacher)) start();
  panel.addEventListener("click", () => store.set("sg_chat:" + teacher, 1), { once: true });
}

/* ---------- профайл ---------- */
const hueOfName = (str) => hueOf(str);
const avatarHTML = (u, cls = "avatar-sm") => `<span class="avatar ${cls}" style="--h:${hueOfName(u.username || u.display_name || "")}">${u.avatar_url ? `<img src="${esc(u.avatar_url)}" alt="">` : esc((u.display_name || "?").trim().split(/\s+/).map((w) => w[0]).join("").slice(0, 2).toUpperCase())}</span>`;
const ringHTML = (score, cls = "") => `<div class="ring ${cls}" style="--p:${+score || 0}" role="img" aria-label="Профайлын бүрдэл ${+score || 0}%"><b>${+score || 0}%</b></div>`;
// localTimes: серверийн бичсэн цагийг үзэгчийн цагийн бүсээр солино.
function localTimes(root = document) { $$("time[data-dt]", root).forEach((t) => (t.textContent = fmtDate(t.dateTime))); }
async function share(title) {
  const url = location.origin + location.pathname;
  if (navigator.share) { try { await navigator.share({ title, url }); } catch {} return; }
  try { await navigator.clipboard.writeText(url); toast("Холбоос хуулагдлаа ✓"); } catch { toast(url); }
}

function profilePage() {
  const root = $(".pf");
  $$("[data-open-qr]").forEach((b) => b.addEventListener("click", () => openModal($("#qrModal"))));
  $$("[data-copy-link]").forEach((b) => b.addEventListener("click", async () => {
    try { await navigator.clipboard.writeText(location.origin + location.pathname); toast("Холбоос хуулагдлаа ✓"); } catch { toast(location.href); }
  }));
  $$("[data-share]").forEach((b) => b.addEventListener("click", () => share(b.dataset.title)));
  localTimes();

  /* Таб: агуулгыг сольж, доогуур зураас нь гулсана. Холбоосоор (#about гэх мэт) шууд нээгдэнэ. */
  const tabs = $("#pfTabs"), ink = $(".pf-ink", tabs), tabBtns = $$(".pf-tab", tabs), panels = $$(".pf-panel");
  const moveInk = () => { const a = $(".pf-tab.active", tabs); if (a) { ink.style.width = a.offsetWidth + "px"; ink.style.transform = `translateX(${a.offsetLeft}px)`; } };
  // Эзэмшигчийн горимд зарим таб студийн удирдлагын хэсгийг (m-*) харуулна.
  const MANAGED = { overview: "overview", courses: "courses", books: "books", live: "live", files: "files", settings: "profile", students: "students" };
  let courseArg = null;
  const showTab = (name, scroll) => {
    const btn = tabBtns.find((b) => b.dataset.tab === name);
    if (!btn || (btn.hasAttribute("data-owner-tab") && !root.classList.contains("is-owner"))) return false;
    tabBtns.forEach((b) => { const on = b.dataset.tab === name; b.classList.toggle("active", on); b.setAttribute("aria-selected", on); });
    const manage = root.classList.contains("is-owner") && MANAGED[name];
    const want = manage ? "m-" + name : name;
    panels.forEach((p) => (p.hidden = p.dataset.panel !== want));
    root.classList.toggle("manage", !!manage);
    if (manage && window.Studio) {
      const box = $(`[data-panel="m-${name}"]`);
      if (name === "courses" && courseArg) window.Studio.mount(box, "course", courseArg);
      else window.Studio.mount(box, MANAGED[name]);
    }
    moveInk(); reveals();
    $(".pf-tab.active", tabs)?.scrollIntoView({ block: "nearest", inline: "center" });
    if (scroll && root.classList.contains("pf-stuck")) scrollTo({ top: $(".pf-body").offsetTop - 60, behavior: "smooth" });
    return true;
  };
  tabs.addEventListener("click", (e) => {
    const b = e.target.closest(".pf-tab"); if (!b) return;
    if (b.dataset.tab === "chat") { window.openRailChat?.(); return; } // чат баруун самбарт нээгдэнэ
    courseArg = null; showTab(b.dataset.tab, true); history.replaceState(null, "", "#" + b.dataset.tab);
  });
  root.addEventListener("click", (e) => {
    const el = e.target.closest("[data-quick]"); if (!el) return;
    if (el.dataset.quick === "chat") { e.preventDefault(); window.openRailChat?.(); }
    if (el.dataset.quick === "new-course") { try { sessionStorage.setItem("sg_new_course", "1"); } catch {} if (location.hash === "#courses") fromHash(); }
  });
  $$("[data-tab-go]").forEach((a) => a.addEventListener("click", (e) => { e.preventDefault(); showTab(a.dataset.tabGo); $(".pf-body").scrollIntoView({ behavior: "smooth" }); }));
  // Хаяг (#courses, #course=<id>, #settings …) студийн хэсгүүдтэй ижил үгсээр ажиллана.
  const fromHash = () => {
    const h = new URLSearchParams(location.hash.slice(1));
    if (h.has("course")) { courseArg = h.get("course"); return showTab("courses"); }
    if (h.has("chat")) { // мэдэгдлээс ирсэн: яриаг баруун самбарт нээнэ
      const id = h.get("chat") || undefined, go = () => window.openRailChat?.(id);
      window.openRailChat ? go() : addEventListener("studio-ready", go, { once: true });
      return showTab("overview") || showTab("courses");
    }
    if (h.has("meet")) { toast(h.get("meet") === "connected" ? "✓ Google Meet холбогдлоо" : "Google Meet холбож чадсангүй", h.get("meet") !== "connected"); return showTab("live"); }
    courseArg = null;
    const key = [...h.keys()][0] || "";
    const owner = root.classList.contains("is-owner");
    return key === "profile" || (owner && key === "about") ? showTab("settings") : owner && key === "qr" ? showTab("overview") : showTab(key);
  };
  root._showTab = showTab; root._fromHash = fromHash;
  addEventListener("hashchange", () => fromHash());
  fromHash() || moveInk();
  addEventListener("resize", moveInk);
  document.fonts?.ready.then(moveInk);
  // Нэр, товчнууд дэлгэцээс гармагц таб цэс дээд талд наалдана.
  const head = $(".pf-head");
  new IntersectionObserver(([e]) => {
    const stuck = !e.isIntersecting && e.boundingClientRect.top < 0;
    head.style.paddingBottom = stuck ? tabs.offsetHeight + "px" : ""; // цэс урсгалаас гарахад агуулга үсрэхгүй
    root.classList.toggle("pf-stuck", stuck); $(".nav")?.toggleAttribute("hidden", stuck); moveInk();
  }, { threshold: 0 }).observe($(".pf-bar"));

  // Урт танилцуулга: 5 мөрөөс хэтэрвэл "Цааш унших".
  const bio = $("#pfBio"), more = $("#pfBioMore");
  if (bio && more && bio.scrollHeight > bio.clientHeight + 2) {
    more.hidden = false;
    more.addEventListener("click", () => { const open = bio.classList.toggle("clamped"); more.textContent = open ? "Цааш унших" : "Хураах"; });
  }

  // Сургалтын шүүлтүүр + хайлт (4+ сургалттай үед л загварт гарна).
  const bar = $("#courseFilter"), cards = $$("#courseGrid .course-card");
  if (bar) {
    let mode = "all", q = "";
    const apply = () => {
      let shown = 0;
      cards.forEach((c) => {
        const ok = (mode === "all" || (mode === "free" ? +c.dataset.free > 0 : c.dataset.paid === "1")) && (!q || c.textContent.toLowerCase().includes(q));
        c.hidden = !ok; if (ok) { shown++; c.classList.add("in"); }
      });
      $("#courseNone").hidden = shown > 0;
    };
    bar.addEventListener("click", (e) => {
      const p = e.target.closest(".pill"); if (!p) return;
      mode = p.dataset.f; $$(".pill", bar).forEach((x) => x.classList.toggle("active", x === p)); apply();
    });
    $("input", bar)?.addEventListener("input", (e) => { q = e.target.value.trim().toLowerCase(); apply(); });
  }

  // Хувийн болгох: суралцагчид өөрийнх нь сургалтыг тэмдэглэнэ, эзэмшигчид засах хэрэгслийг нээнэ.
  if (!Auth.token) return;
  api("/api/me/home").then((h) => {
    const mine = new Map(h.courses.map((c) => [c.course.id, c]));
    cards.forEach((card) => {
      const m = mine.get(card.dataset.course); if (!m) return;
      $(".course-flags", card)?.insertAdjacentHTML("afterbegin", `<span class="flag flag-own">✓ ${m.access === "lessons" ? m.owned_lessons + " хичээл таных" : "Таных"}</span>`);
      const price = $(".price", card); if (price) { price.textContent = "▶ Үргэлжлүүлэх"; price.classList.add("free"); }
    });
    if (h.teacher && h.user.username === root.dataset.teacher) profileOwner(root, h);
  }).catch(() => {});
}

/* Эзэмшигч профайл дээрээ шууд засна: зураг солих, мэдээлэл засах, зочны нүдээр харах. */
function profileOwner(root, h) {
  const LINK_KEYS = ["website", "facebook", "instagram", "youtube"];
  const setMode = (owner) => {
    root.classList.toggle("is-owner", owner);
    $$("[data-owner]").forEach((el) => (el.hidden = !owner));
    $$("[data-visitor]").forEach((el) => (el.hidden = owner));
    $$("[data-owner-tab]").forEach((el) => (el.hidden = !owner));
    $$("[data-visitor-tab]").forEach((el) => (el.hidden = owner)); // эзэмшигчид: Тухай → Тохиргоо, QR → нүүр зураг дээр
    $$(".nav a[href='/me']").forEach((el) => (el.hidden = owner)); // энэ хуудас өөрөө тул давхардуулахгүй
    $("#chatFab").hidden = owner; // өөртэйгөө чатлахгүй
    $("#viewAsBar").hidden = owner;
    $(".pf-ink").style.width = "0"; requestAnimationFrame(() => dispatchEvent(new Event("resize")));
  };
  setMode(true);
  $$("[data-view-as]").forEach((b) => b.addEventListener("click", () => { setMode(!root.classList.contains("is-owner")); root._fromHash() || root._showTab("courses"); }));
  // Студийн хэсгүүд (тойм, сургалт, файл, шууд хичээл, тохиргоо, чат) энэ хуудсан дээр ачаалагдана.
  $$("[data-owner-tab]").forEach((b) => (b.hidden = false));
  if (!window.Studio) {
    const ver = new URL($("script[src*='app.js']").src).searchParams.get("v");
    const sc = document.createElement("script"); sc.src = "/static/studio.js?v=" + ver; sc.defer = true;
    addEventListener("studio-ready", () => root._fromHash() || root._showTab("courses"), { once: true });
    document.body.append(sc);
  } else root._fromHash() || root._showTab("courses");

  const ins = h.teacher.insights, next = ins.tips.find((t) => !t.done);
  $("#ownerCard").innerHTML = `<div class="owner-strength">${ringHTML(ins.score, "ring-sm")}<div class="grow"><strong>Профайлын бүрдэл · ${esc(ins.level)}</strong>
    <span class="muted small">${next ? "Дараагийн алхам: " + esc(next.title) : "Бүрэн бүрдсэн байна 🎉"}</span></div></div>
    ${next ? `<p class="muted small" style="margin:10px 0 0">${esc(next.hint)}</p>` : ""}`;

  // PUT нь бүх талбарыг солидог тул одоогийн утгууд дээр өөрчлөлтөө давхарлаж илгээнэ.
  const save = async (patch) => {
    const u = await api("/api/me");
    const body = { display_name: u.display_name, headline: u.headline || "", bio: u.bio || "", avatar_url: u.avatar_url || "", cover_url: u.cover_url || "",
      subjects: u.subjects || [], location: u.location || "", links: u.links || {}, ...patch };
    const me = await api("/api/me/profile", { method: "PUT", body });
    Auth.set(Auth.token, me);
    return me;
  };

  const file = $("#pfFile");
  let kind = null;
  $$("[data-upload]").forEach((b) => b.addEventListener("click", () => { kind = b.dataset.upload; file.value = ""; file.click(); }));
  file.addEventListener("change", async () => {
    const f = file.files[0]; if (!f || !kind) return;
    const target = kind === "cover" ? $("#pfCover") : $("#pfAvatar");
    const preview = URL.createObjectURL(f), old = $("img", target)?.src;
    const paint = (src) => { $("img", target)?.remove(); if (kind === "avatar") target.textContent = ""; target.insertAdjacentHTML("afterbegin", `<img src="${esc(src)}" alt="">`); };
    paint(preview); root.classList.add("uploading"); // шууд урьдчилан харуулна
    try {
      const fd = new FormData(); fd.append("file", f);
      const info = await api("/api/me/files?visibility=public", { method: "POST", body: fd });
      await save({ [kind + "_url"]: info.path });
      paint(info.path);
      if (kind === "avatar") $$(".pf-mini .avatar, .composer .avatar, .chat-head .avatar").forEach((a) => (a.innerHTML = `<img src="${esc(info.path)}" alt="">`));
      toast(kind === "cover" ? "Нүүр зураг солигдлоо ✓" : "Профайл зураг солигдлоо ✓");
    } catch (e) {
      toast(e.message, true);
      if (old) paint(old); else location.reload();
    } finally { root.classList.remove("uploading"); URL.revokeObjectURL(preview); }
  });

  const modal = $("#editModal"), form = $("#editForm"), err = $("#editErr");
  $$("[data-edit-profile]").forEach((b) => b.addEventListener("click", async () => {
    err.textContent = "";
    try {
      const u = await api("/api/me");
      form.display_name.value = u.display_name; form.headline.value = u.headline || ""; form.bio.value = u.bio || "";
      form.subjects.value = (u.subjects || []).join(", "); form.location.value = u.location || "";
      LINK_KEYS.forEach((k) => (form["link_" + k].value = (u.links || {})[k] || ""));
      openModal(modal); form.display_name.focus();
    } catch (e) { toast(e.message, true); }
  }));
  form.addEventListener("submit", async (e) => {
    e.preventDefault(); err.textContent = "";
    const btn = $("button.btn-gold", form); btn.disabled = true;
    try {
      await save({ display_name: form.display_name.value, headline: form.headline.value, bio: form.bio.value, location: form.location.value,
        subjects: form.subjects.value.split(/[,\n]/).map((x) => x.trim()).filter(Boolean),
        links: Object.fromEntries(LINK_KEYS.map((k) => [k, form["link_" + k].value.trim()]).filter(([, v]) => v)) });
      location.reload(); // хуудас серверээс шинээр рендерлэгдэнэ (кэш аль хэдийн цэвэрлэгдсэн)
    } catch (x) { err.textContent = x.message; btn.disabled = false; }
  });
}

/* ---------- нүүр хуудас: нэвтэрсэн хэрэглэгчийн самбар ---------- */
async function homePage() {
  const dash = $("#dash");
  if (!dash || !Auth.token) return;
  let h;
  try { h = await api("/api/me/home"); }
  catch (e) {
    if (e.status === 401) { authNav(); return; } // токен хүчингүй — танилцуулга хуудас руу буцна
    dash.innerHTML = `<div class="empty">${esc(e.message)} <button class="btn btn-ghost btn-sm" onclick="location.reload()">Дахин оролдох</button></div>`;
    return;
  }
  const u = h.user; Auth.set(Auth.token, u);
  const hr = new Date().getHours();
  const hello = hr < 5 ? "Шөнийн мэнд" : hr < 12 ? "Өглөөний мэнд" : hr < 18 ? "Өдрийн мэнд" : "Оройн мэнд";
  const now = Date.now();
  const until = (t) => {
    const m = Math.round((new Date(t) - now) / 60000);
    if (m <= 0) return "Одоо явагдаж байна";
    if (m < 60) return m + " мин дараа";
    if (m < 1440) return Math.floor(m / 60) + " ц " + (m % 60) + " мин дараа";
    return Math.floor(m / 1440) + " хоногийн дараа";
  };
  const accessLabel = { full: "Бүх хичээл нээлттэй", enrolled: "Элссэн", lessons: "" };
  const last = store.get("sg_last");
  const resume = last && h.courses.find((c) => c.course.id === last.course);
  const SEC_ICO = { "my-rank": "medal", "my-study": "chart", "my-tasks": "exam", "my-pending": "money", "my-studio": "gear", "my-live": "live", "my-courses": "courses", "my-lessons": "play", "my-teachers": "teacher", "my-chats": "chat" };
  const sec = (id, title, body, extra = "") => `<section class="dash-sec" id="${id}"><header><h2>${SEC_ICO[id] ? `<span class="sec-ico">${icon(SEC_ICO[id], 17)}</span>` : ""}${title}</h2>${extra}</header>${body}</section>`;
  const courseCard = (c, i) => `<a class="course-card tilt" href="/c/${esc(c.course.id)}" style="--h:${hueOfName(c.course.title)}">
    <div class="course-art"><div class="course-flags"><span class="flag flag-own">✓ ${c.access === "lessons" ? c.owned_lessons + " хичээл" : accessLabel[c.access]}</span></div><span class="course-num">${String(i + 1).padStart(2, "0")}</span><span class="glare"></span></div>
    <div class="course-body"><h3>${esc(c.course.title)}</h3>${c.teacher ? `<div class="course-by">${avatarHTML(c.teacher)}${esc(c.teacher.display_name)}</div>` : ""}
    <div class="course-meta"><span>${c.course.lesson_count} хичээл</span><span class="price free">▶ Үргэлжлүүлэх</span></div></div></a>`;

  const tp = h.teacher;
  const parts = [`<div class="dash-head">${avatarHTML(u, "avatar-md")}<div class="grow"><h1>${hello}, ${esc(u.display_name.split(" ")[0])}</h1>
      <p>${h.courses.length ? `Танд ${h.courses.length} сургалт${h.meetings.length ? `, ${h.meetings.length} шууд хичээл` : ""} байна.` : tp ? "Багшийн самбар болон суралцах хэсэг тань энд байна." : "Багшийнхаа профайлаас анхны сургалтаа сонгоорой."}</p></div>
      ${tp ? `<a class="btn" href="/t/${esc(u.username)}#overview">Багшлах →</a>` : `<button class="btn" data-student-settings>Тохиргоо</button>`}</div>`,
    `<div class="tiles">
      <a class="tile" href="#my-courses"><em>${icon("courses")}</em><b>${h.courses.length}</b><span>Миний сургалт</span></a>
      <a class="tile" href="#my-lessons"><em>${icon("play")}</em><b>${h.lessons.length}</b><span>Авсан хичээл</span></a>
      <a class="tile" href="#my-live"><em>${icon("live")}</em><b>${h.meetings.length}</b><span>Шууд хичээл</span></a>
      <a class="tile" href="#my-chats"><em>${icon("chat")}</em><b>${h.chats.length}</b><span>Чат</span></a>
      <a class="tile" href="#my-teachers"><em>${icon("teacher")}</em><b>${h.teachers.length}</b><span>Миний багш</span></a>
      ${h.rank ? `<a class="tile tile-rank" href="#my-rank"><em class="rank-shine" data-level="${h.rank.level}">${esc(h.rank.insignia)}</em><b>${esc(h.rank.name)}</b><span>Миний цол · ${h.rank.points} оноо</span></a>` : ""}
      ${h.tasks?.length ? `<a class="tile" href="#my-tasks"><em>${icon("exam")}</em><b>${h.tasks.filter((t) => ["open", "need_pay"].includes(t.status)).length}</b><span>Хийх даалгавар, шалгалт</span></a>` : ""}
    </div>`];
  // Цол: систем шинэ цол олгосон бол баяр хүргэж салют буудуулна.
  if (h.rank) {
    const lvlKey = "sg_rank_lvl_" + (u.id || "u");
    let seen = -1; try { seen = localStorage.getItem(lvlKey) === null ? -1 : +localStorage.getItem(lvlKey); } catch {}
    if (h.rank_awarded || (seen >= 0 && h.rank.level > seen)) setTimeout(() => rankSalute(h.rank), 400);
    try { localStorage.setItem(lvlKey, h.rank.level); } catch {}
  }

  if (resume) parts.push(`<div class="resume"><span class="item-ico">${icon("play", 20)}</span><div class="grow"><span class="eyebrow">Үргэлжлүүлэх</span><strong>${esc(last.lessonTitle || resume.course.title)}</strong><span class="muted small">${esc(resume.course.title)}</span></div>
      <a class="btn btn-gold" href="/c/${esc(last.course)}${last.lesson ? "#l=" + esc(last.lesson) : ""}">Үзэх</a></div>`);

  if (h.rank) {
    const rk = h.rank, left = rk.next ? rk.next - rk.points : 0, ladder = h.rank_ladder || [];
    // Урам зоригийн үг: явцаас хамаарч өөрчлөгдөнө.
    const cheer = !rk.next ? "Дээд цол! Та энэ сургалтын жинхэнэ генерал. 🫡" : rk.progress >= 80 ? `Бараг боллоо! Ердөө ${left} оноо дутуу — өнөөдөр нэг хичээл дуусгавал хүрнэ. 🔥` : rk.progress >= 40 ? `Сайн явж байна — замын тал нь ардаа үлдлээ. ${left} оноо үлдсэн. 💪` : rk.points > 0 ? `Сайн эхлэл! Хичээл бүр оноо нэмнэ — ${left} оноонд дараагийн цол. 🚀` : "Анхны хичээлээ дуусгаад анхны цолоо аваарай! 🎯";
    const ring = `<div class="rank-ring" style="--p:${rk.progress}"><div class="rank-ring-in"><span class="rank-shine" data-level="${rk.level}">${esc(rk.insignia)}</span></div></div>`;
    const steps = ladder.map((st, i) => `<li class="${i < rk.level ? "done" : i === rk.level ? "cur" : i === rk.level + 1 ? "next" : ""}" title="${esc(st.name)} · ${st.points} оноо"><b class="${i <= rk.level ? "rank-shine" : ""}" data-level="${i}">${esc(st.insignia)}</b><span>${esc(st.name)}</span><small>${st.points}</small></li>`).join("");
    const tipLine = (t) => { const m = /\(\+(\d+)/.exec(t); return `<li><i>${m ? "+" + m[1] : "✓"}</i><span>${esc(t.replace(/\s*\(\+.*$/, ""))}</span></li>`; };
    parts.push(sec("my-rank", "Миний цол", `<div class="rank-hero">
      <div class="rank-hero-top">${ring}
        <div class="grow"><span class="eyebrow">Нэгдсэн цол · бүх хичээлээ нэгтгэж систем олгоно</span><strong class="rank-title">${esc(rk.name)}</strong>
          <p class="rank-cheer">${cheer}</p>
          <div class="rank-stats"><span><b>${rk.points}</b> оноо</span>${rk.integration?.bonus ? `<span><b>+${rk.integration.bonus}</b> интеграц</span>` : ""}<span><b>${rk.honest}</b> шударга хичээл</span>${rk.integration?.sections_all ? `<span><b>${rk.integration.sections}/${rk.integration.sections_all}</b> бүлэг бүтэн</span>` : ""}${rk.cheated ? `<span class="bad"><b>${rk.cheated}</b> тооцогдоогүй</span>` : ""}${rk.next ? `<span><b>${left}</b> оноо дараагийн цолд</span>` : ""}</div>
          ${rk.next ? `<div class="rank-next"><div class="meter"><i style="width:${rk.progress}%"></i></div><span>${esc(rk.name)} <em>→</em> ${esc(rk.next_name)} <b>${rk.progress}%</b></span></div>` : ""}</div></div>
      ${ladder.length ? `<div class="rank-ladder-wrap"><ol class="rank-ladder">${steps}</ol></div>` : ""}
      <div class="rank-grid">
        ${rk.tips?.length ? `<div class="rank-box rank-todo"><b>Дараагийн алхам — оноо нэмэх</b><ul>${rk.tips.filter((t) => !t.startsWith("дараагийн")).map(tipLine).join("")}</ul></div>` : ""}
        ${h.course_ranks.length ? `<div class="rank-box"><b>Сургалт бүрээр</b><ul class="rank-courses">${h.course_ranks.map((c) => `<li><a href="/c/${esc(c.course_id)}"><span class="rank-shine" data-level="${c.rank.level}">${esc(c.rank.insignia)}</span><span class="grow"><strong>${esc(c.title)}</strong><small>${esc(c.rank.name)} · ${c.rank.points} оноо${c.rank.integration?.bonus ? ` (интеграц +${c.rank.integration.bonus})` : ""} · ${c.lessons.length} хичээл${c.rank.integration?.courses ? " · бүтэн ✓" : ""}</small></span><span class="chip chip-teal">Үргэлжлүүлэх</span></a></li>`).join("")}</ul></div>` : ""}
      </div></div>`));
  }
  // 📊 Миний суралцсан байдал: бүх хичээлээр (сургалт бүрийн цолын мэдээллээс)
  const studied = (h.course_ranks || []).flatMap((c) => c.lessons || []);
  if (studied.length) {
    const sum = (f) => studied.reduce((a, l) => a + (f(l) || 0), 0);
    const active = sum((l) => l.active_sec), done = studied.filter((l) => l.completed).length, qT = sum((l) => l.quiz_total), qC = sum((l) => l.quiz_correct);
    const vids = studied.filter((l) => l.has_video), vPct = vids.length ? Math.round(sum((l) => l.has_video ? l.video_pct : 0) / vids.length) : 0;
    const refl = studied.filter((l) => l.reflected).length, honest = studied.filter((l) => !l.disqualified).length;
    const lessonsTotal = (h.courses || []).reduce((a, c) => a + (c.course.lesson_count || 0), 0);
    const pct = (a, b) => (b ? Math.round(a / b * 100) : 0);
    const bar = (v, cls = "") => `<span class="st-bar ${cls}"><i style="width:${Math.max(0, Math.min(100, v))}%"></i></span>`;
    const state = (l) => l.disqualified ? `<span class="st-chip bad">⛔ Тооцогдоогүй</span>` : l.completed ? `<span class="st-chip ok">✓ Дууссан</span>` : `<span class="st-chip">Үзэж байна</span>`;
    // Сургалт бүрээр бүлэглэнэ: сүүлд судалснаас нь эхэлж, толгой дээр дарахад хичээлүүд нь дэлгэрнэ.
    const groups = (h.course_ranks || []).filter((c) => c.lessons?.length).map((c) => {
      const ls = c.lessons, s = (f) => ls.reduce((a, l) => a + (f(l) || 0), 0), vids = ls.filter((l) => l.has_video);
      return { c, ls, total: (h.courses || []).find((x) => x.course.id === c.course_id)?.course.lesson_count || ls.length,
        done: ls.filter((l) => l.completed).length, active: s((l) => l.active_sec), qT: s((l) => l.quiz_total), qC: s((l) => l.quiz_correct),
        vPct: vids.length ? Math.round(s((l) => (l.has_video ? l.video_pct : 0)) / vids.length) : -1, cheat: ls.filter((l) => l.disqualified).length,
        last: ls.reduce((m, l) => (l.last_at && (!m || new Date(l.last_at) > new Date(m)) ? l.last_at : m), null) };
    }).sort((a, b) => new Date(b.last || 0) - new Date(a.last || 0));
    let openSet = new Set(); try { openSet = new Set(JSON.parse(localStorage.getItem("sg_study_open") || "[]")); } catch {}
    const row = (l) => `<tr><td><a href="/c/${esc(l.course_id)}#l=${esc(l.lesson_id)}"><strong>${esc(l.title)}</strong><small>${l.last_at ? fmtDate(l.last_at) : ""}</small></a></td>
          <td>${state(l)}</td>
          <td><b>${dur(l.active_sec)}</b>${l.total_sec ? `<small>${pct(l.active_sec, l.total_sec)}% идэвхтэй${l.tab_switches ? ` · ${l.tab_switches} таб` : ""}</small>` : ""}</td>
          <td>${l.quiz_total ? `<b>${l.quiz_correct}/${l.quiz_total}</b>${bar(pct(l.quiz_correct, l.quiz_total), l.quiz_correct === l.quiz_total ? "ok" : "")}` : `<span class="muted">—</span>`}</td>
          <td>${l.has_video ? `<b>${l.video_pct}%</b>${bar(l.video_pct, l.video_pct >= 90 ? "ok" : "")}` : `<span class="muted">—</span>`}</td>
          <td><span class="st-pts ${l.disqualified ? "bad" : l.points >= 75 ? "ok" : ""}" title="${esc((l.reasons || []).join(", "))}"><b>${l.disqualified ? "0" : "+" + l.points}</b><small>${l.disqualified ? "тооцогдоогүй" : "оноо"}</small></span></td></tr>`;
    const group = (g) => { const id = g.c.course_id, on = openSet.has(id), rk = g.c.rank || {};
      return `<div class="st-course ${on ? "open" : ""}" data-cid="${esc(id)}">
        <button type="button" class="st-ch" aria-expanded="${on}" aria-controls="stc-${esc(id)}">
          <span class="st-ch-ico" aria-hidden="true">${esc((g.c.title || "?").trim().charAt(0).toUpperCase())}</span>
          <span class="st-ch-main"><strong>${esc(g.c.title)}</strong><small>${g.done}/${g.total} хичээл дууссан · ${g.ls.length} хичээл судалсан${g.last ? " · сүүлд " + fmtDate(g.last) : ""}</small>${bar(pct(g.done, g.total), g.done && g.done === g.total ? "ok" : "")}</span>
          <span class="st-ch-stats"><span><b>${dur(g.active)}</b><small>идэвхтэй</small></span><span><b>${g.qT ? pct(g.qC, g.qT) + "%" : "—"}</b><small>асуултад зөв</small></span><span><b>${g.vPct >= 0 ? g.vPct + "%" : "—"}</b><small>видео</small></span><span class="${g.cheat ? "bad" : ""}"><b>${g.ls.length - g.cheat}/${g.ls.length}</b><small>шударга</small></span></span>
          <span class="st-ch-rank" title="Энэ сургалтын цол"><b>${rk.points || 0}</b><small>${esc(rk.name || "оноо")}</small></span>
          <span class="st-ch-chev" aria-hidden="true">${icon("chev", 18)}</span>
        </button>
        <div class="st-cbody" id="stc-${esc(id)}" ${on ? "" : "hidden"}>
          <div class="st-table-wrap"><table class="st-table"><thead><tr><th>Хичээл</th><th>Төлөв</th><th>Идэвхтэй</th><th>Асуулга</th><th>Видео</th><th>Нэмсэн оноо</th></tr></thead><tbody>${g.ls.map(row).join("")}</tbody></table></div>
          <a class="st-go" href="/c/${esc(id)}">Сургалт руу орох ${icon("next", 14)}</a>
        </div></div>`; };
    parts.push(sec("my-study", "Миний суралцсан байдал", `<div class="study">
      <div class="st-tiles">
        <div class="st-tile"><small>Идэвхтэй суралцсан</small><b>${dur(active)}</b><span>${studied.length} хичээлд · ${sum((l) => l.sessions)} удаа</span></div>
        <div class="st-tile"><small>Дуусгасан хичээл</small><b>${done}<em>/${lessonsTotal || studied.length}</em></b>${bar(pct(done, lessonsTotal || studied.length))}</div>
        <div class="st-tile"><small>Асуултад зөв</small><b>${qT ? pct(qC, qT) + "%" : "—"}</b><span>${qC}/${qT} асуулт</span></div>
        <div class="st-tile"><small>Видео үзэлт</small><b>${vids.length ? vPct + "%" : "—"}</b><span>${vids.length} видео хичээл</span></div>
        <div class="st-tile"><small>Дүгнэлт бичсэн</small><b>${refl}</b><span>хичээлд</span></div>
        <div class="st-tile"><small>Шударга суралцсан</small><b>${honest}<em>/${studied.length}</em></b><span>хуулах оролдлогогүй</span></div>
      </div>
      <div class="st-courses-head"><b>Сургалт бүрээр</b><span class="muted small">${groups.length} сургалт · дээр нь дарж хичээл бүрийг харна</span>${groups.length > 1 ? `<button type="button" class="link st-all" data-st-all>${openSet.size >= groups.length ? "Бүгдийг хумих" : "Бүгдийг дэлгэх"}</button>` : ""}</div>
      <div class="st-courses">${groups.map(group).join("")}</div></div>`));
    if (!window._stBound) { // нэг л удаа: сургалтын толгой дарахад дэлгэрнэ/хумигдана, нээлттэйг санана
      window._stBound = true;
      const save = () => { try { localStorage.setItem("sg_study_open", JSON.stringify([...$$(".st-course.open")].map((x) => x.dataset.cid))); } catch {} };
      const toggle = (el, on) => { el.classList.toggle("open", on); $(".st-ch", el).setAttribute("aria-expanded", on); $(".st-cbody", el).hidden = !on; };
      document.addEventListener("click", (e) => {
        const ch = e.target.closest(".st-ch");
        if (ch) { const el = ch.closest(".st-course"); toggle(el, !el.classList.contains("open")); save(); return; }
        const all = e.target.closest("[data-st-all]");
        if (all) { const els = $$(".st-course"), on = els.some((x) => !x.classList.contains("open")); els.forEach((x) => toggle(x, on)); all.textContent = on ? "Бүгдийг хумих" : "Бүгдийг дэлгэх"; save(); }
      });
    }
  }
  if (h.tasks?.length) {
    const label = (t) => ({ open: ["Хийх", "chip-amber"], not_started: [`${fmtDate(t.due.start_at)}-д эхэлнэ`, ""], need_pay: [(t.due.need_late ? "Хоцорсон · " : "Төлбөртэй · ") + money(t.due.fee) + " төлж нээнэ", "chip-amber"], closed: ["Хаалттай", ""], submitted: ["Илгээсэн · дүгнэхийг хүлээж байна", "chip-teal"],
      graded: [`Дүн: ${t.score}/${t.max_score || 100}`, "chip-teal"], passed: [`Тэнцсэн · ${t.exam_best}%`, "chip-teal"], failed: [`Тэнцээгүй · шилдэг ${t.exam_best}%`, "chip-amber"] })[t.status] || ["", ""];
    parts.push(sec("my-tasks", "Даалгавар, шалгалт", `<ul class="items">${h.tasks.map((t) => { const [txt, cls] = label(t); return `
      <li><a class="item ${["open", "need_pay"].includes(t.status) && t.due.at ? "warn" : ""}" href="/c/${esc(t.course_id)}#l=${esc(t.lesson_id)}"><span class="item-ico">${icon(t.kind === "exam" ? "exam" : "clip", 20)}</span><span class="grow"><strong>${esc(t.title)}</strong><small>${esc(t.course_title)}${t.due.at ? ` · ${t.due.late ? "хугацаа дууссан" : "хугацаа"}: ${fmtDate(t.due.at)}` : " · хугацаагүй"}${t.feedback ? ` · ${esc(t.feedback)}` : ""}</small></span><span class="chip ${cls}">${txt}</span></a></li>`; }).join("")}</ul>`));
  }
  if (h.pending.length) parts.push(sec("my-pending", "Төлбөр хүлээгдэж буй", `<ul class="items">${h.pending.map((o) => `
      <li><a class="item warn" href="/c/${esc(o.course_id)}${o.lesson_id ? "#l=" + esc(o.lesson_id) : ""}"><span class="item-ico">${icon("money", 20)}</span><span class="grow"><strong>${esc(o.title)}</strong><small>${fmtDate(o.created_at)} · ${o.kind === "lesson" ? "нэг хичээл" : "бүтэн сургалт"}</small></span><strong>${money(o.amount)}</strong><span class="chip chip-amber">Төлөх →</span></a></li>`).join("")}</ul>`));

  if (tp) {
    const ins = tp.insights, todo = ins.tips.filter((t) => !t.done).slice(0, 3);
    parts.push(sec("my-studio", "Багшийн самбар", `<div class="panel"><div class="strength">${ringHTML(ins.score)}<div><strong>Профайлын бүрдэл · ${esc(ins.level)}</strong>
        <p class="muted small" style="margin:.2em 0 0">${tp.published}/${tp.courses} сургалт нийтлэгдсэн · ${tp.students} суралцагч · ${tp.profile_views} профайл үзэлт</p>
        <div class="hero-cta" style="margin-top:12px"><a class="btn btn-gold btn-sm" href="/t/${esc(u.username)}#overview">Удирдлага нээх</a><a class="btn btn-ghost btn-sm" href="/t/${esc(u.username)}">Нээлттэй профайл</a><a class="btn btn-ghost btn-sm" href="/t/${esc(u.username)}#chat">Чат</a></div></div></div>
        ${todo.length ? `<ul class="tips">${todo.map((t) => `<li><a class="tip" href="/t/${esc(u.username)}${esc(t.link)}"><i>✓</i><span><strong>${esc(t.title)}</strong><small>${esc(t.hint)}</small></span><span class="chip chip-gold">Хийх →</span></a></li>`).join("")}</ul>` : ""}</div>`));
  }

  if (h.meetings.length) parts.push(sec("my-live", "Шууд хичээл", `<ul class="items">${h.meetings.map((m) => `
      <li><div class="item live"><span class="item-ico">${icon("live", 20)}</span><span class="grow"><strong>${esc(m.title)}</strong><small>${fmtDate(m.starts_at)} · ${m.duration_min} мин · ${esc(m.course_title)}${m.price ? ` · ${m.bought ? "✓ худалдаж авсан" : money(m.price)}` : ""}</small></span><span class="chip chip-amber">${until(m.starts_at)}</span>
      ${m.meet_url ? `<a class="btn btn-accent btn-sm" href="${esc(m.meet_url)}" target="_blank" rel="noopener">Нэгдэх</a>`
        : m.price ? `<a class="btn btn-gold btn-sm" href="/c/${esc(m.course_id)}#meet=${esc(m.id)}">Худалдаж авах · ${money(m.price)}</a>`
        : `<a class="btn btn-ghost btn-sm" href="/c/${esc(m.course_id)}">Сургалт</a>`}</div></li>`).join("")}</ul>`));
  else parts.push(`<span id="my-live"></span>`);

  parts.push(sec("my-courses", "Миний сургалтууд", h.courses.length ? `<div class="course-grid">${h.courses.map(courseCard).join("")}</div>`
    : `<div class="empty">Та одоогоор сургалтад элсээгүй байна.<br>Багшийнхаа хэрэглэгчийн нэрийг оруулаад профайл руу нь ороорой — эсвэл QR кодыг нь уншуулаарай.
       <form class="find" id="findTeacher"><input name="u" required pattern="[A-Za-z0-9_]{3,32}" placeholder="Багшийн нэр, жишээ: demo" aria-label="Багшийн хэрэглэгчийн нэр"><button class="btn btn-gold">Профайл нээх</button></form></div>`));

  if (h.lessons.length) parts.push(sec("my-lessons", "Дангаар авсан хичээлүүд", `<ul class="items">${h.lessons.map((l) => `
      <li><a class="item" href="/c/${esc(l.course_id)}#l=${esc(l.lesson_id)}"><span class="item-ico">${icon("play", 20)}</span><span class="grow"><strong>${esc(l.title)}</strong><small>${l.paid_at ? fmtDate(l.paid_at) + "-д авсан" : ""}</small></span><span class="chip chip-teal">Үзэх</span></a></li>`).join("")}</ul>`));
  else parts.push(`<span id="my-lessons"></span>`);

  const teachers = h.teachers.length ? sec("my-teachers", "Миний багш нар", `<div class="people">${h.teachers.map((t) => `
      <a class="person" href="/t/${esc(t.username)}">${avatarHTML(t)}<span class="grow"><strong>${esc(t.display_name)}</strong><small>${esc(t.headline || "@" + t.username)}</small></span></a>`).join("")}</div>`) : `<span id="my-teachers"></span>`;
  const chats = h.chats.length ? sec("my-chats", "Чат", `<ul class="items">${h.chats.map((c) => `
      <li><a class="item" href="${c.kind === "group" ? `/c/${esc(c.course_id)}#chat=group` : c.teacher?.username && !c.kind ? `/t/${esc(c.teacher.username)}#chat` : "#my-chats"}" data-rail-open="${esc(c.id)}">${c.kind === "group" || c.kind === "team" ? `<span class="avatar avatar-sm group">${c.kind === "team" ? "🧑‍🤝‍🧑" : "👥"}</span>` : c.teacher ? avatarHTML(c.teacher) : `<span class="avatar avatar-sm">?</span>`}<span class="grow"><strong>${esc(c.kind === "group" || c.kind === "team" ? c.title : c.teacher?.display_name || c.title || "")}</strong><small>${c.kind === "group" ? "Бүлэг · " + esc(c.teacher?.display_name || "") + " · " : c.kind === "team" ? "Сурагчдын бүлэг · " : c.kind === "dm" ? "Ангийн найз · " : ""}${esc(c.last_message || "")}</small></span><time class="muted small">${fmtDate(c.last_message_at)}</time></a></li>`).join("")}</ul>`) : `<span id="my-chats"></span>`;
  parts.push(`<div class="dash-cols">${teachers}${chats}</div>`);

  dash.innerHTML = parts.join("") + (tp ? "" : `<div class="modal" id="stuModal"><div class="modal-card"><button class="icon-btn modal-x" data-close aria-label="Хаах">✕</button>
      <h3 class="h3">Тохиргоо</h3><form class="form" id="stuForm"><label>Нэр<input name="display_name" required maxlength="80" value="${esc(u.display_name)}"></label>
      <p class="muted small" style="margin:0">Имэйл: ${esc(u.email || "")}</p><p class="form-error" role="alert"></p>
      <div class="hero-cta" style="margin:0;justify-content:flex-end"><button type="button" class="btn btn-ghost" data-close>Болих</button><button class="btn btn-gold">Хадгалах</button></div></form></div></div>`);
  $("[data-student-settings]")?.addEventListener("click", () => openModal($("#stuModal")));
  $("#stuForm")?.addEventListener("submit", async (e) => {
    e.preventDefault();
    const f = e.target;
    try {
      const me = await api("/api/me/profile", { method: "PUT", body: { display_name: f.display_name.value, headline: u.headline || "", bio: u.bio || "", avatar_url: u.avatar_url || "", cover_url: u.cover_url || "",
        subjects: u.subjects || [], location: u.location || "", links: u.links || {} } });
      Auth.set(Auth.token, me); closeModal($("#stuModal")); toast("Хадгалагдлаа ✓"); homePage();
    } catch (x) { $(".form-error", f).textContent = x.message; }
  });
  dash.addEventListener("click", (e) => { const a = e.target.closest("[data-rail-open]"); if (a && window.openRailChat) { e.preventDefault(); window.openRailChat(a.dataset.railOpen); } });
  $("#findTeacher")?.addEventListener("submit", (e) => { e.preventDefault(); location.href = "/t/" + encodeURIComponent(e.target.u.value.trim().toLowerCase()); });
  // Хоосон хэсгийн хавтан дээр дарахад үсрэх газаргүй бол сургалтууд руу аваачна.
  $$(".tile", dash).forEach((t) => { const tgt = $(t.getAttribute("href")); if (!tgt?.classList.contains("dash-sec")) t.setAttribute("href", "#my-courses"); });
}

/* ---------- сургалтын хуудас ---------- */
// Цагийг хүний хэлээр: 0 → шууд, 24 → 1 хоног, 168 → 7 хоног, 720 → 1 сар.
function humanHours(h) {
  h = +h || 0;
  if (!h) return "шууд";
  if (h % 720 === 0) return h / 720 + " сар";
  if (h % 168 === 0) return h / 168 * 7 + " хоног";
  if (h % 24 === 0) return h / 24 + " хоног";
  return h + " цаг";
}
window.SG_humanHours = humanHours;
async function coursePage() {
  const root = $(".course-page");
  if (!root) return;
  const id = root.dataset.course, bundle = +root.dataset.price, mode = root.dataset.mode, buy = $("#buyBtn"), drip = root.dataset.drip === "1";
  let all = false, access = null;
  // Дараалсан нээлтийн төлөв, явцыг хичээл бүр дээр харуулна.
  const applyStates = (a) => {
    access = a;
    const states = a.states || {}, prog = a.progress || {};
    $$(".lesson").forEach((li) => {
      const st = states[li.dataset.lesson], p = prog[li.dataset.lesson], lbl = $(".lesson-state", li), blk = a.blocks?.[li.dataset.lesson];
      li.classList.toggle("is-blocked", !!blk);
      li.classList.toggle("is-done", !!p?.completed_at);
      const dripLocked = (st && !st.open && !li.classList.contains("is-free")) || !!blk;
      li.classList.toggle("is-drip", !!dripLocked);
      if (!lbl) return;
      if (dripLocked) {
        lbl.hidden = false; lbl.className = "lesson-state " + (st.reason === "timer" ? "timer" : "");
        lbl.textContent = blk ? `⛔ Сануулга хэтэрсэн тул хаагдсан${blk.until ? " · " + fmtDate(blk.until) + " хүртэл" : " · багш нээх хүртэл"}` : st.reason === "timer" ? `⏳ ${fmtDate(st.unlock_at)}-д нээгдэнэ`
          : st.reason === "quiz" ? `🔒 «${st.prev_title}» хичээлийн асуултуудад бүгдэд нь зөв хариулсны дараа нээгдэнэ (${st.quiz_left}/${st.quiz_total} үлдсэн)`
          : st.reason === "active" ? `🕒 «${st.prev_title}» хичээлийг дахиад ${st.active_left} мин идэвхтэй судлаарай`
          : st.reason === "exam" ? `📝 «${st.prev_title}» шалгалтад тэнцсэний дараа нээгдэнэ`
          : st.reason === "complete" ? `☑️ «${st.prev_title}» хичээлийг дуусгасны дараа нээгдэнэ`
          : st.reason === "manual" ? "🔐 Багш нээх хүртэл хүлээнэ үү"
          : `🔒 Эхлээд «${st.prev_title}» хичээлийг үзнэ үү`;
      } else if (p?.completed_at) { lbl.hidden = false; lbl.className = "lesson-state"; lbl.textContent = "✓ Дууссан · " + fmtDate(p.completed_at); }
      else if (drip && +li.dataset.unlock && li.dataset.always !== "1" && !li.classList.contains("is-free")) { lbl.hidden = false; lbl.className = "lesson-state"; lbl.textContent = `⏱ Өмнөхийг үзснээс ${humanHours(li.dataset.unlock)}-ийн дараа`; }
      else lbl.hidden = true;
    });
    // Бүлэг тус бүрийн явц
    $$(".lesson-group").forEach((g) => {
      const ls = $$(".lesson", g), done = ls.filter((li) => li.classList.contains("is-done")).length, el = $(".lg-prog", g);
      if (!el || !ls.length) return;
      el.hidden = !done; el.textContent = done === ls.length ? "✓ Бүлэг дууссан" : `${done}/${ls.length} дууссан`; el.classList.toggle("done", done === ls.length);
      const bought = ls.filter((li) => li.classList.contains("is-free") || li.classList.contains("unlocked")).length;
      g.classList.toggle("is-open-all", bought === ls.length);
    });
    const bar = $("#courseProgress");
    if (bar && a.total) { bar.hidden = false; $("i", bar).style.width = Math.round(a.done / a.total * 100) + "%"; $("span", bar).textContent = `${a.done}/${a.total} хичээл дууссан`; }
    // Цол хичээл бүрт биш: хичээл бүр нэгдсэн цолд оноо НЭМНЭ (хуулах оролдлоготой бол тооцогдохгүй).
    const ranks = a.ranks || {};
    $$(".lesson").forEach((li) => {
      const r = ranks[li.dataset.lesson]; let b = $(".lesson-rank", li);
      if (!r) { b?.remove(); return; }
      if (!b) { b = document.createElement("span"); b.className = "lesson-rank"; $(".lesson-title", li)?.append(b); }
      b.className = "lesson-rank " + (r.disqualified ? "disq" : r.points >= 75 ? "high" : "");
      b.title = (r.disqualified ? "Хуулах оролдлоготой тул оноо тооцогдоогүй" : "Нэгдсэн цолд нэмсэн оноо") + ((r.reasons || []).length ? ": " + r.reasons.join(", ") : "");
      b.textContent = r.disqualified ? "⛔ Тооцогдоогүй" : `+${r.points} оноо`;
    });
    if (a.rank && bar) {
      let rl = $("#courseRank");
      if (!rl) { rl = document.createElement("div"); rl.id = "courseRank"; rl.className = "course-rank"; bar.after(rl); }
      const rk = a.rank;
      rl.hidden = !rk.lessons;
      const tot = a.rank_total && a.rank_total.points > rk.points ? a.rank_total : null;
      const shown = tot || rk;
      // Шинэ цол: сервер дохио өгсөн эсвэл сүүлд харснаас түвшин дээшилсэн бол баяр хүргэж салют буудуулна.
      const lvlKey = "sg_rank_lvl_" + (Auth.user?.id || "u");
      let seen = -1; try { seen = localStorage.getItem(lvlKey) === null ? -1 : +localStorage.getItem(lvlKey); } catch {}
      if (a.rank_awarded || (seen >= 0 && shown.level > seen)) rankSalute(shown);
      try { localStorage.setItem(lvlKey, shown.level); } catch {}
      const ig = rk.integration || {};
      rl.innerHTML = `<span class="cr-sign" data-level="${rk.level}">${esc(rk.insignia)}</span>
        <span class="cr-main"><small class="cr-label">Сургалтын нэгдсэн цол</small><b class="rank-name">${esc(rk.name)}</b></span>
        <span class="cr-pts"><b>${rk.points}</b> оноо</span>
        <span class="cr-sum">Хичээлүүд ${ig.lesson_points ?? rk.points}${ig.bonus ? ` + интеграц ${ig.bonus}` : ""}${ig.sections_all ? ` · ${ig.sections}/${ig.sections_all} бүлэг бүтэн` : ""}${ig.courses ? " · сургалт бүтэн ✓" : ""}${rk.cheated ? ` · <span class="cr-bad">${rk.cheated} хичээл тооцогдоогүй</span>` : ""}</span>
        <span class="meter cr-meter"><i style="width:${rk.progress}%"></i></span>
        <span class="cr-next">${rk.next ? `Дараагийн «${esc(rk.next_name)}» цол ${rk.next} оноонд` : "Дээд цол"}${tot ? ` · бүх сургалтаар: <b>${esc(tot.name)}</b> ${tot.points} оноо` : ""}</span>
        ${rk.tips?.length ? `<ul class="cr-tips">${rk.tips.map((t) => `<li>${esc(t)}</li>`).join("")}</ul>` : ""}`;
    }
  };
  const refreshAccess = () => api(`/api/courses/${id}/access`).then((a) => { applyStates(a); return a; }).catch(() => null);
  // Асуулга/шалгалт амжилттай → access шинэчилнэ; дараагийн хичээл нээгдсэн бол автоматаар шилжинэ.
  const autoAdvance = async (force) => {
    const curRow = $(".lesson.open"), rowsBefore = lessonRows(), i = curRow ? rowsBefore.indexOf(curRow) : -1;
    const next = i >= 0 ? rowsBefore[i + 1] : null, wasLocked = next?.classList.contains("is-drip");
    await refreshAccess();
    if (!next || next.classList.contains("is-drip") || !canOpen(next)) return;
    if (!wasLocked && force !== true) return; // аль хэдийн нээлттэй байсан бол асуулт хариулахад үсрэхгүй
    toast("🎉 Амжилттай! Дараагийн хичээл нээгдлээ — шилжиж байна…");
    setTimeout(() => { if ($(".lesson.open") === curRow) expandRow(next, true); }, 1600);
  };
  document.addEventListener("sg:quiz-mastered", () => autoAdvance(false)); // асуулгыг дуусгамагц түгжээ шууд нээгдэж шилжинэ
  const unlockLesson = (lid) => {
    const li = $(`.lesson[data-lesson="${lid}"]`);
    if (!li || li.classList.contains("is-free")) return;
    li.classList.add("unlocked"); li.classList.remove("is-locked");
    const chip = $(".lesson-price", li); if (chip) { chip.textContent = "✓ Нээлттэй"; chip.className = "chip chip-gold lesson-price"; }
    const b = $("[data-play]", li); if (b) { b.className = "btn btn-sm btn-glass"; b.innerHTML = "▶ Үзэх"; }
  };
  const unlockAll = () => {
    all = true;
    buy.textContent = "✓ Бүх хичээл нээлттэй — үзэж эхлэх";
    $$(".lesson").forEach((l) => unlockLesson(l.dataset.lesson));
  };
  if (Auth.token) refreshAccess().then((a) => { if (!a) return; if (a.all) unlockAll(); a.lessons.forEach(unlockLesson); if (mode === "free" && a.enrolled) buy.textContent = "✓ Та элссэн — үзэж эхлэх"; });
  else if (drip) $$(".lesson:not(.is-free)").forEach((li) => { const lbl = $(".lesson-state", li); if (lbl && +li.dataset.unlock && li.dataset.always !== "1") { lbl.hidden = false; lbl.textContent = `⏱ Өмнөхийг үзснээс ${humanHours(li.dataset.unlock)}-ийн дараа`; } });
  // Товлосон шууд хичээлүүд (Google Meet): үнэгүй нь сургалтын дүрмээр, төлбөртэйг тусад нь худалдаж авна.
  const meetsHTML = (d) => `<div class="section-head reveal in" id="meetHead"><span class="eyebrow">Шууд хичээл</span></div><div class="meet-list" id="meetList">${d.meetings.map((m) => `
      <div class="meet-item" id="meet-${esc(m.id)}"><time>${fmtDate(m.starts_at)}</time><span class="grow" style="flex:1">${esc(m.title)} · ${m.duration_min} мин
        ${m.price ? `<span class="chip chip-amber">${m.bought ? "✓ Худалдаж авсан" : money(m.price)}${m.members_free && !m.bought ? " · элссэн бол үнэгүй" : ""}</span>` : ""}</span>
      ${m.meet_url ? `<a class="btn btn-accent btn-sm" href="${esc(m.meet_url)}" target="_blank" rel="noopener">Нэгдэх</a>`
        : m.price ? `<button class="btn btn-gold btn-sm" data-meet-buy="${esc(m.id)}" data-title="${esc(m.title)}">Худалдаж авах · ${money(m.price)}</button>`
        : `<span class="chip">🔒 Сургалтад элссэн хүмүүст</span>`}</div>`).join("")}</div>`;
  const loadMeets = () => api(`/api/courses/${id}/meetings`).then((d) => {
    $("#meetHead")?.remove(); $("#meetList")?.remove();
    if (!d.meetings.length) return;
    $(".lessons").parentElement.insertAdjacentHTML("afterbegin", meetsHTML(d));
    const want = /#meet=([\w-]+)/.exec(location.hash)?.[1], el = want && $("#meet-" + want);
    if (el) { el.scrollIntoView({ block: "center" }); el.classList.add("meet-focus"); }
  }).catch(() => {});
  loadMeets();
  document.addEventListener("click", async (e) => {
    const b = e.target.closest("[data-meet-buy]"); if (!b) return;
    if (needLogin()) return;
    b.disabled = true;
    try {
      const d = await api(`/api/meetings/${b.dataset.meetBuy}/buy`, { method: "POST" });
      if (d.unlocked) { await loadMeets(); return; }
      pay(d.order, d.payment, "Шууд хичээл: " + b.dataset.title, async () => { toast("🎉 Шууд хичээл нээгдлээ!"); celebrate(); await loadMeets(); }, { name: "Шууд хичээл" });
    } catch (err) { toast(err.message, true); } finally { b.disabled = false; }
  });

  const needLogin = () => { if (Auth.token) return false; location.href = "/login?next=" + encodeURIComponent(location.pathname); return true; };

  // Төлбөр: захиалга → «Та төлбөрөө төлнө үү» цонх (QR) → төлөгдмөгц автоматаар нээнэ.
  const pay = (order, payment, label, onPaid, extra) => payWindow(order, payment, label, onPaid, extra);
  const courseTitle = $(".course-page h1")?.textContent.trim() || "Сургалт";
  // Багцын сонголт: хичээл дангаар төлөх цонхонд «Бүх хичээл багцаар» гэж сольж болно.
  const bundleOpt = (lid) => bundle > 0 && !all ? {
    name: "Бүх хичээл багцаар", sub: `${lessonRows().length} хичээл бүгд нээгдэнэ`, amount: bundle, label: `«${courseTitle}» — бүх хичээл`,
    start: () => api(`/api/courses/${id}/enroll`, { method: "POST" }),
    onPaid: async () => { unlockAll(); toast("🎉 Бүх хичээл нээгдлээ!"); celebrate(); await refreshAccess(); const li = lid && $(`.lesson[data-lesson="${CSS.escape(lid)}"]`); if (li) expandRow(li, true); },
  } : null;
  // Төлбөртэй (худалдаж аваагүй) хичээл дээр дарах → шууд төлбөрийн цонх.
  const paywall = async (li) => {
    if (needLogin()) return;
    const lid = li.dataset.lesson, price = +li.dataset.price;
    try {
      if (price > 0) {
        const d = await api(`/api/courses/${id}/lessons/${lid}/buy`, { method: "POST" });
        if (d.unlocked) { unlockLesson(lid); return expandRow(li, true); }
        pay(d.order, d.payment, `«${rowTitle(li)}» хичээл`, async () => { unlockLesson(lid); toast("🎉 Хичээл нээгдлээ!"); celebrate(); await refreshAccess(); expandRow(li, true); },
          { name: "Энэ хичээл", sub: "дангаар", alt: bundleOpt(lid) });
        return;
      }
      if (bundle > 0) { // зөвхөн багцаар нээгддэг хичээл
        const o = bundleOpt(lid), d = await o.start();
        if (d.enrolled) { unlockAll(); return expandRow(li, true); }
        pay(d.order, d.payment, o.label, o.onPaid, { name: o.name, sub: o.sub });
        return;
      }
      toast("Энэ хичээл одоогоор нээгдээгүй байна");
    } catch (err) { toast(err.message, true); }
  };
  const isLocked = (li) => li.classList.contains("is-locked") && !li.classList.contains("unlocked");


  buy.addEventListener("click", async () => {
    if (all || mode === "lessons") { $("#lessons").scrollIntoView({ behavior: "smooth", block: "start" }); return; }
    if (needLogin()) return;
    buy.disabled = true;
    try {
      const d = await api(`/api/courses/${id}/enroll`, { method: "POST" });
      if (d.enrolled) { if (bundle) unlockAll(); else buy.textContent = "✓ Та элссэн — үзэж эхлэх"; toast("🎉 Амжилттай!"); celebrate(); return; }
      pay(d.order, d.payment, `«${courseTitle}» — бүх хичээл`, () => { unlockAll(); toast("🎉 Бүх хичээл нээгдлээ!"); celebrate(); refreshAccess(); }, { name: "Бүх хичээл багцаар" });
    } catch (e) { toast(e.message, true); } finally { buy.disabled = false; }
  });

  // Хичээлийн цонх хаагдахад анхаарлын хяналт дуусна, видеоны явц илгээгдэнэ, дүгнэлт асууна.
  let stopWatch = null, stopVideos = null, reflectDone = false, curLesson = null;
  new MutationObserver(() => {
    const lm = $("#lessonModal");
    if (lm.classList.contains("open")) { lm.dataset.wasOpen = "1"; return; }
    if (!lm.dataset.wasOpen) return; // "inline" нэмэх зэрэг бусад өөрчлөлт — хаагдаагүй
    delete lm.dataset.wasOpen;
    if (lm.classList.contains("inline")) setTimeout(() => { if (!lm.classList.contains("open")) collapseAll(); }, 0);
    stopWatch?.(); stopWatch = null;
    stopVideos?.(); stopVideos = null;
    if (curLesson && Auth.token && !reflectDone) askReflection(curLesson.id, curLesson.lid, curLesson.title);
    curLesson = null;
  }).observe($("#lessonModal"), { attributes: true, attributeFilter: ["class"] });
  // Хичээлийг үзэж дууссаны дараа "юу сурсан бэ?" гэж товч дүгнэлт асууна — идэвхтэй суралцсаны нотолгоо.
  const askReflection = (cid, lid, title) => {
    try { if (sessionStorage.getItem("sg_refl_" + lid)) return; } catch {}
    const key = "sg_reflect_at_" + lid;
    let last = 0; try { last = +localStorage.getItem(key) || 0; } catch {}
    if (Date.now() - last < 3600e3) return; // цагт нэг удаа л зовоохгүй
    const el = document.createElement("div"); el.className = "watch-alert"; el.innerHTML = `<div class="watch-card refl-card">
      <span class="watch-ico">✍️</span><h3>«${esc(title)}» хичээлээс юу сурсан бэ?</h3>
      <p class="muted small">2-3 өгүүлбэрээр бичихэд хангалттай. Энэ нь идэвхтэй суралцсаны тэмдэглэгээ болно.</p>
      <textarea rows="3" maxlength="2000" placeholder="Жишээ нь: Энэ хичээлээр … гэдгийг ойлголоо…"></textarea>
      <p class="form-error" role="alert"></p>
      <div class="refl-acts"><button class="btn btn-ghost btn-sm" data-skip>Алгасах</button><button class="btn btn-gold" data-send>Илгээх</button></div></div>`;
    document.body.append(el);
    try { localStorage.setItem(key, Date.now()); } catch {}
    const close = () => el.remove();
    $(".refl-card [data-skip]", el).onclick = close;
    $(".refl-card [data-send]", el).onclick = async () => {
      const ta = $("textarea", el), err = $(".form-error", el), b = $("[data-send]", el);
      err.textContent = ""; b.disabled = true;
      try {
        await api(`/api/courses/${cid}/lessons/${lid}/reflect`, { method: "POST", body: { text: ta.value } });
        try { sessionStorage.setItem("sg_refl_" + lid, "1"); } catch {}
        toast("Баярлалаа! ✓"); close();
      } catch (e) { err.textContent = e.message; } finally { b.disabled = false; }
    };
  };
  // Хичээлийн дараалал (хуудасны мөрүүдийн дарааллаар): өмнөх / дараагийн.
  const lessonRows = () => $$(".lesson[data-lesson]");
  const rowTitle = (li) => $(".lesson-title", li)?.childNodes[0]?.textContent?.trim() || "";
  const navHTML = (lid) => {
    const rows = lessonRows(), i = rows.findIndex((r) => r.dataset.lesson === lid), prev = rows[i - 1], next = rows[i + 1];
    const btn = (li, dir) => {
      if (!li) return `<span></span>`;
      const locked = li.classList.contains("is-locked") && !li.classList.contains("unlocked"), drip = li.classList.contains("is-drip");
      const st = $(".lesson-state", li)?.textContent || "";
      return `<button type="button" class="ln ${dir} ${drip ? "is-lock" : ""}" data-nav="${esc(li.dataset.lesson)}"><small>${dir === "next" ? "Дараагийн хичээл" : "Өмнөх хичээл"}</small><b>${esc(rowTitle(li))}</b>${drip ? `<em>${esc(st)}</em>` : locked ? `<em>🔒 ${li.dataset.price > 0 ? "Худалдаж авах" : "Багцаар нээх"}</em>` : `<em>${dir === "next" ? "Үзэх →" : "← Үзэх"}</em>`}</button>`;
    };
    return `${btn(prev, "prev")}${btn(next, "next")}`;
  };
  // Мөрийг доош задалж хичээлийн үндсэн агуулгыг (видео, текст, асуулт, хэлэлцүүлэг) мөрөн дотор харуулна.
  const lessonModalEl = () => $("#lessonModal");
  const dockModal = () => { const m = lessonModalEl(); if (m && m.classList.contains("inline")) { m.classList.remove("inline"); document.body.append(m); } };
  const collapseAll = () => {
    const m = lessonModalEl();
    if (m?.classList.contains("inline") && m.classList.contains("open")) { delete m.dataset.wasOpen; closeModal(m); } // мөр солих үед ажиглагч дахин хумихгүй
    dockModal();
    $$(".lesson.open").forEach((r) => { r.classList.remove("open"); $(".lesson-more", r)?.remove(); });
  };
  const canOpen = (row) => !(row.classList.contains("is-locked") && !row.classList.contains("unlocked")) && !row.classList.contains("is-drip");
  // Дарааллаар түгжээтэй хичээл: жижиг цонх — "Эхлээд өмнөх хичээлээ судал", өмнөх хичээл рүү шууд очих товч.
  const dripPrompt = (row) => {
    $(".drip-pop")?.remove();
    const lid = row.dataset.lesson, st = access?.states?.[lid] || {}, rows = lessonRows(), i = rows.indexOf(row);
    let prev = rows.slice(0, i).reverse().find((r) => st.prev_title && rowTitle(r) === st.prev_title) || rows[i - 1];
    const pTitle = st.prev_title || (prev ? rowTitle(prev) : "");
    const reason = st.reason === "timer" ? `⏳ Энэ хичээл <b>${esc(fmtDate(st.unlock_at))}</b>-д нээгдэнэ. Өмнөх «${esc(pTitle)}» хичээлээ давтаж бэлдээрэй.`
      : st.reason === "quiz" ? `🧩 «<b>${esc(pTitle)}</b>» хичээлийн асуултуудад бүгдэд нь зөв хариулбал цаг, өдрөөс үл хамааран энэ хичээл <b>шууд</b> нээгдэнэ. <span class="dp-left">${st.quiz_left}/${st.quiz_total} асуулт үлдсэн</span>`
      : st.reason === "active" ? `🕒 «<b>${esc(pTitle)}</b>» хичээлийн идэвхтэй суралцах хугацаа (${st.active_min} мин) гүйцээгүй байна. <span class="dp-left">${st.active_left} мин дутуу</span>`
      : st.reason === "exam" ? `📝 «<b>${esc(pTitle)}</b>» шалгалтад тэнцмэгц энэ хичээл шууд нээгдэнэ.`
      : st.reason === "complete" ? `☑️ «<b>${esc(pTitle)}</b>» хичээлийг судлаад «Дууслаа» дармагц энэ хичээл нээгдэнэ.`
      : st.reason === "manual" ? `🔐 Энэ хичээлийг багш тань гараар нээнэ.`
      : `📘 Эхлээд «<b>${esc(pTitle)}</b>» хичээлийг судалж дуусгаад дараа нь энэ хичээлийг үзнэ.`;
    const pop = document.createElement("div"); pop.className = "drip-pop"; pop.setAttribute("role", "dialog"); pop.setAttribute("aria-label", "Өмнөх хичээл");
    pop.innerHTML = `<div class="dp-card"><button class="icon-btn dp-x" aria-label="Хаах">✕</button>
      <div class="dp-steps"><span class="dp-step cur">1<small>${esc(pTitle || "Өмнөх")}</small></span><i></i><span class="dp-step lock">🔒<small>${esc(rowTitle(row))}</small></span></div>
      <h3>Эхлээд өмнөх хичээлээ судлаарай</h3><p>${reason}</p>
      <div class="dp-acts">${prev && canOpen(prev) ? `<button class="btn btn-gold" data-dp-go>▶ «${esc(pTitle).slice(0, 34)}» үзэх</button>` : ""}<button class="btn btn-ghost" data-dp-close>Ойлголоо</button></div></div>`;
    document.body.append(pop);
    const close = () => { pop.classList.add("out"); setTimeout(() => pop.remove(), 200); };
    pop.addEventListener("click", async (e) => {
      if (e.target === pop || e.target.closest(".dp-x, [data-dp-close]")) return close();
      if (e.target.closest("[data-dp-go]")) { close(); prev.scrollIntoView({ block: "center", behavior: "smooth" }); await expandRow(prev); }
    });
    $("[data-dp-go], [data-dp-close]", pop)?.focus();
  };
  const expandRow = async (row, scroll) => {
    const blk = access?.blocks?.[row.dataset.lesson];
    if (blk) { // сануулгын хязгаар хэтэрсэн: орж болохгүй
      $(".drip-pop")?.remove();
      const pop = document.createElement("div"); pop.className = "drip-pop";
      pop.innerHTML = `<div class="dp-card"><button class="icon-btn dp-x" aria-label="Хаах">✕</button><div class="dp-steps"><span class="dp-step lock" style="background:var(--coral-soft);color:var(--coral)">⛔</span></div>
        <h3>${(blk.reason || "").startsWith("Шалгалт") ? "Шалгалт хаагдсан байна" : "Энэ хичээл хаагдсан байна"}</h3><p>${(blk.reason || "").startsWith("Шалгалт") ? "Шалгалтын үеэр зөрчил гарсан тул хаагдсан." : `Та хичээл үзэж байхдаа өөр цонх руу ${access?.max_warnings || 3}-аас олон удаа шилжсэн тул хичээл зогсож хаагдсан.`} ${blk.until ? `<b>${esc(fmtDate(blk.until))}</b> хүртэл хүлээнэ үү.` : "Багш тань дахин нээх хүртэл хүлээнэ үү — багшид мэдэгдэл очсон."}</p>
        <div class="dp-acts"><button class="btn btn-ghost" data-dp-close>Ойлголоо</button></div></div>`;
      document.body.append(pop);
      pop.onclick = (e) => { if (e.target === pop || e.target.closest(".dp-x, [data-dp-close]")) pop.remove(); };
      return;
    }
    if (isLocked(row)) { await paywall(row); return; } // төлөөгүй: шууд «Та төлбөрөө төлнө үү» цонх
    if (row.classList.contains("is-drip")) { dripPrompt(row); return; }
    const was = row.classList.contains("open");
    collapseAll();
    if (was) return;
    const lid = row.dataset.lesson, p = access?.progress?.[lid], rk = access?.ranks?.[lid];
    const status = p?.completed_at ? `<span class="st-chip ok">✓ Дууссан · ${fmtDate(p.completed_at)}</span>` : p?.viewed_at ? `<span class="st-chip">Үзэж эхэлсэн · ${fmtDate(p.viewed_at)}</span>` : `<span class="st-chip">Шинэ хичээл</span>`;
    const facts = [status, rk ? `<span class="st-chip ${rk.disqualified ? "bad" : rk.points >= 75 ? "ok" : ""}" title="${esc((rk.reasons || []).join(", "))}">${rk.disqualified ? "⛔ Оноо тооцогдоогүй" : `+${rk.points} оноо нэгдсэн цолд`}</span>` : ""].join("");
    row.classList.add("open");
    if (!canOpen(row)) { // түгжээтэй: шалтгаан ба нээх товч
      const why = row.classList.contains("is-drip") ? esc($(".lesson-state", row)?.textContent || "Түгжээтэй") : "Энэ хичээл төлбөртэй";
      row.insertAdjacentHTML("beforeend", `<div class="lesson-more"><div class="lm-facts">${facts}<span class="st-chip bad">🔒 ${why}</span></div>${row.classList.contains("is-drip") ? "" : `<div class="lm-acts"><button type="button" class="btn btn-gold btn-sm" data-more-play>🔒 Нээх</button></div>`}</div>`);
      return;
    }
    row.insertAdjacentHTML("beforeend", `<div class="lesson-more"><div class="lm-facts">${facts}</div><div class="lm-body"><div class="loader"></div></div></div>`);
    const m = lessonModalEl(); m.classList.add("inline"); $(".lm-body", row).replaceChildren(m);
    try { await play(lid); } catch (err) { toast(err.message, true); collapseAll(); return; }
    if (scroll) row.scrollIntoView({ block: "start", behavior: "smooth" });
  };
  const play = async (lid) => {
    const l = await api(`/api/courses/${id}/lessons/${lid}`);
    $("#lessonTitle").textContent = l.title;
    const nav = $("#lessonNav");
    if (nav) { nav.innerHTML = navHTML(lid); nav.hidden = lessonRows().length < 2; nav.onclick = async (e) => { const b = e.target.closest("[data-nav]"); if (!b || b.disabled) return; const li = lessonRows().find((r) => r.dataset.lesson === b.dataset.nav); if (!li) return; if (isLocked(li)) { paywall(li); return; } if (li.classList.contains("is-drip")) { dripPrompt(li); return; } if (lessonModalEl().classList.contains("inline") && canOpen(li)) { await expandRow(li, true); return; } $("[data-play]", li)?.click(); $(".lesson-modal")?.scrollTo?.({ top: 0, behavior: "smooth" }); }; }
    const done = $("#lessonDone"), p = access?.progress?.[lid];
    if (done) {
      done.hidden = !Auth.token;
      const b = $("[data-complete]", done); b.disabled = !!p?.completed_at; b.textContent = p?.completed_at ? "✓ Дууссан" : "✓ Энэ хичээлийг дууслаа";
      b.onclick = async () => { b.disabled = true; try { await api(`/api/courses/${id}/lessons/${lid}/complete`, { method: "POST" }); b.textContent = "✓ Дууссан"; toast("Сайн байна! Дараагийн хичээл рүү 🎉"); await refreshAccess(); } catch (e) { toast(e.message, true); b.disabled = false; } };
    }
    if (Auth.token && drip) setTimeout(refreshAccess, 300); // үзсэн нь дараагийнхыг нээж магадгүй
    store.set("sg_last", { course: id, lesson: lid, lessonTitle: l.title, at: Date.now() }); // нүүр хуудасны "Үргэлжлүүлэх"
    const body = $("#lessonContent"), player = $("#player");
    player.innerHTML = mediaHTML(l.video_url, l.title); player.hidden = !l.video_url;
    if (l.blocks?.length) { // блоктой: текст, зураг, дуу, видео, файл, асуулт дарааллаараа
      body.classList.remove("pre"); body.innerHTML = blocksHTML(l.blocks);
      mountQuizzes(body, `/api/courses/${id}/lessons/${lid}`, access?.progress?.[lid]?.quiz || {});
    } else { body.classList.add("pre"); body.textContent = l.content || ""; }
    hydrateBooks(player); hydrateBooks(body); readingRuler(body);
    const modal = $("#lessonModal");
    const isOwner = !!Auth.user?.username && $(".teacher-chip")?.getAttribute("href") === "/t/" + Auth.user.username;
    if (l.exam) { // шалгалт: асуултууд зөвхөн "эхлүүлэх"-ээр ирнэ
      if (done) done.hidden = true;
      examCard(body, { courseId: id, lessonId: lid, title: l.title, onFinish: () => autoAdvance(false) });
    }
    if (l.assignment) { // даалгавар: хариу илгээснээр дуусна
      if (done) done.hidden = true;
      assignmentCard(body, { courseId: id, lessonId: lid });
    }
    const dbox = $("#lessonDiscuss");
    if (dbox) { dbox.hidden = true; dbox.innerHTML = ""; if (l.discussion && !l.exam) discussionPanel(dbox, { courseId: id, lessonId: lid }); }
    // Видео: бүрэн үзэхээс өмнө урагш гүйлгэхгүй; хэдэн удаа аль хэсгийг үзсэнийг бичинэ.
    const watched = {};
    for (const [k, v] of Object.entries(access?.progress?.[lid]?.quiz || {})) if (k.startsWith("watch_") && v) watched[k.slice(6)] = true;
    stopVideos?.();
    stopVideos = guardVideos(body, { watched, owner: isOwner, modal, progressBase: Auth.token && !isOwner ? `/api/courses/${id}/lessons/${lid}` : null,
      onWatched: (bid) => {
        if (!Auth.token) return;
        api(`/api/courses/${id}/lessons/${lid}/watched/${bid}`, { method: "POST" }).then(() => toast("✓ Видеог бүрэн үзлээ — одоо гүйлгэж болно")).catch(() => {});
      } });
    // Идэвхтэй суралцах хугацаа
    const need = (l.active_min || 0) * 60, cb = done && $("[data-complete]", done), meter = done && $(".active-need", done);
    if (meter) meter.hidden = !need;
    // Идэвхтэй хугацаа бүрэн гүйцмэгц: асуултууд (байвал) бүгд зөв бол автоматаар "дууслаа" → дараагийн хичээл рүү шилжинэ.
    let autoDone = false;
    const quizIds = (l.blocks || []).filter((b) => b.type === "quiz").map((b) => b.id);
    const quizzesOk = () => { const q = access?.progress?.[lid]?.quiz || {}; return quizIds.every((qid) => q[qid]); };
    const finishByTime = async () => {
      if (autoDone || l.exam || l.assignment || isOwner || !Auth.token) return;
      autoDone = true;
      if (!quizzesOk()) { toast(`✓ ${l.active_min} мин гүйцлээ — одоо асуултуудаа дуусгаарай, дараагийн хичээл нээгдэнэ`); autoDone = false; return; }
      for (let i = 0; i < 4; i++) { // сервер дээрх идэвхтэй хугацаа бага зэрэг хоцорч бүртгэгдэж болно
        try {
          await api(`/api/courses/${id}/lessons/${lid}/complete`, { method: "POST" });
          if (cb) { cb.disabled = true; cb.textContent = "✓ Дууссан"; }
          toast("✓ Хичээлийн хугацаа бүрэн гүйцлээ — дараагийн хичээл рүү шилжиж байна…");
          await autoAdvance(true);
          return;
        } catch (e) { if (e.status !== 409) { autoDone = false; return; } await new Promise((r) => setTimeout(r, 6000)); }
      }
      autoDone = false;
    };
    const onActive = (sec) => {
      if (!need || !cb || p?.completed_at) return;
      meter.textContent = `🕒 Идэвхтэй суралцсан: ${Math.floor(Math.min(sec, need) / 60)}/${l.active_min} мин`;
      meter.classList.toggle("ok", sec >= need);
      cb.disabled = sec < need;
      if (sec < need) cb.textContent = `🕒 ${l.active_min} мин идэвхтэй үзсний дараа дуусгана`; else { cb.textContent = "✓ Энэ хичээлийг дууслаа"; finishByTime(); }
    };
    stopWatch?.();
    stopWatch = l.exam ? null : watchLesson({ courseId: id, lessonId: lid, modal, onStop: () => { closeModal(modal); refreshAccess(); }, onActive, maxWarn: access?.max_warnings });
    reflectDone = !!(Auth.token && access?.progress?.[lid]?.completed_at) || isOwner || !Auth.token || !!l.exam;
    curLesson = reflectDone ? null : { id, lid, title: l.title };
    openModal(modal);
  };

  // Нүүр хуудаснаас #l=<хичээл> холбоосоор ирвэл тухайн хичээлийг шууд нээнэ (эрхгүй бол мөрийг тодруулна).
  const want = new URLSearchParams(location.hash.slice(1)).get("l");
  if (want) {
    const li = $(`.lesson[data-lesson="${CSS.escape(want)}"]`);
    if (li) { li.scrollIntoView({ block: "center" }); play(want).catch((err) => { if (err.status === 402 && Auth.token) return paywall(li); li.animate([{ boxShadow: "0 0 0 4px rgba(31,60,143,.35)" }, { boxShadow: "0 0 0 0 transparent" }], { duration: 1600 }); }); }
  }

  document.addEventListener("click", async (e) => {
    const row = e.target.closest(".lesson[data-lesson]");
    if (row && !e.target.closest("button, a, .lesson-more")) { await expandRow(row); return; }
    if (e.target.closest("[data-more-play]")) { const r = e.target.closest(".lesson"); $("[data-play]", r)?.click(); return; }
    const mn = e.target.closest("[data-more-next]");
    if (mn) { const r = lessonRows().find((x) => x.dataset.lesson === mn.dataset.moreNext); if (r) await expandRow(r, true); return; }
    const b = e.target.closest("[data-play]");
    if (b && !e.target.closest(".lesson-more") && canOpen(b.closest(".lesson"))) { await expandRow(b.closest(".lesson")); return; }
    if (!b) return;
    const li = b.closest(".lesson"), lid = li.dataset.lesson;
    if (access?.blocks?.[lid]) { await expandRow(li); return; }
    if (isLocked(li)) { b.disabled = true; try { await paywall(li); } finally { b.disabled = false; } return; } // хичээл, багц, шалгалт, даалгавар — төлбөрийн цонх
    if (li.classList.contains("is-drip")) { dripPrompt(li); return; }
    try { await play(lid); } catch (err) { toast(err.message, true); }
  });
}

/* ---------- Төлбөрийн цонх: «Та төлбөрөө төлнө үү» — QR (QPay эсвэл демо), банкны апп, төлөгдмөгц автоматаар нээнэ ---------- */
// payWindow(order, payment, label, onPaid, { name, sub, alt }) — alt: өөр сонголт (жишээ нь бүх хичээл багцаар):
//   { name, sub, amount, label, start: async () => ({ order, payment } | { unlocked | enrolled }), onPaid }
function payWindow(order, payment, label, onPaid, extra = {}) {
  $("#payWin")?.remove();
  const el = document.createElement("div");
  el.className = "modal pay-win"; el.id = "payWin"; el.setAttribute("aria-hidden", "true");
  document.body.append(el);
  const opts = [{ name: extra.name || "Энэ хичээл", sub: extra.sub || "", amount: order.amount, label, onPaid, d: { order, payment } }];
  if (extra.alt) opts.push({ ...extra.alt, d: null });
  const mobile = matchMedia("(max-width: 700px), (pointer: coarse)").matches;
  let sel = 0, timer = 0, gone = false, busy = false;
  const shut = () => { gone = true; clearTimeout(timer); document.removeEventListener("visibilitychange", onVis); closeModal(el); setTimeout(() => el.remove(), 300); };
  const stage = (o) => {
    if (busy || !o.d) return `<div class="pw-load"><span class="loader"></span></div>`;
    const p = o.d.payment || {}, ord = o.d.order;
    if (p.error) return `<div class="pw-empty"><b>⚠️ ${esc(p.error)}</b><small>Цонхоо хаагаад дахин дарна уу.</small></div>`;
    if (!p.qr) return `<div class="pw-empty"><b>Онлайн төлбөр (QPay) хараахан тохируулагдаагүй байна.</b><small>Багштайгаа холбогдоно уу · захиалга #${esc(ord.id.slice(-6).toUpperCase())} · ${money(ord.amount)}</small></div>`;
    const qpay = p.provider === "qpay";
    return `<div class="pw-body"><figure class="pw-qr"><span class="pw-qr-frame"><img src="${p.qr}" alt="Төлбөрийн QR код" width="200" height="200"></span><figcaption>${qpay ? "QPay · бүх банкны апп" : "Демо QR · туршилтын горим"}</figcaption></figure>
      <div class="pw-side"><p class="pw-amount">${money(ord.amount)}</p>
      <ol class="pw-steps"><li><span>${qpay ? "Банкны апп-аа нээгээд <b>QR уншуулах</b>-ыг сонгоно" : "Утасныхаа камераар QR-ыг уншуулна"}</span></li><li><span>${qpay ? "Дүнг шалгаад төлбөрөө баталгаажуулна" : "Нээгдсэн хуудсанд <b>«Төлөх»</b> дарна"}</span></li><li><span>Энэ цонх өөрөө шинэчлэгдэж <b>шууд нээгдэнэ</b></span></li></ol>
      <p class="pw-status" role="status"><i></i>Төлбөр хүлээж байна… <button type="button" class="pw-recheck" data-pw-check>Шалгах</button></p></div></div>
      ${qpay && p.urls?.length ? `<div class="pw-banks ${mobile ? "open" : ""}"><button type="button" class="pw-banks-t" data-pw-banks aria-expanded="${mobile}">📱 Утсан дээрээ байна уу? Банкны апп-аар шууд төлөх</button>
        <div class="pw-bank-list">${p.urls.map((u) => `<a class="pw-bank" href="${esc(u.link)}" rel="noopener">${u.logo ? `<img src="${esc(u.logo)}" alt="" loading="lazy">` : ""}<span>${esc(u.description || u.name)}</span></a>`).join("")}</div></div>` : ""}
      ${!qpay && mobile && p.pay_url ? `<a class="btn btn-glass btn-sm pw-here" href="${esc(p.pay_url)}" target="_blank" rel="noopener">Энэ утсан дээр төлөх</a>` : ""}`;
  };
  const draw = () => {
    const o = opts[sel];
    el.innerHTML = `<div class="modal-card pw-card" role="dialog" aria-modal="true" aria-labelledby="pwTitle"><button class="icon-btn modal-x" data-close aria-label="Хаах">✕</button>
      <span class="eyebrow">Төлбөр</span><h3 class="h3" id="pwTitle">Та төлбөрөө төлнө үү</h3><p class="pw-what">${esc(o.label || "")}</p>
      ${opts.length > 1 ? `<div class="pw-opts" role="radiogroup" aria-label="Юуг төлөх вэ">${opts.map((x, i) => `<button type="button" role="radio" aria-checked="${i === sel}" class="pw-opt ${i === sel ? "on" : ""}" data-pw-opt="${i}"><b>${esc(x.name)}</b><span>${money(x.amount)}</span>${x.sub ? `<small>${esc(x.sub)}</small>` : ""}</button>`).join("")}</div>` : ""}
      <div class="pw-stage">${stage(o)}</div>
      ${o.d?.payment?.dev_pay && !busy ? `<button type="button" class="btn btn-glass btn-block btn-sm pw-dev" data-pw-dev>Төлсөн гэж баталгаажуулах (демо)</button>` : ""}
      <p class="pw-foot">🔒 Төлбөр орсон даруйд автоматаар нээгдэнэ. Цонхоо хаасан ч төлбөр тань хүчинтэй.</p></div>`;
  };
  const paid = (o) => {
    if (gone) return; gone = true; clearTimeout(timer); document.removeEventListener("visibilitychange", onVis);
    const st = $(".pw-stage", el); if (st) st.innerHTML = `<div class="pw-done"><span class="pw-check" aria-hidden="true">✓</span><b>Төлбөр амжилттай!</b><small>${esc(o.label || "")} нээгдлээ</small></div>`;
    $(".pw-dev", el)?.remove(); $(".pw-opts", el)?.remove();
    setTimeout(() => { closeModal(el); setTimeout(() => el.remove(), 300); o.onPaid?.(); }, 1400);
  };
  const poll = async (now) => {
    clearTimeout(timer);
    const o = opts[sel]; if (gone || !o.d?.order || !o.d.payment?.provider) return;
    if (!now) { timer = setTimeout(() => poll(true), 3000); return; }
    const r = await api(`/api/orders/${o.d.order.id}`).catch(() => null);
    if (gone || opts[sel] !== o) return;
    if (r?.status === "paid") return paid(o);
    timer = setTimeout(() => poll(true), 3000);
  };
  const onVis = () => { if (!document.hidden) poll(true); }; // утсаар төлөөд буцаж ирэхэд шууд шалгана
  document.addEventListener("visibilitychange", onVis);
  el.addEventListener("click", async (e) => {
    if (e.target === el || e.target.closest("[data-close]")) return shut();
    const ch = e.target.closest("[data-pw-opt]");
    if (ch) {
      const i = +ch.dataset.pwOpt; if (i === sel || busy) return;
      sel = i; clearTimeout(timer);
      const o = opts[i];
      if (!o.d) {
        busy = true; draw();
        try {
          const d = await o.start();
          if (d.unlocked || d.enrolled) { busy = false; return paid(o); }
          o.d = d;
        } catch (err) { toast(err.message, true); sel = 0; }
        busy = false;
      }
      draw(); poll(); return;
    }
    const dev = e.target.closest("[data-pw-dev]");
    if (dev) {
      const o = opts[sel]; dev.disabled = true;
      try { await api(`/api/orders/${o.d.order.id}/dev-pay`, { method: "POST" }); paid(o); } catch (err) { toast(err.message, true); dev.disabled = false; }
      return;
    }
    if (e.target.closest("[data-pw-check]")) { const s = $(".pw-status", el); s?.classList.add("checking"); await poll(true); s?.classList.remove("checking"); return; }
    const bk = e.target.closest("[data-pw-banks]");
    if (bk) { const w = bk.closest(".pw-banks"); w.classList.toggle("open"); bk.setAttribute("aria-expanded", w.classList.contains("open")); }
  });
  draw(); openModal(el); poll();
  $(".modal-x", el)?.focus();
  return { close: shut };
}
window.SG_pay = payWindow; // шалгалт, даалгаврын төлбөр, шууд хичээл, ном, багтаамж — бүгд ижил цонх

function celebrate() {
  if (reduce) return;
  const colors = ["#1f3c8f", "#eaa02e", "#3a5bb8", "#f6c56f", "#172c6b"];
  for (let i = 0; i < 90; i++) {
    const p = document.createElement("i");
    p.style.cssText = `position:fixed;z-index:200;left:50%;top:40%;width:8px;height:14px;border-radius:2px;pointer-events:none;background:${colors[i % 5]}`;
    document.body.append(p);
    const a = Math.random() * Math.PI * 2, v = 200 + Math.random() * 400;
    p.animate([{ transform: "translate(0,0) rotate(0)", opacity: 1 },
      { transform: `translate(${Math.cos(a) * v}px,${Math.sin(a) * v + 300}px) rotate(${Math.random() * 720}deg)`, opacity: 0 }],
      { duration: 1400 + Math.random() * 800, easing: "cubic-bezier(.2,.8,.2,1)" }).onfinish = () => p.remove();
  }
}

/* ---------- нэвтрэх хуудас ---------- */
/* ---------- Сургалтын хайлт (нэвтрэх хуудсан дээр) ---------- */
function findCourses() {
  const box = $("#find"); if (!box) return;
  const input = $("input[name=q]", box), list = $(".find-list", box), meta = $(".find-meta", box), pager = $(".find-pager", box);
  const st = { q: "", tag: "", page: 1 }, cache = new Map();
  let ctl = null, timer = 0;
  const mark = (text) => { // хайсан үгийг тодруулна (кирилл үгэнд)
    let h = esc(text);
    for (const w of st.q.split(/\s+/).filter((x) => x.length >= 2)) h = h.replace(new RegExp(w.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"), "giu"), (m) => `<mark>${m}</mark>`);
    return h;
  };
  const card = (it) => `<li class="find-item">
      <a class="find-main" href="/c/${esc(it.course_id)}"><strong>${mark(it.title)}</strong>
        ${it.match ? `<span class="find-match">📖 ${mark(it.match)}</span>` : ""}
        <span class="find-tagrow">${it.tags.map((t) => `<span class="ftag ${t === "Үнэгүй" ? "ftag-free" : t === "Төлбөртэй" ? "ftag-paid" : t === "Сертификаттай" ? "ftag-cert" : t === "Шалгалттай" ? "ftag-exam" : ""}">${esc(t)}</span>`).join("")}</span>
        <small class="muted">${it.lessons} хичээл${it.price ? " · " + money(it.price) : ""}${it.views ? " · " + it.views + " үзэлт" : ""}</small></a>
      <a class="find-teacher" href="/t/${esc(it.teacher.username)}" title="Багшийн профайл">${avatarHTML(it.teacher, "avatar-sm")}<span>${mark(it.teacher.display_name)}<small class="muted">Профайл үзэх →</small></span></a></li>`;
  const pages = (p, n) => { // 1 … 4 5 6 … 20
    const set = new Set([1, n, p - 1, p, p + 1].filter((x) => x >= 1 && x <= n));
    const out = []; let prev = 0;
    [...set].sort((a, b) => a - b).forEach((x) => { if (x - prev > 1) out.push("…"); out.push(x); prev = x; });
    return out;
  };
  const render = (d) => {
    meta.textContent = d.total ? `${d.total} сургалт олдлоо${d.pages > 1 ? ` · ${d.page}/${d.pages} хуудас` : ""}` : `«${st.q}» гэсэн сургалт олдсонгүй — өөр үгээр, эсвэл латинаар хайж үзээрэй`;
    list.innerHTML = d.items.map(card).join("");
    pager.innerHTML = d.pages > 1 ? `<button type="button" data-page="${d.page - 1}" ${d.page <= 1 ? "disabled" : ""} aria-label="Өмнөх">‹</button>
      ${pages(d.page, d.pages).map((x) => x === "…" ? `<span>…</span>` : `<button type="button" data-page="${x}" ${x === d.page ? 'aria-current="page"' : ""}>${x}</button>`).join("")}
      <button type="button" data-page="${d.page + 1}" ${d.page >= d.pages ? "disabled" : ""} aria-label="Дараах">›</button>` : "";
  };
  // Google шиг: хайлтын үг оруулаагүй үед зөвхөн хайлтын мөр харагдана.
  const tags = $(".find-tags", box);
  const clear = () => { list.innerHTML = ""; meta.textContent = ""; pager.innerHTML = ""; tags.hidden = true; box.classList.remove("has-results"); };
  const load = async () => {
    if (!st.q) return clear();
    tags.hidden = false; box.classList.add("has-results");
    const key = `${st.q}|${st.tag}|${st.page}`;
    if (cache.has(key)) return render(cache.get(key));
    ctl?.abort(); ctl = new AbortController();
    box.classList.add("loading");
    try {
      // Protobuf API (Connect-JSON): POST /surgalt.v1.Surgalt/Search — gRPC-тэй ижил үйлчилгээ, ижил ClickHouse.
      const res = await fetch("/surgalt.v1.Surgalt/Search", { method: "POST", signal: ctl.signal, headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query: st.q, tag: st.tag, page: st.page }) });
      const raw = await res.json(); if (!res.ok) throw new Error(raw?.message || "Хайлт амжилтгүй");
      // proto3 JSON: lowerCamelCase, int64 нь мөр, 0/хоосон утга орхигддог.
      const d = { total: raw.total || 0, page: raw.page || 1, pages: raw.pages || 1, items: (raw.items || []).map((it) => ({
        course_id: it.courseId, title: it.title || "", match: it.match, tags: it.tags || [], price: +it.price || 0, lessons: it.lessons || 0, views: +it.views || 0,
        teacher: { username: it.teacher?.username || "", display_name: it.teacher?.displayName || "", avatar_url: it.teacher?.avatarUrl || "" } })) };
      if (cache.size > 60) cache.clear();
      cache.set(key, d); render(d);
    } catch (e) { if (e.name !== "AbortError") meta.textContent = e.message; } finally { box.classList.remove("loading"); }
  };
  input.addEventListener("input", () => { clearTimeout(timer); timer = setTimeout(() => { st.q = input.value.trim(); st.page = 1; load(); }, 220); });
  $(".find-tags", box).addEventListener("click", (e) => {
    const b = e.target.closest("[data-tag]"); if (!b) return;
    $$("[data-tag]", box).forEach((x) => x.setAttribute("aria-pressed", x === b)); st.tag = b.dataset.tag; st.page = 1; load();
  });
  pager.addEventListener("click", (e) => {
    const b = e.target.closest("[data-page]"); if (!b || b.disabled) return;
    st.page = +b.dataset.page; load(); box.scrollIntoView({ behavior: "smooth", block: "start" });
  });
  $(".find-bar", box).addEventListener("submit", () => { clearTimeout(timer); st.q = input.value.trim(); st.page = 1; load(); });
  $(".find-chips")?.addEventListener("click", (e) => { // түгээмэл хайлт
    const b = e.target.closest("[data-q]"); if (!b) return;
    input.value = b.dataset.q; st.q = b.dataset.q; st.page = 1; load(); input.focus();
  });
  const q0 = new URLSearchParams(location.search).get("q"); // /?q=... холбоосоор ирвэл
  if (q0) { input.value = q0; st.q = q0.trim(); load(); }
}

function loginPage() {
  const q = new URLSearchParams(location.search), h = new URLSearchParams(location.hash.slice(1));
  const err = $("#authError");
  // Багш студи рүү, суралцагч бүх зүйл нь байрлах нүүр хуудас руу очно.
  const home = (u) => (u?.role === "teacher" ? (u.username ? "/t/" + u.username : "/me") : "/");
  const next = (u) => { const n = h.get("next") || q.get("next") || home(u); return n.startsWith("/") && !n.startsWith("//") && !n.startsWith("/\\") ? n : home(u); };
  // OAuth-оос буцаж ирсэн
  if (h.get("token")) {
    const tok = h.get("token");
    history.replaceState(null, "", location.pathname);
    api("/api/me", { token: tok }).then((u) => { Auth.set(tok, u); location.replace(next(u)); }).catch((e) => (err.textContent = e.message));
    return;
  }
  if (h.get("error")) err.textContent = h.get("error");
  const teacher = $("#asTeacher");
  teacher.checked = q.get("role") === "teacher";
  const tabs = $(".tabs"), lf = $("#loginForm"), rf = $("#registerForm");
  $$(".tab").forEach((t) => t.addEventListener("click", () => {
    $$(".tab").forEach((x) => x.classList.toggle("active", x === t));
    tabs.dataset.active = t.dataset.tab;
    lf.hidden = t.dataset.tab !== "login"; rf.hidden = t.dataset.tab !== "register"; err.textContent = "";
  }));
  if (q.get("role") === "teacher") $$(".tab")[1].click();
  // Нэвтрэх товчны таних тэмдгүүд (үйлчилгээ тус бүрийн албан ёсны өнгөөр).
  const GOOGLE_G = `<svg viewBox="0 0 48 48" width="20" height="20" aria-hidden="true"><path fill="#EA4335" d="M24 9.500c3.540 0 6.710 1.220 9.210 3.600l6.850-6.850C35.900 2.380 30.470 0 24 0 14.620 0 6.510 5.380 2.560 13.220l7.980 6.190C12.430 13.720 17.740 9.500 24 9.500z"/><path fill="#4285F4" d="M46.980 24.550c0-1.570-.150-3.090-.380-4.550H24v9.020h12.940c-.580 2.960-2.260 5.480-4.780 7.180l7.730 6c4.510-4.180 7.090-10.360 7.090-17.650z"/><path fill="#FBBC05" d="M10.530 28.590c-.480-1.450-.760-2.990-.760-4.590s.270-3.140.760-4.590l-7.980-6.190C.920 16.460 0 20.120 0 24c0 3.880.920 7.540 2.560 10.780l7.970-6.190z"/><path fill="#34A853" d="M24 48c6.480 0 11.930-2.130 15.890-5.810l-7.730-6c-2.150 1.450-4.920 2.300-8.160 2.300-6.260 0-11.570-4.220-13.470-9.910l-7.980 6.190C6.510 42.620 14.620 48 24 48z"/></svg>`;
  const FACEBOOK_F = `<svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true"><circle cx="12" cy="12" r="12" fill="#1877F2"/><path fill="#fff" d="M16.500 15.500l.500-3.500h-3.300V9.800c0-1 .500-1.900 2-1.900H17V5c-.300 0-1.300-.200-2.500-.200-2.600 0-4.200 1.600-4.200 4.400V12H7.500v3.500h2.800V24h3.400v-8.500z"/></svg>`;
  const icons = { google: [GOOGLE_G, "Google"], facebook: [FACEBOOK_F, "Facebook"],
    microsoft: [`<span class="social-ico" style="background:#0078d4">⊞</span>`, "Microsoft"],
    instagram: [`<span class="social-ico" style="background:linear-gradient(45deg,#f58529,#dd2a7b,#8134af)">◎</span>`, "Instagram"] };
  const social = $("#social");
  const renderSocial = (ps) => {
    social.innerHTML = ps.map((p) => {
      const [ic, name] = icons[p.name] || [`<span class="social-ico" style="background:#555">•</span>`, p.label];
      const u = `${p.url}?role=${teacher.checked ? "teacher" : "student"}&next=${encodeURIComponent(next({ role: teacher.checked ? "teacher" : "student" }))}`;
      return `<a class="btn btn-social" href="${esc(u)}">${ic}<span>${esc(name)}-ээр үргэлжлүүлэх</span></a>`;
    }).join("");
    $(".divider").hidden = !ps.length;
  };
  let providers = [];
  api("/api/auth/providers", { token: null }).then((ps) => { providers = ps; renderSocial(ps); }).catch(() => renderSocial([]));
  teacher.addEventListener("change", () => renderSocial(providers));
  const done = (d) => { Auth.set(d.token, d.user); location.href = next(d.user); };
  lf.addEventListener("submit", async (e) => {
    e.preventDefault(); err.textContent = "";
    try { done(await api("/api/auth/login", { method: "POST", body: { email: lf.email.value, password: lf.password.value }, token: null })); }
    catch (x) { err.textContent = x.message; $(".auth-card").animate([{ transform: "translateX(0)" }, { transform: "translateX(-10px)" }, { transform: "translateX(10px)" }, { transform: "translateX(0)" }], { duration: 350 }); }
  });
  // Нууц үг харуулах/нуух
  $$("[data-pw-toggle]").forEach((b) => b.addEventListener("click", () => {
    const i = b.previousElementSibling, show = i.type === "password";
    i.type = show ? "text" : "password"; b.classList.toggle("on", show); b.setAttribute("aria-label", show ? "Нууц үг нуух" : "Нууц үг харуулах");
  }));
  // Давтсан нууц үгийг бичих зуур шалгана.
  const match = $("#pwMatch");
  const checkMatch = () => {
    const a = rf.password.value, b = rf.password2.value, ok = a === b;
    rf.password2.setCustomValidity(ok ? "" : "Нууц үг таарахгүй байна");
    match.textContent = !b ? "" : ok ? "✓ Нууц үг таарч байна" : "Нууц үг таарахгүй байна";
    match.className = "pw-match " + (!b ? "" : ok ? "ok" : "bad");
    return ok;
  };
  rf.password.addEventListener("input", checkMatch); rf.password2.addEventListener("input", checkMatch);
  rf.addEventListener("submit", async (e) => {
    e.preventDefault(); err.textContent = "";
    if (!checkMatch()) { err.textContent = "Нууц үг таарахгүй байна"; rf.password2.focus(); return; }
    const body = { email: rf.email.value, password: rf.password.value, role: teacher.checked ? "teacher" : "student" };
    try { done(await api("/api/auth/register", { method: "POST", body, token: null })); } catch (x) { err.textContent = x.message; }
  });
}

/* ---------- эхлүүлэх ---------- */
window.SG = { pay: payWindow, icon, embedDoc, ChatThread, pdfLib, BookReader, richHTML, blocksHTML, mountQuizzes, fmtBytes, extOf, $, $$, esc, api, Auth, toast, money, fmtDate, fmtTime, Live, mediaHTML, book3dHTML, hydrateBooks, Flipbook, openModal, closeModal, msgHTML, linkify, celebrate, reveals, counters, avatarHTML, ringHTML, hueOfName, fmtDay, WEEKDAYS, WEEKDAYS_SHORT };
// Хөдөлгөөнийг цөөлсөн: хазайлт, соронзон товч, курсор дагасан гэрэл, нээлтийн хөшиг ашиглахгүй.
splitText(); reveals(); counters(); navScroll(); ripples(); authNav(); chatRail(); hydrateBooks();
if (page === "me") { // өөрийн хуудас руу: багш профайл, суралцагч нүүр; нэвтрээгүй бол нэвтрэх
  const u = Auth.user, h = location.hash;
  if (!Auth.token) location.replace("/login?next=" + encodeURIComponent("/me" + h));
  else if (u?.role === "teacher" && u.username) location.replace("/t/" + u.username + h);
  else api("/api/me").then((me) => { Auth.set(Auth.token, me); location.replace(me.role === "teacher" ? "/t/" + me.username + h : "/" + h); }).catch(() => location.replace("/login"));
}
if (page === "home") homePage();
if (page === "profile") profilePage();
if (page === "profile" || page === "course") chatWidget();
if (page === "course") coursePage();
if (page === "login") loginPage();
if (page === "home") findCourses();
// Үзэлт: хуудас нээгдэхэд хөтөч мэдээлнэ (нэвтэрсэн бол токентой) — багш өөрийнхийгөө үзвэл сервер тоолохгүй.
{
  const v = page === "profile" ? ["profile", $("main.pf")?.dataset.teacher] : page === "course" ? ["course", $("main.course-page")?.dataset.course] : null;
  if (v?.[1]) api("/api/views", { method: "POST", body: { kind: v[0], id: v[1] } }).catch(() => {});
}
if (page === "book") bookPage();
})();
