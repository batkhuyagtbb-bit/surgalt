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
  shield: '<path d="M12 3 4.5 6v5.5c0 4.6 3.2 8.3 7.5 9.5 4.3-1.2 7.5-4.9 7.5-9.5V6L12 3Z"/><path d="m9 12 2 2 4-4"/>',
  cal: '<rect x="3" y="5" width="18" height="16" rx="2"/><path d="M8 3v4M16 3v4M3 10h18"/>',
  pin: '<path d="M12 21s-7-6.2-7-11.5a7 7 0 0 1 14 0C19 14.8 12 21 12 21z"/><circle cx="12" cy="9.5" r="2.5"/>',
  courses: '<path d="m2 9 10-5 10 5-10 5z"/><path d="M6 11.5V16c0 1.2 2.7 3 6 3s6-1.8 6-3v-4.5"/>',
  book: '<path d="M4 5a2 2 0 0 1 2-2h14v16H6a2 2 0 0 0-2 2z"/><path d="M4 21V5M9 7h7"/>',
  live: '<rect x="3" y="6" width="12" height="12" rx="2"/><path d="m15 10 6-3v10l-6-3z"/>',
  chat: '<path d="M21 12a8 8 0 0 1-11.6 7.1L4 20l1-4.6A8 8 0 1 1 21 12Z"/>',
  cloud: '<path d="M7 18.5a4.5 4.5 0 0 1-.7-8.95A6 6 0 0 1 17.8 8a4.75 4.75 0 0 1-.55 10.5Z"/>',
  board: '<rect x="3" y="4" width="18" height="12" rx="2"/><path d="M7 20l2-4M17 20l-2-4M12 16v4"/><path d="m7 12 3-3 2 2 4-4"/>',
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
  edit: '<path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z"/>',
};
const icon = (n, size = 18) => `<svg data-ico="${n}" viewBox="0 0 24 24" width="${size}" height="${size}" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${UI_ICONS[n] || ""}</svg>`;
// Хөдөлгөөнт дүрсэнд (animateIcons) шинээр ашигласан нэрсийн статик зам (байхгүй бол л нэмнэ).
for (const [k, v] of Object.entries({"alert":"<path d=\"m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3\"/><path d=\"M12 9v4\"/><path d=\"M12 17h.01\"/>","phone":"<path d=\"M13.83 16.57a1 1 0 0 0 1.21-.3l.36-.47A2 2 0 0 1 17 15h3a2 2 0 0 1 2 2v3a2 2 0 0 1-2 2A18 18 0 0 1 2 4a2 2 0 0 1 2-2h3a2 2 0 0 1 2 2v3a2 2 0 0 1-.8 1.6l-.47.35a1 1 0 0 0-.29 1.24 14 14 0 0 0 6.39 6.38\"/>","card":"<rect width=\"20\" height=\"14\" x=\"2\" y=\"5\" rx=\"2\"/><path d=\"M2 10h20\"/>","bell":"<path d=\"M6 9a6 6 0 0 1 12 0c0 6 2.5 7.5 2.5 7.5h-17S6 15 6 9\"/><path d=\"M10 20a2.2 2.2 0 0 0 4 0\"/>","exam":"<rect width=\"8\" height=\"4\" x=\"8\" y=\"2\" rx=\"1\"/><path d=\"M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2\"/><path d=\"m9 14 2 2 4-4\"/>","doc":"<path d=\"M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z\"/><path d=\"M14 2v4a2 2 0 0 0 2 2h4\"/><path d=\"M10 9H8M16 13H8M16 17H8\"/>","headphones":"<path d=\"M3 14h3a2 2 0 0 1 2 2v3a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-7a9 9 0 0 1 18 0v7a2 2 0 0 1-2 2h-1a2 2 0 0 1-2-2v-3a2 2 0 0 1 2-2h3\"/>","folder":"<path d=\"M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z\"/>","eye":"<path d=\"M2.06 12.35a1 1 0 0 1 0-.7 10.75 10.75 0 0 1 19.88 0 1 1 0 0 1 0 .7 10.75 10.75 0 0 1-19.88 0\"/><circle cx=\"12\" cy=\"12\" r=\"3\"/>","check":"<path d=\"M20 6 9 17l-5-5\"/>","live":"<path d=\"m16 13 5.22 3.48a.5.5 0 0 0 .78-.42V7.94a.5.5 0 0 0-.76-.43L16 10.5\"/><rect x=\"2\" y=\"6\" width=\"14\" height=\"12\" rx=\"2\"/>","book":"<path d=\"M12 7v14\"/><path d=\"M3 18a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h5a4 4 0 0 1 4 4 4 4 0 0 1 4-4h5a1 1 0 0 1 1 1v13a1 1 0 0 1-1 1h-6a3 3 0 0 0-3 3 3 3 0 0 0-3-3z\"/>","files":"<path d=\"M20 7h-3a2 2 0 0 1-2-2V2\"/><path d=\"M9 18a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h7l4 4v10a2 2 0 0 1-2 2Z\"/><path d=\"M3 7.6v12.8A1.6 1.6 0 0 0 4.6 22h9.8\"/>"})) UI_ICONS[k] ||= v;
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

// Нүүрний гарчиг: үсгүүд дээр энд тэнд жижиг одод ээлжлэн гялсхийнэ (app.css .ht-star). Байрлалыг гялсхийх агшинд
// (animationstart) санамсаргүй үгийн дээгүүр сонгоно — үгс орж ирэх хөдөлгөөн дууссаны дараах байрлалаар.
function heroStars() {
  const h = $(".home-title"); if (!h || reduce || $(".ht-star", h)) return;
  const words = $$(".w > span", h); if (!words.length) return;
  const place = (st) => {
    const hr = h.getBoundingClientRect(), w = words[Math.floor(Math.random() * words.length)].getBoundingClientRect();
    if (!hr.width || !w.width) return;
    st.style.left = (w.left - hr.left + w.width * (0.08 + Math.random() * 0.84)).toFixed(1) + "px";
    st.style.top = (w.top - hr.top + w.height * (0.12 + Math.random() * 0.62)).toFixed(1) + "px";
    st.style.setProperty("--s", (14 + Math.random() * 14).toFixed(0) + "px");
    const blue = Math.random() < 0.3; st.classList.toggle("b", blue);
    st.querySelector("path").setAttribute("fill", blue ? "url(#htStarB)" : "url(#htStarG)");
  };
  const run = (st, first) => {
    st.classList.remove("go"); void st.offsetWidth;
    st.style.setProperty("--dl", ((first ? 1.8 : 0.3) + Math.random() * 2.4).toFixed(2) + "s");
    st.classList.add("go");
  };
  // Нэг удаа: алтан ба цэнхэр градиент (төв нь цагаан гэрэл)
  h.insertAdjacentHTML("beforeend", `<svg width="0" height="0" style="position:absolute" aria-hidden="true"><defs><radialGradient id="htStarG"><stop offset="0" stop-color="#fff"/><stop offset=".28" stop-color="#fff3cc"/><stop offset=".62" stop-color="#ffc24a"/><stop offset="1" stop-color="#e08a10"/></radialGradient><radialGradient id="htStarB"><stop offset="0" stop-color="#fff"/><stop offset=".3" stop-color="#e6edff"/><stop offset=".65" stop-color="#7d9cff"/><stop offset="1" stop-color="#3557d6"/></radialGradient></defs></svg>`);
  for (let k = 0; k < 4; k++) {
    const st = document.createElement("span"); st.className = "ht-star"; st.setAttribute("aria-hidden", "true");
    st.innerHTML = `<svg viewBox="0 0 24 24"><path d="M12 0C12.9 7.6 16.4 11.1 24 12 16.4 12.9 12.9 16.4 12 24 11.1 16.4 7.6 12.9 0 12 7.6 11.1 11.1 7.6 12 0Z" fill="url(#htStarG)"/></svg>`;
    st.addEventListener("animationstart", () => place(st)); st.addEventListener("animationend", () => run(st));
    h.append(st); run(st, true); // төгсгөлд нэмнэ: үгсийн :nth-child (өнгө) шилжихгүй
  }
}

// Нүүрний танилцуулга кино (intro2, iframe): бүдэг харагдаж, хулгана авчрахад тодорч тоглоно, холдоход зогсоно.
// Кино дотор дарж үзэж байвал (iframe focus) холдсон ч үргэлжилнэ. Hover-гүй (мэдрэгчтэй) төхөөрөмжид энгийн хэвээр.
function homeFilm() {
  const fig = $(".home-film"), fr = fig && $("iframe", fig);
  if (!fr || !matchMedia("(hover: hover)").matches) return;
  fig.classList.add("hover-play");
  const win = () => { try { return fr.contentWindow; } catch { return null; } };
  let over = false;
  const off = () => { fig.classList.remove("is-on"); win()?.__introPause?.(); };
  fig.addEventListener("mouseenter", () => { over = true; fig.classList.add("is-on"); win()?.__introPlay?.(); });
  fig.addEventListener("mouseleave", () => { over = false; if (document.activeElement !== fr) off(); });
  // кино дотор дарж үзсэний дараа хуудасны өөр газар дарвал бүдгэрч зогсоно (iframe доторх дарал энд ирэхгүй)
  document.addEventListener("pointerdown", (e) => { if (!over && !fig.contains(e.target) && fig.classList.contains("is-on")) off(); });
}

// Сайтын хандалт (хянагчийн самбарын өдөр/7 хоног/сарын статистик): хуудас бүр нэг удаа, хөтчийн санамсаргүй ID-гаар.
function siteHit() {
  if (page === "admin" || navigator.webdriver) return;
  let vid = ""; try { vid = localStorage.getItem("sg_vid") || ""; if (!vid) { vid = crypto.randomUUID?.() || Math.random().toString(36).slice(2) + Date.now().toString(36); localStorage.setItem("sg_vid", vid); } } catch {}
  let ref = ""; try { const r = document.referrer && new URL(document.referrer); ref = r ? (r.host === location.host ? "self" : r.host) : ""; } catch {}
  const body = JSON.stringify({ p: location.pathname, vid, ref }), headers = { "Content-Type": "application/json" };
  const tok = store.get("sg_token"); if (tok) headers.Authorization = "Bearer " + tok;
  fetch("/api/hit", { method: "POST", headers, body, keepalive: true }).catch(() => {});
}

/* ---------- auth + API ---------- */
const Auth = {
  get token() { return store.get("sg_token"); },
  get user() { return store.get("sg_user"); },
  set(token, user) { store.set("sg_token", token); store.set("sg_user", user); },
  clear() { store.set("sg_token", null); store.set("sg_user", null); },
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

// /api/me/home-ийг нэг хуудсанд давхар асуухгүй: 5 секундийн дотор ирсэн дуудлагууд нэг хариуг хуваалцана.
// lite — чатын самбарт (чат, багш нар л хэрэгтэй): бүтэн хариу аль хэдийн ирж байвал түүнийг ашиглана.
const homeMemo = { p: null, at: 0, lite: false };
function meHome(lite = false) {
  if (homeMemo.p && Date.now() - homeMemo.at < 5000 && (lite || !homeMemo.lite)) return homeMemo.p;
  homeMemo.at = Date.now(); homeMemo.lite = lite;
  const p = api("/api/me/home" + (lite ? "?lite=1" : ""));
  homeMemo.p = p;
  p.catch(() => { if (homeMemo.p === p) homeMemo.p = null; });
  return p;
}

// Хуудасласан жагсаалт: бие нь массив хэвээр, нийт тоо X-Total-Count толгойд (олон мянган мөрийг нэг дор татахгүй).
async function apiList(path) {
  const res = await api(path, { raw: true });
  const data = res.status === 204 ? null : await res.json().catch(() => null);
  if (!res.ok) { const e = new Error(data?.error || "Алдаа гарлаа (" + res.status + ")"); e.status = res.status; e.data = data; throw e; }
  const items = Array.isArray(data) ? data : data?.items || [];
  return { items, total: +(res.headers.get("X-Total-Count") || items.length) };
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

// Хэсэг солигдоход зөөлөн гарч ирнэ (профайлын таб, студийн хэсэг). Дуусахад класс авна — дараагийн дахин рендерт хөдлөхгүй.
function panelIn(el) {
  if (!el || reduce) return;
  clearTimeout(el._pfIn);
  el.classList.remove("pf-in", "fade-in"); void el.offsetWidth; el.classList.add("pf-in");
  el._pfIn = setTimeout(() => el.classList.remove("pf-in"), 900);
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
    // clip — hidden шиг scroll container болдоггүй: grid/flex жагсаалтын мөр (ж: чатын жагсаалт) агуулгаасаа доош шахагдахгүй.
    if (!el.style.overflow) el.style.overflow = CSS.supports?.("overflow", "clip") ? "clip" : "hidden";
    const s = document.createElement("span");
    s.className = "ripple";
    s.style.cssText = `width:${d}px;height:${d}px;left:${e.clientX - r.left - d / 2}px;top:${e.clientY - r.top - d / 2}px`;
    el.append(s);
    s.addEventListener("animationend", () => s.remove());
  }, { passive: true });
}

/* ---------- modal ---------- */
// Нээгдэх агшинд эцгийнх нь «зөөлөн гарах» анимейшнийг (stacking context үүсгэдэг) зогсооно — цонх бусад хэсгийн доор дарагдахгүй.
function openModal(el) { if (!el) return; el.parentElement?.closest(".fade-in")?.classList.remove("fade-in"); el.classList.add("open"); el.setAttribute("aria-hidden", "false"); }
function closeModal(el) {
  if (!el) return;
  el.classList.remove("open"); el.setAttribute("aria-hidden", "true");
  $$("video,audio", el).forEach((v) => v.pause());
  const p = $(".player", el); if (p) setTimeout(() => { if (!el.classList.contains("open")) p.innerHTML = ""; }, 400);
}
document.addEventListener("click", (e) => {
  const m = e.target.closest(".modal");
  if (m && ((e.target === m && !m.classList.contains("inline")) || e.target.closest("[data-close]"))) closeModal(m);
});
addEventListener("keydown", (e) => { if (e.key === "Escape") $$(".modal.open").forEach(closeModal); });

/* ---------- WebSocket: чат + мэдэгдэл (нэг холболт) ---------- */
const Live = {
  ws: null, token: null, tries: 0, handlers: new Set(),
  subs: new Map(), // нэмэлт суваг (ж: үүлэн үг) → бүртгэлийн мессеж; дахин холбогдоход дахин илгээнэ
  on(fn) { this.handlers.add(fn); },
  off(fn) { this.handlers.delete(fn); },
  send(msg) { if (this.ws?.readyState === 1) { try { this.ws.send(JSON.stringify(msg)); } catch {} } },
  sub(key, msg) { this.subs.set(key, msg); this.send(msg); },
  unsub(key) { const m = this.subs.get(key); if (!m) return; this.subs.delete(key); this.send({ ...m, t: "unsub" }); },
  // Бүлэгт шинээр орсны дараа (сувгууд холбогдох үед тооцогддог) дахин холбогдоно.
  reconnect() { const t = this.token || Auth.token; const ws = this.ws; this.ws = null; try { ws?.close(); } catch {} this.token = null; this.connect(t); },
  connect(token) {
    if (!token || (this.ws && this.token === token && this.ws.readyState <= 1)) return;
    this.token = token;
    try { this.ws?.close(); } catch {}
    const ws = new WebSocket((location.protocol === "https:" ? "wss://" : "ws://") + location.host + "/api/chat/ws?token=" + encodeURIComponent(token));
    this.ws = ws;
    // Тасраад дахин холбогдвол тасарсан хугацаанд ирсэн мессежийг сонсогчид дахин татна.
    ws.onopen = () => { this.tries = 0; this.subs.forEach((m) => this.send(m)); if (this.opened) this.handlers.forEach((h) => h({ type: "reconnected" })); this.opened = true; };
    ws.onmessage = (e) => { let d; try { d = JSON.parse(e.data); } catch { return; } this.handlers.forEach((h) => h(d)); };
    ws.onclose = () => {
      if (this.ws !== ws) return;
      // Экспоненциал backoff + jitter: сервер дахин эхлэхэд мянга мянган клиент нэг зэрэг давхцахгүй.
      const delay = Math.min(30000, 1000 * 2 ** this.tries++) * (0.5 + Math.random());
      setTimeout(() => { if (this.ws === ws) { this.ws = null; this.connect(this.token); } }, delay);
    };
  },
};
// Утас унтраад сэрэхэд холболт дуугүй тасарсан байж болно — табыг харах мөчид шалгаж дахин холбогдоно.
document.addEventListener("visibilitychange", () => { if (!document.hidden && Live.token && (!Live.ws || Live.ws.readyState > 1)) { Live.ws = null; Live.tries = 0; Live.connect(Live.token); } });

/* ---------- Чатын самбар: нэвтэрсэн хүн бүрт баруун талд байнга ----------
   Багш: суралцагчид, зочид, бүлгүүд. Суралцагч: багш нартайгаа хийсэн яриа, элссэн сургалтын бүлгүүд.
   Өргөн дэлгэцэд хуудасны баруун талд наалдсан; нарийн дэлгэцэд дээд цэсний товчоор гулсаж гарна. */
function chatRail() {
  const u = Auth.user;
  if (!u || !Auth.token || !$(".nav-links") || !["home", "profile", "course", "me"].includes(page)) {
    document.documentElement.classList.remove("has-rail", "rail-boot", "rail-collapsed"); // <head>-д урьдчилан тавьсныг буцаана
    return;
  }
  const teacher = u.role === "teacher";
  const I = {
    more: `<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor" aria-hidden="true"><circle cx="5" cy="12" r="1.9"/><circle cx="12" cy="12" r="1.9"/><circle cx="19" cy="12" r="1.9"/></svg>`,
    trash: `<svg viewBox="0 0 24 24" width="17" height="17" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M3 6h18M8 6V4h8v2M19 6l-1 14H6L5 6M10 11v6M14 11v6"/></svg>`,
    chat: `<svg data-ico="chat" viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a8 8 0 0 1-11.6 7.1L4 20l1-4.6A8 8 0 1 1 21 12Z"/></svg>`,
    x: `<svg data-ico="x" viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M6 6l12 12M18 6 6 18"/></svg>`,
    back: `<svg data-ico="back" viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 6l-6 6 6 6"/></svg>`,
    search: `<svg data-ico="search" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/></svg>`,
    live: `<svg data-ico="live" viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linejoin="round"><rect x="3" y="6" width="12" height="12" rx="2"/><path d="m15 10 6-3v10l-6-3z"/></svg>`,
    group: `<svg data-ico="group" viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><circle cx="9" cy="8" r="3.2"/><path d="M3 19c0-3.2 2.7-5 6-5s6 1.8 6 5"/><circle cx="17" cy="9" r="2.4"/><path d="M15.5 14.3c3-.3 5.5 1.5 5.5 4.2"/></svg>`,
    people: `<svg data-ico="people" viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><circle cx="8" cy="8" r="3"/><circle cx="16" cy="8" r="3"/><path d="M2.5 19c0-3 2.5-5 5.5-5s5.5 2 5.5 5M10.5 19c0-3 2.5-5 5.5-5s5.5 2 5.5 5"/></svg>`,
    plus: `<svg data-ico="plus" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"><path d="M12 5v14M5 12h14"/></svg>`,
    collapse: `<svg data-ico="collapse" viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m7 6 6 6-6 6M13 6l6 6-6 6"/></svg>`,
  };
  for (const k in I) I[k] = I[k].replace("<svg ", `<svg data-ico="${k}" `); // хөдөлгөөнт дүрсэнд тааруулах нэр
  const sameDay = (t) => new Date(t).toDateString() === new Date().toDateString();
  // Хумьсан хавтас "би энд байна": шинэ мессеж ирэхэд нэг удаа дохино.
  const nudgeTab = () => { const tab = $("#railTab"); if (!tab) return; tab.classList.remove("nudge"); void tab.offsetWidth; tab.classList.add("nudge"); };
  // Самбар нэмэгдэх агшинд гулсах/шахагдах transition-гүй (rail-boot), дараагийн кадруудад сэргээнэ.
  document.documentElement.classList.add("rail-boot", "has-rail");
  requestAnimationFrame(() => requestAnimationFrame(() => setTimeout(() => document.documentElement.classList.remove("rail-boot"), 50)));
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
  let online = new Set(), seenAt = {}; // жагсаалтын хүмүүсээс одоо сайтад байгаа нь, сүүлд идэвхтэй байсан мөч (/api/me/presence)
  // Өргөн дэлгэцэд чат баруун талд наалддаг; хумих (⟩) товчоор нуугдаж, баруун ирмэгийн хавтсаар дахин нээгдэнэ.
  const wide = () => matchMedia("(min-width:1280px)").matches;
  document.body.insertAdjacentHTML("beforeend", `<button class="rail-tab" id="railTab" aria-label="Чат нээх" title="Чат нээх"><span class="rail-tab-ico">${I.chat}<i class="rail-tab-dot"></i></span><span class="rail-tab-txt">Чат</span><span class="bell-badge" id="railTabBadge" hidden></span></button>`);
  const setCollapsed = (c) => { document.documentElement.classList.toggle("rail-collapsed", c); try { localStorage.setItem("sg_rail_collapsed", c ? "1" : "0"); } catch {} };
  try { if (localStorage.getItem("sg_rail_collapsed") === "1") setCollapsed(true); } catch {}
  // Нээлттэй яриа, самбарын төлөвийг тухайн табд хадгална: refresh хийхэд яг тэр яриа дахин нээгдэнэ.
  const ss = { get: (k) => { try { return sessionStorage.getItem(k); } catch { return null; } }, set: (k, v) => { try { v == null ? sessionStorage.removeItem(k) : sessionStorage.setItem(k, v); } catch {} } };
  const setRail = (open) => {
    if (wide()) { setCollapsed(!open); }
    document.body.classList.toggle("rail-open", open && !wide()); $("#railToggle").setAttribute("aria-expanded", String(open));
    ss.set("sg_rail_open", open ? "1" : null);
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
      list.forEach((c) => out.set(c.id, { id: c.id, kind: c.kind || "", title: c.visitor_name, sub: c.kind === "group" ? "Бүлэг · сургалтын суралцагчид" : c.user_id ? "Суралцагч" : "Зочин", peer: c.kind === "group" || c.kind === "team" ? "" : (c.teacher_id === u.id ? c.user_id : c.teacher_id) || "", last: c.last_message, at: c.last_message_at }));
    }
    const h = await meHome(true).catch(() => null);
    (h?.chats || []).forEach((c) => { if (!out.has(c.id)) out.set(c.id, { id: c.id, kind: c.kind || "", title: c.kind === "group" || c.kind === "team" ? c.title : c.teacher?.display_name || "", sub: c.kind === "group" ? "Бүлэг · " + (c.teacher?.display_name || "") : c.kind === "team" ? "Сурагчдын бүлэг" : c.kind === "dm" ? "Ангийн найз" : "Багш", user: c.kind === "team" ? null : c.teacher, peer: c.kind === "group" || c.kind === "team" ? "" : c.teacher?.id || "", course: c.course_id || "", last: c.last_message, at: c.last_message_at }); });
    return [...out.values()].sort((a, b) => new Date(b.at) - new Date(a.at));
  };
  // Хүмүүс: багшид — суралцагчид нь, суралцагчид — багш нар нь (яриа эхлээгүй байсан ч нэг дарж эхэлнэ).
  let people = [];
  const loadPeople = async () => {
    try {
      if (teacher) {
        const { items, total } = await apiList(peopleURL());
        people = items.map(toPerson); peopleTotal = total; peopleBase = items.length;
      } else {
        const [h, mates] = await Promise.all([meHome(true), apiList(peopleURL()).catch(() => ({ items: [], total: 0 }))]);
        people = (h.teachers || []).map((t) => ({ id: t.id, user: t, name: t.display_name, sub: "Багш" + (t.headline ? " · " + t.headline : ""), convId: "", start: () => api(`/api/teachers/${t.username}/chat`, { method: "POST" }) }))
          .concat(mates.items.map(toPerson));
        peopleTotal = mates.total; peopleBase = mates.items.length;
      }
    } catch { people = []; peopleTotal = peopleBase = 0; }
  };
  // Олон мянган суралцагчтай үед эхний 200-г л ачаална; бусдыг нь нэрээр серверээс хайж жагсаалтад нэмнэ.
  let peopleTotal = 0, peopleBase = 0, findT = 0, findQ = "";
  const PEOPLE = 200;
  const peopleURL = (q = "") => (teacher ? `/api/me/students?sort=rail&limit=${q ? 50 : PEOPLE}` : `/api/me/classmates?limit=${q ? 50 : PEOPLE}`) + (q ? "&q=" + encodeURIComponent(q) : "");
  const toPerson = (r) => teacher
    ? { id: r.user.id, user: r.user, name: r.user.display_name, sub: "Суралцагч" + (r.courses?.[0]?.course_title ? " · " + r.courses[0].course_title : ""), convId: r.conv_id || "", start: () => api(`/api/me/students/${r.user.id}/chat`, { method: "POST" }) }
    : { id: r.user.id, user: r.user, name: r.user.display_name, sub: "Ангийн найз · " + (r.via || r.courses?.[0] || ""), convId: r.conv_id || "", mate: true, start: () => api(`/api/me/dm/${r.user.id}`, { method: "POST" }) };
  const findPeople = () => {
    clearTimeout(findT);
    const q = $("#railSearch").value.trim();
    if (q.length < 2 || peopleTotal <= peopleBase || q === findQ) return;
    findT = setTimeout(async () => {
      findQ = q;
      try {
        const { items } = await apiList(peopleURL(q));
        const known = new Set(people.map((p) => p.id));
        people = people.concat(items.filter((r) => !known.has(r.user.id)).map(toPerson));
        if ($("#railSearch").value.trim() === q) draw();
      } catch {}
    }, 300);
  };
  let firstDraw = true;
  const draw = () => {
    const anim = firstDraw ? "rail-in" : ""; firstDraw = false;
    const q = $("#railSearch").value.trim().toLowerCase(), hit = (s) => !q || s.toLowerCase().includes(q);
    const shown = convs.filter((c) => hit(c.title));
    const convUsers = new Set(convs.map((c) => c.user?.id).filter(Boolean));
    const isOn = (id) => !!id && online.has(id);
    const ppl = people.filter((p) => hit(p.name) && !(p.convId && convs.some((c) => c.id === p.convId)) && !convUsers.has(p.id))
      .sort((a, b) => isOn(b.id) - isOn(a.id)); // онлайн нь дээрээ
    const userOf = (c) => c.user || people.find((p) => p.id === c.peer)?.user; // багшийн талын яриад зураг алга — суралцагчийн жагсаалтаас
    const letter = (name) => `<span class="avatar avatar-sm" style="--h:${hueOfName(name)}">${esc(name.slice(0, 1).toUpperCase())}</span>`;
    // Онлайн: яг одоо сайтад байгаа хүмүүс хамгийн дээр нэг мөрөнд (ногоон цагирагтай) — нэг дарж чатлана.
    const live = new Map();
    for (const c of convs) if (isOn(c.peer) && !live.has(c.peer)) { const cu = userOf(c); live.set(c.peer, { id: c.peer, name: cu?.display_name || c.title || "", user: cu, conv: c.id }); }
    for (const p of people) if (isOn(p.id) && !live.has(p.id)) live.set(p.id, { id: p.id, name: p.name || "", user: p.user, conv: p.convId });
    const lv = [...live.values()].filter((x) => hit(x.name));
    let i = 0;
    // Нэр: 1-р мөрөнд эхний үг, 2-р мөрөнд үлдсэн нь (үгийн дундуур тасрахгүй, урт бол «…»).
    const twoLines = (n) => { const [a = "", ...b] = n.trim().split(/\s+/); return `<b>${esc(a)}</b>` + (b.length ? `<b>${esc(b.join(" "))}</b>` : ""); };
    const liveItem = (x) => `<button type="button" class="rail-live-item ${anim}" style="--i:${i++}" data-live="${esc(x.id)}" data-conv="${esc(x.conv || "")}" title="${esc(x.name)} · онлайн">${x.user ? avatarHTML(x.user) : letter(x.name)}<span>${twoLines(x.name)}</span></button>`;
    // Мөр бүрд hover-д дугуй «⋯» товч (товч дотор товч байж болохгүй тул хажууд нь, мөрийн бүрхүүл дотор).
    const item = (c) => { const n = unread.get(c.id) || 0, cu = userOf(c); return `<div class="rail-row ${anim}" style="--i:${i++}"><button class="rail-item ${n ? "unread" : ""} ${c.id === cur ? "active" : ""} ${isOn(c.peer) ? "on" : ""}" data-id="${esc(c.id)}">${c.kind === "group" || c.kind === "team" ? `<span class="avatar avatar-sm group ${c.kind}">${c.kind === "team" ? I.people : I.group}</span>` : cu ? avatarHTML(cu) : letter(c.title)}
      <span class="grow"><span class="rail-row1"><strong>${esc(c.title)}</strong>${c.at ? `<time class="rail-time">${sameDay(c.at) ? fmtTime(c.at) : fmtDay(c.at)}</time>` : ""}</span><small>${c.last ? esc(c.last) : `<i>${esc(c.sub)}</i>`}</small></span>${n ? `<span class="rail-unread">${n > 9 ? "9+" : n}</span>` : ""}</button><button type="button" class="rail-more" data-more="${esc(c.id)}" aria-label="Чатын үйлдэл" aria-haspopup="menu" aria-expanded="false" title="Бусад">${I.more}</button></div>`; };
    const person = (p) => `<button class="rail-item ${anim} rail-person ${isOn(p.id) ? "on" : ""}" style="--i:${Math.min(i++, 12)}" data-person="${esc(p.id)}">${avatarHTML(p.user)}<span class="grow"><strong>${esc(p.name)}</strong><small><b>${isOn(p.id) ? "Онлайн · " : ""}${esc(p.sub)}</b></small></span><span class="rail-new" aria-hidden="true">${I.chat}</span></button>`;
    const sec = (title, n, cls = "") => `<div class="rail-sec ${cls}"><span>${title}</span><em>${n}</em></div>`;
    $("#railList").innerHTML = (lv.length ? sec("Онлайн", lv.length, "live") + `<div class="rail-live">${lv.map(liveItem).join("")}</div>` : "") +
      (shown.length ? sec("Сүүлийн яриа", shown.length) + shown.map(item).join("") : "") + (ppl.length ? sec("Хүмүүс", !q && peopleTotal > peopleBase ? peopleTotal : ppl.length) + ppl.map(person).join("") + (!q && peopleTotal > peopleBase ? `<p class="muted small rail-more-hint">Бусад ${peopleTotal - peopleBase} хүнийг нэрээр нь хайж олно уу</p>` : "") : "") ||
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
  // Ярианы гарчиг: жагсаалтаас нэр (суралцагчид багшийн нэр), эс бөгөөс ярианы мэдээллээс.
  let curConv = null;
  const whoHTML = (c) => {
    const known = convs.find((x) => x.id === c.id), name = known?.title || c.visitor_name || "?";
    // Хувийн ярианы нөгөө тал: онлайн бол аватар дээр ногоон цэг + «Онлайн», эс бөгөөс «Идэвхтэй байсан · 5 минутын өмнө».
    const pid = c.kind === "group" || c.kind === "team" ? "" : known?.peer || (c.teacher_id === u.id ? c.user_id : c.teacher_id) || "";
    const on = !!pid && online.has(pid), was = !on && pid && seenAt[pid], ku = known?.user || people.find((p) => p.id === pid)?.user;
    const av = c.kind === "group" || c.kind === "team" ? `<span class="avatar avatar-sm group ${c.kind}">${c.kind === "team" ? I.people : I.group}</span>`
      : ku ? avatarHTML(ku, "avatar-sm") : `<span class="avatar avatar-sm" style="--h:${hueOfName(name)}">${esc(name.slice(0, 1).toUpperCase())}</span>`;
    // Чатлаж буй багш дээр дарвал профайл руу, сургалтын бүлэг дээр дарвал сургалт руу (суралцагч, ангийн найз — нээлттэй профайлгүй).
    const course = known?.course || c.course_id || "";
    const href = c.kind === "group" && course ? `/c/${encodeURIComponent(course)}` : !teacher && !c.kind && known?.user?.username ? `/t/${encodeURIComponent(known.user.username)}` : "";
    const role = c.kind === "team" ? "Сурагчдын бүлэг" : c.kind === "dm" ? "Ангийн найз" : curGroup ? "Бүлэг чат · сургалтын суралцагчид" : known?.sub || (c.user_id ? "Суралцагч" : "Зочин");
    const body = `<span class="rw-av ${on ? "on" : ""}">${av}</span><span class="rw-txt"><strong>${esc(name)}</strong><small class="muted">${on ? `<b class="rw-on">Онлайн</b> · ` : was ? `Идэвхтэй байсан · ${seenAgo(was)} · ` : ""}${role}</small></span>`;
    return href ? `<a class="rw-link" href="${href}" title="${c.kind === "group" ? "Сургалт руу очих" : "Багшийн профайл руу очих"}">${body}</a>` : body;
  };
  // show=false: самбарын нээлттэй/хумьсан төлөвийг өөрчлөхгүй (refresh-ийн дараа сэргээх, дахин холбогдох).
  const open = async (id, show = true) => {
    const same = cur === id && !thread.hidden;
    if (meetConv && meetConv !== id) meetOff(); // өөр яриа нээвэл Meet-ийн холбоос хүлээхээ болино
    cur = id; unread.delete(id); badge(); if (show) setRail(true); draw();
    ss.set("sg_rail_conv", id);
    home.hidden = true; thread.hidden = false;
    if (!same) { $("#railWho").innerHTML = `<span class="muted small">Ачаалж байна…</span>`; msgs.innerHTML = ""; }
    try {
      const d = await api(`/api/chat/${id}/messages`);
      if (cur !== id) return;
      const c = d.conversation; curGroup = c.kind === "group" || c.kind === "team"; myRole = d.me; curConv = c;
      $("#railWho").innerHTML = whoHTML(c);
      th.set(d); if (show) form.body.focus();
      if (!same) { msgs.classList.remove("fade-in"); void msgs.offsetWidth; msgs.classList.add("fade-in"); } // өөр яриа нээгдэхэд зөөлөн
    } catch (e) {
      if (!show && e.status >= 400 && e.status < 500) { ss.set("sg_rail_conv", null); cur = null; thread.hidden = true; home.hidden = false; draw(); return; } // хадгалсан яриа байхгүй болсон
      $("#railWho").innerHTML = `<span class="form-error" style="margin:0">${esc(e.message)}</span>`;
    }
  };
  $("#railBack").onclick = () => { meetOff(); cur = null; ss.set("sg_rail_conv", null); thread.hidden = true; home.hidden = false; draw(); };
  $("#railSearch").addEventListener("input", () => { draw(); findPeople(); });
  $("#railList").addEventListener("click", (e) => { const mo = e.target.closest("[data-more]"); if (mo) return moreMenu(mo); const lv = e.target.closest("[data-live]"); if (lv) return lv.dataset.conv ? open(lv.dataset.conv) : startWith(lv.dataset.live); const p = e.target.closest("[data-person]"); if (p) return startWith(p.dataset.person); const it = e.target.closest(".rail-item"); if (it) open(it.dataset.id); });
  $("#railList").addEventListener("scroll", () => closeMenu(), { passive: true });
  // «⋯» цэс: «Чат устгах» → баталгаажуулах (зөвхөн өөрийн талд устгана, нөгөө тал хэвээр харна).
  function closeMenu() { const m = $(".rail-menu"); if (!m) return; m._off?.(); m.remove(); $$(".rail-more[aria-expanded=true]").forEach((b) => b.setAttribute("aria-expanded", "false")); }
  function moreMenu(btn) {
    const was = btn.getAttribute("aria-expanded") === "true"; closeMenu(); if (was) return;
    const id = btn.dataset.more, conv = convs.find((x) => x.id === id); if (!conv) return;
    const m = document.createElement("div"); m.className = "rail-menu"; m.setAttribute("role", "menu");
    const place = () => { const r = btn.getBoundingClientRect(); m.style.top = Math.max(8, Math.min(r.bottom + 6, innerHeight - m.offsetHeight - 8)) + "px"; m.style.left = Math.max(8, Math.min(r.right - m.offsetWidth, innerWidth - m.offsetWidth - 8)) + "px"; };
    const step1 = () => { m.innerHTML = `<button type="button" role="menuitem" class="danger" data-rm="ask">${I.trash}<span>Чат устгах</span></button>`; place(); };
    m.addEventListener("click", async (e) => {
      const b = e.target.closest("[data-rm]"); if (!b) return;
      e.stopPropagation();
      if (b.dataset.rm === "ask") { m.innerHTML = `<p><b>«${esc(conv.title || "Чат")}» чатыг устгах уу?</b>Мессежүүд зөвхөн таны талд устгагдана — нөгөө тал харсаар байна.</p><div class="rm-acts"><button type="button" data-rm="no">Болих</button><button type="button" class="rm-del" data-rm="yes">Устгах</button></div>`; place(); $(".rm-del", m).focus(); return; }
      if (b.dataset.rm === "no") return closeMenu();
      b.disabled = true;
      try {
        await api(`/api/chat/${encodeURIComponent(id)}`, { method: "DELETE" });
        closeMenu(); convs = convs.filter((x) => x.id !== id); unread.delete(id); badge();
        if (cur === id) { cur = null; ss.set("sg_rail_conv", null); thread.hidden = true; home.hidden = false; }
        draw(); toast("🗑 Чат устгагдлаа");
      } catch (x) { b.disabled = false; toast(x.message, true); }
    });
    document.body.append(m); step1(); btn.setAttribute("aria-expanded", "true");
    const away = (e) => { if (!m.contains(e.target) && e.target !== btn && !btn.contains(e.target)) closeMenu(); };
    const esc_ = (e) => { if (e.key === "Escape") closeMenu(); };
    setTimeout(() => { document.addEventListener("pointerdown", away); document.addEventListener("keydown", esc_); addEventListener("resize", closeMenu); }, 0);
    m._off = () => { document.removeEventListener("pointerdown", away); document.removeEventListener("keydown", esc_); removeEventListener("resize", closeMenu); };
  }
  // 📹 Google Meet: Google-ээ холбосон багшид холбоос серверт автоматаар үүсэж шууд илгээгдэнэ. Холбоогүй бол
  // meet.google.com/new шинэ табд нээгдэж уулзалт үүснэ → багш тэндээс холбоосыг хуулаад буцаж ирэхэд энд гарч ирнэ →
  // «Илгээх». (meet.google.com/new-ийг өөрийг нь илгээвэл дарсан хүн бүр тусдаа шинэ уулзалтад орно — жинхэнэ
  // уулзалтын холбоосыг илгээнэ.)
  const MEET_RE = /https:\/\/meet\.google\.com\/[a-z]{3}-[a-z]{4}-[a-z]{3}/i;
  let meetConv = null, meetUrl = "";
  const meetOff = () => { meetConv = null; meetUrl = ""; removeEventListener("focus", meetGrab); document.removeEventListener("visibilitychange", meetGrab); form.querySelector(".meet-bar")?.remove(); };
  const meetBar = (state) => {
    let bar = form.querySelector(".meet-bar");
    if (!bar) { bar = document.createElement("div"); bar.className = "meet-bar"; bar.setAttribute("role", "status"); form.prepend(bar); }
    const x = `<button type="button" class="mb-x" data-mb="x" aria-label="Болих" title="Болих">✕</button>`;
    bar.classList.toggle("got", state === "got");
    bar.innerHTML = state === "got"
      ? `<span class="mb-ico">${I.live}</span><span class="mb-txt"><b>Холбоос бэлэн</b><small class="mb-url">${esc(meetUrl.replace("https://", ""))}</small></span><button type="button" class="mb-send" data-mb="send">Илгээх</button>${x}`
      : state === "new"
        ? `<span class="mb-ico">${I.live}</span><span class="mb-txt"><b>Meet холболт салсан</b><small>Шинэ уулзалт нээж холбоосыг илгээнэ</small></span><button type="button" class="mb-send" data-mb="new">Meet нээх</button>${x}`
        : `<span class="mb-ico">${I.live}</span><span class="mb-txt"><b>Meet шинэ табд нээгдлээ</b><small>Холбоосыг хуулаад (⧉) буцаж ирээрэй</small></span><button type="button" class="mb-paste" data-mb="paste">Тавих</button>${x}`;
  };
  const meetFrom = (text) => { const m = (text || "").match(MEET_RE); if (!m || !meetConv) return false; meetUrl = m[0]; meetBar("got"); return true; };
  async function meetGrab() {
    if (!meetConv || meetUrl || document.visibilityState !== "visible") return;
    try { meetFrom(await navigator.clipboard.readText()); } catch {} // санах ойг унших зөвшөөрөлгүй бол «Тавих»
  }
  const meetNew = () => {
    window.open("https://meet.google.com/new", "_blank", "noopener");
    meetConv = cur; meetUrl = ""; meetBar("wait");
    addEventListener("focus", meetGrab); document.addEventListener("visibilitychange", meetGrab);
  };
  form.addEventListener("click", async (e) => {
    const mb = e.target.closest("[data-mb]")?.dataset.mb;
    if (mb === "x") return meetOff();
    if (mb === "new") return meetNew();
    if (mb === "paste") {
      try { if (meetFrom(await navigator.clipboard.readText())) return; } catch {}
      toast("Хуулсан Meet холбоос олдсонгүй — meet.google.com/… холбоосыг хуулаад дахин «Тавих» дарна уу", true); return;
    }
    if (mb === "send") {
      const conv = meetConv, url = meetUrl; if (!conv || !url) return;
      e.target.closest("[data-mb]").disabled = true;
      try { const m = await api(`/api/chat/${conv}/messages`, { method: "POST", body: { body: "📹 Google Meet уулзалт бэлэн боллоо, нэгдээрэй: " + url } }); meetOff(); if (conv === cur) append(m); toast("📹 Meet холбоос илгээгдлээ"); }
      catch (x) { e.target.closest("[data-mb]").disabled = false; toast(x.message, true); }
      return;
    }
    if (!e.target.closest("#railMeet") || !cur) return;
    if (!u.meet_connected) return meetNew();
    try { const d = await api(`/api/chat/${cur}/meet`, { method: "POST" }); append(d.message); toast("📹 Meet холбоос илгээгдлээ"); }
    catch (x) { if (x.status === 428 || x.status === 503) { meetConv = cur; meetUrl = ""; meetBar("new"); } else toast(x.message, true); }
  });
  // Багш Meet-ийн холбоосыг бичих талбарт шууд буулгавал ч мөн «Илгээх» бэлэн болно.
  form.addEventListener("paste", (e) => { if (meetConv && !meetUrl && meetFrom(e.clipboardData?.getData("text"))) e.preventDefault(); });
  // Онлайн төлөв: жагсаалтын хүмүүсээс хэн одоо сайтад байгааг 30 секунд тутам асууна (таб нуугдсан үед асуухгүй).
  // Сервер зөвхөн жагсаалтад тань харагдсан хүмүүсийнхийг өгдөг тул жагсаалт ачаалсны дараа асууна.
  let presT = 0;
  const pollPresence = async () => {
    clearTimeout(presT);
    if (document.hidden) return;
    try {
      const d = await api("/api/me/presence"), next = new Set(d.online || []);
      const changed = next.size !== online.size || [...next].some((id) => !online.has(id));
      online = next; seenAt = d.seen || {};
      if (changed) draw();
      if (curConv && cur === curConv.id && !thread.hidden) $("#railWho").innerHTML = whoHTML(curConv);
    } catch {}
    presT = setTimeout(pollPresence, 60000); // минут тутам (жагсаалт ~200 хүнээр хязгаартай, санах ойгоос)
  };
  document.addEventListener("visibilitychange", () => { if (!document.hidden) pollPresence(); });
  Live.on((d) => {
    if (d.type === "reconnected") { if (cur) open(cur, false); loadPeople().then(load).then(pollPresence); return; } // тасарсан хугацааны мессежүүд; сервер дахин асвал онлайн төлөвийн жагсаалт ч шинэчлэгдэнэ
    if (THREAD_EVENTS.includes(d.type)) { th.event(d); return; }
    if (d.type !== "message") return;
    const m = d.message;
    if (m.conversation_id === cur) append(m);
    else if (m.sender_id !== u.id) { unread.set(m.conversation_id, (unread.get(m.conversation_id) || 0) + 1); badge(); if (window.innerWidth >= 1280 && document.documentElement.classList.contains("rail-collapsed")) nudgeTab(); }
    clearTimeout(timer); timer = setTimeout(load, 400);
  });
  Live.connect(Auth.token);
  // Refresh-ийн өмнө нээлттэй байсан яриаг (жагсаалт ачаалсны дараа — гарчиг зөв нэртэй) сэргээнэ;
  // нарийн дэлгэцэд самбар нээлттэй байсан бол шууд нээнэ.
  if (ss.get("sg_rail_open") === "1" && !wide()) setRail(true);
  { const saved = ss.get("sg_rail_conv"); if (saved) open(saved, false); } // жагсаалт хүлээлгүй шууд яриа (жагсаалт→яриа солигдож анивчихгүй)
  loadPeople().then(load).then(() => { if (curConv && cur === curConv.id) $("#railWho").innerHTML = whoHTML(curConv); pollPresence(); });
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
  $$("[data-logout]").forEach((b) => b.addEventListener("click", () => { Auth.clear(); store.set("sg_main", null); location.href = "/"; }));
  if (u?.is_admin && !$(".nav-admin")) $(".nav-links")?.insertAdjacentHTML("afterbegin", `<a href="/admin" class="nav-link nav-admin">Мэдүүлэг</a>`);
  if (u && Auth.token) { bell(); consentGate().then((shown) => { if (!shown) phoneGate(); }); }
  orgNav(u);
  // Удирддаг байгууллага (хянагч шинээр холбосон г.м.) — хадгалсан мэдээлэл 2 минутаас хуучин бол /api/me-ээс шинэчилнэ.
  if (u && Auth.token) {
    let at = 0; try { at = +sessionStorage.getItem("sg_me_at") || 0; } catch {}
    if (Date.now() - at > 120000) {
      try { sessionStorage.setItem("sg_me_at", String(Date.now())); } catch {}
      const tok = Auth.token;
      api("/api/me").then((me) => { if (Auth.token !== tok) return; Auth.set(tok, me); orgNav(me); }).catch(() => {});
    }
  }
}

/* ---------- Байгууллага: удирдагч өөрийн бүртгэлээс байгууллагын профайл руу шилжиж удирдана (Facebook Page шиг) ----------
   sg_main — шилжихээс өмнөх хувийн бүртгэлийн токен; «буцах»-д сэргээнэ. */
function orgNav(u) {
  const links = $(".nav-links"); if (!links) return;
  $$(".nav-org").forEach((x) => x.remove()); $(".org-acting")?.remove();
  if (!u || !Auth.token) return;
  const main = store.get("sg_main");
  if (u.org && main?.token) {
    links.insertAdjacentHTML("afterbegin", `<button type="button" class="btn btn-sm nav-org nav-org-back" data-org-back title="Хувийн бүртгэл рүү буцах">← ${esc(main.user?.display_name || "Хувийн бүртгэл")}</button>`);
    document.body.insertAdjacentHTML("afterbegin", `<div class="org-acting" role="status"><span>🏫 Та <b>${esc(u.display_name)}</b> байгууллагын нэрийн өмнөөс ажиллаж байна</span><button type="button" class="linkish" data-org-back>Хувийн бүртгэл рүү буцах</button></div>`);
    return;
  }
  if (!u.orgs?.length) return;
  const btn = (o) => `<button type="button" class="nav-org-item" data-org-switch="${esc(o.id)}" title="«${esc(o.display_name)}» байгууллагыг удирдах"><span class="avatar avatar-xs" style="--h:${hueOfName(o.username)}">${o.avatar_url ? `<img src="${esc(o.avatar_url)}" alt="">` : esc((o.display_name || "?").trim().slice(0, 1).toUpperCase())}</span><span class="nav-org-name">${esc(o.display_name)}</span></button>`;
  links.insertAdjacentHTML("afterbegin", u.orgs.length === 1
    ? `<span class="nav-org">${btn(u.orgs[0])}</span>`
    : `<details class="nav-org"><summary class="nav-org-item">🏫 Байгууллага <i>${u.orgs.length}</i></summary><div class="nav-org-menu">${u.orgs.map(btn).join("")}</div></details>`);
}
document.addEventListener("click", async (e) => {
  const sw = e.target.closest("[data-org-switch]"), back = e.target.closest("[data-org-back]");
  if (sw) {
    sw.disabled = true;
    try {
      const d = await api(`/api/me/orgs/${sw.dataset.orgSwitch}/switch`, { method: "POST" });
      store.set("sg_main", { token: Auth.token, user: Auth.user });
      Auth.set(d.token, d.user); location.href = "/t/" + d.user.username + "#courses";
    } catch (x) { toast(x.message, true); sw.disabled = false; }
  }
  if (back) {
    const m = store.get("sg_main"); store.set("sg_main", null);
    try { sessionStorage.removeItem("sg_me_at"); } catch {}
    if (m?.token) { Auth.set(m.token, m.user); location.href = "/me"; } else { Auth.clear(); location.href = "/login"; }
  }
});

/* ---------- Утас баталгаажуулах (Verify.MN): хэрэглэгч өөрийн утаснаас 144773 руу кодыг SMS-ээр илгээнэ ----------
   smsVerify(box, v, check, done): заавар, код, «SMS бичих», хугацаа тоолуур; 3 секунд тутам check(token)-оор шалгаж,
   баталгаажмагц done(хариу). Хугацаа дуусвал/алдаа гарвал restart() дуудах товч. */
function smsVerify(box, v, check, done, restart) {
  const ph = String(v.phone || "").replace(/^(\d{4})(\d{4})$/, "$1 $2");
  box.innerHTML = `<div class="sv">
    <p class="sv-lead">${icon("phone", 18)}<span><b>${esc(ph)}</b> дугаараасаа <b>${esc(v.shortcode)}</b> руу доорх кодыг SMS-ээр илгээнэ үү</span></p>
    <div class="sv-code" aria-label="Илгээх код">${esc(v.text).split("").map((c) => `<i>${c}</i>`).join("")}</div>
    <a class="btn btn-gold btn-block sv-sms" href="${esc(v.sms_uri)}">${icon("chat", 18)} SMS бичих</a>
    <p class="sv-status" role="status"><span class="sv-dot"></span><span class="sv-msg">SMS хүлээж байна…</span><b class="sv-left"></b></p>
    <p class="sv-note">Компьютер дээр байгаа бол утаснаасаа <b>${esc(v.shortcode)}</b> дугаар руу <b>${esc(v.text)}</b> гэж бичиж илгээнэ үү. SMS-ийн үнэ 150₮ (операторын тарифаар).</p>
    <button type="button" class="linkish sv-again" hidden>Дахин код авах</button></div>`;
  const msg = $(".sv-msg", box), left = $(".sv-left", box), again = $(".sv-again", box), st = $(".sv-status", box);
  const end = new Date(v.expires_at).getTime();
  let stop = false, timer = 0;
  const finish = (text, bad) => { stop = true; clearTimeout(timer); st.classList.toggle("bad", !!bad); msg.textContent = text; left.textContent = ""; again.hidden = !restart; };
  again.onclick = () => restart?.();
  const countdown = () => {
    if (stop || !box.isConnected) return clearInterval(iv);
    const sec = Math.max(0, Math.round((end - Date.now()) / 1000));
    left.textContent = `${Math.floor(sec / 60)}:${String(sec % 60).padStart(2, "0")}`;
  };
  const iv = setInterval(countdown, 1000);
  countdown();
  const tick = async () => {
    if (stop || !box.isConnected) return;
    try {
      const r = await check(v.token);
      if (stop) return;
      if (r.status === "verified") { stop = true; st.classList.add("ok"); msg.textContent = "Баталгаажлаа ✓"; left.textContent = ""; return done(r); }
      if (r.status === "expired" || r.status === "none") return finish(r.error || "Хугацаа дууссан", true);
    } catch (x) { if (x.status && x.status < 500) return finish(x.message, true); } // сүлжээ/серверийн түр алдаа — дахин шалгана
    if (Date.now() > end + 5000) return finish("Хугацаа дууссан — дахин код авна уу", true);
    timer = setTimeout(tick, 3000);
  };
  timer = setTimeout(tick, 2500);
}

// Нэвтэрсэн хэрэглэгчээс нууц үг сэргээх утасны дугаарыг (Verify.MN идэвхтэй үед) нэг удаа асууна — зөвхөн бичээд
// хадгална, SMS шаардахгүй (шинэ хэрэглэгчид ачаалал болохгүй). Утсаар сэргээх үед тэр дугаараас SMS ирснээр баталгаажна.
async function phoneGate() {
  if (["legal", "login", "reset", "error", "admin"].includes(page) || !Auth.token || Auth.user?.phone_verified || Auth.user?.phone_pending || $(".cg")) return;
  let u; try { u = await api("/api/me"); } catch { return; }
  Auth.set(Auth.token, u);
  if (!u.phone_required || $(".cg")) return;
  const el = document.createElement("div");
  el.className = "cg pg"; el.setAttribute("role", "dialog"); el.setAttribute("aria-modal", "true"); el.setAttribute("aria-labelledby", "pgT");
  el.innerHTML = `<div class="cg-card">
    <span class="cg-ico" aria-hidden="true">${icon("phone", 26)}</span>
    <h3 id="pgT">Утасны дугаараа үлдээнэ үү</h3>
    <p>Нууц үгээ мартвал энэ дугаараараа хялбар сэргээнэ. Дугаар тань бусдад харагдахгүй.</p>
    <form class="pg-form"><label>Гар утасны дугаар<input name="phone" type="tel" inputmode="numeric" autocomplete="tel-national" maxlength="12" placeholder="9911 2233" required value="${esc((u.phone || "").replace(/^(\d{4})(\d{4})$/, "$1 $2"))}"></label>
      <button class="btn btn-gold btn-block">Хадгалах</button></form>
    <p class="cg-err" role="alert"></p>
  </div>`;
  const behind = [...document.body.children];
  behind.forEach((c) => (c.inert = true));
  document.body.append(el); document.documentElement.classList.add("cg-open");
  const f = $(".pg-form", el), errEl = $(".cg-err", el);
  f.phone.focus();
  f.onsubmit = async (e) => {
    e.preventDefault(); errEl.textContent = "";
    const b = $("button", f); b.disabled = true;
    try {
      const me = await api("/api/me/phone", { method: "PUT", body: { phone: f.phone.value } });
      Auth.set(Auth.token, me); behind.forEach((c) => (c.inert = false));
      el.remove(); document.documentElement.classList.remove("cg-open");
      toast("✅ Дугаар хадгалагдлаа. Нууц үгээ мартвал «Утсаар» сэргээнэ.");
    } catch (x) { errEl.textContent = x.message; b.disabled = false; }
  };
}

// Нөхцөл, нууцлалын бодлогын одоогийн хувилбарыг зөвшөөрөөгүй бол (Google г.м.-ээр шинээр орсон, эсвэл бодлого шинэчлэгдсэн)
// зөвшөөрөх хүртэл сайтыг хааж асууна. Хуудсуудыг өөрсдийг нь уншихад саад болохгүйн тулд тэнд асуухгүй.
async function consentGate() {
  const ver = $('meta[name="terms-version"]')?.content;
  if (!ver || page === "legal" || Auth.user?.consent_version === ver) return;
  let u; try { u = await api("/api/me"); } catch { return; }
  Auth.set(Auth.token, u);
  if (!u.consent_needed || $(".cg")) return;
  const teacher = u.role === "teacher", again = !!u.consent_version && u.consent_version !== ver;
  const pts = teacher
    ? ["Оруулсан агуулгынхаа зохиогчийн эрхийг та хариуцна.", "Суралцагчдын мэдээллийг зөвхөн сургалт явуулахад ашиглана.", "Профайлаа үзсэн, дугаараа нээсэн хүний нэр танд харагдана."]
    : ["Хичээл, шалгалтын үеэр идэвх тань тэмдэглэгдэнэ — камер, микрофон ашиглахгүй.", "Багш зөвхөн өөрийн сургалтын суралцагчдын ахиц, дүнг харна.", "Таны мэдээллийг худалдахгүй, зар сурталчилгаанд ашиглахгүй."];
  const el = document.createElement("div");
  el.className = "cg"; el.setAttribute("role", "dialog"); el.setAttribute("aria-modal", "true"); el.setAttribute("aria-labelledby", "cgT");
  el.innerHTML = `<div class="cg-card">
    <span class="cg-ico" aria-hidden="true">${icon("shield", 26)}</span>
    <h3 id="cgT">${again ? "Нөхцөл шинэчлэгдлээ" : "Нөхцөл ба нууцлал"}</h3>
    <p>${again ? "Үйлчилгээний нөхцөл, нууцлалын бодлого шинэчлэгдсэн тул" : "surgalt.mn-ийг ашиглахын тулд"} доорх баримтуудыг уншиж, зөвшөөрнө үү.</p>
    <ul>${pts.map((t) => `<li>${esc(t)}</li>`).join("")}</ul>
    <p class="cg-links"><a href="/terms" target="_blank" rel="noopener">Үйлчилгээний нөхцөл ↗</a><a href="/privacy" target="_blank" rel="noopener">Нууцлалын бодлого ↗</a></p>
    <p class="cg-err" role="alert"></p>
    <div class="cg-act"><button type="button" class="btn btn-ghost" data-cg-out>Гарах</button><button type="button" class="btn btn-gold" data-cg-ok>Зөвшөөрч байна</button></div>
  </div>`;
  const behind = [...document.body.children];
  behind.forEach((c) => (c.inert = true));
  document.body.append(el); document.documentElement.classList.add("cg-open");
  const ok = $("[data-cg-ok]", el); ok.focus();
  $("[data-cg-out]", el).onclick = () => { Auth.clear(); location.href = "/"; };
  ok.onclick = async () => {
    ok.disabled = true;
    try {
      Auth.set(Auth.token, await api("/api/me/consent", { method: "POST", body: { version: ver } }));
      behind.forEach((c) => (c.inert = false));
      el.remove(); document.documentElement.classList.remove("cg-open");
      toast("Баярлалаа! Зөвшөөрлийг тань хадгаллаа.");
      phoneGate(); // дараа нь утсаа баталгаажуулна
    } catch (x) {
      if (x.status === 409) return location.reload(); // хуудас хуучирсан — шинэ хувилбарыг ачаална
      $(".cg-err", el).textContent = x.message; ok.disabled = false;
    }
  };
  return true; // цонх гарсан — утасны цонхыг зөвшөөрсний дараа
}

function bell() {
  const links = $(".nav-links");
  if (!links || $("#bell")) return;
  const dots = `<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor" aria-hidden="true"><circle cx="5" cy="12" r="2"/><circle cx="12" cy="12" r="2"/><circle cx="19" cy="12" r="2"/></svg>`;
  // Facebook маягийн самбар: гарчиг + «⋯» цэс, «Бүгд / Уншаагүй» таб, «Шинэ / Өмнөх» хэсэг, аватар дээр төрлийн тэмдэг, цэнхэр цэг.
  links.insertAdjacentHTML("afterbegin", `
    <div class="bell-wrap"><button class="icon-btn bell" id="bell" aria-label="Мэдэгдэл" aria-haspopup="dialog" aria-expanded="false" aria-controls="bellPanel">${icon("bell", 21)}<span class="bell-badge" hidden></span></button>
    <section class="np" id="bellPanel" role="dialog" aria-labelledby="npTitle" hidden>
      <header class="np-head"><h2 id="npTitle">Мэдэгдэл</h2><button type="button" class="np-more" id="npMore" aria-label="Мэдэгдлийн тохиргоо" aria-haspopup="menu" aria-expanded="false">${dots}</button>
        <div class="np-menu" id="npMenu" role="menu" hidden><button type="button" role="menuitem" data-np="readall">${icon("check", 18)}<span>Бүгдийг уншсан болгох</span></button><button type="button" role="menuitem" data-np="perm" hidden>${icon("bell", 18)}<span>Браузерт мэдэгдэх</span></button></div></header>
      <div class="np-tabs" role="tablist"><button type="button" role="tab" class="on" aria-selected="true" data-np-tab="all">Бүгд</button><button type="button" role="tab" aria-selected="false" data-np-tab="unread">Уншаагүй</button></div>
      <div class="np-body" id="bellList"><div class="np-skel" aria-hidden="true"><i></i><i></i><i></i></div></div>
    </section></div>`);
  const btn = $("#bell"), badge = $(".bell-badge", btn), panel = $("#bellPanel"), list = $("#bellList"), menu = $("#npMenu"), more = $("#npMore");
  // «Харсан» хил: самбарыг сүүлд нээх үеийн хамгийн шинэ мэдэгдлийн ID (энэ төхөөрөмж дээр). Тэмдэг (badge) зөвхөн
  // түүнээс хойших уншаагүйг тоолно; цэнхэр цэг нь тухайн мэдэгдэл дээр дартал (эсвэл «Бүгдийг уншсан») үлдэнэ.
  const SEEN = "sg_notif_seen";
  const getSeen = () => { try { return localStorage.getItem(SEEN) || ""; } catch { return ""; } };
  const setSeen = (id) => { try { localStorage.setItem(SEEN, id); } catch {} };
  let items = [], tab = "all", limit = 12, floor = getSeen(), loaded = false, hasOlder = false, busy = false;
  // Мэдэгдэл бүрийн дүрс: гарчгийн эхний emoji-оос, эс бөгөөс төрлөөс; өнгө нь төрлөөр (чат — цэнхэр, мөнгө — ногоон г.м.).
  const N_EMOJI = { "💬": "chat", "↩": "chat", "👥": "users", "📅": "cal", "❌": "cal", "🏦": "money", "💰": "money", "✅": "check", "⚠️": "alert", "⚠": "alert", "🚩": "alert", "⛔": "lock", "🔓": "lock", "🔒": "lock", "🔑": "lock", "📹": "live", "📣": "bell", "📞": "phone", "📎": "clip", "💾": "files", "🎓": "courses", "📚": "book", "👀": "eye", "👁": "eye", "📝": "exam", "🏆": "medal", "🎖": "medal", "🛒": "card", "💳": "card" };
  const N_TYPE = { message: "chat", comment: "chat", reply: "chat", booking: "cal", payout: "money", purchase: "card", enroll: "courses", profile_view: "eye", course_view: "eye", lesson_view: "play", phone_view: "phone", storage: "files", report: "alert", violation: "alert", account: "shield", blocked: "lock", unblock: "lock", assignment: "clip", grade: "check", exam: "exam", remind: "bell", rank: "medal" };
  const TONE = { chat: "blue", users: "blue", money: "green", card: "green", check: "green", phone: "green", courses: "violet", medal: "violet", exam: "violet", book: "violet", cal: "orange", bell: "orange", alert: "red", lock: "red", live: "red", clip: "teal", files: "teal", play: "teal", eye: "gray", shield: "gray" };
  const kind = (n) => {
    const t = (n.title || "").trim(), m = /^(\p{Extended_Pictographic}️?|↩)\s*/u.exec(t);
    const name = (m && N_EMOJI[m[1]]) || N_TYPE[n.type] || "bell";
    return { name, tone: TONE[name] || "blue", title: m ? t.slice(m[0].length) : t };
  };
  const ago = (t) => {
    const m = (Date.now() - new Date(t)) / 60000;
    return m < 1 ? "Саяхан" : m < 60 ? Math.floor(m) + " мин" : m < 1440 ? Math.floor(m / 60) + " цаг" : m < 10080 ? Math.floor(m / 1440) + " өдөр" : m < 50400 ? Math.floor(m / 10080) + " 7 хоног" : fmtDay(t, new Date(t).getFullYear() !== new Date().getFullYear());
  };
  const item = (n) => {
    const k = kind(n), unread = !n.read;
    const ub = n.type === "blocked" && /^unblock:([^:]+):(.+)$/.exec(n.link || "");
    const av = n.actor ? `${avatarHTML(n.actor, "np-avatar")}<i class="np-badge t-${k.tone}">${icon(k.name, 13)}</i>` : `<span class="np-sys t-${k.tone}">${icon(k.name, 26)}</span>`;
    const acts = ub ? `<span class="np-acts"><button type="button" class="np-btn" data-unblock-user="${esc(ub[1])}" data-unblock-lesson="${esc(ub[2])}">Дахин нээх</button></span>` : "";
    const inner = `<span class="np-av">${av}</span><span class="np-txt"><span class="np-t">${esc(k.title)}</span>${n.body ? `<span class="np-b">${esc(n.body)}</span>` : ""}<time datetime="${esc(n.created_at)}" title="${esc(fmtDate(n.created_at))}">${ago(n.created_at)}</time>${acts}</span>${unread ? `<i class="np-dot" aria-label="Уншаагүй"></i>` : ""}`;
    const cls = "np-item" + (unread ? " unread" : "");
    return ub || !n.link ? `<div class="${cls}" data-nid="${esc(n.id)}">${inner}</div>` : `<a class="${cls}" href="${esc(n.link)}" data-nid="${esc(n.id)}">${inner}</a>`;
  };
  // «Шинэ» = самбарыг энэ удаа нээхээс өмнө хараагүй нь (анх энэ төхөөрөмж дээр бол — уншаагүй нь).
  const isNew = (n) => (floor ? n.id > floor : !n.read);
  const sec = (title, extra = "") => `<div class="np-sec"><h3>${title}</h3>${extra}</div>`;
  const view = () => (tab === "unread" ? items.filter((n) => !n.read) : items);
  const draw = () => {
    if (!loaded) return;
    const xs = view();
    if (!xs.length) {
      list.innerHTML = `<div class="np-empty"><span>${icon("bell", 30)}</span><b>${tab === "unread" ? "Уншаагүй мэдэгдэл алга" : "Одоогоор мэдэгдэл алга"}</b><small>${tab === "unread" ? "Бүгдийг нь уншсан байна." : "Шинэ зүйл болоход энд харагдана."}</small></div>`;
      return;
    }
    const vis = xs.slice(0, limit), fresh = vis.filter(isNew), old = vis.filter((n) => !isNew(n));
    const all = items.some((n) => !n.read) ? `<button type="button" class="np-link" data-np="readall">Бүгдийг уншсан</button>` : "";
    list.innerHTML = (fresh.length ? sec("Шинэ", all) + fresh.map(item).join("") : "") + (old.length ? sec("Өмнөх", fresh.length ? "" : all) + old.map(item).join("") : "") +
      (xs.length > limit || hasOlder ? `<button type="button" class="np-prev" data-np="prev"${busy ? " disabled" : ""}>${busy ? "Ачаалж байна…" : "Өмнөх мэдэгдлүүдийг харах"}</button>` : "");
  };
  const newest = () => items.reduce((a, n) => (n.id > a ? n.id : a), "");
  const setBadge = () => { const s = getSeen(), k = items.filter((n) => !n.read && n.id > s).length; badge.hidden = !k; badge.textContent = k > 99 ? "99+" : k; };
  const load = async () => {
    try {
      const d = await api("/api/me/notifications"), mine = new Set(items.filter((n) => n.read).map((n) => n.id));
      const page = (d.items || []).map((n) => (mine.has(n.id) ? { ...n, read: true } : n)); // саяхан дарсныг сервер хараахан бичээгүй байж болно
      const lo = page.length ? page[page.length - 1].id : "", tail = lo ? items.filter((n) => n.id < lo) : []; // «Өмнөх»-өөр ачаалсныг хадгална
      items = page.concat(tail); if (!tail.length) hasOlder = !!d.more;
      loaded = true; setBadge(); if (!panel.hidden) draw();
    } catch { if (!loaded && !panel.hidden) list.innerHTML = `<div class="np-empty"><b>Ачаалж чадсангүй</b><small>Дахин оролдоно уу.</small></div>`; }
  };
  // Ачаалсан мэдэгдэл дуусвал хамгийн хуучны өмнөхийг серверээс дараагийн хуудсаар татна.
  const older = async () => {
    if (busy || !hasOlder || !items.length) return;
    busy = true; draw();
    try {
      const d = await api("/api/me/notifications?before=" + encodeURIComponent(items[items.length - 1].id)), have = new Set(items.map((n) => n.id));
      items = items.concat((d.items || []).filter((n) => !have.has(n.id))); hasOlder = !!d.more;
    } catch (x) { toast(x.message, true); } finally { busy = false; }
  };
  // Дээр нь дарсан мэдэгдлийг уншсан болгоно (keepalive — хуудас шилжсэн ч хүсэлт хүрнэ).
  const markOne = (id) => fetch("/api/me/notifications/read", { method: "POST", keepalive: true, headers: { "Content-Type": "application/json", Authorization: "Bearer " + Auth.token }, body: JSON.stringify({ id }) }).catch(() => {});
  const closeMenu = () => { menu.hidden = true; more.setAttribute("aria-expanded", "false"); };
  const open = (show) => {
    panel.hidden = !show; btn.setAttribute("aria-expanded", String(show)); closeMenu();
    if (!show) return;
    floor = getSeen(); limit = 12; draw();
    const top = newest(); if (top) setSeen(top);
    setBadge(); load();
  };
  load();
  btn.addEventListener("click", (e) => { e.stopPropagation(); open(panel.hidden); });
  // composedPath: дарсан товч draw()-аар DOM-оос салсан ч (жишээ нь «Өмнөх…») самбар дотор дарсанд тооцогдоно.
  const wrap = btn.closest(".bell-wrap");
  document.addEventListener("click", (e) => { if (!panel.hidden && !e.composedPath().includes(wrap)) open(false); });
  addEventListener("keydown", (e) => { if (e.key === "Escape" && !panel.hidden) { if (!menu.hidden) return closeMenu(); open(false); btn.focus(); } });
  more.addEventListener("click", (e) => {
    e.stopPropagation();
    const show = menu.hidden; menu.hidden = !show; more.setAttribute("aria-expanded", String(show));
    $("[data-np=perm]", menu).hidden = !("Notification" in window && Notification.permission === "default");
    $("[data-np=readall]", menu).disabled = !items.some((n) => !n.read);
  });
  panel.addEventListener("click", async (e) => {
    if (!e.target.closest("#npMenu, #npMore")) closeMenu();
    const tb = e.target.closest("[data-np-tab]");
    if (tb) {
      tab = tb.dataset.npTab; limit = 12;
      $$("[data-np-tab]", panel).forEach((b) => { const on = b === tb; b.classList.toggle("on", on); b.setAttribute("aria-selected", String(on)); });
      return draw();
    }
    const act = e.target.closest("[data-np]")?.dataset.np;
    if (act === "prev") {
      limit += 15;
      for (let i = 0; i < 3 && hasOlder && view().length < limit; i++) { const n = items.length; await older(); if (items.length === n) break; } // уншаагүй табад хуучин хуудсанд цөөн байж болно
      return draw();
    }
    if (act === "readall") {
      closeMenu(); items = items.map((n) => ({ ...n, read: true })); setBadge(); draw();
      api("/api/me/notifications/read", { method: "POST" }).catch(() => {});
      return toast("✓ Бүх мэдэгдлийг уншсан болголоо");
    }
    if (act === "perm") { closeMenu(); await Notification.requestPermission(); return; }
    const it = e.target.closest(".np-item"); if (!it) return;
    const n = items.find((x) => x.id === it.dataset.nid);
    if (n && !n.read) { n.read = true; markOne(n.id); it.classList.remove("unread"); it.querySelector(".np-dot")?.remove(); setBadge(); }
    const b = e.target.closest("[data-unblock-user]");
    if (b) {
      e.preventDefault(); b.disabled = true;
      try {
        const r = await api(`/api/me/students/${b.dataset.unblockUser}/unblock`, { method: "POST", body: { lesson_id: b.dataset.unblockLesson } });
        b.classList.add("done"); b.textContent = r.exam ? "✓ Шалгалт дахин нээгдлээ (+1 оролдлого)" : r.unblocked ? "✓ Дахин нээгдлээ — суралцагчид мэдэгдлээ" : "✓ Аль хэдийн нээлттэй";
      } catch (x) { toast(x.message, true); b.disabled = false; }
      return;
    }
    if (it.tagName === "A") open(false);
  });
  let refetch = 0;
  Live.on((d) => {
    if (d.type !== "notification") return;
    const n = { ...d.notification, read: false };
    if (items.some((x) => x.id === n.id)) return;
    items.unshift(n); loaded = true;
    if (!panel.hidden) { setSeen(n.id); draw(); } // нээлттэй үед тэмдэг нэмэгдэхгүй — шууд «Шинэ» хэсэгт
    setBadge();
    if (n.actor_id) { clearTimeout(refetch); refetch = setTimeout(load, 700); } // үүсгэсэн хүний зураг, нэрийг серверээс
    btn.animate([{ transform: "rotate(0)" }, { transform: "rotate(18deg)" }, { transform: "rotate(-14deg)" }, { transform: "rotate(0)" }], { duration: 600 });
    toast(n.title);
    if (document.hidden && "Notification" in window && Notification.permission === "granted") {
      const bn = new Notification(n.title, { body: n.body || "", tag: n.id });
      bn.onclick = () => { focus(); if (n.link && !n.link.startsWith("unblock:")) { markOne(n.id); location.href = n.link; } else open(true); };
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
// Embed-ийн HTML-ийг бүтэн баримт болгоно (хоосон зай, хэмжээ тохируулна). Цонх нь багшийн тавьсан өндөртэй
// (px) тогтмол — агуулга урт бол дотроо гүйлгэнэ (өөрөө сунаж томрохгүй). Өндөр (height) заасан дотоод
// iframe (Google Forms гэх мэт) тэр өндрөөрөө; өндөргүй ганц iframe цонхыг бүтэн дүүргэнэ.
// HTML embed доторх маркер: эх хуудасны lessonMarker-тэй postMessage-ээр ажиллана (sandbox тул DOM-д хүрэхгүй).
// эх → embed: {sgHL:{list, marking, quiz}}; embed → эх: {sgHLready}, {sgHLsel:{s,e,t}}, {sgHLclick:{id,x,y}}.
const EMBED_HL_JS = `(function(){var list=[],marking=false,B=document.body;
var st=document.createElement("style");st.textContent="mark.hl{color:inherit;border-radius:3px;padding:0 1px;cursor:pointer;-webkit-box-decoration-break:clone;box-decoration-break:clone}.hl-y{background:rgba(255,214,0,.45)}.hl-g{background:rgba(52,199,89,.35)}.hl-r{background:rgba(255,69,58,.3)}.hl-b{background:rgba(10,132,255,.3)}mark.has-note{border-bottom:2px dotted currentColor}.hlq mark.hl:not(.shown){color:transparent;background:repeating-linear-gradient(135deg,#cbd2e1 0 6px,#e1e6f0 6px 12px)}.hlq mark.hl:not(.shown) *{visibility:hidden}.hlm,.hlm *{user-select:text;-webkit-user-select:text;cursor:text}.hlm ::selection{background:rgba(255,214,0,.6)}";document.head.appendChild(st);
function tn(){var w=document.createTreeWalker(B,NodeFilter.SHOW_TEXT,{acceptNode:function(n){return n.parentNode.closest("script,style")?2:1}}),o=[];while(w.nextNode())o.push(w.currentNode);return o}
function txt(){return tn().map(function(n){return n.data}).join("")}
function off(node,o){var r=document.createRange();r.setStart(B,0);r.setEnd(node,o);var n=0,a=tn();for(var i=0;i<a.length;i++){var t=a[i];if(t===node)return n+o;if(r.comparePoint(t,t.length)===0)n+=t.length;else break}return n}
function unwrap(){[].slice.call(document.querySelectorAll("mark.hl")).forEach(function(m){var p=m.parentNode;while(m.firstChild)p.insertBefore(m.firstChild,m);m.remove();p.normalize()})}
function wrap(h){var pos=0;tn().forEach(function(t){var a=pos,b=pos+t.data.length;pos=b;if(b<=h.s||a>=h.e)return;var node=t,s=Math.max(h.s,a)-a,e=Math.min(h.e,b)-a;if(e<node.data.length)node.splitText(e);if(s>0)node=node.splitText(s);var m=document.createElement("mark");m.className="hl hl-"+h.c+(h.n?" has-note":"");m.dataset.hid=h.id;if(h.n)m.title=h.n;node.parentNode.insertBefore(m,node);m.appendChild(node)})}
function paint(){unwrap();var full=txt();list.forEach(function(h){if(full.slice(h.s,h.e)!==h.t){var i=full.indexOf(h.t);if(i<0)return;h.s=i;h.e=i+h.t.length}wrap(h)})}
function isW(c){return!!c&&/[\\p{L}\\p{N}]/u.test(c)}
function sel(){var g=getSelection();if(!marking||!g.rangeCount||g.isCollapsed)return;var r=g.getRangeAt(0),full=txt(),s=off(r.startContainer,r.startOffset),e=off(r.endContainer,r.endOffset);while(s<e&&/\\s/.test(full[s]))s++;while(e>s&&/\\s/.test(full[e-1]))e--;if(e>s){var lo=0,hi=full.length,pos=0,a=tn();for(var i=0;i<a.length;i++){var x=pos,y=pos+a[i].data.length;pos=y;if(x<=s&&s<y)lo=x;if(x<e&&e<=y){hi=y;break}}while(s>lo&&isW(full[s-1])&&isW(full[s]))s--;while(e<hi&&isW(full[e])&&isW(full[e-1]))e++}e=Math.min(e,s+5000);g.removeAllRanges();if(e-s>=1)parent.postMessage({sgHLsel:{s:s,e:e,t:full.slice(s,e)}},"*")}
addEventListener("message",function(ev){var d=ev.data&&ev.data.sgHL;if(!d||ev.source!==parent)return;list=d.list||[];marking=!!d.marking;document.documentElement.classList.toggle("hlq",!!d.quiz);document.documentElement.classList.toggle("hlm",marking);paint()});
document.addEventListener("pointerup",function(){if(marking)setTimeout(sel,0)});
document.addEventListener("click",function(ev){var ln=ev.target.closest&&ev.target.closest("a[href]");if(ln&&/^https?:/i.test(ln.href)){ev.preventDefault();parent.postMessage({sgLink:ln.href},"*");return}var m=ev.target.closest&&ev.target.closest("mark.hl");if(!m)return;if(document.documentElement.classList.contains("hlq")){m.classList.toggle("shown");return}if(!getSelection().isCollapsed)return;var rc=m.getBoundingClientRect();parent.postMessage({sgHLclick:{id:m.dataset.hid,x:rc.left,y:rc.bottom}},"*")});
parent.postMessage({sgHLready:1},"*")})()`;
const embedDoc = (html) => `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><style>html,body{margin:0;padding:0;height:100%;font-family:system-ui,sans-serif}html{overflow:auto}body{display:flow-root}iframe,video,img{max-width:100%}iframe[height]{width:100%;border:0;display:block}body>iframe:only-child:not([height]){width:100%;height:100%;border:0;display:block}</style></head><body>${html}<script>${EMBED_HL_JS}</script></body></html>`;
// Embed-д зөвхөн холбоос (https://…) буулгасан бол: хичээл дотор шууд ажиллах хаяг. Түгээмэл үйлчилгээг
// embed хэлбэрт нь хувиргана. Өөрийн сайтын хаягийг зөвшөөрөхгүй (sandbox-оос гарахаас сэргийлнэ).
function embedLink(text) {
  const t = (text || "").trim();
  if (!/^https:\/\/[^\s<>"']+$/i.test(t)) return null;
  let u; try { u = new URL(t); } catch { return null; }
  if (u.host === location.host || u.hostname.endsWith("." + location.hostname)) return null;
  const host = u.hostname.replace(/^www\./, ""), p = u.pathname;
  if (/(^|\.)youtube\.com$|^youtu\.be$|(^|\.)vimeo\.com$/.test(host)) return { video: true, src: t, url: t };
  let src = t, m;
  if (host === "docs.google.com" && (m = p.match(/^\/forms\/d\/(e\/)?([\w-]+)/))) src = `https://docs.google.com/forms/d/${m[1] || ""}${m[2]}/viewform?embedded=true`;
  else if (host === "docs.google.com" && (m = p.match(/^\/(document|spreadsheets|presentation)\/d\/([\w-]+)/))) src = `https://docs.google.com/${m[1]}/d/${m[2]}/${m[1] === "presentation" ? "embed" : "preview"}`;
  else if (host === "drive.google.com" && (m = p.match(/^\/file\/d\/([\w-]+)/))) src = `https://drive.google.com/file/d/${m[1]}/preview`;
  else if (host === "canva.com" && /^\/design\/[\w-]+/.test(p) && !u.searchParams.has("embed")) { u.search = u.search ? u.search + "&embed" : "?embed"; src = u.toString(); }
  return { src, url: t };
}
function blockHTML(b) {
  const cap = b.text && b.type !== "text" && b.type !== "heading" ? `<figcaption>${esc(b.text)}</figcaption>` : "";
  switch (b.type) {
    case "heading": return `<h3 class="rb-h">${esc(b.text)}</h3>`;
    // HTML embed: sandbox (allow-same-origin-гүй) — скрипт ажиллана, гэхдээ сайтын нэвтрэлт, өгөгдөлд хүрэхгүй.
    case "embed": {
      // Зөвхөн холбоос буулгасан бол: тухайн хуудсыг хичээл дотор шууд ажиллуулна (YouTube/Vimeo — видео тоглуулагчаар).
      // Хичээлээс гадагш гаргахгүй: шинэ таб/цонх нээх (popup), эх хуудсыг шилжүүлэх эрхгүй.
      const lk = embedLink(b.text), h = Math.max(80, Math.min(2000, +b.height || 420));
      if (lk?.video) return `<figure class="rb-video"><div class="player">${mediaHTML(lk.src)}</div></figure>`;
      if (lk) return `<figure class="rb-embed rb-embed-link"><iframe src="${esc(lk.src)}" sandbox="allow-scripts allow-same-origin allow-forms allow-presentation" allow="fullscreen; autoplay; encrypted-media; picture-in-picture; clipboard-write" allowfullscreen loading="lazy" referrerpolicy="strict-origin-when-cross-origin" style="height:${h}px" title="Embed"></iframe></figure>`;
      return `<figure class="rb-embed" data-bid="${esc(b.id || "")}"><iframe sandbox="allow-scripts allow-forms allow-presentation" allow="fullscreen; autoplay; encrypted-media; picture-in-picture" allowfullscreen loading="lazy" referrerpolicy="no-referrer" style="height:${h}px" srcdoc="${esc(embedDoc(b.text))}" title="Embed"></iframe></figure>`;
    }
    case "text": return `<div class="rb-text" data-bid="${esc(b.id || "")}">${b.html || richHTML(b.text)}</div>`; // html: сервер цэвэрлэсэн Word шиг агуулга
    case "image": return `<figure class="rb-fig"><img src="${esc(b.url)}" alt="${esc(b.text || "")}" loading="lazy">${cap}</figure>`;
    case "audio": return `<figure class="rb-audio" data-bid="${esc(b.id || "")}"><span class="rb-audio-ico">${icon("headphones", 22)}</span><div><b>${esc(b.name || "Дуу бичлэг")}</b><audio src="${esc(b.url)}" controls preload="metadata" controlslist="nodownload" oncontextmenu="return false"></audio></div>${cap}</figure>`;
    case "video": return `<figure class="rb-video" data-bid="${esc(b.id)}" ${b.parts?.length > 1 ? `data-parts="${esc(JSON.stringify(b.parts))}"` : ""}><div class="player">${mediaHTML(b.url, b.name)}</div>${b.parts?.length > 1 ? `<div class="vparts" aria-label="Видеоны хэсгүүд">${b.parts.map((_, i) => `<button type="button" data-vpart="${i}" class="${i ? "" : "on"}">${i + 1}-р хэсэг</button>`).join("")}<span class="muted small">6 минутын ${b.parts.length} хэсэг · дараалан тоглоно</span></div>` : ""}${cap}</figure>`;
    case "file": {
      const meta = esc([extOf(b.url).slice(1).toUpperCase(), fmtBytes(b.size)].filter(Boolean).join(" · ")) + (b.text ? " · " + esc(b.text) : "");
      const kind = docKind(b.url);
      // PDF, PowerPoint, Word, Excel: хичээл дотор шууд (попапгүй), татах боломжгүй — багш зөвшөөрсөн бол л татна.
      if (kind && kind !== "legacy") return `<figure class="rb-doc" data-bid="${esc(b.id || "")}" ${b.download ? "data-dl" : "data-nodl"}>${docViewHTML(b.url, b.name || DOC_LABEL[kind])}${cap}</figure>`;
      return b.download ? `<a class="rb-file" href="${esc(b.url)}" target="_blank" rel="noopener" download><span class="rb-file-ico">${icon("doc", 22)}</span><span><b>${esc(b.name || "Файл")}</b><small>${meta}</small></span><span class="btn btn-sm btn-glass">⬇ Татах</span></a>`
        : `<div class="rb-file"><span class="rb-file-ico">🔒</span><span><b>${esc(b.name || "Файл")}</b><small>${meta} · ${kind === "legacy" ? "Хуучин форматыг шууд үзүүлэх боломжгүй — багш PDF, .pptx, .docx болгож оруулна" : "Зөвхөн хичээл дотор үзнэ"}</small></span></div>`;
    }
    case "cloud": { // үүлэн үг: суралцагч 1-3 үг илгээж, бүгдийн хариулт үүл болно (mountClouds)
      const n = Math.max(1, Math.min(3, +b.max || 1));
      return `<figure class="rb-cloud" data-bid="${esc(b.id || "")}"><figcaption class="cl-q">${icon("cloud", 20)}<span>${esc(b.text)}</span></figcaption>
        <form class="cl-form">${Array.from({ length: n }, (_, i) => `<input class="cl-in" maxlength="30" autocomplete="off" placeholder="${n > 1 ? `${i + 1}-р үг` : "Нэг үгээр бичнэ үү"}" aria-label="Үг ${i + 1}">`).join("")}<button class="btn btn-gold btn-sm" type="submit">Илгээх</button></form>
        <div class="cl-cloud" aria-live="polite"><p class="muted small">Ачаалж байна…</p></div><p class="cl-meta muted small"></p></figure>`;
    }
    case "board": { // цагаан самбар: «Самбар нээх» дарахад board.js (Fabric) ачаалагдана (mountBoards)
      const t = b.text || "Цагаан самбар";
      return `<figure class="rb-board" data-bid="${esc(b.id || "")}" data-title="${esc(t)}"><div class="rbb-art" aria-hidden="true">${BOARD_ART}</div>
        <figcaption class="rbb-body"><span class="rbb-kicker">${icon("board", 15)}Цагаан самбар · багаар ажиллах</span><b class="rbb-title">${esc(t)}</b>
        <span class="rbb-feats"><i>✏️ Зурах, бичих</i><i>🖼️ Зураг, дүрс, эможи</i><i>👥 Багаар</i><i>⚡ Шууд харагдана</i></span>
        <button type="button" class="btn btn-gold btn-sm" data-board-open>${icon("expand", 16)}Самбар нээх</button></figcaption></figure>`;
    }
    case "quiz": return quizFormHTML(b, false);
  }
  return "";
}
const BOARD_ART = `<svg viewBox="0 0 150 112" width="150" height="112"><rect x="4" y="5" width="142" height="92" rx="13" fill="#fff" stroke="#cbd2e1" stroke-width="2"/>
  <path d="M18 72c10-18 22-28 34-10s20 8 30-12" fill="none" stroke="#1f3c8f" stroke-width="4" stroke-linecap="round"/>
  <g transform="rotate(6 115 35)"><rect x="96" y="17" width="38" height="35" rx="3" fill="#fff3a3"/><path d="M103 30h24M103 38h15" stroke="#b08900" stroke-width="3" stroke-linecap="round"/></g>
  <path d="m31 19 4.6 9.3 10.2 1.5-7.4 7.2 1.8 10.2-9.2-4.8-9.2 4.8 1.8-10.2-7.4-7.2 10.2-1.5z" fill="#eaa02e"/>
  <circle cx="113" cy="76" r="12" fill="none" stroke="#e11d48" stroke-width="3.5"/>
  <path d="M56 85h26m-6-6 7 6-7 6" fill="none" stroke="#16a34a" stroke-width="4" stroke-linecap="round" stroke-linejoin="round"/>
  <path d="M58 98v9M92 98v9" stroke="#cbd2e1" stroke-width="4" stroke-linecap="round"/></svg>`;
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
/* ---------- Цагаан самбар: Fabric ба board.js-ийг зөвхөн «Самбар нээх» үед ачаална ---------- */
function loadCSS(href) {
  if (document.querySelector(`link[href="${href}"]`)) return Promise.resolve();
  return new Promise((res, rej) => { const l = document.createElement("link"); l.rel = "stylesheet"; l.href = href; l.onload = res; l.onerror = () => { l.remove(); rej(new Error("Самбарын загвар ачаалагдсангүй — интернэтээ шалгана уу")); }; document.head.append(l); });
}
async function openBoard(opts) {
  const v = document.querySelector('script[src*="/static/app.js"]')?.src.split("v=")[1] || "";
  await Promise.all([loadCSS("/static/board.css?v=" + v), loadScript("/static/vendor/fabric-5.5.2.min.js")]);
  await loadScript("/static/board.js?v=" + v);
  window.SGBoard.open(opts);
}
// owner — тухайн хичээлийн багш (горим, багуудыг удирдана).
function mountBoards(root, base, owner = false) {
  $$(".rb-board[data-bid]:not([data-m])", root).forEach((fig) => {
    fig.dataset.m = 1;
    const btn = $("[data-board-open]", fig);
    btn.onclick = async () => {
      if (!base || !Auth.token) { toast("Самбарт нэгдэхийн тулд нэвтэрнэ үү", true); return; }
      const html = btn.innerHTML; btn.disabled = true; btn.textContent = "Нээж байна…";
      try { await openBoard({ base: `${base}/board/${encodeURIComponent(fig.dataset.bid)}`, lid: base.split("/lessons/")[1]?.split("/")[0] || "", bid: fig.dataset.bid, title: fig.dataset.title, teacher: owner }); }
      catch (e) { toast(e.message, true); }
      finally { btn.disabled = false; btn.innerHTML = html; }
    };
  });
}
/* ---------- Үүлэн үг: суралцагчдын хариулт үүл болж, шинэ хариулт WebSocket-оор шууд ирнэ ---------- */
const CLOUD_HUES = ["#2f56d6", "#d9622b", "#1f9a4a", "#b45309", "#7c3aed", "#0e7490", "#be185d", "#1f3c8f"];
// viewOnly — багш (өөрийн хичээл): маягтгүй, зөвхөн үр дүн.
function mountClouds(root, base, viewOnly = false) {
  $$(".rb-cloud[data-bid]:not([data-m])", root).forEach((fig) => {
    fig.dataset.m = 1;
    const url = base && `${base}/cloud/${encodeURIComponent(fig.dataset.bid)}`;
    const box = $(".cl-cloud", fig), meta = $(".cl-meta", fig), form = $(".cl-form", fig);
    // Багшийн засварлагч (өөр <form> дотор) — дотоод form-ыг хөтөч хасдаг тул оролтуудыг шууд нууна.
    const hideForm = () => $$(".cl-form, .cl-in, button[type=submit]", fig).forEach((x) => (x.hidden = true));
    if (!url) { hideForm(); box.innerHTML = `<p class="muted small">Нэвтэрч орвол хариулж, бусдын үгийг харна.</p>`; return; }
    if (viewOnly || !form) hideForm();
    let last = "", lastMine = [];
    const render = (d) => {
      if (d.mine) lastMine = d.mine; else d = { ...d, mine: lastMine }; // түлхэлтэд «миний үг» ирэхгүй
      const key = JSON.stringify([d.words, d.mine]); if (key === last) return; last = key;
      const ws = d.words || [], top = ws[0]?.n || 1, mine = new Set(d.mine || []);
      const arr = []; ws.forEach((w, i) => (i % 2 ? arr.push(w) : arr.unshift(w))); // том үгс голд
      box.innerHTML = ws.length ? arr.map((w) => {
        const k = Math.sqrt(w.n / top), hue = CLOUD_HUES[[...w.w].reduce((a, ch) => a + ch.charCodeAt(0), 0) % CLOUD_HUES.length];
        return `<span class="cl-w${mine.has(w.w) ? " mine" : ""}" style="font-size:${(15 + k * 37).toFixed(1)}px;color:${hue}" title="${w.n} хүн">${esc(w.w)}</span>`;
      }).join("") : `<p class="muted small">${viewOnly ? "Хараахан хариулт алга." : "Хараахан хариулт алга — эхний үгийг та бичээрэй."}</p>`;
      meta.textContent = d.people ? `${d.people} хүн хариулсан` : "";
      if (d.mine?.length && !viewOnly && form) {
        $$(".cl-in", form).forEach((inp, i) => { if (document.activeElement !== inp) inp.value = d.mine[i] || ""; });
        $("button", form).textContent = "Шинэчлэх";
      }
    };
    const load = () => api(url).then(render).catch(() => {});
    form?.addEventListener("submit", async (e) => {
      e.preventDefault();
      const words = $$(".cl-in", form).map((i) => i.value.trim()).filter(Boolean);
      if (!words.length) { $(".cl-in", form).focus(); return; }
      const b = $("button", form); b.disabled = true;
      try { render(await api(url, { method: "POST", body: { words } })); toast("☁ Үг тань үүлэнд нэмэгдлээ"); }
      catch (err) { toast(err.message, true); } finally { b.disabled = false; }
    });
    load();
    // Шинэ хариулт WebSocket-оор (блок бүрт 4 секундэд нэг удаа хүртэл) түлхэгдэнэ; холболт тасарсан үед 30 секунд тутам асууна.
    const [, cid, lid] = url.match(/\/courses\/([^/]+)\/lessons\/([^/]+)\//) || [], bid = fig.dataset.bid, topic = `cloud:${lid}|${bid}`;
    const onLive = (d) => {
      if (!fig.isConnected) { Live.unsub(topic); Live.off(onLive); return; }
      if (d.type === "cloud" && d.lesson === lid && d.block === bid) render(d);
      else if (d.type === "reconnected") load(); // тасарсан хугацааны хариултууд
    };
    if (cid && lid) { Live.sub(topic, { t: "sub", topic: "cloud", course: cid, lesson: lid, block: bid }); Live.on(onLive); }
    const t = setInterval(() => {
      if (!fig.isConnected) { clearInterval(t); Live.unsub(topic); Live.off(onLive); return; }
      if (!document.hidden) load();
    }, 30000);
  });
}

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
async function discussionPanel(box, { courseId, lessonId, preload }) {
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
  const load = async (pre) => { try { d = pre || await api(`${base}/discussion`); render(); } catch (e) { box.innerHTML = e.status === 409 ? "" : `<p class="form-error">${esc(e.message)}</p>`; if (e.status === 409) box.hidden = true; } };
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
  await load(preload); // хичээлийн хариунд ирсэн бол дахин асуухгүй
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

/* ---------- Маркер: суралцагч хичээлийн текстийг өнгөөр тодруулж, тэмдэглэл бичиж, давтана ----------
   Тодруулга зөвхөн тухайн суралцагчид (сервер дээр өөрийнх нь нэр дээр) хадгалагдана. Маркер асаалттай үед
   л текст сонгогдоно; хуулах, тайрах, баруун товч хамгаалалттай хэвээр. */
const HL_COLORS = [["y", "Чухал"], ["g", "Ойлгосон"], ["r", "Ойлгоогүй"], ["b", "Нэр томьёо"]];
// ctx.teacher — багшийн нэр (хоосон бол «Багшаас асуух» харагдахгүй), ctx.title — хичээлийн нэр,
// ctx.self — багш өөрийн хичээлээ үзэж байна: «Багшаас асуух» урьдчилан харах байдлаар (илгээхгүй, тайлбар өгнө).
function lessonMarker(body, base, ctx = {}) {
  const card = body.parentElement;
  $$(".hl-bar, .hl-pop, .mn", card).forEach((x) => x.remove());
  body.classList.remove("hl-on", "hl-quiz");
  const units = () => (body.classList.contains("pre") ? [body] : $$(".rb-text[data-bid]", body));
  const bidOf = (u) => (u === body ? "content" : u.dataset.bid);
  // HTML embed-үүд (sandbox iframe): тодруулгыг postMessage-ээр дамжуулна (EMBED_HL_JS).
  const frames = () => $$(".rb-embed[data-bid] > iframe[srcdoc]", body);
  const frameBid = (f) => f.parentElement.dataset.bid;
  // Видео/аудио (мөчөөр), PDF, PowerPoint, Word (хуудас, слайдаар) дээрх тэмдэглэл.
  const medias = () => $$(".rb-video[data-bid], .rb-audio[data-bid], .rb-doc[data-bid]", body);
  const mediaOf = (b) => medias().find((f) => f.dataset.bid === b);
  if (!units().length && !frames().length && !medias().length) return () => {};
  let hs = [], color = "y", saveT = 0, dirty = false, pop = null;
  const bar = document.createElement("div"); bar.className = "hl-bar";
  bar.innerHTML = `<button type="button" class="hl-btn hl-toggle" aria-pressed="false" title="Асаагаад тодруулах текстээ чирж сонгоно" aria-label="Маркер">🖍<span class="hl-lbl"> Маркер</span></button>
    <button type="button" class="hl-btn hl-a11y" data-a11y-open aria-expanded="false" aria-controls="a11yPanel" title="Хүртээмж — харааны бэрхшээлтэй хүмүүст: фонт, контраст, зай, zoom" aria-label="Хүртээмжийн тохиргоо">${A11Y_SVG('<circle cx="12" cy="12" r="10"/><circle cx="12" cy="6.8" r="1.5" fill="currentColor" stroke="none"/><path d="M6.8 9.6 12 10.6l5.2-1M12 10.6v4.2M12 14.8l-2.6 4.4M12 14.8l2.6 4.4"/>', 18)}<span class="hl-lbl"> Хүртээмж</span></button>
    <span class="hl-colors" role="radiogroup" aria-label="Маркерын өнгө" hidden>${HL_COLORS.map(([c, t], i) => { const ask = c === "r" && ctx.teacher, tt = ask ? `${t} — тодруулаад багшаасаа асууж эсвэл тэмдэглэж үлдээнэ` : t; return `<button type="button" role="radio" class="hl-sw" data-c="${c}" aria-checked="${i === 0}" title="${tt}" aria-label="${tt}"><i class="hl-dot hl-${c}"></i><span class="hl-lbl">${t}${ask ? " 💬" : ""}</span></button>`; }).join("")}</span>
    <span class="grow"></span>
    <button type="button" class="hl-btn hl-quiz-btn" aria-pressed="false" title="Тодруулсан хэсгүүдийг нууна — санаж байгаад дарж шалгана" aria-label="Давтах" disabled>🧠<span class="hl-lbl"> Давтах</span></button>
    <button type="button" class="hl-btn hl-list-btn" aria-expanded="false" title="Миний тэмдэглэл" aria-label="Миний тэмдэглэл" disabled>📝<span class="hl-lbl"> Миний тэмдэглэл</span> <b>0</b></button>
    <div class="hl-list" hidden></div><div class="hl-ask" hidden></div>`;
  body.before(bar);
  if (!units().length && !frames().length) { $(".hl-toggle", bar).hidden = true; $(".hl-quiz-btn", bar).hidden = true; } // зөвхөн видео/баримт: тэмдэглэл л
  const listBox = $(".hl-list", bar), askBox = $(".hl-ask", bar);
  // «Ойлгоогүй» (улаан) тодруулга: тодруулмагц цонх гарна — багшаас асуух (ишлэлтэйгээр багштай чат руу шууд)
  // эсвэл зөвхөн тэмдэглээд үлдээх. Бичсэн зүйл тодруулгын тэмдэглэл болно (асуусан бол «❓ …»).
  let marking = false, askIds = [];
  const clip = (s, n) => (s.length > n ? s.slice(0, n) + "…" : s);
  const askHs = () => hs.filter((h) => askIds.includes(h.id));
  const closeAsk = () => { askBox.hidden = true; askBox.innerHTML = ""; askIds = []; };
  const openAsk = (ids) => {
    askIds = ids; const items = askHs(); if (!items.length) return;
    closePop(); listBox.hidden = true; askBox.hidden = false;
    const note = items[0].n && !items[0].n.startsWith("❓ ") ? items[0].n : "";
    askBox.innerHTML = `<b><i class="hl-dot hl-r"></i> Ойлгоогүй хэсэг</b><blockquote>${esc(clip(items.map((h) => h.t).join(" … "), 300))}</blockquote>
      <textarea rows="2" maxlength="1000" placeholder="${ctx.teacher ? "Юуг ойлгосонгүй вэ? Багшаасаа асуух асуулт эсвэл өөртөө тэмдэглэл…" : "Өөртөө тэмдэглэл…"}">${esc(note)}</textarea>
      ${ctx.teacher ? `<p class="muted small hl-ask-hint">${ctx.self ? "Урьдчилан харах: суралцагч «Багшаас асуух» дарвал асуулт нь ишлэлтэйгээ танд чатаар шууд ирнэ." : "«Багшаас асуух» дарвал асуулт тань ишлэлтэйгээ багшийн чат руу шууд очно."}</p>` : ""}
      <div class="hl-pop-acts"><button type="button" class="btn btn-ghost btn-sm" data-ask-keep>📌 Зөвхөн тэмдэглэх</button>${ctx.teacher ? `<button type="button" class="btn btn-gold btn-sm" data-ask-send>💬 Багшаас асуух</button>` : ""}</div>`;
    $("textarea", askBox).focus();
  };
  const keepAsk = () => {
    const v = $("textarea", askBox)?.value.trim();
    for (const h of askHs()) if (v) h.n = v;
    paint(); save(); closeAsk(); toast("📌 Тэмдэглэгдлээ — «Миний тэмдэглэл»-ээс харна");
  };
  const sendAsk = async () => {
    const ta = $("textarea", askBox), q = ta?.value.trim(); if (!q) { ta?.focus(); toast("Багшаасаа асуух асуултаа бичнэ үү"); return; }
    if (ctx.self) { keepAsk(); toast("Энэ таны өөрийн хичээл — суралцагч асуувал асуулт нь ишлэлтэйгээ танд чатаар ирнэ"); return; }
    const btn = $("[data-ask-send]", askBox); btn.disabled = true;
    const items = askHs(), quote = items.map((h) => h.t).join(" … ");
    const msg = `❓ «${ctx.title || "Хичээл"}» хичээлээс асуулт${quote ? `\n> ${clip(quote.replace(/\s+/g, " "), 300)}` : ""}\n\n${q}`;
    try {
      const d = await api(`/api/teachers/${encodeURIComponent(ctx.teacher)}/chat`, { method: "POST" });
      await api(`/api/chat/${d.conversation.id}/messages`, { method: "POST", body: { body: msg } });
      for (const h of items) { h.c = "r"; h.n = clip("❓ " + q, 990); }
      paint(); save(); askIds = [];
      askBox.innerHTML = `<b>✓ Багшид илгээлээ</b><p class="muted small" style="margin:6px 0 0">Хариуг чатаас харна. Хичээлээ үргэлжлүүлээрэй.</p>
        <div class="hl-pop-acts"><button type="button" class="btn btn-ghost btn-sm" data-ask-x>Хаах</button><button type="button" class="btn btn-glass btn-sm" data-ask-open="${esc(d.conversation.id)}">Чат нээх</button></div>`;
    } catch (e) { btn.disabled = false; toast(e.message, true); }
  };
  askBox.addEventListener("keydown", (e) => { if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) { e.preventDefault(); ctx.teacher ? sendAsk() : keepAsk(); } });
  // Блокийн текст доторх байрлал (тэмдэгтээр) — тэмдэглэгээ (mark) байсан ч ижил.
  const offsetIn = (u, node, off) => { const r = document.createRange(); r.setStart(u, 0); r.setEnd(node, off); return r.toString().length; };
  const textNodes = (u) => { const w = document.createTreeWalker(u, NodeFilter.SHOW_TEXT), out = []; while (w.nextNode()) out.push(w.currentNode); return out; };
  const unwrap = () => $$("mark.hl", body).forEach((m) => { const p = m.parentNode; while (m.firstChild) p.insertBefore(m.firstChild, m); m.remove(); p.normalize(); });
  const wrap = (u, h) => {
    let pos = 0;
    for (const t of textNodes(u)) {
      const a = pos, b = pos + t.data.length; pos = b;
      if (b <= h.s || a >= h.e) continue;
      let node = t; const s = Math.max(h.s, a) - a, e = Math.min(h.e, b) - a;
      if (e < node.data.length) node.splitText(e);
      if (s > 0) node = node.splitText(s);
      const m = document.createElement("mark"); m.className = `hl hl-${h.c}${h.n ? " has-note" : ""}`; m.dataset.hid = h.id;
      if (h.n) m.title = h.n;
      node.before(m); m.append(node);
    }
  };
  // Багш агуулгаа засвал: хадгалсан текстийг блок дотроос дахин хайж олно; олдохгүй бол жагсаалтад л үлдэнэ.
  const locate = (h) => {
    const u = units().find((x) => bidOf(x) === h.b); if (!u) return null;
    const txt = u.textContent; if (txt.slice(h.s, h.e) === h.t) return u;
    const i = txt.indexOf(h.t); if (i < 0) return null;
    h.s = i; h.e = i + h.t.length; return u;
  };
  const paint = () => {
    unwrap();
    const fb = new Set(frames().map(frameBid));
    for (const h of hs) { if (h.k) { h._lost = !mediaOf(h.b); continue; } if (fb.has(h.b)) { h._lost = false; continue; } const u = locate(h); h._lost = !u; if (u) wrap(u, h); }
    syncFrames(); paintMedia(); paintMarks();
    const n = hs.length; $(".hl-list-btn b", bar).textContent = n;
    $(".hl-list-btn", bar).disabled = $(".hl-quiz-btn", bar).disabled = !n;
    if (!listBox.hidden) renderList();
  };
  const flush = async () => {
    clearTimeout(saveT); if (!dirty) return; dirty = false;
    try { await api(base, { method: "PUT", body: { highlights: hs.map(({ _lost, ...h }) => h) } }); } catch (e) { dirty = true; toast(e.message, true); }
  };
  const save = () => { dirty = true; clearTimeout(saveT); saveT = setTimeout(flush, 500); };
  const hid = () => Date.now().toString(36) + Math.random().toString(36).slice(2, 7);
  // Сонгосон текстийг блок тус бүрээр (байрлал, бүтэн үг хүртэл) задална.
  const selSegs = () => {
    const sel = getSelection(); if (!sel.rangeCount || sel.isCollapsed) return [];
    const r = sel.getRangeAt(0), out = [];
    for (const u of units()) {
      if (!r.intersectsNode(u)) continue;
      const ur = document.createRange(); ur.selectNodeContents(u);
      const txt = u.textContent;
      let s = r.compareBoundaryPoints(Range.START_TO_START, ur) > 0 ? offsetIn(u, r.startContainer, r.startOffset) : 0;
      let e = r.compareBoundaryPoints(Range.END_TO_END, ur) < 0 ? offsetIn(u, r.endContainer, r.endOffset) : txt.length;
      while (s < e && /\s/.test(txt[s])) s++;
      while (e > s && /\s/.test(txt[e - 1])) e--;
      // Үгийн дундаас тасрахгүй — бүтэн үг хүртэл, гэхдээ тухайн текст хэсгээс (гарчиг, догол мөр) хэтрэхгүй.
      const isW = (ch) => !!ch && /[\p{L}\p{N}]/u.test(ch);
      if (e > s) {
        let lo = 0, hi = txt.length, pos = 0;
        for (const t of textNodes(u)) { const a = pos, b = pos + t.data.length; pos = b; if (a <= s && s < b) lo = a; if (a < e && e <= b) { hi = b; break; } }
        while (s > lo && isW(txt[s - 1]) && isW(txt[s])) s--; while (e < hi && isW(txt[e]) && isW(txt[e - 1])) e++;
      }
      e = Math.min(e, s + 5000);
      if (e - s >= 1) out.push({ b: bidOf(u), s, e, t: txt.slice(s, e) });
    }
    sel.removeAllRanges();
    return out;
  };
  const fromSelection = () => {
    const segs = selSegs(); if (!segs.length) return;
    const ids = [];
    for (const g of segs) {
      if (hs.length >= 1000) { toast("Нэг хичээл дээр 1000 хүртэл тодруулга хийнэ", true); break; }
      hs = hs.filter((h) => h.b !== g.b || h.e <= g.s || h.s >= g.e); // давхацсан хуучныг шинээр солино
      const id = hid(); hs.push({ id, ...g, c: color }); ids.push(id);
    }
    if (!ids.length) return;
    paint(); save();
    if (color === "r") openAsk(ids); // ойлгоогүй: багшаас асуух эсвэл тэмдэглээд үлдээх
  };
  const closePop = () => { pop?.remove(); pop = null; };
  const openPop = (h, mark) => {
    closePop();
    pop = document.createElement("div"); pop.className = "hl-pop"; pop.setAttribute("role", "dialog"); pop.setAttribute("aria-label", "Тодруулга");
    pop.innerHTML = `${h.img ? `<img class="hl-pop-img" src="${esc(h.img)}" alt="">` : ""}<div class="hl-pop-colors">${HL_COLORS.map(([c, t]) => `<button type="button" class="hl-sw" data-pc="${c}" aria-pressed="${h.c === c}" title="${t}"><i class="hl-dot hl-${c}"></i>${t}</button>`).join("")}</div>
      <textarea rows="3" maxlength="1000" placeholder="Тэмдэглэл: өөрийн үгээр тайлбарлах, асуулт, жишээ…">${esc(h.n || "")}</textarea>
      <div class="hl-pop-acts"><button type="button" class="btn btn-ghost btn-sm" data-pdel>🗑 Устгах</button>${ctx.teacher ? `<button type="button" class="btn btn-glass btn-sm" data-pask>💬 Багшаас асуух</button>` : ""}<button type="button" class="btn btn-gold btn-sm" data-pok>Болсон</button></div>`;
    card.append(pop);
    const rc = mark.getBoundingClientRect(), cr = card.getBoundingClientRect();
    pop.style.top = `${rc.bottom - cr.top + card.scrollTop + 6}px`;
    pop.style.left = `${Math.max(8, Math.min(rc.left - cr.left, cr.width - pop.offsetWidth - 8))}px`;
    const ta = $("textarea", pop);
    ta.oninput = () => { h.n = ta.value.trim() || undefined; save(); };
    pop.onclick = (e) => {
      const c = e.target.closest("[data-pc]");
      if (c) { h.c = c.dataset.pc; $$("[data-pc]", pop).forEach((x) => x.setAttribute("aria-pressed", x === c)); paint(); save(); return; }
      if (e.target.closest("[data-pdel]")) { hs = hs.filter((x) => x !== h); closePop(); paint(); save(); return; }
      if (e.target.closest("[data-pask]")) { paint(); openAsk([h.id]); return; }
      if (e.target.closest("[data-pok]")) { closePop(); paint(); }
    };
    ta.focus();
  };
  const renderList = () => {
    listBox.innerHTML = HL_COLORS.map(([c, t]) => {
      const items = hs.filter((h) => h.c === c);
      return items.length ? `<div class="hl-grp"><h5><i class="hl-dot hl-${c}"></i>${t} <span class="muted">${items.length}</span></h5>${items.map((h) => `<button type="button" class="hl-item" data-go="${esc(h.id)}" ${h._lost ? "disabled" : ""}>
        ${h.k ? `<span>${h.img ? `<img class="hl-thumb" src="${esc(h.img)}" alt="">` : ""}<b class="hl-when">${h.k === "time" ? "▶" : "📄"} ${esc(h.t || "")}</b> ${esc(clip(mediaTitle(h.b), 60))}</span>` : `<span>${esc(h.t.length > 160 ? h.t.slice(0, 160) + "…" : h.t)}</span>`}${h.n ? `<small>📝 ${esc(h.n)}</small>` : ""}${h._lost ? `<small class="muted">Багш агуулгыг өөрчилсөн тул байршил олдсонгүй</small>` : ""}</button>`).join("")}</div>` : "";
    }).join("") || `<p class="muted small">Тэмдэглэл алга.</p>`;
  };
  const syncFrames = () => frames().forEach((f) => {
    const b = frameBid(f);
    f.contentWindow?.postMessage({ sgHL: { list: hs.filter((h) => h.b === b).map(({ id, s, e, c, n, t }) => ({ id, s, e, c, n, t })), marking, quiz: body.classList.contains("hl-quiz") } }, "*");
  });
  // Гадны холбоос: шинэ таб биш, хичээлийн цонх дотор (курсор хичээлээс гарсанд тооцогдохгүй) нээнэ.
  let lwKey = null;
  const closeLink = () => { $(".lw", card)?.remove(); if (lwKey) { document.removeEventListener("keydown", lwKey, true); lwKey = null; } };
  const openLink = (href) => {
    let u; try { u = new URL(href, location.href); } catch { return; }
    if (!/^https?:$/.test(u.protocol)) return;
    closeLink();
    const lk = embedLink(u.href.replace(/^http:/i, "https:"));
    const w = document.createElement("div"); w.className = "lw"; w.setAttribute("role", "dialog"); w.setAttribute("aria-modal", "true"); w.setAttribute("aria-label", "Холбоос");
    w.innerHTML = `<div class="lw-card"><div class="lw-head"><b>🔗 ${esc(u.hostname.replace(/^www\./, ""))}</b><span class="lw-url">${esc(clip(u.href, 90))}</span><button type="button" class="icon-btn" data-lw-x aria-label="Хаах">✕</button></div>
      <div class="lw-body">${lk?.video ? `<div class="player">${mediaHTML(lk.src)}</div>` : lk ? `<iframe src="${esc(lk.src)}" sandbox="allow-scripts allow-same-origin allow-forms allow-presentation" allow="fullscreen; autoplay; encrypted-media; picture-in-picture; clipboard-write" allowfullscreen referrerpolicy="strict-origin-when-cross-origin" title="Холбоос"></iframe>` : `<p class="muted lw-none">Энэ холбоосыг нээх боломжгүй.</p>`}</div>
      <p class="muted small lw-foot">Хуудас харагдахгүй байвал тухайн сайт өөр сайт дотор харагдахыг хориглосон байна.</p></div>`;
    card.append(w);
    w.onclick = (e) => { if (e.target === w || e.target.closest("[data-lw-x]")) closeLink(); };
    lwKey = (e) => { if (e.key === "Escape") { e.stopPropagation(); closeLink(); } };
    document.addEventListener("keydown", lwKey, true);
    $("[data-lw-x]", w).focus();
  };
  const onMsg = (e) => {
    const f = e.data && frames().find((x) => x.contentWindow === e.source); if (!f) return;
    const b = frameBid(f), sl = e.data.sgHLsel, ck = e.data.sgHLclick;
    if (e.data.sgHLready) return syncFrames();
    if (typeof e.data.sgLink === "string") return openLink(e.data.sgLink);
    if (sl && marking && Number.isInteger(sl.s) && Number.isInteger(sl.e) && sl.e > sl.s && typeof sl.t === "string") {
      if (hs.length >= 1000) return toast("Нэг хичээл дээр 1000 хүртэл тодруулга хийнэ", true);
      hs = hs.filter((h) => h.b !== b || h.e <= sl.s || h.s >= sl.e);
      const id = hid(); hs.push({ id, b, s: sl.s, e: Math.min(sl.e, sl.s + 5000), t: sl.t.slice(0, 5000), c: color });
      paint(); save(); if (color === "r") openAsk([id]);
      return;
    }
    if (ck) { const h = hs.find((x) => x.id === ck.id && x.b === b); if (!h) return; const fr = f.getBoundingClientRect(); openPop(h, { getBoundingClientRect: () => ({ left: fr.left + (+ck.x || 0), bottom: fr.top + (+ck.y || 0) }) }); }
  };
  const setOn = (on) => {
    marking = on; body.classList.toggle("hl-on", on); syncFrames();
    const t = $(".hl-toggle", bar); t.setAttribute("aria-pressed", on); $(".hl-lbl", t).textContent = on ? " Маркер асаалттай" : " Маркер";
    $(".hl-colors", bar).hidden = !on;
    if (on) { setQuiz(false); toast("Тодруулах текстээ хулганаар чирж (утсанд удаан дарж) сонгоно уу"); }
  };
  const setQuiz = (q) => {
    body.classList.toggle("hl-quiz", q); $(".hl-quiz-btn", bar).setAttribute("aria-pressed", q);
    $$("mark.hl.shown", body).forEach((m) => m.classList.remove("shown")); syncFrames();
    if (q) { setOn(false); closeAsk(); closePop(); toast("Тодруулсан хэсгүүд нуугдлаа — эхлээд өөрөө санаад, дараа нь дарж шалгаарай 🧠"); }
  };
  bar.onclick = (e) => {
    if (e.target.closest(".hl-toggle")) return setOn(!marking);
    const sw = e.target.closest("[data-c]");
    if (sw) { color = sw.dataset.c; $$("[data-c]", bar).forEach((x) => x.setAttribute("aria-checked", x === sw)); return; }
    if (e.target.closest(".hl-quiz-btn")) return setQuiz(!body.classList.contains("hl-quiz"));
    if (e.target.closest("[data-ask-x]")) return closeAsk();
    if (e.target.closest("[data-ask-keep]")) return keepAsk();
    if (e.target.closest("[data-ask-send]")) return sendAsk();
    const oc = e.target.closest("[data-ask-open]");
    if (oc) { closeAsk(); window.openRailChat?.(oc.dataset.askOpen); return; }
    if (e.target.closest(".hl-list-btn")) { closeAsk(); listBox.hidden = !listBox.hidden; $(".hl-list-btn", bar).setAttribute("aria-expanded", !listBox.hidden); if (!listBox.hidden) renderList(); return; }
    const go = e.target.closest("[data-go]");
    if (go) {
      const m = $(`mark.hl[data-hid="${CSS.escape(go.dataset.go)}"]`, body), h = hs.find((x) => x.id === go.dataset.go); listBox.hidden = true;
      if (h?.k) return goMedia(h);
      if (m) { m.scrollIntoView({ behavior: "smooth", block: "center" }); m.classList.remove("flash"); void m.offsetWidth; m.classList.add("flash"); }
      else if (h) $(`.rb-embed[data-bid="${CSS.escape(h.b)}"]`, body)?.scrollIntoView({ behavior: "smooth", block: "center" }); // embed доторх тодруулга
    }
  };
  const onUp = () => { if (marking) setTimeout(fromSelection, 0); };
  const onClick = (e) => {
    const a = e.target.closest?.("a[href]");
    if (a && /^https?:/i.test(a.href)) { let ext = false; try { ext = new URL(a.href).host !== location.host; } catch {} if (ext) { e.preventDefault(); openLink(a.href); return; } }
    const m = e.target.closest?.("mark.hl"); if (!m) return;
    if (body.classList.contains("hl-quiz")) { m.classList.toggle("shown"); return; }
    if (!getSelection().isCollapsed) return;
    const h = hs.find((x) => x.id === m.dataset.hid); if (h) openPop(h, m);
  };
  const onDoc = (e) => { if (pop && !pop.contains(e.target) && !e.target.closest?.("mark.hl, [data-mn-go]")) { closePop(); paint(); } };
  /* медиа тэмдэглэл: «📝 Энэ мөчид / хуудсанд тэмдэглэх» + тэмдэглэлүүд (дарж тэр мөч, хуудас руу) */
  const fmtT = (s) => { s = Math.max(0, Math.floor(+s || 0)); const hh = Math.floor(s / 3600), mm = Math.floor((s % 3600) / 60), ss = String(s % 60).padStart(2, "0"); return hh ? `${hh}:${String(mm).padStart(2, "0")}:${ss}` : `${mm}:${ss}`; };
  const isDoc = (fig) => fig.classList.contains("rb-doc");
  const mPlayer = (fig) => $("video, audio, iframe[data-yt]", fig);
  const mNow = (fig) => { const v = mPlayer(fig); return v ? (v._sgNow ? v._sgNow() : v.currentTime || 0) : 0; };
  const mDur = (fig) => { const v = mPlayer(fig); return v ? (v._sgDur ? v._sgDur() : v.duration || 0) : 0; };
  const mDv = (fig) => $(".doc-view", fig)?._dv;
  const pageWord = (fig) => ($(".doc-view.dv-slides", fig) ? "слайд" : "хуудас");
  function mediaTitle(b) { const fig = mediaOf(b); return fig ? ($(".dv-title b", fig)?.textContent || $("b", fig)?.textContent || (isDoc(fig) ? "Баримт" : "Видео")) : ""; }
  function goMedia(h) {
    const fig = mediaOf(h.b); if (!fig) return;
    fig.scrollIntoView({ behavior: "smooth", block: "center" });
    if (h.k === "time") { const v = mPlayer(fig); if (v?._sgSeek) v._sgSeek(h.sec); else if (v) { try { v.currentTime = h.sec; } catch {} } }
    else mDv(fig)?.go?.(h.p);
  }
  function mountMedia() {
    for (const fig of medias()) {
      if (fig.querySelector(":scope > .mn")) continue;
      const doc = isDoc(fig), el = document.createElement("div"); el.className = "mn"; el.dataset.mn = fig.dataset.bid;
      el.innerHTML = `<div class="mn-line" hidden></div><div class="mn-row"><button type="button" class="mn-add" title="${doc ? "Одоо харж буй хуудсанд тэмдэглэл хөтлөх" : "Энэ мөчид тэмдэглэл хөтлөх — дараа нь дарж яг энэ мөч рүү очно"}">📝 <span>${doc ? "Энэ хуудсанд тэмдэглэх" : "Энэ мөчид тэмдэглэх"}</span><b class="mn-at"></b></button>${doc ? `<button type="button" class="mn-add mn-box" title="Хуудсан дээрх чухал хэсгийг дөрвөлжингөөр хүрээлж, зургийг нь тэмдэглэлд хадгална">▢ <span>Хэсгийг хүрээлж тэмдэглэх</span></button>` : ""}<div class="mn-chips"></div></div><div class="mn-form" hidden></div>`;
      const cap = fig.querySelector(":scope > figcaption"); cap ? cap.before(el) : fig.append(el);
    }
  }
  function paintMedia() {
    for (const el of $$(".mn", body)) {
      const fig = el.parentElement, b = el.dataset.mn, d = +el.dataset.dur || 0;
      const list = hs.filter((h) => h.k && h.b === b).sort((x, y) => (x.sec || x.p || 0) - (y.sec || y.p || 0));
      $(".mn-chips", el).innerHTML = list.map((h) => `<button type="button" class="mn-chip" data-mn-go="${esc(h.id)}" title="${esc(h.n || "Тэмдэглэл")}"><i class="hl-dot hl-${h.c}"></i>${h.img ? `<img src="${esc(h.img)}" alt="">` : ""}<b>${esc(h.t || "")}</b>${h.n ? `<span>${esc(clip(h.n, 38))}</span>` : ""}</button>`).join("");
      const line = $(".mn-line", el);
      line.hidden = isDoc(fig) || !d || !list.length;
      if (!line.hidden) line.innerHTML = list.map((h) => `<button type="button" class="mn-dot" data-mn-go="${esc(h.id)}" style="left:${Math.min(100, (h.sec / d) * 100)}%" title="${esc(h.t + (h.n ? " — " + h.n : ""))}"><i class="hl-dot hl-${h.c}"></i></button>`).join("");
    }
  }
  const mTick = () => {
    for (const el of $$(".mn", body)) {
      const fig = el.parentElement, at = $(".mn-at", el);
      if (isDoc(fig)) { const p = mDv(fig)?.page?.(); at.textContent = p ? ` · ${p}-р ${pageWord(fig)}` : ""; continue; }
      at.textContent = " · " + fmtT(mNow(fig));
      const d = Math.round(mDur(fig)); if (d && +el.dataset.dur !== d) { el.dataset.dur = d; paintMedia(); }
    }
  };
  const mTimer = setInterval(() => { if (!document.hidden) { mTick(); paintMarks(); } }, 1000);
  const openForm = (el, pre) => {
    const fig = el.parentElement, doc = isDoc(fig), form = $(".mn-form", el);
    let nw;
    if (doc) { const p = pre?.p || mDv(fig)?.page?.() || 1; nw = { k: "page", p, t: `${p}-р ${pageWord(fig)}`, box: pre?.box, blob: pre?.blob }; }
    else { const t = mNow(fig), v = mPlayer(fig); nw = { k: "time", sec: Math.round(t * 10) / 10, t: fmtT(t) }; try { v?._sgPause ? v._sgPause() : v?.pause?.(); } catch {} }
    nw.c = color === "r" ? "y" : color;
    form._new = nw; form.hidden = false;
    if (form._url) URL.revokeObjectURL(form._url);
    form._url = nw.blob ? URL.createObjectURL(nw.blob) : "";
    form.innerHTML = `${form._url ? `<img class="mn-shot" src="${form._url}" alt="Хүрээлсэн хэсэг">` : ""}<div class="mn-fh"><b>${doc ? "📄" : "▶"} ${esc(nw.t)}</b><span class="mn-sw">${HL_COLORS.map(([c, t]) => `<button type="button" class="hl-sw" data-mc="${c}" aria-pressed="${c === nw.c}" title="${t}"><i class="hl-dot hl-${c}"></i><span class="hl-lbl">${t}</span></button>`).join("")}</span></div>
      <textarea rows="2" maxlength="1000" placeholder="${doc ? "Энэ хуудсанд юу чухал байв? Өөрийн үгээр тэмдэглэнэ…" : "Энэ мөчид юу ярьсан бэ? Гол санаа, асуултаа тэмдэглэнэ…"}"></textarea>
      <div class="hl-pop-acts"><button type="button" class="btn btn-ghost btn-sm" data-mn-x>Болих</button><button type="button" class="btn btn-gold btn-sm" data-mn-save>Хадгалах</button></div>`;
    $("textarea", form).focus();
  };
  const saveForm = async (el) => {
    const form = $(".mn-form", el), nw = form._new; if (!nw || form._busy) return;
    if (hs.length >= 1000) return toast("Нэг хичээл дээр 1000 хүртэл тэмдэглэл хөтөлнө", true);
    const n = $("textarea", form).value.trim();
    const h = { id: hid(), b: el.dataset.mn, s: 0, e: 0, c: nw.c, t: nw.t, k: nw.k };
    if (nw.k === "time") h.sec = nw.sec; else h.p = nw.p;
    if (nw.box) h.box = nw.box;
    if (nw.blob) { // хүрээлсэн хэсгийн зургийг серверт
      form._busy = true; const b = $("[data-mn-save]", form); b.disabled = true; b.textContent = "Хадгалж байна…";
      try {
        const r = await fetch("/api/me/note-images", { method: "POST", headers: { Authorization: "Bearer " + Auth.token, "Content-Type": nw.blob.type || "image/png" }, body: nw.blob });
        const d = await r.json().catch(() => null); if (!r.ok) throw new Error(d?.error || "Зургийг хадгалж чадсангүй");
        h.img = d.url;
      } catch (e) { toast(e.message, true); }
      finally { form._busy = false; }
      if (!form._new) return;
    }
    if (n) h.n = n.slice(0, 1000);
    if (form._url) { URL.revokeObjectURL(form._url); form._url = ""; }
    hs.push(h); form._new = null; form.hidden = true; form.innerHTML = "";
    paint(); save(); toast("📝 Тэмдэглэгдлээ — «Миний тэмдэглэл»-ээс бүгдийг харна");
  };
  const onMn = (e) => {
    const el = e.target.closest?.(".mn"); if (!el || !body.contains(el)) return;
    const form = $(".mn-form", el);
    if (e.target.closest(".mn-box")) { form.hidden = true; form._new = null; return startBox(el); }
    if (e.target.closest(".mn-add")) { if (!form.hidden) { form.hidden = true; form._new = null; return; } return openForm(el); }
    const sw = e.target.closest("[data-mc]");
    if (sw && form._new) { form._new.c = sw.dataset.mc; $$("[data-mc]", form).forEach((x) => x.setAttribute("aria-pressed", x === sw)); return; }
    if (e.target.closest("[data-mn-x]")) { form.hidden = true; form._new = null; form.innerHTML = ""; return; }
    if (e.target.closest("[data-mn-save]")) return saveForm(el);
    const g = e.target.closest("[data-mn-go]");
    if (g) { const h = hs.find((x) => x.id === g.dataset.mnGo); if (h) { goMedia(h); openPop(h, g); } }
  };
  const onMnKey = (e) => { if (e.key === "Enter" && (e.ctrlKey || e.metaKey) && e.target.closest?.(".mn-form")) { e.preventDefault(); saveForm(e.target.closest(".mn")); } };
  /* ▢ Хэсгийг хүрээлж тэмдэглэх: баримтын хуудсан дээр дөрвөлжин зурж, тэр хэсгийн зургийг хадгална. */
  let boxSel = null;
  const pageEl = (dv, p) => $(`.face[data-p="${p}"]`, dv) || $$("section.docx", dv)[p - 1] || (dv._dv?.page?.() === p ? $(".sl-slide", dv) : null);
  const cancelBox = () => { if (!boxSel) return; boxSel.ov.remove(); document.removeEventListener("keydown", boxSel.key, true); boxSel = null; };
  function startBox(el) {
    const fig = el.parentElement, dv = $(".doc-view", fig), host = dv && $(".dv-body", dv);
    if (!dv?._dv || !host) return toast("Баримт ачаалагдаж дуустал түр хүлээнэ үү");
    cancelBox();
    dv.scrollIntoView({ behavior: "smooth", block: "center" });
    const ov = document.createElement("div"); ov.className = "mn-sel";
    ov.innerHTML = `<p class="mn-sel-hint">▢ Чухал хэсгийг хулганаар (утсанд хуруугаар) дөрвөлжин хүрээлнэ · <button type="button">Болих</button></p><i class="mn-rect" hidden></i>`;
    host.append(ov);
    const key = (e) => { if (e.key === "Escape") { e.preventDefault(); e.stopPropagation(); cancelBox(); } };
    document.addEventListener("keydown", key, true);
    boxSel = { el, fig, dv, ov, key };
    let a = null; const rc = $(".mn-rect", ov);
    $("button", ov).onclick = (e) => { e.stopPropagation(); cancelBox(); };
    ov.addEventListener("pointerdown", (e) => { if (e.target.closest("button")) return; e.preventDefault(); e.stopPropagation(); ov.setPointerCapture(e.pointerId); a = { x: e.clientX, y: e.clientY }; });
    ov.addEventListener("pointermove", (e) => {
      if (!a) return; const r = ov.getBoundingClientRect(); rc.hidden = false;
      Object.assign(rc.style, { left: Math.min(a.x, e.clientX) - r.left + "px", top: Math.min(a.y, e.clientY) - r.top + "px", width: Math.abs(e.clientX - a.x) + "px", height: Math.abs(e.clientY - a.y) + "px" });
    });
    ov.addEventListener("pointerup", (e) => {
      if (!a) return; e.stopPropagation();
      const s = { left: Math.min(a.x, e.clientX), top: Math.min(a.y, e.clientY), right: Math.max(a.x, e.clientX), bottom: Math.max(a.y, e.clientY) }; a = null;
      if (s.right - s.left < 12 || s.bottom - s.top < 12) { rc.hidden = true; return; }
      finishBox(s);
    });
  }
  async function finishBox(s) {
    const { el, dv, ov } = boxSel, cx = (s.left + s.right) / 2, cy = (s.top + s.bottom) / 2;
    ov.style.pointerEvents = "none"; const under = document.elementsFromPoint(cx, cy); ov.style.pointerEvents = "";
    let page = under.find((x) => x.matches?.(".face[data-p]")), p = page ? +page.dataset.p : 0;
    if (!page && (page = under.find((x) => x.matches?.("section.docx")))) p = $$("section.docx", dv).indexOf(page) + 1;
    if (!page && (page = under.find((x) => x.matches?.(".sl-slide")) || $(".sl-slide", dv))) p = dv._dv.page();
    if (!page || !p) { toast("Хуудсан дээр хүрээлнэ үү", true); return; }
    const pr = page.getBoundingClientRect();
    const x1 = clampN(s.left, pr.left, pr.right), x2 = clampN(s.right, pr.left, pr.right), y1 = clampN(s.top, pr.top, pr.bottom), y2 = clampN(s.bottom, pr.top, pr.bottom);
    if (x2 - x1 < 8 || y2 - y1 < 8) { toast("Хуудсан дээр хүрээлнэ үү", true); return; }
    const r2 = (v) => Math.round(v * 100) / 100;
    const box = [r2(((x1 - pr.left) / pr.width) * 100), r2(((y1 - pr.top) / pr.height) * 100), r2(((x2 - x1) / pr.width) * 100), r2(((y2 - y1) / pr.height) * 100)];
    ov.classList.add("busy"); $(".mn-sel-hint", ov).textContent = "Зургийг бэлдэж байна…";
    let blob = null;
    try { blob = await captureBox(page, { x: x1 - pr.left, y: y1 - pr.top, w: x2 - x1, h: y2 - y1 }, pr); } catch {}
    cancelBox();
    if (!blob) toast("Зургийг авч чадсангүй — хүрээ, тэмдэглэл хадгалагдана", true);
    openForm(el, { p, box, blob });
  }
  const clampN = (v, lo, hi) => Math.max(lo, Math.min(hi, v));
  async function captureBox(page, r, pr) {
    const src = page.querySelector(":scope > canvas"); // PDF: хуудас аль хэдийн canvas дээр зурагдсан
    let c;
    if (src) {
      const sx = src.width / pr.width, sy = src.height / pr.height;
      c = document.createElement("canvas"); c.width = Math.max(1, Math.round(r.w * sx)); c.height = Math.max(1, Math.round(r.h * sy));
      c.getContext("2d").drawImage(src, r.x * sx, r.y * sy, r.w * sx, r.h * sy, 0, 0, c.width, c.height);
    } else { // Word, PowerPoint: HTML-ийг зураг болгоно
      await loadScript(DOC_LIBS.h2c);
      // Word харагч CSS zoom-оор багтаадаг — хуулбар дээр zoom-ыг арилгаж, хүрээг жинхэнэ хэмжээнд шилжүүлнэ.
      const zr = page.closest(".dx-doc"), z = (zr && parseFloat(zr.style.zoom)) || 1;
      c = await window.html2canvas(page, { backgroundColor: "#ffffff", scale: Math.max(1.5, 2 * z), logging: false, useCORS: true, x: r.x / z, y: r.y / z, width: r.w / z, height: r.h / z,
        onclone: (d) => d.querySelectorAll(".dx-doc").forEach((x) => (x.style.zoom = "1")),
        ignoreElements: (n) => n.classList?.contains("mn-mark") || n.classList?.contains("mn-sel") });
    }
    return new Promise((res) => c.toBlob((b) => (b && b.type === "image/webp" ? res(b) : c.toBlob(res, "image/png")), "image/webp", 0.9));
  }
  // Хүрээлсэн хэсгүүдийг хуудсан дээр өнгөөрөө харуулна (хуудас солигдох, дахин зурагдахад тик бүрт шалгана).
  function paintMarks() {
    for (const fig of medias()) {
      if (!isDoc(fig)) continue;
      const dv = $(".doc-view", fig); if (!dv?._dv) continue;
      const want = new Set();
      for (const h of hs) {
        if (h.k !== "page" || !h.box || h.b !== fig.dataset.bid) continue;
        const page = pageEl(dv, h.p); if (!page) continue;
        want.add(h.id);
        let m = page.querySelector(`:scope > .mn-mark[data-mk="${CSS.escape(h.id)}"]`);
        if (!m) {
          $$(`.mn-mark[data-mk="${CSS.escape(h.id)}"]`, dv).forEach((x) => x.remove());
          m = document.createElement("button"); m.type = "button"; m.dataset.mk = h.id; page.append(m);
          ["pointerdown", "pointerup", "mousedown", "touchstart"].forEach((t) => m.addEventListener(t, (e) => e.stopPropagation()));
          m.addEventListener("click", (e) => { e.stopPropagation(); e.preventDefault(); const x = hs.find((y) => y.id === m.dataset.mk); if (x) openPop(x, m); });
        }
        m.className = `mn-mark mk-${h.c}`; m.title = h.n || "Тэмдэглэл";
        Object.assign(m.style, { left: h.box[0] + "%", top: h.box[1] + "%", width: h.box[2] + "%", height: h.box[3] + "%" });
      }
      $$(".mn-mark", dv).forEach((m) => { if (!want.has(m.dataset.mk)) m.remove(); });
    }
  }
  mountMedia(); mTick();
  body.addEventListener("click", onMn); body.addEventListener("keydown", onMnKey);
  body.addEventListener("pointerup", onUp); body.addEventListener("keyup", onUp); body.addEventListener("click", onClick);
  document.addEventListener("pointerdown", onDoc); addEventListener("message", onMsg);
  (ctx.preload ? Promise.resolve(ctx.preload) : api(base)).then((d) => { // хичээлийн хариунд ирсэн бол дахин асуухгүй
    hs = d.highlights || []; paint();
    const nid = new URLSearchParams(location.hash.slice(1)).get("n"), h = nid && hs.find((x) => x.id === nid);
    if (h) setTimeout(() => { if (h.k) return goMedia(h); const m = $(`mark.hl[data-hid="${CSS.escape(h.id)}"]`, body); if (m) { m.scrollIntoView({ behavior: "smooth", block: "center" }); m.classList.remove("flash"); void m.offsetWidth; m.classList.add("flash"); } }, 900);
  }).catch(() => {});
  return () => {
    flush(); closePop(); closeLink(); clearInterval(mTimer);
    body.removeEventListener("click", onMn); body.removeEventListener("keydown", onMnKey);
    body.removeEventListener("pointerup", onUp); body.removeEventListener("keyup", onUp); body.removeEventListener("click", onClick);
    document.removeEventListener("pointerdown", onDoc); removeEventListener("message", onMsg);
  };
}

/* ---------- Хичээл үзэх үеийн анхаарал: өөр таб, цонх руу шилжвэл сануулга, 3 дахь удаад зогсоно ---------- */
const WATCH_MAX_WARN = 3;
// tone: "danger" (таб солих — улаан, сэгсрэлт, дохио), "stop" (хичээл зогссон), "info" (идэвхийн шалгалт).
// Бүтэн дэлгэцтэй үед (PDF, ном) body-д нэмсэн цонх харагдахгүй тул бүтэн дэлгэцийн элемент дотор нэмнэ;
// видео, iframe бүтэн дэлгэц хүүхэд элемент харуулдаггүй тул бүтэн дэлгэцээс гаргана.
function overlayHost(exit) {
  const fs = document.fullscreenElement || document.webkitFullscreenElement;
  if (!fs) return document.body;
  if (exit || fs.tagName === "IFRAME" || fs.tagName === "VIDEO") { (document.exitFullscreen || document.webkitExitFullscreen)?.call(document); return document.body; }
  return fs;
}
function watchAlert(html, btn, onClose, tone = "info") {
  $(".watch-alert")?.remove();
  const el = document.createElement("div");
  el.className = `watch-alert wa-${tone}`; el.setAttribute("role", "alertdialog"); el.setAttribute("aria-modal", "true");
  el.innerHTML = `<div class="watch-card">${html}<button class="btn ${tone === "info" ? "btn-gold" : "wa-btn"} btn-block">${btn}</button></div>`;
  overlayHost(tone === "stop").append(el); // зогсоход хичээлийн цонх хаагдсан тул бүтэн дэлгэцээс гарна
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
    const notes = kind === "stop" ? [[523, 0], [392, 0.2], [262, 0.4]] : kind === "ping" ? [[660, 0], [880, 0.18]] : [[988, 0], [740, 0.16], [988, 0.32]];
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
// lazy — хичээл суралцагчийн оролцоогүйгээр (багш хоригийг цуцлахад) нээгдсэн: суралцагч анх хөдлөх хүртэл
// идэвхгүй, гадуур, идэвхийн шалгалтыг тоолохгүй (компьютерийн дэргэд байхгүй үед дахин хаагдахаас сэргийлнэ).
function watchLesson({ courseId, lessonId, modal, onStop, onActive, maxWarn, lazy }) {
  const WATCH_MAX_WARN = Math.max(1, +maxWarn || 3); // багш сургалтын тохиргоонд тоогоор оруулна
  if (!Auth.token) return () => {};
  let sid = null, owner = false, warns = 0, stopped = false, away = false, awayAt = 0, last = Date.now(), lastInput = Date.now(), idleLogged = false, armed = !lazy;
  let acc = { active: 0, idle: 0, away: 0 }, events = [], pingAt = Date.now() + WATCH_PING_SEC * 1000, pingTimer = null, wmTimer = null;
  // Идэвхийн оноо E = идэвхтэй / нийт (сүүлийн 5 минут). 3+ минут хэмжээд E < 0.1 бол хичээл зогсоно.
  const win = [];
  const engagement = () => { const a = win.reduce((n, x) => n + x[0], 0), t = win.reduce((n, x) => n + x[1], 0); return { e: t ? a / t : 1, t }; };
  const card = $(".modal-card", modal);
  const badge = document.createElement("div");
  badge.className = "watch-badge";
  // «Юуг тэмдэглэдэг вэ?» — шинэ таб нээвэл таб сольсонд тооцогдох тул тайлбарыг хичээлийн цонхон дотор харуулна.
  const why = document.createElement("div"); why.className = "wb-info"; why.id = "wbInfo"; why.hidden = true;
  why.innerHTML = `<b>Хичээлийн цонх нээлттэй үед л</b> таб, цонх сольсон, курсор цонхноос гарсан, 2 минутаас удаан хөдөлгөөнгүй байсан, хуулах гэсэн үйлдлийг тэмдэглэнэ. Камер, микрофон ашиглахгүй, дэлгэц бичихгүй. Тэмдэглэлийг тухайн сургалтын багш харж, 180 хоногийн дараа устна.`;
  const paint = () => { badge.innerHTML = `${icon("eye", 14)} Анхаарлын горим · <span class="wb-long">өөр таб руу шилжвэл </span>сануулга <b>${warns}/${WATCH_MAX_WARN}</b><button type="button" class="wb-why" aria-controls="wbInfo" aria-expanded="${!why.hidden}">Юуг тэмдэглэдэг вэ?</button>`; badge.classList.toggle("warned", warns > 0); };
  badge.addEventListener("click", (e) => { const b = e.target.closest(".wb-why"); if (!b) return; why.hidden = !why.hidden; b.setAttribute("aria-expanded", String(!why.hidden)); });
  paint(); card.prepend(badge); badge.after(why);
  const wm = document.createElement("div"); wm.className = "wm"; wm.setAttribute("aria-hidden", "true");
  const playing = () => $$("video, audio", modal).some((v) => !v.paused && !v.ended) || !!modal._ytPlaying;
  // Хичээлийн embed (iframe) дотор ажиллаж байхад хулгана, товчлуурын үйлдэл энэ хуудсанд ирэхгүй — идэвхгүйд тооцохгүй.
  const inFrame = () => { const a = document.activeElement; return a?.tagName === "IFRAME" && modal.contains(a); };
  const tick = () => {
    const now = Date.now(), dt = Math.round((now - last) / 1000); last = now;
    if (dt <= 0 || !armed) return;
    if (away) acc.away += dt;
    else if (now - lastInput > WATCH_IDLE_MS && !playing() && !inFrame()) { acc.idle += dt; if (!idleLogged) { idleLogged = true; events.push({ type: "idle", detail: "2 минутаас дээш хөдөлгөөнгүй" }); } }
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
  const flashOn = (msg = "⚠️ Хичээл рүүгээ буцна уу!") => { if (owner || flashT) return; let f = false; flashT = setInterval(() => { document.title = (f = !f) ? msg : title0; }, 900); };
  const flashOff = () => { clearInterval(flashT); flashT = 0; document.title = title0; };
  const leave = () => { if (away || stopped || !armed) return; tick(); away = true; awayAt = Date.now(); pauseMedia(); flashOn(); };
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
      <p>Идэвхийн шалгалт. <b class="ping-left">${left}</b> секундэд хариулахгүй бол хичээл зогсоно.</p>`, "Тийм, үзэж байна", () => { answered = true; lastInput = Date.now(); flashOff(); });
    alarm("ping"); flashOn("🙋 Та хичээлээ үзэж байна уу?"); // өөр зүйл рүү харж байсан ч анзаарна
    const t = setInterval(() => {
      if (answered || stopped) { clearInterval(t); return; }
      left--; const el = $(".watch-alert .ping-left"); if (el) el.textContent = left;
      if (left <= 0) { clearInterval(t); $(".watch-alert")?.remove(); flashOff(); events.push({ type: "ping_missed", detail: "30 секундэд хариулаагүй" }); stop("Идэвхийн шалгалтад хариулаагүй", "Та идэвхийн шалгалтад 30 секундэд хариулаагүй тул хичээл зогслоо."); }
    }, 1000);
  };
  pingTimer = setInterval(() => { if (armed && !away && Date.now() >= pingAt) { pingAt = Date.now() + WATCH_PING_SEC * 1000; ping(); } }, 1000);
  // Таб нуугдах: найдвартай. Цонхны blur: видео тоглуулагч (iframe) дээр дарахад ч гардаг тул шалгана.
  const onVis = () => (document.hidden ? leave() : back());
  const onBlur = () => setTimeout(() => { if (document.activeElement?.tagName !== "IFRAME" && !document.hasFocus()) leave(); }, 0);
  const onFocus = () => { if (!document.hidden) back(); };
  const onInput = () => {
    lastInput = Date.now(); idleLogged = false;
    if (!armed) { armed = true; last = Date.now(); pingAt = Date.now() + WATCH_PING_SEC * 1000; } // одооноос тоолно
  };
  const inputs = ["pointermove", "pointerdown", "keydown", "wheel", "touchstart"];
  // Курсор хичээлийн цонхноос 2.5 сек+ гарвал тусдаа (шар) сануулга; буцаж ороход алга болж, хугацааг нь тэмдэглэнэ.
  const fine = matchMedia("(hover: hover) and (pointer: fine)").matches;
  let outT = 0, outAt = 0, outs = 0, ca = null;
  const area = modal.closest("#lessonStage") || card; // хичээл үзэх горимд хөтөлбөрийн самбар ч хичээлийн цонхонд тооцогдоно
  // Цагаан самбар (бүтэн дэлгэцээр нээгддэг) — хичээлийн нэг хэсэг тул курсор «гадуур» гэж тооцохгүй.
  const overlayOpen = () => $(".watch-alert") || $(".dv-full") || document.documentElement.classList.contains("bd-on") || $$(".modal.open").some((m) => m !== modal);
  const cursorBack = () => {
    clearTimeout(outT); if (!ca) return;
    events.push({ type: "cursor_out", detail: `${Math.max(1, Math.round((Date.now() - outAt) / 1000))} сек курсор хичээлийн цонхноос гадуур (${outs} дахь удаа)` });
    const el = ca; ca = null; el.classList.add("out"); setTimeout(() => el.remove(), 250); card.classList.remove("cursor-out");
  };
  const onLeaveCard = (e) => {
    if (!fine || owner || stopped || !armed || e.pointerType === "touch" || (e.relatedTarget && (area.contains(e.relatedTarget) || e.relatedTarget.closest?.(".bd")))) return;
    clearTimeout(outT); outAt = Date.now();
    outT = setTimeout(() => { if (stopped || away || owner || ca || overlayOpen() || area.matches(":hover")) return; outs++; ca = cursorAlert(outs); card.classList.add("cursor-out"); }, 2500);
  };
  area.addEventListener("pointerleave", onLeaveCard); area.addEventListener("pointerenter", cursorBack);
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
  // Маркер асаалттай үед л хичээлийн текстийг сонгож болно (хуулах нь хориотой хэвээр).
  const marking = (t) => (t.nodeType === 3 ? t.parentElement : t)?.closest?.(".lesson-content.hl-on .rb-text, .lesson-content.hl-on.pre");
  const block = (e) => { if (!editable(e.target) && !owner && !(e.type === "selectstart" && marking(e.target))) e.preventDefault(); };
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
    area.removeEventListener("pointerleave", onLeaveCard); area.removeEventListener("pointerenter", cursorBack);
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
      // Маркерын тэмдэглэл (мөчөөр): 6 минутын хэсгүүдтэй бол нийт хугацаагаар.
      v._sgNow = () => offset + v.currentTime;
      v._sgDur = () => (parts.length > 1 ? SEG * (parts.length - 1) + (idx === parts.length - 1 ? v.duration || 0 : SEG) : v.duration || 0);
      v._sgSeek = (t) => {
        const i = parts.length > 1 ? Math.min(parts.length - 1, Math.floor(t / SEG)) : 0;
        if (!free && i > unlockedPart) return note();
        const set = () => { v.currentTime = Math.max(0, t - i * SEG); };
        if (parts.length > 1 && i !== idx) { loadPart(i, false); v.addEventListener("loadedmetadata", set, { once: true }); } else set();
      };
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
      v._sgNow = () => { try { return p.getCurrentTime() || 0; } catch { return 0; } };
      v._sgDur = () => { try { return p.getDuration() || 0; } catch { return 0; } };
      v._sgSeek = (t) => { try { if (!free && t > max + 0.5) { note(); t = max; } p.seekTo(Math.max(0, t), true); } catch {} };
      v._sgPause = () => { try { p.pauseVideo(); } catch {} };
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
// Рубрикийн хүснэгт (харах): сонгосон нүд тодорч, мөр бүрийн оноо ба нийт гарна (суралцагч, журнал, үнэлэх цонх).
function rubricViewHTML(r, picks) {
  if (!r?.levels?.length || !r.criteria?.length) return "";
  const top = (c) => Math.max(0, ...c.points);
  const max = r.criteria.reduce((a, c) => a + top(c), 0);
  const got = picks ? r.criteria.reduce((a, c) => a + (picks[c.id] != null ? c.points[picks[c.id]] || 0 : 0), 0) : 0;
  return `<div class="rb-wrap"><table class="rb-view ${picks ? "graded" : ""}">
    <thead><tr><th>Шалгуур</th>${r.levels.map((lv) => `<th>${esc(lv)}</th>`).join("")}${picks ? "<th>Авсан</th>" : ""}</tr></thead>
    <tbody>${r.criteria.map((c) => { const pk = picks?.[c.id]; return `<tr><th>${esc(c.name)}</th>${r.levels.map((_, j) => `<td class="${pk === j ? "on" : ""}"><b>${c.points[j]} оноо</b>${c.desc[j] ? `<small>${esc(c.desc[j])}</small>` : ""}</td>`).join("")}${picks ? `<td class="rb-got"><b>${pk != null ? c.points[pk] : "—"}</b><small>/${top(c)}</small></td>` : ""}</tr>`; }).join("")}</tbody>
    ${picks ? `<tfoot><tr><th colspan="${r.levels.length + 1}">Нийт дүн</th><td class="rb-got"><b>${got}</b><small>/${max}</small></td></tr></tfoot>` : ""}</table></div>`;
}

async function assignmentCard(box, { courseId, lessonId }) {
  const base = `/api/courses/${courseId}/lessons/${lessonId}`;
  const card = document.createElement("section"); card.className = "exam-card asg-card"; box.append(card);
  if (!Auth.token) { card.innerHTML = `<h3>${icon("clip", 20)} Даалгавар</h3><p>Хариу илгээхийн тулд нэвтэрнэ үү.</p><a class="btn btn-gold" href="/login?next=${encodeURIComponent(location.pathname)}">Нэвтрэх</a>`; return; }
  const draw = async () => {
    let info;
    try { info = await api(`${base}/assignment`); } catch (e) { card.innerHTML = `<p class="form-error">${esc(e.message)}</p>`; return; }
    const a = info.assignment, due = info.due || { open: true }, sub = info.submission;
    card.innerHTML = `<h3>${icon("clip", 20)} Даалгавар</h3>
      <ul class="exam-facts">${due.start_at ? `<li>▶ Эхлэх: <b>${fmtDate(due.start_at)}</b></li>` : ""}${due.at ? `<li>📅 ${due.late ? `<span class="an-bad">Хугацаа дууссан</span> (${fmtDate(due.at)})` : `<b>${fmtDate(due.at)}</b> хүртэл`}</li>` : "<li>📅 Хугацаагүй</li>"}<li>🎯 ${a.grading === "rubric" ? `Рубрикаар · <b>${a.max_score}</b> оноо` : `Дээд оноо: <b>${a.max_score || 100}</b>`}</li>${due.entry_fee ? `<li>💳 Төлбөр: <b>${money(due.entry_fee)}</b>${due.paid ? " ✓" : ""}</li>` : ""}<li>✍️ Хариу: текст ба холбоос</li></ul>
      ${dueNotice(due)}
      ${sub ? `<div class="sub-mine"><b>Таны хариу</b> <small class="muted">${fmtDate(sub.submitted_at)}${sub.late ? " · хоцорсон" : ""}</small>${sub.text ? `<p>${esc(sub.text)}</p>` : ""}${sub.links?.length ? `<p>${sub.links.map((u) => `<a href="${esc(u)}" target="_blank" rel="noopener noreferrer">🔗 ${esc(u.replace(/^https?:\/\//, "").slice(0, 60))}</a>`).join("<br>")}</p>` : ""}${sub.files?.length ? `<p>${sub.files.map((f) => `<a href="${esc(f.url)}" target="_blank" rel="noopener">${icon("doc", 14)} ${esc(f.name)}</a>`).join(" · ")}</p>` : ""}
        ${sub.score !== undefined ? `<div class="sub-score">✅ Дүн: <b>${sub.score}/${a.max_score || 100}</b>${a.grading === "rubric" ? rubricViewHTML(a.rubric, sub.rubric || {}) : ""}${sub.feedback ? `<p class="sub-fb"><b>Багшийн тайлбар:</b> ${esc(sub.feedback)}</p>` : ""}</div>` : `<p class="muted small">Багш хараахан дүгнээгүй байна — дүгнэмэгц энд автоматаар гарч ирнэ.</p>`}</div>` : ""}
      ${a.grading === "rubric" && sub?.score === undefined ? `<details class="asg-rubric" open><summary>Үнэлэх шалгуур (рубрик) · ${a.max_score} оноо</summary>${rubricViewHTML(a.rubric)}</details>` : ""}
      ${due.closed || due.not_started ? "" : due.need_pay ? `<button class="btn btn-gold btn-lg" data-late-pay>💳 ${money(due.fee)} төлж даалгавраа илгээх</button>`
        : `<form class="sub-form"><label>${sub ? "Хариугаа шинэчлэх" : "Хариу"}<textarea name="text" rows="5" maxlength="20000" placeholder="Хариугаа энд бичнэ үү…">${esc(sub?.text || "")}</textarea></label>
          <label>Холбоос (Google Docs, GitHub, видео… мөр тус бүрд нэг, 5 хүртэл)<textarea name="links" rows="2" placeholder="https://…">${esc((sub?.links || []).join("\n"))}</textarea></label>
          <p class="form-error" role="alert"></p><button class="btn btn-gold">${sub ? "Дахин илгээх" : "Илгээх"}</button></form>`}`;
    bindLatePay(card, base, "Даалгаврын төлбөр", draw);
    if (!card._live) { // багш дүгнэмэгц дүн, рубрик автоматаар орж ирнэ
      card._live = (d) => { if (!card.isConnected) return Live.handlers.delete(card._live); if (d.type === "notification" && d.notification?.type === "grade" && (d.notification.link || "").endsWith("#l=" + lessonId)) draw(); };
      Live.on(card._live);
    }
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
  el.innerHTML = `<header class="exam-bar"><strong>${icon("exam", 18)} ${esc(title)}</strong><span class="exam-prog"></span><span class="exam-timer" aria-live="polite"></span><button class="btn btn-gold btn-sm" data-exam-submit>Илгээх</button></header>
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
  h2c: "https://cdnjs.cloudflare.com/ajax/libs/html2canvas/1.4.1/html2canvas.min.js", // Word/PowerPoint-ийн хүрээлсэн хэсгийг зураг болгох
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
const zoomGroupHTML = () => `<div class="fb-zoom" role="group" aria-label="Томруулах"><button type="button" class="icon-btn" data-z="-" title="Жижигрүүлэх (−)" aria-label="Жижигрүүлэх">−</button><button type="button" class="fb-zv" data-z="0" title="Анхны хэмжээ (0)">100%</button><button type="button" class="icon-btn" data-z="+" title="Томруулах (+)" aria-label="Томруулах">+</button></div>`;

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
        ${canDl ? `<button type="button" class="icon-btn" data-dv-dl title="Татах" aria-label="Татах">${icon("down", 18)}</button>` : `<span class="dv-lock" title="Зөвхөн хичээл дотор үзнэ — татах боломжгүй">${icon("lock", 16)}</span>`}
        <button type="button" class="icon-btn dv-fs" data-dv-fs title="Бүтэн дэлгэц (F)" aria-label="Бүтэн дэлгэц">${icon("expand", 18)}</button></div>
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
  $(".dv-tools", el).innerHTML = `<span class="sl-count" aria-live="polite"></span><button type="button" class="btn btn-gold btn-sm" data-sl-play>▶ Тоглуулах</button>`;
  body.innerHTML = `<div class="sl-stage" style="--r:${ratio}"><div class="sl-slide"></div>
      <button type="button" class="sl-nav sl-prev" aria-label="Өмнөх слайд">‹</button><button type="button" class="sl-nav sl-next" aria-label="Дараагийн слайд">›</button></div>
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
    page: () => cur + 1,
    go: (p) => show(p - 1, p - 1 > cur ? 1 : -1),
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
  $(".dv-tools", el).innerHTML = zoomGroupHTML() + (wide ? `<button type="button" class="btn btn-gold btn-sm" data-sl-play>▶ Тоглуулах</button>` : "");
  body.innerHTML = `<div class="fb-stage"></div><div class="fb-nav"><button type="button" class="btn btn-glass btn-sm" data-fb-prev aria-label="Өмнөх хуудас">←</button><input type="range" min="0" value="0" aria-label="Хуудас"><span class="fb-count"></span><button type="button" class="btn btn-glass btn-sm" data-fb-next aria-label="Дараагийн хуудас">→</button></div>`;
  const fb = Object.create(FlipCore);
  Object.assign(fb, { el, doc, ratio, zoom: 1, pan: { x: 0, y: 0 }, forceSingle: !!wide });
  fb.build();
  el.addEventListener("click", (e) => {
    const z = e.target.closest("[data-z]"); if (z) fb.setZoom(z.dataset.z === "+" ? fb.zoom * 1.25 : z.dataset.z === "-" ? fb.zoom / 1.25 : 1);
    if (e.target.closest("[data-sl-play]")) DocView.fullscreen(el);
  });
  let rt = 0;
  return {
    page() { return Math.min(doc.numPages, fb.single ? fb.cur + 1 : Math.max(1, fb.cur * 2)); },
    go(p) { fb.go(fb.single ? p - 1 : Math.floor(p / 2)); },
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
      page() { const top = sc.getBoundingClientRect().top + 12; let p = 1; pages.forEach((s, i) => { if (s.getBoundingClientRect().top <= top) p = i + 1; }); return p; },
      go(p) { const s = pages[p - 1]; if (s) sc.scrollTo({ top: sc.scrollTop + s.getBoundingClientRect().top - sc.getBoundingClientRect().top - 8, behavior: "smooth" }); },
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
  $(".dv-tools", el).innerHTML = `<button type="button" class="btn btn-ghost btn-sm" data-xs-reset hidden>↺ Анхны утга</button>`;
  body.innerHTML = `<div class="xs-fx"><span class="xs-ref">A1</span><span class="xs-fn">ƒx</span><span class="xs-f"></span></div>
    <div class="xs-grid"></div><div class="xs-foot"><div class="xs-tabs" role="tablist">${wb.SheetNames.map((n, i) => `<button type="button" role="tab" data-sheet="${i}">${esc(n)}</button>`).join("")}</div>
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
          ${loggedIn ? `<button class="rd-btn rd-nbtn" data-notes title="Миний тэмдэглэл — хуудас бүрт" aria-label="Миний тэмдэглэл">📝<b></b></button>` : ""}
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
    this.notes = []; this.noteColor = "y";
    if (loggedIn) api(`/api/books/${id}/notes`).then((d) => { if (this.id !== id || !this.el) return; this.notes = d.highlights || []; this.notesPaint(); }).catch(() => {});
  },
  /* Номын тэмдэглэл: одоо харж буй хуудсанд өнгө + тэмдэглэл; жагсаалтаас дарж тэр хуудас руу очно. */
  notePage() { return Math.max(1, Math.min(this.readable, this.single ? this.cur + 1 : Math.max(1, this.cur * 2))); },
  notesOpen(on) {
    let p = $(".rd-notes", this.el);
    if (!on) { p?.remove(); $("[data-notes]", this.el)?.classList.remove("on"); return; }
    if (!p) { p = document.createElement("aside"); p.className = "rd-notes"; p.setAttribute("aria-label", "Миний тэмдэглэл"); this.el.append(p); p.addEventListener("click", (e) => this.notesClick(e)); p.addEventListener("keydown", (e) => { if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) { e.preventDefault(); this.noteAdd(); } }); }
    $("[data-notes]", this.el)?.classList.add("on");
    this.notesPaint(true);
  },
  notesPaint(full) {
    if (!this.el) return;
    const n = this.notes.length, b = $("[data-notes] b", this.el); if (b) b.textContent = n ? n : "";
    const has = new Set(this.notes.map((h) => h.p));
    $$(".face[data-p]", this.el).forEach((f) => f.classList.toggle("has-note", has.has(+f.dataset.p)));
    const p = $(".rd-notes", this.el); if (!p) return;
    const pg = this.notePage(), list = [...this.notes].sort((x, y) => x.p - y.p);
    const listHTML = list.length ? list.map((h) => `<div class="rn-item ${h.p === pg ? "here" : ""}"><button type="button" class="rn-go" data-np="${h.p}"><i class="hl-dot hl-${h.c}"></i><b>${h.p}-р хуудас</b>${h.n ? `<span>${esc(h.n)}</span>` : ""}</button><button type="button" class="icon-btn rn-del" data-ndel="${esc(h.id)}" title="Устгах" aria-label="Устгах">✕</button></div>`).join("") : `<p class="muted small">Одоогоор тэмдэглэл алга. Хуудас уншингаа гол санаа, асуултаа тэмдэглээрэй.</p>`;
    if (!full && $(".rn-list", p)) { $(".rn-list", p).innerHTML = listHTML; $(".rn-pg", p).textContent = `📄 ${pg}-р хуудас`; return; }
    p.innerHTML = `<header><b>📝 Миний тэмдэглэл</b><button type="button" class="icon-btn" data-nclose aria-label="Хаах">✕</button></header>
      <div class="rn-form"><div class="rn-fh"><b class="rn-pg">📄 ${pg}-р хуудас</b><span>${HL_COLORS.map(([c, t]) => `<button type="button" class="hl-sw" data-nc="${c}" aria-pressed="${c === this.noteColor}" title="${t}"><i class="hl-dot hl-${c}"></i></button>`).join("")}</span></div>
        <textarea rows="3" maxlength="1000" placeholder="Энэ хуудсанд юу чухал байв? Өөрийн үгээр, асуулт, жишээ…"></textarea>
        <button type="button" class="btn btn-gold btn-sm" data-nadd>Энэ хуудсанд нэмэх</button></div>
      <div class="rn-list">${listHTML}</div>`;
  },
  noteAdd() {
    const p = $(".rd-notes", this.el), ta = p && $("textarea", p); if (!ta) return;
    const n = ta.value.trim(); if (!n) { ta.focus(); return toast("Тэмдэглэлээ бичнэ үү"); }
    if (this.notes.length >= 1000) return toast("1000 хүртэл тэмдэглэл хөтөлнө", true);
    const pg = this.notePage();
    this.notes.push({ id: Date.now().toString(36) + Math.random().toString(36).slice(2, 7), b: "book", s: 0, e: 0, k: "page", p: pg, c: this.noteColor, t: `${pg}-р хуудас`, n: n.slice(0, 1000) });
    ta.value = ""; this.notesPaint(); this.notesSave(); toast("📝 Тэмдэглэгдлээ");
  },
  notesSave() {
    clearTimeout(this.nsT);
    const id = this.id, list = this.notes;
    this.nsT = setTimeout(() => api(`/api/books/${id}/notes`, { method: "PUT", body: { highlights: list } }).catch((e) => toast(e.message, true)), 400);
  },
  notesClick(e) {
    const t = e.target;
    if (t.closest("[data-nclose]")) return this.notesOpen(false);
    if (t.closest("[data-nadd]")) return this.noteAdd();
    const c = t.closest("[data-nc]"); if (c) { this.noteColor = c.dataset.nc; $$("[data-nc]", this.el).forEach((x) => x.setAttribute("aria-pressed", x === c)); return; }
    const d = t.closest("[data-ndel]"); if (d) { this.notes = this.notes.filter((h) => h.id !== d.dataset.ndel); this.notesPaint(); this.notesSave(); return; }
    const g = t.closest("[data-np]"); if (g) { const p = +g.dataset.np; this.go(this.single ? p - 1 : Math.floor(p / 2)); }
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
    if (this.notes) this.notesPaint();
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
    if (t.closest("[data-notes]")) this.notesOpen(!$(".rd-notes", this.el));
  },
  onKey(e) {
    if (e.target.closest?.("textarea, input")) { if (e.key === "Escape") this.notesOpen(false); return; }
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
    if (this.nsT) { clearTimeout(this.nsT); this.nsT = 0; api(`/api/books/${this.id}/notes`, { method: "PUT", body: { highlights: this.notes || [] } }).catch(() => {}); }
    this.notes = null;
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
    const dl = $("#bkBuyDl");
    if (dl && access.download) dl.textContent = "⬇ Эх файлыг татаж авах";
    else if (dl && access.full && access.download_price > 0 && access.download_price < +dl.dataset.price) dl.textContent = `⬇ Татах эрх рүү шилжих · ${money(access.download_price)}`;
    return access;
  }).catch(() => access);
  refresh();
  const login = () => (location.href = "/login?next=" + encodeURIComponent(location.pathname));
  // download=true — татаж авах эрх (онлайн уншихыг агуулна; унших эрхтэй бол зөрүүг төлнө).
  const buy = async (download) => {
    if (!Auth.token) return login();
    if (!download && !+root.dataset.price) return login();
    try {
      const d = await api(`/api/books/${id}/buy${download ? "?mode=download" : ""}`, { method: "POST" });
      if (d.unlocked) { await refresh(); return download ? save() : read(); }
      BookReader.close?.(); // уншигч нээлттэй бол төлбөрийн цонх дээр нь гарна
      payWindow(d.order, d.payment, `«${title}» ном${download ? " · татах эрх" : ""}`, async () => { toast(download ? "🎉 Татах эрх нээгдлээ!" : "🎉 Ном нээгдлээ!"); celebrate(); await refresh(); download ? save() : read(); }, { name: "Энэ ном" });
    } catch (e) { toast(e.message, true); }
  };
  // Эх файлыг татна: нэвтрэлтийн токентой авч (холбоос тараагдахгүй), хөтчөөр хадгална.
  const save = async () => {
    const b = $("#bkBuyDl"), was = b?.textContent;
    if (b) { b.disabled = true; b.textContent = "Татаж байна…"; }
    try {
      const r = await fetch(`/api/books/${id}/download`, { headers: { Authorization: "Bearer " + Auth.token } });
      if (!r.ok) throw new Error((await r.json().catch(() => null))?.error || "Татаж чадсангүй");
      const blob = await r.blob(), m = /filename\*=UTF-8''([^;]+)/.exec(r.headers.get("Content-Disposition") || "");
      const a = document.createElement("a"); a.href = URL.createObjectURL(blob); a.download = m ? decodeURIComponent(m[1]) : title + ".pdf";
      document.body.append(a); a.click(); a.remove(); setTimeout(() => URL.revokeObjectURL(a.href), 5000);
    } catch (e) { toast(e.message, true); } finally { if (b) { b.disabled = false; b.textContent = was; } }
  };
  const read = () => BookReader.open({ id, title, pages: access.full ? pages : Math.min(access.preview_pages || 3, pages), full: access.full || pages <= (access.preview_pages || 3),
    price: +root.dataset.price, loggedIn: !!Auth.token, onBuy: +root.dataset.price ? () => buy(false) : login });
  readBtn.onclick = read;
  if (buyBtn) buyBtn.onclick = () => buy(false);
  const dlBtn = $("#bkBuyDl");
  if (dlBtn) dlBtn.onclick = () => (access.download ? save() : buy(true));
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
// Emoji сонгогчийн ангиллууд (Messenger маягийн). «Сүүлд» — хөтөчид хадгалсан сүүлийн 21.
const EMOJI_TABS = [
  ["😀", "Нүүр", "😀 😃 😄 😁 😆 😅 🤣 😂 🙂 😉 😊 😇 🥰 😍 🤩 😘 😋 😜 🤪 😎 🤗 🤭 🤫 🤔 😐 😴 😮 😲 😳 🥺 😢 😭 😤 😡 🤯 😱 😬 🙄 😏 🥳"],
  ["👍", "Гар", "👍 👎 👌 ✌️ 🤞 🤟 🤙 👋 👏 🙌 🙏 💪 ✍️ 👆 👇 👉 👈 🤝 ✋"],
  ["❤️", "Зүрх", "❤️ 🧡 💛 💚 💙 💜 🖤 🤍 💔 💕 💖 💗 💯 ✨ ⭐ 🌟 🔥 💥"],
  ["📚", "Сургууль", "📚 📖 ✏️ ✍️ 📝 📐 📏 ⚗️ 🔬 🧠 💡 🎓 🏆 🥇 ⏰ 📅 ✅ ❌ ❓ ❗ 💻 🔢"],
  ["🎉", "Баяр", "🎉 🥳 🎊 🎁 🎈 🍰 ☕ 🍎 ⚽ 🏀 🎵 🎨 🚀 🌈 🌸 🌞 🐱 🐶"],
].map(([ic, name, s]) => [ic, name, s.split(" ")]);
const emojiRecent = { get: () => { try { return JSON.parse(localStorage.getItem("sg_emoji_recent")) || []; } catch { return []; } }, add(e) { const r = [e, ...this.get().filter((x) => x !== e)].slice(0, 21); try { localStorage.setItem("sg_emoji_recent", JSON.stringify(r)); } catch {} } };
// Зөвхөн 1-3 emoji-той мессеж → бөмбөлөггүй, том (Messenger шиг).
const emojiOnly = (s) => { s = (s || "").trim(); if (!s || s.length > 40 || !/^(?:\p{Extended_Pictographic}|\p{Emoji_Modifier}|\p{Regional_Indicator}|‍|️|\s)+$/u.test(s)) return false; const n = (s.match(/\p{Extended_Pictographic}|\p{Regional_Indicator}{2}/gu) || []).length; return n >= 1 && n <= 3; };
const STICKERS = ["🎉", "👏", "🏆", "💯", "🔥", "🚀", "🧠", "📚", "✅", "🙏", "😎", "🥳", "🤝", "💪", "🌟", "🎯"];
const ICO_IMG = `<svg data-ico="image" viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linejoin="round"><rect x="3" y="4" width="18" height="16" rx="3"/><circle cx="9" cy="10" r="1.6"/><path d="m21 16-5-5-9 9"/></svg>`;
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
// Ярианы цонхонд (ChatThread.event) дамжуулах үйл явдлууд.
const THREAD_EVENTS = ["reaction", "read", "typing", "edit", "delete", "block"];
const REPORT_REASONS = [["harass", "Доромжилсон, дарамталсан"], ["inappropriate", "Зохисгүй агуулга"], ["spam", "Спам, залилан"], ["other", "Бусад"]];

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
  set(d) { this.list = d.messages || []; this.reads = d.reads || {}; this.meKey = d.me_key || this.meKey; this.replyTo = null; this.block = d.block || {}; this.render(); this.renderBlock(); this.scroll(true); this.markRead(); }
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
    if (d.type === "block") { this.block = d.block || {}; this.renderBlock(); }
    if (d.type === "read" && d.key !== this.meKey) { this.reads[d.key] = d.last_id; this.render(); }
    if (d.type === "edit") { const m = this.list.find((x) => x.id === d.message.id); if (m) { Object.assign(m, d.message); this.render(); } }
    if (d.type === "delete") { const m = this.list.find((x) => x.id === d.message_id); if (m) { m.deleted = true; m.body = ""; m.attachment_url = ""; m.reactions = {}; this.render(); } }
    if (d.type === "typing" && d.key !== this.meKey) { this.typing.set(d.key, { name: d.name, at: Date.now() }); this.render(); this.scroll(); clearTimeout(this._tt); this._tt = setTimeout(() => { this.sweepTyping(); this.render(); }, 4200); }
  }
  sweepTyping() { for (const [k, v] of this.typing) if (Date.now() - v.at > 4000) this.typing.delete(k); }
  // force: яриа нээх үед шууд доод хэсэг (smooth гүйлгэхгүй); шинэ мессежид зөөлөн.
  scroll(force) {
    const b = this.body;
    if (force) { b.style.scrollBehavior = "auto"; b.scrollTop = b.scrollHeight; b.style.scrollBehavior = ""; }
    else if (b.scrollHeight - b.scrollTop - b.clientHeight < 160) b.scrollTop = b.scrollHeight;
  }
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
      const sticker = !m.deleted && (/^::sticker::(.+)$/.exec(m.body || "") || (!m.attachment_url && emojiOnly(m.body) && [m.body, m.body.trim()]));
      const imgUrl = m.deleted ? "" : m.attachment_url || (/^https?:\/\/\S+\.(gif|png|jpe?g|webp)(\?\S*)?$/i.test(m.body || "") ? m.body : "");
      const editing = this.editing === m.id;
      const content = m.deleted ? `<span class="msg-text msg-deleted">Мессеж устгагдсан</span>` : editing ? `<form class="msg-edit"><textarea rows="2" maxlength="2000">${esc(m.body)}</textarea><div><button type="submit" class="btn btn-gold btn-sm">Хадгалах</button><button type="button" class="btn btn-ghost btn-sm" data-edit-cancel>Болих</button></div></form>` : sticker ? `<span class="msg-sticker">${esc(sticker[1])}</span>` : imgUrl ? `<a class="msg-img" href="${esc(imgUrl)}" target="_blank" rel="noopener"><img src="${esc(imgUrl)}" alt="" loading="lazy"></a>${m.attachment_url && m.body && m.body !== "📷 Зураг" ? `<span class="msg-text">${linkify(m.body)}</span>` : ""}` : `<span class="msg-text">${linkify(m.body)}</span>`;
      out.push(`<li class="msg ${mine ? "me" : "them"} ${first ? "first" : ""} ${last ? "last" : ""} ${sticker ? "is-sticker" : ""} ${imgUrl ? "has-img" : ""}" data-id="${esc(m.id)}">${av}<div class="msg-col">${name}
        <div class="msg-row"><div class="bubble" title="${fmtDate(m.created_at)}">${quote}${content}</div>
          <div class="msg-tools">${m.deleted ? "" : `<button type="button" data-react="${esc(m.id)}" title="Реакц">☺</button>`}${m.deleted ? "" : `<button type="button" data-reply="${esc(m.id)}" title="Хариулах">↩</button>`}${mine && !m.deleted ? `<button type="button" data-edit="${esc(m.id)}" title="Засах">✎</button>` : ""}${(mine || this.role() === "teacher") && !m.deleted ? `<button type="button" data-del="${esc(m.id)}" title="Устгах">🗑</button>` : ""}${!mine && !m.deleted ? `<button type="button" data-report="${esc(m.id)}" title="Мэдүүлэх" aria-label="Мэдүүлэх">⚑</button>` : ""}</div></div>
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
    const rpt = t.closest("[data-report]");
    if (rpt) { this.reportDialog(rpt.dataset.report); return; }
    if (t.closest("[data-unblock]")) {
      try { const r = await api(`/api/chat/${conv}/block`, { method: "DELETE", token: this.token() }); this.block = r.block || {}; this.renderBlock(); toast("Хоригийг цуцаллаа"); }
      catch (x) { toast(x.message, true); }
      return;
    }
    const go = t.closest("[data-goto]");
    if (go) { const el = this.ol.querySelector(`[data-id="${CSS.escape(go.dataset.goto)}"]`); if (el) { el.scrollIntoView({ block: "center", behavior: "smooth" }); el.classList.add("flash"); setTimeout(() => el.classList.remove("flash"), 1200); } return; }
    if (t.closest("[data-cancel-reply]")) { this.replyTo = null; this.renderReply(); }
  }
  async react(id, emoji) {
    try { const r = await api(`/api/chat/${this.convId()}/react`, { method: "POST", body: { message_id: id, emoji }, token: this.token() }); const m = this.list.find((x) => x.id === id); if (m) { m.reactions = r.reactions; this.render(); } }
    catch (x) { toast(x.message, true); }
  }
  // 1:1 ярианд аль нэг тал хориглосон бол бичих мөрийн оронд тайлбар (өөрөө хориглосон бол цуцлах товч).
  renderBlock() {
    if (!this.form) return;
    const me = this.role() === "teacher" ? "teacher" : "visitor", them = me === "teacher" ? "visitor" : "teacher";
    const byMe = !!this.block?.[me], byThem = !!this.block?.[them];
    let bar = this.form.parentElement?.querySelector(":scope > .chat-blocked");
    this.form.classList.toggle("is-blocked", byMe || byThem);
    if (!byMe && !byThem) { bar?.remove(); return; }
    if (!bar) { bar = document.createElement("div"); bar.className = "chat-blocked"; this.form.before(bar); }
    bar.innerHTML = byMe ? `<span>🚫 Та энэ хүнийг хориглосон — хоёулаа мессеж бичихгүй.</span><button type="button" class="btn btn-ghost btn-sm" data-unblock>Хоригоо цуцлах</button>`
      : `<span>Энэ ярианд мессеж илгээх боломжгүй.</span>`;
    const ub = bar.querySelector("[data-unblock]"); if (ub) ub.onclick = (e) => this.click({ target: e.target });
  }
  // «⚑ Мэдүүлэх»: шалтгаан, тайлбар; 1:1 ярианд мөн хориглож болно. Хянагч мэдүүлсэн хэсгийг л харна.
  reportDialog(id) {
    const m = this.list.find((x) => x.id === id); if (!m || $(".rp")) return;
    const one = !this.group(), el = document.createElement("div");
    el.className = "rp"; el.setAttribute("role", "dialog"); el.setAttribute("aria-modal", "true"); el.setAttribute("aria-labelledby", "rpT");
    el.innerHTML = `<form class="rp-card">
      <h3 id="rpT">⚑ Мессеж мэдүүлэх</h3>
      <blockquote class="rp-quote"><b>${esc(m.sender_name || (m.sender === "teacher" ? "Багш" : "Зочин"))}</b><span>${esc((m.body || "").replace(/^::sticker::/, "").slice(0, 240))}</span></blockquote>
      <fieldset class="rp-reasons"><legend>Шалтгаан</legend>${REPORT_REASONS.map(([k, t], i) => `<label><input type="radio" name="reason" value="${k}" ${i ? "" : "required"}><span>${t}</span></label>`).join("")}</fieldset>
      <label>Тайлбар <small class="muted">заавал биш</small><textarea name="note" rows="2" maxlength="500" placeholder="Юу болсныг товч бичнэ үү"></textarea></label>
      ${one ? `<label class="consent"><input type="checkbox" name="block"><span>Мөн энэ хүнийг хориглох — цаашид хоёулаа энэ ярианд бичихгүй</span></label>` : ""}
      <p class="rp-note">Мэдүүлсэн мессеж ба түүний өмнөх хэдэн мессежийг surgalt.mn-ийн хянагч харж шийдвэрлэнэ. Мэдүүлснийг тань нөгөө хүнд хэлэхгүй.</p>
      <div class="rp-act"><button type="button" class="btn btn-ghost" data-rp-x>Болих</button><button class="btn btn-gold">Мэдүүлэх</button></div>
    </form>`;
    document.body.append(el);
    const f = $("form", el), onKey = (e) => { if (e.key === "Escape") close(); };
    const close = () => { el.remove(); document.removeEventListener("keydown", onKey); };
    document.addEventListener("keydown", onKey);
    el.addEventListener("click", (e) => { if (e.target === el || e.target.closest("[data-rp-x]")) close(); });
    $("input[name=reason]", el).focus();
    f.onsubmit = async (e) => {
      e.preventDefault();
      const b = $("button.btn-gold", f); b.disabled = true;
      try {
        const r = await api(`/api/chat/${this.convId()}/report`, { method: "POST", token: this.token(), body: { message_id: id, reason: f.reason.value, note: f.note.value, block: !!f.block?.checked } });
        close(); this.block = r.block || {}; this.renderBlock();
        toast(r.duplicate ? "Энэ мессежийг өмнө нь мэдүүлсэн байна" : "Мэдүүлэг хүлээн авлаа — хянагч шалгана. Баярлалаа!");
      } catch (x) { toast(x.message, true); b.disabled = false; }
    };
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
  // Бичих мөр (Messenger шиг): [📹 Meet (багш)] [Aa … 😊] [➤ / 👍]. Хоосон үед 👍 нэг дарж илгээнэ.
  bindForm() {
    const f = this.form, extra = [...f.querySelectorAll("[data-plus]")];
    f.classList.add("composer-bar");
    f.innerHTML = `${extra.length ? `<div class="cb-left">${extra.map((b) => `<button type="button" class="cb-ic" id="${esc(b.id)}" title="${esc(b.textContent.trim())}" aria-label="${esc(b.textContent.trim())}">${b.querySelector("svg")?.outerHTML || "📹"}</button>`).join("")}</div>` : ""}
      <div class="cb-input"><textarea name="body" rows="1" maxlength="2000" placeholder="Aa" aria-label="Мессеж"></textarea>
        <button type="button" class="cb-emoji" data-emoji-open aria-label="Emoji" aria-expanded="false" title="Emoji">${ICO_SMILE}</button></div>
      <button class="cb-send" aria-label="Илгээх" title="Илгээх (Enter)"><span class="cb-like" aria-hidden="true">👍</span><svg data-ico="send" class="cb-arrow" viewBox="0 0 24 24" width="20" height="20" fill="currentColor" aria-hidden="true"><path d="M3.4 20.4 21.9 12 3.4 3.6l-.1 6.6L15 12 3.3 13.8l.1 6.6Z"/></svg></button>`;
    const ta = f.body;
    const grow = () => {
      ta.style.height = "auto"; if (!ta.scrollHeight) { ta.style.height = ""; return; } // цонх далд үед (scrollHeight 0) өндөр тогтоохгүй
      const h = Math.min(140, ta.scrollHeight); ta.style.height = h + "px";
      ta.style.overflowY = ta.scrollHeight > 140 ? "auto" : "hidden"; // жижиг үед гүйлгэх сум гарахгүй
      f.classList.toggle("has-text", !!ta.value.trim());
    };
    // Emoji сонгогч: ангилал, сүүлд хэрэглэсэн; сонгосон emoji-г курсорын байрлалд оруулна.
    const pop = () => f.querySelector(".cb-pop");
    const closePop = () => { pop()?.remove(); f.querySelector("[data-emoji-open]").setAttribute("aria-expanded", "false"); };
    const drawPop = (tab) => {
      const p = pop(); if (!p) return;
      const rec = emojiRecent.get(), tabs = [["🕘", "Сүүлд", rec], ...EMOJI_TABS].filter(([, , l], i) => i || l.length);
      const cur = tabs.find((t) => t[1] === tab) || tabs[0];
      p.innerHTML = `<div class="cb-tabs" role="tablist">${tabs.map(([ic, name]) => `<button type="button" role="tab" data-etab="${esc(name)}" aria-selected="${name === cur[1]}" title="${esc(name)}">${ic}</button>`).join("")}</div>
        <b class="cb-tab-name">${esc(cur[1])}</b><div class="cb-grid">${cur[2].map((e) => `<button type="button" data-emo="${e}" aria-label="${e}">${e}</button>`).join("")}</div>`;
    };
    const insert = (e) => {
      const s = ta.selectionStart ?? ta.value.length, en = ta.selectionEnd ?? s;
      ta.value = ta.value.slice(0, s) + e + ta.value.slice(en); ta.selectionStart = ta.selectionEnd = s + e.length;
      emojiRecent.add(e); grow(); ta.focus();
    };
    f.addEventListener("click", (e) => {
      if (e.target.closest("[data-emoji-open]")) {
        if (pop()) return closePop();
        f.insertAdjacentHTML("beforeend", `<div class="cb-pop" role="dialog" aria-label="Emoji сонгох"></div>`); drawPop(); e.target.closest("[data-emoji-open]").setAttribute("aria-expanded", "true"); return;
      }
      const tb = e.target.closest("[data-etab]"); if (tb) return drawPop(tb.dataset.etab);
      const em = e.target.closest("[data-emo]"); if (em) return insert(em.dataset.emo);
    });
    document.addEventListener("pointerdown", (e) => { if (pop() && !f.contains(e.target)) closePop(); });
    ta.addEventListener("input", () => { grow(); this.sendTyping(); });
    ta.addEventListener("keydown", (e) => {
      if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); f.requestSubmit(); }
      if (e.key === "Escape") { if (pop()) closePop(); else if (this.replyTo) { this.replyTo = null; this.renderReply(); } }
    });
    f.addEventListener("submit", async (e) => {
      e.preventDefault(); closePop();
      const text = ta.value.trim() || "👍"; // хоосон үед 👍 (Messenger шиг)
      ta.value = ""; grow(); if (!(await this.send(text)) && text !== "👍") { ta.value = text; grow(); }
    });
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
  const thread = new ChatThread({ ol: msgs, body, form, token: () => Auth.token, convId: () => conv?.id, role: () => "visitor", group: () => group });
  const scroll = () => (body.scrollTop = body.scrollHeight);
  const open = async () => {
    panel.classList.add("open"); panel.setAttribute("aria-hidden", "false"); $(".chat-fab-dot").hidden = true;
    if (conv) return;
    if (!Auth.token) { intro.hidden = false; $("a", intro)?.focus(); return; } // чат зөвхөн нэвтэрсэн хэрэглэгчид
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
      const d = await api(`/api/teachers/${teacher}/chat`, { method: "POST" });
      conv = d.conversation; group = false;
      intro.hidden = true; msgs.hidden = false; form.hidden = false;
      thread.set(d);
      Live.connect(Auth.token);
      form.body.focus();
    } catch (e) {
      if (e.status === 401) { intro.hidden = false; return; }
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
  Live.on((d) => {
    if (!conv) return;
    if (THREAD_EVENTS.includes(d.type)) { thread.event(d); return; }
    if (d.type !== "message" || d.message.conversation_id !== conv.id) return;
    thread.append(d.message);
    if (!panel.classList.contains("open")) $(".chat-fab-dot").hidden = false;
  });
  // #chat холбоосоор (нүүр хуудасны "Чат" жагсаалтаас) ирсэн бол шууд нээнэ.
  if (location.hash === "#chat=group" && courseId) { history.replaceState(null, "", location.pathname + location.search); window.railStartGroup ? window.railStartGroup(courseId).catch((e) => toast(e.message, true)) : openGroup(); }
  else if (location.hash === "#chat") { history.replaceState(null, "", location.pathname + location.search); window.railStartDirect ? window.railStartDirect(teacher).catch((e) => toast(e.message, true)) : open(); }
  // Өмнө нь чатласан бол шинэ хариуг сонсохын тулд шууд холбогдоно.
  if (Auth.token && store.get("sg_chat:" + teacher)) start();
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

// «Идэвхтэй байсан · 12 минутын өмнө» — сүүлд сайтад байсан мөч (профайл, чатын гарчиг).
function seenAgo(t) { const m = Math.round((Date.now() - new Date(t)) / 60000); return m < 1 ? "саяхан" : m < 60 ? m + " минутын өмнө" : m < 1440 ? Math.floor(m / 60) + " цагийн өмнө" : "өчигдөр"; }

// Багшийн онлайн төлөв: одоо сайтад байвал аватар дээр ногоон цэг + «Онлайн», эс бөгөөс сүүлийн 24 цагт
// идэвхтэй байсан бол «Идэвхтэй байсан · 12 мин өмнө». Таб харагдаж байх үед 45 секунд тутам шинэчилнэ.
function presence(username) {
  const pill = $("#pfLive"), dot = $("#pfPresence");
  if (!pill || !username) return;
  let busy = false;
  const tick = async () => {
    if (document.hidden || busy) return;
    busy = true;
    try {
      const d = await api(`/api/teachers/${encodeURIComponent(username)}/presence`);
      const recent = !d.online && d.last_seen && Date.now() - new Date(d.last_seen) < 864e5;
      pill.classList.toggle("on", !!d.online); dot.hidden = !d.online; pill.hidden = !d.online && !recent;
      pill.innerHTML = d.online ? `<i></i><b>Онлайн</b><span>· одоо сайтад байна</span>` : recent ? `<i></i><span>Идэвхтэй байсан · ${seenAgo(d.last_seen)}</span>` : "";
    } catch {} finally { busy = false; }
  };
  setTimeout(tick, 1200); // эзэмшигч өөрөө: WebSocket холбогдсоны дараа
  setInterval(tick, 45000);
  document.addEventListener("visibilitychange", tick);
}

function profilePage() {
  const root = $(".pf");
  presence(root.dataset.teacher);
  $$("[data-open-qr]").forEach((b) => b.addEventListener("click", () => openModal($("#qrModal"))));
  $$("[data-copy-link]").forEach((b) => b.addEventListener("click", async () => {
    try { await navigator.clipboard.writeText(location.origin + location.pathname); toast("Холбоос хуулагдлаа ✓"); } catch { toast(location.href); }
  }));
  $$("[data-share]").forEach((b) => b.addEventListener("click", () => share(b.dataset.title)));
  // Facebook-т хуваалцах: компьютер дээр жижиг цонхоор (профайлын загварлаг карт пост болно), утсан дээр апп/таб.
  $$("[data-fb-share]").forEach((a) => a.addEventListener("click", (e) => {
    if (!matchMedia("(pointer:fine) and (min-width:700px)").matches) return;
    const w = 620, h = 640, win = window.open(a.href, "fbshare", `width=${w},height=${h},left=${Math.round((screen.width - w) / 2)},top=${Math.round((screen.height - h) / 2)}`);
    if (win) { e.preventDefault(); win.opener = null; }
  }));
  // Цаг авах: багшийн календарьт тэмдэглэсэн сул цагаас сонгож захиална (#appt холбоосоор шууд нээгдэнэ).
  const apptOpen = () => apptWindow(root.dataset.teacher, $(".pf-id h1")?.firstChild?.textContent.trim() || "Багш");
  $$("[data-appt]").forEach((b) => b.addEventListener("click", apptOpen));
  root._appt = apptOpen;
  // Утас: эхний 4 орон харагдана → дарахад бүтэн дугаар (багшид мэдэгдэнэ) → дахин дарахад tel: холбоосоор шууд залгана.
  $$("[data-phone]").forEach((a) => a.addEventListener("click", async (e) => {
    if (a.dataset.tel) return;
    e.preventDefault();
    if (a.classList.contains("loading")) return;
    a.classList.add("loading");
    let vid = ""; try { vid = localStorage.getItem("sg_vid") || ""; } catch {}
    try {
      const d = await api(`/api/teachers/${encodeURIComponent(a.dataset.phone)}/phone`, { method: "POST", body: { vid } });
      a.dataset.tel = d.tel; a.href = "tel:" + d.tel;
      $(".pf-phone-n", a).textContent = d.display; $("small", a).textContent = "Залгах";
      a.classList.add("revealed"); a.setAttribute("aria-label", d.display + " дугаар руу залгах");
    } catch (x) { toast(x.message, true); }
    a.classList.remove("loading");
  }));
  localTimes();

  /* Таб: агуулгыг сольж, доогуур зураас нь гулсана. Холбоосоор (#about гэх мэт) шууд нээгдэнэ. */
  const tabs = $("#pfTabs"), ink = $(".pf-ink", tabs), tabBtns = $$(".pf-tab", tabs), panels = $$(".pf-panel");
  const moveInk = () => { const a = $(".pf-tab.active", tabs); if (a) { ink.style.width = a.offsetWidth + "px"; ink.style.transform = `translateX(${a.offsetLeft}px)`; } };
  // Эзэмшигчийн горимд зарим таб студийн удирдлагын хэсгийг (m-*) харуулна.
  const MANAGED = { overview: "overview", courses: "courses", books: "books", live: "live", files: "files", students: "students", wallet: "wallet" };
  let courseArg = null, swapT = 0;
  // fx: таб дээр дарсан — хуучин хэсэг богино бүдгэрээд, шинэ хэсэг шатлан гарч ирнэ.
  const showTab = (name, scroll, fx) => {
    const btn = tabBtns.find((b) => b.dataset.tab === name);
    if (!btn || (btn.hasAttribute("data-owner-tab") && !root.classList.contains("is-owner"))) return false;
    tabBtns.forEach((b) => { const on = b.dataset.tab === name; b.classList.toggle("active", on); b.setAttribute("aria-selected", on); });
    const manage = root.classList.contains("is-owner") && MANAGED[name];
    const want = manage ? "m-" + name : name;
    const cur = panels.find((p) => !p.hidden), next = panels.find((p) => p.dataset.panel === want);
    const swap = () => {
      panels.forEach((p) => { p.hidden = p !== next; p.classList.remove("pf-out"); });
      root.classList.toggle("manage", !!manage);
      if (cur !== next) panelIn(next);
      moveInk(); reveals();
    };
    clearTimeout(swapT);
    if (fx && cur && next && cur !== next && !reduce) { panels.forEach((p) => p.classList.toggle("pf-out", p === cur)); swapT = setTimeout(swap, 140); }
    else swap();
    if (manage && window.Studio) {
      const box = $(`[data-panel="m-${name}"]`);
      if (name === "courses" && courseArg) window.Studio.mount(box, "course", courseArg);
      else window.Studio.mount(box, MANAGED[name]);
    }
    moveInk();
    $(".pf-tab.active", tabs)?.scrollIntoView({ block: "nearest", inline: "center" });
    if (scroll && root.classList.contains("pf-stuck")) scrollTo({ top: $(".pf-body").offsetTop - 60, behavior: "smooth" });
    return true;
  };
  tabs.addEventListener("click", (e) => {
    const b = e.target.closest(".pf-tab"); if (!b) return;
    if (b.dataset.tab === "chat") { window.openRailChat?.(); return; } // чат баруун самбарт нээгдэнэ
    courseArg = null; showTab(b.dataset.tab, true, true); history.replaceState(null, "", "#" + b.dataset.tab);
  });
  root.addEventListener("click", (e) => {
    const el = e.target.closest("[data-quick]"); if (!el) return;
    if (el.dataset.quick === "chat") { e.preventDefault(); window.openRailChat?.(); }
    if (el.dataset.quick === "new-course") { try { sessionStorage.setItem("sg_new_course", "1"); } catch {} if (location.hash === "#courses") fromHash(); }
  });
  $$("[data-tab-go]").forEach((a) => a.addEventListener("click", (e) => { e.preventDefault(); showTab(a.dataset.tabGo, false, true); $(".pf-body").scrollIntoView({ behavior: "smooth" }); }));
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
    if (h.has("appt")) { setTimeout(() => root._appt?.(), 0); return showTab(root.classList.contains("is-owner") ? "courses" : "live"); }
    courseArg = null;
    const key = [...h.keys()][0] || "";
    const owner = root.classList.contains("is-owner");
    if (owner && ["profile", "settings", "edit", "about"].includes(key)) { // тусдаа «Тохиргоо» байхгүй — «Профайл засах» цонх нээгдэнэ
      const tab = h.get("edit") || "info";
      setTimeout(() => root._openEdit?.(tab), 0);
      history.replaceState(null, "", location.pathname + location.search);
      return showTab("overview") || showTab("courses");
    }
    return owner && key === "qr" ? showTab("overview") : showTab(key);
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
  meHome().then((h) => { // бүтэн: эзэмшигчийн хэрэгсэлд h.teacher хэрэгтэй
    const mine = new Map(h.courses.map((c) => [c.course.id, c]));
    cards.forEach((card) => {
      const m = mine.get(card.dataset.course); if (!m || $(".flag-lock", card)) return; // нууц үгтэй: элссэн ч нууц үгээр л нээгдэнэ
      $(".course-flags", card)?.insertAdjacentHTML("afterbegin", `<span class="flag flag-own">✓ ${m.access === "lessons" ? m.owned_lessons + " хичээл таных" : "Таных"}</span>`);
      const price = $(".price", card); if (price) { price.textContent = "▶ Үргэлжлүүлэх"; price.classList.add("free"); }
    });
    if (h.teacher && h.user.username === root.dataset.teacher) profileOwner(root, h);
  }).catch(() => {});
}

const nameInitials = (n) => (n || "?").trim().split(/\s+/).map((w) => w[0]).join("").slice(0, 2).toUpperCase();

/* Нэрийг шууд (inline) засах: нэр эсвэл ✎ дээр дарахад тэр дор нь засварлана — Enter/гадна дарах — хадгална,
   Esc — болино. Зөвхөн нэр солигдоно (PUT /api/me/name). can() худал бол (ж: «зочны нүдээр») идэвхгүй. */
function inlineName(el, done, can = () => true) {
  if (!el || el._inl) return;
  el._inl = true;
  const host = el.closest("h1,h2") || el.parentElement;
  el.classList.add("inl-name"); el.tabIndex = 0; el.setAttribute("role", "button");
  el.setAttribute("aria-label", "Нэр: " + el.textContent.trim() + " — засахын тулд дарна уу"); el.title = "Нэрээ засах";
  host.insertAdjacentHTML("beforeend", `<button type="button" class="inl-pen" aria-label="Нэрээ засах" title="Нэрээ засах">${icon("edit", 15)}</button>`);
  const pen = host.lastElementChild;
  let orig = "", hint = null;
  const start = () => {
    if (!can() || el.isContentEditable || el.classList.contains("saving")) return;
    orig = el.textContent;
    el.contentEditable = "plaintext-only"; if (el.contentEditable !== "plaintext-only") el.contentEditable = "true";
    el.classList.add("editing"); el.removeAttribute("role"); el.spellcheck = false;
    el.focus();
    const r = document.createRange(); r.selectNodeContents(el); const sel = getSelection(); sel.removeAllRanges(); sel.addRange(r);
    hint = document.createElement("small"); hint.className = "inl-hint"; hint.innerHTML = "<kbd>Enter</kbd> хадгалах · <kbd>Esc</kbd> болих";
    host.after(hint);
  };
  const finish = async (save) => {
    if (!el.isContentEditable) return;
    const v = el.textContent.replace(/\s+/g, " ").trim();
    el.contentEditable = "false"; el.classList.remove("editing"); el.setAttribute("role", "button"); hint?.remove(); hint = null;
    if (!save || v === orig.trim()) { el.textContent = orig; return; }
    if (!v || [...v].length > 80) { el.textContent = orig; toast("Нэр 1–80 тэмдэгт байна", true); return; }
    el.textContent = v; el.classList.add("saving");
    try {
      const me = await api("/api/me/name", { method: "PUT", body: { display_name: v } });
      Auth.set(Auth.token, me); el.textContent = me.display_name;
      el.setAttribute("aria-label", "Нэр: " + me.display_name + " — засахын тулд дарна уу");
      el.classList.remove("saved"); void el.offsetWidth; el.classList.add("saved");
      toast("✓ Нэр шинэчлэгдлээ"); done?.(me, orig.trim());
    } catch (x) { el.textContent = orig; toast(x.message, true); }
    finally { el.classList.remove("saving"); }
  };
  el.addEventListener("click", start);
  pen.addEventListener("click", (e) => { e.preventDefault(); start(); });
  el.addEventListener("keydown", (e) => {
    if (!el.isContentEditable) { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); start(); } return; }
    if (e.key === "Enter") { e.preventDefault(); finish(true); }
    else if (e.key === "Escape") { e.preventDefault(); e.stopPropagation(); finish(false); }
  });
  el.addEventListener("blur", () => finish(true));
  el.addEventListener("paste", (e) => { if (!el.isContentEditable) return; e.preventDefault(); document.execCommand("insertText", false, (e.clipboardData?.getData("text/plain") || "").replace(/\s+/g, " ").slice(0, 80)); });
}

/* Эзэмшигч профайл дээрээ шууд засна: зураг солих, мэдээлэл засах, зочны нүдээр харах. */
function profileOwner(root, h) {
  const LINK_KEYS = ["website", "facebook", "instagram", "youtube"];
  const setMode = (owner) => {
    root.classList.toggle("is-owner", owner);
    $$("[data-owner]").forEach((el) => (el.hidden = !owner));
    $$("[data-visitor]").forEach((el) => (el.hidden = owner));
    $$("[data-owner-tab]").forEach((el) => (el.hidden = !owner));
    $$("[data-visitor-tab]").forEach((el) => (el.hidden = owner)); // эзэмшигчид: Тухай → «Профайл засах», QR → нүүр зураг дээр
    $$(".nav a[href='/me']").forEach((el) => (el.hidden = owner)); // энэ хуудас өөрөө тул давхардуулахгүй
    $("#chatFab").hidden = owner; // өөртэйгөө чатлахгүй
    $("#viewAsBar").hidden = owner;
    $(".pf-ink").style.width = "0"; requestAnimationFrame(() => dispatchEvent(new Event("resize")));
  };
  setMode(true);
  // Нэрээ профайл дээрээ шууд засна (зочны нүдээр харах үед идэвхгүй).
  inlineName($("#pfName"), (me, old) => {
    $$(".pf-mini strong").forEach((s) => (s.textContent = me.display_name));
    document.title = document.title.replace(old, me.display_name);
    if (!me.avatar_url) $$("#pfAvatar, .pf-mini .avatar").forEach((a) => { if (!a.querySelector("img")) a.textContent = nameInitials(me.display_name); });
  }, () => root.classList.contains("is-owner"));
  $$("[data-view-as]").forEach((b) => b.addEventListener("click", () => { setMode(!root.classList.contains("is-owner")); root._fromHash() || root._showTab("courses"); }));
  // Студийн хэсгүүд (тойм, сургалт, файл, шууд хичээл, тохиргоо, чат) энэ хуудсан дээр ачаалагдана.
  $$("[data-owner-tab]").forEach((b) => (b.hidden = false));
  if (!window.Studio) {
    const ver = new URL($("script[src*='app.js']").src).searchParams.get("v");
    const sc = document.createElement("script"); sc.src = "/static/studio.js?v=" + ver; sc.defer = true;
    addEventListener("studio-ready", () => root._fromHash() || root._showTab("courses"), { once: true });
    document.body.append(sc);
  } else root._fromHash() || root._showTab("courses");

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

  // «Профайл засах»: мэдээлэл, профайлын холбоос, зөвлөмж — бүгд нэг цонхонд (тусдаа «Тохиргоо» хэсэг байхгүй).
  const modal = $("#editModal"), form = $("#editForm"), err = $("#editErr");
  const emTab = (name) => {
    $$(".em-tab", modal).forEach((t) => { const on = t.dataset.em === name; t.classList.toggle("on", on); t.setAttribute("aria-selected", String(on)); });
    $$("[data-em-pane]", modal).forEach((p) => (p.hidden = p.dataset.emPane !== name));
  };
  $$(".em-tab", modal).forEach((t) => t.addEventListener("click", () => emTab(t.dataset.em)));
  const subjects = () => form.subjects.value.split(/[,،\n]/).map((x) => x.trim()).filter(Boolean);
  const preview = () => ($("#subjPreview").innerHTML = subjects().slice(0, 8).map((x) => `<span class="tag">${esc(x)}</span>`).join(""));
  const count = () => { const n = [...form.bio.value.trim()].length, c = $("#bioCount"); c.textContent = n >= 120 ? `${n} тэмдэгт ✓` : `${n}/120 — дор хаяж 120 тэмдэгт бичвэл сайн`; c.classList.toggle("ok", n >= 120); };
  form.subjects.addEventListener("input", preview); form.bio.addEventListener("input", count);
  // Профайлын холбоос: бичих зуур чөлөөтэй эсэхийг шалгана; солимогц шинэ холбоосоор дахин нээгдэнэ.
  const un = $("#unForm"), unIn = un.username, unBtn = $("button.btn-gold", un), unHint = $("#unHint");
  let unCur = "", unTimer = 0, unSeq = 0;
  $("#unHost").textContent = location.host + "/t/";
  const say = (text, tone = "") => { unHint.textContent = text; unHint.className = "link-hint " + tone; };
  const unReset = (name) => { unCur = name; unIn.value = name; unBtn.disabled = true; say("Одоогийн холбоос."); };
  unIn.addEventListener("input", () => {
    const v = (unIn.value = unIn.value.toLowerCase().replace(/\s+/g, "_"));
    clearTimeout(unTimer); unBtn.disabled = true;
    if (v === unCur) return say("Одоогийн холбоос.");
    if (!/^[a-z0-9_]{3,32}$/.test(v)) return say("3–32 тэмдэгт: зөвхөн a–z, 0–9, доогуур зураас.", v ? "bad" : "");
    say("Шалгаж байна…");
    const my = ++unSeq;
    unTimer = setTimeout(async () => { // багшийн нэртэй давхцвал энд; суралцагчийнхтай давхцлыг сервер хадгалахдаа шалгана
      const res = await api("/api/teachers/" + encodeURIComponent(v), { raw: true, token: null }).catch(() => null);
      if (my !== unSeq) return;
      if (res?.status === 200) return say("Энэ холбоос эзэнтэй байна.", "bad");
      say(`Чөлөөтэй: ${location.host}/t/${v}`, "ok"); unBtn.disabled = false;
    }, 350);
  });
  $("#unCopy").onclick = () => { const url = `${location.origin}/t/${unCur}`; navigator.clipboard.writeText(url).then(() => toast("Хуулагдлаа ✓"), () => toast(url)); };
  un.addEventListener("submit", async (e) => {
    e.preventDefault();
    const v = unIn.value.trim();
    if (unBtn.disabled || v === unCur) return;
    if (!confirm(`Холбоосыг /t/${v} болгох уу?\n\nХуучин холбоос (/t/${unCur}) болон өмнө нь хэвлэсэн QR код ажиллахгүй болно.`)) return;
    unBtn.disabled = true;
    try {
      const d = await api("/api/me/username", { method: "PUT", body: { username: v } });
      Auth.set(d.token, d.user); Live.connect?.(d.token);
      toast("Профайлын холбоос солигдлоо — шинэ холбоосоор нээгдэнэ");
      setTimeout(() => location.replace("/t/" + d.user.username + "#edit=link"), 900);
    } catch (x) { say(x.message, "bad"); toast(x.message, true); unBtn.disabled = false; }
  });
  // Зөвлөмж: дутуу алхмыг нэг товшилтоор гүйцээнэ (талбар руу очих, зураг оруулах, хэсэг рүү шилжих).
  const FIELD = { bio: "bio", headline: "headline", subjects: "subjects", location: "location", links: "link_website", phone: "phone" };
  const loadTips = async () => {
    try {
      const ins = await api("/api/me/profile/insights"), todo = ins.tips.filter((t) => !t.done);
      $("#emRing").innerHTML = ringHTML(ins.score, "ring-sm");
      $("#emLevel").textContent = `Профайлын бүрдэл · ${ins.level}${todo.length ? ` · ${todo.length} алхам дутуу` : " · бүрэн бүрдсэн 🎉"}`;
      const n = $("#emTipN"); n.hidden = !todo.length; n.textContent = todo.length;
      $("#emTips").innerHTML = [...todo, ...ins.tips.filter((t) => t.done)].map((t) => `<li><button type="button" class="em-tip ${t.done ? "done" : ""}" data-tip="${esc(t.key)}" data-link="${esc(t.link)}" ${t.done ? "disabled" : ""}>
        <i aria-hidden="true">✓</i><span><strong>${esc(t.title)}</strong><small>${esc(t.hint)}</small></span>${t.done ? "" : `<em>Хийх ${icon("next", 14)}</em>`}</button></li>`).join("");
    } catch {}
  };
  $("#emTips").addEventListener("click", (e) => {
    const b = e.target.closest(".em-tip"); if (!b || b.disabled) return;
    const key = b.dataset.tip;
    if (key === "avatar" || key === "cover") { emTab("info"); $(`.em-photos [data-upload="${key}"]`).click(); return; }
    if (FIELD[key]) { emTab("info"); const el = form[FIELD[key]]; el?.focus(); el?.scrollIntoView({ block: "center", behavior: "smooth" }); return; }
    closeModal(modal); location.hash = (b.dataset.link || "#courses").replace(/^#/, ""); // сургалт, шууд хичээл зэрэг хэсэг рүү
  });
  const openEdit = async (tab = "info") => {
    err.textContent = "";
    try {
      const u = await api("/api/me");
      form.display_name.value = u.display_name; form.headline.value = u.headline || ""; form.bio.value = u.bio || "";
      form.subjects.value = (u.subjects || []).join(", "); form.location.value = u.location || "";
      form.phone.value = (u.phone || "").replace(/^(\d{4})(\d{4})$/, "$1 $2");
      LINK_KEYS.forEach((k) => (form["link_" + k].value = (u.links || {})[k] || ""));
      preview(); count(); unReset(u.username);
      emTab(["info", "link", "tips"].includes(tab) ? tab : "info"); openModal(modal); loadTips();
      if (tab === "link") unIn.focus(); else if (tab === "info") form.display_name.focus();
    } catch (e) { toast(e.message, true); }
  };
  root._openEdit = openEdit;
  $$("[data-edit-profile]").forEach((b) => b.addEventListener("click", () => openEdit()));
  form.addEventListener("submit", async (e) => {
    e.preventDefault(); err.textContent = "";
    const btn = $("button.btn-gold", form); btn.disabled = true;
    try {
      await save({ display_name: form.display_name.value, headline: form.headline.value, bio: form.bio.value, location: form.location.value, phone: form.phone.value,
        subjects: form.subjects.value.split(/[,\n]/).map((x) => x.trim()).filter(Boolean),
        links: Object.fromEntries(LINK_KEYS.map((k) => [k, form["link_" + k].value.trim()]).filter(([, v]) => v)) });
      location.reload(); // хуудас серверээс шинээр рендерлэгдэнэ (кэш аль хэдийн цэвэрлэгдсэн)
    } catch (x) { err.textContent = x.message; btn.disabled = false; }
  });
}

/* ---------- нүүр хуудас: нэвтэрсэн хэрэглэгчийн самбар ---------- */
/* ---------- Миний тэмдэглэл → «story»: хичээл бүрийн тодруулга, видео/баримтын тэмдэглэл нүүрэнд (зөвхөн өөрт) ----------
   Цагираг бүр нэг хичээл (ном); эхнийх нь бүх тэмдэглэлээс санамсаргүй сонгосон «Өнөөдрийн давталт».
   Үзэгч: Instagram маягийн дэвсгэр, явцын мөр, дарж/шудраж шилжинэ, удаан дарвал зогсоно, анимац олон янз. */
const ST_CAT = { y: ["⭐", "Чухал"], g: ["✅", "Ойлгосон"], r: ["❓", "Ойлгоогүй"], b: ["📘", "Нэр томьёо"] };
const ST_FX = ["st-rise", "st-zoom", "st-tilt", "st-slide", "st-blur", "st-flip"];
const stTop = (notes) => { const c = { y: 0, g: 0, r: 0, b: 0 }; for (const n of notes) c[n.c] = (c[n.c] || 0) + 1; return Object.entries(c).sort((a, b) => b[1] - a[1])[0][0]; };
const stKey = (s) => s.id + ":" + s.notes.length + ":" + (s.updated_at || "");
async function noteStories(rail) {
  if (!rail || !Auth.token) return;
  let d; try { d = await api("/api/me/notes"); } catch { return; }
  const list = (d.stories || []).filter((s) => s.notes?.length);
  if (!list.length) return;
  const all = list.flatMap((s) => s.notes.map((n) => ({ ...n, _s: s })));
  const stories = [];
  if (all.length >= 3) stories.push({ id: "review-" + new Date().toDateString(), kind: "review", title: "Өнөөдрийн давталт", sub: `${all.length} тэмдэглэлээс`, notes: all.slice().sort(() => Math.random() - 0.5).slice(0, 8), url: "" });
  stories.push(...list);
  const seen = store.get("sg_story_seen") || {};
  rail.innerHTML = `<div class="st-head"><h2><span class="sec-ico">${icon("book", 17)}</span>Миний тэмдэглэл</h2><span class="muted small">Тодруулсан, тэмдэглэсэн бүхэн тань story болно — зөвхөн танд харагдана</span></div>
    <div class="st-list">${stories.map((s, i) => { const top = stTop(s.notes); return `<button type="button" class="st-ring st-c-${top} ${s.kind === "review" ? "st-review" : ""} ${seen[s.id] === stKey(s) ? "seen" : ""}" data-st="${i}" style="--i:${i}">
      <span class="st-av"><span>${s.kind === "review" ? "💡" : s.kind === "book" ? "📕" : ST_CAT[top][0]}</span></span><b>${esc(s.title)}</b><small>${s.kind === "review" ? "Шинэ" : s.notes.length + " тэмдэглэл"}</small></button>`; }).join("")}</div>`;
  rail.hidden = false;
  rail.onclick = (e) => {
    const b = e.target.closest("[data-st]"); if (!b) return;
    storyViewer(stories, +b.dataset.st, (s) => {
      const m = store.get("sg_story_seen") || {}; m[s.id] = stKey(s); store.set("sg_story_seen", m);
      rail.querySelector(`[data-st="${stories.indexOf(s)}"]`)?.classList.add("seen");
    });
  };
}
function storyViewer(stories, si, onSeen) {
  $(".stv")?.remove();
  if (!document.querySelector("link[data-bd-font]")) { const l = document.createElement("link"); l.rel = "stylesheet"; l.dataset.bdFont = 1; l.href = "https://fonts.googleapis.com/css2?family=Caveat:wght@500;700&display=swap"; document.head.append(l); }
  const el = document.createElement("div");
  el.className = "stv"; el.setAttribute("role", "dialog"); el.setAttribute("aria-modal", "true"); el.setAttribute("aria-label", "Миний тэмдэглэл");
  el.innerHTML = `<button type="button" class="stv-side stv-sl" aria-label="Өмнөх story">‹</button>
    <div class="stv-card"><div class="stv-bars"></div>
      <header class="stv-top"><span class="stv-av"></span><span class="stv-ttl"><b></b><small></small></span><button type="button" class="stv-pause" aria-label="Зогсоох">❚❚</button><button type="button" class="stv-x" aria-label="Хаах">✕</button></header>
      <div class="stv-stage" aria-live="polite"></div><footer class="stv-foot"></footer></div>
    <button type="button" class="stv-side stv-sr" aria-label="Дараагийн story">›</button>`;
  document.body.append(el); document.documentElement.classList.add("stv-on");
  const card = $(".stv-card", el), stage = $(".stv-stage", el), bars = $(".stv-bars", el), foot = $(".stv-foot", el);
  let s = null, slides = [], i = 0, t0 = 0, elapsed = 0, dur = 5000, paused = false, raf = 0, down = null, reached = 0;
  // Багшийн статистикт: хэдэн слайд хүртэл үзсэн (хичээлийн story; ном тооцохгүй).
  const logView = () => {
    if (!s || !slides.length) return;
    const lessons = [...new Set((s.kind === "review" ? s.notes.map((n) => n._s) : [s]).filter((x) => x?.kind === "lesson").map((x) => x.id))];
    if (!lessons.length) return;
    fetch("/api/me/story-view", { method: "POST", keepalive: true, headers: { Authorization: "Bearer " + Auth.token, "Content-Type": "application/json" }, body: JSON.stringify({ lessons, slides: reached + 1, total: slides.length, review: s.kind === "review" }) }).catch(() => {});
  };
  const noteURL = (src, n) => !src?.url ? "" : src.kind === "lesson" ? `${src.url}&n=${encodeURIComponent(n.id)}` : src.url;
  const ctaLabel = (src, n) => src.kind === "book" ? "Ном руу очих" : n?.k === "time" ? "Тэр мөч рүү очих" : n?.k === "page" ? "Тэр хуудас руу очих" : "Хичээл рүү очих";
  const floaters = (arr, n) => Array.from({ length: n }, (_, k) => `<i style="--x:${(k * 37) % 100}%;--d:${(k * 0.7) % 5}s;--s:${0.7 + ((k * 13) % 6) / 10}">${arr[k % arr.length]}</i>`).join("");
  const build = (st) => {
    const out = [], cnt = st.notes.length, cats = Object.entries(ST_CAT).map(([c, [e, t]]) => [c, e, t, st.notes.filter((n) => n.c === c).length]).filter((x) => x[3]);
    out.push({ cls: "stv-cover", fx: "st-zoom", dur: 3800, url: st.url, cta: st.url ? (st.kind === "book" ? "Ном руу очих" : "Хичээл рүү очих") : "",
      html: `<div class="stv-blobs"><i></i><i></i><i></i></div><div class="stv-float">${floaters(st.kind === "review" ? ["💡", "✨", "💡", "⭐"] : ["📚", "✏️", "⭐", "💡", "🖍"], 12)}</div>
        <div class="stv-emoji">${st.kind === "review" ? "💡" : st.kind === "book" ? "📕" : "📚"}</div><p class="stv-kick">${esc(st.kind === "review" ? "Өнөөдрийн давталт" : st.sub || "")}</p>
        <h2 class="stv-big">${esc(st.title)}</h2><p class="stv-meta"><b>${cnt}</b> тэмдэглэл</p>
        <div class="stv-cats">${cats.map(([c, e, t, k]) => `<span class="stv-chip stv-chip-${c}">${e} ${t} · ${k}</span>`).join("")}</div>` });
    st.notes.forEach((n, k) => {
      const cat = ST_CAT[n.c] || ST_CAT.y, src = n._s || st;
      let body;
      if (n.img) {
        body = `<figure class="stv-shot"><img src="${esc(n.img)}" alt=""><figcaption>📄 ${esc(n.t || "")}</figcaption></figure>
          ${n.n ? `<p class="stv-note"><em>✍️ Миний тэмдэглэл</em>${esc(n.n)}</p>` : ""}`;
      } else if (n.k === "time" || n.k === "page") {
        body = `<div class="stv-when stv-when-${n.k}"><span>${n.k === "time" ? "▶" : "📄"}</span><b>${esc(n.t || "")}</b></div>
          ${n.n ? `<p class="stv-note stv-note-big">${esc(n.n)}</p>` : `<p class="stv-note">${n.k === "time" ? "Энэ мөчийг тэмдэглэсэн" : "Энэ хуудсыг тэмдэглэсэн"}</p>`}`;
      } else {
        const txt = (n.t || "").length > 240 ? n.t.slice(0, 240) + "…" : n.t || "";
        body = `<blockquote class="stv-quote ${txt.length > 120 ? "long" : ""}"><mark>${txt.split(/\s+/).map((w, j) => `<span style="--d:${300 + j * 55}ms">${esc(w)}</span>`).join(" ")}</mark></blockquote>
          ${n.n ? `<p class="stv-note"><em>✍️ Миний тэмдэглэл</em>${esc(n.n)}</p>` : ""}`;
      }
      out.push({ cls: `stv-hl stv-${n.c}`, fx: ST_FX[k % ST_FX.length], dur: Math.min(12000, 4200 + ((n.t || "").length + (n.n || "").length) * 32),
        url: noteURL(src, n), cta: src.url ? ctaLabel(src, n) : "",
        html: `<div class="stv-blobs"><i></i><i></i></div><span class="stv-cat">${cat[0]} ${cat[1]}</span>${st.kind === "review" ? `<p class="stv-from">${esc(src.title)}</p>` : ""}${body}` });
    });
    out.push({ cls: "stv-outro", fx: "st-zoom", dur: 4200, confetti: true, url: st.url, cta: st.url ? (st.kind === "book" ? "Ном руу очих" : "Хичээл рүү очих") : "", again: true,
      html: `<div class="stv-blobs"><i></i><i></i><i></i></div><div class="stv-emoji">🎉</div><h2 class="stv-big">Сайн байна!</h2><p class="stv-meta"><b>${cnt}</b> тэмдэглэлээ давтлаа.<br>Тогтмол давтвал удаан санана 💡</p>` });
    return out;
  };
  const confetti = () => {
    if (reduce) return;
    const cols = ["#ffd60a", "#34c759", "#ff453a", "#0a84ff", "#ffffff", "#eaa02e"];
    for (let k = 0; k < 70; k++) {
      const p = document.createElement("i"); p.className = "stv-conf"; p.style.background = cols[k % cols.length]; stage.append(p);
      const a = -Math.PI / 2 + (Math.random() - 0.5) * 2.2, v = 180 + Math.random() * 320;
      p.animate([{ transform: "translate(0,0) rotate(0)", opacity: 1 }, { transform: `translate(${Math.cos(a) * v}px,${Math.sin(a) * v + 420}px) rotate(${Math.random() * 900}deg)`, opacity: 0 }],
        { duration: 1600 + Math.random() * 900, easing: "cubic-bezier(.15,.7,.3,1)" }).onfinish = () => p.remove();
    }
  };
  const header = () => {
    const top = stTop(s.notes);
    $(".stv-av", el).className = `stv-av st-c-${top}`; $(".stv-av", el).textContent = s.kind === "review" ? "💡" : s.kind === "book" ? "📕" : ST_CAT[top][0];
    $(".stv-ttl b", el).textContent = s.title;
    $(".stv-ttl small", el).textContent = [s.kind === "review" ? "Санамсаргүй сонголт" : s.sub, s.updated_at ? fmtDate(s.updated_at) : ""].filter(Boolean).join(" · ");
    bars.innerHTML = slides.map(() => `<i><b></b></i>`).join("");
    $(".stv-sl", el).disabled = si === 0; $(".stv-sr", el).disabled = si === stories.length - 1;
  };
  const show = () => {
    const sl = slides[i];
    stage.innerHTML = `<section class="stv-slide ${sl.cls} ${reduce ? "" : sl.fx}">${sl.html}</section>`;
    foot.innerHTML = (sl.again ? `<button type="button" class="stv-cta stv-ghost" data-again>↺ Дахин үзэх</button>` : "") + (sl.url ? `<a class="stv-cta" href="${esc(sl.url)}">${esc(sl.cta)} →</a>` : "");
    $$("b", bars).forEach((b, k) => { b.style.transition = "none"; b.style.width = k < i ? "100%" : "0%"; });
    dur = sl.dur; elapsed = 0; t0 = performance.now(); reached = Math.max(reached, i);
    if (sl.confetti) setTimeout(confetti, 250);
    if (i === slides.length - 1) onSeen?.(s);
  };
  const openStory = (k, fromEnd) => { logView(); si = k; s = stories[k]; slides = build(s); i = fromEnd ? slides.length - 1 : 0; reached = 0; header(); card.classList.remove("stv-swap"); void card.offsetWidth; card.classList.add("stv-swap"); show(); };
  const next = () => { if (i < slides.length - 1) { i++; show(); } else if (si < stories.length - 1) openStory(si + 1); else close(); };
  const prev = () => { if (i > 0) { i--; show(); } else if (si > 0) openStory(si - 1, true); else show(); };
  const setPaused = (p) => { if (p === paused) return; const now = performance.now(); if (p) elapsed += now - t0; else t0 = now; paused = p; $(".stv-pause", el).textContent = p ? "▶" : "❚❚"; card.classList.toggle("paused", p); };
  const tick = (now) => {
    if (!paused) {
      const p = Math.min(1, (elapsed + now - t0) / dur), b = $$("b", bars)[i];
      if (b) b.style.width = p * 100 + "%";
      if (p >= 1) next();
    }
    raf = requestAnimationFrame(tick);
  };
  const onKey = (e) => {
    if (e.key === "Escape") { e.stopPropagation(); close(); }
    else if (e.key === "ArrowRight") next(); else if (e.key === "ArrowLeft") prev();
    else if (e.key === " ") { e.preventDefault(); setPaused(!paused); }
  };
  function close() { logView(); s = null; cancelAnimationFrame(raf); document.removeEventListener("keydown", onKey, true); document.documentElement.classList.remove("stv-on"); el.classList.add("out"); setTimeout(() => el.remove(), 250); }
  card.addEventListener("pointerdown", (e) => {
    if (e.target.closest("a, button")) return;
    down = { x: e.clientX, y: e.clientY, t: performance.now() }; setPaused(true);
  });
  card.addEventListener("pointermove", (e) => { if (down && e.clientY - down.y > 10) card.style.transform = `translateY(${Math.max(0, e.clientY - down.y)}px) scale(${1 - Math.min(0.08, (e.clientY - down.y) / 2000)})`; });
  const up = (e) => {
    if (!down) return;
    const d = down; down = null; card.style.transform = ""; setPaused(false);
    if (e.clientY - d.y > 110) return close(); // доош шудраж хаана
    if (Math.abs(e.clientX - d.x) > 60) return e.clientX < d.x ? (si < stories.length - 1 ? openStory(si + 1) : close()) : si > 0 ? openStory(si - 1) : show();
    if (performance.now() - d.t < 260) { const r = card.getBoundingClientRect(); (e.clientX - r.left < r.width * 0.32 ? prev : next)(); }
  };
  card.addEventListener("pointerup", up); card.addEventListener("pointercancel", () => { down = null; card.style.transform = ""; setPaused(false); });
  el.addEventListener("click", (e) => {
    if (e.target === el || e.target.closest(".stv-x")) return close();
    if (e.target.closest(".stv-pause")) return setPaused(!paused);
    if (e.target.closest(".stv-sl")) return si > 0 && openStory(si - 1);
    if (e.target.closest(".stv-sr")) return si < stories.length - 1 && openStory(si + 1);
    if (e.target.closest("[data-again]")) { i = 0; show(); }
  });
  document.addEventListener("keydown", onKey, true);
  openStory(si);
  raf = requestAnimationFrame(tick);
}
async function homePage() {
  const dash = $("#dash");
  if (!dash || !Auth.token) return;
  let h;
  try { h = await meHome(); }
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
  const SEC_ICO = { "my-rank": "medal", "my-study": "chart", "my-tasks": "exam", "my-pending": "money", "my-live": "live", "my-courses": "courses", "my-lessons": "play", "my-teachers": "teacher", "my-chats": "chat" };
  const sec = (id, title, body, extra = "") => `<section class="dash-sec" id="${id}"><header><h2>${SEC_ICO[id] ? `<span class="sec-ico">${icon(SEC_ICO[id], 17)}</span>` : ""}${title}</h2>${extra}</header>${body}</section>`;
  const courseCard = (c, i) => `<a class="course-card tilt" href="/c/${esc(c.course.id)}" style="--h:${hueOfName(c.course.title)}">
    <div class="course-art"><div class="course-flags"><span class="flag flag-own">✓ ${c.access === "lessons" ? c.owned_lessons + " хичээл" : accessLabel[c.access]}</span></div><span class="course-num">${String(i + 1).padStart(2, "0")}</span><span class="glare"></span></div>
    <div class="course-body"><h3>${esc(c.course.title)}</h3>${c.teacher ? `<div class="course-by">${avatarHTML(c.teacher)}${esc(c.teacher.display_name)}</div>` : ""}
    <div class="course-meta"><span>${c.course.lesson_count} хичээл</span><span class="price free">▶ Үргэлжлүүлэх</span></div></div></a>`;

  // «Сурах» хуудас багш, суралцагч хоёрт яг ижил (багш ч бусдын сургалтад суралцана; багшлах хэсэг нь дээд цэсний «Багшлах»).
  const parts = [`<div class="dash-head">${avatarHTML(u, "avatar-md")}<div class="grow"><h1>${hello}, <span id="stuName">${esc(u.display_name)}</span></h1>
      <p>${h.courses.length ? `Танд ${h.courses.length} сургалт${h.meetings.length ? `, ${h.meetings.length} шууд хичээл` : ""} байна.` : "Багшийнхаа профайлаас анхны сургалтаа сонгоорой."}</p></div>
      <button class="btn" data-student-settings>Профайл засах</button></div>`,
    `<div class="tiles">
      <a class="tile" href="#my-courses"><em>${icon("courses")}</em><b>${h.courses.length}</b><span>Миний сургалт</span></a>
      <a class="tile" href="#my-lessons"><em>${icon("play")}</em><b>${h.lessons.length}</b><span>Авсан хичээл</span></a>
      <a class="tile" href="#my-live"><em>${icon("live")}</em><b>${h.meetings.length}</b><span>Шууд хичээл</span></a>
      <a class="tile" href="#my-chats"><em>${icon("chat")}</em><b>${h.chats.length}</b><span>Чат</span></a>
      <a class="tile" href="#my-teachers"><em>${icon("teacher")}</em><b>${h.teachers.length}</b><span>Миний багш</span></a>
      ${h.rank ? `<a class="tile tile-rank" href="#my-rank"><em class="rank-shine" data-level="${h.rank.level}">${esc(h.rank.insignia)}</em><b>${esc(h.rank.name)}</b><span>Миний цол · ${h.rank.points} оноо</span></a>` : ""}
      ${h.tasks?.length ? `<a class="tile" href="#my-tasks"><em>${icon("exam")}</em><b>${h.tasks.filter((t) => ["open", "need_pay"].includes(t.status)).length}</b><span>Хийх даалгавар, шалгалт</span></a>` : ""}
    </div>`];
  parts.splice(1, 0, `<div class="st-rail" id="stRail" hidden></div>`); // тэмдэглэлийн story (noteStories)
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
        <div class="grow"><span class="eyebrow">Нэгдсэн цол · ${rk.average && rk.integration?.courses_all > 1 ? `${rk.integration.courses_all} сургалтын цолны дундаж` : "бүх хичээлээ нэгтгэж систем олгоно"}</span><strong class="rank-title">${esc(rk.name)}</strong>
          <p class="rank-cheer">${cheer}</p>
          <div class="rank-stats"><span><b>${rk.points}</b> ${rk.average && rk.integration?.courses_all > 1 ? "дундаж оноо" : "оноо"}</span>${rk.integration?.bonus ? `<span><b>+${rk.integration.bonus}</b> интеграц</span>` : ""}<span><b>${rk.honest}</b> шударга хичээл</span>${rk.integration?.sections_all ? `<span><b>${rk.integration.sections}/${rk.integration.sections_all}</b> бүлэг бүтэн</span>` : ""}${rk.cheated ? `<span class="bad"><b>${rk.cheated}</b> тооцогдоогүй</span>` : ""}${rk.next ? `<span><b>${left}</b> оноо дараагийн цолд</span>` : ""}</div>
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

  if (h.meetings.length) parts.push(sec("my-live", "Шууд хичээл", `<ul class="items">${h.meetings.map((m) => m.booking ? `
      <li><div class="item live"><span class="item-ico">${icon("cal", 20)}</span><span class="grow"><strong>${esc(m.title)}</strong><small>${fmtDate(m.starts_at)} · ${m.duration_min} мин · ${m.mode === "offline" ? "Биечлэн · " + esc(m.location || "") : "Онлайн"}${m.price ? " · ✓ төлсөн" : ""}</small></span><span class="chip chip-amber">${until(m.starts_at)}</span>
      ${m.meet_url ? `<a class="btn btn-accent btn-sm" href="${esc(m.meet_url)}" target="_blank" rel="noopener">Нэгдэх</a>` : `<a class="btn btn-ghost btn-sm" href="/t/${esc(m.teacher_username || "")}">Багш</a>`}</div></li>` : `
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

  dash.innerHTML = parts.join("") + (`<div class="modal" id="stuModal"><div class="modal-card"><button class="icon-btn modal-x" data-close aria-label="Хаах">✕</button>
      <h3 class="h3">Профайл засах</h3><form class="form" id="stuForm"><label>Нэр<input name="display_name" required maxlength="80" value="${esc(u.display_name)}"></label>
      <p class="muted small" style="margin:0">Имэйл: ${esc(u.email || "")}</p><p class="form-error" role="alert"></p>
      <div class="hero-cta" style="margin:0;justify-content:flex-end"><button type="button" class="btn btn-ghost" data-close>Болих</button><button class="btn btn-gold">Хадгалах</button></div></form></div></div>`);
  noteStories($("#stRail"));
  $("[data-student-settings]")?.addEventListener("click", () => openModal($("#stuModal")));
  // Нэрээ мэндчилгээн дээрээ шууд засна.
  inlineName($("#stuName"), (me) => {
    const f = $("#stuForm"); if (f) f.display_name.value = me.display_name;
    const av = $(".dash-head .avatar"); if (av && !av.querySelector("img")) av.textContent = nameInitials(me.display_name);
  });
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
  const id = root.dataset.course, bundle = +root.dataset.price, mode = root.dataset.mode, buy = $("#buyBtn"), drip = root.dataset.drip === "1", locked = root.dataset.locked === "1";
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
      const avg = a.rank_total && a.rank_total.points !== rk.points ? a.rank_total : null; // бүх сургалтын цол (сургалтуудын дундаж)
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
        <span class="cr-next">${rk.next ? `Дараагийн «${esc(rk.next_name)}» цол ${rk.next} оноонд` : "Дээд цол"}${avg ? ` · бүх сургалтын дундаж: <b>${esc(avg.name)}</b> ${avg.points} оноо` : ""}</span>
        ${rk.tips?.length ? `<ul class="cr-tips">${rk.tips.map((t) => `<li>${esc(t)}</li>`).join("")}</ul>` : ""}`;
    }
  };
  const refreshAccess = () => api(`/api/courses/${id}/access`).then((a) => { applyStates(a); drawStage(); return a; }).catch(() => null);
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
  const ready = Auth.token ? refreshAccess().then((a) => {
    if (!a) return;
    if (a.all) unlockAll(); a.lessons.forEach(unlockLesson);
    if (mode === "free" && a.enrolled && !locked) buy.textContent = "✓ Та элссэн — үзэж эхлэх";
    if (locked && a.need_code && location.hash === "#unlock") askCode(); // картан дээрх 🔒-оос ирсэн
    drawStage();
  }) : Promise.resolve();
  // Багш хоригийг цуцлахад (WebSocket) хуудас шинэчлэхгүйгээр шууд нээгдэнэ; хаагдсан цонх/хичээл дээр байсан бол шууд үргэлжилнэ.
  let lazyWatch = false; // дараагийн нээх хичээлийн хяналт суралцагч хөдөлтөл хүлээнэ (play-д нэг удаа)
  if (Auth.token) Live.on(async (d) => {
    if (d.type !== "lesson_unblocked") return;
    const mine = (d.lessons || []).filter((x) => x.course_id === id).map((x) => x.lesson_id);
    if (!mine.length) return;
    const was = mine.filter((lid) => access?.blocks?.[lid]), pop = $(".drip-pop");
    await refreshAccess();
    const lid = was.find((x) => !access?.blocks?.[x]) || mine.find((x) => !access?.blocks?.[x]), li = lid && rowById(lid);
    if (!li) return;
    toast("🔓 Багш хичээлийг тань дахин нээлээ — үргэлжлүүлэн үзээрэй");
    if (pop) pop.remove();
    if (pop || !stageOn() || stageLid === lid) { // өөр хичээл үзэж байвал зөвхөн мэдэгдэнэ
      lazyWatch = true; // суралцагч дэргэд байхгүй байж болно: хөдөлтөл хяналт тоолохгүй
      await expandRow(li, true);
      lazyWatch = false;
    }
  });
  if (!Auth.token && locked && location.hash === "#unlock") setTimeout(() => askCode(), 300); // нэвтрэх рүү хөтөлнө
  if (!Auth.token) $$(".lesson.is-free [data-play]").forEach((b) => (b.textContent = "▶ Нэвтэрч үзэх")); // үнэгүй хичээлийг ч нэвтэрч үзнэ
  if (!Auth.token && drip) $$(".lesson:not(.is-free)").forEach((li) => { const lbl = $(".lesson-state", li); if (lbl && +li.dataset.unlock && li.dataset.always !== "1") { lbl.hidden = false; lbl.textContent = `⏱ Өмнөхийг үзснээс ${humanHours(li.dataset.unlock)}-ийн дараа`; } });
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

  const needLogin = (lid) => { if (Auth.token) return false; location.href = "/login?next=" + encodeURIComponent(location.pathname + (lid ? "#l=" + lid : "")); return true; };

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
    if (locked) return askCode(() => expandRow(li, true)); // нууц үгтэй сургалт
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
  // Нууц үгтэй сургалт: багшаас авсан нууц үгээ оруулж бүх хичээлийг нээнэ.
  const askCode = (then) => {
    if (needLogin()) return;
    $("#codeModal")?.remove();
    document.body.insertAdjacentHTML("beforeend", `<div class="modal" id="codeModal"><div class="modal-card code-card"><button class="icon-btn modal-x" data-close aria-label="Хаах">✕</button>
      <div class="code-ico" aria-hidden="true">🔑</div><h3 class="h3">Нууц үгтэй сургалт</h3>
      <p class="muted">Багшаасаа авсан нууц үгээ оруулаад сургалтын бүх хичээлийг нээнэ.</p>
      <form class="form" id="codeForm" autocomplete="off"><input name="code" required minlength="4" maxlength="64" placeholder="Нууц үг" aria-label="Нууц үг" autocomplete="off" autocapitalize="off" spellcheck="false">
        <p class="form-error" role="alert"></p><button class="btn btn-gold btn-block">🔓 Нээх</button></form></div></div>`);
    const m = $("#codeModal"), f = $("#codeForm", m);
    const done = () => { closeModal(m); setTimeout(() => m.remove(), 300); };
    openModal(m); setTimeout(() => f.code.focus(), 60);
    m.addEventListener("click", (e) => { if (e.target === m || e.target.closest("[data-close]")) done(); });
    f.onsubmit = async (e) => {
      e.preventDefault();
      const btn = $("button", f), err = $(".form-error", f); err.textContent = ""; btn.disabled = true;
      try {
        await api(`/api/courses/${id}/unlock`, { method: "POST", body: { code: f.code.value } });
        done(); unlockAll(); toast("🔓 Сургалт нээгдлээ — амжилт хүсье!"); celebrate(); await refreshAccess(); then?.();
      } catch (x) { err.textContent = x.message; f.code.select(); } finally { btn.disabled = false; }
    };
  };
  const isLocked = (li) => li.classList.contains("is-locked") && !li.classList.contains("unlocked");


  buy.addEventListener("click", async () => {
    if (locked && !all) return askCode();
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
  let stopWatch = null, stopVideos = null, stopMarker = null, reflectDone = false, curLesson = null;
  new MutationObserver(() => {
    const lm = $("#lessonModal");
    if (lm.classList.contains("open")) { lm.dataset.wasOpen = "1"; return; }
    if (!lm.dataset.wasOpen) return; // "inline" нэмэх зэрэг бусад өөрчлөлт — хаагдаагүй
    delete lm.dataset.wasOpen;
    if (lm.classList.contains("inline")) setTimeout(() => { if (!lm.classList.contains("open")) (stageOn() ? exitStage() : collapseAll()); }, 0);
    stopWatch?.(); stopWatch = null;
    stopVideos?.(); stopVideos = null;
    stopMarker?.(); stopMarker = null;
    if (curLesson && Auth.token && !reflectDone) askReflection(curLesson.id, curLesson.lid, curLesson.title);
    curLesson = null;
  }).observe($("#lessonModal"), { attributes: true, attributeFilter: ["class"] });
  // Хичээлийг үзэж дууссаны дараа "юу сурсан бэ?" гэж товч дүгнэлт асууна — идэвхтэй суралцсаны нотолгоо.
  const askReflection = (cid, lid, title) => {
    try { if (sessionStorage.getItem("sg_refl_" + lid)) return; } catch {}
    const key = "sg_reflect_at_" + lid;
    let last = 0; try { last = +localStorage.getItem(key) || 0; } catch {}
    if (Date.now() - last < 3600e3) return; // цагт нэг удаа л зовоохгүй
    // Хяналт хичээлийг зогсоосон бол эхлээд тэр сануулгыг уншиж хаасны дараа асууна (дээр нь давхарлахгүй;
    // дараагийн сануулга watchAlert бичиж буй дүгнэлтийг устгахгүйн тулд).
    const busy = $(".watch-alert");
    if (busy) { new MutationObserver((_, mo) => { if (!busy.isConnected) { mo.disconnect(); askReflection(cid, lid, title); } }).observe(document.documentElement, { childList: true, subtree: true }); return; }
    const el = document.createElement("div"); el.className = "watch-alert"; el.innerHTML = `<div class="watch-card refl-card">
      <span class="watch-ico">✍️</span><h3>«${esc(title)}» хичээлээс юу сурсан бэ?</h3>
      <p class="muted small">2-3 өгүүлбэрээр бичихэд хангалттай. Энэ нь идэвхтэй суралцсаны тэмдэглэгээ болно.</p>
      <textarea rows="3" maxlength="2000" placeholder="Жишээ нь: Энэ хичээлээр … гэдгийг ойлголоо…"></textarea>
      <p class="form-error" role="alert"></p>
      <div class="refl-acts"><button class="btn btn-ghost btn-sm" data-skip>Алгасах</button><button class="btn btn-gold" data-send>Илгээх</button></div></div>`;
    overlayHost(true).append(el); $("textarea", el).focus();
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
  const expandRow = async (row, opts) => {
    if (needLogin(row.dataset.lesson)) return; // хичээл (үнэгүйг ч) нэвтэрсэн хэрэглэгч л үзнэ
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
    await openStage(row, opts && typeof opts === "object" ? opts : {});
  };

  /* ---- Хичээл үзэх горим (олон улсын цахим сургалтын загвар): өргөн агуулга + хөтөлбөрийн самбар ----
     Том дэлгэц: хөтөлбөр баруун талд (хумьж болно). Таблет, утас: «Хичээлүүд» товчоор гулсаж гарна. */
  const stage = $("#lessonStage");
  let stageLid = null, pushed = false, railWas = true;
  const rowById = (lid) => $(`.lesson[data-lesson="${CSS.escape(lid)}"]`);
  const wideLP = () => matchMedia("(min-width:1100px)").matches;
  function stageOn() { return !!stage && !stage.hidden; }
  const factsHTML = (lid) => {
    if (!Auth.token) return "";
    const p = access?.progress?.[lid], rk = access?.ranks?.[lid];
    return [p?.completed_at ? `<span class="st-chip ok">✓ Дууссан · ${fmtDate(p.completed_at)}</span>` : p?.viewed_at ? `<span class="st-chip">Үзэж эхэлсэн · ${fmtDate(p.viewed_at)}</span>` : `<span class="st-chip">Шинэ хичээл</span>`,
      rk ? `<span class="st-chip ${rk.disqualified ? "bad" : rk.points >= 75 ? "ok" : ""}" title="${esc((rk.reasons || []).join(", "))}">${rk.disqualified ? "⛔ Оноо тооцогдоогүй" : `+${rk.points} оноо нэгдсэн цолд`}</span>` : ""].join("");
  };
  const tocItem = (r) => {
    const lid = r.dataset.lesson, cur = lid === stageLid;
    const [k, ic, note] = access?.blocks?.[lid] ? ["blocked", "⛔", "Хаагдсан"] : isLocked(r) ? ["locked", "🔒", +r.dataset.price > 0 ? money(+r.dataset.price) : "Багцаар нээгдэнэ"]
      : r.classList.contains("is-drip") ? ["drip", "⏳", $(".lesson-state", r)?.textContent || "Түгжээтэй"] : r.classList.contains("is-done") ? ["done", "✓", ""] : ["open", "", ""];
    return `<li><button type="button" class="lp-li ${k} ${cur ? "cur" : ""}" data-lp-go="${esc(lid)}"${cur ? ' aria-current="true"' : ""}><span class="lp-ic" aria-hidden="true">${cur ? "▶" : ic}</span><span class="lp-lt"><b>${esc($(".lesson-num", r)?.textContent || "")}</b>${esc(rowTitle(r))}${note ? `<small>${esc(note)}</small>` : ""}</span></button></li>`;
  };
  function drawStage() {
    if (!stageOn()) return;
    const r = stageLid && rowById(stageLid), rows = lessonRows(), doneN = rows.filter((x) => x.classList.contains("is-done")).length;
    $("#lpTitle").textContent = r ? `${$(".lesson-num", r)?.textContent || ""} · ${rowTitle(r)}` : "";
    const pg = $("#lpProg"), tot = rows.length;
    pg.hidden = !Auth.token || !tot;
    if (!pg.hidden) { $("i", pg).style.width = Math.round(doneN / tot * 100) + "%"; $("small", pg).textContent = `${doneN}/${tot} дууссан`; }
    const groups = $$(".lesson-group");
    const list = groups.length ? groups.map((g) => { const rs = $$(".lesson[data-lesson]", g), d = rs.filter((x) => x.classList.contains("is-done")).length;
      return `<details class="lp-sec" open><summary><span>${esc(g.dataset.section || "Бусад хичээлүүд")}</span><small>${d}/${rs.length}</small></summary><ol>${rs.map(tocItem).join("")}</ol></details>`; }).join("")
      : `<ol>${rows.map(tocItem).join("")}</ol>`;
    const toc = $("#lpToc"), keep = $(".lp-toc-body", toc)?.scrollTop;
    toc.innerHTML = `<div class="lp-toc-head"><b>Сургалтын хөтөлбөр</b><small>${doneN}/${tot} дууссан</small><button type="button" class="icon-btn lp-toc-x" data-lp-toc-close aria-label="Хаах">✕</button></div><div class="lp-toc-body">${list}</div>`;
    const body = $(".lp-toc-body", toc), cur = $(".lp-li.cur", body);
    if (keep != null) body.scrollTop = keep; else if (cur) body.scrollTop = Math.max(0, cur.offsetTop - body.clientHeight / 3);
    $("#lessonFacts").innerHTML = stageLid ? factsHTML(stageLid) : "";
    stage.style.setProperty("--nav-h", ($(".nav")?.offsetHeight || 64) + "px");
    stage.style.setProperty("--lp-top-h", ($(".lp-top", stage)?.offsetHeight || 60) + "px");
    syncTocBtn();
  }
  const syncTocBtn = () => { const b = $("[data-lp-toc]", stage); if (b) b.setAttribute("aria-expanded", String(wideLP() ? !stage.classList.contains("toc-hidden") : stage.classList.contains("toc-open"))); };
  const closeToc = () => { stage?.classList.remove("toc-open"); syncTocBtn(); };
  const toggleToc = () => {
    if (wideLP()) { const hide = !stage.classList.contains("toc-hidden"); stage.classList.toggle("toc-hidden", hide); try { localStorage.setItem("sg_lp_toc", hide ? "0" : "1"); } catch {} }
    else stage.classList.toggle("toc-open");
    syncTocBtn();
  };
  async function openStage(row, { push = true } = {}) {
    const lid = row.dataset.lesson, m = lessonModalEl(), first = !stageOn();
    if (!first && lid === stageLid && m.classList.contains("open")) { closeToc(); return; }
    $(".drip-pop")?.remove();
    if (first) {
      railWas = document.documentElement.classList.contains("rail-collapsed");
      document.documentElement.classList.add("lp-focus", "rail-collapsed"); // өргөн харагдахын тулд чатыг түр хумина
      try { stage.classList.toggle("toc-hidden", localStorage.getItem("sg_lp_toc") === "0"); } catch {}
      root.classList.add("lp-on"); stage.hidden = false;
    } else if (m.classList.contains("open")) { delete m.dataset.wasOpen; closeModal(m); } // өөр хичээл рүү: өмнөхийн хяналт play()-д зогсоно
    stageLid = lid;
    $$(".lesson.open").forEach((r) => r.classList.remove("open")); row.classList.add("open");
    if (m.parentElement !== $("#lpMain")) { m.classList.add("inline"); $("#lpMain").replaceChildren(m); }
    const url = location.pathname + location.search + "#l=" + lid;
    if (first && push) { history.pushState({ lp: lid }, "", url); pushed = true; } else history.replaceState({ lp: lid }, "", url);
    closeToc(); drawStage(); scrollTo({ top: 0 });
    try { await play(lid); } catch (err) { toast(err.message, true); if (first) exitStage(); return; }
    drawStage();
  }
  function exitStage({ fromPop } = {}) {
    if (!stageOn()) return;
    const m = lessonModalEl(), lid = stageLid;
    stage.hidden = true; root.classList.remove("lp-on"); closeToc(); stageLid = null;
    document.documentElement.classList.remove("lp-focus"); if (!railWas) document.documentElement.classList.remove("rail-collapsed");
    if (m.classList.contains("open")) closeModal(m); // ажиглагч: хяналт зогсоож, дүгнэлт асууна
    dockModal(); $$(".lesson.open").forEach((r) => r.classList.remove("open"));
    if (!fromPop) { if (pushed) history.back(); else history.replaceState(null, "", location.pathname + location.search); }
    pushed = false;
    const r = lid && rowById(lid);
    if (r) { r.scrollIntoView({ block: "center" }); r.classList.add("lp-was"); setTimeout(() => r.classList.remove("lp-was"), 1800); }
  }
  stage?.addEventListener("click", (e) => {
    const go = e.target.closest("[data-lp-go]");
    if (go) { closeToc(); const r = rowById(go.dataset.lpGo); if (r) expandRow(r); return; }
    if (e.target.closest("[data-lp-exit]")) return exitStage();
    if (e.target.closest("[data-lp-toc-close]")) return closeToc();
    if (e.target.closest("[data-lp-toc]")) return toggleToc();
  });
  addEventListener("keydown", (e) => { if (e.key === "Escape" && stage?.classList.contains("toc-open")) closeToc(); });
  addEventListener("resize", () => { if (stageOn()) { if (wideLP()) closeToc(); syncTocBtn(); } });
  addEventListener("popstate", () => { // хөтчийн «буцах»: хөтөлбөр рүү эсвэл өмнөх хичээл рүү
    const lid = new URLSearchParams(location.hash.slice(1)).get("l");
    if (!lid) { pushed = false; exitStage({ fromPop: true }); return; }
    const r = rowById(lid); if (r && lid !== stageLid) expandRow(r, { push: false });
  });
  const play = async (lid) => {
    // Тодруулга, хэлэлцүүлэг хичээлтэй нэг хүсэлтээр ирнэ (тус тусдаа 2 хүсэлтийн оронд).
    const l = await api(`/api/courses/${id}/lessons/${lid}${Auth.token ? "?include=highlights,discussion" : ""}`);
    $("#lessonTitle").textContent = l.title;
    const nav = $("#lessonNav");
    if (nav) { nav.innerHTML = navHTML(lid); nav.hidden = lessonRows().length < 2; nav.onclick = async (e) => { const b = e.target.closest("[data-nav]"); if (!b || b.disabled) return; const li = lessonRows().find((r) => r.dataset.lesson === b.dataset.nav); if (li) await expandRow(li); }; }
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
    body.classList.remove("fade-in"); void body.offsetWidth; body.classList.add("fade-in"); // хичээл солигдоход зөөлөн
    stopMarker?.(); // маркер: шалгалтаас бусад хичээлд, нэвтэрсэн суралцагчид (тодруулга зөвхөн өөрт нь)
    $$(".hl-bar, .hl-pop", body.parentElement).forEach((x) => x.remove());
    const tHref = $(".teacher-chip")?.getAttribute("href") || "", tUser = tHref.startsWith("/t/") ? tHref.slice(3) : "";
    stopMarker = Auth.token && !l.exam ? lessonMarker(body, `/api/courses/${id}/lessons/${lid}/highlights`, { teacher: tUser, self: !!tUser && Auth.user?.username === tUser, title: l.title, preload: l._highlights }) : null;
    const modal = $("#lessonModal");
    const isOwner = !!Auth.user?.username && $(".teacher-chip")?.getAttribute("href") === "/t/" + Auth.user.username;
    mountClouds(body, Auth.token ? `/api/courses/${id}/lessons/${lid}` : "", isOwner);
    mountBoards(body, Auth.token ? `/api/courses/${id}/lessons/${lid}` : "", isOwner);
    if (l.exam) { // шалгалт: асуултууд зөвхөн "эхлүүлэх"-ээр ирнэ
      if (done) done.hidden = true;
      examCard(body, { courseId: id, lessonId: lid, title: l.title, onFinish: () => autoAdvance(false) });
    }
    if (l.assignment) { // даалгавар: хариу илгээснээр дуусна
      if (done) done.hidden = true;
      assignmentCard(body, { courseId: id, lessonId: lid });
    }
    const dbox = $("#lessonDiscuss");
    if (dbox) { dbox.hidden = true; dbox.innerHTML = ""; if (l.discussion && !l.exam) discussionPanel(dbox, { courseId: id, lessonId: lid, preload: l._discussion }); }
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
    const lazy = lazyWatch; lazyWatch = false;
    stopWatch = l.exam ? null : watchLesson({ courseId: id, lessonId: lid, modal, onStop: () => { closeModal(modal); refreshAccess(); }, onActive, maxWarn: access?.max_warnings, lazy });
    reflectDone = !!(Auth.token && access?.progress?.[lid]?.completed_at) || isOwner || !Auth.token || !!l.exam;
    curLesson = reflectDone ? null : { id, lid, title: l.title };
    openModal(modal);
  };

  // Нүүр хуудаснаас #l=<хичээл> холбоосоор ирвэл тухайн хичээлийг шууд нээнэ (эрхгүй бол мөрийг тодруулна).
  const want = new URLSearchParams(location.hash.slice(1)).get("l");
  if (want) {
    const li = rowById(want);
    if (li) ready.then(() => {
      if (!Auth.token) { li.scrollIntoView({ block: "center" }); li.animate([{ boxShadow: "0 0 0 4px rgba(31,60,143,.35)" }, { boxShadow: "0 0 0 0 transparent" }], { duration: 1600 }); return; }
      expandRow(li, { push: false });
    });
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
      <div class="pw-side"><p class="pw-amount">${money(p.amount || ord.amount)}</p>${p.fee > 0 ? `<p class="pw-fee">${money(p.base)} + үйлчилгээний хураамж ${money(p.fee)} (${p.fee_pct}%)</p>` : ""}
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

/* ---------- Цаг авах: багшийн сул цагаас сонгож захиална (үнэгүй бол шууд, төлбөртэй бол QR) ---------- */
function apptWindow(username, tname) {
  $("#apptWin")?.remove();
  const el = document.createElement("div");
  el.className = "modal ap-win"; el.id = "apptWin"; el.setAttribute("aria-hidden", "true");
  el.innerHTML = `<div class="modal-card"><button class="icon-btn modal-x" data-close aria-label="Хаах">✕</button>
    <span class="eyebrow">Цаг авах</span><h3 class="h3">${esc(tname)}-тай уулзах</h3><div class="ap-body"><div class="pw-load"><span class="loader"></span></div></div></div>`;
  document.body.append(el); openModal(el);
  const body = $(".ap-body", el), shut = () => { closeModal(el); setTimeout(() => el.remove(), 300); };
  const hm = (d) => `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
  const dkey = (d) => `${d.getFullYear()}-${d.getMonth() + 1}-${d.getDate()}`;
  const dayName = (d) => { const t = new Date(); t.setHours(0, 0, 0, 0); const n = Math.round((new Date(d).setHours(0, 0, 0, 0) - t) / 864e5); return n === 0 ? "Өнөөдөр" : n === 1 ? "Маргааш" : WEEKDAYS_SHORT[d.getDay()]; };
  const when = (d, min) => `${WEEKDAYS[d.getDay()]}, ${d.getMonth() + 1}/${d.getDate()} · ${hm(d)}–${hm(new Date(d.getTime() + min * 6e4))}`;
  const place = (x) => x.mode === "offline" ? `${icon("pin", 14)} Биечлэн · ${esc(x.location || "")}` : `${icon("live", 14)} Онлайн · Google Meet`;
  let data = null, day = "", pick = null;
  const mineHTML = () => data.mine.length ? `<div class="ap-mine"><b>Таны захиалга</b>${data.mine.map((x) => `<div class="ap-mine-row"><span>${when(new Date(x.starts_at), x.duration_min)}<small>${place(x)}</small></span>
    ${x.meet_url ? `<a class="btn btn-accent btn-sm" href="${esc(x.meet_url)}" target="_blank" rel="noopener">Нэгдэх</a>` : ""}</div>`).join("")}</div>` : "";
  const draw = () => {
    if (!data.slots.length) {
      body.innerHTML = mineHTML() + `<div class="empty">Одоогоор сул цаг алга.<br>Багштай чатаар тохиролцоорой.</div><div class="hero-cta" style="justify-content:center"><button type="button" class="btn btn-gold" data-chat>${icon("chat", 16)}Чатлах</button></div>`;
      return;
    }
    const days = [...new Map(data.slots.map((x) => [dkey(x.d), x.d])).entries()];
    const list = data.slots.filter((x) => dkey(x.d) === day), sel = data.slots.find((x) => x.id === pick);
    body.innerHTML = mineHTML() + `<div class="ap-days" role="tablist" aria-label="Өдөр">${days.map(([k, d]) => `<button type="button" class="ap-day" role="tab" aria-selected="${k === day}" data-day="${k}"><b>${dayName(d)}</b><span>${d.getMonth() + 1}/${d.getDate()}</span></button>`).join("")}</div>
      <div class="ap-times">${list.map((x) => `<button type="button" class="ap-time" aria-pressed="${x.id === pick}" data-pick="${x.id}"><b>${hm(x.d)}</b><small>${x.duration_min} мин · ${x.mode === "offline" ? "Биечлэн" : "Онлайн"}</small><em>${x.price ? money(x.price) : "Үнэгүй"}</em></button>`).join("")}</div>
      ${sel ? `<div class="ap-pick"><div class="ap-sum"><b>${when(sel.d, sel.duration_min)}</b><small>${place(sel)}</small></div>
        <label class="ap-note"><span>Юу ярилцах вэ? <small class="muted">заавал биш</small></span><textarea maxlength="500" rows="2" placeholder="Жишээ нь: ЭЕШ-ийн бэлтгэл, логарифмын бодлого"></textarea></label>
        <button type="button" class="btn btn-gold btn-block btn-lg" data-go>${Auth.token ? (sel.price ? `Төлөөд захиалах · ${money(sel.price)}` : "Цаг захиалах") : "Нэвтэрч захиалах"}</button></div>`
      : `<p class="muted small ap-hint">Тохирох цагаа сонгоно уу.</p>`}`;
  };
  const load = async () => {
    try { data = await api(`/api/teachers/${encodeURIComponent(username)}/slots`); }
    catch (e) { body.innerHTML = `<p class="form-error">${esc(e.message)}</p>`; return; }
    data.slots.forEach((x) => (x.d = new Date(x.starts_at)));
    if (!data.slots.some((x) => dkey(x.d) === day)) day = data.slots[0] ? dkey(data.slots[0].d) : "";
    if (!data.slots.some((x) => x.id === pick)) pick = null;
    draw();
  };
  const done = (x) => {
    body.innerHTML = `<div class="ap-done"><span class="ap-ok" aria-hidden="true">✓</span><h4>Цаг баталгаажлаа</h4><p><b>${when(new Date(x.starts_at), x.duration_min)}</b></p><p class="muted">${place(x)}</p>
      ${x.meet_url ? `<a class="btn btn-accent" href="${esc(x.meet_url)}" target="_blank" rel="noopener">${icon("live", 16)}Google Meet холбоос</a>` : x.mode === "online" ? `<p class="muted small">Meet холбоосыг багш илгээнэ.</p>` : ""}
      <p class="muted small">Таны нүүр хуудасны «Шууд хичээл» хэсэгт хадгалагдлаа.</p>
      <div class="hero-cta" style="justify-content:center"><a class="btn btn-ghost" href="/#my-live">Миний хуваарь</a><button type="button" class="btn btn-gold" data-close>Болсон</button></div></div>`;
  };
  el.addEventListener("click", async (e) => {
    if (e.target === el || e.target.closest("[data-close]")) { e.preventDefault(); shut(); return; }
    const dy = e.target.closest("[data-day]"), pk = e.target.closest("[data-pick]");
    if (dy) { day = dy.dataset.day; pick = null; draw(); return; }
    if (pk) { pick = pk.dataset.pick; draw(); $(".ap-pick", body)?.scrollIntoView({ block: "nearest", behavior: "smooth" }); return; }
    if (e.target.closest("[data-chat]")) { shut(); $("[data-open-chat]")?.click(); return; }
    const go = e.target.closest("[data-go]"); if (!go) return;
    if (!Auth.token) { location.href = `/login?next=${encodeURIComponent(location.pathname + "#appt")}`; return; }
    const sel = data.slots.find((x) => x.id === pick); if (!sel) return;
    go.disabled = true;
    try {
      const r = await api(`/api/slots/${sel.id}/book`, { method: "POST", body: { note: $("textarea", body)?.value || "" } });
      if (r.booked) { done(r.slot); return; }
      payWindow(r.order, r.payment, "Уулзах цаг", async () => {
        const d2 = await api(`/api/teachers/${encodeURIComponent(username)}/slots`).catch(() => null);
        const mine = d2?.mine.find((x) => x.id === sel.id);
        if (mine) done(mine); else load();
      }, { name: "Уулзах цаг", sub: when(sel.d, sel.duration_min) });
      go.disabled = false;
    } catch (x) { toast(x.message, true); go.disabled = false; if (x.status === 409 || x.status === 410) { pick = null; load(); } }
  });
  load();
}
window.SG_appt = apptWindow;

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
        ${it.match ? `<span class="find-match">${icon("book", 13)} ${mark(it.match)}</span>` : ""}
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
  $$(".rc-ico[data-i]").forEach((e) => (e.innerHTML = icon(e.dataset.i, 30)));
  const roleStep = $("#roleStep"), roleChosen = $("#roleChosen"), tabs = $(".tabs"), lf = $("#loginForm"), rf = $("#registerForm");
  if (h.get("token")) {
    const tok = h.get("token"), ask = h.get("ask") === "role";
    history.replaceState(null, "", location.pathname);
    api("/api/me", { token: tok }).then((u) => {
      Auth.set(tok, u);
      if (!ask) return location.replace(next(u));
      // Google/Facebook-ээр шинээр орсон: багш уу, суралцагч уу гэдгийг асуугаад тохирох хэсэг рүү.
      tabs.hidden = true; lf.hidden = true; $("#social").hidden = true; $(".divider").hidden = true; $(".auth-legal").hidden = true;
      $(".role-q b", roleStep).textContent = `Тавтай морил${u.display_name ? ", " + u.display_name.split(" ")[0] : ""}! Та хэн бэ?`;
      roleStep.hidden = false;
      roleStep.onclick = async (e) => {
        const c = e.target.closest("[data-role]"); if (!c) return;
        $$(".role-card", roleStep).forEach((x) => { x.disabled = true; x.classList.toggle("on", x === c); });
        try { const d = await api("/api/me/role", { method: "POST", body: { role: c.dataset.role }, token: tok }); Auth.set(d.token, d.user); location.replace(home(d.user)); }
        catch (x) { err.textContent = x.message; $$(".role-card", roleStep).forEach((y) => (y.disabled = false)); }
      };
    }).catch((e) => (err.textContent = e.message));
    return;
  }
  if (h.get("error")) err.textContent = h.get("error");
  // Бүртгүүлэх: эхлээд багш уу, суралцагч уу гэдгийг сонгуулна; нэвтрэхэд төрөл асуухгүй.
  let role = ["teacher", "student"].includes(q.get("role")) ? q.get("role") : "", tab = "login", forgotOn = false;
  const sync = () => {
    if (forgotOn) return; // «Нууц үгээ мартсан» нээлттэй үед (жишээ нь /login#forgot) товчнууд ачаалагдсаны дараа хаагдахгүй
    const reg = tab === "register", needRole = reg && !role;
    roleStep.hidden = !needRole; roleChosen.hidden = !reg || !role;
    if (role) $("span", roleChosen).innerHTML = `${icon(role === "teacher" ? "teacher" : "courses", 18)}<b>${role === "teacher" ? "Багш" : "Суралцагч"}</b>аар бүртгүүлж байна`;
    $$(".role-card", roleStep).forEach((c) => { const on = c.dataset.role === role; c.classList.toggle("on", on); c.setAttribute("aria-checked", on); });
    lf.hidden = reg; rf.hidden = !reg || needRole; $("#forgotForm").hidden = true;
    $("#social").hidden = needRole; $(".divider").hidden = needRole || !providers.length;
    renderSocial(providers);
  };
  $$(".tab").forEach((t) => t.addEventListener("click", () => {
    $$(".tab").forEach((x) => x.classList.toggle("active", x === t));
    tabs.dataset.active = tab = t.dataset.tab; err.textContent = ""; sync();
  }));
  roleStep.addEventListener("click", (e) => { const c = e.target.closest("[data-role]"); if (!c || h.get("token")) return; role = c.dataset.role; sync(); (rf.email || rf).focus?.(); });
  roleChosen.addEventListener("click", (e) => { if (e.target.closest("[data-role-change]")) { role = ""; sync(); } });
  // Нэвтрэх товчны таних тэмдгүүд (үйлчилгээ тус бүрийн албан ёсны өнгөөр).
  const GOOGLE_G = `<svg viewBox="0 0 48 48" width="20" height="20" aria-hidden="true"><path fill="#EA4335" d="M24 9.500c3.540 0 6.710 1.220 9.210 3.600l6.850-6.850C35.900 2.380 30.470 0 24 0 14.620 0 6.510 5.380 2.560 13.220l7.980 6.190C12.430 13.720 17.740 9.500 24 9.500z"/><path fill="#4285F4" d="M46.980 24.550c0-1.570-.150-3.090-.380-4.550H24v9.020h12.940c-.580 2.960-2.260 5.480-4.780 7.180l7.730 6c4.510-4.180 7.090-10.360 7.090-17.650z"/><path fill="#FBBC05" d="M10.530 28.590c-.480-1.450-.760-2.990-.760-4.590s.270-3.140.760-4.590l-7.980-6.190C.920 16.460 0 20.120 0 24c0 3.880.920 7.540 2.560 10.780l7.970-6.190z"/><path fill="#34A853" d="M24 48c6.480 0 11.930-2.130 15.890-5.810l-7.730-6c-2.150 1.450-4.920 2.300-8.160 2.300-6.260 0-11.570-4.220-13.470-9.910l-7.980 6.190C6.510 42.620 14.620 48 24 48z"/></svg>`;
  const FACEBOOK_F = `<svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true"><circle cx="12" cy="12" r="12" fill="#1877F2"/><path fill="#fff" d="M16.500 15.500l.500-3.500h-3.300V9.800c0-1 .500-1.900 2-1.900H17V5c-.300 0-1.300-.200-2.500-.200-2.600 0-4.200 1.600-4.200 4.400V12H7.500v3.500h2.800V24h3.400v-8.500z"/></svg>`;
  const icons = { google: [GOOGLE_G, "Google"], facebook: [FACEBOOK_F, "Facebook"],
    microsoft: [`<span class="social-ico" style="background:#0078d4">⊞</span>`, "Microsoft"],
    instagram: [`<span class="social-ico" style="background:linear-gradient(45deg,#f58529,#dd2a7b,#8134af)">◎</span>`, "Instagram"] };
  const social = $("#social");
  // Facebook, Messenger, Instagram доторх хөтөчөөс Google нэвтрэлтийг Google өөрөө хаадаг (403 disallowed_useragent) —
  // товчийг дарахад алдааны хуудас биш, жинхэнэ хөтөчөөр нээх заавар гаргана.
  const inApp = /FBAN|FBAV|FB_IAB|FBIOS|Instagram|Messenger/i.test(navigator.userAgent);
  const renderSocial = (ps) => {
    const nx = h.get("next") || q.get("next") || "", rl = tab === "register" ? role : ""; // нэвтрэх: шинэ хэрэглэгчээс дараа нь асууна
    social.innerHTML = ps.map((p) => {
      const [ic, name] = icons[p.name] || [`<span class="social-ico" style="background:#555">•</span>`, p.label];
      const u = `${p.url}?${new URLSearchParams({ ...(rl ? { role: rl } : {}), ...(nx ? { next: nx } : {}) })}`;
      if (inApp && p.name === "google") return `<button type="button" class="btn btn-social" data-inapp>${ic}<span>${esc(name)}-ээр үргэлжлүүлэх</span></button>`;
      return `<a class="btn btn-social" href="${esc(u)}">${ic}<span>${esc(name)}-ээр үргэлжлүүлэх</span></a>`;
    }).join("") + (inApp ? `<div class="inapp-note" hidden></div>` : "");
    $(".divider").hidden = !ps.length;
  };
  social.addEventListener("click", (e) => {
    if (!e.target.closest("[data-inapp]")) return;
    const note = $(".inapp-note", social), url = location.href.split("#")[0];
    const android = /Android/i.test(navigator.userAgent);
    note.innerHTML = `<b>Facebook доторх хөтөч Google-ээр нэвтрэхийг зөвшөөрдөггүй</b>
      <span>${android ? "Доорх товчоор Chrome-оор нээгээд" : "Баруун дээд/доод буланд байрлах <b>⋯</b> → <b>«Open in browser» (Хөтчөөр нээх)</b>-ийг дарж Safari-д нээгээд"} Google-ээр нэвтэрнэ үү. Эсвэл Facebook, имэйлээр шууд нэвтэрч болно.</span>
      <span class="inapp-act">${android ? `<a class="btn btn-gold btn-sm" href="${esc(`intent://${location.host}${location.pathname}${location.search}#Intent;scheme=https;S.browser_fallback_url=${encodeURIComponent(url)};end`)}">Chrome-оор нээх</a>` : ""}<button type="button" class="btn btn-ghost btn-sm" data-copy-link>Холбоос хуулах</button></span>`;
    note.hidden = false;
    $("[data-copy-link]", note).onclick = (ev) => navigator.clipboard?.writeText(url).then(() => (ev.target.textContent = "Хуулагдлаа ✓"), () => prompt("Холбоосыг хуулна уу:", url));
  });
  let providers = [];
  api("/api/auth/providers", { token: null }).then((ps) => { providers = ps; sync(); }).catch(() => sync());
  if (role) $$(".tab")[1].click(); else sync();
  const done = (d) => { Auth.set(d.token, d.user); location.href = next(d.user); };
  lf.addEventListener("submit", async (e) => {
    e.preventDefault(); err.textContent = "";
    try { done(await api("/api/auth/login", { method: "POST", body: { email: lf.email.value, password: lf.password.value }, token: null })); }
    catch (x) { err.textContent = x.message; $(".auth-card").animate([{ transform: "translateX(0)" }, { transform: "translateX(-10px)" }, { transform: "translateX(10px)" }, { transform: "translateX(0)" }], { duration: 350 }); }
  });
  // Нууц үг мартсан: имэйл рүү сэргээх холбоос.
  const ff = $("#forgotForm");
  // Хялбар тоон captcha: нэг код нэг л удаа шалгагдана — илгээх бүрийн дараа шинийг татна.
  let capId = "";
  const newCaptcha = async () => {
    const img = $(".captcha img", ff); ff.captcha.value = "";
    try { const c = await api("/api/auth/captcha", { token: null }); capId = c.id; img.src = c.image; }
    catch (x) { err.textContent = x.message; }
  };
  const showForgot = (on) => {
    ff.hidden = !on; tabs.hidden = on; err.textContent = ""; forgotOn = on;
    if (!on) { sync(); return; }
    lf.hidden = true; rf.hidden = true; roleStep.hidden = true; roleChosen.hidden = true; $("#social").hidden = true; $(".divider").hidden = true;
    ff.email.value = lf.email.value; (fgMode === "phone" ? ff.phone : ff.email).focus();
    newCaptcha();
  };
  $("[data-forgot]").addEventListener("click", () => showForgot(true));
  $("[data-forgot-back]").addEventListener("click", () => showForgot(false));
  $("[data-captcha-new]", ff).addEventListener("click", () => { newCaptcha(); ff.captcha.focus(); });
  // Сэргээх арга: имэйл эсвэл (Verify.MN идэвхтэй бол) баталгаажуулсан утас — утаснаасаа SMS илгээж шууд сэргээнэ.
  const LEDE = { email: "Бүртгэлтэй имэйлээ бичнэ үү. Нууц үг сэргээх холбоос илгээнэ.", phone: "Бүртгэлдээ үлдээсэн утасны дугаараа бичнэ үү. Тэр дугаараасаа нэг SMS илгээгээд нууц үгээ шууд солино." };
  let fgMode = "email";
  const sms = $(".fg-sms", ff), go = $(".fg-go", ff);
  const setMode = (m) => {
    fgMode = m; err.textContent = ""; sms.hidden = true; sms.innerHTML = "";
    $$("[data-fg]", ff).forEach((b) => { const on = b.dataset.fg === m; b.classList.toggle("on", on); b.setAttribute("aria-selected", on); });
    $$("[data-fg-in]", ff).forEach((l) => { const on = l.dataset.fgIn === m; l.hidden = !on; $("input", l).disabled = !on; });
    $(".cap-field", ff).hidden = false; go.hidden = false;
    $(".forgot-lede", ff).textContent = LEDE[m];
    go.textContent = m === "phone" ? "Үргэлжлүүлэх" : "Сэргээх холбоос илгээх";
  };
  $(".fg-mode", ff).addEventListener("click", (e) => { const b = e.target.closest("[data-fg]"); if (b) { setMode(b.dataset.fg); (fgMode === "phone" ? ff.phone : ff.email).focus(); } });
  api("/api/auth/recovery", { token: null }).then((r) => {
    if (!r.phone) return;
    $(".fg-mode", ff).hidden = false;
    if (!r.email) setMode("phone");
  }).catch(() => {});
  // Утсаар баталгаажсаны дараа: нэг бүртгэл бол шууд шинэ нууц үг тавих хуудас руу, олон бол сонгуулна.
  const phoneDone = (r) => {
    const a = r.accounts || [];
    if (a.length === 1) { location.href = "/reset#t=" + a[0].token; return; }
    sms.innerHTML = `<p class="fg-pick-h">Энэ дугаартай ${a.length} бүртгэл байна — алийнх нь нууц үгийг солих вэ?</p>
      <div class="fg-pick">${a.map((x) => `<a class="btn btn-ghost btn-block" href="/reset#t=${esc(x.token)}"><b>${esc(x.name || x.username)}</b><small>${esc(x.email || "@" + x.username)} · ${x.role === "teacher" ? "Багш" : "Суралцагч"}</small></a>`).join("")}</div>`;
  };
  ff.addEventListener("submit", async (e) => {
    e.preventDefault(); err.textContent = "";
    go.disabled = true;
    try {
      if (fgMode === "phone") {
        const v = await api("/api/auth/forgot/phone", { method: "POST", body: { phone: ff.phone.value, captcha_id: capId, captcha: ff.captcha.value }, token: null });
        $$("[data-fg-in], .cap-field", ff).forEach((x) => (x.hidden = true)); go.hidden = true; sms.hidden = false;
        smsVerify(sms, v, (token) => api("/api/auth/forgot/phone/check", { method: "POST", body: { token }, token: null }), phoneDone, () => { setMode("phone"); newCaptcha(); });
      } else {
        await api("/api/auth/forgot", { method: "POST", body: { email: ff.email.value, captcha_id: capId, captcha: ff.captcha.value }, token: null });
        $(".forgot-lede", ff).innerHTML = `<b>Имэйлээ шалгана уу.</b> Хэрэв <b>${esc(ff.email.value)}</b> хаягаар бүртгэл байвал нууц үг сэргээх холбоос илгээлээ (Spam хавтсыг ч харна уу). Холбоос 1 цаг хүчинтэй.`;
        go.textContent = "Дахин илгээх";
      }
    } catch (x) { err.textContent = x.message; if (x.data?.code === "captcha") ff.captcha.focus(); }
    newCaptcha();
    go.disabled = false;
  });
  if (location.hash === "#forgot") showForgot(true);
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
  rf.accept.addEventListener("invalid", () => rf.accept.setCustomValidity("Үргэлжлүүлэхийн тулд нөхцөл, нууцлалын бодлогыг зөвшөөрнө үү"));
  rf.accept.addEventListener("change", () => rf.accept.setCustomValidity(""));
  rf.addEventListener("submit", async (e) => {
    e.preventDefault(); err.textContent = "";
    if (!checkMatch()) { err.textContent = "Нууц үг таарахгүй байна"; rf.password2.focus(); return; }
    if (!role) { err.textContent = "Эхлээд багш уу, суралцагч уу гэдгээ сонгоно уу"; return sync(); }
    const body = { email: rf.email.value, password: rf.password.value, role, accept_terms: rf.accept.checked };
    try { done(await api("/api/auth/register", { method: "POST", body, token: null })); } catch (x) { err.textContent = x.message; }
  });
}

// Хянагчийн хуудас (/admin): чатын мэдүүлгийг харж шийдвэрлэнэ (сонговол мэдүүлсэн мессежийг устгана).
// Бүртгэлээ устгах: юу устахыг тайлбарлаж «УСТГАХ» гэж бичүүлнэ (буцаах боломжгүй).
function deleteAccountDialog() {
  if ($(".rp")) return;
  const el = document.createElement("div");
  el.className = "rp"; el.setAttribute("role", "dialog"); el.setAttribute("aria-modal", "true"); el.setAttribute("aria-labelledby", "daT");
  el.innerHTML = `<form class="rp-card">
    <h3 id="daT">Бүртгэлээ устгах уу?</h3>
    <ul class="da-list">
      <li>Нэр, имэйл, профайл, суралцсан явц, дүн, хичээлийн хяналтын бүртгэл, мэдэгдэл бүрмөсөн устна.</li>
      <li>Чатын мессежүүд тань «устгагдсан» болж, нэр тань арилна.</li>
      <li>Удахгүй болох цагийн захиалга цуцлагдаж, багшид мэдэгдэнэ.</li>
      <li>Худалдан авсан сургалт, номдоо дахин хандах боломжгүй. Төлбөрийн баримт хуулийн дагуу нэргүйгээр хадгалагдана.</li>
    </ul>
    <p class="rp-note">Устгахаас өмнө өгөгдлөө татаж авахыг зөвлөж байна. Энэ үйлдлийг буцаах боломжгүй.</p>
    <label>Баталгаажуулахын тулд «УСТГАХ» гэж бичнэ үү<input name="confirm" autocomplete="off" spellcheck="false"></label>
    <div class="rp-act"><button type="button" class="btn btn-ghost" data-rp-x>Болих</button><button class="btn btn-danger" disabled>Бүртгэлээ устгах</button></div>
  </form>`;
  document.body.append(el);
  const f = $("form", el), go = $(".btn-danger", f), onKey = (e) => { if (e.key === "Escape") close(); };
  const close = () => { el.remove(); document.removeEventListener("keydown", onKey); };
  document.addEventListener("keydown", onKey);
  el.addEventListener("click", (e) => { if (e.target === el || e.target.closest("[data-rp-x]")) close(); });
  f.confirm.addEventListener("input", () => (go.disabled = f.confirm.value.trim().toUpperCase() !== "УСТГАХ"));
  f.confirm.focus();
  f.onsubmit = async (e) => {
    e.preventDefault(); if (go.disabled) return;
    go.disabled = true;
    try { await api("/api/me/delete", { method: "POST", body: { confirm: f.confirm.value } }); Auth.clear(); location.href = "/?deleted=1"; }
    catch (x) { toast(x.message, true); go.disabled = false; }
  };
}

// Нууцлал, нөхцөлийн хуудас: уншиж буй хэсгийг агуулгын жагсаалтад тодруулна.
function legalPage() {
  const links = $$(".pv-toc a"), byId = new Map(links.map((a) => [a.hash.slice(1), a]));
  if (!links.length || !("IntersectionObserver" in window)) return;
  const io = new IntersectionObserver((es) => es.forEach((e) => {
    if (e.isIntersecting) links.forEach((a) => a.classList.toggle("on", a === byId.get(e.target.id)));
  }), { rootMargin: "-25% 0px -65% 0px" });
  $$(".pv-doc > section[id]").forEach((s) => io.observe(s));
}

// Нууц үг харуулах/нуух — бүх хуудсанд (нэвтрэх, бүртгүүлэх, сэргээх).
document.addEventListener("click", (e) => {
  const b = e.target.closest("[data-pw-toggle]"); if (!b) return;
  const i = b.previousElementSibling, show = i.type === "password";
  i.type = show ? "text" : "password"; b.classList.toggle("on", show); b.setAttribute("aria-label", show ? "Нууц үг нуух" : "Нууц үг харуулах");
});

// Шинэ нууц үг тавих (/reset#t=<токен> — имэйлийн холбоос).
function resetPage() {
  const f = $("#resetForm"), err = $("#resetError"); if (!f) return;
  const tok = new URLSearchParams(location.hash.slice(1)).get("t");
  if (tok) history.replaceState(null, "", location.pathname); // токеныг хаягийн мөрөөс арилгана
  else { f.hidden = true; err.innerHTML = `Холбоос дутуу байна. <a href="/login#forgot">Дахин сэргээх хүсэлт илгээх</a>`; return; }
  f.addEventListener("submit", async (e) => {
    e.preventDefault(); err.textContent = "";
    if (f.password.value !== f.password2.value) { err.textContent = "Нууц үг таарахгүй байна"; f.password2.focus(); return; }
    const b = $("button.btn-gold", f); b.disabled = true;
    try {
      const d = await api("/api/auth/reset", { method: "POST", body: { token: tok, password: f.password.value }, token: null });
      Auth.set(d.token, d.user); toast("Нууц үг солигдлоо ✓");
      setTimeout(() => (location.href = d.user.role === "teacher" && d.user.username ? "/t/" + d.user.username : "/"), 600);
    } catch (x) { err.innerHTML = `${esc(x.message)} <a href="/login#forgot">Дахин хүсэлт илгээх</a>`; b.disabled = false; }
  });
}

/* ---------- эхлүүлэх ---------- */
store.set("sg_guest", null); // зочны чат хасагдсан — хуучин зочны токеныг устгана
/* ---------- Зураг томруулах: хичээл доторх зураг дээр 2 удаа товшиход хуудсан дотроо цонхолж харуулна ---------- */
// Олон зурагтай бол ‹ › (сум товч, шудрах), зураг дээр дарахад томорно; Escape хичээлээс гаргахгүй, зөвхөн цонхыг хаана.
const LBX_SEL = ".rb-fig img, .rb-text img, .rq-img";
const lbxOK = (im) => im && !im.closest(".be-rt, .rq-spotwrap, a, .lbx") && !document.querySelector(".lbx");
function lightbox(img) {
  const scope = img.closest(".lesson-content, .rb, .modal-card") || document;
  const list = $$(LBX_SEL, scope).filter((x) => !x.closest(".be-rt, .rq-spotwrap, a") && x.naturalWidth);
  let i = Math.max(0, list.indexOf(img));
  const el = document.createElement("div");
  el.className = "lbx"; el.setAttribute("role", "dialog"); el.setAttribute("aria-modal", "true"); el.setAttribute("aria-label", "Зураг томруулж харах");
  el.innerHTML = `<div class="lbx-stage"><img alt="" draggable="false"></div>
    <button type="button" class="lbx-x" aria-label="Хаах" title="Хаах (Esc)"><svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" aria-hidden="true"><path d="M6 6l12 12M18 6 6 18"/></svg></button>
    ${list.length > 1 ? `<button type="button" class="lbx-nav lbx-prev" aria-label="Өмнөх зураг">‹</button><button type="button" class="lbx-nav lbx-next" aria-label="Дараах зураг">›</button>` : ""}
    <div class="lbx-bar"><span class="lbx-cap"></span>${list.length > 1 ? `<span class="lbx-n"></span>` : ""}</div>`;
  document.body.append(el); document.documentElement.classList.add("lbx-open");
  const stage = $(".lbx-stage", el), im = $("img", stage), last = document.activeElement, t0 = Date.now();
  const unzoom = () => { stage.classList.remove("z"); im.style.width = ""; };
  const show = () => {
    const src = list[i] || img;
    unzoom(); im.src = src.currentSrc || src.src; im.alt = src.alt || "";
    $(".lbx-cap", el).textContent = src.closest("figure")?.querySelector("figcaption")?.textContent || src.alt || "";
    const n = $(".lbx-n", el); if (n) n.textContent = `${i + 1} / ${list.length}`;
  };
  const go = (d) => { if (list.length > 1) { i = (i + d + list.length) % list.length; show(); } };
  // Томруулах: дарсан цэг нь төвдөө үлдэнэ.
  const zoom = (e) => {
    if (stage.classList.contains("z")) return unzoom();
    const r = im.getBoundingClientRect(), fx = (e.clientX - r.left) / r.width, fy = (e.clientY - r.top) / r.height;
    const w = Math.max(im.naturalWidth, r.width * 2);
    if (w <= r.width + 4) return;
    stage.classList.add("z"); im.style.width = w + "px";
    stage.scrollLeft = fx * im.offsetWidth - stage.clientWidth / 2; stage.scrollTop = fy * im.offsetHeight - stage.clientHeight / 2;
  };
  const close = () => {
    removeEventListener("keydown", onKey, true);
    el.classList.remove("on"); document.documentElement.classList.remove("lbx-open");
    setTimeout(() => el.remove(), 200); last?.focus?.({ preventScroll: true });
  };
  const onKey = (e) => {
    const k = { Escape: close, ArrowRight: () => go(1), ArrowLeft: () => go(-1) }[e.key];
    if (k) { e.preventDefault(); e.stopImmediatePropagation(); k(); }
  };
  addEventListener("keydown", onKey, true);
  let sw = null; // шудрах (хуруу)
  stage.addEventListener("pointerdown", (e) => { sw = stage.classList.contains("z") ? null : { x: e.clientX, y: e.clientY }; });
  stage.addEventListener("pointerup", (e) => {
    if (!sw) return;
    const dx = e.clientX - sw.x, dy = e.clientY - sw.y; sw = null;
    if (Math.abs(dx) > 50 && Math.abs(dx) > Math.abs(dy)) { el._swiped = true; go(dx < 0 ? 1 : -1); }
  });
  el.addEventListener("click", (e) => {
    if (el._swiped) { el._swiped = false; return; }
    if (Date.now() - t0 < 450) return; // нээсэн 2 дахь товшилт цонхны зураг дээр давхар бууна
    if (e.target.closest(".lbx-x")) return close();
    if (e.target.closest(".lbx-prev")) return go(-1);
    if (e.target.closest(".lbx-next")) return go(1);
    if (e.target === im) return zoom(e);
    if (e.target === el || e.target === stage) close();
  });
  show(); requestAnimationFrame(() => el.classList.add("on")); $(".lbx-x", el).focus({ preventScroll: true });
}
// 2 удаа товшиход: хулгана — dblclick; хуруу — 2 товшилт 320мс дотор (dblclick ирэхгүй хөтөчид).
document.addEventListener("dblclick", (e) => {
  const im = e.target.closest?.(LBX_SEL); if (!lbxOK(im)) return;
  e.preventDefault(); getSelection()?.removeAllRanges(); lightbox(im);
});
let lbxTap = null;
document.addEventListener("pointerup", (e) => {
  if (e.pointerType !== "touch") return;
  const im = e.target.closest?.(LBX_SEL); if (!lbxOK(im)) { lbxTap = null; return; }
  const now = Date.now();
  if (lbxTap?.im === im && now - lbxTap.t < 320) { lbxTap = null; getSelection()?.removeAllRanges(); lightbox(im); }
  else lbxTap = { im, t: now };
});

/* ---------- Хүртээмж: харааны бэрхшээлтэй хүмүүст зориулсан байнга дагадаг цэс (бүх хуудсанд, зүүн доод буланд) ---------- */
// Сонголт бүр дарах тутам түвшин нэмэгдэж (3 хүрээд унтарна), тохиргоо энэ төхөөрөмж дээр хадгалагдана.
const A11Y_SVG = (p, w = 34) => `<svg width="${w}" height="${w}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${p}</svg>`;
const A11Y = [ // [түлхүүр, нэр, түвшний тоо (1 = асаах/унтраах), түвшний нэр, дүрс]
  ["fs", "Фонт томруулах", 3, ["", "Том", "Илүү том", "Маш том"], A11Y_SVG('<path d="M3 10h7M6.5 10v9M12 5h9M16.5 5v14"/>')],
  ["cur", "Хулгана томруулах", 1, null, A11Y_SVG('<path d="M5 3l14 8-6.5 1.6L9.6 19z"/>')],
  ["ln", "Холбоос тодруулах", 1, null, A11Y_SVG('<path d="M9 17H7A5 5 0 0 1 7 7h2M15 7h2a5 5 0 0 1 0 10h-2M8 12h8"/>')],
  ["ct", "Контраст +", 3, ["", "Өндөр контраст", "Хар дэвсгэр", "Хар цагаан"], A11Y_SVG('<circle cx="12" cy="12" r="9"/><path d="M12 3a9 9 0 0 1 0 18z" fill="currentColor"/>')],
  ["tx", "Текст тодруулах", 1, null, '<span class="a11y-df" aria-hidden="true">Df</span>'],
  ["lh", "Текстийн мөрийн зай ихэсгэх", 3, ["", "1.9", "2.3", "2.7"], A11Y_SVG('<path d="M5 4v16M3 6.5 5 4l2 2.5M3 17.5 5 20l2-2.5M10 6h11M10 10h11M10 14h11M10 18h11"/>')],
  ["ls", "Үсгийн зай ихэсгэх", 3, ["", "Бага", "Дунд", "Их"], A11Y_SVG('<path d="M6 9l-3 3 3 3M18 9l3 3-3 3"/><path d="M7 12h10" stroke-dasharray="2 2.4"/>')],
  ["zm", "Zoom", 3, ["", "115%", "130%", "150%"], A11Y_SVG('<circle cx="10.5" cy="10.5" r="6.5"/><path d="M20 20l-4.6-4.6M10.5 7.5v6M7.5 10.5h6"/>')],
];
function mountA11y() {
  let st = {}; try { st = JSON.parse(localStorage.getItem("sg_a11y") || "{}") || {}; } catch {}
  const fab = document.createElement("button"), panel = document.createElement("div");
  fab.type = "button"; fab.className = "a11y-fab"; fab.title = "Хүртээмжийн тохиргоо";
  fab.setAttribute("aria-label", "Хүртээмжийн тохиргоо — харааны бэрхшээлтэй хүмүүст"); fab.setAttribute("aria-controls", "a11yPanel"); fab.setAttribute("aria-expanded", "false");
  fab.innerHTML = A11Y_SVG('<circle cx="12" cy="12" r="10"/><circle cx="12" cy="6.8" r="1.5" fill="currentColor" stroke="none"/><path d="M6.8 9.6 12 10.6l5.2-1M12 10.6v4.2M12 14.8l-2.6 4.4M12 14.8l2.6 4.4"/>', 32);
  panel.className = "a11y-panel"; panel.id = "a11yPanel"; panel.hidden = true;
  panel.setAttribute("role", "dialog"); panel.setAttribute("aria-label", "Хүртээмжийн тохиргоо");
  const apply = () => {
    const h = document.documentElement;
    [...h.classList].filter((c) => c.startsWith("a11y-")).forEach((c) => h.classList.remove(c));
    for (const [k, , n] of A11Y) { const v = st[k] | 0; if (v) h.classList.add(n === 1 ? `a11y-${k}` : `a11y-${k}${v}`); }
    fab.classList.toggle("on", A11Y.some(([k]) => st[k]));
    try { localStorage.setItem("sg_a11y", JSON.stringify(st)); } catch {}
  };
  const draw = () => {
    panel.innerHTML = `<div class="a11y-head"><b>Хүртээмж</b><button type="button" class="a11y-x" aria-label="Хаах">✕</button></div>
      <div class="a11y-grid">${A11Y.map(([k, t, n, subs, ic]) => { const v = st[k] | 0; return `<button type="button" class="a11y-opt" data-a11y="${k}" aria-pressed="${v > 0}" aria-label="${t}${n > 1 ? `, ${v ? v + "-р түвшин" : "унтраалттай"}` : ""}">
        <span class="a11y-ic">${ic}</span><span class="a11y-t">${t}</span>${n > 1 ? `<span class="a11y-sub">${subs[v] || ""}</span><span class="a11y-lv">${[1, 2, 3].map((x) => `<i class="${x <= v ? "on" : ""}"></i>`).join("")}</span>` : ""}</button>`; }).join("")}</div>
      <button type="button" class="a11y-reset">${A11Y_SVG('<path d="M20 11a8 8 0 1 0-2.3 5.7M20 4v7h-7"/>', 22)}Тохиргоог шинэчлэх</button><p class="a11y-live" aria-live="polite"></p>`;
  };
  const say = (t) => { const l = $(".a11y-live", panel); if (l) l.textContent = t; };
  const onKey = (e) => { if (e.key === "Escape") { e.preventDefault(); e.stopImmediatePropagation(); const a = anchor; toggle(false); (a || fab).focus(); } };
  // Маркерын самбар дахь товчноос нээвэл цонх товчны доор (самбар гүйлгэхэд дагадаг), эс бөгөөс зүүн доод буланд.
  let anchor = null;
  const place = () => {
    if (!anchor || panel.hidden) return;
    const r = anchor.getBoundingClientRect(), top = Math.max(8, Math.min(r.bottom + 8, innerHeight - 240));
    Object.assign(panel.style, { top: top + "px", bottom: "auto", maxHeight: innerHeight - top - 12 + "px" });
    if (innerWidth > 640) panel.style.left = Math.max(8, Math.min(r.left, innerWidth - panel.offsetWidth - 8)) + "px";
  };
  const toggle = (on, from) => {
    panel.hidden = !on; anchor = on ? from || null : null;
    $$("[data-a11y-open]").concat(fab).forEach((b) => b.setAttribute("aria-expanded", String(on && (b === from || (!from && b === fab)))));
    panel.removeAttribute("style");
    if (on) { draw(); place(); addEventListener("keydown", onKey, true); addEventListener("scroll", place, { passive: true }); addEventListener("resize", place); $(".a11y-opt", panel)?.focus(); }
    else { removeEventListener("keydown", onKey, true); removeEventListener("scroll", place); removeEventListener("resize", place); }
  };
  fab.addEventListener("click", () => toggle(panel.hidden));
  document.addEventListener("click", (e) => {
    const b = e.target.closest("[data-a11y-open]");
    if (b) return toggle(panel.hidden || anchor !== b, b);
    const path = e.composedPath(); // цонх дахин зурагдахад дарсан товч DOM-оос салдаг тул анхны замаар шалгана
    if (!panel.hidden && !path.includes(panel) && !path.includes(fab)) toggle(false); // гадуур дарвал хаана
  });
  panel.addEventListener("click", (e) => {
    const o = e.target.closest("[data-a11y]");
    if (o) {
      const [k, t, n, subs] = A11Y.find((x) => x[0] === o.dataset.a11y);
      st[k] = ((st[k] | 0) + 1) % (n + 1); if (!st[k]) delete st[k];
      apply(); draw(); $(`[data-a11y="${k}"]`, panel).focus();
      return say(`${t}: ${st[k] ? (n > 1 ? subs[st[k]] : "асаалттай") : "унтраалттай"}`);
    }
    if (e.target.closest(".a11y-reset")) { st = {}; apply(); draw(); $(".a11y-reset", panel).focus(); return say("Бүх тохиргоо анхны байдалдаа орлоо"); }
    if (e.target.closest(".a11y-x")) { const a = anchor; toggle(false); (a || fab).focus(); }
  });
  document.body.append(fab, panel); apply();
}
// Бүх хуудсанд (зүүн доод буланд); iframe дотор (хичээлд суулгасан хуудас) давхардуулахгүй.
if (window.self === window.top) mountA11y();

window.SG = { THREAD_EVENTS, panelIn, mountClouds, mountBoards, openBoard, pay: payWindow, rubricView: rubricViewHTML, icon, embedDoc, ChatThread, pdfLib, BookReader, richHTML, blocksHTML, mountQuizzes, fmtBytes, extOf, $, $$, esc, api, apiList, Auth, toast, money, fmtDate, fmtTime, Live, mediaHTML, book3dHTML, hydrateBooks, Flipbook, openModal, closeModal, msgHTML, linkify, celebrate, reveals, counters, avatarHTML, ringHTML, hueOfName, fmtDay, WEEKDAYS, WEEKDAYS_SHORT };
// Хөдөлгөөнийг цөөлсөн: хазайлт, соронзон товч, курсор дагасан гэрэл, нээлтийн хөшиг ашиглахгүй.
splitText(); reveals(); counters(); navScroll(); ripples(); authNav(); chatRail(); hydrateBooks();
if (page === "me") { // өөрийн хуудас руу: багш профайл, суралцагч нүүр; нэвтрээгүй бол нэвтрэх
  const u = Auth.user, h = location.hash;
  if (!Auth.token) location.replace("/login?next=" + encodeURIComponent("/me" + h));
  else if (u?.role === "teacher" && u.username) location.replace("/t/" + u.username + h);
  else api("/api/me").then((me) => { Auth.set(Auth.token, me); location.replace(me.role === "teacher" ? "/t/" + me.username + h : "/" + h); }).catch(() => location.replace("/login"));
}
// Интро (Three.js робот): анх орох үед нэг удаа, эсвэл ?intro=1. WebGL байхгүй, хөдөлгөөн багасгах тохиргоотой бол гарахгүй.
if (page === "home") {
  const force = new URLSearchParams(location.search).has("intro");
  let seen = false; try { seen = !!localStorage.getItem("sg_intro"); } catch {}
  const gl = (() => { try { return !!document.createElement("canvas").getContext("webgl2"); } catch { return false; } })();
  if ((force || !seen) && gl && !matchMedia("(prefers-reduced-motion: reduce)").matches) {
    try { localStorage.setItem("sg_intro", "1"); } catch {}
    const ver = new URL($("script[src*='app.js']").src).searchParams.get("v") || "";
    import(`/static/intro.js?v=${ver}`).then((m) => m.runIntro()).catch(() => document.documentElement.classList.remove("intro-on"));
  }
}
// Сахилга батын хуурамч цонх: анхааралтай → таб сольсон (1/3) → хөндийрсөн (2/3) → хаагдсан (3/3) → багшид мэдэгдэл.
function startDemo(el, still) {
  const att = $(".hd-att", el), dots = $$(".hd-strikes i", el), cap = $(".hd-cap", el);
  const ATT = [96, 71, 48, 12, 12], HITS = [0, 1, 2, 3, 3], HOLD = [2200, 2500, 2500, 2700, 3000];
  const CAP = ["Анхааралтай үзэж байна — идэвхтэй минут тоологдоно", "Таб сольсон — улаан дохио, дуут сануулга · 1/3", "Хичээлээс хөндийрсөн — курсор гарсан · 2/3",
    "Хичээл түр хаагдлаа — 3/3, багшийн тогтоосон хугацаагаар", "Багшид мэдэгдэл очлоо — шалгаад дахин нээж болно"];
  let s = 0, t = 0, shown = 96;
  const set = (n) => {
    s = n; el.dataset.s = n; dots.forEach((d, i) => d.classList.toggle("hit", i < HITS[n]));
    el.style.setProperty("--att", ATT[n] / 100); if (cap) cap.textContent = `${n + 1}/5 · ${CAP[n]}`;
    const from = shown, t0 = performance.now(); shown = ATT[n];
    const step = (x) => { const p = Math.min(1, (x - t0) / 600); att.textContent = Math.round(from + (ATT[n] - from) * p) + "%"; if (p < 1) requestAnimationFrame(step); };
    requestAnimationFrame(step);
  };
  if (still) { set(3); return () => {}; }
  const loop = () => { t = setTimeout(() => { set((s + 1) % 5); loop(); }, HOLD[s]); };
  set(0); loop();
  return () => clearTimeout(t);
}

// Нүүр: «surgalt.mn хэрхэн ажилладаг вэ?» — нэгтгэсэн хэсэг. Зүүнд оюуны зураглал (төв + 8 салбар; сонгосон салбарын дэд
// зангилаа гадагшаа гарна), баруунд дэлгэрэнгүй самбар (хөдөлгөөнт жишээ). Горим: бүтэц, суралцагчийн/багшийн аялал (алхам
// бүрт холбогдох салбар тодорно). Өөрөө аялуулж тоглоно; хэрэглэгч сонговол зогсоно (▶ дахин); дэлгэцээс гарвал түр зогсоно.
// #explore, #explore=stu, #explore=tch холбоосоор шууд очно.
function explore() {
  const ex = $("#explore"); if (!ex) return;
  const still = matchMedia("(prefers-reduced-motion: reduce)").matches;
  const map = $(".ex-map", ex), svg = $(".ex-wires", ex), leaves = $(".ex-leaves", ex), core = $(".ex-core", ex), brs = $$(".ex-br", ex);
  const panel = $(".ex-panel", ex), chips = $(".ex-chips", ex), bar = $(".ex-prog i", ex), play = $(".ex-play", ex), tabs = $$(".ex-modes [data-mode]", ex);
  const tpl = { map: ["core", ...brs.map((b) => b.dataset.node)].map((n) => $(`template[data-node="${n}"]`, ex)), stu: $$('template[data-step="stu"]', ex), tch: $$('template[data-step="tch"]', ex) };
  const HOLD = 6500;
  let mode = "map", i = 0, timer = 0, stopDemo = null, inView = false, playing = !still;
  const sel = () => (mode === "map" ? (i === 0 ? "core" : String(i - 1)) : tpl[mode][i].dataset.branch);
  const geo = () => { const W = map.clientWidth, H = map.clientHeight; return { W, H, cx: W / 2, cy: H / 2, rx: Math.min(175, W / 2 - 160), ry: Math.min(170, H / 2 - 120) }; };
  const layout = () => {
    const g = geo(); if (!g.W) return;
    core.style.left = g.cx + "px"; core.style.top = g.cy + "px";
    let w = "";
    brs.forEach((b, k) => {
      const a = (-90 + k * 45) * Math.PI / 180, x = g.cx + g.rx * Math.cos(a), y = g.cy + g.ry * Math.sin(a);
      Object.assign(b.dataset, { a, x, y }); b.style.left = x + "px"; b.style.top = y + "px";
      w += `<path class="ex-w" data-b="${k}" pathLength="1" d="M${g.cx},${g.cy} Q${(g.cx + x) / 2 + (y - g.cy) * 0.12},${(g.cy + y) / 2 - (x - g.cx) * 0.12} ${x},${y}"/>`;
    });
    svg.setAttribute("viewBox", `0 0 ${g.W} ${g.H}`); svg.innerHTML = w + `<g class="ex-lw"></g>`;
    mark();
  };
  const mark = () => {
    const s = sel(), b = brs.find((x) => x.dataset.node === s), lw = $(".ex-lw", svg);
    core.classList.toggle("on", s === "core"); brs.forEach((x) => x.classList.toggle("on", x === b));
    $$(".ex-w[data-b]", svg).forEach((p) => p.classList.toggle("on", p.dataset.b === s));
    leaves.innerHTML = ""; if (lw) lw.innerHTML = "";
    const g = geo(); if (!b || !g.W || !lw) return;
    // Дэд зангилаа салбарын гадна талд: хэвтээ чиглэлд хол, босоо дагуу нягт; босоо чиглэлд ойр, хэвтээ дагуу сийрэг.
    const L = b.dataset.leaves.split("|"), bx = +b.dataset.x, by = +b.dataset.y, a0 = +b.dataset.a, c = Math.abs(Math.cos(a0)), sn = Math.abs(Math.sin(a0));
    const out = 112 * c + 54 * sn, gap = 36 * c + 100 * sn;
    L.forEach((t, k) => {
      const o = (k - (L.length - 1) / 2) * gap, x = bx + Math.cos(a0) * out - Math.sin(a0) * o, y = by + Math.sin(a0) * out + Math.cos(a0) * o;
      leaves.insertAdjacentHTML("beforeend", `<span class="ex-leaf" style="left:${x}px;top:${y}px;--d:${k * 0.08}s">${esc(t)}</span>`);
      lw.insertAdjacentHTML("beforeend", `<path class="ex-w leaf" pathLength="1" style="--d:${k * 0.08}s" d="M${bx},${by} L${x},${y}"/>`);
    });
    requestAnimationFrame(() => requestAnimationFrame(() => { $$(".ex-leaf", leaves).forEach((x) => x.classList.add("in")); $$(".leaf", lw).forEach((x) => x.classList.add("in")); }));
  };
  const bars = () => { bar.style.animation = "none"; void bar.offsetWidth; bar.style.animation = playing && inView ? `ex-bar ${HOLD}ms linear forwards` : "none"; };
  const tick = () => { clearTimeout(timer); bars(); if (playing && inView) timer = setTimeout(() => go(i + 1), HOLD); };
  const show = () => {
    stopDemo?.(); stopDemo = null;
    panel.replaceChildren(tpl[mode][i].content.cloneNode(true));
    panel.classList.remove("swap"); void panel.offsetWidth; panel.classList.add("swap");
    const d = $(".how-demo", panel); if (d && inView) stopDemo = startDemo(d, still);
    // Идэвхтэй товчийг зөвхөн хэвтээ жагсаалт дотор харуулна (scrollIntoView бүх хуудсыг босоо гүйлгэдэг байсан).
    $$("button", chips).forEach((c, k) => { c.setAttribute("aria-selected", String(k === i)); if (k === i && chips.scrollWidth > chips.clientWidth) chips.scrollTo({ left: c.offsetLeft - (chips.clientWidth - c.offsetWidth) / 2, behavior: "smooth" }); });
    mark();
  };
  const go = (n) => { const len = tpl[mode].length; i = ((n % len) + len) % len; show(); tick(); };
  const setPlay = (on) => { playing = on && !still; play.textContent = playing ? "❚❚" : "▶"; play.setAttribute("aria-label", playing ? "Түр зогсоох" : "Тоглуулах"); tick(); };
  const setMode = (m, n = 0) => {
    mode = m; ex.dataset.mode = m;
    tabs.forEach((t) => t.setAttribute("aria-selected", String(t.dataset.mode === m)));
    const names = m === "map" ? ["surgalt.mn", ...brs.map((b) => b.textContent.trim())] : tpl[m].map((t, k) => `${k + 1}. ${$("h3", t.content).textContent}`);
    chips.innerHTML = names.map((x, k) => `<button type="button" role="tab" data-i="${k}">${esc(x)}</button>`).join("");
    go(n);
  };
  tabs.forEach((t) => t.addEventListener("click", () => { setMode(t.dataset.mode); setPlay(true); }));
  chips.addEventListener("click", (e) => { const c = e.target.closest("[data-i]"); if (c) { setPlay(false); go(+c.dataset.i); } });
  const pickNode = (n) => { setPlay(false); if (mode !== "map") setMode("map", n); else go(n); };
  core.addEventListener("click", () => pickNode(0));
  brs.forEach((b, k) => { b.addEventListener("click", () => pickNode(k + 1)); b.addEventListener("mouseenter", () => { if (mode === "map") pickNode(k + 1); }); });
  $(".ex-prev", ex).onclick = () => { setPlay(false); go(i - 1); };
  $(".ex-next", ex).onclick = () => { setPlay(false); go(i + 1); };
  play.onclick = () => setPlay(!playing);
  new ResizeObserver(layout).observe(map);
  new IntersectionObserver(([e]) => {
    inView = e.isIntersecting; ex.classList.toggle("run", inView); if (inView) ex.classList.add("seen");
    stopDemo?.(); stopDemo = null;
    const d = $(".how-demo", panel); if (d && inView) stopDemo = startDemo(d, still);
    tick();
  }, { threshold: 0.25 }).observe(ex);
  // Нүүрнээс шууд очих: #explore, #explore=stu, #explore=tch
  const fromHash = () => {
    const m = location.hash.match(/^#explore(?:=(stu|tch|map))?$/); if (!m) return;
    setMode(m[1] || "map"); setPlay(true);
    ex.scrollIntoView({ behavior: still ? "auto" : "smooth", block: "start" });
  };
  setMode("map"); setPlay(!still); layout();
  addEventListener("hashchange", fromHash); fromHash();
}
/* ---------- Нүүр: Шилдэг багш нар, Идэвхтэй суралцагчид — зэрэглэл × 10, гүйдэг карт (сум, чирэх, автоматаар) ---------- */
const TT_BOARDS = [ // [түлхүүр, дүрс, таб, нэгж, тайлбар]
  ["views", "eye", "Хамгийн их хандалттай", "үзэлт", "Профайл нь хамгийн олон удаа үзэгдсэн"],
  ["students", "users", "Хамгийн их авагдсан", "элсэлт", "Сургалтыг нь хамгийн олон суралцагч авсан"],
  ["courses", "courses", "Хамгийн олон сургалттай", "сургалт", "Хамгийн олон сургалт нийтэлсэн"],
  ["talks", "chat", "Хамгийн идэвхтэй харилцаа", "хүнтэй харилцсан", "Хамгийн олон хүнтэй чат, сэтгэгдлээр харилцсан"],
];
const TS_BOARDS = [
  ["courses", "courses", "Олон сургалтад суралцсан", "сургалт", "Хамгийн олон сургалтад идэвхтэй суралцсан"],
  ["rank", "medal", "Өндөр цолтой", "оноо", "Цолын оноо хамгийн өндөр"],
  ["creative", "edit", "Бүтээлч суралцагч", "бүтээл", "Хичээлийн дүгнэлт, даалгавар, үүлэн үг, тодруулга хамгийн их"],
  ["discipline", "check", "Сахилга баттай", "мин сануулгагүй", "Сануулга авалгүй хамгийн удаан идэвхтэй суралцсан"],
  ["talks", "chat", "Харилцаа сайтай", "мессеж, сэтгэгдэл", "Чат, хичээлийн сэтгэгдлээр хамгийн идэвхтэй харилцсан"],
];
const ttNum = (v) => Number(v || 0).toLocaleString("en-US");
const TT_CROWN = `<svg viewBox="0 0 24 24" width="15" height="15" aria-hidden="true"><path fill="currentColor" d="M3 8l4.5 3.5L12 5l4.5 6.5L21 8l-2 10H5z"/></svg>`;
const ttRank = (x) => `<span class="tt-rank" aria-hidden="true">${x.rank === 1 ? TT_CROWN : ""}${x.rank}</span>`;
const ttBar = (x, top) => `<span class="tt-bar" aria-hidden="true"><i style="--w:${top ? Math.max(6, Math.round((x.value / top) * 100)) : 0}%"></i></span>`;
const ttCls = (x) => `tt-card${x.rank <= 3 ? ` tt-m${x.rank}` : ""}`;
// Урьдчилан харах (?demo=1): бодит өгөгдөл цөөн үед загварыг харуулах жишээ — «Жишээ өгөгдөл» гэж тэмдэглэж, холбоосгүй.
const TT_NAMES = [["Б.Энхжаргал", "Математикийн багш · ЭЕШ бэлтгэл"], ["Д.Тэмүүлэн", "Англи хэл · IELTS 8.0"], ["Г.Сарангэрэл", "Физик, олимпиадын дасгалжуулагч"],
  ["Ц.Мөнхбат", "Програмчлал · Python, JavaScript"], ["О.Ариунаа", "Хими · ЭЕШ 800 оноо"], ["Н.Батбаяр", "Монгол хэл, уран зохиол"], ["Э.Номин", "Биологийн багш"],
  ["С.Ганзориг", "Нягтлан бодох бүртгэл"], ["Л.Оюунчимэг", "Япон хэл · JLPT N2"], ["Ж.Төгөлдөр", "Гитарын хичээл"], ["Х.Солонго", "Зураг, дизайн · Figma"], ["П.Баярсайхан", "Түүх, нийгэм судлал"]];
const TS_NAMES = ["Анужин Б.", "Тэмүүжин Г.", "Мишээл Д.", "Билгүүн Э.", "Номуун С.", "Хулан Ц.", "Ананд О.", "Сувд Л.", "Эрдэнэ Н.", "Золбоо Х.", "Маралаа Ж.", "Тэлмүүн П."];
function ttDemo(kind) {
  const ppl = kind === "students"
    ? TS_NAMES.map((name, i) => ({ key: "", name, avatar_url: "", courses: 9 - ((i * 4) % 8), points: 4200 - ((i * 977) % 3900), creative: 160 - ((i * 37) % 140),
      discipline: 1800 - ((i * 431) % 1600), active_min: 2000 - ((i * 389) % 1500), warnings: (i * 3) % 5, talks: 240 - ((i * 59) % 220) }))
    : TT_NAMES.map(([display_name, headline], i) => ({ username: "", display_name, headline, subjects: [], avatar_url: "",
      views: 4800 - ((i * 1373) % 4300), enrolls: 620 - ((i * 211) % 560), students: 540 - ((i * 197) % 500), courses: 12 - ((i * 5) % 11), talks: 380 - ((i * 149) % 350) }));
  if (kind === "students") ppl.forEach((x) => { const st = (RANK_STEPS.filter((s) => x.points >= s[0]).pop() || RANK_STEPS[0]); x.level = st[1]; x.insignia = st[2]; });
  const board = (f) => ppl.map((x) => ({ ...x, value: x[f] })).sort((a, b) => b.value - a.value).slice(0, 10).map((x, i) => ({ ...x, rank: i + 1 }));
  return kind === "students"
    ? { boards: { courses: board("courses"), rank: board("points"), creative: board("creative"), discipline: board("discipline"), talks: board("talks") } }
    : { boards: { views: board("views"), students: board("enrolls"), courses: board("courses"), talks: board("talks") } };
}
const RANK_STEPS = [[0, "Шинэ цэрэг", "·"], [60, "Байлдагч", "▪"], [150, "Ахлах байлдагч", "▪▪"], [260, "Дэд түрүүч", "▾"], [400, "Түрүүч", "▾▾"], [560, "Ахлах түрүүч", "▾▾▾"],
  [750, "Сургагч ахлагч", "◆"], [980, "Дэд дэслэгч", "★"], [1250, "Дэслэгч", "★★"], [1560, "Ахлах дэслэгч", "★★★"], [1900, "Ахмад", "★★★★"], [2300, "Хошууч", "✦"],
  [2750, "Дэд хурандаа", "✦✦"], [3300, "Хурандаа", "✦✦✦"], [4000, "Бригадын генерал", "✪"], [4800, "Хошууч генерал", "✪✪"], [5700, "Дэслэгч генерал", "✪✪✪"], [6800, "Генерал", "✪✪✪✪"]];

// leaderboard — таб, гүйдэг карт, сум, цэг, автоматаар шилжих (хулгана/фокус/хүрэлт үед зогсоно).
function leaderboard(root, data, defs, card, demo) {
  const boards = defs.filter(([k]) => data.boards?.[k]?.length);
  if (!boards.length) return false;
  root.hidden = false;
  if (demo) {
    root.classList.add("tt-demo");
    $(".tt-head .eyebrow", root).insertAdjacentHTML("afterend", `<span class="tt-demo-tag">Жишээ өгөгдөл — бодит зэрэглэл биш</span>`);
    root.addEventListener("click", (e) => { if (e.target.closest(".tt-card a")) e.preventDefault(); });
  }
  const tabs = $(".tt-tabs", root), track = $(".tt-track", root), dots = $(".tt-dots", root), prev = $(".tt-prev", root), next = $(".tt-next", root);
  const still = matchMedia("(prefers-reduced-motion: reduce)").matches;
  let cur = boards[0][0], timer = 0, paused = false;
  tabs.innerHTML = boards.map(([k, ic, t]) => `<button type="button" role="tab" data-tt="${k}" aria-selected="false">${icon(ic, 17)}<span>${t}</span></button>`).join("");
  const pages = () => Math.max(1, Math.ceil((track.scrollWidth - 4) / track.clientWidth));
  const syncNav = () => {
    const max = track.scrollWidth - track.clientWidth - 2;
    prev.disabled = track.scrollLeft <= 2; next.disabled = track.scrollLeft >= max;
    const p = pages(), at = Math.round(track.scrollLeft / Math.max(1, track.clientWidth));
    dots.innerHTML = p > 1 ? Array.from({ length: p }, (_, i) => `<i class="${i === Math.min(at, p - 1) ? "on" : ""}"></i>`).join("") : "";
  };
  const show = (k) => {
    cur = k;
    const b = defs.find((x) => x[0] === k), list = data.boards[k], top = list[0]?.value || 0;
    $$("[data-tt]", tabs).forEach((t) => t.setAttribute("aria-selected", String(t.dataset.tt === k)));
    track.setAttribute("aria-label", `${b[2]} — ${b[4]}`);
    track.innerHTML = list.map((x) => card(x, top, b)).join("");
    track.classList.remove("in"); void track.offsetWidth; track.classList.add("in");
    track.scrollLeft = 0; syncNav();
  };
  const step = (dir) => {
    const w = track.querySelector(".tt-card")?.getBoundingClientRect().width || 240;
    const max = track.scrollWidth - track.clientWidth - 2;
    if (dir > 0 && track.scrollLeft >= max) return track.scrollTo({ left: 0, behavior: still ? "auto" : "smooth" });
    track.scrollBy({ left: dir * (w + 16) * Math.max(1, Math.floor(track.clientWidth / (w + 16))), behavior: still ? "auto" : "smooth" });
  };
  const restart = () => { clearInterval(timer); if (!still) timer = setInterval(() => { if (!paused && !document.hidden && root.getBoundingClientRect().top < innerHeight) step(1); }, 6000); };
  tabs.addEventListener("click", (e) => { const t = e.target.closest("[data-tt]"); if (t) { show(t.dataset.tt); restart(); } });
  tabs.addEventListener("keydown", (e) => { // сумаар таб солих
    if (!["ArrowLeft", "ArrowRight"].includes(e.key)) return;
    const ks = boards.map((x) => x[0]), i = (ks.indexOf(cur) + (e.key === "ArrowRight" ? 1 : ks.length - 1)) % ks.length;
    show(ks[i]); $(`[data-tt="${ks[i]}"]`, tabs).focus(); e.preventDefault();
  });
  prev.onclick = () => { step(-1); restart(); };
  next.onclick = () => { step(1); restart(); };
  track.addEventListener("scroll", () => requestAnimationFrame(syncNav), { passive: true });
  addEventListener("resize", syncNav);
  root.addEventListener("pointerenter", () => (paused = true)); root.addEventListener("pointerleave", () => (paused = false));
  root.addEventListener("focusin", () => (paused = true)); root.addEventListener("focusout", () => (paused = false));
  track.addEventListener("touchstart", () => (paused = true), { passive: true });
  show(cur); restart();
  root._show = () => show(cur);
  return true;
}

async function topTeachers() {
  const root = $("#topTeachers"); if (!root) return;
  const demo = new URLSearchParams(location.search).has("demo");
  let data;
  if (demo) data = ttDemo("teachers");
  else try { data = await api("/api/top-teachers", { token: null }); } catch { return; }
  leaderboard(root, data, TT_BOARDS, (x, top, b) => `<li class="${ttCls(x)}" style="--d:${(x.rank - 1) * 45}ms"><a class="tt-in" href="${x.username ? "/t/" + encodeURIComponent(x.username) : "#topTeachers"}" aria-label="${x.rank}-р байр: ${esc(x.display_name)}, ${ttNum(x.value)} ${b[3]}">
      ${ttRank(x)}${avatarHTML(x, "tt-av")}
      <b class="tt-name">${esc(x.display_name || x.username)}</b>
      <small class="tt-line">${esc(x.headline || x.subjects.join(" · ") || "Багш")}</small>
      <span class="tt-metric"><b>${ttNum(x.value)}</b><small>${b[3]}</small></span>${ttBar(x, top)}
      <span class="tt-stats">${x.courses ? `<span>${icon("courses", 14)}${ttNum(x.courses)}</span>` : ""}${x.students ? `<span>${icon("users", 14)}${ttNum(x.students)}</span>` : ""}${x.views ? `<span>${icon("eye", 14)}${ttNum(x.views)}</span>` : ""}</span>
      <span class="tt-go">Профайл үзэх ${icon("next", 14)}</span></a></li>`, demo);
}

// Идэвхтэй суралцагчид — нэрийг товчилж, холбоосгүй; өөрийн карт дээр «Нуух», нуусан бол «Дахин харуулах».
async function topStudents() {
  const root = $("#topStudents"); if (!root) return;
  const demo = new URLSearchParams(location.search).has("demo");
  let data;
  if (demo) data = ttDemo("students");
  else try { data = await api("/api/top-students", { token: null }); } catch { return; }
  const meKey = !demo && Auth.user?.rank_key;
  const shown = leaderboard(root, data, TS_BOARDS, (x, top, b) => {
    const mine = meKey && x.key === meKey;
    return `<li class="${ttCls(x)}${mine ? " tt-me" : ""}" style="--d:${(x.rank - 1) * 45}ms"><div class="tt-in" role="group" aria-label="${x.rank}-р байр: ${esc(x.name)}, ${ttNum(x.value)} ${b[3]}">
      ${ttRank(x)}${mine ? `<span class="tt-you">Та</span>` : ""}${avatarHTML({ display_name: x.name, username: x.key || x.name, avatar_url: x.avatar_url }, "tt-av")}
      <b class="tt-name">${esc(x.name)}</b>
      <small class="tt-line tt-lv"><span class="tt-ins" aria-hidden="true">${esc(x.insignia || "")}</span>${esc(x.level || "")}</small>
      <span class="tt-metric"><b>${ttNum(x.value)}</b><small>${b[3]}</small></span>${ttBar(x, top)}
      <span class="tt-stats">${x.courses ? `<span title="Сургалт">${icon("courses", 14)}${ttNum(x.courses)}</span>` : ""}${x.creative ? `<span title="Бүтээл">${icon("edit", 14)}${ttNum(x.creative)}</span>` : ""}${x.active_min ? `<span title="Идэвхтэй минут">${icon("clock", 14)}${ttNum(x.active_min)}</span>` : ""}${b[0] === "discipline" ? `<span title="Сануулга" class="${x.warnings ? "tt-warn" : "tt-ok"}">${icon(x.warnings ? "alert" : "check", 14)}${ttNum(x.warnings)}</span>` : ""}</span>
      ${mine ? `<button type="button" class="tt-hide" data-rank-hide>Жагсаалтаас нуух</button>` : ""}</div></li>`;
  }, demo);
  if (!Auth.user || demo) return;
  const foot = document.createElement("p");
  foot.className = "tt-foot";
  const drawFoot = () => { foot.innerHTML = Auth.user?.rank_hidden ? `Та өөрийгөө энэ жагсаалтаас нуусан байна. <button type="button" class="linkish" data-rank-show>Дахин харуулах</button>` : ""; foot.hidden = !Auth.user?.rank_hidden; };
  root.append(foot); drawFoot();
  root.addEventListener("click", async (e) => {
    const hide = e.target.closest("[data-rank-hide]"), showMe = e.target.closest("[data-rank-show]");
    if (!hide && !showMe) return;
    if (hide && !confirm("Нүүр хуудасны «Идэвхтэй суралцагчид» жагсаалтад таныг харуулахгүй. Үргэлжлүүлэх үү?")) return;
    try {
      const me = await api("/api/me/rank-hidden", { method: "PUT", body: { hidden: !!hide } });
      Auth.set(Auth.token, me);
      toast(hide ? "Таныг жагсаалтаас нууллаа" : "Та жагсаалтад дахин харагдана (хэдэн минутын дараа)");
      const fresh = await api("/api/top-students", { token: null }).catch(() => null);
      if (fresh) { data.boards = fresh.boards; root._show?.(); }
      drawFoot();
    } catch (x) { toast(x.message, true); }
  });
  if (!shown && Auth.user?.rank_hidden) root.hidden = false;
}

if (page === "home") { homePage(); explore(); topTeachers(); topStudents(); }
if (page === "profile") profilePage();
if (page === "profile" || page === "course") chatWidget();
if (page === "course") coursePage();
if (page === "login") loginPage();
if (page === "home") findCourses();
// 404: «Буцах» — сайтын өөр хуудаснаас ирсэн бол буцна, гаднаас (буруу холбоос) ирсэн бол нүүр хуудас руу.
if (page === "error") $("[data-nf-back]")?.addEventListener("click", () => {
  if (history.length > 1 && document.referrer.startsWith(location.origin)) history.back(); else location.href = "/";
});
// Үзэлт: хуудас нээгдэхэд бүртгэж, хэр удаан (идэвхтэй, харагдаж байсан) байсныг гарахдаа илгээнэ.
// Багш өөрийнхийгөө үзвэл сервер тоолохгүй. Давтагдаагүй зочныг хөтчийн санамсаргүй ID-гаар ялгана.
{
  const v = page === "profile" ? ["profile", $("main.pf")?.dataset.teacher] : page === "course" ? ["course", $("main.course-page")?.dataset.course] : null;
  if (v?.[1]) {
    let vid = ""; try { vid = localStorage.getItem("sg_vid") || ""; if (!vid) { vid = crypto.randomUUID?.() || Math.random().toString(36).slice(2) + Date.now().toString(36); localStorage.setItem("sg_vid", vid); } } catch {}
    let ref = ""; try { if (document.referrer) { const u = new URL(document.referrer); ref = u.host === location.host ? "self" : u.host; } } catch {}
    api("/api/views", { method: "POST", body: { kind: v[0], id: v[1], vid, ref } }).then((d) => {
      if (!d?.visit) return;
      let acc = 0, since = document.hidden ? 0 : Date.now(), sent = 0;
      const flush = () => {
        if (since) { acc += Date.now() - since; since = document.hidden ? 0 : Date.now(); }
        const sec = Math.round(acc / 1000); if (sec <= sent) return; sent = sec;
        const url = `/api/views/${d.visit}/end`, body = JSON.stringify({ t: d.t, seconds: sec });
        if (!navigator.sendBeacon?.(url, new Blob([body], { type: "application/json" }))) fetch(url, { method: "POST", body, headers: { "Content-Type": "application/json" }, keepalive: true }).catch(() => {});
      };
      document.addEventListener("visibilitychange", () => { if (document.hidden) flush(); else since = Date.now(); });
      addEventListener("pagehide", flush);
      setInterval(() => { if (!document.hidden) flush(); }, 60000); // урт айлчлалыг минут тутам хадгална
    }).catch(() => {});
  }
}
if (page === "book") bookPage();
if (page === "legal") legalPage();
if (new URLSearchParams(location.search).has("deleted")) { // бүртгэлээ устгасны дараа
  toast("Бүртгэл тань устгагдлаа. surgalt.mn-ийг ашигласанд баярлалаа.");
  history.replaceState(null, "", location.pathname + location.hash);
}
if (page === "reset") resetPage();
animateIcons();
siteHit();
if (page === "home" && !document.documentElement.classList.contains("authed")) { heroStars(); homeFilm(); }
// 3D дэвсгэр (Three.js): нүүрэнд зөвхөн зочинд, нэвтрэх хуудсанд — хуудас ачаалагдсаны дараа сул үед.
if ((page === "home" && !document.documentElement.classList.contains("authed")) || page === "login") {
  (window.requestIdleCallback || ((f) => setTimeout(f, 400)))(() => {
    const s = document.createElement("script"), v = document.querySelector('script[src*="/static/app.js"]')?.src.split("v=")[1] || "";
    s.src = "/static/herofx.js?v=" + v; s.async = true; document.head.append(s);
  }, { timeout: 1500 });
}
// Хуудас угсрагдаж (фонт, эхний зурлага) дууссаны дараа зөөлөн гаргана — салж нийлэх нь харагдахгүй.
Promise.race([document.fonts?.ready || Promise.resolve(), new Promise((r) => setTimeout(r, 600))])
  .then(() => requestAnimationFrame(() => requestAnimationFrame(() => document.documentElement.classList.remove("pre"))));
})();

/* ---------- Хөдөлгөөнт дүрс (AnimateIcons · Lucide, MIT, өөрийн серверээс): ----------
   SVG дүрс (data-ico) эхэлж харагдана; сан ачаалагдмагц хөдөлгөөнт хувилбараар солигдож, эцэг товч, холбоос,
   карт дээр хулгана авчрах / focus авахад тоглоно. Хөдөлгөөн багасгах тохиргоотой үед огт ачаалахгүй. */
function animateIcons() {
  if (matchMedia("(prefers-reduced-motion: reduce)").matches) return;
  const MAP = { bell: "Bell", chat: "MessageCircle", book: "BookOpen", users: "Users", user: "User", teacher: "User", profile: "User", gear: "Settings", eye: "Eye", lock: "Lock", check: "Check", clock: "Clock", money: "Wallet", chart: "ChartColumn", live: "Video", clip: "Paperclip", medal: "Medal", courses: "GraduationCap", cal: "Calendar", pin: "MapPin", shield: "ShieldCheck", search: "Search", home: "House", plus: "Plus", x: "X", edit: "SquarePen", trash: "Trash2", heart: "Heart", star: "Star", cloud: "Cloud", link: "Link", files: "Files", doc: "FileText", exam: "ClipboardCheck", play: "Play", text: "Type", image: "Image", mic: "Mic", heading: "Heading", send: "Send", back: "ArrowLeft", next: "ArrowRight", chev: "ChevronRight", chevron: "ChevronRight", down: "ChevronDown", up: "ChevronUp", download: "Download", upload: "Upload", share: "Share", qr: "QrCode", globe: "Globe", card: "CreditCard", slides: "Presentation", sheet: "Table", expand: "Expand", grip: "GripVertical", list: "List", quiz: "MessageCircleQuestionMark", target: "Target", pulse: "Activity", alert: "TriangleAlert", camera: "Camera", phone: "Phone", layers: "Layers", work: "Briefcase", repeat: "Repeat", ext: "ExternalLink", out: "LogOut", group: "Users", people: "Users", collapse: "ChevronsRight", monitor: "Monitor", playc: "CirclePlay", headphones: "Headphones", folder: "Folder" };
  const v = document.querySelector('script[src*="/static/app.js"]')?.src.split("v=")[1] || "";
  const s = document.createElement("script");
  s.src = "/static/vendor/animateicons-lucide-0.1.0.js?v=" + v; s.async = true;
  s.onload = () => {
    const A = window.AnimateIcons; if (!A?.createIcon) return;
    const hostOf = (el) => el.closest("a,button,summary,label,[role=tab],[role=button],.rail-item,.pv-card,.card,li") || el.parentElement;
    const swap = (svg) => {
      svg.dataset.ai = "1";
      const data = A.icons[MAP[svg.dataset.ico]]; if (!data) return;
      let el; try { el = A.createIcon(data, { size: +svg.getAttribute("width") || 18, isAnimated: false }); } catch { return; }
      el.classList.add("ai-ico"); el.setAttribute("aria-hidden", "true");
      const sv = el.querySelector("svg");
      if (sv) { sv.setAttribute("stroke-width", svg.getAttribute("stroke-width") || "1.9"); sv.dataset.ico = svg.dataset.ico; sv.dataset.ai = "1"; svg.classList.forEach((c) => sv.classList.add(c)); }
      svg.replaceWith(el);
      const host = hostOf(el); if (!host) return;
      (host._ai ||= []).push(el);
      if (host._aiOn) return; host._aiOn = 1;
      const go = () => host._ai.forEach((i) => i.isConnected && i.startAnimation?.()), stop = () => host._ai.forEach((i) => i.stopAnimation?.());
      host.addEventListener("pointerenter", go); host.addEventListener("focusin", go);
      host.addEventListener("pointerleave", stop); host.addEventListener("focusout", stop);
    };
    const scan = (root) => root.querySelectorAll?.("svg[data-ico]:not([data-ai])").forEach(swap);
    scan(document);
    const q = new Set(); let t = 0;
    new MutationObserver((ms) => {
      for (const m of ms) m.addedNodes.forEach((n) => n.nodeType === 1 && q.add(n));
      if (!t) t = requestAnimationFrame(() => { t = 0; q.forEach((n) => (n.matches?.("svg[data-ico]:not([data-ai])") ? n.isConnected && swap(n) : scan(n))); q.clear(); });
    }).observe(document.body, { childList: true, subtree: true });
  };
  (window.requestIdleCallback || ((f) => setTimeout(f, 300)))(() => document.head.append(s), { timeout: 1500 });
}
