/* Surgalt — интерактив давхарга (build алхамгүй, цэвэр JS) */
(() => {
"use strict";
const $ = (s, r = document) => r.querySelector(s);
const $$ = (s, r = document) => [...r.querySelectorAll(s)];
const reduce = matchMedia("(prefers-reduced-motion: reduce)").matches;
const page = document.body.dataset.page;
const esc = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
const money = (n) => (n ? Number(n).toLocaleString("en-US") + "₮" : "Үнэгүй");
// Огноог өөрсдөө хэлбэржүүлнэ: олон браузерт mn-MN локаль байхгүй тул англиар гардаг.
const WEEKDAYS = ["Ням", "Даваа", "Мягмар", "Лхагва", "Пүрэв", "Баасан", "Бямба"];
const WEEKDAYS_SHORT = ["Ня", "Да", "Мя", "Лх", "Пү", "Ба", "Бя"];
const pad2 = (n) => String(n).padStart(2, "0");
const fmtTime = (t) => { const d = new Date(t); return pad2(d.getHours()) + ":" + pad2(d.getMinutes()); };
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
  };
  document.documentElement.classList.add("has-rail");
  document.body.insertAdjacentHTML("beforeend", `<aside class="rail" id="studioRail" aria-label="Чат">
    <div class="rail-view" id="railHome">
      <header class="rail-head"><strong><i></i>Чат</strong><button class="btn btn-sm rail-group-btn" id="railNewGroup" title="${teacher ? "Сургалтын бүлэг чат нээх" : "Ангийн найзуудтайгаа бүлэг үүсгэх"}">＋ Бүлэг чат</button><button class="icon-btn rail-close" data-rail-close aria-label="Чат хумих" title="Хумих">»</button></header>
      <label class="rail-search">${I.search}<input type="search" id="railSearch" placeholder="Нэрээр хайх…" aria-label="Чат хайх"></label>
      <div class="rail-list" id="railList">${[1, 2, 3, 4].map(() => `<div class="rail-skel"><i></i><span><b></b><b></b></span></div>`).join("")}</div></div>
    <div class="rail-view" id="railThread" hidden>
      <header class="rail-head"><button class="icon-btn" id="railBack" aria-label="Жагсаалт руу буцах">${I.back}</button><div class="grow" id="railWho"></div><button class="icon-btn rail-close" data-rail-close aria-label="Чат хаах">${I.x}</button></header>
      <div class="chat-body" id="railBody"><ol class="chat-msgs" id="railMsgs"></ol></div>
      <form class="chat-input" id="railForm">${teacher ? `<button type="button" id="railMeet" data-plus>${I.live} Google Meet үүсгээд илгээх</button>` : ""}</form></div>
  </aside><div class="rail-scrim" id="railScrim"></div>`);
  $(".nav-links").insertAdjacentHTML("afterbegin", `<button class="icon-btn rail-toggle" id="railToggle" aria-label="Чат" aria-controls="studioRail" aria-expanded="false">${I.chat}<span class="bell-badge" id="railBadge" hidden></span></button>`);
  const rail = $("#studioRail"), home = $("#railHome"), thread = $("#railThread"), msgs = $("#railMsgs"), body = $("#railBody"), form = $("#railForm");
  let convs = [], unread = new Set(), timer = 0, cur = null, curGroup = false, myRole = "visitor";
  // Өргөн дэлгэцэд чат баруун талд наалддаг; хумих (⟩) товчоор нуугдаж, баруун ирмэгийн хавтсаар дахин нээгдэнэ.
  const wide = () => matchMedia("(min-width:1280px)").matches;
  document.body.insertAdjacentHTML("beforeend", `<button class="rail-tab" id="railTab" aria-label="Чат нээх" title="Чат нээх">${I.chat}<span>Чат</span><span class="bell-badge" id="railTabBadge" hidden></span></button>`);
  const setCollapsed = (c) => { document.documentElement.classList.toggle("rail-collapsed", c); try { localStorage.setItem("sg_rail_collapsed", c ? "1" : "0"); } catch {} };
  try { if (localStorage.getItem("sg_rail_collapsed") === "1") setCollapsed(true); } catch {}
  const setRail = (open) => {
    if (wide()) { setCollapsed(!open); }
    document.body.classList.toggle("rail-open", open && !wide()); $("#railToggle").setAttribute("aria-expanded", String(open));
  };
  const badge = () => { const n = unread.size; for (const id of ["railBadge", "railTabBadge"]) { const b = $("#" + id); if (b) { b.hidden = !n; b.textContent = n; } } };
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
    const item = (c) => `<button class="rail-item ${anim} ${unread.has(c.id) ? "unread" : ""}" style="--i:${i++}" data-id="${esc(c.id)}">${c.kind === "group" || c.kind === "team" ? `<span class="avatar avatar-sm group">${c.kind === "team" ? "🧑‍🤝‍🧑" : "👥"}</span>` : c.user ? avatarHTML(c.user) : `<span class="avatar avatar-sm" style="--h:${hueOfName(c.title)}">${esc(c.title.slice(0, 1).toUpperCase())}</span>`}
      <span class="grow"><strong>${esc(c.title)}</strong><small><b>${esc(c.sub)}</b>${c.last ? " · " + esc(c.last) : ""}</small></span>${c.at ? `<time class="rail-time">${fmtTime(c.at)}</time>` : ""}</button>`;
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
    cur = id; unread.delete(id); badge(); setRail(true);
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
    else if (m.sender_id !== u.id) { unread.add(m.conversation_id); badge(); }
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
    <div class="bell-wrap"><button class="icon-btn bell" id="bell" aria-label="Мэдэгдэл">🔔<span class="bell-badge" hidden></span></button>
    <div class="bell-panel glass" id="bellPanel" hidden><div class="bell-head"><strong>Мэдэгдэл</strong><button class="btn btn-ghost btn-sm" id="bellPerm" hidden>🔔 Браузерт мэдэгдэх</button></div><ol class="bell-list" id="bellList"><li class="muted small">Ачаалж байна…</li></ol></div></div>`);
  const badge = $(".bell-badge"), panel = $("#bellPanel"), list = $("#bellList");
  let unread = 0;
  const setBadge = (n) => { unread = n; badge.hidden = !n; badge.textContent = n > 99 ? "99+" : n; };
  const item = (n) => `<li class="bell-item ${n.read ? "" : "unread"}"><a href="${esc(n.Link || n.link || "#")}"><strong>${esc(n.title)}</strong>${n.body ? `<span>${esc(n.body)}</span>` : ""}<time>${fmtDate(n.created_at)}</time></a></li>`;
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
      bn.onclick = () => { focus(); if (n.link) location.href = n.link; };
    }
  });
  Live.connect(Auth.token);
}

