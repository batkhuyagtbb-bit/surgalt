/* Хянагчийн самбар (admin.html): тойм, багш нар, суралцагчид, гүйлгээ, орлого-зарлага, татах хүсэлт, мэдүүлэг.
   Тусдаа замаар нууц үгээр (HttpOnly cookie) эсвэл ADMIN_EMAILS-ийн бүртгэлээр (Bearer) нэвтэрнэ. */
(() => {
"use strict";
const $ = (s, r = document) => r.querySelector(s);
const $$ = (s, r = document) => [...r.querySelectorAll(s)];
const esc = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
const num = (n) => Number(n || 0).toLocaleString("en-US");
const tg = (n) => num(n) + "₮";
const pct = (a, b) => (b ? Math.round((a / b) * 100) : 0);
const bytes = (b) => { b = +b || 0; if (b < 1024) return b + " B"; const u = ["KB", "MB", "GB", "TB"]; let i = -1; do { b /= 1024; i++; } while (b >= 1024 && i < 3); return (b >= 100 ? Math.round(b) : b.toFixed(1)) + " " + u[i]; };
const pad = (n) => String(n).padStart(2, "0");
const dt = (t) => { if (!t) return "—"; const d = new Date(t); return `${d.getFullYear()}.${pad(d.getMonth() + 1)}.${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`; };
const dd = (t) => { if (!t) return "—"; const d = new Date(t); return `${d.getFullYear()}.${pad(d.getMonth() + 1)}.${pad(d.getDate())}`; };
const isoDay = (d) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
const MONTHS = ["1-р сар", "2-р сар", "3-р сар", "4-р сар", "5-р сар", "6-р сар", "7-р сар", "8-р сар", "9-р сар", "10-р сар", "11-р сар", "12-р сар"];
const monthName = (k) => { const [y, m] = k.split("-"); return `${y} ${MONTHS[+m - 1]}`; };

// Дүрс (Lucide маягийн шугаман SVG)
const ICONS = {
  chart: '<path d="M3 3v18h18"/><path d="M7 15l4-4 3 3 5-6"/>',
  teacher: '<circle cx="12" cy="7" r="3.5"/><path d="M5 21c0-4 3.1-6.5 7-6.5s7 2.5 7 6.5"/><path d="M15 3.5h5v3"/>',
  users: '<circle cx="9" cy="8" r="3.5"/><path d="M2.5 20c0-3.5 3-5.5 6.5-5.5s6.5 2 6.5 5.5"/><path d="M16 4.6a3.5 3.5 0 0 1 0 6.8M18 14.7c2 .7 3.5 2.4 3.5 5.3"/>',
  card: '<rect x="2" y="5" width="20" height="14" rx="2"/><path d="M2 10h20M6 15h4"/>',
  money: '<rect x="3" y="6" width="18" height="12" rx="2"/><circle cx="12" cy="12" r="2.5"/><path d="M7 9v.01M17 15v.01"/>',
  bank: '<path d="M3 10h18L12 4 3 10z"/><path d="M5 10v8M9.5 10v8M14.5 10v8M19 10v8M3 20h18"/>',
  flag: '<path d="M5 21V4M5 4h11l-2 4 2 4H5"/>',
  refresh: '<path d="M21 12a9 9 0 1 1-2.6-6.4L21 8"/><path d="M21 3v5h-5"/>',
  down: '<path d="M12 4v11M7 10l5 5 5-5M5 20h14"/>',
  ext: '<path d="M14 4h6v6M20 4l-9 9M18 14v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1h5"/>',
  search: '<circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/>',
  eye: '<path d="M2 12s3.6-7 10-7 10 7 10 7-3.6 7-10 7S2 12 2 12Z"/><circle cx="12" cy="12" r="3"/>',
  live: '<circle cx="12" cy="12" r="3"/><path d="M6.3 6.3a8 8 0 0 0 0 11.4M17.7 6.3a8 8 0 0 1 0 11.4"/>',
  db: '<ellipse cx="12" cy="5" rx="8" ry="3"/><path d="M4 5v6c0 1.7 3.6 3 8 3s8-1.3 8-3V5"/><path d="M4 11v6c0 1.7 3.6 3 8 3s8-1.3 8-3v-6"/>',
  trash: '<path d="M4 7h16M9 7V4h6v3M6 7l1 13h10l1-13"/>',
};
const ic = (n, s = 18) => `<svg viewBox="0 0 24 24" width="${s}" height="${s}" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${ICONS[n] || ""}</svg>`;

function toast(msg, err) {
  const t = document.createElement("div");
  t.className = "toast" + (err ? " err" : ""); t.textContent = msg;
  ($("#toasts") || document.body).append(t);
  setTimeout(() => { t.style.transition = ".4s"; t.style.opacity = "0"; }, 3200);
  setTimeout(() => t.remove(), 3700);
}

async function api(path, { method = "GET", body, raw } = {}) {
  const headers = {};
  let tok = null; try { tok = JSON.parse(localStorage.getItem("sg_token")); } catch {}
  if (tok) headers.Authorization = "Bearer " + tok;
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const res = await fetch(path, { method, headers, credentials: "same-origin", body: body !== undefined ? JSON.stringify(body) : undefined });
  if (raw) return res;
  const data = res.status === 204 ? null : await res.json().catch(() => null);
  if (!res.ok) {
    if ((res.status === 401 || res.status === 403) && !path.startsWith("/api/admin/login")) showLogin(true);
    const e = new Error(data?.error || "Алдаа гарлаа (" + res.status + ")"); e.status = res.status; throw e;
  }
  return data;
}

// Файл татах (cookie ба Bearer аль алинд ажиллана)
async function download(path, name) {
  const res = await api(path, { raw: true });
  if (!res.ok) { toast("Татаж чадсангүй (" + res.status + ")", true); return; }
  const a = document.createElement("a");
  a.href = URL.createObjectURL(await res.blob()); a.download = name;
  document.body.append(a); a.click(); a.remove(); setTimeout(() => URL.revokeObjectURL(a.href), 5000);
}

/* ---------- жижиг бүрэлдэхүүн ---------- */
const kpi = (label, value, sub = "", tone = "") => `<div class="kp ${tone}"><span>${esc(label)}</span><b>${value}</b>${sub ? `<small>${sub}</small>` : ""}</div>`;
const seg = (name, opts, cur) => `<div class="seg" data-seg="${name}">${opts.map(([v, l]) => `<button type="button" data-val="${v}" class="${String(v) === String(cur) ? "on" : ""}">${esc(l)}</button>`).join("")}</div>`;
const empty = (t) => `<p class="adm-empty">${esc(t)}</p>`;
const who = (p) => p && p.id ? `<span class="who"><b>${esc(p.name || p.username || "—")}</b>${p.username ? `<small>@${esc(p.username)}</small>` : ""}</span>` : `<span class="muted">—</span>`;
const avatar = (name, url) => url ? `<img class="av" src="${esc(url)}" alt="" loading="lazy">` : `<span class="av">${esc((name || "?").trim().slice(0, 1).toUpperCase())}</span>`;
const STATUS = { paid: ["Төлөгдсөн", "chip-teal"], pending: ["Хүлээгдэж буй", "chip-amber"], rejected: ["Татгалзсан", "chip-coral"] };
const chip = (st) => `<span class="chip ${STATUS[st]?.[1] || ""}">${esc(STATUS[st]?.[0] || st)}</span>`;

// Багана график: data[i][key]; second — ард нь цайвар багана (жишээ нь хуудас үзэлт)
function chart(data, { key, second, color = "brand", fmt = num, label = (d) => d.day, tip }) {
  if (!data.length) return empty("Өгөгдөл алга");
  const max = Math.max(1, ...data.map((d) => Math.max(+d[key] || 0, second ? +d[second] || 0 : 0)));
  const nice = (() => { const p = Math.pow(10, Math.floor(Math.log10(max))); return Math.ceil(max / p) * p; })();
  const step = Math.ceil(data.length / 10);
  const cols = data.map((d, i) => {
    const a = pct(+d[key] || 0, nice), b = second ? pct(+d[second] || 0, nice) : 0;
    const t = tip ? tip(d) : `${label(d)} · ${fmt(d[key])}`;
    return `<div class="ch-col" data-tip="${esc(t)}">${second ? `<i class="ch-b2" style="height:${b}%"></i>` : ""}<i class="ch-b ${color}" style="height:${Math.max(a, d[key] ? 1.5 : 0)}%"></i>${i % step === 0 ? `<em class="${(i / step) % 2 ? "alt" : ""}">${esc(label(d))}</em>` : ""}</div>`;
  }).join("");
  return `<div class="ch"><div class="ch-y"><span>${fmt(nice)}</span><span>${fmt(nice / 2)}</span><span>0</span></div><div class="ch-plot ${data.length > 60 ? "dense" : ""}">${cols}</div></div>`;
}
// Хоёр цуваа зэрэгцсэн (орлого / зарлага)
function chart2(data, { a, b, label, fmt = tg }) {
  if (!data.length) return empty("Өгөгдөл алга");
  const max = Math.max(1, ...data.map((d) => Math.max(+d[a] || 0, +d[b] || 0)));
  const nice = (() => { const p = Math.pow(10, Math.floor(Math.log10(max))); return Math.ceil(max / p) * p; })();
  return `<div class="ch"><div class="ch-y"><span>${fmt(nice)}</span><span>${fmt(nice / 2)}</span><span>0</span></div><div class="ch-plot pair">${data.map((d) =>
    `<div class="ch-col" data-tip="${esc(`${label(d)} · орлого ${fmt(d[a])} · зарлага ${fmt(d[b])}`)}"><i class="ch-b brand" style="height:${pct(+d[a] || 0, nice)}%"></i><i class="ch-b coral" style="height:${pct(+d[b] || 0, nice)}%"></i><em>${esc(label(d))}</em></div>`).join("")}</div></div>`;
}
// Жагсаалт + харьцангуй зурвас
const bars = (items, fmt = num) => {
  if (!items?.length) return empty("Өгөгдөл алга");
  const max = Math.max(1, ...items.map((x) => x.count));
  return `<ul class="hb">${items.map((x) => `<li><span class="hb-n" title="${esc(x.name)}">${esc(x.name || "—")}</span><span class="hb-v">${fmt(x.count)}</span><i style="width:${pct(x.count, max)}%"></i></li>`).join("")}</ul>`;
};

// Хүснэгтийн эрэмбэ: th[data-sort] дарахад
function sortable(table, rows, render, initial) {
  let key = initial?.[0], dir = initial?.[1] || -1;
  const draw = () => {
    const sorted = key ? [...rows].sort((x, y) => { const a = x[key], b = y[key]; return (typeof a === "string" || typeof b === "string" ? String(a ?? "").localeCompare(String(b ?? "")) : (+(a ?? 0) - +(b ?? 0))) * dir; }) : rows;
    $("tbody", table).innerHTML = sorted.length ? sorted.map(render).join("") : `<tr><td colspan="20" class="adm-empty-td">Илэрц алга</td></tr>`;
    $$("th[data-sort]", table).forEach((th) => th.dataset.dir = th.dataset.sort === key ? (dir > 0 ? "asc" : "desc") : "");
  };
  $("thead", table).addEventListener("click", (e) => {
    const th = e.target.closest("th[data-sort]"); if (!th) return;
    if (key === th.dataset.sort) dir = -dir; else { key = th.dataset.sort; dir = th.dataset.type === "text" ? 1 : -1; }
    draw();
  });
  draw();
  return { set(r) { rows = r; draw(); } };
}

/* ---------- самбарын бүтэц ---------- */
const state = { days: 30, months: 12, tx: { status: "paid", kind: "", from: "", to: "", q: "" }, me: null };
const VIEWS = {};
let viewSeq = 0;

function showLogin(expired) {
  $("#admShell").hidden = true; $("#admLogin").hidden = false;
  if (expired) { const e = $("#admLoginErr"); e.textContent = "Нэвтрэх хугацаа дууссан эсвэл эрхгүй байна. Дахин нэвтэрнэ үү."; e.hidden = false; }
  setTimeout(() => $("#admLoginForm [name=login]")?.focus(), 50);
}

async function start() {
  $("#admLogin").hidden = true; $("#admShell").hidden = false;
  $("#admWho").textContent = state.me?.login || "";
  route();
  badges();
}

async function badges() {
  try {
    const [ps, rs] = await Promise.all([api("/api/admin/payouts?status=pending"), api("/api/admin/reports?status=open")]);
    const set = (el, n) => { el.hidden = !n; el.textContent = n; };
    set($("#bPay"), ps.length); set($("#bRep"), rs.length);
  } catch {}
}

function route() {
  const v = (location.hash || "#overview").slice(1);
  const view = VIEWS[v] ? v : "overview";
  $$(".adm-nav a").forEach((a) => a.classList.toggle("on", a.dataset.v === view));
  const V = VIEWS[view];
  $("#admTitle").textContent = V.title; $("#admSub").textContent = V.sub || "";
  const tools = $("#admTools"), el = $("#admView");
  tools.innerHTML = ""; tools.onclick = el.onclick = el.onsubmit = null; el.innerHTML = `<div class="adm-loading"><i></i><i></i><i></i></div>`;
  const seq = ++viewSeq;
  Promise.resolve(V.render(tools, el, () => seq === viewSeq)).catch((x) => { if (seq === viewSeq) el.innerHTML = empty(x.message); });
  $("#admMain").scrollTo?.(0, 0); window.scrollTo(0, 0);
}

/* ---------- Тойм ---------- */
VIEWS.overview = {
  title: "Тойм", sub: "Сайтын хандалт, хэрэглэгч, орлого, агуулга нэг дор",
  async render(tools, el, live) {
    tools.innerHTML = seg("days", [[7, "7 хоног"], [30, "30 хоног"], [90, "90 хоног"], [365, "1 жил"]], state.days) + `<button type="button" class="btn btn-ghost btn-sm icon-btn2" data-refresh title="Шинэчлэх">${ic("refresh", 16)}</button>`;
    tools.onclick = (e) => {
      const b = e.target.closest("[data-seg=days] button"); if (b) { state.days = +b.dataset.val; route(); }
      if (e.target.closest("[data-refresh]")) route();
    };
    const d = await api("/api/admin/overview?days=" + state.days);
    if (!live()) return;
    const T = d.traffic, U = d.users, I = d.income, C = d.content;
    const per = state.days === 365 ? "1 жилд" : state.days + " хоногт";
    const dayLbl = (x) => x.day.slice(5).replace("-", ".");
    el.innerHTML = `
    <h2 class="adm-h">Хандалт</h2>
    <section class="kp-grid">
      ${kpi("Одоо сайтад", num(T.online.unique), "сүүлийн 5 минутад", "live")}
      ${kpi("Өнөөдөр", num(T.today.unique) + " <small>хүн</small>", num(T.today.views) + " хуудас үзэлт")}
      ${kpi("Сүүлийн 7 хоног", num(T.week.unique) + " <small>хүн</small>", num(T.week.views) + " хуудас үзэлт")}
      ${kpi("Сүүлийн 30 хоног", num(T.month.unique) + " <small>хүн</small>", num(T.month.views) + " хуудас үзэлт")}
    </section>
    <section class="adm-card">
      <header><h3>Өдөр бүрийн зочин</h3><span class="lg"><i class="brand"></i>Давтагдаагүй зочин <i class="soft"></i>Хуудас үзэлт</span></header>
      ${chart(d.daily, { key: "unique", second: "views", label: dayLbl, tip: (x) => `${x.day} · ${num(x.unique)} зочин · ${num(x.views)} үзэлт · ${num(x.members)} нэвтэрсэн` })}
      <p class="muted small">${per}: ${num(T.period.unique)} давтагдаагүй зочин, ${num(T.period.views)} хуудас үзэлт. Хөтчөөр тоолно (бот, хайлтын систем тоологдохгүй).</p>
    </section>
    <div class="adm-grid3">
      <section class="adm-card"><header><h3>Их үзсэн хуудас</h3></header>${bars(T.pages)}</section>
      <section class="adm-card"><header><h3>Хаанаас ирсэн</h3></header>${bars(T.referrers)}</section>
      <section class="adm-card"><header><h3>Төхөөрөмж</h3></header>${bars((T.devices || []).map((x) => ({ ...x, name: { mobile: "Утас", desktop: "Компьютер", tablet: "Таблет" }[x.name] || x.name })))}</section>
    </div>

    <h2 class="adm-h">Хэрэглэгч ба орлого</h2>
    <section class="kp-grid">
      ${kpi("Нийт хэрэглэгч", num(U.total), `${num(U.teacher || 0)} багш · ${num(U.student || 0)} суралцагч`)}
      ${kpi("Шинэ бүртгэл", num(U.today || 0) + " <small>өнөөдөр</small>", `7 хоногт ${num(U.week || 0)} · 30 хоногт ${num(U.month || 0)}`)}
      ${kpi("Орлого (өнөөдөр)", tg(I.today.total), `7 хоногт ${tg(I.week.total)} · 30 хоногт ${tg(I.month.total)}`, "good")}
      ${kpi("Нийт орлого", tg(I.all.total), `${num(I.all.count)} гүйлгээ · хураамж ${tg(I.fees?.total)} · хүлээгдэж буй ${num(I.pending.count)}`)}
      ${kpi("Багшид шилжүүлэх", tg(d.payouts.pending.total), `${num(d.payouts.pending.count)} хүсэлт хүлээгдэж байна`, d.payouts.pending.count ? "warn" : "")}
      ${kpi("Багшид шилжүүлсэн", tg(d.payouts.paid.total), `${num(d.payouts.paid.count)} шилжүүлэг`)}
    </section>
    <div class="adm-grid2">
      <section class="adm-card"><header><h3>Өдөр бүрийн орлого</h3><span class="muted small">${per}: ${tg(I.period.total)}</span></header>${chart(d.daily, { key: "revenue", color: "teal", fmt: tg, label: dayLbl, tip: (x) => `${x.day} · ${tg(x.revenue)} · ${x.orders} гүйлгээ` })}</section>
      <section class="adm-card"><header><h3>Өдөр бүрийн шинэ бүртгэл</h3></header>${chart(d.daily, { key: "signups", color: "amber", label: dayLbl, tip: (x) => `${x.day} · ${x.signups} бүртгэл` })}</section>
    </div>

    <h2 class="adm-h">Агуулга</h2>
    <section class="kp-grid small">
      ${kpi("Сургалт", num(C.courses || 0), `${num(C.courses_published || 0)} нийтэлсэн`)}
      ${kpi("Хичээл", num(C.lessons || 0), `${num(C.course_views || 0)} сургалтын үзэлт`)}
      ${kpi("Ном, өгүүлэл", num(C.books || 0), `${num(C.books_published || 0)} нийтэлсэн`)}
      ${kpi("Файл", num(C.files || 0), bytes(C.file_bytes))}
      ${kpi("Элсэлт", num(C.enrollments || 0), `${num(C.lesson_purchases || 0)} хичээл тусад нь`)}
      ${kpi("Нээлттэй мэдүүлэг", num(d.open_reports), d.open_reports ? `<a href="#reports" class="link">Шийдвэрлэх →</a>` : "алга", d.open_reports ? "warn" : "")}
    </section>

    <div class="adm-grid2">
      <section class="adm-card"><header><h3>Шилдэг багш (борлуулалтаар)</h3><a class="link small" href="#teachers">Бүгд →</a></header>
        ${d.top_teachers.length ? `<ol class="rank">${d.top_teachers.map((t) => `<li data-teacher="${esc(t.id)}">${who(t)}<b>${tg(t.revenue)}</b></li>`).join("")}</ol>` : empty("Борлуулалт алга")}</section>
      <section class="adm-card"><header><h3>Сүүлийн гүйлгээ</h3><a class="link small" href="#transactions">Бүгд →</a></header>
        ${d.recent_orders.length ? `<ul class="feed">${d.recent_orders.map((o) => `<li><span><b>${esc(o.title || o.kind_label)}</b><small>${esc(o.buyer.name || "—")} → ${esc(o.teacher.name || "платформ")} · ${dt(o.paid_at || o.created_at)}</small></span><b>${tg(o.amount)}</b></li>`).join("")}</ul>` : empty("Гүйлгээ алга")}</section>
    </div>
    <section class="adm-card"><header><h3>Сүүлд бүртгүүлсэн</h3></header>
      ${d.recent_users.length ? `<ul class="feed">${d.recent_users.map((u) => `<li data-${u.role === "teacher" ? "teacher" : "student"}="${esc(u.id)}"><span><b>${esc(u.name || u.username)}</b><small>@${esc(u.username)} · ${dt(u.created_at)}</small></span><span class="chip ${u.role === "teacher" ? "chip-gold" : ""}">${u.role === "teacher" ? "Багш" : "Суралцагч"}</span></li>`).join("")}</ul>` : empty("Бүртгэл алга")}</section>`;
  },
};

/* ---------- Багш нар ---------- */
VIEWS.teachers = {
  title: "Багш нар", sub: "Агуулга, суралцагч, борлуулалт, татсан ба үлдэгдэл — мөрөн дээр дарж дэлгэрэнгүйг харна",
  async render(tools, el, live) {
    tools.innerHTML = `<label class="adm-search">${ic("search", 16)}<input type="search" placeholder="Нэр, имэйл, @нэр" aria-label="Багш хайх"></label>`;
    const d = await api("/api/admin/teachers");
    if (!live()) return;
    const all = d.teachers;
    const sum = (k) => all.reduce((n, t) => n + (t[k] || 0), 0);
    el.innerHTML = `
    <section class="kp-grid">
      ${kpi("Нийт багш", num(all.length), `${num(all.filter((t) => t.courses_published).length)} нь нийтэлсэн сургалттай`)}
      ${kpi("Агуулга", num(sum("courses")) + " <small>сургалт</small>", `${num(sum("lessons"))} хичээл · ${num(sum("books"))} ном · ${num(sum("files"))} файл`)}
      ${kpi("Нийт борлуулалт", tg(sum("revenue")), `${num(sum("sales"))} гүйлгээ`, "good")}
      ${kpi("Татсан (шилжүүлсэн)", tg(sum("paid_out")), `хүлээгдэж буй ${tg(sum("pending"))}`)}
      ${kpi("Багшийн үлдэгдэл", tg(sum("available")), `шимтгэл ${tg(sum("commission"))} хассан, татах боломжтой`)}
    </section>
    <div class="adm-table-wrap"><table class="adm-table" id="tTable"><thead><tr>
      <th data-sort="name" data-type="text">Багш</th><th data-sort="courses">Сургалт</th><th data-sort="lessons">Хичээл</th><th data-sort="books">Ном</th>
      <th data-sort="file_bytes">Файл</th><th data-sort="students">Суралцагч</th><th data-sort="revenue">Борлуулалт</th><th data-sort="paid_out">Татсан</th>
      <th data-sort="pending">Хүлээгдэж буй</th><th data-sort="available">Үлдэгдэл</th><th data-sort="profile_views">Үзэлт</th><th data-sort="created_at" data-type="text">Бүртгүүлсэн</th>
    </tr></thead><tbody></tbody></table></div>`;
    const row = (t) => `<tr data-teacher="${esc(t.id)}" tabindex="0">
      <td><span class="adm-person" title="${esc(t.email || "")}">${avatar(t.name, t.avatar_url)}<span><b>${esc(t.name || t.username)}</b><small>@${esc(t.username)}</small></span></span></td>
      <td class="n">${num(t.courses)}${t.courses ? `<small>${num(t.courses_published)} нийтэлсэн</small>` : ""}</td><td class="n">${num(t.lessons)}</td><td class="n">${num(t.books)}</td>
      <td class="n">${num(t.files)}<small>${bytes(t.file_bytes)}</small></td><td class="n">${num(t.students)}</td>
      <td class="n"><b>${tg(t.revenue)}</b><small>${num(t.sales)} удаа</small></td><td class="n">${tg(t.paid_out)}</td>
      <td class="n ${t.pending ? "warn" : ""}">${tg(t.pending)}</td><td class="n">${tg(t.available)}</td>
      <td class="n">${num(t.profile_views)}<small>${num(t.course_views)} сургалт</small></td><td>${dd(t.created_at)}</td></tr>`;
    const tbl = sortable($("#tTable"), all, row);
    $("input", tools).addEventListener("input", (e) => {
      const q = e.target.value.trim().toLowerCase();
      tbl.set(q ? all.filter((t) => `${t.name} ${t.username} ${t.email}`.toLowerCase().includes(q)) : all);
    });
  },
};

async function openTeacher(id) {
  const body = drawer("Багш", `<div class="adm-loading"><i></i><i></i><i></i></div>`);
  let d; try { d = await api("/api/admin/teachers/" + encodeURIComponent(id)); } catch (x) { body.innerHTML = empty(x.message); return; }
  const t = d.teacher, p = d.profile;
  $("#admDrawerTitle").textContent = t.name || t.username;
  const tabs = [
    ["courses", `Сургалт (${d.courses.length})`], ["books", `Ном (${d.books.length})`], ["files", `Файл (${d.files.length})`],
    ["sales", `Борлуулалт (${d.sales.length})`], ["payouts", `Татсан (${d.payouts.length})`], ["students", `Суралцагч (${d.students.length})`],
  ];
  const PST = { pending: "Хүлээгдэж буй", paid: "Шилжүүлсэн", rejected: "Татгалзсан" };
  const panes = {
    courses: d.courses.length ? `<div class="adm-table-wrap"><table class="adm-table mini"><thead><tr><th>Сургалт</th><th>Үнэ</th><th>Хичээл</th><th>Суралцагч</th><th>Үзэлт</th><th>Борлуулалт</th></tr></thead><tbody>${d.courses.map((c) =>
      `<tr><td><a class="link" href="/c/${esc(c.id)}" target="_blank" rel="noopener">${esc(c.title)}</a><small>${c.published ? "Нийтэлсэн" : "Ноорог"} · ${dd(c.created_at)}</small></td><td class="n">${c.price ? tg(c.price) : "Үнэгүй"}</td><td class="n">${num(c.lessons)}</td><td class="n">${num(c.students)}</td><td class="n">${num(c.views)}</td><td class="n"><b>${tg(c.revenue)}</b><small>${num(c.sales)} удаа</small></td></tr>`).join("")}</tbody></table></div>` : empty("Сургалт алга"),
    books: d.books.length ? `<div class="adm-table-wrap"><table class="adm-table mini"><thead><tr><th>Ном</th><th>Үнэ</th><th>Хуудас</th><th>Үзэлт</th><th>Уншсан</th><th>Борлуулалт</th></tr></thead><tbody>${d.books.map((b) =>
      `<tr><td><a class="link" href="/b/${esc(b.id)}" target="_blank" rel="noopener">${esc(b.title)}</a><small>${b.kind === "article" ? "Өгүүлэл" : "Ном"} · ${b.published ? "Нийтэлсэн" : "Ноорог"}</small></td><td class="n">${b.price ? tg(b.price) : "Үнэгүй"}</td><td class="n">${num(b.pages)}</td><td class="n">${num(b.views)}</td><td class="n">${num(b.reads)}</td><td class="n"><b>${tg(b.revenue)}</b><small>${num(b.sales)} удаа</small></td></tr>`).join("")}</tbody></table></div>` : empty("Ном алга"),
    files: d.files.length ? `<div class="adm-table-wrap"><table class="adm-table mini"><thead><tr><th>Файл</th><th>Хэмжээ</th><th>Төрөл</th><th>Огноо</th></tr></thead><tbody>${d.files.map((f) =>
      `<tr><td>${f.url ? `<a class="link" href="${esc(f.url)}" target="_blank" rel="noopener">${esc(f.name)}</a>` : esc(f.name)}<small>${f.visibility === "private" ? "Хаалттай" : "Нээлттэй"}${f.status && f.status !== "ready" ? " · " + esc(f.status) : ""}</small></td><td class="n">${bytes(f.size)}</td><td>${esc((f.type || "").split(";")[0])}</td><td>${dd(f.at)}</td></tr>`).join("")}</tbody></table></div>` : empty("Файл алга"),
    sales: d.sales.length ? `<div class="adm-table-wrap"><table class="adm-table mini"><thead><tr><th>Огноо</th><th>Юу</th><th>Худалдан авагч</th><th>Дүн</th></tr></thead><tbody>${d.sales.map((o) =>
      `<tr><td>${dt(o.paid_at || o.created_at)}</td><td>${esc(o.title || "—")}<small>${esc(o.kind_label)}</small></td><td>${who(o.buyer)}</td><td class="n"><b>${tg(o.amount)}</b></td></tr>`).join("")}</tbody></table></div>` : empty("Борлуулалт алга"),
    payouts: d.payouts.length ? `<div class="adm-table-wrap"><table class="adm-table mini"><thead><tr><th>Огноо</th><th>Дүн</th><th>Данс</th><th>Төлөв</th></tr></thead><tbody>${d.payouts.map((x) =>
      `<tr><td>${dt(x.created_at)}<small>${x.auto ? "Хуваариар" : "Гараар"}</small></td><td class="n"><b>${tg(x.amount)}</b></td><td>${esc(x.bank)} ${esc(x.account_no)}<small>${esc(x.account_name)}</small></td><td><span class="chip ${x.status === "paid" ? "chip-teal" : x.status === "pending" ? "chip-amber" : "chip-coral"}">${PST[x.status] || esc(x.status)}</span>${x.processed_at ? `<small>${dt(x.processed_at)}${x.note ? " · " + esc(x.note) : ""}</small>` : ""}</td></tr>`).join("")}</tbody></table></div>` : empty("Татсан түүх алга"),
    students: d.students.length ? `<div class="adm-table-wrap"><table class="adm-table mini"><thead><tr><th>Суралцагч</th><th>Сургалт</th><th>Анх элссэн</th></tr></thead><tbody>${d.students.map((s) =>
      `<tr data-student="${esc(s.id)}" tabindex="0"><td>${who(s)}</td><td class="n">${num(s.courses)}</td><td>${dd(s.since)}</td></tr>`).join("")}</tbody></table></div>` : empty("Суралцагч алга"),
  };
  body.innerHTML = `
    <div class="dr-head">${avatar(t.name, t.avatar_url)}<div><b>${esc(t.name || t.username)}</b><small>@${esc(t.username)} · ${esc(t.email || "имэйлгүй")}${p.phone ? " · " + esc(p.phone) : ""}</small><small>${p.headline ? esc(p.headline) + " · " : ""}${p.location ? esc(p.location) + " · " : ""}Бүртгүүлсэн ${dd(t.created_at)}</small></div>
      <a class="btn btn-ghost btn-sm" href="/t/${esc(t.username)}" target="_blank" rel="noopener">${ic("ext", 15)} Профайл</a></div>
    <section class="kp-grid small">
      ${kpi("Борлуулалт", tg(t.revenue), `${num(t.sales)} удаа`, "good")}
      ${kpi("Татсан", tg(t.paid_out), "данс руу шилжүүлсэн")}
      ${kpi("Хүлээгдэж буй", tg(t.pending), "татах хүсэлт", t.pending ? "warn" : "")}
      ${kpi("Платформын шимтгэл", tg(t.commission), "борлуулалтаас")}
      ${kpi("Үлдэгдэл", tg(t.available), "шимтгэл хассан, татаж болох")}
      ${kpi("Суралцагч", num(t.students), `${num(t.profile_views)} профайл үзэлт`)}
      ${kpi("Агуулга", `${num(t.courses)} <small>сургалт</small>`, `${num(t.lessons)} хичээл · ${num(t.books)} ном · ${num(t.files)} файл (${bytes(t.file_bytes)})`)}
    </section>
    <dl class="dr-acct">${d.account ? `<div><dt>Банк</dt><dd>${esc(d.account.bank)}</dd></div><div><dt>Данс</dt><dd><code>${esc(d.account.account_no)}</code></dd></div><div><dt>Эзэмшигч</dt><dd>${esc(d.account.account_name)}</dd></div><div><dt>Татах хуваарь</dt><dd>${esc({ manual: "Гараар", instant: "Шууд", daily: "Өдөр бүр", weekly: "7 хоног бүр", monthly: "Сар бүр", quarterly: "Улирал бүр" }[d.account.schedule] || d.account.schedule)}</dd></div>` : `<div><dt>Данс</dt><dd class="muted">Холбоогүй</dd></div>`}
      ${p.storage_extra_bytes ? `<div><dt>Нэмэлт багтаамж</dt><dd>${bytes(p.storage_extra_bytes)}${p.storage_expires_at ? " · " + dd(p.storage_expires_at) + " хүртэл" : ""}</dd></div>` : ""}</dl>
    <nav class="dr-tabs">${tabs.map(([k, l], i) => `<button type="button" data-tab="${k}" class="${i ? "" : "on"}">${esc(l)}</button>`).join("")}</nav>
    <div class="dr-pane">${panes.courses}</div>`;
  $(".dr-tabs", body).addEventListener("click", (e) => {
    const b = e.target.closest("[data-tab]"); if (!b) return;
    $$(".dr-tabs button", body).forEach((x) => x.classList.toggle("on", x === b));
    $(".dr-pane", body).innerHTML = panes[b.dataset.tab];
  });
}

/* ---------- Суралцагчид ---------- */
VIEWS.students = {
  title: "Суралцагчид", sub: "Бүх суралцагч, элсэлт ба зарцуулсан дүн — мөрөн дээр дарж дэлгэрэнгүйг харна",
  async render(tools, el, live) {
    tools.innerHTML = seg("sf", [["all", "Бүгд"], ["paying", "Төлбөр төлсөн"], ["enrolled", "Элссэн"], ["idle", "Элсээгүй"]], "all") +
      `<label class="adm-search">${ic("search", 16)}<input type="search" placeholder="Нэр, имэйл, @нэр" aria-label="Суралцагч хайх"></label>`;
    const d = await api("/api/admin/students");
    if (!live()) return;
    const all = d.students, month = Date.now() - 30 * 864e5;
    const paying = all.filter((s) => s.spent > 0);
    el.innerHTML = `
    <section class="kp-grid">
      ${kpi("Нийт суралцагч", num(all.length), `30 хоногт ${num(all.filter((s) => new Date(s.created_at) >= month).length)} шинэ`)}
      ${kpi("Элссэн", num(all.filter((s) => s.courses || s.lessons).length), `${pct(all.filter((s) => s.courses || s.lessons).length, all.length)}% нь сургалтад элссэн`)}
      ${kpi("Төлбөр төлсөн", num(paying.length), `${pct(paying.length, all.length)}% нь худалдан авалт хийсэн`, "good")}
      ${kpi("Нийт зарцуулсан", tg(paying.reduce((n, s) => n + s.spent, 0)), paying.length ? `нэг хүнд дунджаар ${tg(Math.round(paying.reduce((n, s) => n + s.spent, 0) / paying.length))}` : "")}
    </section>
    <div class="adm-table-wrap"><table class="adm-table" id="sTable"><thead><tr>
      <th data-sort="name" data-type="text">Суралцагч</th><th data-sort="courses">Сургалт</th><th data-sort="lessons">Хичээл</th><th data-sort="teachers">Багш</th>
      <th data-sort="orders">Худалдан авалт</th><th data-sort="spent">Зарцуулсан</th><th data-sort="last_paid_at" data-type="text">Сүүлд төлсөн</th><th data-sort="created_at" data-type="text">Бүртгүүлсэн</th>
    </tr></thead><tbody></tbody></table></div>`;
    const row = (s) => `<tr data-student="${esc(s.id)}" tabindex="0">
      <td><span class="adm-person">${avatar(s.name || s.username)}<span><b>${esc(s.name || s.username)}</b><small>@${esc(s.username)}${s.email ? " · " + esc(s.email) : ""}</small></span></span></td>
      <td class="n">${num(s.courses)}</td><td class="n">${num(s.lessons)}</td><td class="n">${num(s.teachers)}</td><td class="n">${num(s.orders)}</td>
      <td class="n"><b>${tg(s.spent)}</b></td><td>${s.last_paid_at ? dd(s.last_paid_at) : "—"}</td><td>${dd(s.created_at)}</td></tr>`;
    const tbl = sortable($("#sTable"), all, row);
    let f = "all", q = "";
    const apply = () => tbl.set(all.filter((s) => (f === "all" || (f === "paying" && s.spent > 0) || (f === "enrolled" && (s.courses || s.lessons)) || (f === "idle" && !s.courses && !s.lessons))
      && (!q || `${s.name} ${s.username} ${s.email}`.toLowerCase().includes(q))));
    tools.onclick = ((e) => { const b = e.target.closest("[data-seg=sf] button"); if (!b) return; f = b.dataset.val; $$("[data-seg=sf] button", tools).forEach((x) => x.classList.toggle("on", x === b)); apply(); });
    $("input", tools).addEventListener("input", (e) => { q = e.target.value.trim().toLowerCase(); apply(); });
  },
};

async function openStudent(id) {
  const body = drawer("Суралцагч", `<div class="adm-loading"><i></i><i></i><i></i></div>`);
  let d; try { d = await api("/api/admin/students/" + encodeURIComponent(id)); } catch (x) { body.innerHTML = empty(x.message); return; }
  const s = d.student;
  $("#admDrawerTitle").textContent = s.name || s.username;
  body.innerHTML = `
    <div class="dr-head">${avatar(s.name || s.username)}<div><b>${esc(s.name || s.username)}</b><small>@${esc(s.username)} · ${esc(s.email || "имэйлгүй")}${d.profile.phone ? " · " + esc(d.profile.phone) : ""}</small><small>Бүртгүүлсэн ${dt(s.created_at)}</small></div></div>
    <section class="kp-grid small">
      ${kpi("Сургалт", num(s.courses), `${num(s.lessons)} хичээл тусад нь`)}
      ${kpi("Багш", num(s.teachers))}
      ${kpi("Зарцуулсан", tg(s.spent), `${num(s.orders)} худалдан авалт`, "good")}
    </section>
    <h3 class="dr-h">Элсэлт</h3>
    ${d.enrollments.length ? `<div class="adm-table-wrap"><table class="adm-table mini"><thead><tr><th>Сургалт</th><th>Багш</th><th>Огноо</th></tr></thead><tbody>${d.enrollments.map((e) =>
      `<tr><td><a class="link" href="/c/${esc(e.course_id)}" target="_blank" rel="noopener">${esc(e.course || "—")}</a><small>${e.lesson ? "Нэг хичээл" : "Бүтэн сургалт"}</small></td><td>${who(e.teacher)}</td><td>${dd(e.at)}</td></tr>`).join("")}</tbody></table></div>` : empty("Элсэлт алга")}
    <h3 class="dr-h">Захиалга, төлбөр</h3>
    ${d.orders.length ? `<div class="adm-table-wrap"><table class="adm-table mini"><thead><tr><th>Огноо</th><th>Юу</th><th>Багш</th><th>Дүн</th><th>Төлөв</th></tr></thead><tbody>${d.orders.map((o) =>
      `<tr><td>${dt(o.paid_at || o.created_at)}</td><td>${esc(o.title || "—")}<small>${esc(o.kind_label)}</small></td><td>${who(o.teacher)}</td><td class="n"><b>${tg(o.amount)}</b></td><td>${chip(o.status)}</td></tr>`).join("")}</tbody></table></div>` : empty("Захиалга алга")}`;
}

/* ---------- Гүйлгээ ---------- */
VIEWS.transactions = {
  title: "Гүйлгээ", sub: "Бүх захиалга, төлбөр — шүүж, CSV-ээр татна",
  async render(tools, el, live) {
    const f = state.tx;
    tools.innerHTML = seg("st", [["paid", "Төлөгдсөн"], ["pending", "Хүлээгдэж буй"], ["all", "Бүгд"]], f.status) +
      `<button type="button" class="btn btn-ghost btn-sm" data-csv>${ic("down", 16)} CSV</button>`;
    el.innerHTML = `<form class="adm-filters" id="txF">
      <label>Төрөл<select name="kind"><option value="">Бүгд</option></select></label>
      <label>Эхлэх<input type="date" name="from" value="${esc(f.from)}"></label>
      <label>Дуусах<input type="date" name="to" value="${esc(f.to)}"></label>
      <label class="grow">Хайх<input type="search" name="q" value="${esc(f.q)}" placeholder="Нэр, худалдан авагч, багш, захиалгын дугаар"></label>
      <div class="adm-quick"><button type="button" data-range="0">Өнөөдөр</button><button type="button" data-range="6">7 хоног</button><button type="button" data-range="29">30 хоног</button><button type="button" data-range="month">Энэ сар</button><button type="button" data-range="">Бүх хугацаа</button></div>
    </form><div id="txOut"><div class="adm-loading"><i></i><i></i><i></i></div></div>`;
    const form = $("#txF");
    const qs = () => new URLSearchParams({ status: f.status, kind: f.kind, from: f.from, to: f.to, q: f.q }).toString();
    let kindsFilled = false, t;
    const load = async () => {
      const d = await api("/api/admin/transactions?" + qs());
      if (!live()) return;
      if (!kindsFilled) { form.kind.insertAdjacentHTML("beforeend", d.kinds.map((k) => `<option value="${esc(k.key)}">${esc(k.label)}</option>`).join("")); form.kind.value = f.kind; kindsFilled = true; }
      $("#txOut").innerHTML = `
        <section class="kp-grid">
          ${kpi("Төлөгдсөн", tg(d.totals.paid.total), `${num(d.totals.paid.count)} гүйлгээ`, "good")}
          ${kpi("Хүлээгдэж буй", tg(d.totals.pending.total), `${num(d.totals.pending.count)} төлөгдөөгүй захиалга`)}
          ${kpi("Үйлчилгээний хураамж", tg(d.totals.fees?.total), `${num(d.totals.fees?.count)} гүйлгээнээс`)}
          ${kpi("Дундаж", tg(d.totals.paid.count ? Math.round(d.totals.paid.total / d.totals.paid.count) : 0), "нэг гүйлгээнд")}
        </section>
        ${d.by_kind.length ? `<div class="kchips">${d.by_kind.map((k) => `<span><b>${esc(k.label)}</b>${tg(k.total)} · ${num(k.count)}</span>`).join("")}</div>` : ""}
        <div class="adm-table-wrap"><table class="adm-table" id="txT"><thead><tr><th data-sort="t" data-type="text">Огноо</th><th data-sort="title" data-type="text">Юу</th><th data-sort="buyerN" data-type="text">Худалдан авагч</th><th data-sort="teacherN" data-type="text">Багш</th><th data-sort="amount">Дүн</th><th data-sort="status" data-type="text">Төлөв</th></tr></thead><tbody></tbody></table></div>
        ${d.matched > d.rows.length ? `<p class="muted small">${num(d.matched)} гүйлгээнээс эхний ${num(d.rows.length)}-г харууллаа. Бүгдийг CSV-ээр татна уу.</p>` : ""}`;
      const rows = d.rows.map((o) => ({ ...o, t: o.paid_at || o.created_at, buyerN: o.buyer.name, teacherN: o.teacher.name }));
      sortable($("#txT"), rows, (o) => `<tr><td>${dt(o.t)}<small class="mono">${esc(o.id.slice(0, 12))}</small></td><td>${esc(o.title || "—")}<small>${esc(o.kind_label)}</small></td>
        <td ${o.buyer.id ? `data-student="${esc(o.buyer.id)}" class="clk"` : ""}>${who(o.buyer)}</td><td ${o.teacher.id ? `data-teacher="${esc(o.teacher.id)}" class="clk"` : ""}>${o.teacher.id ? who(o.teacher) : `<span class="muted">Платформ</span>`}</td>
        <td class="n"><b>${tg(o.amount + (o.fee || 0))}</b>${o.fee ? `<small>${tg(o.amount)} + ${tg(o.fee)}</small>` : ""}</td><td>${chip(o.status)}</td></tr>`);
    };
    const reload = () => { clearTimeout(t); t = setTimeout(() => load().catch((x) => ($("#txOut").innerHTML = empty(x.message))), 250); };
    form.addEventListener("input", (e) => { f[e.target.name] = e.target.value; reload(); });
    form.addEventListener("submit", (e) => e.preventDefault());
    form.addEventListener("click", (e) => {
      const b = e.target.closest("[data-range]"); if (!b) return;
      const now = new Date(), r = b.dataset.range;
      if (r === "") { f.from = f.to = ""; }
      else if (r === "month") { f.from = isoDay(new Date(now.getFullYear(), now.getMonth(), 1)); f.to = isoDay(now); }
      else { f.from = isoDay(new Date(now.getFullYear(), now.getMonth(), now.getDate() - +r)); f.to = isoDay(now); }
      form.from.value = f.from; form.to.value = f.to; reload();
    });
    tools.onclick = ((e) => {
      const b = e.target.closest("[data-seg=st] button");
      if (b) { f.status = b.dataset.val; $$("[data-seg=st] button", tools).forEach((x) => x.classList.toggle("on", x === b)); reload(); }
      if (e.target.closest("[data-csv]")) download("/api/admin/transactions?" + qs() + "&format=csv", `guilgee-${isoDay(new Date())}.csv`);
    });
    await load();
  },
};

/* ---------- Орлого, зарлага ---------- */
VIEWS.finance = {
  title: "Орлого, зарлага", sub: "Сар бүрийн орлого (борлуулалт, багтаамж), зарлага (багшид шилжүүлсэн, платформын зардал), цэвэр дүн",
  async render(tools, el, live) {
    tools.innerHTML = seg("months", [[6, "6 сар"], [12, "12 сар"], [24, "24 сар"]], state.months);
    tools.onclick = (e) => { const b = e.target.closest("[data-seg=months] button"); if (b) { state.months = +b.dataset.val; route(); } };
    const d = await api("/api/admin/finance?months=" + state.months);
    if (!live()) return;
    const R = d.range, A = d.all;
    el.innerHTML = `
    <section class="kp-grid">
      ${kpi("Орлого", tg(R.income), `борлуулалт ${tg(R.sales)} · багтаамж ${tg(R.storage)} · хураамж ${tg(R.fees)}`, "good")}
      ${kpi("Зарлага", tg(R.outgo), `багшид ${tg(R.payouts)} · зардал ${tg(R.expenses)}`, "bad")}
      ${kpi("Цэвэр", tg(R.net), `сүүлийн ${state.months} сард`, R.net < 0 ? "bad" : "")}
      ${kpi("Багшид өгөх үлдэгдэл", tg(A.owed), `шимтгэл ${d.commission_pct}% хассан · хүсэлт гаргасан ${tg(A.pending)}`, A.pending ? "warn" : "")}
      ${kpi("Платформын ашиг", tg(A.profit), `шимтгэл ${tg(A.commission)} + хураамж ${tg(A.fees)} + багтаамж − зардал`)}
      ${kpi("Данс дахь мөнгө", tg(A.cash), "нийт орлого − шилжүүлсэн − зардал")}
    </section>
    <section class="adm-card"><header><h3>Сар бүрийн орлого ба зарлага</h3><span class="lg"><i class="brand"></i>Орлого <i class="coral"></i>Зарлага</span></header>
      ${chart2(d.months, { a: "income", b: "outgo", label: (m) => MONTHS[+m.month.slice(5) - 1].replace("-р сар", "") })}</section>
    <div class="adm-table-wrap"><table class="adm-table"><thead><tr><th>Сар</th><th>Борлуулалт</th><th>Шимтгэл</th><th>Багтаамж</th><th>Хураамж</th><th>Нийт орлого</th><th>Багшид шилжүүлсэн</th><th>Зардал</th><th>Нийт зарлага</th><th>Цэвэр</th></tr></thead><tbody>
      ${[...d.months].reverse().map((m) => `<tr><td>${monthName(m.month)}<small>${num(m.orders)} гүйлгээ</small></td><td class="n">${tg(m.sales)}</td><td class="n">${tg(m.commission)}</td><td class="n">${tg(m.storage)}</td><td class="n">${tg(m.fees)}</td><td class="n"><b>${tg(m.income)}</b></td><td class="n">${tg(m.payouts)}</td><td class="n">${tg(m.expenses)}</td><td class="n">${tg(m.outgo)}</td><td class="n ${m.net < 0 ? "bad" : "good"}"><b>${tg(m.net)}</b></td></tr>`).join("")}
      <tr class="tot"><td>Нийт</td><td class="n">${tg(R.sales)}</td><td class="n">${tg(R.commission)}</td><td class="n">${tg(R.storage)}</td><td class="n">${tg(R.fees)}</td><td class="n">${tg(R.income)}</td><td class="n">${tg(R.payouts)}</td><td class="n">${tg(R.expenses)}</td><td class="n">${tg(R.outgo)}</td><td class="n">${tg(R.net)}</td></tr>
    </tbody></table></div>
    <div class="adm-grid2 fin">
      <section class="adm-card"><header><h3>Зардал бүртгэх</h3></header>
        <form class="exp-form" id="expF">
          <label>Огноо<input type="date" name="date" required value="${isoDay(new Date())}" max="${isoDay(new Date())}"></label>
          <label>Ангилал<input name="category" list="expCats" required maxlength="40" placeholder="Сервер, дэд бүтэц"><datalist id="expCats">${d.categories.map((c) => `<option value="${esc(c)}">`).join("")}</datalist></label>
          <label>Дүн (₮)<input name="amount" type="number" min="1" step="1" required inputmode="numeric"></label>
          <label class="wide">Тайлбар<input name="note" maxlength="300" placeholder="Жишээ: 10-р сарын сервер, ClickHouse"></label>
          <button class="btn btn-gold btn-sm">Бүртгэх</button>
        </form>
        ${d.by_category.length ? `<h4 class="dr-h">Ангиллаар (сүүлийн ${state.months} сар)</h4>${bars(d.by_category.map((c) => ({ name: c.category, count: c.total })), tg)}` : ""}
      </section>
      <section class="adm-card"><header><h3>Бүртгэсэн зардал</h3><span class="muted small">${num(d.expenses.length)} · ${tg(R.expenses)}</span></header>
        ${d.expenses.length ? `<ul class="feed exp">${d.expenses.map((e) => `<li><span><b>${esc(e.category)}</b><small>${dd(e.date)}${e.note ? " · " + esc(e.note) : ""} · ${esc(e.created_by)}</small></span><b>${tg(e.amount)}</b><button type="button" class="adm-x sm" data-del-exp="${esc(e.id)}" title="Устгах" aria-label="Устгах">${ic("trash", 15)}</button></li>`).join("")}</ul>` : empty("Энэ хугацаанд зардал бүртгээгүй")}
      </section>
    </div>`;
    $("#expF").addEventListener("submit", async (e) => {
      e.preventDefault();
      const fm = e.target, b = $("button", fm); b.disabled = true;
      try { await api("/api/admin/expenses", { method: "POST", body: { date: fm.date.value, category: fm.category.value, amount: +fm.amount.value, note: fm.note.value } }); toast("Зардал бүртгэгдлээ ✓"); route(); }
      catch (x) { toast(x.message, true); b.disabled = false; }
    });
    el.onclick = (async (e) => {
      const b = e.target.closest("[data-del-exp]"); if (!b) return;
      if (!confirm("Энэ зардлыг устгах уу?")) return;
      try { await api("/api/admin/expenses/" + encodeURIComponent(b.dataset.delExp), { method: "DELETE" }); toast("Устгалаа"); route(); } catch (x) { toast(x.message, true); }
    });
  },
};

/* ---------- Татах хүсэлт ---------- */
const PST = { pending: ["Хүлээгдэж буй", "chip-amber"], paid: ["Шилжүүлсэн", "chip-teal"], rejected: ["Татгалзсан", "chip-coral"] };
const payCard = (p) => `<article class="ad-card" data-pid="${esc(p.id)}">
  <header><span class="chip ${PST[p.status]?.[1] || ""}">${PST[p.status]?.[0] || esc(p.status)}</span><small class="muted">${p.auto ? "Хуваариар" : "Гараар"}</small><time>${dt(p.created_at)}</time></header>
  <p class="ad-pay"><b>${tg(p.amount)}</b><span><a class="link" href="/t/${esc(p.teacher_username)}" target="_blank" rel="noopener">${esc(p.teacher_name || p.teacher_username)}</a> · ${esc(p.teacher_email || "")}</span></p>
  <dl class="ad-acct"><div><dt>Банк</dt><dd>${esc(p.bank_name)}</dd></div><div><dt>Данс</dt><dd><code>${esc(p.account_no)}</code> <button type="button" class="btn btn-ghost btn-sm" data-copy="${esc(p.account_no)}">Хуулах</button></dd></div><div><dt>Эзэмшигч</dt><dd>${esc(p.account_name)}</dd></div></dl>
  ${p.status === "pending" ? `<form class="ad-act ad-payact"><input name="note" maxlength="300" placeholder="Гүйлгээний дугаар эсвэл татгалзсан шалтгаан" aria-label="Тайлбар"><button class="btn btn-gold btn-sm" value="paid">✓ Шилжүүлсэн</button><button class="btn btn-danger btn-sm" value="rejected">Татгалзах</button></form>`
    : `<p class="ad-done">${esc(p.processed_by || "")}${p.processed_at ? " · " + dt(p.processed_at) : ""}${p.note ? ` — ${esc(p.note)}` : ""}</p>`}
</article>`;
VIEWS.payouts = {
  title: "Орлого татах хүсэлт", sub: "Багшийн данс руу банкаар шилжүүлээд «Шилжүүлсэн» гэж тэмдэглэнэ. Татгалзвал дүн багшийн үлдэгдэлд буцна. Аль ч тохиолдолд багшид мэдэгдэнэ.",
  st: "pending",
  async render(tools, el, live) {
    const self = VIEWS.payouts;
    tools.innerHTML = seg("pst", [["pending", "Хүлээгдэж буй"], ["done", "Шийдвэрлэсэн"]], self.st);
    tools.onclick = (e) => { const b = e.target.closest("[data-seg=pst] button"); if (b) { self.st = b.dataset.val; route(); } };
    const ps = await api("/api/admin/payouts?status=" + (self.st === "pending" ? "pending" : "all"));
    if (!live()) return;
    const shown = self.st === "pending" ? ps : ps.filter((p) => p.status !== "pending");
    const sum = shown.reduce((n, p) => n + (p.status === "rejected" ? 0 : p.amount), 0);
    el.innerHTML = `<div class="ad-list">${shown.length ? `<p class="muted small">${shown.length} хүсэлт · нийт ${tg(sum)}</p>` + shown.map(payCard).join("") : empty(self.st === "pending" ? "Хүлээгдэж буй хүсэлт алга 🎉" : "Шийдвэрлэсэн хүсэлт алга")}</div>`;
    el.onsubmit = async (e) => {
      const pf = e.target.closest(".ad-payact"); if (!pf) return;
      e.preventDefault();
      const status = e.submitter?.value || "paid", card = pf.closest(".ad-card");
      if (status === "rejected" && !pf.note.value.trim()) { toast("Татгалзсан шалтгаанаа бичнэ үү", true); pf.note.focus(); return; }
      if (status === "paid" && !confirm("Энэ дүнг багшийн данс руу шилжүүлсэн үү?")) return;
      $$("button", pf).forEach((b) => (b.disabled = true));
      try { await api(`/api/admin/payouts/${encodeURIComponent(card.dataset.pid)}`, { method: "POST", body: { status, note: pf.note.value } }); toast(status === "paid" ? "Шилжүүлсэн гэж тэмдэглэлээ ✓" : "Татгалзлаа"); badges(); route(); }
      catch (x) { toast(x.message, true); $$("button", pf).forEach((b) => (b.disabled = false)); }
    };
  },
};

/* ---------- Мэдүүлэг ---------- */
const KIND = { group: "Сургалтын бүлэг чат", team: "Сурагчдын бүлэг", dm: "Ангийн найзын чат", "": "Багштай чат" };
const repCard = (r) => `<article class="ad-card" data-id="${esc(r.id)}">
  <header><span class="chip ${r.status === "open" ? "chip-amber" : ""}">${esc(r.reason_label || r.reason)}</span><small class="muted">${esc(KIND[r.conv_kind] ?? "Чат")}</small><time>${dt(r.created_at)}</time></header>
  <p class="ad-who"><b>${esc(r.reporter_name || "Зочин")}</b> мэдүүлэв <span aria-hidden="true">→</span> <b>${esc(r.target_name || "Зочин")}</b></p>
  ${r.note ? `<p class="ad-note">«${esc(r.note)}»</p>` : ""}
  <ol class="ad-ctx">${(r.context || []).map((m) => `<li class="${m.reported ? "hit" : ""}"><b>${esc(m.name || "?")}</b><span>${esc(m.body)}</span><time>${dt(m.at).slice(11)}</time></li>`).join("")}</ol>
  ${r.status === "open" ? `<form class="ad-act ad-repact"><input name="resolution" maxlength="500" placeholder="Шийдвэр (ж: сануулсан, мессежийг устгасан)" aria-label="Шийдвэр"><label class="check"><input type="checkbox" name="del">Мессежийг устгах</label><button class="btn btn-gold btn-sm">Шийдвэрлэх</button></form>`
    : `<p class="ad-done">✓ ${esc(r.resolved_by || "")}${r.resolved_at ? " · " + dt(r.resolved_at) : ""}${r.resolution ? ` — ${esc(r.resolution)}` : ""}</p>`}
</article>`;
VIEWS.reports = {
  title: "Чатын мэдүүлэг", sub: "Мэдүүлсэн мессеж ба түүний өмнөх хэдэн мессежийг л харуулна. Шийдвэрлэхэд бүртгэлтэй мэдүүлэгчид мэдэгдэнэ.",
  st: "open",
  async render(tools, el, live) {
    const self = VIEWS.reports;
    tools.innerHTML = seg("rst", [["open", "Нээлттэй"], ["resolved", "Шийдвэрлэсэн"]], self.st);
    tools.onclick = (e) => { const b = e.target.closest("[data-seg=rst] button"); if (b) { self.st = b.dataset.val; route(); } };
    const rs = await api("/api/admin/reports?status=" + self.st);
    if (!live()) return;
    el.innerHTML = `<div class="ad-list">${rs.length ? rs.map(repCard).join("") : empty(self.st === "open" ? "Нээлттэй мэдүүлэг алга 🎉" : "Шийдвэрлэсэн мэдүүлэг алга")}</div>`;
    el.onsubmit = async (e) => {
      const f = e.target.closest(".ad-repact"); if (!f) return;
      e.preventDefault();
      const b = $("button", f); b.disabled = true;
      try { await api(`/api/admin/reports/${encodeURIComponent(f.closest(".ad-card").dataset.id)}/resolve`, { method: "POST", body: { resolution: f.resolution.value, delete_message: f.del.checked } }); toast("Шийдвэрлэлээ ✓"); badges(); route(); }
      catch (x) { toast(x.message, true); b.disabled = false; }
    };
  },
};

/* ---------- Систем ---------- */
const uptime = (sec) => { sec = +sec || 0; const d = Math.floor(sec / 86400), h = Math.floor((sec % 86400) / 3600), m = Math.floor((sec % 3600) / 60); return d ? `${d} өдөр ${h} цаг` : h ? `${h} цаг ${m} мин` : `${m} мин`; };
const useBar = (used, total) => total ? `<span class="use" title="${pct(used, total)}% ашиглагдсан"><i style="width:${Math.min(100, pct(used, total))}%"></i></span>` : "";
const FOLDER = { teachers: "Багш нарын файл (хичээл, ном, видео)", "(үндсэн)": "Үндсэн хавтас" };
VIEWS.system = {
  title: "Систем", sub: "Файлын сан, өгөгдлийн сангийн хэмжээ, дискний сул зай, серверийн төлөв",
  fresh: false,
  async render(tools, el, live) {
    const self = VIEWS.system;
    tools.innerHTML = `<button type="button" class="btn btn-ghost btn-sm" data-fresh>${ic("refresh", 16)} Шинээр тооцох</button>`;
    tools.onclick = (e) => { if (e.target.closest("[data-fresh]")) { self.fresh = true; route(); } };
    const d = await api("/api/admin/system" + (self.fresh ? "?fresh=1" : "")); self.fresh = false;
    if (!live()) return;
    const F = d.files, D = d.db, S = d.server;
    const fUsed = F ? F.disk_total - F.disk_free : 0, low = F && F.disk_total && F.disk_free / F.disk_total < 0.1;
    const dbDisk = (D?.disks || [])[0];
    el.innerHTML = `
    <h2 class="adm-h">Файлын сан</h2>
    ${F ? `<section class="kp-grid">
      ${kpi("Файлын сангийн хэмжээ", bytes(F.bytes), `${num(F.files)} файл`)}
      ${kpi("Дискний сул зай", bytes(F.disk_free), `нийт ${bytes(F.disk_total)} · ашиглагдсан ${pct(fUsed, F.disk_total)}% ${useBar(fUsed, F.disk_total)}`, low ? "warn" : "")}
      ${kpi("Файлын сангийн эзлэх хувь", pct(F.bytes, F.disk_total) + "%", "дискний нийт багтаамжаас")}
    </section>
    <section class="adm-card"><header><h3>Хавтсаар</h3><span class="muted small">${esc(F.root)} · ${dt(d.files_at)}-д тооцсон</span></header>
      ${bars((F.folders || []).map((f) => ({ name: `${FOLDER[f.name] || f.name} · ${num(f.files)} файл`, count: f.bytes })), bytes)}
      ${d.files_err ? `<p class="muted small">⚠️ ${esc(d.files_err)}</p>` : ""}</section>` : empty("Файлын сан тохируулаагүй")}

    <h2 class="adm-h">Өгөгдлийн сан</h2>
    ${D ? `<section class="kp-grid">
      ${kpi("Өгөгдлийн сангийн хэмжээ", bytes(D.bytes), D.raw ? `шахаагүй ${bytes(D.raw)} · ${(D.raw / D.bytes).toFixed(1)}× шахалт` : "дискэн дээрх (шахсан) хэмжээ")}
      ${kpi("Нийт мөр", num(D.rows), `${num((D.tables || []).length)} хүснэгт`)}
      ${dbDisk ? kpi("Сангийн серверийн сул зай", bytes(dbDisk.free), `нийт ${bytes(dbDisk.total)} · ашиглагдсан ${pct(dbDisk.total - dbDisk.free, dbDisk.total)}% ${useBar(dbDisk.total - dbDisk.free, dbDisk.total)}`, dbDisk.total && dbDisk.free / dbDisk.total < 0.1 ? "warn" : "") : ""}
      ${kpi("Хөдөлгүүр", esc(D.engine || "—"), [D.version, D.database].filter(Boolean).map(esc).join(" · "))}
    </section>
    ${(D.tables || []).length ? `<div class="adm-table-wrap"><table class="adm-table" id="dbT"><thead><tr><th data-sort="name" data-type="text">Хүснэгт</th><th data-sort="rows">Мөр</th><th data-sort="bytes">Дискэн дээр</th><th data-sort="raw">Шахаагүй</th><th data-sort="ratio">Шахалт</th></tr></thead><tbody></tbody></table></div>` : ""}` : ""}
    ${d.db_err ? `<p class="adm-empty">⚠️ Өгөгдлийн сангийн хэмжээг уншиж чадсангүй: ${esc(d.db_err)}</p>` : ""}

    <h2 class="adm-h">Сервер</h2>
    <section class="kp-grid small">
      ${kpi("Ажилласан хугацаа", uptime(S.uptime_sec), `${dt(S.started_at)}-с хойш`)}
      ${kpi("Санах ой", bytes(S.mem_alloc), `процесс ${bytes(S.mem_sys)}`)}
      ${kpi("Шууд холболт", num(S.ws), "WebSocket (чат, мэдэгдэл)")}
      ${kpi("Хувилбар", esc(S.version || "—"), `${esc(S.go || "")} · ${num(S.goroutines)} goroutine`)}
    </section>`;
    if ($("#dbT")) sortable($("#dbT"), D.tables.map((t) => ({ ...t, ratio: t.bytes ? t.raw / t.bytes : 0 })), (t) => `<tr><td><code>${esc(t.name)}</code></td><td class="n">${num(t.rows)}</td><td class="n"><b>${bytes(t.bytes)}</b></td><td class="n">${t.raw ? bytes(t.raw) : "—"}</td><td class="n">${t.ratio ? t.ratio.toFixed(1) + "×" : "—"}</td></tr>`, ["bytes", -1]);
  },
};

/* ---------- дэлгэрэнгүй цонх ---------- */
let lastFocus = null;
function drawer(title, html) {
  const d = $("#admDrawer");
  if (d.hidden) lastFocus = document.activeElement;
  $("#admDrawerTitle").textContent = title;
  const body = $("#admDrawerBody"); body.innerHTML = html; body.scrollTop = 0;
  d.hidden = false; document.documentElement.classList.add("adm-noscroll");
  requestAnimationFrame(() => d.classList.add("on"));
  $(".adm-x", d).focus();
  return body;
}
function closeDrawer() {
  const d = $("#admDrawer"); if (d.hidden) return;
  d.classList.remove("on"); document.documentElement.classList.remove("adm-noscroll");
  setTimeout(() => { d.hidden = true; }, 220);
  lastFocus?.focus?.();
}

/* ---------- эхлэл ---------- */
function init() {
  $$(".adm-nav .ai").forEach((s) => (s.outerHTML = ic(s.dataset.i, 18)));
  window.addEventListener("hashchange", () => { if (!$("#admShell").hidden) { closeDrawer(); route(); } });
  $("#admLoginForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const f = e.target, b = $("button", f), err = $("#admLoginErr");
    b.disabled = true; err.hidden = true;
    try { state.me = await api("/api/admin/login", { method: "POST", body: { login: f.login.value, password: f.password.value } }); f.password.value = ""; start(); }
    catch (x) { err.textContent = x.message; err.hidden = false; f.password.select(); }
    b.disabled = false;
  });
  $("#admOut").addEventListener("click", async () => {
    try { await api("/api/admin/logout", { method: "POST" }); } catch {}
    if (state.me?.via === "oauth") { location.href = "/"; return; }
    state.me = null; showLogin(false);
  });
  // Мөр / нэр дээр дарахад дэлгэрэнгүй
  document.addEventListener("click", (e) => {
    if (e.target.closest("[data-close]")) { closeDrawer(); return; }
    if (e.target.closest("a, button, input, select, label")) return;
    const t = e.target.closest("[data-teacher]"), s = e.target.closest("[data-student]");
    if (t) openTeacher(t.dataset.teacher); else if (s) openStudent(s.dataset.student);
  });
  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape") closeDrawer();
    if (e.key === "Enter" && e.target.matches?.("tr[data-teacher], tr[data-student]")) e.target.click();
  });
  document.addEventListener("click", (e) => {
    const c = e.target.closest("[data-copy]"); if (!c) return;
    navigator.clipboard.writeText(c.dataset.copy).then(() => toast("Дансны дугаар хуулагдлаа ✓"), () => toast(c.dataset.copy));
  });
  // Графикийн тайлбар
  const tip = document.createElement("div"); tip.className = "ch-tip"; tip.hidden = true; document.body.append(tip);
  document.addEventListener("mouseover", (e) => {
    const c = e.target.closest?.(".ch-col");
    if (!c) { tip.hidden = true; return; }
    tip.textContent = c.dataset.tip; tip.hidden = false;
    const r = c.getBoundingClientRect(), w = tip.offsetWidth;
    tip.style.left = Math.min(innerWidth - w - 8, Math.max(8, r.left + r.width / 2 - w / 2)) + "px";
    tip.style.top = Math.max(8, r.top - tip.offsetHeight - 8) + "px";
  });
  api("/api/admin/me").then((me) => {
    if (me.ok) { state.me = me; start(); return; }
    showLogin(false);
    if (!me.password) {
      const err = $("#admLoginErr");
      err.textContent = "Нууц үгээр нэвтрэх тохируулаагүй байна (ADMIN_LOGIN, ADMIN_PASSWORD_HASH). ADMIN_EMAILS-д бүртгэлтэй Google бүртгэлээр нэвтэрч болно.";
      err.hidden = false; $$("#admLoginForm input, #admLoginForm button").forEach((x) => (x.disabled = true));
    }
  }).catch(() => showLogin(false));
}
if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init); else init();
})();