/* ---------- медиа: видео, аудио, PDF (3D ном), зураг ---------- */
function extOf(url) {
  try { const p = new URL(url, location.href).pathname.toLowerCase(); const i = p.lastIndexOf("."); return i < 0 ? "" : p.slice(i); } catch { return ""; }
}
function mediaHTML(url, title) {
  if (!url) return "";
  const yt = url.match(/(?:youtube\.com\/(?:watch\?v=|embed\/|shorts\/)|youtu\.be\/)([\w-]{11})/);
  if (yt) return `<iframe data-yt="${yt[1]}" src="https://www.youtube-nocookie.com/embed/${yt[1]}?rel=0&enablejsapi=1&origin=${encodeURIComponent(location.origin)}" allow="accelerometer; autoplay; encrypted-media; picture-in-picture" allowfullscreen></iframe>`;
  const vm = url.match(/vimeo\.com\/(\d+)/);
  if (vm) return `<iframe src="https://player.vimeo.com/video/${vm[1]}" allow="autoplay; fullscreen; picture-in-picture" allowfullscreen></iframe>`;
  const ext = extOf(url);
  if ([".webm", ".mp4", ".mov", ".m4v"].includes(ext)) return `<video src="${esc(url)}" controls playsinline preload="metadata" controlslist="nodownload" disablepictureinpicture oncontextmenu="return false"></video>`;
  if ([".mp3", ".m4a", ".wav"].includes(ext)) return `<audio src="${esc(url)}" controls controlslist="nodownload" oncontextmenu="return false" style="width:100%"></audio>`;
  if ([".webp", ".jpg", ".jpeg", ".png", ".gif"].includes(ext)) return `<img src="${esc(url)}" alt="">`;
  if (ext === ".pdf") return `<div class="book-stage">${book3dHTML(url, title)}<button class="btn btn-gold" data-book="${esc(url)}" data-title="${esc(title || "")}">📖 Ном шиг нээж унших</button></div>`;
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
function blockHTML(b) {
  const cap = b.text && b.type !== "text" && b.type !== "heading" ? `<figcaption>${esc(b.text)}</figcaption>` : "";
  switch (b.type) {
    case "heading": return `<h3 class="rb-h">${esc(b.text)}</h3>`;
    case "text": return `<div class="rb-text">${richHTML(b.text)}</div>`;
    case "image": return `<figure class="rb-fig"><img src="${esc(b.url)}" alt="${esc(b.text || "")}" loading="lazy">${cap}</figure>`;
    case "audio": return `<figure class="rb-audio"><span class="rb-audio-ico">🎧</span><div><b>${esc(b.name || "Дуу бичлэг")}</b><audio src="${esc(b.url)}" controls preload="metadata" controlslist="nodownload" oncontextmenu="return false"></audio></div>${cap}</figure>`;
    case "video": return `<figure class="rb-video" data-bid="${esc(b.id)}"><div class="player">${mediaHTML(b.url, b.name)}</div>${cap}</figure>`;
    case "file": {
      const meta = esc([extOf(b.url).slice(1).toUpperCase(), fmtBytes(b.size)].filter(Boolean).join(" · ")) + (b.text ? " · " + esc(b.text) : "");
      if (extOf(b.url) === ".pdf") return `<figure class="rb-file-pdf" ${b.download ? "data-dl" : "data-nodl"}>${mediaHTML(b.url, b.name || "PDF")}${cap}</figure>`;
      return b.download ? `<a class="rb-file" href="${esc(b.url)}" target="_blank" rel="noopener" download><span class="rb-file-ico">📄</span><span><b>${esc(b.name || "Файл")}</b><small>${meta}</small></span><span class="btn btn-sm btn-glass">⬇ Татах</span></a>`
        : `<div class="rb-file"><span class="rb-file-ico">🔒</span><span><b>${esc(b.name || "Файл")}</b><small>${meta} · Багш татахыг зөвшөөрөөгүй</small></span></div>`;
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
  const colors = ["#ffb347", "#ff5e7e", "#4fd1c5", "#7aa2ff", "#ffe066", "#c084fc", "#34d399", "#fb7185"];
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
function lockedControls(v, isFree, note) {
  if (isFree()) return () => {};
  const wrap = v.parentElement; if (!wrap) return () => {};
  v.controls = false; v.removeAttribute("controls");
  wrap.classList.add("vg-locked");
  const bar = document.createElement("div");
  bar.className = "vg-bar";
  bar.innerHTML = `<button type="button" class="vg-play" aria-label="Тоглуулах">▶</button><span class="vg-time">0:00 / 0:00</span>
    <span class="vg-prog" title="Эхний удаад гүйлгэх боломжгүй — дуустал үзнэ үү"><i></i></span><span class="vg-lock" aria-hidden="true">🔒</span>
    <button type="button" class="vg-mute" aria-label="Дуу">🔊</button><button type="button" class="vg-full" aria-label="Бүтэн дэлгэц">⛶</button>`;
  wrap.append(bar);
  const fmt = (t) => `${Math.floor(t / 60)}:${String(Math.floor(t % 60)).padStart(2, "0")}`;
  const play = $(".vg-play", bar), time = $(".vg-time", bar), prog = $(".vg-prog i", bar), mute = $(".vg-mute", bar), full = $(".vg-full", bar);
  const sync = () => { play.textContent = v.paused ? "▶" : "❚❚"; time.textContent = `${fmt(v.currentTime)} / ${fmt(v.duration || 0)}`; prog.style.width = v.duration ? v.currentTime / v.duration * 100 + "%" : "0"; mute.textContent = v.muted ? "🔇" : "🔊"; };
  const toggle = () => { if (v.paused) v.play().catch(() => {}); else v.pause(); };
  play.onclick = toggle;
  v.addEventListener("click", toggle);
  mute.onclick = () => { v.muted = !v.muted; sync(); };
  full.onclick = () => { if (document.fullscreenElement) document.exitFullscreen?.(); else wrap.requestFullscreen?.(); };
  $(".vg-prog", bar).onclick = () => note?.();
  ["timeupdate", "play", "pause", "loadedmetadata", "volumechange", "durationchange"].forEach((n) => v.addEventListener(n, sync));
  const onKey = (e) => { // сум, PageUp/Down, тоонуудаар үсрэхийг хаана
    if (!wrap.contains(document.activeElement) && document.fullscreenElement !== wrap) return;
    if (["ArrowLeft", "ArrowRight", "Home", "End", "PageUp", "PageDown"].includes(e.key) || /^[0-9]$/.test(e.key)) { e.preventDefault(); note?.(); }
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

/* ---------- Хичээл үзэх үеийн анхаарал: өөр таб, цонх руу шилжвэл сануулга, 3 дахь удаад зогсоно ---------- */
const WATCH_MAX_WARN = 3;
function watchAlert(html, btn, onClose) {
  $(".watch-alert")?.remove();
  const el = document.createElement("div");
  el.className = "watch-alert"; el.setAttribute("role", "alertdialog"); el.setAttribute("aria-modal", "true");
  el.innerHTML = `<div class="watch-card">${html}<button class="btn btn-gold btn-block">${btn}</button></div>`;
  document.body.append(el);
  const b = $("button", el); b.focus();
  b.onclick = () => { el.remove(); onClose?.(); };
}
// watchLesson нь хичээлийн цонх нээлттэй байх хугацааг хянаж, хаагдахад зогсоох функц буцаана.
// Өөр таб/цонх руу шилжвэл сануулга (3 дахь удаад зогсоно), 5 минут тутам "Та үзэж байна уу?" (30 сек),
// хөдөлгөөнт усан тэмдэг (нэр, ID, IP), идэвхтэй хугацааны тоолуур.
const WATCH_PING_SEC = 300, WATCH_PING_ANSWER = 30, WATCH_IDLE_MS = 120000;
function watchLesson({ courseId, lessonId, modal, onStop, onActive }) {
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
  const leave = () => { if (away || stopped) return; tick(); away = true; awayAt = Date.now(); pauseMedia(); };
  const back = () => {
    if (!away || stopped) return;
    tick(); away = false;
    const sec = Math.max(1, Math.round((Date.now() - awayAt) / 1000));
    if (owner) return; // багш өөрийн хичээлийг шалгаж байна
    warns++; paint();
    events.push({ type: "tab_switch", detail: `${sec} сек өөр цонхонд байсан (сануулга ${warns}/${WATCH_MAX_WARN})` });
    if (warns >= WATCH_MAX_WARN) return stop(`${WATCH_MAX_WARN} удаа хичээлээс гарсан`, `Та ${WATCH_MAX_WARN} удаа хичээлээс гарсан тул хичээл зогслоо.`);
    send();
    watchAlert(`<span class="watch-ico">⚠️</span><h3>Та хичээлээс гарсан байна</h3>
      <p>Хичээл үзэж байхдаа өөр таб, цонх руу шилжихгүй байна уу. Видео түр зогслоо.</p>
      <p class="watch-count">Сануулга <b>${warns}/${WATCH_MAX_WARN}</b> · ${WATCH_MAX_WARN}-р удаад хичээл зогсоно</p>`, "Ойлголоо, үргэлжлүүлэх");
  };
  const stop = (detail, msg) => {
    stopped = true;
    events.push({ type: "auto_block", detail });
    send("auto_block"); cleanup(true);
    onStop?.();
    watchAlert(`<span class="watch-ico">⛔</span><h3>Анхаарал идэвхгүй тул хичээл зогслоо</h3>
      <p>${msg} Энэ тухай багшид мэдэгдсэн. Идэвхгүй байсан хугацаа суралцсан цагт тооцогдохгүй.</p>
      <p class="muted small">Дахин үзэхдээ хичээлээ анхааралтай, бусад цонхоо хаагаад үзээрэй.</p>`, "Ойлголоо");
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
  const note = () => toast("Видеог эхлээд нэг удаа бүрэн үзнэ үү — дараа нь гүйлгэж болно");
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
      let lastBucket = -1;
      v.addEventListener("timeupdate", () => {
        if (!v.seeking && v.currentTime > max && v.currentTime - max < 3) max = v.currentTime;
        const b = Math.floor(v.currentTime / BUCKET); if (b !== lastBucket) { lastBucket = b; mark(v.currentTime); }
      });
      v.addEventListener("seeking", () => { if (!free && v.currentTime > max + 1) { v.currentTime = max; note(); } });
      v.addEventListener("ratechange", () => { if (!free && v.playbackRate > 2) v.playbackRate = 2; });
      const release = lockedControls(v, () => free, note);
      v.addEventListener("ended", () => { if (!free) { free = true; onWatched?.(bid); release(); } flush(); });
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
  if (!Auth.token) { card.innerHTML = `<h3>📝 Шалгалт</h3><p>Шалгалт өгөхийн тулд нэвтэрнэ үү.</p><a class="btn btn-gold" href="/login?next=${encodeURIComponent(location.pathname)}">Нэвтрэх</a>`; return; }
  const draw = async () => {
    let info;
    try { info = await api(`${base}/exam`); } catch (e) { card.innerHTML = `<p class="form-error">${esc(e.message)}</p>`; return; }
    const ex = info.exam, best = info.attempts.filter((a) => a.status !== "active").reduce((m, a) => Math.max(m, a.pct), -1);
    const active = info.attempts.find((a) => a.status === "active"), due = info.due || { open: true };
    card.innerHTML = `<h3>📝 Шалгалт</h3>
      <ul class="exam-facts"><li>❓ <b>${info.questions}</b> асуулт</li><li>⏱ ${ex.time_min ? `<b>${ex.time_min}</b> минут` : "Хугацаа хязгааргүй"}</li>${due.start_at ? `<li>▶ Эхлэх: <b>${fmtDate(due.start_at)}</b></li>` : ""}${due.at ? `<li>📅 ${due.late ? `<span class="an-bad">Хугацаа дууссан</span> (${fmtDate(due.at)})` : `<b>${fmtDate(due.at)}</b> хүртэл`}</li>` : ""}${due.entry_fee ? `<li>💳 Оролцооны төлбөр: <b>${money(due.entry_fee)}</b>${due.paid ? " ✓" : ""}</li>` : ""}
        <li>🎯 Тэнцэх: <b>${ex.pass_pct}%</b></li><li>🔁 ${info.left < 0 ? "Оролдлого хязгааргүй" : `Үлдсэн оролдлого: <b>${info.left}</b>`}</li>${best >= 0 ? `<li>🏆 Таны шилдэг: <b>${best}%</b></li>` : ""}</ul>
      <div class="exam-rules"><b>⚠️ Дүрэм:</b> Шалгалтын үеэр өөр таб, цонх руу шилжих эсвэл текст хуулах үед шалгалт <b>шууд хаагдаж</b>, тэр хүртэлх хариултаар дүгнэгдэнэ. Энэ тухай багшид мэдэгдэнэ.</div>
      ${info.attempts.length ? `<table class="tbl"><thead><tr><th>Огноо</th><th>Оноо</th><th>Төлөв</th></tr></thead><tbody>${info.attempts.map((a) => `<tr><td>${fmtDate(a.started_at)}</td><td>${a.status === "active" ? "—" : a.pct + "%" + (a.passed ? " ✓" : "")}</td><td>${a.status === "terminated" ? `⛔ Хаагдсан · ${esc(a.reason || "")}` : a.status === "submitted" ? (a.passed ? "Тэнцсэн" : "Тэнцээгүй") : a.status === "expired" ? "Хугацаа хэтэрсэн" : "Үргэлжилж байна"}</td></tr>`).join("")}</tbody></table>` : ""}
      ${dueNotice(due)}
      ${due.closed || due.not_started ? "" : due.need_pay ? `<button class="btn btn-gold btn-lg" data-late-pay>💳 ${money(due.fee)} төлж шалгалтаа нээх</button>` : active || info.left !== 0 ? `<button class="btn btn-gold btn-lg" data-exam-start>${active ? "▶ Шалгалтаа үргэлжлүүлэх" : "▶ Шалгалт эхлүүлэх"}</button>` : `<p class="muted">Оролдлогын тоо дууссан.</p>`}`;
    bindLatePay(card, base, "Шалгалтын төлбөр", draw);
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
function bindLatePay(card, base, label, after) {
  const b = $("[data-late-pay]", card); if (!b) return;
  b.onclick = async () => {
    b.disabled = true;
    try { const d = await api(`${base}/late-pay`, { method: "POST" }); if (d.unlocked) return after(); window.SG_pay?.(d.order, d.payment, label, () => { toast("✓ Нээгдлээ"); after(); }); }
    catch (e) { toast(e.message, true); } finally { b.disabled = false; }
  };
}

// Даалгавар: нөхцөл (блокууд дээр), хугацаа, хариу илгээх (текст + файл), багшийн дүн.
async function assignmentCard(box, { courseId, lessonId }) {
  const base = `/api/courses/${courseId}/lessons/${lessonId}`;
  const card = document.createElement("section"); card.className = "exam-card asg-card"; box.append(card);
  if (!Auth.token) { card.innerHTML = `<h3>📎 Даалгавар</h3><p>Хариу илгээхийн тулд нэвтэрнэ үү.</p><a class="btn btn-gold" href="/login?next=${encodeURIComponent(location.pathname)}">Нэвтрэх</a>`; return; }
  const draw = async () => {
    let info;
    try { info = await api(`${base}/assignment`); } catch (e) { card.innerHTML = `<p class="form-error">${esc(e.message)}</p>`; return; }
    const a = info.assignment, due = info.due || { open: true }, sub = info.submission;
    card.innerHTML = `<h3>📎 Даалгавар</h3>
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
    <div class="exam-body"><div class="exam-qs">${data.questions.map((b, i) => quizFormHTML(b, true, i + 1)).join("")}</div>
      <button class="btn btn-gold btn-lg btn-block" data-exam-submit>Шалгалтаа илгээх</button></div>`;
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
  const progress = () => { const n = Object.keys(answers()).length; $(".exam-prog", el).textContent = `${n}/${data.questions.length} хариулсан`; try { sessionStorage.setItem(key, JSON.stringify(answers())); } catch {} };
  el.addEventListener("change", progress); el.addEventListener("input", progress); progress();
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
  const violate = (type, label) => { if (finished) return; beatNow([{ type, detail: "Шалгалтын үеэр: " + label }]); submit(type, `⛔ Шалгалт хаагдлаа: ${label}.`); };
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
    $(".exam-body > [data-exam-submit]", el)?.remove();
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
    if (!e.target.closest("[data-exam-submit]")) return;
    const n = Object.keys(answers()).length;
    if (n < data.questions.length && !confirm(`${data.questions.length - n} асуултад хариулаагүй байна. Илгээх үү?`)) return;
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
const BRAND_HUES = [222, 214, 206, 230, 36, 28, 218, 198];
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
function hydrateBooks(root = document) { $$(".book3d:not([data-h])", root).forEach((b) => { b.dataset.h = 1; coverIO?.observe(b); }); }
document.addEventListener("click", (e) => {
  const b = e.target.closest("[data-book], .book3d");
  if (!b) return;
  e.preventDefault();
  // Хичээл доторх PDF: багш зөвшөөрөөгүй бол татах холбоосгүй нээнэ.
  const nodl = !!b.closest("[data-nodl]") || (!!b.closest("#lessonModal") && !b.closest("[data-dl]"));
  Flipbook.open(b.dataset.book || b.dataset.pdf, b.dataset.title, { nodl });
});

const Flipbook = {
  el: null,
  async open(url, title, opts = {}) {
    this.close();
    const el = document.createElement("div");
    el.className = "flipbook";
    el.innerHTML = `<div class="fb-bar"><strong>${esc(title || "Баримт")}</strong>${opts.nodl ? `<span class="chip">🔒 Зөвхөн унших</span>` : `<a class="btn btn-ghost btn-sm" href="${esc(url)}" target="_blank" rel="noopener">⬇</a>`}<button class="icon-btn" data-fb-close aria-label="Хаах">✕</button></div>
      <div class="fb-stage"><div class="loader"></div></div>
      <div class="fb-nav"><button class="btn btn-glass btn-sm" data-fb-prev>←</button><input type="range" min="0" value="0"><span class="fb-count"></span><button class="btn btn-glass btn-sm" data-fb-next>→</button></div>`;
    document.body.append(el);
    this.el = el;
    requestAnimationFrame(() => el.classList.add("open"));
    el.querySelector("[data-fb-close]").onclick = () => this.close();
    this.key = (e) => { if (e.key === "ArrowRight") this.go(this.cur + 1); if (e.key === "ArrowLeft") this.go(this.cur - 1); if (e.key === "Escape") this.close(); };
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
    this.single = sw < 760;
    const cols = this.single ? 1 : 2;
    let pw = Math.min(sw / cols, sh / this.ratio), ph = pw * this.ratio;
    // Хуудас бүр нэг "навч"; хоёр талтай горимд навч нэг бүр 2 хуудас (урд/ард).
    const per = this.single ? 1 : 2;
    this.leaves = Math.ceil(n / per);
    stage.innerHTML = `<div class="fb-book ${this.single ? "single" : ""}" style="width:${pw * cols}px;height:${ph}px"></div>`;
    const book = $(".fb-book", stage);
    this.pw = pw; this.ph = ph; this.book = book; this.rendered = new Set();
    for (let i = 0; i < this.leaves; i++) {
      const f = i * per + 1, b = per === 2 ? f + 1 : 0;
      const leaf = document.createElement("div");
      leaf.className = "leaf";
      leaf.innerHTML = `<div class="face front" data-p="${f}"><div class="page-loading">${f}</div><span class="pnum">${f}</span></div>` +
        (b && b <= n ? `<div class="face back" data-p="${b}"><div class="page-loading">${b}</div><span class="pnum">${b}</span></div>` : `<div class="face back"></div>`);
      leaf.addEventListener("click", (e) => { e.stopPropagation(); this.go(leaf.classList.contains("flipped") ? i : i + 1); });
      book.append(leaf);
    }
    this.leafEls = $$(".leaf", book);
    const range = $("input[type=range]", el);
    range.max = this.leaves; range.oninput = () => this.go(+range.value, true);
    $("[data-fb-prev]", el).onclick = () => this.go(this.cur - 1);
    $("[data-fb-next]", el).onclick = () => this.go(this.cur + 1);
    // Шудрах (swipe)
    let x0 = null;
    stage.onpointerdown = (e) => (x0 = e.clientX);
    stage.onpointerup = (e) => { if (x0 != null && Math.abs(e.clientX - x0) > 40) this.go(this.cur + (e.clientX < x0 ? 1 : -1)); x0 = null; };
    this.cur = 0;
    this.go(0, true);
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
    // Ойролцоох хуудсуудыг л зурна (том PDF ч хурдан).
    for (let p = Math.max(1, k * per - 3); p <= Math.min(n, k * per + 5); p++) this.render(p);
  },
  async render(p) {
    if (this.rendered.has(p)) return;
    this.rendered.add(p);
    const face = $(`.face[data-p="${p}"]`, this.el);
    if (!face) return;
    const pg = await this.doc.getPage(p);
    const base = pg.getViewport({ scale: 1 });
    const vp = pg.getViewport({ scale: (this.pw / base.width) * Math.min(2, devicePixelRatio || 1) });
    const cv = document.createElement("canvas"); cv.width = vp.width; cv.height = vp.height;
    await pg.render({ canvasContext: cv.getContext("2d"), viewport: vp }).promise;
    $(".page-loading", face)?.replaceWith(cv);
  },
  close() {
    if (!this.el) return;
    const el = this.el; this.el = null; this.leafEls = null;
    removeEventListener("keydown", this.key);
    el.classList.remove("open");
    setTimeout(() => el.remove(), 400);
    this.doc?.destroy(); this.doc = null;
  },
};

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
      $("#payAmount").textContent = money(d.order.amount);
      const dev = $("#devPayBtn"); dev.hidden = !d.payment.dev_pay;
      const done = async () => { closeModal($("#payModal")); toast("🎉 Ном нээгдлээ!"); celebrate(); await refresh(); read(); };
      dev.onclick = async () => { dev.disabled = true; try { await api(`/api/orders/${d.order.id}/dev-pay`, { method: "POST" }); done(); } catch (e) { toast(e.message, true); } finally { dev.disabled = false; } };
      openModal($("#payModal"));
      const poll = setInterval(async () => {
        if (!$("#payModal").classList.contains("open")) return clearInterval(poll);
        const o = await api(`/api/orders/${d.order.id}`).catch(() => null);
        if (o?.status === "paid") { clearInterval(poll); done(); }
      }, 4000);
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
    Object.assign(this, { list: [], reads: {}, meKey: "", typing: new Map(), replyTo: null, lastReadSent: "", typeAt: 0 }, o);
    this.ol.classList.add("thread");
    this.ol.addEventListener("click", (e) => this.click(e));
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
      const foot = last || rx.length ? `<div class="msg-foot"><time>${fmtTime(m.created_at)}</time>${mine && seen && seen.id === m.id ? `<span class="msg-seen">✓✓ Үзсэн${this.group() && seen.n > 1 ? ` · ${seen.n}` : ""}</span>` : ""}</div>` : "";
      const sticker = /^::sticker::(.+)$/.exec(m.body || "");
      const imgUrl = m.attachment_url || (/^https?:\/\/\S+\.(gif|png|jpe?g|webp)(\?\S*)?$/i.test(m.body || "") ? m.body : "");
      const content = sticker ? `<span class="msg-sticker">${esc(sticker[1])}</span>` : imgUrl ? `<a class="msg-img" href="${esc(imgUrl)}" target="_blank" rel="noopener"><img src="${esc(imgUrl)}" alt="" loading="lazy"></a>${m.attachment_url && m.body && m.body !== "📷 Зураг" ? `<span class="msg-text">${linkify(m.body)}</span>` : ""}` : `<span class="msg-text">${linkify(m.body)}</span>`;
      out.push(`<li class="msg ${mine ? "me" : "them"} ${first ? "first" : ""} ${last ? "last" : ""} ${sticker ? "is-sticker" : ""} ${imgUrl ? "has-img" : ""}" data-id="${esc(m.id)}">${av}<div class="msg-col">${name}
        <div class="msg-row"><div class="bubble" title="${fmtDate(m.created_at)}">${quote}${content}</div>
          <div class="msg-tools"><button type="button" data-react="${esc(m.id)}" title="Реакц">☺</button>${mine ? "" : `<button type="button" data-reply="${esc(m.id)}" title="Хариулах">↩</button>`}</div></div>
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
    bar.innerHTML = `<span>↩ <b>${esc(this.replyTo.sender_name || (this.replyTo.sender === "teacher" ? "Багш" : "Зочин"))}</b>-д хариулж байна: <em>${esc(this.replyTo.body.slice(0, 80))}</em></span><button type="button" data-cancel-reply aria-label="Болих">✕</button>`;
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
    $("input", bar).addEventListener("input", (e) => { q = e.target.value.trim().toLowerCase(); apply(); });
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
  const sec = (id, title, body, extra = "") => `<section class="dash-sec" id="${id}"><header><h2>${title}</h2>${extra}</header>${body}</section>`;
  const courseCard = (c, i) => `<a class="course-card tilt" href="/c/${esc(c.course.id)}" style="--h:${hueOfName(c.course.title)}">
    <div class="course-art"><div class="course-flags"><span class="flag flag-own">✓ ${c.access === "lessons" ? c.owned_lessons + " хичээл" : accessLabel[c.access]}</span></div><span class="course-num">${String(i + 1).padStart(2, "0")}</span><span class="glare"></span></div>
    <div class="course-body"><h3>${esc(c.course.title)}</h3>${c.teacher ? `<div class="course-by">${avatarHTML(c.teacher)}${esc(c.teacher.display_name)}</div>` : ""}
    <div class="course-meta"><span>${c.course.lesson_count} хичээл</span><span class="price free">▶ Үргэлжлүүлэх</span></div></div></a>`;

  const tp = h.teacher;
  const parts = [`<div class="dash-head">${avatarHTML(u, "avatar-md")}<div class="grow"><h1>${hello}, ${esc(u.display_name.split(" ")[0])} 👋</h1>
      <p>${h.courses.length ? `Танд ${h.courses.length} сургалт${h.meetings.length ? `, ${h.meetings.length} шууд хичээл` : ""} байна.` : tp ? "Багшийн самбар болон суралцах хэсэг тань энд байна." : "Багшийнхаа профайлаас анхны сургалтаа сонгоорой."}</p></div>
      ${tp ? `<a class="btn" href="/t/${esc(u.username)}#overview">Миний профайл →</a>` : `<button class="btn" data-student-settings>Тохиргоо</button>`}</div>`,
    `<div class="tiles">
      <a class="tile" href="#my-courses"><em>🎓</em><b>${h.courses.length}</b><span>Миний сургалт</span></a>
      <a class="tile" href="#my-lessons"><em>📘</em><b>${h.lessons.length}</b><span>Авсан хичээл</span></a>
      <a class="tile" href="#my-live"><em>📹</em><b>${h.meetings.length}</b><span>Шууд хичээл</span></a>
      <a class="tile" href="#my-chats"><em>💬</em><b>${h.chats.length}</b><span>Чат</span></a>
      <a class="tile" href="#my-teachers"><em>👩‍🏫</em><b>${h.teachers.length}</b><span>Миний багш</span></a>
      ${h.rank ? `<a class="tile tile-rank" href="#my-rank"><em class="rank-shine" data-level="${h.rank.level}">${esc(h.rank.insignia)}</em><b>${esc(h.rank.name)}</b><span>Миний цол · ${h.rank.points} оноо</span></a>` : ""}
      ${h.tasks?.length ? `<a class="tile" href="#my-tasks"><em>📎</em><b>${h.tasks.filter((t) => ["open", "need_pay"].includes(t.status)).length}</b><span>Хийх даалгавар, шалгалт</span></a>` : ""}
    </div>`];
  // Цол: систем шинэ цол олгосон бол баяр хүргэж салют буудуулна.
  if (h.rank) {
    const lvlKey = "sg_rank_lvl_" + (u.id || "u");
    let seen = -1; try { seen = localStorage.getItem(lvlKey) === null ? -1 : +localStorage.getItem(lvlKey); } catch {}
    if (h.rank_awarded || (seen >= 0 && h.rank.level > seen)) setTimeout(() => rankSalute(h.rank), 400);
    try { localStorage.setItem(lvlKey, h.rank.level); } catch {}
  }

  if (resume) parts.push(`<div class="resume"><span class="item-ico">▶</span><div class="grow"><span class="eyebrow">Үргэлжлүүлэх</span><strong>${esc(last.lessonTitle || resume.course.title)}</strong><span class="muted small">${esc(resume.course.title)}</span></div>
      <a class="btn btn-gold" href="/c/${esc(last.course)}${last.lesson ? "#l=" + esc(last.lesson) : ""}">Үзэх</a></div>`);

  if (h.rank) {
    const rk = h.rank;
    parts.push(sec("my-rank", "🎖 Миний цол", `<div class="rank-card">
      <div class="rank-main"><span class="rank-big rank-shine" data-level="${rk.level}">${esc(rk.insignia)}</span><div class="grow"><strong class="rank-title">${esc(rk.name)}</strong>
        <span class="muted small">${rk.points} оноо · ${rk.honest} шударга хичээл${rk.cheated ? ` · <span class="cr-bad">${rk.cheated} хичээлд хуулах оролдлогоос цол олгоогүй</span>` : ""}</span>
        <div class="meter cr-meter" style="margin-top:8px"><i style="width:${rk.progress}%"></i></div>
        <span class="muted small">${rk.next ? `Дараагийн «${esc(rk.next_name)}» цол ${rk.next} оноонд — ${rk.next - rk.points} дутуу` : "Дээд цол — баяр хүргэе!"}</span></div></div>
      ${rk.tips?.length ? `<div class="rank-tips"><b>Систем зөвлөж байна</b><ul>${rk.tips.map((t) => `<li>${esc(t)}</li>`).join("")}</ul></div>` : ""}
      ${h.course_ranks.length ? `<ul class="rank-courses">${h.course_ranks.map((c) => `<li><a href="/c/${esc(c.course_id)}"><span class="rank-shine" data-level="${c.rank.level}">${esc(c.rank.insignia)}</span><span class="grow"><strong>${esc(c.title)}</strong><small>${esc(c.rank.name)} · ${c.rank.points} оноо · ${c.lessons.length} хичээл үнэлэгдсэн</small></span><span class="chip chip-teal">Үзэх</span></a></li>`).join("")}</ul>` : ""}</div>`));
  }
  if (h.tasks?.length) {
    const label = (t) => ({ open: ["Хийх", "chip-amber"], not_started: [`${fmtDate(t.due.start_at)}-д эхэлнэ`, ""], need_pay: [(t.due.need_late ? "Хоцорсон · " : "Төлбөртэй · ") + money(t.due.fee) + " төлж нээнэ", "chip-amber"], closed: ["Хаалттай", ""], submitted: ["Илгээсэн · дүгнэхийг хүлээж байна", "chip-teal"],
      graded: [`Дүн: ${t.score}/${t.max_score || 100}`, "chip-teal"], passed: [`Тэнцсэн · ${t.exam_best}%`, "chip-teal"], failed: [`Тэнцээгүй · шилдэг ${t.exam_best}%`, "chip-amber"] })[t.status] || ["", ""];
    parts.push(sec("my-tasks", "📎 Даалгавар, шалгалт", `<ul class="items">${h.tasks.map((t) => { const [txt, cls] = label(t); return `
      <li><a class="item ${["open", "need_pay"].includes(t.status) && t.due.at ? "warn" : ""}" href="/c/${esc(t.course_id)}#l=${esc(t.lesson_id)}"><span class="item-ico">${t.kind === "exam" ? "📝" : "📎"}</span><span class="grow"><strong>${esc(t.title)}</strong><small>${esc(t.course_title)}${t.due.at ? ` · ${t.due.late ? "хугацаа дууссан" : "хугацаа"}: ${fmtDate(t.due.at)}` : " · хугацаагүй"}${t.feedback ? ` · ${esc(t.feedback)}` : ""}</small></span><span class="chip ${cls}">${txt}</span></a></li>`; }).join("")}</ul>`));
  }
  if (h.pending.length) parts.push(sec("my-pending", "Төлбөр хүлээгдэж буй", `<ul class="items">${h.pending.map((o) => `
      <li><a class="item warn" href="/c/${esc(o.course_id)}${o.lesson_id ? "#l=" + esc(o.lesson_id) : ""}"><span class="item-ico">₮</span><span class="grow"><strong>${esc(o.title)}</strong><small>${fmtDate(o.created_at)} · ${o.kind === "lesson" ? "нэг хичээл" : "бүтэн сургалт"}</small></span><strong>${money(o.amount)}</strong><span class="chip chip-amber">Төлөх →</span></a></li>`).join("")}</ul>`));

  if (tp) {
    const ins = tp.insights, todo = ins.tips.filter((t) => !t.done).slice(0, 3);
    parts.push(sec("my-studio", "Багшийн самбар", `<div class="panel"><div class="strength">${ringHTML(ins.score)}<div><strong>Профайлын бүрдэл · ${esc(ins.level)}</strong>
        <p class="muted small" style="margin:.2em 0 0">${tp.published}/${tp.courses} сургалт нийтлэгдсэн · ${tp.students} суралцагч · ${tp.profile_views} профайл үзэлт</p>
        <div class="hero-cta" style="margin-top:12px"><a class="btn btn-gold btn-sm" href="/t/${esc(u.username)}#overview">Удирдлага нээх</a><a class="btn btn-ghost btn-sm" href="/t/${esc(u.username)}">Нээлттэй профайл</a><a class="btn btn-ghost btn-sm" href="/t/${esc(u.username)}#chat">Чат</a></div></div></div>
        ${todo.length ? `<ul class="tips">${todo.map((t) => `<li><a class="tip" href="/t/${esc(u.username)}${esc(t.link)}"><i>✓</i><span><strong>${esc(t.title)}</strong><small>${esc(t.hint)}</small></span><span class="chip chip-gold">Хийх →</span></a></li>`).join("")}</ul>` : ""}</div>`));
  }

  if (h.meetings.length) parts.push(sec("my-live", "Шууд хичээл", `<ul class="items">${h.meetings.map((m) => `
      <li><div class="item live"><span class="item-ico">📹</span><span class="grow"><strong>${esc(m.title)}</strong><small>${fmtDate(m.starts_at)} · ${m.duration_min} мин · ${esc(m.course_title)}</small></span><span class="chip chip-teal">${until(m.starts_at)}</span>
      ${m.meet_url ? `<a class="btn btn-teal btn-sm" href="${esc(m.meet_url)}" target="_blank" rel="noopener">Нэгдэх</a>` : `<a class="btn btn-ghost btn-sm" href="/c/${esc(m.course_id)}">Сургалт</a>`}</div></li>`).join("")}</ul>`));
  else parts.push(`<span id="my-live"></span>`);

  parts.push(sec("my-courses", "Миний сургалтууд", h.courses.length ? `<div class="course-grid">${h.courses.map(courseCard).join("")}</div>`
    : `<div class="empty">Та одоогоор сургалтад элсээгүй байна.<br>Багшийнхаа хэрэглэгчийн нэрийг оруулаад профайл руу нь ороорой — эсвэл QR кодыг нь уншуулаарай.
       <form class="find" id="findTeacher"><input name="u" required pattern="[A-Za-z0-9_]{3,32}" placeholder="Багшийн нэр, жишээ: demo" aria-label="Багшийн хэрэглэгчийн нэр"><button class="btn btn-gold">Профайл нээх</button></form></div>`));

  if (h.lessons.length) parts.push(sec("my-lessons", "Дангаар авсан хичээлүүд", `<ul class="items">${h.lessons.map((l) => `
      <li><a class="item" href="/c/${esc(l.course_id)}#l=${esc(l.lesson_id)}"><span class="item-ico">▶</span><span class="grow"><strong>${esc(l.title)}</strong><small>${l.paid_at ? fmtDate(l.paid_at) + "-д авсан" : ""}</small></span><span class="chip chip-teal">Үзэх</span></a></li>`).join("")}</ul>`));
  else parts.push(`<span id="my-lessons"></span>`);

  const teachers = h.teachers.length ? sec("my-teachers", "Миний багш нар", `<div class="people">${h.teachers.map((t) => `
      <a class="person" href="/t/${esc(t.username)}">${avatarHTML(t)}<span class="grow"><strong>${esc(t.display_name)}</strong><small>${esc(t.headline || "@" + t.username)}</small></span></a>`).join("")}</div>`) : `<span id="my-teachers"></span>`;
  const chats = h.chats.length ? sec("my-chats", "Чат", `<ul class="items">${h.chats.map((c) => `
      <li><a class="item" href="${c.kind === "group" ? `/c/${esc(c.course_id)}#chat=group` : `/t/${esc(c.teacher.username)}#chat`}" data-rail-open="${esc(c.id)}">${c.kind === "group" ? `<span class="avatar avatar-sm group">👥</span>` : avatarHTML(c.teacher)}<span class="grow"><strong>${esc(c.kind === "group" ? c.title : c.teacher.display_name)}</strong><small>${c.kind === "group" ? "Бүлэг · " + esc(c.teacher.display_name) + " · " : ""}${esc(c.last_message)}</small></span><time class="muted small">${fmtDate(c.last_message_at)}</time></a></li>`).join("")}</ul>`) : `<span id="my-chats"></span>`;
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
      const st = states[li.dataset.lesson], p = prog[li.dataset.lesson], lbl = $(".lesson-state", li);
      li.classList.toggle("is-done", !!p?.completed_at);
      const dripLocked = st && !st.open && !li.classList.contains("is-free");
      li.classList.toggle("is-drip", !!dripLocked);
      if (!lbl) return;
      if (dripLocked) {
        lbl.hidden = false; lbl.className = "lesson-state " + (st.reason === "timer" ? "timer" : "");
        lbl.textContent = st.reason === "timer" ? `⏳ ${fmtDate(st.unlock_at)}-д нээгдэнэ`
          : st.reason === "quiz" ? `🔒 «${st.prev_title}» хичээлийн асуултуудад бүгдэд нь зөв хариулсны дараа нээгдэнэ (${st.quiz_left}/${st.quiz_total} үлдсэн)`
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
    // Цэргийн цол: хичээл бүрийн үнэлгээ (хуулах оролдлоготой бол цолгүй) ба нэгдсэн цол.
    const ranks = a.ranks || {};
    $$(".lesson").forEach((li) => {
      const r = ranks[li.dataset.lesson]; let b = $(".lesson-rank", li);
      if (!r) { b?.remove(); return; }
      if (!b) { b = document.createElement("span"); b.className = "lesson-rank"; $(".lesson-title", li)?.append(b); }
      b.className = "lesson-rank " + (r.disqualified ? "disq" : r.points >= 75 ? "high rank-shine" : "");
      b.title = (r.reasons || []).join(", ");
      b.textContent = r.disqualified ? "⛔ Цолгүй" : `🎖 ${r.rank} · ${r.points}`;
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
      rl.innerHTML = `<span class="cr-sign rank-shine" data-level="${rk.level}">${esc(rk.insignia)}</span><b class="rank-name">${esc(rk.name)}</b><span class="muted small">${rk.points} оноо${rk.next ? ` · дараагийн цол «${esc(rk.next_name)}» ${rk.next} оноонд` : " · дээд цол"}${tot ? ` · бүх сургалтаар: <b>${esc(tot.name)}</b> ${tot.points}` : ""}${rk.cheated ? ` · <span class="cr-bad">${rk.cheated} хичээлд хуулах оролдлогоос цол олгоогүй</span>` : ""}</span>
        <span class="meter cr-meter"><i style="width:${rk.progress}%"></i></span>
        ${rk.tips?.length ? `<ul class="cr-tips">${rk.tips.map((t) => `<li>${esc(t)}</li>`).join("")}</ul>` : ""}`;
    }
  };
  const refreshAccess = () => api(`/api/courses/${id}/access`).then((a) => { applyStates(a); return a; }).catch(() => null);
  document.addEventListener("sg:quiz-mastered", () => refreshAccess()); // асуулгыг дуусгамагц түгжээ шууд нээгдэнэ
  const unlockLesson = (lid) => {
    const li = $(`.lesson[data-lesson="${lid}"]`);
    if (!li || li.classList.contains("is-free")) return;
    li.classList.add("unlocked"); li.classList.remove("is-locked");
    const chip = $(".lesson-price", li); if (chip) { chip.textContent = "✓ Нээлттэй"; chip.className = "chip chip-teal lesson-price"; }
    const b = $("[data-play]", li); if (b) { b.className = "btn btn-sm btn-teal"; b.innerHTML = "▶ Үзэх"; }
  };
  const unlockAll = () => {
    all = true;
    buy.textContent = "✓ Бүх хичээл нээлттэй — үзэж эхлэх";
    $$(".lesson").forEach((l) => unlockLesson(l.dataset.lesson));
  };
  if (Auth.token) refreshAccess().then((a) => { if (!a) return; if (a.all) unlockAll(); a.lessons.forEach(unlockLesson); if (mode === "free" && a.enrolled) buy.textContent = "✓ Та элссэн — үзэж эхлэх"; });
  else if (drip) $$(".lesson:not(.is-free)").forEach((li) => { const lbl = $(".lesson-state", li); if (lbl && +li.dataset.unlock && li.dataset.always !== "1") { lbl.hidden = false; lbl.textContent = `⏱ Өмнөхийг үзснээс ${humanHours(li.dataset.unlock)}-ийн дараа`; } });
  // Товлосон шууд хичээлүүд (Google Meet)
  api(`/api/courses/${id}/meetings`).then((d) => {
    if (!d.meetings.length) return;
    const sec = $(".lessons").parentElement;
    sec.insertAdjacentHTML("afterbegin", `<div class="section-head reveal in"><span class="eyebrow">Шууд хичээл</span></div><div class="meet-list">${d.meetings.map((m) => `
      <div class="meet-item"><time>${fmtDate(m.starts_at)}</time><span class="grow" style="flex:1">${esc(m.title)} · ${m.duration_min} мин</span>
      ${m.meet_url ? `<a class="btn btn-teal btn-sm" href="${esc(m.meet_url)}" target="_blank" rel="noopener">📹 Нэгдэх</a>` : `<span class="chip">🔒 Худалдан авсан хүмүүст</span>`}</div>`).join("")}</div>`);
  }).catch(() => {});

  const needLogin = () => { if (Auth.token) return false; location.href = "/login?next=" + encodeURIComponent(location.pathname); return true; };

  // Төлбөр: захиалга → модал → (демо товч эсвэл webhook-ийг хүлээнэ)
  const pay = (order, payment, label, onPaid) => {
    $("#payAmount").textContent = money(order.amount);
    $("#payNote").textContent = label + " — төлбөр баталгаажмагц автоматаар нээгдэнэ.";
    const dev = $("#devPayBtn");
    dev.hidden = !payment.dev_pay;
    dev.onclick = async () => {
      dev.disabled = true;
      try { await api(`/api/orders/${order.id}/dev-pay`, { method: "POST" }); closeModal($("#payModal")); onPaid(); }
      catch (e) { toast(e.message, true); } finally { dev.disabled = false; }
    };
    openModal($("#payModal"));
    const poll = setInterval(async () => {
      if (!$("#payModal").classList.contains("open")) return clearInterval(poll);
      const o = await api(`/api/orders/${order.id}`).catch(() => null);
      if (o?.status === "paid") { clearInterval(poll); closeModal($("#payModal")); onPaid(); }
    }, 4000);
  };  window.SG_pay = pay; // шалгалт, даалгаврын хоцролтын төлбөрт ч ижил цонх


  buy.addEventListener("click", async () => {
    if (all || mode === "lessons") { $("#lessons").scrollIntoView({ behavior: "smooth", block: "start" }); return; }
    if (needLogin()) return;
    buy.disabled = true;
    try {
      const d = await api(`/api/courses/${id}/enroll`, { method: "POST" });
      if (d.enrolled) { if (bundle) unlockAll(); else buy.textContent = "✓ Та элссэн — үзэж эхлэх"; toast("🎉 Амжилттай!"); celebrate(); return; }
      pay(d.order, d.payment, "Бүх хичээл багцаар", () => { unlockAll(); toast("🎉 Бүх хичээл нээгдлээ!"); celebrate(); });
    } catch (e) { toast(e.message, true); } finally { buy.disabled = false; }
  });

  // Хичээлийн цонх хаагдахад анхаарлын хяналт дуусна, видеоны явц илгээгдэнэ, дүгнэлт асууна.
  let stopWatch = null, stopVideos = null, reflectDone = false, curLesson = null;
  new MutationObserver(() => {
    if ($("#lessonModal").classList.contains("open")) return;
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
  const play = async (lid) => {
    const l = await api(`/api/courses/${id}/lessons/${lid}`);
    $("#lessonTitle").textContent = l.title;
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
    hydrateBooks(player); hydrateBooks(body);
    const modal = $("#lessonModal");
    const isOwner = !!Auth.user?.username && $(".teacher-chip")?.getAttribute("href") === "/t/" + Auth.user.username;
    if (l.exam) { // шалгалт: асуултууд зөвхөн "эхлүүлэх"-ээр ирнэ
      if (done) done.hidden = true;
      examCard(body, { courseId: id, lessonId: lid, title: l.title, onFinish: refreshAccess });
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
    stopVideos = guardVideos(body, { watched, owner: isOwner || !Auth.token, modal, progressBase: Auth.token && !isOwner ? `/api/courses/${id}/lessons/${lid}` : null,
      onWatched: (bid) => {
        api(`/api/courses/${id}/lessons/${lid}/watched/${bid}`, { method: "POST" }).then(() => toast("✓ Видеог бүрэн үзлээ — одоо гүйлгэж болно")).catch(() => {});
      } });
    // Идэвхтэй суралцах хугацаа
    const need = (l.active_min || 0) * 60, cb = done && $("[data-complete]", done), meter = done && $(".active-need", done);
    if (meter) meter.hidden = !need;
    const onActive = (sec) => {
      if (!need || !cb || p?.completed_at) return;
      meter.textContent = `🕒 Идэвхтэй суралцсан: ${Math.floor(Math.min(sec, need) / 60)}/${l.active_min} мин`;
      meter.classList.toggle("ok", sec >= need);
      cb.disabled = sec < need;
      if (sec < need) cb.textContent = `🕒 ${l.active_min} мин идэвхтэй үзсний дараа дуусгана`; else cb.textContent = "✓ Энэ хичээлийг дууслаа";
    };
    stopWatch?.();
    stopWatch = l.exam ? null : watchLesson({ courseId: id, lessonId: lid, modal, onStop: () => closeModal(modal), onActive });
    reflectDone = !!(Auth.token && access?.progress?.[lid]?.completed_at) || isOwner || !Auth.token || !!l.exam;
    curLesson = reflectDone ? null : { id, lid, title: l.title };
    openModal(modal);
  };

  // Нүүр хуудаснаас #l=<хичээл> холбоосоор ирвэл тухайн хичээлийг шууд нээнэ (эрхгүй бол мөрийг тодруулна).
  const want = new URLSearchParams(location.hash.slice(1)).get("l");
  if (want) {
    const li = $(`.lesson[data-lesson="${CSS.escape(want)}"]`);
    if (li) { li.scrollIntoView({ block: "center" }); play(want).catch(() => li.animate([{ boxShadow: "0 0 0 4px rgba(31,60,143,.35)" }, { boxShadow: "0 0 0 0 transparent" }], { duration: 1600 })); }
  }

  document.addEventListener("click", async (e) => {
    const b = e.target.closest("[data-play]");
    if (!b) return;
    const li = b.closest(".lesson"), lid = li.dataset.lesson, price = +li.dataset.price;
    if (li.classList.contains("is-drip")) { li.animate([{ transform: "translateX(0)" }, { transform: "translateX(-8px)" }, { transform: "translateX(8px)" }, { transform: "translateX(0)" }], { duration: 350 }); toast($(".lesson-state", li).textContent); return; }
    if (!li.classList.contains("is-locked")) { try { await play(lid); } catch (err) { toast(err.message, true); } return; }
    if (!price) { // зөвхөн багцаар
      li.animate([{ transform: "translateX(0)" }, { transform: "translateX(-8px)" }, { transform: "translateX(8px)" }, { transform: "translateX(0)" }], { duration: 350 });
      toast("Энэ хичээл сургалтын багцад багтсан");
      buy.animate([{ transform: "scale(1)" }, { transform: "scale(1.08)" }, { transform: "scale(1)" }], { duration: 500 });
      return;
    }
    if (needLogin()) return;
    b.disabled = true;
    try {
      const d = await api(`/api/courses/${id}/lessons/${lid}/buy`, { method: "POST" });
      if (d.unlocked) { unlockLesson(lid); await play(lid); return; }
      pay(d.order, d.payment, $(".lesson-title", li).textContent, async () => { unlockLesson(lid); toast("🎉 Хичээл нээгдлээ!"); celebrate(); await play(lid).catch(() => {}); });
    } catch (err) { toast(err.message, true); } finally { b.disabled = false; }
  });
}

function celebrate() {
  if (reduce) return;
  const colors = ["#1f3c8f", "#eaa02e", "#2f55c0", "#f6c56f", "#14b8a6"];
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
window.SG = { ChatThread, pdfLib, BookReader, richHTML, blocksHTML, mountQuizzes, fmtBytes, extOf, $, $$, esc, api, Auth, toast, money, fmtDate, fmtTime, Live, mediaHTML, book3dHTML, hydrateBooks, Flipbook, openModal, closeModal, msgHTML, linkify, celebrate, reveals, counters, avatarHTML, ringHTML, hueOfName, fmtDay, WEEKDAYS, WEEKDAYS_SHORT };
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
if (page === "book") bookPage();
})();
