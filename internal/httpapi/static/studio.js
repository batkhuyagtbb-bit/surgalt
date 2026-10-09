/* surgalt.mn — багшийн удирдлагын хэсгүүд (тойм, сургалт, файл, шууд хичээл, тохиргоо, чат).
   Багшийн өөрийн нээлттэй профайл дээр ачаалагдаж, window.Studio.mount-аар таб дотор рендерлэнэ. */
(async () => {
"use strict";
const isStudio = false; // тусдаа студи хуудас байхгүй; бүх зүйл профайл дээр
const { $, $$, esc, api, Auth, toast, money, fmtDate, fmtTime, Live, mediaHTML, book3dHTML, hydrateBooks, msgHTML, celebrate, ringHTML, hueOfName, fmtDay, WEEKDAYS, WEEKDAYS_SHORT } = window.SG;
if (!Auth.token) return;
let me;
try { me = await api("/api/me"); Auth.set(Auth.token, me); } catch { return; }
let main = null;
const teacher = me.role === "teacher";
const fmtSize = (b) => b >= 1 << 30 ? (b / (1 << 30)).toFixed(2) + " GB" : b >= 1 << 20 ? (b / (1 << 20)).toFixed(1) + " MB" : Math.max(1, Math.round(b / 1024)) + " KB";
const initialsOf = (name) => (name || "?").trim().split(/\s+/).map((w) => w[0]).join("").slice(0, 2).toUpperCase();
const avatar = (u, cls = "avatar-md") => `<div class="avatar ${cls}" style="--h:${hueOfName(u.username)}">${u.avatar_url ? `<img src="${esc(u.avatar_url)}" alt="">` : esc(initialsOf(u.display_name))}</div>`;
const countUp = (el, end, fmt = (n) => n.toLocaleString("en-US")) => {
  if (!el) return;
  const t0 = performance.now();
  const f = (t) => { const p = Math.min(1, (t - t0) / 900); el.textContent = fmt(Math.round(end * (1 - Math.pow(1 - p, 4)))); if (p < 1) requestAnimationFrame(f); };
  requestAnimationFrame(f);
};
const panel = (html, d = 0, cls = "") => `<section class="panel ${cls}" style="--d:${d}">${html}</section>`;

/* Нэг хэв маягийн шугаман дүрсүүд (emoji биш — бүх дэлгэцэд ижил, хурц харагдана). */
const ICON = {
  home: '<path d="M3 11.5 12 4l9 7.5"/><path d="M5 10v9a1 1 0 0 0 1 1h4v-6h4v6h4a1 1 0 0 0 1-1v-9"/>',
  courses: '<path d="m2 9 10-5 10 5-10 5z"/><path d="M6 11.5V16c0 1.2 2.7 3 6 3s6-1.8 6-3v-4.5"/>',
  files: '<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>',
  live: '<rect x="3" y="6" width="12" height="12" rx="2"/><path d="m15 10 6-3v10l-6-3z"/>',
  chat: '<path d="M21 12a8 8 0 0 1-11.6 7.1L4 20l1-4.6A8 8 0 1 1 21 12Z"/>',
  profile: '<circle cx="12" cy="8" r="4"/><path d="M4 21c0-4 3.6-6 8-6s8 2 8 6"/>',
  ext: '<path d="M14 4h6v6"/><path d="m20 4-9 9"/><path d="M19 14v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1h5"/>',
  out: '<path d="M9 4H5a1 1 0 0 0-1 1v14a1 1 0 0 0 1 1h4"/><path d="m16 8 4 4-4 4"/><path d="M20 12H9"/>',
  money: '<rect x="3" y="6" width="18" height="12" rx="2"/><circle cx="12" cy="12" r="2.5"/><path d="M7 9v.01M17 15v.01"/>',
  users: '<circle cx="9" cy="8" r="3.5"/><path d="M2.5 20c0-3.5 3-5.5 6.5-5.5s6.5 2 6.5 5.5"/><path d="M16 4.6a3.5 3.5 0 0 1 0 6.8M18 14.7c2 .7 3.5 2.4 3.5 5.3"/>',
  eye: '<path d="M2 12s3.6-7 10-7 10 7 10 7-3.6 7-10 7S2 12 2 12Z"/><circle cx="12" cy="12" r="3"/>',
  clock: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
  cal: '<rect x="3" y="5" width="18" height="16" rx="2"/><path d="M8 3v4M16 3v4M3 10h18"/>',
  book: '<path d="M4 5a2 2 0 0 1 2-2h14v16H6a2 2 0 0 0-2 2z"/><path d="M4 21V5M9 7h7"/>',
  search: '<circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/>',
  plus: '<path d="M12 5v14M5 12h14"/>',
  chevron: '<path d="m9 6 6 6-6 6"/>',
  globe: '<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c2.500 2.500 3.800 5.500 3.800 9s-1.300 6.500-3.800 9c-2.500-2.500-3.800-5.500-3.800-9S9.500 5.500 12 3Z"/>',
  lock: '<rect x="5" y="11" width="14" height="9" rx="2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/>',
  edit: '<path d="M4 20h4L19 9l-4-4L4 16z"/><path d="m13.500 6.500 4 4"/>',
  clip: '<path d="m20 11-8.500 8.500a5 5 0 0 1-7-7L13 4a3.300 3.300 0 0 1 4.700 4.700l-8.400 8.400a1.700 1.700 0 0 1-2.400-2.400L14.500 7"/>',
  link: '<path d="M10 14a4 4 0 0 0 5.700 0l3-3a4 4 0 0 0-5.700-5.700l-1 1"/><path d="M14 10a4 4 0 0 0-5.700 0l-3 3a4 4 0 0 0 5.700 5.700l1-1"/>',
  gear: '<circle cx="12" cy="12" r="3"/><path d="M12 3v2.500M12 18.500V21M3 12h2.500M18.500 12H21M5.600 5.600l1.800 1.800M16.600 16.600l1.800 1.800M5.600 18.400l1.800-1.800M16.600 7.400l1.800-1.800"/>',
  back: '<path d="M15 6l-6 6 6 6"/>',
  x: '<path d="M6 6l12 12M18 6 6 18"/>',
  grip: '<circle cx="9" cy="6" r="1.2"/><circle cx="15" cy="6" r="1.2"/><circle cx="9" cy="12" r="1.2"/><circle cx="15" cy="12" r="1.2"/><circle cx="9" cy="18" r="1.2"/><circle cx="15" cy="18" r="1.2"/>',
  list: '<path d="M9 6h11M9 12h11M9 18h11"/><path d="M4 6h.01M4 12h.01M4 18h.01"/>',
  image: '<rect x="3" y="4" width="18" height="16" rx="2"/><circle cx="9" cy="10" r="2"/><path d="m21 16-5-5-9 9"/>',
  mic: '<rect x="9" y="3" width="6" height="11" rx="3"/><path d="M5 11a7 7 0 0 0 14 0M12 18v3"/>',
  quiz: '<circle cx="12" cy="12" r="9"/><path d="M9.5 9a2.5 2.5 0 1 1 3.500 2.300c-.900.400-1 1-1 1.700M12 17h.01"/>',
  heading: '<path d="M6 4v16M18 4v16M6 12h12"/>',
  text: '<path d="M4 6h16M4 12h16M4 18h10"/>',
  up: '<path d="m6 15 6-6 6 6"/>',
  down: '<path d="m6 9 6 6 6-6"/>',
};
Object.assign(ICON, {
  chart: '<path d="M3 3v18h18"/><path d="M7 16v-4M12 16V8M17 16v-7"/>',
  target: '<circle cx="12" cy="12" r="9"/><circle cx="12" cy="12" r="5"/><circle cx="12" cy="12" r="1.2"/>',
  pulse: '<path d="M3 12h4l3-8 4 16 3-8h4"/>',
  alert: '<path d="M12 3l9.5 17h-19z"/><path d="M12 10v4M12 17.5h.01"/>',
});
const ico = (n, size = 20) => `<svg viewBox="0 0 24 24" width="${size}" height="${size}" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${ICON[n] || ""}</svg>`;

Live.connect(Auth.token);

const logout = () => { Auth.clear(); location.href = "/"; };
const setDrawer = () => {};
let searchCourses = null;
let cleanup = null;

const views = { overview, courses, books, course: (id) => courseEditor(id), files, live, chat: (id) => chat(id), learning, students };

/* Профайл хуудаснаас дуудагдана: тухайн хэсгийг өгсөн контейнерт рендерлэнэ. */
window.Studio = {
  me: () => me,
  teacher,
  async mount(container, view, arg) {
    cleanup?.(); cleanup = null;
    main = container;
    main.innerHTML = `<div class="loader"></div>`;
    try { await (views[view] || views.overview)(arg); }
    catch (e) { main.innerHTML = panel(`<p class="form-error">${esc(e.message)}</p>`); }
  },
};

/* ---------- Нүүр (тойм) ---------- */
async function overview() {
  const [sales, courses, storage, meetings, home] = await Promise.all([
    api("/api/me/sales?limit=8"), api("/api/me/courses"), api("/api/me/storage"), api("/api/me/meetings").catch(() => []), api("/api/me/home")]);
  me = home.user; Auth.set(Auth.token, me);
  const tp = home.teacher;
  const views = courses.reduce((n, c) => n + (c.views || 0), 0);
  const titleOf = Object.fromEntries(courses.map((c) => [c.id, c.title]));
  const now = new Date(), hr = now.getHours();
  const hello = hr < 5 ? "Шөнийн мэнд" : hr < 12 ? "Өглөөний мэнд" : hr < 18 ? "Өдрийн мэнд" : "Оройн мэнд";
  const dayKey = (d) => new Date(d).toDateString();
  const until = (t) => {
    const m = Math.round((new Date(t) - Date.now()) / 60000);
    if (m <= 0) return "Одоо явагдаж байна";
    if (m < 60) return m + " мин дараа";
    if (m < 1440) return Math.floor(m / 60) + " ц " + (m % 60) + " мин дараа";
    return Math.floor(m / 1440) + " хоногийн дараа";
  };
  const days = Array.from({ length: 7 }, (_, i) => { const d = new Date(now); d.setHours(0, 0, 0, 0); d.setDate(d.getDate() + i); return d; });
  const perDay = (d) => meetings.filter((m) => dayKey(m.starts_at) === dayKey(d));
  const next = meetings[0];
  const pct = storage.quota ? Math.round(storage.used / storage.quota * 100) : 0;
  const kpi = (ic, id, label, sub, href) => `<a class="anx-kpi ovx-kpi" href="${href}"><span class="anx-ic">${ico(ic, 18)}</span><small>${label}</small><b id="${id}">0</b><div class="anx-sub">${sub}</div></a>`;
  // Ухаалаг зөвлөмж: одоо байгаа өгөгдлөөс хамгийн чухал 4 алхам (ноорог, хоосон сургалт, ойрын шууд хичээл, сан, борлуулалт).
  const soon = meetings.find((m) => new Date(m.starts_at) - now < 24 * 3600e3 && new Date(m.starts_at).getTime() + m.duration_min * 60000 > now);
  const drafts = courses.filter((c) => !c.published), empty = courses.filter((c) => !c.lesson_count);
  const todo = [
    soon && { ic: "live", tone: "amber", t: `${until(soon.starts_at)} шууд хичээл`, s: `«${soon.title}» · ${fmtTime(soon.starts_at)}`, href: "#live" },
    drafts.length && { ic: "lock", tone: "", t: `${drafts.length} сургалт ноорог хэвээр`, s: "Нийтэлбэл профайл дээр харагдаж, худалдаалагдана", href: drafts.length === 1 ? `#course=${drafts[0].id}` : "#courses" },
    empty[0] && { ic: "plus", tone: "", t: `«${empty[0].title}» сургалтад хичээл алга`, s: "Эхний хичээлээ нийтлээрэй", href: `#course=${empty[0].id}` },
    pct >= 80 && { ic: "files", tone: "bad", t: `Файлын сан ${pct}% дүүрсэн`, s: "Илүүдэл файлаа цэвэрлэх эсвэл багтаамж нэмэх", href: "#files" },
    !sales.count && courses.length && { ic: "money", tone: "", t: "Анхны борлуулалтаа хүлээж байна", s: "Үнэгүй хичээл нэмж, профайлынхаа холбоосыг түгээгээрэй", href: "#courses" },
    tp.students && { ic: "chart", tone: "", t: `${tp.students} суралцагчийн идэвх`, s: "Анхаарал, зөрчил, суралцсан оноог хянах", href: "#students" },
  ].filter(Boolean).slice(0, 4);

  main.innerHTML = `<div class="ovx">
    <section class="ovx-hero">
      <div class="ovx-hello"><span>${hello},</span><h1>${esc(me.display_name)}</h1>
        <p>${ico("cal", 15)}${fmtDay(now, true)} · ${WEEKDAYS[now.getDay()]} гараг<span class="ovx-dot"></span>${ico("courses", 15)}${tp.published}/${tp.courses} сургалт нийтлэгдсэн</p></div>
      <div class="ovx-clock" aria-label="Цаг"><b id="clock">--:--</b></div>
      <div class="ovx-quick">
        <a class="ovx-q" href="#courses" data-quick="new-course">${ico("plus", 17)}Шинэ сургалт</a>
        <a class="ovx-q" href="#live">${ico("live", 17)}Шууд хичээл товлох</a>
        <a class="ovx-q" href="#students">${ico("chart", 17)}Хяналт ба статистик</a>
        <button type="button" class="ovx-q" id="copyLink">${ico("link", 17)}Профайлын холбоос хуулах</button>
      </div>
    </section>

    <div class="anx-kpis">
      ${kpi("money", "kSum", "Нийт орлого", `${sales.count} борлуулалт`, "#ov-sales")}
      ${kpi("users", "kStu", "Суралцагч", "элссэн, хичээл авсан", "#students")}
      ${kpi("eye", "kPv", "Профайл үзэлт", `<span id="kPvSub">сүүлийн 30 хоног · хэн, хэр удаан ↓</span>`, "#ov-visits")}
      ${kpi("book", "kCv", "Сургалт үзэлт", `${courses.length} сургалт`, "#courses")}
    </div>

    <div class="ovx-grid">
      <div class="ovx-col">
      <section class="anx-card ovx-courses"><div class="anx-card-head"><h3>${ico("courses", 18)}Миний сургалтууд <span class="chip">${courses.length}</span></h3><a class="link" href="#courses">Бүгдийг удирдах ${ico("chevron", 14)}</a></div>
        ${courses.length ? `<div class="ovx-clist">${courses.slice(0, 7).map((c) => `<a class="ovx-course" href="#course=${esc(c.id)}">
          <span class="ovx-cv" aria-hidden="true">${esc(c.title.trim()[0] || "?")}</span>
          <span class="ovx-cmain"><strong>${esc(c.title)}</strong><small>${c.lesson_count} хичээл · ${c.views || 0} үзэлт · ${c.price ? money(c.price) : "хичээлээр / үнэгүй"}</small></span>
          <span class="chip ${c.published ? "chip-teal" : ""}">${c.published ? "Нийтлэгдсэн" : "Ноорог"}</span><span class="ovx-go" aria-hidden="true">${ico("chevron", 16)}</span></a>`).join("")}</div>
          ${courses.length > 7 ? `<a class="link ovx-more" href="#courses">Бусад ${courses.length - 7} сургалт ${ico("chevron", 14)}</a>` : ""}`
        : `<div class="anx-empty">${ico("courses", 28)}<p>Анхны сургалтаа үүсгээд хичээлээ нийтэлж эхлээрэй.</p><a class="btn btn-gold btn-sm" href="#courses" data-quick="new-course">${ico("plus", 16)}Шинэ сургалт</a></div>`}
      </section>
        <section class="anx-card" id="ov-sales"><div class="anx-card-head"><h3>${ico("money", 18)}Сүүлийн борлуулалт</h3>${sales.count ? `<span class="muted small">нийт ${sales.count} борлуулалт</span>` : ""}</div>
      ${sales.recent.length ? `<div class="an-table-wrap"><table class="tbl"><thead><tr><th>Сургалт / хичээл</th><th>Худалдан авагч</th><th>Дүн</th><th>Огноо</th></tr></thead><tbody>
      ${sales.recent.map((x) => `<tr><td>${esc(x.course_title)}</td><td>@${esc(x.buyer_username)}</td><td><strong>${money(x.amount)}</strong></td><td class="muted">${fmtDate(x.paid_at)}</td></tr>`).join("")}</tbody></table></div>`
        : `<div class="anx-empty">${ico("money", 28)}<p>Анхны борлуулалтаа хүлээж байна ✨</p></div>`}</section>
      </div>
      <div class="ovx-col">
        <section class="anx-card"><div class="anx-card-head"><h3>${ico("target", 18)}Анхаарах зүйлс</h3></div>
          ${todo.length ? `<ul class="ovx-todo">${todo.map((x) => `<li><a href="${esc(x.href)}" class="${x.tone}"><span class="ovx-ti">${ico(x.ic, 17)}</span><span><strong>${esc(x.t)}</strong><small>${esc(x.s)}</small></span>${ico("chevron", 15)}</a></li>`).join("")}</ul>`
            : `<p class="muted small" style="margin:0">Бүх зүйл хэвийн байна ✓</p>`}</section>
        <section class="anx-card"><div class="anx-card-head"><h3>${ico("cal", 18)}Шууд хичээлийн хуваарь</h3><a class="link" href="#live">Товлох ${ico("chevron", 14)}</a></div>
          ${next ? `<div class="ovx-next"><small>Дараагийнх · ${fmtDate(next.starts_at)}</small><strong>${esc(next.title)}</strong><span class="muted small">${next.duration_min} мин${titleOf[next.course_id] ? " · " + esc(titleOf[next.course_id]) : ""}</span>
            <div class="ovx-next-acts"><span class="chip chip-amber">${until(next.starts_at)}</span><a class="btn btn-gold btn-sm" href="${esc(next.meet_url)}" target="_blank" rel="noopener">${ico("live", 15)}Live эхлүүлэх</a></div></div>` : ""}
          <p class="ovx-days-h">Ойрын 7 хоног · өдөр дээр дарж хуваарийг харна</p>
          <div class="days ovx-days" id="days">${days.map((d, i) => { const n = perDay(d).length; return `<button class="day" data-i="${i}"><small>${WEEKDAYS_SHORT[d.getDay()]}</small><b>${String(d.getDate()).padStart(2, "0")}</b>${n ? `<i>${n}</i>` : ""}</button>`; }).join("")}</div>
          <div class="agenda" id="agenda"></div></section>
        <section class="anx-card"><div class="anx-card-head"><h3>${ico("files", 18)}Файлын сан</h3><a class="link" href="#files">Удирдах ${ico("chevron", 14)}</a></div>
          <div class="anx-meter ovx-store"><i id="sMeter" style="width:0;background:${pct >= 80 ? "var(--coral)" : "var(--brand)"}"></i></div>
          <p class="muted small" style="margin:.6em 0 0">${fmtSize(storage.used)} / ${fmtSize(storage.quota)} ашигласан · ${pct}%</p></section>
      </div>
    </div>

    <section class="anx-card" id="ov-visits"><div class="anx-card-head"><h3>${ico("eye", 18)}Профайлын үзэлт ба зочид</h3>
      <div class="anx-tools"><div class="anx-seg" role="group" aria-label="Хуудас">${[["profile", "Профайл"], ["course", "Сургалтууд"], ["all", "Бүгд"]].map(([k, t]) => `<button type="button" data-vk="${k}" aria-pressed="${k === "profile"}">${t}</button>`).join("")}</div>
        <div class="anx-seg" role="group" aria-label="Хугацаа">${[7, 30, 90].map((d) => `<button type="button" data-vd="${d}" aria-pressed="${d === 30}">${d} хоног</button>`).join("")}</div></div></div>
      <div id="vsBody"><div class="pw-load"><span class="loader"></span></div></div></section>

  </div>`;

  // Хавтанд багтахын тулд том дүнг товчилно (1,250,000₮ -> 1.25 сая₮).
  const short = (n) => (n >= 1e9 ? +(n / 1e9).toFixed(2) + " тэрбум₮" : n >= 1e6 ? +(n / 1e6).toFixed(2) + " сая₮" : n ? money(n) : "0₮");
  countUp($("#kSum"), sales.total_amount, short);
  countUp($("#kStu"), tp.students); countUp($("#kCv"), views); // профайл үзэлтийг доорх үзэлтийн статистикаас (сүүлийн 30 хоног)
  requestAnimationFrame(() => { const m = $("#sMeter"); if (m) m.style.width = Math.min(100, pct) + "%"; });
  $("#copyLink").onclick = () => navigator.clipboard.writeText(`${location.origin}/t/${me.username}`).then(() => toast("Профайлын холбоос хуулагдлаа ✓"), () => toast(`${location.origin}/t/${me.username}`));

  // Профайлын үзэлт: хэдэн хүн, хэдэн удаа, хэр удаан, хаанаас, ямар төхөөрөмжөөр.
  const vst = { kind: "profile", days: 30 };
  const secs = (n) => (n < 60 ? `${n} сек` : n < 3600 ? `${Math.floor(n / 60)} мин${n % 60 ? ` ${n % 60} сек` : ""}` : `${Math.floor(n / 3600)} ц ${Math.round(n % 3600 / 60)} мин`);
  const DEV = { mobile: "Утас", desktop: "Компьютер", tablet: "Таблет" };
  const barList = (rows, label = (x) => esc(x.name)) => { const mx = Math.max(1, ...rows.map((x) => x.count)); return rows.length ? `<ul class="vs-bars">${rows.map((x) => `<li><span>${label(x)}</span><i style="width:${x.count / mx * 100}%"></i><b>${x.count}</b></li>`).join("")}</ul>` : `<p class="muted small" style="margin:0">Мэдээлэл алга</p>`; };
  const visChart = (daily) => {
    const mx = Math.max(0, ...daily.map((d) => d.views));
    if (!mx) return `<div class="anx-empty">${ico("eye", 28)}<p>Энэ хугацаанд үзэлт бүртгэгдээгүй байна. Профайлынхаа холбоосыг түгээгээрэй.</p></div>`;
    const top = [2, 4, 6, 10, 20, 40, 60, 100, 200, 500, 1000, 2000, 5000, 10000].find((x) => x >= mx) || mx, lbl = (day) => day.slice(5).replace("-", "/");
    return `<div class="anx-chart vs-chart"><div class="anx-y" aria-hidden="true"><span>${top}</span><span>${Math.round(top / 2)}</span><span>0</span></div>
      <div class="anx-plot"><div class="anx-cols">${daily.map((d) => `<div class="anx-col" title="${esc(d.day)}: ${d.views} үзэлт · ${d.unique} зочин${d.seconds ? " · " + secs(d.seconds) : ""}"><i class="ac" style="height:${d.views / top * 100}%"></i></div>`).join("")}</div></div>
      <div class="anx-x" aria-hidden="true"><span>${lbl(daily[0].day)}</span><span>${lbl(daily[Math.floor((daily.length - 1) / 2)].day)}</span><span>${lbl(daily[daily.length - 1].day)}</span></div></div>`;
  };
  const loadVisits = async () => {
    const box = $("#vsBody"); if (!box) return;
    let d; try { d = await api(`/api/me/visits?days=${vst.days}&kind=${vst.kind}`); } catch (e) { box.innerHTML = `<p class="form-error">${esc(e.message)}</p>`; return; }
    const t = d.totals;
    if (vst.kind === "profile" && vst.days === 30 && !box.dataset.kpi) { box.dataset.kpi = "1"; countUp($("#kPv"), t.views); $("#kPvSub").textContent = `сүүлийн 30 хоног · ${t.unique} зочин ↓`; }
    box.innerHTML = `<div class="anx-stats vs-stats">
        <div class="anx-stat"><small>Нийт үзэлт</small><b>${t.views}</b></div>
        <div class="anx-stat"><small>Давтагдаагүй зочин</small><b>${t.unique}</b></div>
        <div class="anx-stat"><small>Бүртгэлтэй хэрэглэгч</small><b>${t.members}</b></div>
        <div class="anx-stat"><small>Дундаж хугацаа</small><b>${t.measured ? secs(t.avg_sec) : "—"}</b></div>
        <div class="anx-stat"><small>Нийт хугацаа</small><b>${t.total_sec ? secs(t.total_sec) : "—"}</b></div>
        <div class="anx-stat ${t.bounce_pct >= 60 ? "bad" : ""}"><small>10 секундэд гарсан</small><b>${t.measured ? t.bounce_pct + "%" : "—"}</b></div></div>
      <div class="vs-grid">
        <div class="vs-main"><h4 class="anx-sub-h">Өдөр бүрийн үзэлт</h4>${visChart(d.daily)}
          ${d.courses.length && vst.kind !== "profile" ? `<h4 class="anx-sub-h">Сургалт бүрээр</h4><div class="an-table-wrap"><table class="tbl"><thead><tr><th>Сургалт</th><th>Үзэлт</th><th>Зочин</th><th>Дундаж хугацаа</th></tr></thead><tbody>${d.courses.map((c) => `<tr><td>${esc(c.title || "—")}</td><td><b>${c.views}</b></td><td>${c.unique}</td><td>${c.avg_sec ? secs(c.avg_sec) : "—"}</td></tr>`).join("")}</tbody></table></div>` : ""}
          <h4 class="anx-sub-h">Сүүлийн зочид</h4>
          ${d.recent.length ? `<ul class="vs-recent">${d.recent.slice(0, 12).map((v) => `<li><span class="vs-av ${v.member ? "" : "guest"}" style="--h:${hueOfName(v.name)}">${v.member ? esc(initialsOf(v.name)) : ico("profile", 16)}</span>
            <span class="vs-who"><b>${esc(v.name)}</b><small>${esc(v.kind === "course" ? v.title || "Сургалт" : "Профайл")} · ${fmtDate(v.at)}</small></span>
            <span class="vs-meta"><b>${v.seconds ? secs(v.seconds) : "—"}</b><small>${esc(DEV[v.device] || v.device)} · ${esc(v.referrer)}</small></span></li>`).join("")}</ul>`
            : `<p class="muted small" style="margin:0">Одоогоор зочин алга.</p>`}</div>
        <div class="vs-side"><h4 class="anx-sub-h">Хэр удаан байсан бэ</h4>${barList(d.durations)}
          <h4 class="anx-sub-h">Төхөөрөмж</h4>${barList(d.devices, (x) => esc(DEV[x.name] || x.name))}
          <h4 class="anx-sub-h">Хаанаас ирсэн бэ</h4>${barList(d.referrers)}</div>
      </div>`;
  };
  $("#ov-visits").onclick = (e) => {
    const k = e.target.closest("[data-vk]"), dd = e.target.closest("[data-vd]");
    if (k) { vst.kind = k.dataset.vk; $$("[data-vk]").forEach((b) => b.setAttribute("aria-pressed", String(b === k))); loadVisits(); }
    if (dd) { vst.days = +dd.dataset.vd; $$("[data-vd]").forEach((b) => b.setAttribute("aria-pressed", String(b === dd))); loadVisits(); }
  };
  loadVisits();

  const agenda = (i) => {
    const list = perDay(days[i]);
    $("#agenda").innerHTML = list.map((m) => { const end = new Date(new Date(m.starts_at).getTime() + m.duration_min * 60000); return `
      <div class="slot"><div class="slot-time"><b>${fmtTime(m.starts_at)}</b><small>${fmtTime(end)}</small></div>
        <div class="slot-body"><div class="slot-top"><span class="chip chip-gold">Live</span>${titleOf[m.course_id] ? `<span class="muted small">${esc(titleOf[m.course_id])}</span>` : ""}<span class="chip chip-amber" style="margin-left:auto">${until(m.starts_at)}</span></div>
        <strong>${esc(m.title)}</strong><small class="muted">${m.duration_min} мин${m.price ? ` · ${money(m.price)} · ${m.buyers || 0} худалдаж авсан` : ""}</small>
        <div class="slot-actions"><a class="btn btn-sm btn-danger" href="${esc(m.meet_url)}" target="_blank" rel="noopener">${ico("live", 15)}Live</a>${m.course_id ? `<a class="btn btn-sm btn-ghost" href="#course=${esc(m.course_id)}">${ico("courses", 15)}Сургалт</a>` : ""}</div></div></div>`; }).join("") ||
      `<p class="muted small" style="padding:18px 4px;margin:0">Энэ өдөр товлосон хичээл алга.</p>`;
  };
  $("#days").onclick = (e) => {
    const b = e.target.closest(".day"); if (!b) return;
    const on = !b.classList.contains("active"); // дахин дарвал хумигдана
    $$(".day", $("#days")).forEach((x) => x.classList.toggle("active", on && x === b));
    if (on) agenda(+b.dataset.i); else $("#agenda").innerHTML = "";
  };
  if (!next) agenda(0); // товлосон хичээлгүй бол өнөөдрийг харуулна

  const tick = () => { const c = $("#clock"); if (c) { const d = new Date(); c.innerHTML = `${fmtTime(d)}<small>:${String(d.getSeconds()).padStart(2, "0")}</small>`; } };
  tick(); const iv = setInterval(tick, 1000);
  cleanup = () => clearInterval(iv);
}

/* ---------- Сургалтууд: групп шиг удирдана ----------
   Сургалт бүр нэг "групп". Дотор нь багш хичээлээ мэдээ нийтэлж байгаа мэт бичиж,
   хавсралт нэмж, урсгал (feed) хэлбэрээр харж, тэр дор нь засна. */

const courseFormHTML = (c = {}) => `
  <label>Сургалтын нэр<input name="title" required maxlength="200" value="${esc(c.title || "")}" placeholder="ЭЕШ Математик — Бүрэн бэлтгэл"></label>
  <label>Тайлбар<textarea name="description" rows="4" placeholder="Энэ сургалтаар юу сурах вэ?">${esc(c.description || "")}</textarea></label>
  <div class="form-row"><label>Багц үнэ (₮)<input name="price" type="number" min="0" step="1000" value="${c.price || 0}"><small class="muted">0 бол хичээл тус бүрээр зарна</small></label>
  <label class="check" style="align-self:center"><input type="checkbox" name="published" ${c.published ? "checked" : ""}> Нийтлэх (профайл дээр харагдана)</label></div>
  <fieldset><legend>Хичээлийн нээлт</legend>
    <label class="check"><input type="checkbox" name="drip" ${c.drip ? "checked" : ""}> Дарааллаар нээгдэнэ — суралцагч өмнөх хичээлээ үзэж, доторх асуултуудад нь бүгдэд нь зөв хариулсны дараа дараагийнх нь (тохируулсан цаг/хоногийн дараа) нээгдэнэ</label>
    <fieldset class="warn-set"><legend>Анхаарлын хяналт</legend>
      <div class="pe-grid pe-grid-2"><label>Таб/цонх солих сануулгын тоо<input name="max_warnings" type="number" min="1" max="20" value="${c.max_warnings || 3}"><small>Энэ тоонд хүрмэгц хичээл зогсож, тэр хичээл рүү дахин орж чадахгүй.</small></label>
      <label>Хаагдсаны дараа<select name="block_minutes">${[[30, "30 минутын дараа автоматаар (анхдагч)"], [5, "5 минутын дараа автоматаар"], [10, "10 минутын дараа автоматаар"], [15, "15 минутын дараа автоматаар"], [60, "1 цагийн дараа автоматаар"], [180, "3 цагийн дараа автоматаар"], [1440, "1 өдрийн дараа автоматаар"], [10080, "7 хоногийн дараа автоматаар"], [-1, "Багш «Дахин нээх» дарах хүртэл"]].map(([v, t]) => `<option value="${v}" ${(c.block_minutes || (c.block_hours || 0) * 60 || 30) === v ? "selected" : ""}>${t}</option>`).join("")}</select><small>Давтан хаагдвал хугацаа 2 дахин уртасна. Хаагдмагц танд улаан «Дахин нээх» товчтой мэдэгдэл очно — хугацаанаас өмнө ч нээж болно.</small></label></div></fieldset>
    <label class="check"><input type="checkbox" name="unlock_all_paid" ${c.unlock_all_paid !== false ? "checked" : ""}> Багцын төлбөр төлсөн суралцагчид бүх хичээл шууд нээлттэй</label>
    <small class="muted">Хичээл бүрийн хүлээх хугацааг (шууд, цаг, 7 хоног, сар) хичээл дээр нь тохируулна.</small></fieldset>
  <input type="hidden" name="camera" value="${esc(c.camera || "optional")}">
  <label class="check"><input type="checkbox" name="certificate" ${c.certificate ? "checked" : ""}> 🎓 Сургалтыг дүүргэсэн суралцагчид сертификат олгоно (хайлтад «Сертификаттай» шошго гарна)</label>`;
const courseBody = (f) => ({ title: f.title.value, description: f.description.value, price: +f.price.value || 0, published: f.published.checked, drip: f.drip.checked, unlock_all_paid: f.unlock_all_paid.checked, certificate: f.certificate.checked, camera: f.camera.value,
  max_warnings: f.max_warnings ? +f.max_warnings.value || 3 : undefined, block_minutes: f.block_minutes ? +f.block_minutes.value || 30 : undefined, block_hours: 0 });
const FORMATS = [["", "— Сургалтын хэлбэр —"], ["lecture", "Лекц"], ["seminar", "Семинар"], ["practice", "Дадлага"], ["lab", "Лаборатори"]];
const MODES = [["", "— Заах аргын төрөл —"], ["classroom", "Танхимын"], ["online", "Цахим"], ["blended", "Холимог"]];
const kindName = (k) => (FORMATS.find(([v]) => v === k)?.[1] || MODES.find(([v]) => v === k)?.[1] || "");
const UNLOCKS = [[0, "Шууд (өмнөхийг үзмэгц)"], [1, "1 цагийн дараа"], [24, "1 хоногийн дараа"], [72, "3 хоногийн дараа"], [168, "7 хоногийн дараа"], [336, "14 хоногийн дараа"], [720, "1 сарын дараа"]];

async function courses() {
  const list = await api("/api/me/courses");
  searchCourses = list;
  main.innerHTML = `<div class="gp-list-head"><div><h1>Миний сургалтууд</h1><p class="muted">${list.length} сургалт · ${list.filter((c) => c.published).length} нийтлэгдсэн</p></div>
      <button class="btn btn-gold" id="newCourse">${ico("plus", 18)}Шинэ сургалт үүсгэх</button></div>
    <div class="group-grid gp-grid">${list.map((c) => `
      <a class="group" href="#course=${esc(c.id)}" style="--h:${hueOfName(c.title)}"><div class="group-art"><span class="chip">${c.published ? "Нийтлэгдсэн" : "Ноорог"}</span><b>${esc(c.title.trim()[0] || "?")}</b></div>
      <div class="group-body"><strong>${esc(c.title)}</strong><small>${ico("book", 14)}${c.lesson_count} хичээл · ${c.free_lesson_count} үнэгүй</small><small>${ico("eye", 14)}${c.views || 0} үзэлт · ${money(c.price) === "Үнэгүй" ? "Хичээлээр" : money(c.price)}</small>
      <span class="btn btn-glass btn-sm" style="margin-top:6px">Удирдах</span></div></a>`).join("")}${list.length ? "" : `
      <button class="group group-new" id="newCourse2">${ico("plus", 26)}<strong>Анхны сургалтаа үүсгэх</strong></button>`}</div>
    <div class="modal" id="courseModal"><div class="modal-card"><button class="icon-btn modal-x" data-close aria-label="Хаах">${ico("x", 18)}</button>
      <h3 class="h3">Шинэ сургалт үүсгэх</h3><form class="form" id="courseForm">${courseFormHTML()}<p class="form-error" role="alert"></p>
      <div class="hero-cta" style="margin:0;justify-content:flex-end"><button type="button" class="btn btn-ghost" data-close>Болих</button><button class="btn btn-gold">Үүсгэх</button></div></form></div></div>`;
  const f = $("#courseForm"), open = () => { SG.openModal($("#courseModal")); f.title.focus(); };
  $("#newCourse").onclick = open; if ($("#newCourse2")) $("#newCourse2").onclick = open;
  try { if (sessionStorage.getItem("sg_new_course")) { sessionStorage.removeItem("sg_new_course"); open(); } } catch {}
  f.onsubmit = async (e) => {
    e.preventDefault();
    try { const c = await api("/api/courses", { method: "POST", body: courseBody(f) }); toast("Сургалт үүслээ ✓"); location.hash = "course=" + c.id; }
    catch (err) { $(".form-error", f).textContent = err.message; }
  };
}

let library = null;
async function loadLibrary() { library = await api("/api/me/files"); return library; }

/* ---------- Хичээлийн блок засварлагч: текст, гарчиг, зураг, дуу, видео, файл, асуулт ---------- */
const BE = {
  text: ["text", "Текст"], heading: ["heading", "Гарчиг"], image: ["image", "Зураг"], audio: ["mic", "Дуу"],
  video: ["live", "Видео"], file: ["clip", "Файл"], quiz: ["quiz", "Асуулт"], embed: ["link", "HTML embed"],
};
// Төрөл тус бүрийн хэмжээний хязгаар (MB). Зургийг сервер WebP болгож 1000px хүртэл багасгана.
const BE_LIMIT = { image: 20, audio: 100, video: 2048, file: 200 };
const BE_ACCEPT = { image: "image/png,image/jpeg,image/webp,image/gif", audio: "audio/*,.mp3,.m4a,.wav,.webm", video: "video/*,.mp4,.webm,.mov,.m4v", file: ".pdf,.doc,.docx,.ppt,.pptx,.xls,.xlsx,.odt,.odp,.rtf,.txt,.zip,.epub" };
const beId = () => Math.random().toString(36).slice(2, 10).padEnd(6, "0");
const stripQ = (u) => (u && u.startsWith("/files/") ? u.split("?")[0] : u || "");
const typeOfFile = (f) => /^image\//.test(f.type) ? "image" : /^audio\//.test(f.type) ? "audio" : /^video\//.test(f.type) ? "video" : "file";
const typeOfURL = (u) => { const e = SG.extOf(u); return /youtu\.?be|vimeo/.test(u) || [".mp4", ".webm", ".mov", ".m4v"].includes(e) ? "video" : [".mp3", ".m4a", ".wav"].includes(e) ? "audio" : [".webp", ".jpg", ".jpeg", ".png", ".gif"].includes(e) ? "image" : "file"; };

// Хуучин хичээл (content + video_url) → блокууд.
function lessonBlocks(l) {
  if (l?.blocks?.length) return l.blocks;
  const out = [];
  if (l?.video_url) out.push({ id: beId(), type: typeOfURL(stripQ(l.video_url)), url: l.video_url, name: stripQ(l.video_url).startsWith("/files/") ? decodeURIComponent(stripQ(l.video_url).split("/").pop().replace(/^[0-9a-f]{16}__/, "")) : "Видео холбоос" });
  if (l?.content) out.push({ id: beId(), type: "text", text: l.content });
  return out.length ? out : [{ id: beId(), type: "text", text: "" }];
}

// contenteditable → аюулгүй тэмдэглэгээ (richHTML-ийн эсрэг).
function domToMd(root) {
  const inl = (n) => {
    if (n.nodeType === 3) return n.nodeValue.replace(/ /g, " ");
    if (n.nodeType !== 1) return "";
    const t = n.tagName, kids = () => [...n.childNodes].map(inl).join(""), st = n.getAttribute("style") || "";
    if (t === "BR") return "\n";
    const k = kids(); if (!k.trim()) return k;
    if (t === "B" || t === "STRONG" || /font-weight:\s*(bold|[6-9]00)/.test(st)) return `**${k}**`;
    if (t === "I" || t === "EM" || /font-style:\s*italic/.test(st)) return `*${k}*`;
    if (t === "U" || /underline/.test(st)) return `__${k}__`;
    if (t === "MARK" || /background/.test(st)) return `==${k}==`;
    if (t === "A" && /^https?:\/\//.test(n.getAttribute("href") || "")) return `[${k}](${n.getAttribute("href")})`;
    return k;
  };
  const lines = []; let cur = "";
  const BLOCK = /^(DIV|P|H\d|UL|OL|BLOCKQUOTE|LI)$/;
  const flush = () => { if (cur !== "") lines.push(...cur.split("\n")); cur = ""; };
  const walk = (n) => {
    for (const c of n.childNodes) {
      const t = c.nodeType === 1 ? c.tagName : "";
      if (!BLOCK.test(t)) { cur += inl(c); continue; } // текст ба мөр доторх хэлбэр нэг мөрөнд
      flush();
      if (t === "UL" || t === "OL") { [...c.children].forEach((li, i) => lines.push((t === "OL" ? `${i + 1}. ` : "- ") + inl(li).replace(/\n+/g, " ").trim())); lines.push(""); }
      else if (t === "BLOCKQUOTE") { inl(c).split("\n").forEach((x) => lines.push("> " + x)); lines.push(""); }
      else if (c.querySelector("ul,ol,blockquote,div,p")) walk(c);
      else { lines.push(inl(c)); if (t === "P") lines.push(""); }
    }
    flush();
  };
  walk(root);
  return lines.join("\n").replace(/\n{3,}/g, "\n\n").trim();
}

const QKINDS = [["single", "◉ Нэг сонголт"], ["multi", "☑ Олон сонголт"], ["text", "✎ Бичгээр"], ["match", "⇄ Харгалзуулах"], ["image", "🎯 Зурган дээр заах"]];
function quizBodyHTML(kind, id, q) {
  if (kind === "single" || kind === "multi") {
    const opts = q.options?.length ? q.options : ["", ""];
    return `<div class="be-opts">${opts.map((o, i) => beOptHTML(id, kind === "multi", o, (q.correct || []).includes(i))).join("")}</div>
      <div class="be-qfoot"><button type="button" class="btn btn-glass btn-sm" data-opt-add>${ico("plus", 15)}Хариулт нэмэх</button><span class="muted small">Зөв хариулт(ууд)-ын өмнөх тэмдгийг сонгоно.</span></div>`;
  }
  if (kind === "text") {
    const ans = q.answers?.length ? q.answers : [""];
    return `<div class="be-answers">${ans.map((a) => `<div class="be-opt"><span class="be-ok">✓</span><input class="be-ans" maxlength="200" placeholder="Зөвд тооцох хариулт" value="${esc(a)}"><button type="button" class="icon-btn" data-row-del aria-label="Хасах">${ico("x", 14)}</button></div>`).join("")}</div>
      <div class="be-qfoot"><button type="button" class="btn btn-glass btn-sm" data-ans-add>${ico("plus", 15)}Өөр хувилбар нэмэх</button><span class="muted small">Том жижиг үсэг, илүүдэл зай, цэг таслалыг тооцохгүй.</span></div>`;
  }
  if (kind === "match") {
    const L = q.left?.length ? q.left : ["", ""], R = q.right?.length ? q.right : ["", ""];
    return `<div class="be-pairs">${L.map((l, i) => pairHTML(l, R[i] || "")).join("")}</div>
      <div class="be-qfoot"><button type="button" class="btn btn-glass btn-sm" data-pair-add>${ico("plus", 15)}Хос нэмэх</button><span class="muted small">Зүүн талыг зөв хостой нь нэг мөрөнд бичнэ — суралцагчид баруун тал холилдож харагдана.</span></div>`;
  }
  return `<label class="be-radius">Зөв хэсгийн хэмжээ<input type="range" class="be-r" min="2" max="40" value="${q.spot?.r || 8}"></label>
    <p class="muted small" style="margin:0">Дээрх зураг дээр зөв хариултын хэсгийг дарж тэмдэглэнэ үү.</p>`;
}
const pairHTML = (l, r) => `<div class="be-opt be-pair"><input class="be-l" maxlength="200" placeholder="Зүүн (ж: Япон)" value="${esc(l)}"><span>⇄</span><input class="be-rr" maxlength="200" placeholder="Баруун (ж: Токио)" value="${esc(r)}"><button type="button" class="icon-btn" data-row-del aria-label="Хасах">${ico("x", 14)}</button></div>`;
function qimgHTML(kind, url, spot) {
  if (!url) return `<button type="button" class="btn btn-glass btn-sm" data-qimg>${ico("image", 15)}${kind === "image" ? "Зураг оруулах (заавал)" : "Асуултад зураг нэмэх"}</button><div class="progress" hidden><i></i></div>`;
  const r = spot?.r || 8;
  return `<div class="be-spotwrap ${kind === "image" ? "pick" : ""}"><img src="${esc(url)}" alt="" draggable="false">${kind === "image" && spot?.x != null && spot?.x !== "" ? `<i class="be-spot" style="left:${spot.x}%;top:${spot.y}%;width:${r * 2}%;height:${r * 2}%"></i>` : ""}</div>
    <div class="be-media-tools"><span class="muted small">${kind === "image" ? "Зураг дээр дарж зөв хэсгийг заана" : "Асуултын зураг"}</span><span><button type="button" class="btn btn-glass btn-sm" data-qimg>Солих</button> <button type="button" class="btn btn-ghost btn-sm" data-qimg-del>Хасах</button></span></div>`;
}
function beBlockHTML(b) {
  const [icn, label] = BE[b.type];
  let body = "";
  if (b.type === "text") body = `<div class="be-rt-bar" role="toolbar" aria-label="Текстийн хэлбэр">
      <button type="button" data-cmd="bold" title="Тод (Ctrl+B)"><b>B</b></button><button type="button" data-cmd="italic" title="Налуу (Ctrl+I)"><i>I</i></button>
      <button type="button" data-cmd="underline" title="Доогуур зураас (Ctrl+U)"><u>U</u></button><button type="button" data-cmd="mark" title="Тодруулах"><mark>A</mark></button>
      <span class="be-sep"></span><button type="button" data-cmd="insertUnorderedList" title="Жагсаалт">• —</button><button type="button" data-cmd="insertOrderedList" title="Дугаартай жагсаалт">1.</button>
      <button type="button" data-cmd="quote" title="Ишлэл">❝</button><button type="button" data-cmd="link" title="Холбоос">${ico("link", 15)}</button><button type="button" data-cmd="removeFormat" title="Хэлбэр арилгах">⨯</button></div>
    <div class="be-rt rb-text" contenteditable="true" data-ph="Текстээ бичнэ үү… (сонгоод дээрх товчоор хэлбэржүүлнэ)">${SG.richHTML(b.text || "")}</div>`;
  else if (b.type === "heading") body = `<input class="be-h" maxlength="200" placeholder="Гарчиг бичнэ үү" value="${esc(b.text || "")}">`;
  else if (b.type === "embed") body = `<textarea class="be-embed" rows="5" maxlength="60000" spellcheck="false" placeholder="HTML код буулгана уу — жишээ: <iframe src=&quot;https://…&quot;></iframe>, Google Forms, Canva, GeoGebra, Desmos, Padlet, Quizlet…">${esc(b.text || "")}</textarea>
      <div class="be-embed-bar"><label class="small">Өндөр<input class="be-embed-h" type="number" min="80" max="2000" step="20" value="${b.height || 420}"> px</label><button type="button" class="btn btn-glass btn-sm" data-embed-preview>Урьдчилан харах</button><span class="muted small">Тусгаарлагдсан хүрээнд аюулгүй ажиллана</span></div>
      <div class="be-embed-prev"></div>`;
  else if (b.type === "quiz") {
    const q = b.quiz || { kind: "single", question: "", options: ["", ""], correct: [0], explain: "" };
    const kind = q.kind || (q.multi ? "multi" : "single");
    body = `<div class="be-qhead"><select class="be-qkind" aria-label="Асуултын төрөл">${QKINDS.map(([k, t]) => `<option value="${k}" ${k === kind ? "selected" : ""}>${t}</option>`).join("")}</select>
        <label class="be-pts">Оноо<input class="be-points" type="number" min="0" max="100" value="${q.points || 1}"></label></div>
      <input class="be-q" maxlength="1000" placeholder="Асуултаа бичнэ үү" value="${esc(q.question || "")}">
      <div class="be-qimg" data-url="${esc(stripQ(q.image || ""))}" data-preview="${esc(q.image || "")}" data-x="${q.spot?.x ?? ""}" data-y="${q.spot?.y ?? ""}">${qimgHTML(kind, q.image, q.spot)}</div>
      <div class="be-qbody">${quizBodyHTML(kind, b.id, q)}</div>
      <input class="be-ex" maxlength="1000" placeholder="Тайлбар (заавал биш) — хариулсны дараа харагдана" value="${esc(q.explain || "")}">`;
  } else body = `<div class="be-media" data-url="${esc(stripQ(b.url || ""))}" data-name="${esc(b.name || "")}" data-size="${b.size || 0}">${b.url ? beMediaPreview(b) : beDropHTML(b.type)}</div>
      <input class="be-cap" maxlength="500" placeholder="Тайлбар (заавал биш)" value="${esc(b.text || "")}" ${b.url ? "" : "hidden"}>
      ${b.type === "file" ? `<label class="check small be-dl"><input type="checkbox" ${b.download ? "checked" : ""}> Суралцагч татаж авахыг зөвшөөрөх (анхдагч: зөвхөн үзнэ, PDF нь 3D номоор нээгдэнэ)</label>` : ""}`;
  return `<div class="be-block" data-id="${esc(b.id)}" data-type="${b.type}">
    <div class="be-side"><button type="button" class="ol-handle" data-be-drag title="Чирж зөөх" aria-label="${label} хэсгийг зөөх (↑↓ товч)">${ico("grip", 18)}</button></div>
    <div class="be-main"><span class="be-tag">${ico(icn, 14)}${label}</span>${body}</div>
    <div class="be-acts"><button type="button" class="icon-btn" data-be-del title="Устгах" aria-label="Устгах">${ico("x", 16)}</button></div></div>`;
}
const beOptHTML = (id, multi, val, on) => `<div class="be-opt"><input type="${multi ? "checkbox" : "radio"}" name="c-${esc(id)}" ${on ? "checked" : ""} title="Зөв хариулт" aria-label="Зөв хариулт">
  <input class="be-opt-t" maxlength="300" placeholder="Хариулт" value="${esc(val)}"><button type="button" class="icon-btn" data-opt-del aria-label="Хариулт хасах">${ico("x", 14)}</button></div>`;
const beDropHTML = (type) => `<div class="be-drop"><span>${ico(BE[type][0], 22)}</span><div><b>${type === "image" ? "Зураг" : type === "audio" ? "Дуу бичлэг" : type === "video" ? "Видео" : "Файл (PDF, Word, PowerPoint…)"}</b>
    <small class="muted">Энд чирж оруулах эсвэл сонгоно · ${BE_LIMIT[type] >= 1024 ? BE_LIMIT[type] / 1024 + " GB" : BE_LIMIT[type] + " MB"} хүртэл${type === "image" ? " · автоматаар жижгэрнэ" : ""}</small></div>
    <div class="be-drop-acts"><button type="button" class="btn btn-gold btn-sm" data-pick>${ico("clip", 15)}Файл сонгох</button>
    ${type === "audio" ? `<button type="button" class="btn btn-glass btn-sm" data-rec>${ico("mic", 15)}Бичлэг хийх</button>` : ""}
    ${type === "video" ? `<button type="button" class="btn btn-glass btn-sm" data-vurl>${ico("link", 15)}YouTube холбоос</button>` : ""}</div>
    <div class="progress" hidden><i></i></div></div>`;
function beMediaPreview(b) {
  const u = b.preview || b.url;
  const tools = `<div class="be-media-tools"><span class="muted small">${esc(b.name || "")}${b.size ? " · " + SG.fmtBytes(b.size) : ""}</span><button type="button" class="btn btn-glass btn-sm" data-replace>Солих</button></div>`;
  if (b.type === "image") return `<img class="be-img" src="${esc(u)}" alt="">${tools}`;
  if (b.type === "audio") return `<audio src="${esc(u)}" controls preload="metadata" style="width:100%"></audio>${tools}`;
  if (b.type === "video") return `<div class="player be-player">${SG.mediaHTML(u, b.name)}</div>${tools}`;
  return `<div class="rb-file"><span class="rb-file-ico">📄</span><span><b>${esc(b.name || "Файл")}</b><small>${esc(SG.extOf(stripQ(u)).slice(1).toUpperCase())}</small></span></div>${tools}`;
}

// Нэг маягт дээр засварлагчийг холбоно. f._be.collect() → блокууд.
// Рубрик засварлагч: түвшин (багана) ба шалгуур (мөр)-ийг динамикаар нэмж/хасна; нүд бүрт тайлбар, оноо.
function mountRubricEditor(box, init) {
  const rid = () => Math.random().toString(36).slice(2, 10).padEnd(6, "0");
  const st = init?.levels?.length >= 2 ? JSON.parse(JSON.stringify(init))
    : { levels: ["Маш сайн", "Сайн", "Дунд", "Хангалтгүй"], criteria: [{ id: rid(), name: "Агуулга", points: [10, 7, 4, 1], desc: ["", "", "", ""] }, { id: rid(), name: "Бүтэц, хэлбэр", points: [5, 4, 2, 1], desc: ["", "", "", ""] }] };
  st.criteria.forEach((c) => { c.points = st.levels.map((_, j) => +(c.points?.[j] ?? 0)); c.desc = st.levels.map((_, j) => c.desc?.[j] ?? ""); });
  const top = (c) => Math.max(0, ...c.points.map((p) => +p || 0));
  const total = () => st.criteria.reduce((a, c) => a + top(c), 0);
  const draw = () => {
    box.innerHTML = `<div class="rb-wrap"><table class="rb-ed">
      <thead><tr><th class="rb-corner">Шалгуур <span>↓</span> · Түвшин <span>→</span></th>${st.levels.map((lv, j) => `<th><div class="rb-lv"><input data-lv="${j}" value="${esc(lv)}" maxlength="60" placeholder="Түвшний нэр" aria-label="${j + 1}-р түвшний нэр">${st.levels.length > 2 ? `<button type="button" class="icon-btn rb-x" data-del-lv="${j}" title="Энэ түвшинг (баганыг) хасах" aria-label="Түвшин хасах">${ico("x", 13)}</button>` : ""}</div></th>`).join("")}
        ${st.levels.length < 8 ? `<th class="rb-add-col"><button type="button" class="rb-add" data-add-lv title="Түвшин (багана) нэмэх">${ico("plus", 16)}<span>Багана</span></button></th>` : ""}</tr></thead>
      <tbody>${st.criteria.map((c, i) => `<tr><th><div class="rb-cr"><textarea data-cn="${i}" rows="2" maxlength="200" placeholder="Шалгуур (жишээ нь: Агуулга)" aria-label="${i + 1}-р шалгуур">${esc(c.name)}</textarea>${st.criteria.length > 1 ? `<button type="button" class="icon-btn rb-x" data-del-cr="${i}" title="Энэ шалгуурыг (мөрийг) хасах" aria-label="Шалгуур хасах">${ico("x", 13)}</button>` : ""}</div><small class="rb-max" data-max="${i}">дээд ${top(c)} оноо</small></th>
        ${st.levels.map((_, j) => `<td><textarea data-cd="${i}:${j}" rows="2" maxlength="500" placeholder="Тайлбар (заавал биш)" aria-label="${esc(c.name || "Шалгуур")} — ${esc(st.levels[j])} тайлбар">${esc(c.desc[j] || "")}</textarea><label class="rb-pts"><input type="number" data-cp="${i}:${j}" min="0" max="1000" value="${c.points[j]}" aria-label="оноо"><span>оноо</span></label></td>`).join("")}${st.levels.length < 8 ? "<td class=\"rb-add-col\"></td>" : ""}</tr>`).join("")}</tbody></table></div>
      <div class="rb-foot">${st.criteria.length < 20 ? `<button type="button" class="btn btn-glass btn-sm" data-add-cr>${ico("plus", 15)}Шалгуур (мөр) нэмэх</button>` : ""}<span class="rb-total">Нийт дээд оноо <b>${total()}</b></span></div>`;
  };
  box.addEventListener("input", (e) => {
    const t = e.target, d = t.dataset;
    if (d.lv != null) st.levels[+d.lv] = t.value;
    else if (d.cn != null) st.criteria[+d.cn].name = t.value;
    else if (d.cd) { const [i, j] = d.cd.split(":").map(Number); st.criteria[i].desc[j] = t.value; }
    else if (d.cp) {
      const [i, j] = d.cp.split(":").map(Number); st.criteria[i].points[j] = Math.max(0, Math.min(1000, +t.value || 0));
      if (+t.value > 1000) t.value = 1000; else if (+t.value < 0) t.value = 0;
      $(".rb-total b", box).textContent = total(); const m = $(`[data-max="${i}"]`, box); if (m) m.textContent = `дээд ${top(st.criteria[i])} оноо`;
    }
  });
  box.addEventListener("click", (e) => {
    const b = e.target.closest("button"); if (!b || !box.contains(b)) return;
    if (b.dataset.addLv != null) { st.levels.push(`${st.levels.length + 1}-р түвшин`); st.criteria.forEach((c) => { c.points.push(0); c.desc.push(""); }); draw(); $(`[data-lv="${st.levels.length - 1}"]`, box)?.select(); }
    else if (b.dataset.delLv != null) { const j = +b.dataset.delLv; st.levels.splice(j, 1); st.criteria.forEach((c) => { c.points.splice(j, 1); c.desc.splice(j, 1); }); draw(); }
    else if (b.dataset.addCr != null) { st.criteria.push({ id: rid(), name: "", points: st.criteria[0] ? [...st.criteria[0].points] : st.levels.map(() => 0), desc: st.levels.map(() => "") }); draw(); $(`[data-cn="${st.criteria.length - 1}"]`, box)?.focus(); }
    else if (b.dataset.delCr != null) { st.criteria.splice(+b.dataset.delCr, 1); draw(); }
  });
  draw();
  return {
    value: () => ({ levels: st.levels.map((x) => x.trim()), criteria: st.criteria.map((c) => ({ id: c.id, name: c.name.trim(), points: c.points.map((p) => +p || 0), desc: c.desc.map((x) => x.trim()) })) }),
    check() {
      if (st.levels.some((x) => !x.trim())) throw new Error("Рубрикийн түвшин (багана) бүрт нэр өгнө үү");
      if (st.criteria.some((c) => !c.name.trim())) throw new Error("Рубрикийн шалгуур (мөр) бүрт нэр өгнө үү");
      if (!total()) throw new Error("Рубрикийн нийт оноо 0-ээс их байх ёстой");
    },
  };
}

function mountBlockEditor(f, blocks) {
  const rb = f.querySelector?.(".gr-rubric");
  if (rb) { let init = null; try { init = JSON.parse(rb.dataset.rubric || "null"); } catch {} f._rb = mountRubricEditor(rb, init); }
  const box = $(".be", f);
  box.innerHTML = `<div class="be-list">${blocks.map(beBlockHTML).join("")}</div>
    <div class="be-add"><span class="muted small">Нэмэх:</span>${Object.entries(BE).map(([t, [icn, label]]) => `<button type="button" class="btn btn-glass btn-sm" data-add="${t}">${ico(icn, 15)}${label}</button>`).join("")}</div>
    <div class="be-import"><button type="button" class="btn btn-glass btn-sm" data-import>${ico("files", 15)}Excel-ээс асуулт оруулах</button>
      <a class="link small" href="/api/quiz-template.xlsx" download>Excel загвар татах</a><input type="file" class="be-xlsx" accept=".xlsx,.csv" hidden></div>
    <p class="muted small be-tip">${ico("grip", 13)} Хэсгүүдийг бариад чирж байрлалыг солино. Компьютерээсээ файл чирж оруулж болно.</p>
    <input type="file" class="be-file" hidden>`;
  const list = $(".be-list", box), fileIn = $(".be-file", box);
  let target = null;
  const add = (type, after, data = {}) => {
    const b = { id: beId(), type, ...data }; const tmp = document.createElement("div"); tmp.innerHTML = beBlockHTML(b);
    const el = tmp.firstElementChild; after ? after.after(el) : list.append(el);
    (el.querySelector(".be-rt, .be-h, .be-q") || el).focus?.();
    return el;
  };
  const setMedia = (el, info) => {
    const m = $(".be-media", el); m.dataset.url = info.path; m.dataset.name = info.original_name || ""; m.dataset.size = info.size || 0;
    m.innerHTML = beMediaPreview({ type: el.dataset.type, url: info.path, preview: info.url, name: info.original_name, size: info.size });
    $(".be-cap", el).hidden = false; SG.hydrateBooks?.(m);
  };
  const upload = (el, file) => {
    const type = el.dataset.type, lim = BE_LIMIT[type];
    if (file.size > lim * 1048576) { toast(`Файл хэт том: ${SG.fmtBytes(file.size)} — ${BE[type][1].toLowerCase()} ${lim} MB хүртэл`, true); return; }
    const m = $(".be-media", el); if (!$(".be-drop", m)) m.innerHTML = beDropHTML(type);
    const bar = $(".progress", m), fill = $("i", bar); bar.hidden = false; fill.style.width = "3%";
    const x = new XMLHttpRequest(), fd = new FormData(); fd.append("file", file);
    x.open("POST", "/api/me/files?visibility=private"); x.setRequestHeader("Authorization", "Bearer " + Auth.token);
    x.upload.onprogress = (e) => { if (e.lengthComputable) fill.style.width = Math.max(3, Math.round(e.loaded / e.total * 100)) + "%"; };
    x.onload = () => { let d = null; try { d = JSON.parse(x.responseText); } catch {} bar.hidden = true;
      if (x.status >= 300) return toast(d?.error || "Хуулж чадсангүй", true);
      setMedia(el, d); toast(d.status === "processing" ? "Хуулагдлаа — видеог боловсруулж байна" : "Хуулагдлаа ✓"); library = null; };
    x.onerror = () => { bar.hidden = true; toast("Сүлжээний алдаа — дахин оролдоно уу", true); };
    x.send(fd);
  };
  const pick = (el) => { target = el; fileIn.accept = BE_ACCEPT[el.dataset.type] || ""; fileIn.value = ""; fileIn.click(); };
  fileIn.onchange = () => { if (fileIn.files[0] && target) upload(target, fileIn.files[0]); };

  // Дуу бичих (микрофон → webm).
  let rec = null;
  const record = async (el, btn) => {
    if (rec) { rec.stop(); return; }
    let stream; try { stream = await navigator.mediaDevices.getUserMedia({ audio: true }); } catch { toast("Микрофон ашиглах зөвшөөрөл өгнө үү", true); return; }
    const chunks = [], mr = new MediaRecorder(stream); rec = mr; const t0 = Date.now();
    btn.classList.add("rec-on"); const tick = setInterval(() => { const s = Math.round((Date.now() - t0) / 1000); btn.textContent = `■ Зогсоох ${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`; }, 500);
    mr.ondataavailable = (e) => e.data.size && chunks.push(e.data);
    mr.onstop = () => { clearInterval(tick); stream.getTracks().forEach((t) => t.stop()); rec = null;
      const blob = new Blob(chunks, { type: mr.mimeType || "audio/webm" }), ext = (mr.mimeType || "").includes("mp4") ? "m4a" : "webm";
      upload(el, new File([blob], `bichleg-${new Date().toISOString().slice(0, 16).replace(/[:T]/g, "-")}.${ext}`, { type: blob.type })); };
    mr.start();
  };

  // Асуултын одоогийн утгууд (төрлөөс хамаарна).
  const quizOf = (el) => {
    const kind = $(".be-qkind", el).value, qi = $(".be-qimg", el);
    const q = { kind, question: $(".be-q", el).value.trim(), points: +$(".be-points", el).value || 1, explain: $(".be-ex", el).value, image: qi.dataset.url || "" };
    if (kind === "single" || kind === "multi") { const opts = $$(".be-opts .be-opt", el); q.options = opts.map((o) => $(".be-opt-t", o).value); q.correct = opts.map((o, i) => ($("input", o).checked ? i : -1)).filter((i) => i >= 0); q.multi = kind === "multi"; }
    if (kind === "text") q.answers = $$(".be-ans", el).map((i) => i.value);
    if (kind === "match") { q.left = $$(".be-l", el).map((i) => i.value); q.right = $$(".be-rr", el).map((i) => i.value); }
    if (kind === "image" && qi.dataset.x !== "" && qi.dataset.x != null) q.spot = { x: +qi.dataset.x, y: +qi.dataset.y, r: +($(".be-r", el)?.value || 8) };
    return q;
  };
  const drawSpot = (el) => {
    const qi = $(".be-qimg", el), wrap = $(".be-spotwrap", qi); if (!wrap || qi.dataset.x === "" || qi.dataset.x == null) return;
    const r = +($(".be-r", el)?.value || 8); let dot = $(".be-spot", wrap);
    if (!dot) { dot = document.createElement("i"); dot.className = "be-spot"; wrap.append(dot); }
    Object.assign(dot.style, { left: qi.dataset.x + "%", top: qi.dataset.y + "%", width: r * 2 + "%", height: r * 2 + "%" });
  };
  const uploadQimg = (el, file) => {
    if (!file) return;
    if (file.size > BE_LIMIT.image * 1048576) return toast(`Зураг ${BE_LIMIT.image} MB хүртэл`, true);
    const qi = $(".be-qimg", el), bar = $(".progress", qi); if (bar) bar.hidden = false;
    const x = new XMLHttpRequest(), fd = new FormData(); fd.append("file", file);
    x.open("POST", "/api/me/files?visibility=private"); x.setRequestHeader("Authorization", "Bearer " + Auth.token);
    x.upload.onprogress = (e) => { if (bar && e.lengthComputable) $("i", bar).style.width = Math.round(e.loaded / e.total * 100) + "%"; };
    x.onload = () => { let d = null; try { d = JSON.parse(x.responseText); } catch {} if (x.status >= 300) return toast(d?.error || "Хуулж чадсангүй", true);
      qi.dataset.url = d.path; qi.dataset.preview = d.url; qi.dataset.x = ""; qi.innerHTML = qimgHTML($(".be-qkind", el).value, d.url, null); library = null; };
    x.onerror = () => toast("Сүлжээний алдаа", true);
    x.send(fd);
  };
  // Excel/CSV-ээс асуултууд → засварлагчийн төгсгөлд нэмнэ (хадгалахаас өмнө шалгаж болно).
  const importXlsx = async (input) => {
    const file = input.files[0]; if (!file) return;
    try {
      const fd = new FormData(); fd.append("file", file);
      const d = await api("/api/me/quiz-import", { method: "POST", body: fd });
      d.blocks.forEach((b) => add("quiz", null, { quiz: b.quiz }));
      d.problems ||= []; toast(`${d.blocks.length} асуулт нэмэгдлээ${d.problems.length ? ` · ${d.problems.length} мөр алгассан` : ""}`);
      if (d.problems.length) alert("Дараах мөрүүдийг алгаслаа:\n\n" + d.problems.join("\n"));
    } catch (e) { toast(e.message, true); }
  };
  const exec = (cmd, rt) => {
    rt.focus();
    if (cmd === "mark") document.execCommand("hiliteColor", false, "#fff1a8");
    else if (cmd === "quote") document.execCommand("formatBlock", false, "blockquote");
    else if (cmd === "link") { const u = prompt("Холбоос (https://…)"); if (u && /^https?:\/\//.test(u.trim())) document.execCommand("createLink", false, u.trim()); else if (u) toast("Холбоос https:// -ээр эхэлнэ", true); }
    else document.execCommand(cmd);
  };
  box.addEventListener("mousedown", (e) => { if (e.target.closest("[data-cmd]")) e.preventDefault(); }); // сонголтыг алдахгүй
  box.addEventListener("click", (e) => {
    const t = e.target, el = t.closest(".be-block");
    const c = t.closest("[data-cmd]"); if (c && el) return exec(c.dataset.cmd, $(".be-rt", el));
    if (t.closest("[data-import]")) { const fi = $(".be-xlsx", box); fi.value = ""; fi.click(); return; }
    const ep = t.closest("[data-embed-preview]");
    if (ep) { const el = ep.closest(".be-block"), html = $(".be-embed", el).value.trim(), box = $(".be-embed-prev", el); box.innerHTML = html ? `<iframe sandbox="allow-scripts allow-popups allow-forms allow-presentation" allowfullscreen style="width:100%;height:${+$(".be-embed-h", el).value || 420}px;border:1px solid var(--line);border-radius:12px" srcdoc="${esc(SG.embedDoc(html))}"></iframe>` : ""; return; }
    const a = t.closest("[data-add]"); if (a) { const nb = add(a.dataset.add); if (["image", "audio", "video", "file"].includes(a.dataset.add) && a.dataset.add !== "video") pick(nb); return; }
    if (!el) return;
    if (t.closest("[data-be-del]")) { if ((el.dataset.type === "text" && $(".be-rt", el).textContent.trim()) || el.dataset.type === "quiz") { if (!confirm("Энэ хэсгийг устгах уу?")) return; } el.remove(); return; }
    if (t.closest("[data-pick], [data-replace]")) return pick(el);
    if (t.closest("[data-rec]")) return record(el, t.closest("[data-rec]"));
    if (t.closest("[data-vurl]")) { const u = prompt("YouTube эсвэл Vimeo холбоос"); if (u && /^https?:\/\//.test(u.trim())) setMedia(el, { path: u.trim(), url: u.trim(), original_name: "Видео холбоос" }); return; }
    if (t.closest("[data-opt-add]")) { const opts = $(".be-opts", el); if (opts.children.length >= 8) return toast("Дээд тал нь 8 хариулт"); opts.insertAdjacentHTML("beforeend", beOptHTML(el.dataset.id, $(".be-qkind", el).value === "multi", "", false)); opts.lastElementChild.querySelector(".be-opt-t").focus(); return; }
    if (t.closest("[data-opt-del]")) { const opts = $(".be-opts", el); if (opts.children.length <= 2) return toast("Дор хаяж 2 хариулт"); t.closest(".be-opt").remove(); }
    if (t.closest("[data-ans-add]")) { const a = $(".be-answers", el); if (a.children.length >= 10) return toast("Дээд тал нь 10"); a.insertAdjacentHTML("beforeend", `<div class="be-opt"><span class="be-ok">✓</span><input class="be-ans" maxlength="200" placeholder="Зөвд тооцох хариулт"><button type="button" class="icon-btn" data-row-del aria-label="Хасах">${ico("x", 14)}</button></div>`); a.lastElementChild.querySelector("input").focus(); return; }
    if (t.closest("[data-pair-add]")) { const a = $(".be-pairs", el); if (a.children.length >= 10) return toast("Дээд тал нь 10 хос"); a.insertAdjacentHTML("beforeend", pairHTML("", "")); a.lastElementChild.querySelector("input").focus(); return; }
    if (t.closest("[data-row-del]")) { const row = t.closest(".be-opt"), min = row.parentElement.classList.contains("be-pairs") ? 2 : 1; if (row.parentElement.children.length <= min) return toast("Үүнээс цөөн байж болохгүй"); row.remove(); return; }
    if (t.closest("[data-qimg]")) { let fi = $(".be-qfile", el); if (!fi) { el.insertAdjacentHTML("beforeend", `<input type="file" class="be-qfile" accept="image/*" hidden>`); fi = $(".be-qfile", el); } fi.value = ""; fi.click(); return; }
    if (t.closest("[data-qimg-del]")) { const qi = $(".be-qimg", el); qi.dataset.url = ""; qi.dataset.preview = ""; qi.dataset.x = ""; qi.innerHTML = qimgHTML($(".be-qkind", el).value, "", null); return; }
    const pickImg = t.closest(".be-spotwrap.pick");
    if (pickImg) { // зөв хэсгийг заах
      const r = pickImg.getBoundingClientRect(), qi = $(".be-qimg", el);
      qi.dataset.x = Math.round((e.clientX - r.left) / r.width * 1000) / 10; qi.dataset.y = Math.round((e.clientY - r.top) / r.height * 1000) / 10;
      return drawSpot(el);
    }
  });
  // Асуултын төрөл солиход одоо бичсэнээ хадгалаад шинэ төрлийн маягт гаргана.
  box.addEventListener("change", async (e) => {
    const el = e.target.closest(".be-block");
    if (e.target.classList.contains("be-qkind") && el) {
      const q = quizOf(el), kind = e.target.value;
      $(".be-qbody", el).innerHTML = quizBodyHTML(kind, el.dataset.id, q);
      const qi = $(".be-qimg", el); qi.innerHTML = qimgHTML(kind, qi.dataset.preview || (qi.dataset.url ? qi.dataset.url : ""), q.spot);
      return;
    }
    if (e.target.classList.contains("be-r") && el) return drawSpot(el);
    if (e.target.classList.contains("be-xlsx")) return importXlsx(e.target);
    if (e.target.classList.contains("be-qfile") && el) return uploadQimg(el, e.target.files[0]);
  });
  box.addEventListener("input", (e) => { const el = e.target.closest(".be-block"); if (e.target.classList.contains("be-r") && el) drawSpot(el); });
  box.addEventListener("keydown", (e) => {
    if (e.target.closest("[contenteditable]") && (e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") { e.preventDefault(); exec("link", e.target.closest("[contenteditable]")); }
    if (e.target.closest(".be-h, .be-q, .be-opt-t, .be-cap, .be-ex") && e.key === "Enter") e.preventDefault(); // маягт илгээгдэхгүй
    const h = e.target.closest("[data-be-drag]"); if (!h || (e.key !== "ArrowUp" && e.key !== "ArrowDown")) return;
    e.preventDefault(); const el = h.closest(".be-block");
    if (e.key === "ArrowUp" && el.previousElementSibling) el.previousElementSibling.before(el);
    if (e.key === "ArrowDown" && el.nextElementSibling) el.nextElementSibling.after(el);
    h.focus();
  });
  // Буулгах (paste): зураг шууд зураг блок болно, бусад нь энгийн текст.
  box.addEventListener("paste", (e) => {
    const rt = e.target.closest?.("[contenteditable]"); if (!rt) return;
    const img = [...(e.clipboardData?.files || [])].find((f) => /^image\//.test(f.type));
    e.preventDefault();
    if (img) { upload(add("image", rt.closest(".be-block")), img); return; }
    document.execCommand("insertText", false, e.clipboardData.getData("text/plain"));
  });
  // Хэсгүүдийг чирж зөөх (хулгана, хуруу).
  let drag = null;
  box.addEventListener("pointerdown", (e) => {
    const h = e.target.closest("[data-be-drag]"); if (!h || e.button > 0) return;
    e.preventDefault(); const el = h.closest(".be-block"), r = el.getBoundingClientRect();
    const ghost = el.cloneNode(true); ghost.classList.add("ol-ghost", "be-ghost"); Object.assign(ghost.style, { width: r.width + "px", left: r.left + "px", top: r.top + "px", maxHeight: "120px", overflow: "hidden" });
    document.body.append(ghost); el.classList.add("ol-dragging"); drag = { el, ghost, dy: Math.min(e.clientY - r.top, 60) };
    const move = (ev) => {
      ghost.style.top = ev.clientY - drag.dy + "px";
      if (ev.clientY < 90) scrollBy(0, -14); else if (ev.clientY > innerHeight - 60) scrollBy(0, 14);
      const sibs = $$(":scope > .be-block:not(.ol-dragging)", list), before = sibs.find((x) => { const b = x.getBoundingClientRect(); return ev.clientY < b.top + b.height / 2; });
      list.insertBefore(el, before || null);
    };
    const up = () => { removeEventListener("pointermove", move); removeEventListener("pointerup", up); removeEventListener("pointercancel", up); ghost.remove(); el.classList.remove("ol-dragging"); drag = null; };
    addEventListener("pointermove", move); addEventListener("pointerup", up); addEventListener("pointercancel", up);
  });
  // Компьютерээс файл чирж оруулах: байрлалд нь төрлөөр нь блок үүснэ.
  box.addEventListener("dragover", (e) => { if ([...(e.dataTransfer?.types || [])].includes("Files")) { e.preventDefault(); box.classList.add("be-dropping"); } });
  box.addEventListener("dragleave", (e) => { if (!box.contains(e.relatedTarget)) box.classList.remove("be-dropping"); });
  box.addEventListener("drop", (e) => {
    const fl = [...(e.dataTransfer?.files || [])]; if (!fl.length) return;
    e.preventDefault(); box.classList.remove("be-dropping");
    const onBlock = e.target.closest(".be-block"), emptyMedia = onBlock && $(".be-drop", onBlock) && fl.length === 1;
    if (emptyMedia) return upload(onBlock, fl[0]);
    let after = onBlock || null;
    for (const file of fl) { const el = add(typeOfFile(file), after); upload(el, file); after = el; }
  });

  f._be = {
    collect() {
      return $$(":scope > .be-block", list).map((el) => {
        const id = el.dataset.id, type = el.dataset.type;
        if (type === "text") { const text = domToMd($(".be-rt", el)); return text ? { id, type, text } : null; }
        if (type === "heading") { const text = $(".be-h", el).value.trim(); return text ? { id, type, text } : null; }
        if (type === "embed") { const text = $(".be-embed", el).value.trim(); return text ? { id, type, text, height: +$(".be-embed-h", el).value || 420 } : null; }
        if (type === "quiz") {
          const quiz = quizOf(el);
          if (!quiz.question && !quiz.image && [...(quiz.options || []), ...(quiz.answers || []), ...(quiz.left || [])].every((o) => !o.trim())) return null;
          return { id, type, quiz };
        }
        const m = $(".be-media", el); if (!m.dataset.url) return null;
        return { id, type, url: m.dataset.url, name: m.dataset.name, size: +m.dataset.size || 0, text: $(".be-cap", el).value.trim(), download: !!$(".be-dl input", el)?.checked };
      }).filter(Boolean);
    },
    uploading: () => $$(".be-drop .progress:not([hidden])", box).length > 0,
  };
}
// Блокуудаас товч текст (нийтлэлийн жагсаалт, хайлтад).
const blocksSummary = (bs) => bs.filter((b) => b.type === "text" || b.type === "heading").map((b) => b.text.replace(/\*\*|__|==|^[>-]\s|\[([^\]]*)\]\([^)]*\)/gm, "$1")).join("\n").slice(0, 1500);

async function courseEditor(id) {
  let [{ course: c, lessons, comments: commentCounts = {} }, lib, roster] = await Promise.all([api(`/api/me/courses/${id}`), loadLibrary(), api(`/api/me/courses/${id}/students`).catch(() => [])]);
  const stripSig = (u) => (u && u.startsWith("/files/") ? u.split("?")[0] : u || ""); // гарын үсэгтэй URL-аас query-г хасна
  const fileName = (u) => { const f = lib.files.find((x) => x.path === stripSig(u)); return f ? f.original_name : decodeURIComponent(stripSig(u).split("/").pop() || ""); };
  const mediaLabel = (u) => (!u ? "" : /youtu\.?be|vimeo/.test(u) ? "Видео холбоос" : u.startsWith("/files/") ? fileName(u) : u);
  const isImage = (u) => /\.(webp|jpe?g|png|gif)$/i.test(stripSig(u));
  const dueBadge = (due) => !due ? "" : `${due.start_at ? `<span class="aud">▶ ${fmtDate(due.start_at)}-с</span>` : ""}${due.at ? `<span class="aud">📅 ${fmtDate(due.at)} хүртэл${due.late === "paid" ? ` · хоцорвол ${money(due.late_fee)}` : due.late === "closed" ? " · дараа нь хаалттай" : ""}</span>` : ""}${due.fee ? `<span class="aud aud-paid">💳 ${money(due.fee)}</span>` : ""}`;
  const kindBadge = (l) => (l.exam ? `<span class="kind kind-exam">📝 Шалгалт</span>${dueBadge(l.exam.due)}` : l.assignment ? `<span class="kind kind-asg">📎 Даалгавар</span>${dueBadge(l.assignment.due)}` : "") + (l.discussion && !l.exam ? `<span class="aud" title="Хэлэлцүүлэгтэй">💬${commentCounts[l.id] ? " " + commentCounts[l.id] : ""}</span>` : "");
  const audience = (l) => l.is_free ? `<span class="aud aud-free">${ico("globe", 14)}Үнэгүй · бүгдэд нээлттэй</span>`
    : `<span class="aud aud-paid">${ico("lock", 14)}${l.price ? money(l.price) : "Зөвхөн багцаар"}</span>`;
  const libOptions = () => lib.files.filter((f) => f.status !== "failed").map((f) => `<option value="${esc(f.path)}">${esc(f.original_name)} · ${fmtSize(f.size)}${f.status === "processing" ? " (боловсруулж байна)" : ""}</option>`).join("");

  // Бүлгүүд (модуль). Хоосон (хичээлгүй) бүлэг зөвхөн клиент дээр байна: хичээл чирж оруулмагц хадгалагдана.
  let pending = []; // [{name, at}] — хоосон бүлэг ба хөтөлбөр дээрх байр
  let view = (() => { try { return localStorage.getItem("sg_course_view") || "outline"; } catch { return "outline"; } })();
  const groupsModel = () => {
    const out = [];
    for (const l of [...lessons].sort((a, b) => a.position - b.position)) {
      const k = l.section || ""; let g = out.find((x) => x.name === k);
      if (!g) out.push((g = { name: k, items: [] })); g.items.push(l);
    }
    pending = pending.filter((p) => p.name && !out.some((g) => g.name === p.name));
    for (const p of [...pending].sort((a, b) => a.at - b.at)) out.splice(Math.min(p.at, out.length), 0, { name: p.name, items: [] });
    return out;
  };
  const sections = () => groupsModel().map((g) => g.name).filter(Boolean);
  const sectionPicker = (cur) => `<div class="pe-section"><span class="muted small">${ico("files", 15)}Бүлэг:</span>
      <select name="section_pick" aria-label="Хичээлийн бүлэг"><option value="">— Бүлэггүй —</option>${sections().map((n) => `<option value="${esc(n)}" ${n === cur ? "selected" : ""}>${esc(n)}</option>`).join("")}<option value="__new">+ Шинэ бүлэг үүсгэх…</option></select>
      <input name="section_new" maxlength="80" placeholder="Шинэ бүлгийн нэр (ж: 1-р бүлэг. Алгебр)" hidden></div>`;
  // Нийтлэл бичих/засах маягт (шинэ хичээл ба засварт ижил). preset — бүлгийн толгойноос "+ Хичээл нэмэх" дарахад.
  const toLocal = (iso) => { if (!iso) return ""; const d = new Date(iso); if (isNaN(d)) return ""; const p = (n) => String(n).padStart(2, "0"); return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`; };
  // Хугацаа ба төлбөр (шалгалт, даалгаварт ижил): эхлэх/дуусах цаг, хоцорсон тохиолдол, оролцооны төлбөр.
  const dueHTML = (pfx, due, what) => `<div class="pe-grid pe-grid-2">
        <label>Эхлэх цаг<input name="${pfx}_start" type="datetime-local" value="${toLocal(due?.start_at)}"><small>Хоосон бол шууд нээлттэй</small></label>
        <label>Дуусах цаг<input name="${pfx}_due" type="datetime-local" value="${toLocal(due?.at)}"><small>Хоосон бол хугацаагүй</small></label></div>
      <div class="pe-grid pe-grid-2">
        <label>Дуусах цаг өнгөрсөн бол<select name="${pfx}_late"><option value="free" ${!due?.late || due.late === "free" ? "selected" : ""}>Төлбөргүй үргэлжлүүлнэ</option><option value="paid" ${due?.late === "paid" ? "selected" : ""}>Хоцролтын төлбөр төлж нээнэ</option><option value="closed" ${due?.late === "closed" ? "selected" : ""}>Хаалттай болно</option></select></label>
        <label class="pe-fee" ${due?.late === "paid" ? "" : "hidden"}>Хоцролтын төлбөр (₮)<input name="${pfx}_fee" type="number" min="0" step="500" value="${due?.late_fee || 5000}"></label></div>
      <label class="pe-entry">${what} төлбөр (₮)<input name="${pfx}_entry" type="number" min="0" step="500" value="${due?.fee || 0}"><small>0 бол үнэгүй. Тавьсан бол суралцагч сургалтад элссэн ч төлж байж ${what === "Шалгалтын" ? "шалгалт өгнө" : "хариу илгээнэ"}.</small></label>`;
  const ruleBadge = (l) => { const r = UNLOCK_RULES.find(([v]) => v === (l.unlock_rule ?? "")) || UNLOCK_RULES[0]; return `<span class="aud" title="${esc(r[3])}">${r[1]} ${esc(r[2])}${["view", "complete", ""].includes(l.unlock_rule ?? "") && l.unlock_after_h ? " · " + window.SG_humanHours(l.unlock_after_h) : ""}</span>`; };
  const kindOf = (l) => l?.exam ? "exam" : l?.assignment ? "assignment" : "lesson";
  // Энэ хичээл хэзээ нээгдэх вэ (дараалалтай сургалтад): багш гараар сонгоно.
  const UNLOCK_RULES = [
    ["view", "⏱", "Цагаар", "Өмнөх хичээлийг үзсэнээс хойш тохируулсан хугацааны дараа"],
    ["quiz", "🧩", "Асуултад бүрэн зөв хариулсан бол", "Өмнөх хичээлийн бүх асуултад зөв хариулмагц шууд"],
    ["active", "🕒", "Хугацааг бүрэн судалсан бол", "Өмнөх хичээлийн идэвхтэй минутыг бүрэн гүйцээмэгц шууд"],
    ["quiz_active", "✅", "Асуулт + хугацаа хоёулаа", "Асуултууд бүгд зөв БА минут бүрэн гүйцсэн үед"],
    ["complete", "☑️", "«Дууслаа» дармагц", "Өмнөхийг дууссан гэж тэмдэглэснээс хойш (хугацаа нэмж болно)"],
    ["exam", "📝", "Шалгалтад тэнцсэн бол", "Өмнөх хичээл шалгалт бол тэнцмэгц шууд"],
    ["manual", "🔐", "Багш гараар нээнэ", "Та «Шууд нээлттэй болгох» дартал хаалттай"],
    ["", "✨", "Автомат", "Минут → асуулт/шалгалт → эс бөгөөс хугацаа (систем өөрөө)"],
  ];
  const unlockHTML = (l) => { const cur = l?.unlock_rule ?? "view", h = l?.unlock_after_h || 0; return `<div class="pe-unlock"><b>Энэ хичээл хэзээ нээгдэх вэ?</b>
      <div class="ur-grid">${UNLOCK_RULES.map(([v, ic, t, d]) => `<label class="ur ${cur === v ? "on" : ""}"><input type="radio" name="unlock_rule" value="${v}" ${cur === v ? "checked" : ""}><span class="ur-ic">${ic}</span><span><strong>${t}</strong><small>${d}</small></span></label>`).join("")}</div>
      <div class="pe-grid pe-grid-2 ur-time" ${["view", "complete", ""].includes(cur) ? "" : "hidden"}><label>Хүлээх хугацаа<select name="unlock_after_h">${UNLOCKS.map(([hv, t]) => `<option value="${hv}" ${h === hv ? "selected" : ""}>${t}</option>`).join("")}${l && !UNLOCKS.some(([hv]) => hv === h) ? `<option value="${h}" selected>${h} цаг</option>` : ""}</select></label>
        <label class="check" style="align-self:end"><input type="checkbox" name="always_open" ${l?.always_open ? "checked" : ""}> Дарааллаас үл хамааран нээлттэй</label></div></div>`; };
  const KIND_TXT = {
    lesson: { title: "Хичээлийн гарчиг", content: "Агуулга", hint: "Текст, зураг, видео, файл, асуулт — дарааллаар нь нэмнэ. Чирж зөөж болно." },
    exam: { title: "Шалгалтын нэр", content: "Шалгалтын асуултууд", hint: "«Асуулт» товчоор асуултуудаа нэмнэ. Тайлбар текст нэмж болно. Суралцагчид асуултууд нэг дор өгөгдөж, өөр цонх руу шилжих эсвэл хуулах үед шалгалт шууд хаагдана." },
    assignment: { title: "Даалгаврын нэр", content: "Даалгаврын нөхцөл", hint: "Юу хийх, юуг хэрхэн илгээхийг энд бичнэ. Суралцагч хариугаа текст ба холбоосоор илгээж, та оноо, тайлбар өгнө." },
  };
  const step = (n, title, body, cls = "", hidden = false) => `<section class="pe-step ${cls}" ${hidden ? "hidden" : ""}><span class="pe-num" aria-hidden="true">${n}</span><div class="pe-step-body"><h4>${title}</h4>${body}</div></section>`;
  // Хичээл оруулах маягт: 1 Төрөл → 2 Нэр, бүлэг → 3 Агуулга → 4 Тохиргоо (төрлөөс хамаарна) → Нэмэлт → Нийтлэх.
  const editorHTML = (l, preset, kind) => { kind = kind || kindOf(l); const T = KIND_TXT[kind]; return `<form class="form post-editor pe-v2" ${l ? `data-edit="${esc(l.id)}"` : `id="composerForm"`} data-kind="${kind}">
      ${step(1, "Төрөл", `<div class="seg pe-kind" role="radiogroup" aria-label="Төрөл">
        <label><input type="radio" name="kind" value="lesson" ${kind === "lesson" ? "checked" : ""}><span>📄 Хичээл</span></label>
        <label><input type="radio" name="kind" value="exam" ${kind === "exam" ? "checked" : ""}><span>📝 Шалгалт</span></label>
        <label><input type="radio" name="kind" value="assignment" ${kind === "assignment" ? "checked" : ""}><span>📎 Даалгавар</span></label></div>
        <p class="pe-kind-hint muted small" data-k="lesson" ${kind === "lesson" ? "" : "hidden"}>Ердийн хичээл: үзэж, асуултад хариулж, дуусгана.</p>
        <p class="pe-kind-hint muted small" data-k="exam" ${kind === "exam" ? "" : "hidden"}>Хугацаатай, хамгаалалттай шалгалт. Оноогоор тэнцэнэ.</p>
        <p class="pe-kind-hint muted small" data-k="assignment" ${kind === "assignment" ? "" : "hidden"}>Суралцагч хариу илгээж, та дүгнэнэ.</p>`)}
      ${step(2, "Нэр ба бүлэг", `<input name="title" required maxlength="200" class="pe-title" placeholder="${T.title}" value="${esc(l?.title || "")}">
        ${sectionPicker(l ? l.section || "" : preset || "")}`)}
      ${step(3, `<span data-content-title>${T.content}</span>`, `<p class="muted small pe-content-hint" data-content-hint>${T.hint}</p><div class="be" data-be></div>`)}
      ${step(4, "Шалгалтын тохиргоо", `<div class="pe-grid pe-grid-3">
          <label>Хугацаа (минут)<input name="ex_time" type="number" min="0" max="600" value="${l?.exam?.time_min ?? 30}"><small>0 бол хязгааргүй</small></label>
          <label>Оролдлого<input name="ex_attempts" type="number" min="0" max="100" value="${l?.exam?.attempts ?? 1}"><small>0 бол хязгааргүй</small></label>
          <label>Тэнцэх хувь<input name="ex_pass" type="number" min="0" max="100" value="${l?.exam?.pass_pct ?? 60}"></label></div>
        <div class="pe-checks"><label class="check"><input type="checkbox" name="ex_shuffle" ${l?.exam?.shuffle ? "checked" : ""}> Асуултыг холих</label>
          <label class="check"><input type="checkbox" name="ex_show" ${l?.exam ? (l.exam.show_answers ? "checked" : "") : "checked"}> Дууссаны дараа зөв хариултыг харуулах</label></div>
        <h5>Хугацаа ба төлбөр</h5>${dueHTML("ex", l?.exam?.due, "Шалгалтын")}`, "pe-exam", kind !== "exam")}
      ${step(4, "Даалгаврын тохиргоо", `<h5>Үнэлэх арга</h5>
        <div class="gr-modes" role="radiogroup" aria-label="Үнэлэх арга">
          <label class="gr-mode ${l?.assignment?.grading === "rubric" ? "" : "on"}"><input type="radio" name="asg_grading" value="" ${l?.assignment?.grading === "rubric" ? "" : "checked"}><span class="gr-ic" aria-hidden="true">123</span><span><strong>Тоогоор</strong><small>0-ээс дээд оноо хүртэл нэг тоо өгнө</small></span></label>
          <label class="gr-mode ${l?.assignment?.grading === "rubric" ? "on" : ""}"><input type="radio" name="asg_grading" value="rubric" ${l?.assignment?.grading === "rubric" ? "checked" : ""}><span class="gr-ic" aria-hidden="true">▦</span><span><strong>Рубрикаар</strong><small>Шалгуур × түвшин хүснэгт — оноо өөрөө бодогдоно</small></span></label>
        </div>
        <div class="pe-grid pe-grid-3 gr-score" ${l?.assignment?.grading === "rubric" ? "hidden" : ""}><label>Дээд оноо<input name="asg_max" type="number" min="1" max="1000" value="${l?.assignment?.max_score || 100}"></label></div>
        <div class="gr-rubric" ${l?.assignment?.grading === "rubric" ? "" : "hidden"} data-rubric="${esc(JSON.stringify(l?.assignment?.rubric || null))}"></div>
        <h5>Хугацаа ба төлбөр</h5>${dueHTML("asg", l?.assignment?.due, "Даалгаврын")}
        <p class="muted small" style="margin:8px 0 0">Хариу: текст мэдээлэл ба холбоос (Google Docs, видео, GitHub г.м). Файл илгээхгүй.</p>`, "pe-asg", kind !== "assignment")}
      <details class="pe-step pe-more" ${l?.active_min || l?.format || l?.mode || c.drip ? "open" : ""}><summary><span class="pe-num" aria-hidden="true">${ico("gear", 14)}</span><span class="pe-step-body"><h4>Нэмэлт тохиргоо <small class="muted">хэлбэр, идэвхтэй хугацаа${c.drip ? ", хэзээ нээгдэх" : ""}</small></h4></span></summary>
        <div class="pe-more-body">
          <div class="pe-grid pe-grid-3">
            <label>Сургалтын хэлбэр<select name="format">${FORMATS.map(([v, t]) => `<option value="${v}" ${(l?.format || "") === v ? "selected" : ""}>${t}</option>`).join("")}</select></label>
            <label>Заах аргын төрөл<select name="mode_kind">${MODES.map(([v, t]) => `<option value="${v}" ${(l?.mode || "") === v ? "selected" : ""}>${t}</option>`).join("")}</select></label>
            <label>Идэвхтэй суралцах хугацаа (мин)<input name="active_min" type="number" min="0" max="600" value="${l?.active_min || 0}"><small>0 бол шаардахгүй. Тавьсан бол энэ хугацаанд идэвхтэй үзэж байж «дууслаа» дарна.</small></label></div>
          <label class="check" style="margin-top:10px"><input type="checkbox" name="discussion" ${l ? (l.discussion ? "checked" : "") : "checked"}> 💬 Хэлэлцүүлэгтэй — хичээлийн доор суралцагчид лайк дарж, сэтгэгдэл, асуулт бичиж, хоорондоо ярилцана (та ч хариулна)</label>
          ${c.drip ? unlockHTML(l) : ""}
        </div></details>
      <div class="pe-foot">
        <div class="pe-access"><span class="muted small">Хэн үзэх вэ</span>
          <div class="seg" role="radiogroup" aria-label="Хэн үзэх вэ">
            <label><input type="radio" name="mode" value="free" ${l?.is_free ? "checked" : ""}><span>${ico("globe", 15)}Үнэгүй</span></label>
            <label><input type="radio" name="mode" value="paid" ${l?.is_free ? "" : "checked"}><span>${ico("lock", 15)}Төлбөртэй</span></label></div>
          <label class="pe-price" ${l?.is_free ? "hidden" : ""}><input name="price" type="number" min="0" step="500" value="${l ? l.price || 0 : c.price ? 0 : 10000}" aria-label="Үнэ"><span>₮</span></label></div>
        <span class="grow"></span>
        ${l ? `<button type="button" class="btn btn-ghost btn-sm" data-cancel>Болих</button>` : `<label class="check small pe-hidden" title="Бэлтгэж дуусаагүй бол хаалттай үүсгээд, бэлэн болмогц хөтөлбөрөөс нээнэ"><input type="checkbox" name="hidden_new"> Хаалттай (бэлтгэж дуусаагүй)</label>`}
        <button class="btn btn-gold">${l ? "Хадгалах" : kind === "exam" ? "Шалгалт нийтлэх" : kind === "assignment" ? "Даалгавар нийтлэх" : "Хичээл нийтлэх"}</button></div>
      <p class="muted small pe-hint">${c.price ? "Үнэ 0 бол зөвхөн сургалтын багцаар нээгдэнэ." : "Энэ сургалт багц үнэгүй тул төлбөртэй хичээл бүр өөрийн үнэтэй байна."}</p>
      <p class="form-error" role="alert"></p></form>`; };

  // Блоктой нийтлэл: эхний зураг + агуулгын тоо + урьдчилан харах.
  const blocksPostHTML = (l) => {
    const n = (t) => l.blocks.filter((b) => b.type === t).length, img = l.blocks.find((b) => b.type === "image");
    const parts = [["text", "текст"], ["image", "зураг"], ["audio", "дуу"], ["video", "видео"], ["file", "файл"], ["quiz", "асуулт"]].filter(([t]) => n(t)).map(([t, w]) => `${n(t)} ${w}`).join(" · ");
    return `${img ? `<div class="post-media"><img src="${esc(img.url)}" alt="" loading="lazy"></div>` : ""}
      <button class="post-attach" data-open-media>${ico("book", 22)}<span><strong>Агуулгыг суралцагчийн нүдээр харах</strong><small class="muted">${parts}</small></span>${ico("chevron", 18)}</button><div class="post-media post-preview" hidden></div>`;
  };
  const postHTML = (l) => `<article class="post ${l.hidden ? "is-hidden" : ""}" data-lid="${esc(l.id)}">
      <header class="post-head">${avatar(me, "avatar-sm")}<div class="grow"><strong>${esc(me.display_name)}</strong>
        <small class="muted">Хичээл ${String(l.position).padStart(2, "0")} · ${fmtDate(l.created_at)} · ${audience(l)}${c.drip ? (l.always_open ? ` · <span class="aud">${ico("globe", 14)}Дарааллаас гадуур</span>` : l.position > 1 ? ` · <span class="aud">⏱ ${window.SG_humanHours(l.unlock_after_h)}</span>` : "") : ""}</small></div>
        <button class="icon-btn" data-edit-post aria-label="Хичээл засах" title="Засах">${ico("edit", 18)}</button></header>
      <div class="post-body">${l.hidden ? `<p class="post-hidden">${ico("lock", 15)}Хаалттай — бэлтгэж дуусаагүй, суралцагчдад харагдахгүй</p>` : ""}<h3>${esc(l.title)}</h3>${l.exam || l.assignment ? `<p style="margin:0 0 8px;display:flex;gap:6px;flex-wrap:wrap">${kindBadge(l)}</p>` : ""}${l.format || l.mode ? `<p style="margin:0 0 8px;display:flex;gap:6px;flex-wrap:wrap">${l.format ? `<span class="kind">${esc(kindName(l.format))}</span>` : ""}${l.mode ? `<span class="kind">${esc(kindName(l.mode))}</span>` : ""}</p>` : ""}${l.content ? `<p class="post-text">${SG.linkify(l.content)}</p>` : ""}</div>
      ${l.blocks?.length ? blocksPostHTML(l) : l.video_url ? (isImage(l.video_url) ? `<div class="post-media"><img src="${esc(l.video_url)}" alt="" loading="lazy"></div>`
        : `<button class="post-attach" data-open-media>${ico(/\.pdf$/i.test(stripSig(l.video_url)) ? "book" : "live", 22)}<span><strong>${esc(mediaLabel(l.video_url))}</strong><small class="muted">Дарж нээнэ</small></span>${ico("chevron", 18)}</button><div class="post-media" hidden></div>`) : ""}
      <footer class="post-foot"><button data-edit-post>${ico("edit", 16)}Засах</button><button data-toggle-hidden>${ico(l.hidden ? "eye" : "lock", 16)}${l.hidden ? "Суралцагчдад нээх" : "Хаах"}</button>${l.assignment ? `<button data-grade>${ico("users", 16)}Хариунуудыг дүгнэх</button>` : ""}<button data-toggle-free>${ico(l.is_free ? "lock" : "globe", 16)}${l.is_free ? "Төлбөртэй болгох" : "Үнэгүй болгох"}</button>
        ${c.drip ? `<button data-toggle-always title="Дарааллаас үл хамааран нээлттэй эсэх">${ico(l.always_open ? "lock" : "globe", 16)}${l.always_open ? "Дараалалд оруулах" : "Шууд нээлттэй болгох"}</button>` : ""}
        ${c.published ? `<a href="/c/${esc(c.id)}#l=${esc(l.id)}" target="_blank" rel="noopener">${ico("eye", 16)}Суралцагчийн нүдээр</a>` : ""}${l.discussion && !l.exam && c.published ? `<a href="/c/${esc(c.id)}#l=${esc(l.id)}" target="_blank" rel="noopener">${ico("chat", 16)}Хэлэлцүүлэг${commentCounts[l.id] ? ` (${commentCounts[l.id]})` : ""}</a>` : ""}
        <button data-del-post class="post-del" title="Хичээлийг файлуудтай нь хамт устгах">${ico("x", 16)}Устгах</button></footer></article>`;

  const free = () => lessons.filter((l) => l.is_free).length;
  // Бүлэгтэй бол хөтөлбөрийн дарааллаар бүлэг бүрийн доор; бүлэггүй бол нийтлэл шиг шинэ нь дээрээ.
  const feedHTML = () => {
    if (!lessons.length) return `<div class="empty">Одоогоор хичээл алга.<br>Дээрх хэсэгт анхны хичээлээ нийтлээрэй.</div>`;
    if (!sections().length) return [...lessons].sort((a, b) => b.position - a.position).map(postHTML).join("");
    const ordered = [...lessons].sort((a, b) => a.position - b.position), groups = [];
    for (const l of ordered) { const k = l.section || ""; let g = groups.find((x) => x.name === k); if (!g) groups.push((g = { name: k, items: [] })); g.items.push(l); }
    return groups.map((g, i) => `<section class="sec" data-sec="${esc(g.name)}">
        <header class="sec-head"><span class="lg-num">${String(i + 1).padStart(2, "0")}</span>
          <h2>${g.name ? esc(g.name) : "Бүлэггүй хичээлүүд"}<small>${g.items.length} хичээл · ${g.items.filter((l) => l.is_free).length} үнэгүй</small></h2>
          <button class="btn btn-gold btn-sm" data-sec-add>${ico("plus", 15)}Хичээл нэмэх</button></header>
        ${g.items.map(postHTML).join("")}</section>`).join("");
  };
  // Хөтөлбөр: бүлгийн нэрийг шууд бичнэ, хичээлийг чирж (эсвэл ↑↓ товчоор) бүлэг хооронд зөөнө.
  // Хичээлийг суралцагчдад нээх / хаах (бэлтгэж дуусаагүй) шилжүүлэгч.
  const visHTML = (l) => `<label class="ol-vis ${l.hidden ? "off" : ""}" title="${l.hidden ? "Хаалттай — суралцагчдад харагдахгүй. Бэлэн болмогц нээнэ үү." : "Нээлттэй — суралцагчдад харагдана. Бэлтгэж дуусаагүй бол хаана уу."}">
      <input type="checkbox" data-vis ${l.hidden ? "" : "checked"} aria-label="«${esc(l.title)}» хичээлийг суралцагчдад харуулах"><span class="switch" aria-hidden="true"></span><em>${l.hidden ? "Хаалттай" : "Нээлттэй"}</em></label>`;
  const rowHTML = (l, n) => `<li class="ol-row ${l.hidden ? "is-hidden" : ""}" data-lid="${esc(l.id)}">
      <button type="button" class="ol-handle" data-drag="row" aria-label="«${esc(l.title)}» хичээлийг зөөх (↑↓ товч)" title="Чирж зөөх">${ico("grip", 18)}</button>
      <span class="ol-num">${String(n).padStart(2, "0")}</span>
      <span class="ol-title"><strong>${esc(l.title)}</strong><small>${[l.format && kindName(l.format), l.mode && kindName(l.mode)].filter(Boolean).map((k) => `<span class="kind">${esc(k)}</span>`).join("")}${audience(l)}${c.drip && !l.always_open && n > 1 ? ruleBadge(l) : ""}${kindBadge(l)}${l.active_min ? `<span class="aud">🕒 ${l.active_min} мин идэвхтэй</span>` : ""}</small></span>
      <span class="ol-acts">${visHTML(l)}
      <button type="button" class="icon-btn" data-edit-post aria-label="Засах" title="Засах">${ico("edit", 17)}</button>
      <button type="button" class="icon-btn ol-del" data-del-post aria-label="Устгах" title="Устгах">${ico("x", 17)}</button></span></li>`;
  const outlineHTML = () => {
    const gs = groupsModel(); let n = 0;
    return `<div class="ol" id="olSecs">${gs.map((g, i) => `<section class="ol-sec" data-sec="${esc(g.name)}">
        <header class="ol-sec-head"><button type="button" class="ol-handle" data-drag="sec" aria-label="Бүлгийг зөөх" title="Бүлгийг чирж зөөх">${ico("grip", 18)}</button>
          <span class="lg-num">${String(i + 1).padStart(2, "0")}</span>
          <input class="ol-sec-name" value="${esc(g.name)}" maxlength="80" placeholder="${g.name || !g.items.length ? "Бүлгийн нэрээ бичнэ үү…" : "Бүлэггүй хичээлүүд — нэр бичвэл бүлэг болно"}" aria-label="Бүлгийн нэр">
          <small class="muted">${g.items.length} хичээл${g.items.some((l) => l.hidden) ? ` · <b class="ol-hidden-n">${g.items.filter((l) => l.hidden).length} хаалттай</b>` : ""}</small>
          <button type="button" class="btn btn-glass btn-sm" data-sec-add>${ico("plus", 15)}Хичээл</button>
          ${g.name ? `<button type="button" class="icon-btn" data-sec-del aria-label="Бүлгийг задлах" title="Бүлгийг задлах (хичээлүүд үлдэнэ)">${ico("x", 16)}</button>` : ""}</header>
        <ol class="ol-list">${g.items.map((l) => rowHTML(l, ++n)).join("")}</ol></section>`).join("")}</div>
      <button type="button" class="ol-add" id="olAddSec">${ico("plus", 18)}Бүлэг нэмэх</button>
      <p class="muted small ol-tip">${ico("grip", 14)} Хичээлийг бариад чирж өөр бүлэг рүү зөөнө. Гараар: зөөх товч дээр ↑ ↓.</p>`;
  };
  const render = () => {
    main.innerHTML = `<section class="gp-head" style="--h:${hueOfName(c.title)}">
        <div class="gp-cover"><a class="gp-back" href="#courses" aria-label="Сургалтууд руу буцах">${ico("back", 18)}Сургалтууд</a><b>${esc(c.title.trim()[0] || "?")}</b></div>
        <div class="gp-bar">
          <div class="grow gp-info"><h1>${esc(c.title)}</h1>
            <p class="gp-desc">${c.description ? esc(c.description) : `<span class="muted">Тайлбар нэмээгүй байна — «Тохиргоо»-оос нэмнэ.</span>`}</p>
          </div>
          <div class="gp-hside">
            <button type="button" class="gp-roster" id="gpRoster" aria-label="Суралцагчид (${roster.length})">
              <span class="gp-avs">${roster.slice(0, 5).map((r) => avatar(r.user, "avatar-sm")).join("") || `<span class="gp-av0">${ico("users", 16)}</span>`}</span>
              <span class="gp-rn"><b>${roster.length}</b> суралцагч</span>${ico("chevron", 15)}</button>
            <div class="gp-acts">
              ${c.published ? `<a class="btn btn-glass btn-sm" href="/c/${esc(c.id)}" target="_blank" rel="noopener">${ico("ext", 16)}Харах</a>` : `<button class="btn btn-gold btn-sm" id="gpPublish">${ico("globe", 16)}Нийтлэх</button>`}
              <button class="btn btn-glass btn-sm" id="gpChat" title="Элссэн бүх суралцагчтай нэг дор ярилцана">${ico("chat", 16)}Бүлэг чат</button>
              <a class="btn btn-glass btn-sm" href="#live" title="Google Meet-ээр шууд хичээл товлох">${ico("live", 16)}Шууд хичээл</a>
              <button class="btn btn-glass btn-sm" id="gpSettings">${ico("gear", 16)}Тохиргоо</button>
            </div></div>
          <ul class="gp-facts">
              <li class="${c.published ? "ok" : "draft"}">${ico(c.published ? "globe" : "lock", 15)}${c.published ? "Нийтлэгдсэн · профайл дээр харагдана" : "Ноорог — зөвхөн танд харагдана"}</li>
              <li>${ico("money", 15)}${c.price ? `Багц <b>${money(c.price)}</b>` : "Хичээл тус бүрээр зарна"}</li>
              <li>${ico("clock", 15)}${c.drip ? `<b>Дарааллаар</b> нээгдэнэ${c.unlock_all_paid && c.price ? " · багц төлбөл бүгд шууд" : ""}` : "Бүх хичээл нэг дор нээлттэй"}</li>
              <li>${ico("book", 15)}<b>${lessons.length}</b> хичээл · <b>${free()}</b> үнэгүй</li>
              <li>${ico("eye", 15)}<b>${c.views || 0}</b> үзэлт</li>
              <li>${ico("cal", 15)}${fmtDate(c.created_at)} үүсгэсэн</li>
            </ul>
        </div></section>
      <div class="gp-body gp-full">
        <div class="gp-feed">
          <section class="card composer-box" id="composer">
            <button class="composer" id="composerOpen">${avatar(me, "avatar-sm")}<span class="composer-input">Шинэ хичээл нийтлэх…</span></button>

            <div id="composerBody" hidden>${editorHTML(null)}</div></section>
          ${lessons.length || pending.length ? `<div class="view-tabs" role="tablist">
            <button role="tab" data-view="outline" aria-selected="${view === "outline"}">${ico("list", 16)}Хөтөлбөр</button>
            <button role="tab" data-view="posts" aria-selected="${view === "posts"}">${ico("book", 16)}Нийтлэлүүд</button>
            <button role="tab" data-view="journal" aria-selected="${view === "journal"}">${ico("users", 16)}Журнал</button></div>` : ""}
          <div id="gpView">${view === "journal" && lessons.length ? `<div class="jr" id="journal"><div class="loader"></div></div>` : view === "outline" && (lessons.length || pending.length) ? outlineHTML() : feedHTML()}</div>
        </div></div>
      <div class="modal" id="gpRosterModal"><div class="modal-card" style="width:min(580px,100%)"><button class="icon-btn modal-x" data-close aria-label="Хаах">${ico("x", 18)}</button>
        <h3 class="h3">Суралцагчид <span class="chip">${roster.length}</span></h3>
        <div class="items">${roster.map(studentRow).join("") || `<p class="muted small" style="margin:0">Хараахан хэн ч элсээгүй байна.</p>`}</div>
        ${roster.length ? `<a class="link" href="#students" style="display:inline-flex;margin-top:12px">${ico("chart", 15)}Хяналт ба статистик ${ico("chevron", 14)}</a>` : ""}</div></div>
      <div class="modal" id="gpModal"><div class="modal-card"><button class="icon-btn modal-x" data-close aria-label="Хаах">${ico("x", 18)}</button>
        <h3 class="h3">Сургалтын тохиргоо</h3><form class="form" id="gpForm">${courseFormHTML(c)}<p class="form-error" role="alert"></p>
        <div class="hero-cta" style="margin:0;justify-content:flex-end"><button type="button" class="btn btn-ghost" data-close>Болих</button><button class="btn btn-gold">Хадгалах</button></div></form></div></div>`;
    hydrateBooks(main);
    const cf = $("#composerForm"); if (cf) mountBlockEditor(cf, lessonBlocks(null));
    if (view === "journal" && $("#journal")) loadJournal();
  };

  /* ---------- Журнал: суралцагч × хичээл (сургуулийн журнал шиг) — нүд, нэр дээр дарахад дэлгэрэнгүй ---------- */
  let jr = null;
  const dur2 = (s) => !s ? "0" : s < 60 ? s + " сек" : s < 3600 ? Math.round(s / 60) + " мин" : Math.floor(s / 3600) + " ц " + Math.round((s % 3600) / 60) + " мин";
  const jrCell = (l, x) => { // нүдний текст, ангилал
    if (!x) return ["", ""];
    if (l.kind === "exam" && x.exam) return [x.exam.best >= 0 ? x.exam.best + "%" : "—", x.exam.passed ? "ok" : "bad"];
    if (l.kind === "assignment" && x.asg) return [x.asg.score != null ? String(x.asg.score) : "✓", x.asg.score != null ? (x.asg.score >= x.asg.max * 0.6 ? "ok" : "bad") : "wait"];
    if (x.disq) return ["⛔", "disq"];
    if (x.viewed) return [String(x.pts), x.done ? "ok" : "part"];
    return ["", ""];
  };
  const loadJournal = async () => {
    const box = $("#journal"); if (!box) return;
    try { jr = await api(`/api/me/courses/${c.id}/journal`); } catch (e) { box.innerHTML = `<p class="form-error">${esc(e.message)}</p>`; return; }
    drawJournal();
  };
  const drawJournal = (q = "") => {
    const box = $("#journal"); if (!box || !jr) return;
    const L = jr.lessons, S = jr.students.filter((s) => !q || s.name.toLowerCase().includes(q.toLowerCase())), C = jr.cells;
    const secs = []; for (const l of L) { const k = l.section || ""; const last = secs[secs.length - 1]; if (last && last.name === k) last.n++; else secs.push({ name: k, n: 1 }); }
    const hasSecs = secs.some((x) => x.name);
    const head = `<thead>${hasSecs ? `<tr class="jr-secs"><th class="jr-name" rowspan="2">Суралцагч</th>${secs.map((x) => `<th colspan="${x.n}" title="${esc(x.name || "Бүлэггүй")}">${esc(x.name || "—")}</th>`).join("")}<th rowspan="2">Дууссан</th><th rowspan="2">Дундаж</th><th rowspan="2">Цол</th></tr>` : ""}
      <tr>${hasSecs ? "" : `<th class="jr-name">Суралцагч</th>`}${L.map((l) => `<th class="jr-l ${l.hidden ? "is-hidden" : ""}" title="${esc(l.title)}${l.hidden ? " (хаалттай)" : ""}">${String(l.position).padStart(2, "0")}${l.kind === "exam" ? "<i>📝</i>" : l.kind === "assignment" ? "<i>📎</i>" : ""}</th>`).join("")}${hasSecs ? "" : "<th>Дууссан</th><th>Дундаж</th><th>Цол</th>"}</tr></thead>`;
    const body = S.map((s) => `<tr data-uid="${esc(s.user_id)}"><th class="jr-name" data-jr-student><span class="jr-who">${esc(s.name)}</span><small>${s.enrolled ? (s.username ? "@" + esc(s.username) : "") : "элсээгүй · үнэгүй хичээл"}</small></th>
      ${L.map((l) => { const [t, k] = jrCell(l, C[s.user_id]?.[l.id]); return `<td class="jr-c ${k}" data-lid="${esc(l.id)}">${t}</td>`; }).join("")}
      <td class="jr-sum">${s.done}/${L.filter((l) => !l.hidden).length}</td><td class="jr-sum">${s.avg || "—"}</td><td class="jr-rank" title="${s.rank.points} оноо">${esc(s.rank.insignia || "")} ${esc(s.rank.name)}</td></tr>`).join("");
    const foot = `<tfoot><tr><th class="jr-name">Дууссан</th>${L.map((l) => { const n = S.filter((s) => { const x = C[s.user_id]?.[l.id]; return l.kind === "exam" ? x?.exam?.passed : l.kind === "assignment" ? x?.asg : x?.done; }).length; return `<td>${S.length ? n + "/" + S.length : ""}</td>`; }).join("")}<td colspan="3"></td></tr></tfoot>`;
    box.innerHTML = `<div class="jr-head"><div><h2>Журнал</h2><span class="muted small">${jr.students.length} суралцагч · ${L.length} хичээл · нүд дээр дарж дэлгэрэнгүйг харна</span></div>
        <label class="rail-search jr-search">${ico("search", 15)}<input type="search" placeholder="Нэрээр шүүх…" value="${esc(q)}" aria-label="Нэрээр шүүх"></label>
        <button class="btn btn-glass btn-sm" data-jr-csv>${ico("files", 15)}Excel</button></div>
      <div class="jr-legend"><span class="jr-c ok">68</span>дууссан<span class="jr-c part">24</span>үзэж байна<span class="jr-c bad">40%</span>тэнцээгүй<span class="jr-c wait">✓</span>дүгнээгүй<span class="jr-c disq">⛔</span>хуулах оролдлого</div>
      ${S.length ? `<div class="jr-wrap"><table class="jr-t">${head}<tbody>${body}</tbody>${foot}</table></div>` : `<div class="empty">${q ? "Хайлтад тохирох суралцагч алга." : "Одоогоор суралцагч алга."}</div>`}`;
    const inp = $(".jr-search input", box); inp.oninput = () => { const p = inp.selectionStart; drawJournal(inp.value); const i2 = $(".jr-search input"); i2.focus(); i2.setSelectionRange(p, p); };
  };
  const jrCellModal = (uid, lid) => {
    const s = jr.students.find((x) => x.user_id === uid), l = jr.lessons.find((x) => x.id === lid), x = jr.cells[uid]?.[lid] || {};
    const rows = [];
    rows.push(["Төлөв", x.disq ? "⛔ Хуулах оролдлого — оноо тооцогдоогүй" : x.done ? "✓ Дууссан" : x.viewed ? "Үзэж байна" : "Үзээгүй"]);
    if (x.viewed) rows.push(["Нэгдсэн цолд нэмсэн оноо", x.disq ? "0" : "+" + x.pts]);
    if (x.viewed) rows.push(["Идэвхтэй хугацаа", `${dur2(x.active)}${x.total ? ` (нийт ${dur2(x.total)}, ${x.sessions || 0} удаа)` : ""}`]);
    if (x.q_n) rows.push(["Асуулга", `${x.q_ok}/${x.q_n} зөв`]);
    if (x.has_video) rows.push(["Видео үзэлт", x.video + "%"]);
    if (x.viewed) rows.push(["Дүгнэлт", x.refl ? "✍️ Бичсэн" : "Бичээгүй"]);
    if (x.tabs) rows.push(["Таб солилт", x.tabs + " удаа"]);
    if (x.exam) rows.push(["Шалгалт", `Шилдэг ${x.exam.best}% · ${x.exam.passed ? "✓ тэнцсэн" : "тэнцээгүй"} · ${x.exam.tries} оролдлого${x.exam.terminated ? ` · ⛔ ${x.exam.terminated} хаагдсан` : ""}`]);
    if (x.asg) rows.push(["Даалгавар", `${x.asg.score != null ? `Дүн ${x.asg.score}/${x.asg.max}${l.rubric ? " · рубрикаар" : ""}` : "Илгээсэн — дүгнээгүй"}${x.asg.late ? " · хоцорсон" : ""} · ${fmtDate(x.asg.at)}${x.asg.feedback ? ` · «${esc(x.asg.feedback)}»` : ""}`]);
    if (x.last) rows.push(["Сүүлд", fmtDate(x.last)]);
    jrModal(`<span class="eyebrow">${String(l.position).padStart(2, "0")} · ${esc(l.title)}</span><h3 class="h3">${esc(s.name)}</h3>
      <dl class="jr-dl">${rows.map(([k, v]) => `<dt>${k}</dt><dd>${v}</dd>`).join("")}</dl>
      ${x.reasons?.length ? `<p class="muted small" style="margin:10px 0 0">Үндэслэл: ${x.reasons.map(esc).join(", ")}</p>` : ""}
      ${l.rubric && x.asg?.rubric ? `<h4 class="jr-rb-h">Рубрикийн дүн</h4>${SG.rubricView(l.rubric, x.asg.rubric)}` : ""}
      ${l.kind === "assignment" && x.asg ? `<div class="hero-cta" style="margin:14px 0 0"><button class="btn btn-gold btn-sm" data-jr-grade="${esc(l.id)}">${ico("users", 15)}Дүгнэх</button></div>` : ""}
      ${l.kind === "exam" && x.exam ? `<div class="hero-cta jr-reopen" style="margin:14px 0 0"><button class="btn btn-gold btn-sm" data-reopen-exam="${esc(l.id)}" data-reopen-user="${esc(uid)}">🔓 Шалгалтыг дахин нээх (+1 оролдлого)</button><span class="muted small">${x.exam.terminated ? "Хаагдсан шалгалтыг" : "Оролдлого дууссан ч"} дахин өгөх боломж олгоно — суралцагчид мэдэгдэл очно.</span></div>` : ""}`, l.rubric && x.asg?.rubric ? 900 : 640);
  };
  const jrStudentModal = (uid) => {
    const s = jr.students.find((x) => x.user_id === uid), row = jr.cells[uid] || {};
    jrModal(`<span class="eyebrow">Суралцагч</span><h3 class="h3">${esc(s.name)}</h3>
      <div class="an-tiles"><div class="an-tile"><small>Цол</small><b>${esc(s.rank.insignia || "")} ${esc(s.rank.name)}</b><span class="muted small">${s.rank.points} оноо</span></div>
        <div class="an-tile"><small>Дууссан</small><b>${s.done}/${jr.lessons.filter((l) => !l.hidden).length}</b></div>
        <div class="an-tile"><small>Дундаж оноо</small><b>${s.avg || "—"}</b></div>
        <div class="an-tile"><small>Идэвхтэй</small><b>${dur2(s.active)}</b>${s.last ? `<span class="muted small">сүүлд ${fmtDate(s.last)}</span>` : ""}</div></div>
      <table class="tbl"><thead><tr><th>Хичээл</th><th>Төлөв</th><th>Оноо / дүн</th></tr></thead><tbody>${jr.lessons.map((l) => { const x = row[l.id], [t, k] = jrCell(l, x);
        return `<tr><td>${String(l.position).padStart(2, "0")} · ${esc(l.title)}</td><td>${!x ? `<span class="muted">Үзээгүй</span>` : x.disq ? "⛔ Хуулах оролдлого" : x.done || x.exam?.passed ? "✓ Дууссан" : x.asg ? "Илгээсэн" : "Үзэж байна"}</td><td><span class="jr-c ${k}">${t || "—"}</span></td></tr>`; }).join("")}</tbody></table>`);
  };
  const jrModal = (html, w = 640) => {
    document.body.insertAdjacentHTML("beforeend", `<div class="modal" id="jrModal"><div class="modal-card" style="width:min(${w}px,100%)"><button class="icon-btn modal-x" data-close aria-label="Хаах">${ico("x", 18)}</button>${html}</div></div>`);
    const m = $("#jrModal"); SG.openModal(m);
    m.addEventListener("click", (e) => {
      const g = e.target.closest("[data-jr-grade]"); if (g) { SG.closeModal(m); setTimeout(() => m.remove(), 300); return gradeModal(g.dataset.jrGrade); }
      const ro = e.target.closest("[data-reopen-exam]"); if (ro) return reopenExam(ro, loadJournal);
      if (e.target === m || e.target.closest("[data-close]")) { SG.closeModal(m); setTimeout(() => m.remove(), 300); }
    });
  };
  const jrCSV = () => {
    const L = jr.lessons, cell = (v) => `"${String(v ?? "").replace(/"/g, '""')}"`;
    const lines = [["Суралцагч", ...L.map((l) => `${String(l.position).padStart(2, "0")} ${l.title}`), "Дууссан", "Дундаж", "Цол", "Оноо"].map(cell).join(",")];
    for (const s of jr.students) lines.push([s.name, ...L.map((l) => jrCell(l, jr.cells[s.user_id]?.[l.id])[0]), `${s.done}/${L.filter((l) => !l.hidden).length}`, s.avg || "", s.rank.name, s.rank.points].map(cell).join(","));
    const a = document.createElement("a"); a.href = URL.createObjectURL(new Blob(["\ufeff" + lines.join("\n")], { type: "text/csv;charset=utf-8" }));
    a.download = `zhurnal-${c.title.replace(/[\\/:*?"<>|]+/g, "_")}.csv`; a.click(); setTimeout(() => URL.revokeObjectURL(a.href), 2000);
  };

  // Даалгаврын хариунууд: багш оноо, тайлбар өгнө.
  const gradeModal = async (lid) => {
    const d = await api(`/api/courses/${c.id}/lessons/${lid}/submissions`);
    const max = d.assignment?.max_score || 100, rub = d.assignment?.grading === "rubric" && d.assignment.rubric;
    // Рубрикаар: мөр бүрээс нэг нүд сонгоно — нийт оноо шууд бодогдоно.
    const rbForm = (s) => `<form class="sub-grade sub-grade-rb"><div class="rb-wrap"><table class="rb-grade"><thead><tr><th>Шалгуур</th>${rub.levels.map((lv) => `<th>${esc(lv)}</th>`).join("")}</tr></thead>
      <tbody>${rub.criteria.map((cr) => `<tr><th>${esc(cr.name)}</th>${rub.levels.map((_, j) => `<td><label class="rb-cell ${s.rubric?.[cr.id] === j ? "on" : ""}"><input type="radio" name="rb_${esc(cr.id)}" value="${j}" ${s.rubric?.[cr.id] === j ? "checked" : ""} required><b>${cr.points[j]} оноо</b>${cr.desc[j] ? `<small>${esc(cr.desc[j])}</small>` : ""}</label></td>`).join("")}</tr>`).join("")}</tbody></table></div>
      <div class="rb-grade-foot"><span class="rb-total">Нийт <b>${s.score ?? 0}</b>/${max}</span><label class="grow">Тайлбар<input name="feedback" maxlength="5000" value="${esc(s.feedback || "")}" placeholder="Юу сайн, юуг сайжруулах вэ?"></label><button class="btn btn-gold btn-sm">${s.graded_at ? "Шинэчлэх" : "Дүгнэх"}</button></div></form>`;
    const row = (s) => `<article class="sub-item" data-uid="${esc(s.user_id)}"><header><b>${esc(s.user_name)}</b><small class="muted">${fmtDate(s.submitted_at)}${s.late ? ` · <span class="an-bad">хоцорсон</span>` : ""}</small></header>
      ${s.text ? `<p class="sub-text">${SG.linkify(s.text)}</p>` : ""}${s.links?.length ? `<p class="sub-files">${s.links.map((u) => `<a href="${esc(u)}" target="_blank" rel="noopener noreferrer">🔗 ${esc(u.replace(/^https?:\/\//, "").slice(0, 70))}</a>`).join("<br>")}</p>` : ""}${s.files?.length ? `<p class="sub-files">${s.files.map((f) => `<a href="${esc(f.url)}" target="_blank" rel="noopener">📄 ${esc(f.name)}</a>`).join(" ")}</p>` : ""}
      ${rub ? rbForm(s) : `<form class="sub-grade"><label>Оноо (0-${max})<input name="score" type="number" min="0" max="${max}" value="${s.score ?? ""}" required></label><label class="grow">Тайлбар<input name="feedback" maxlength="5000" value="${esc(s.feedback || "")}" placeholder="Юу сайн, юуг сайжруулах вэ?"></label><button class="btn btn-gold btn-sm">${s.graded_at ? "Шинэчлэх" : "Дүгнэх"}</button>${s.graded_at ? `<small class="muted">✓ ${fmtDate(s.graded_at)}</small>` : ""}</form>`}</article>`;
    document.body.insertAdjacentHTML("beforeend", `<div class="modal" id="gradeModal"><div class="modal-card" style="width:min(820px,100%)"><button class="icon-btn modal-x" data-close aria-label="Хаах">${ico("x", 18)}</button>
      <h3 class="h3">${esc(d.lesson)} — хариунууд (${d.submissions.length})</h3>${d.assignment?.due?.at || d.assignment?.due?.start_at ? `<p class="muted small">${d.assignment.due.start_at ? `Эхлэх: ${fmtDate(d.assignment.due.start_at)} · ` : ""}${d.assignment.due.at ? `Дуусах: ${fmtDate(d.assignment.due.at)} · хоцорвол ${d.assignment.due.late === "paid" ? money(d.assignment.due.late_fee) : d.assignment.due.late === "closed" ? "хаалттай" : "төлбөргүй"}` : ""}${d.assignment.due.fee ? ` · оролцооны төлбөр ${money(d.assignment.due.fee)}` : ""}</p>` : ""}
      <div class="sub-list">${d.submissions.map(row).join("") || `<p class="muted">Хариу ирээгүй байна.</p>`}</div></div></div>`);
    const m = $("#gradeModal"); SG.openModal(m);
    m.addEventListener("click", (e) => { if (e.target === m || e.target.closest("[data-close]")) { SG.closeModal(m); setTimeout(() => m.remove(), 300); } });
    m.addEventListener("change", (e) => { // рубрик: сонгосон нүд тодорч, нийт оноо шууд бодогдоно
      const r = e.target.closest(".rb-cell input"); if (!r || !rub) return;
      const f = r.closest("form");
      $$(`input[name="${CSS.escape(r.name)}"]`, f).forEach((x) => x.closest(".rb-cell").classList.toggle("on", x.checked));
      $(".rb-total b", f).textContent = rub.criteria.reduce((a, cr) => { const v = f.querySelector(`input[name="rb_${CSS.escape(cr.id)}"]:checked`); return a + (v ? cr.points[+v.value] : 0); }, 0);
    });
    m.addEventListener("submit", async (e) => {
      const f = e.target.closest(".sub-grade"); if (!f) return; e.preventDefault();
      const uid = f.closest("[data-uid]").dataset.uid, b = $("button", f); b.disabled = true;
      const body = rub ? { feedback: f.feedback.value, rubric: Object.fromEntries(rub.criteria.map((cr) => [cr.id, +(f.querySelector(`input[name="rb_${CSS.escape(cr.id)}"]:checked`)?.value ?? -1)])) }
        : { score: +f.score.value, feedback: f.feedback.value };
      try {
        const r = await api(`/api/courses/${c.id}/lessons/${lid}/submissions/${uid}`, { method: "PUT", body });
        toast(`Дүгнэлээ ✓ ${r.score}/${max} — суралцагч ба журналд орлоо`); b.textContent = "Шинэчлэх";
        if ($("#journal")) loadJournal(); // журнал автоматаар шинэчлэгдэнэ
      } catch (err) { toast(err.message, true); } finally { b.disabled = false; }
    });
  };
  const saveCourse = async (body) => { c = { ...c, ...(await api(`/api/courses/${c.id}`, { method: "PUT", body })) }; searchCourses = null; };
  const lessonBody = (f, l) => {
    const isFree = f.mode.value === "free";
    const blocks = f._be.collect();
    const kind = f.kind.value;
    const due = (pfx) => { const iso = (v) => (v ? new Date(v).toISOString() : null), late = f[pfx + "_late"].value; return { start_at: iso(f[pfx + "_start"].value), at: iso(f[pfx + "_due"].value), late, late_fee: late === "paid" ? +f[pfx + "_fee"].value || 0 : 0, fee: +f[pfx + "_entry"].value || 0 }; };
    const exam = kind === "exam" ? { time_min: +f.ex_time.value || 0, attempts: +f.ex_attempts.value || 0, pass_pct: +f.ex_pass.value || 0, shuffle: f.ex_shuffle.checked, show_answers: f.ex_show.checked, due: due("ex") } : null;
    const grading = kind === "assignment" && f.asg_grading?.value === "rubric" ? "rubric" : "";
    if (grading) f._rb?.check();
    const assignment = kind === "assignment" ? { due: due("asg"), max_score: +f.asg_max.value || 100, allow_files: false, grading, rubric: grading ? f._rb?.value() : null } : null;
    if (exam && !blocks.some((b) => b.type === "quiz")) throw new Error("Шалгалтад дор хаяж нэг асуулт нэмнэ үү");
    for (const d of [exam?.due, assignment?.due]) {
      if (d?.late === "paid" && !d.late_fee) throw new Error("Хоцролтын төлбөрийг оруулна уу");
      if (d?.start_at && d?.at && d.at <= d.start_at) throw new Error("Дуусах цаг эхлэх цагаас хойш байна");
    }
    return { title: f.title.value, content: blocksSummary(blocks), video_url: "", blocks, active_min: +f.active_min.value || 0, exam, assignment, is_free: isFree, price: isFree ? 0 : +f.price.value || 0,
      unlock_after_h: f.unlock_after_h ? +f.unlock_after_h.value || 0 : l?.unlock_after_h || 0, always_open: f.always_open ? f.always_open.checked : !!l?.always_open,
      format: f.format.value, mode: f.mode_kind.value, section: sectionOf(f), discussion: f.discussion.checked,
      unlock_rule: f.querySelector("input[name=unlock_rule]:checked")?.value ?? l?.unlock_rule ?? "", hidden: !l && !!f.hidden_new?.checked };
  };
  const sectionOf = (f) => (f.section_pick.value === "__new" ? f.section_new.value : f.section_pick.value).trim();
  const lessonPut = (l, patch) => api(`/api/courses/${c.id}/lessons/${l.id}`, { method: "PUT", body: { title: l.title, content: l.content, video_url: stripSig(l.video_url), is_free: l.is_free, price: l.price || 0, unlock_after_h: l.unlock_after_h || 0, always_open: !!l.always_open, format: l.format || "", mode: l.mode || "", section: l.section || "", blocks: (l.blocks || []).map((b) => (b.url ? { ...b, url: stripQ(b.url) } : b.quiz?.image ? { ...b, quiz: { ...b.quiz, image: stripQ(b.quiz.image) } } : b)), active_min: l.active_min || 0, exam: l.exam || null, assignment: l.assignment || null, discussion: !!l.discussion, unlock_rule: l.unlock_rule || "", ...patch } });
  const reload = async () => { const d = await api(`/api/me/courses/${id}`); c = d.course; lessons = d.lessons; commentCounts = d.comments || {}; roster = await api(`/api/me/courses/${id}/students`).catch(() => roster); render(); };

  // Засварлах маягт + блок засварлагч.
  const putEditor = (box, l, preset, kind) => { box.innerHTML = editorHTML(l, preset, kind); mountBlockEditor($(".post-editor", box), lessonBlocks(l)); };
  // ---- Хөтөлбөрийн DOM → хадгалах ----
  const domSecs = () => $$("#olSecs > .ol-sec");
  const domNames = () => domSecs().map((x) => x.dataset.sec);
  const domItems = () => domSecs().flatMap((x) => $$(":scope > .ol-list > .ol-row", x).map((r) => ({ id: r.dataset.lid, section: x.dataset.sec })));
  const saveOutline = async (msg = "Хөтөлбөр хадгалагдлаа ✓", focusLid) => {
    const names = domNames(), items = domItems();
    pending = names.map((name, at) => ({ name, at })).filter((p) => p.name && !items.some((it) => it.section === p.name));
    const before = [...lessons].sort((a, b) => a.position - b.position).map((l) => l.id + "|" + (l.section || "")).join(",");
    if (items.map((it) => it.id + "|" + it.section).join(",") === before) return render();
    try { lessons = await api(`/api/courses/${c.id}/lesson-order`, { method: "PUT", body: { items } }); toast(msg); }
    catch (err) { toast(err.message, true); await reload(); return; }
    render();
    if (focusLid) $(`.ol-row[data-lid="${CSS.escape(focusLid)}"] .ol-handle`)?.focus();
  };
  // Шинэ хичээлийг сонгосон бүлгийнхээ төгсгөлд байрлуулна (бүлэг хөтөлбөрийн дунд байсан ч).
  const placeNew = async (lid, name) => {
    const gs = groupsModel(), ti = gs.findIndex((g) => g.name === name);
    if (ti < 0) return;
    const items = [];
    gs.forEach((g, i) => { g.items.filter((l) => l.id !== lid).forEach((l) => items.push({ id: l.id, section: g.name })); if (i === ti) items.push({ id: lid, section: name }); });
    if (items.length !== lessons.length) return;
    lessons = await api(`/api/courses/${c.id}/lesson-order`, { method: "PUT", body: { items } }).catch(() => lessons);
  };
  const renameSection2 = async (secEl, input) => {
    const to = input.value.replace(/\s+/g, " ").trim(), from = secEl.dataset.sec;
    if (to === from) return;
    if (to && to !== from && domNames().includes(to)) toast(`«${to}» бүлэгтэй нэгтгэлээ`);
    secEl.dataset.sec = to;
    if (!$$(".ol-row", secEl).length) { // хоосон бүлэг: зөвхөн клиент дээр
      const names = domNames(); pending = names.map((name, at) => ({ name, at })).filter((p) => p.name && !lessons.some((l) => (l.section || "") === p.name));
      if (to) toast("Бүлэг нэмэгдлээ — хичээлүүдээ чирж оруулна уу"); return render();
    }
    await saveOutline(to ? "Бүлгийн нэр хадгалагдлаа ✓" : "Бүлэг задарлаа");
  };

  // ---- Чирж зөөх (хулгана ба хуруу: pointer events) ----
  let drag = null;
  const onPointerDown = (e) => {
    const h = e.target.closest("[data-drag]");
    if (!h || e.button > 0 || $(".ol-row.editing", main)) return;
    const kind = h.dataset.drag, item = h.closest(kind === "sec" ? ".ol-sec" : ".ol-row");
    if (!item) return;
    e.preventDefault();
    const src = kind === "sec" ? $(".ol-sec-head", item) : item, r = src.getBoundingClientRect();
    const ghost = src.cloneNode(true); ghost.classList.add("ol-ghost");
    Object.assign(ghost.style, { width: r.width + "px", left: r.left + "px", top: r.top + "px" });
    document.body.append(ghost);
    if (kind === "sec") $("#olSecs").classList.add("ol-compact");
    item.classList.add("ol-dragging");
    drag = { kind, item, ghost, dy: e.clientY - r.top, x: r.left + 24, moved: false };
    window.addEventListener("pointermove", onPointerMove); window.addEventListener("pointerup", onPointerUp); window.addEventListener("pointercancel", onPointerUp);
  };
  const onPointerMove = (e) => {
    if (!drag) return;
    drag.moved = true; drag.ghost.style.top = e.clientY - drag.dy + "px";
    if (e.clientY < 90) scrollBy(0, -14); else if (e.clientY > innerHeight - 60) scrollBy(0, 14);
    const el = document.elementFromPoint(drag.x, e.clientY);
    const below = (els) => els.find((x) => { const b = x.getBoundingClientRect(); return e.clientY < b.top + b.height / 2; }) || null;
    if (drag.kind === "row") {
      const list = el?.closest(".ol-sec")?.querySelector(":scope > .ol-list");
      if (list && main.contains(list)) list.insertBefore(drag.item, below($$(":scope > .ol-row:not(.ol-dragging)", list)));
    } else {
      const box = $("#olSecs"); if (box) box.insertBefore(drag.item, below($$(":scope > .ol-sec:not(.ol-dragging)", box)));
    }
  };
  const onPointerUp = () => {
    window.removeEventListener("pointermove", onPointerMove); window.removeEventListener("pointerup", onPointerUp); window.removeEventListener("pointercancel", onPointerUp);
    if (!drag) return;
    const d = drag; drag = null; d.ghost.remove(); d.item.classList.remove("ol-dragging"); $("#olSecs")?.classList.remove("ol-compact");
    if (d.moved) saveOutline(d.kind === "sec" ? "Бүлгийн дараалал хадгалагдлаа ✓" : "Хичээл зөөгдлөө ✓");
  };
  // Гараар зөөх: зөөх товч дээр ↑ ↓ (бүлгийн хил давж шилжинэ).
  const onKeyDown = (e) => {
    const inp = e.target.closest(".ol-sec-name");
    if (inp && e.key === "Enter") { e.preventDefault(); inp.blur(); return; }
    if (inp && e.key === "Escape") { inp.value = inp.closest(".ol-sec").dataset.sec; inp.blur(); return; }
    const h = e.target.closest("[data-drag=row]");
    if (!h || (e.key !== "ArrowUp" && e.key !== "ArrowDown")) return;
    e.preventDefault();
    const row = h.closest(".ol-row"), list = row.parentElement, sec = list.closest(".ol-sec"), up = e.key === "ArrowUp";
    const sib = up ? row.previousElementSibling : row.nextElementSibling;
    if (sib) up ? list.insertBefore(row, sib) : list.insertBefore(row, sib.nextElementSibling);
    else { const other = up ? sec.previousElementSibling : sec.nextElementSibling; if (!other) return; const ol = $(":scope > .ol-list", other); up ? ol.append(row) : ol.prepend(row); }
    saveOutline("Хичээл зөөгдлөө ✓", row.dataset.lid);
  };
  const onFocusOut = (e) => {
    const inp = e.target.closest?.(".ol-sec-name"); if (!inp) return;
    const secEl = inp.closest(".ol-sec");
    if (secEl.classList.contains("ol-new") && !inp.value.trim()) { secEl.remove(); return; }
    renameSection2(secEl, inp);
  };

  // Нэг л хичээл дээр ажиллана: өөр хичээл (эсвэл шинэ хичээлийн маягт) нээхэд бусад засварлагч хаагдана.
  const editingNow = () => $$(".post.editing, .ol-row.editing", main).length > 0 || ($("#composerBody") && !$("#composerBody").hidden);
  const closeEditors = () => {
    if ($$(".post-editor[data-dirty]", main).some((f) => f.offsetParent) && !confirm("Нээлттэй хичээлд хадгалаагүй өөрчлөлт байна. Хаагаад үргэлжлүүлэх үү?")) return false;
    render(); return true;
  };
  const onClick = async (e) => {
    const t = e.target;
    const openComposer = (preset, kind) => { if ($$(".post.editing, .ol-row.editing", main).length && !closeEditors()) return; const b = $("#composerBody"); if (preset !== undefined || kind) putEditor(b, null, preset, kind); $("#composerOpen").hidden = true; b.hidden = false; $("#composer").scrollIntoView({ behavior: "smooth", block: "start" }); $("#composerForm").title.focus(); };
    // Нэмэх товч → төрөл сонгох жижиг цэс: Хичээл / Шалгалт / Даалгавар.
    const kindMenu = (anchor, preset) => {
      $(".kind-menu")?.remove();
      const m = document.createElement("div"); m.className = "kind-menu"; m.setAttribute("role", "menu");
      m.innerHTML = `<button type="button" role="menuitem" data-new-kind="lesson">📄 <b>Хичээл</b><small>текст, видео, файл, асуулт</small></button>
        <button type="button" role="menuitem" data-new-kind="exam">📝 <b>Шалгалт</b><small>хугацаа, оролдлого, хоцорвол төлбөртэй/төлбөргүй</small></button>
        <button type="button" role="menuitem" data-new-kind="assignment">📎 <b>Даалгавар</b><small>хариу илгээж дүгнүүлнэ, хоцорвол төлбөртэй/төлбөргүй</small></button>`;
      document.body.append(m);
      const place = () => { const r = anchor.getBoundingClientRect(), w = 300; m.style.top = Math.min(r.bottom + 6, innerHeight - 190) + "px"; m.style.left = Math.max(8, Math.min(r.right - w, innerWidth - w - 8)) + "px"; };
      place();
      const kill = () => { m.remove(); removeEventListener("scroll", place, true); removeEventListener("resize", place); };
      m.onclick = (ev) => { const k = ev.target.closest("[data-new-kind]")?.dataset.newKind; kill(); if (k) openComposer(preset ?? "", k); };
      addEventListener("scroll", place, true); addEventListener("resize", place); // гүйлгэхэд товчоо дагана
      setTimeout(() => document.addEventListener("click", (ev) => { if (!m.contains(ev.target)) kill(); }, { once: true }), 0);
    };
    if (t.closest("#composerOpen")) return kindMenu(t.closest("#composerOpen"));
    const sec = t.closest(".sec, .ol-sec");
    if (t.closest("[data-sec-add]") && sec) return kindMenu(t.closest("[data-sec-add]"), sec.dataset.sec);
    if (t.closest("[data-grade]")) { const lid = t.closest("[data-lid]")?.dataset.lid; if (lid) return gradeModal(lid); }
    if (jr && t.closest("#journal")) {
      const td = t.closest("td.jr-c[data-lid]"), nm = t.closest("[data-jr-student]");
      if (td) return jrCellModal(td.closest("tr").dataset.uid, td.dataset.lid);
      if (nm) return jrStudentModal(nm.closest("tr").dataset.uid);
      if (t.closest("[data-jr-csv]")) return jrCSV();
    }
    const tab = t.closest("[data-view]");
    if (tab) { view = tab.dataset.view; try { localStorage.setItem("sg_course_view", view); } catch {} return render(); }
    if (t.closest("#olAddSec")) {
      const names = domNames(); pending.push({ name: "", at: names.length });
      $("#olSecs").insertAdjacentHTML("beforeend", `<section class="ol-sec ol-new" data-sec=""><header class="ol-sec-head"><span class="lg-num">${String(names.length + 1).padStart(2, "0")}</span>
        <input class="ol-sec-name" maxlength="80" placeholder="Бүлгийн нэрээ бичээд Enter дарна уу" aria-label="Шинэ бүлгийн нэр"></header><ol class="ol-list"></ol></section>`);
      pending.pop(); $("#olSecs > .ol-sec:last-child .ol-sec-name").focus(); return;
    }
    if (t.closest("[data-sec-del]") && sec) {
      const name = sec.dataset.sec; pending = pending.filter((p) => p.name !== name);
      sec.dataset.sec = ""; $$(".ol-row", sec).length ? await saveOutline("Бүлэг задарлаа — хичээлүүд бүлэггүй боллоо") : render();
      return;
    }
    if (t.closest("#gpSettings")) { SG.openModal($("#gpModal")); return; }
    if (t.closest("#gpRoster")) { SG.openModal($("#gpRosterModal")); return; }
    if (t.closest("#gpChat")) {
      const b = $("#gpChat"); b.disabled = true;
      try { const d = await api(`/api/courses/${c.id}/chat`, { method: "POST" }); Live.reconnect(); window.openRailChat?.(d.conversation.id); }
      catch (err) { toast(err.message, true); } finally { b.disabled = false; }
      return;
    }
    if (t.closest("#gpPublish")) {
      try { await saveCourse({ title: c.title, description: c.description, price: c.price, published: true, drip: !!c.drip, unlock_all_paid: !!c.unlock_all_paid, certificate: !!c.certificate, camera: c.camera || "optional" }); toast("Сургалт нийтлэгдлээ ✓"); celebrate(); render(); } catch (err) { toast(err.message, true); }
      return;
    }
    const post = t.closest(".post, .ol-row"), l = post && lessons.find((x) => x.id === post.dataset.lid);
    if (t.closest("[data-cancel]")) return render();
    if (t.closest("[data-edit-post]") && l) {
      if (post.classList.contains("editing")) return;
      let row = post;
      if (editingNow()) { // бусад нь хаагдана
        if (!closeEditors()) return;
        row = $(`.post[data-lid="${CSS.escape(l.id)}"], .ol-row[data-lid="${CSS.escape(l.id)}"]`, main); if (!row) return;
      }
      row.classList.add("editing"); putEditor(row, l);
      row.scrollIntoView({ block: "start", behavior: "smooth" }); $("input[name=title]", row)?.focus({ preventScroll: true });
      return;
    }
    if (t.closest("[data-del-post]") && l) {
      const nFiles = new Set([l.video_url, ...(l.blocks || []).flatMap((b) => [b.url, b.quiz?.image])].map(stripQ).filter((u) => u.startsWith("/files/"))).size;
      if (!confirm(`«${l.title}» хичээлийг устгах уу?${nFiles ? `\nЗөвхөн энэ хичээлд ашигласан файлууд (${nFiles}) хамт устна.` : ""}\nСуралцагчдын явцын түүх хадгалагдана.`)) return;
      try { const r = await api(`/api/courses/${c.id}/lessons/${l.id}`, { method: "DELETE" }); toast(r.deleted_files ? `Хичээл ${r.deleted_files} файлын хамт устгагдлаа` : "Хичээл устгагдлаа"); await reload(); }
      catch (err) { toast(err.message, true); }
      return;
    }
    if (t.closest("[data-open-media]") && l) { // агуулгыг суралцагчийн харах байдлаар урьдчилан үзнэ
      const box = $(".post-preview", post) || $(".post-media", post); box.hidden = !box.hidden;
      if (!box.hidden && !box.innerHTML) { box.innerHTML = SG.blocksHTML(lessonBlocks(l)); hydrateBooks(box); SG.mountQuizzes(box, `/api/courses/${c.id}/lessons/${l.id}`); }
      return;
    }
    if (t.closest("[data-toggle-hidden]") && l) return setHidden(l, !l.hidden);
    if (t.closest("[data-toggle-always]") && l) {
      try { await lessonPut(l, { always_open: !l.always_open }); toast(l.always_open ? "Хичээл дараалалд орлоо" : "Хичээл дарааллаас үл хамааран нээлттэй боллоо ✓"); await reload(); } catch (err) { toast(err.message, true); }
      return;
    }
    if (t.closest("[data-toggle-free]") && l) {
      const toFree = !l.is_free;
      if (!toFree && !c.price && !l.price) { putEditor(post, { ...l, is_free: false }); toast("Төлбөртэй хичээлийн үнийг оруулна уу"); return; }
      try { await lessonPut(l, { is_free: toFree, price: toFree ? 0 : l.price || 0 });
        toast(toFree ? "Хичээл үнэгүй боллоо ✓" : "Хичээл төлбөртэй боллоо ✓"); await reload(); } catch (err) { toast(err.message, true); }
    }
  };
  const setHidden = async (l, hidden, ctl) => {
    if (ctl) ctl.disabled = true;
    try {
      await api(`/api/courses/${c.id}/lessons/${l.id}/visibility`, { method: "PUT", body: { hidden } });
      l.hidden = hidden;
      toast(hidden ? "Хичээл хаагдлаа — суралцагчдад харагдахгүй" : "Хичээл нээгдлээ — суралцагчдад харагдана ✓");
      render();
    } catch (err) { if (ctl) ctl.checked = !hidden; toast(err.message, true); } finally { if (ctl) ctl.disabled = false; }
  };
  const onChange = (e) => {
    const vis = e.target.closest("[data-vis]");
    if (vis) { const l = lessons.find((x) => x.id === vis.closest("[data-lid]")?.dataset.lid); if (l) setHidden(l, !vis.checked, vis); return; }
    const f = e.target.closest(".post-editor");
    if (!f) return;
    if (e.target.name === "mode") $(".pe-price", f).hidden = f.mode.value === "free";
    if (e.target.name === "kind") {
      const k = f.kind.value, T = KIND_TXT[k]; f.dataset.kind = k;
      $(".pe-exam", f).hidden = k !== "exam"; $(".pe-asg", f).hidden = k !== "assignment";
      $$(".pe-kind-hint", f).forEach((p) => (p.hidden = p.dataset.k !== k));
      f.title.placeholder = T.title; $("[data-content-title]", f).textContent = T.content; $("[data-content-hint]", f).textContent = T.hint;
      if (!f.dataset.edit) $(".pe-foot .btn-gold", f).textContent = k === "exam" ? "Шалгалт нийтлэх" : k === "assignment" ? "Даалгавар нийтлэх" : "Хичээл нийтлэх";
    }
    if (e.target.name === "asg_grading") { const rub = e.target.value === "rubric"; $$(".gr-mode", f).forEach((x) => x.classList.toggle("on", x.contains(e.target))); $(".gr-score", f).hidden = rub; $(".gr-rubric", f).hidden = !rub; }
    if (e.target.name === "unlock_rule") { $$(".ur", f).forEach((x) => x.classList.toggle("on", x.contains(e.target))); const tm = $(".ur-time", f); if (tm) tm.hidden = !["view", "complete", ""].includes(e.target.value); }
    if (/_late$/.test(e.target.name)) { const fee = e.target.closest(".pe-grid")?.querySelector(".pe-fee"); if (fee) fee.hidden = e.target.value !== "paid"; }
    if (e.target.name === "section_pick") { const isNew = f.section_pick.value === "__new"; f.section_new.hidden = !isNew; f.section_new.required = isNew; if (isNew) f.section_new.focus(); }
  };
  const onSubmit = async (e) => {
    const f = e.target; e.preventDefault();
    const errBox = $(".form-error", f); if (errBox) errBox.textContent = "";
    try {
      if (f.id === "gpForm") { await saveCourse(courseBody(f)); SG.closeModal($("#gpModal")); toast("Тохиргоо хадгалагдлаа ✓"); render(); return; }
      if (!f.classList.contains("post-editor")) return;
      if (f._be?.uploading()) { toast("Файл хуулагдаж дуустал түр хүлээнэ үү"); return; }
      if (f.dataset.edit) { await api(`/api/courses/${c.id}/lessons/${f.dataset.edit}`, { method: "PUT", body: lessonBody(f, lessons.find((x) => x.id === f.dataset.edit)) }); toast("Хичээл шинэчлэгдлээ ✓"); }
      else {
        const body = lessonBody(f), nl = await api(`/api/courses/${c.id}/lessons`, { method: "POST", body });
        toast("Хичээл нийтлэгдлээ ✓");
        if (body.section) { lessons = [...lessons, nl]; await placeNew(nl.id, body.section); }
      }
      await reload();
    } catch (err) { if (errBox) errBox.textContent = err.message; toast(err.message, true); }
  };
  const onEdInput = (e) => { const pe = e.target.closest?.(".post-editor"); if (pe) pe.dataset.dirty = "1"; }; // хадгалаагүй өөрчлөлт
  const evs = [["click", onClick], ["change", onChange], ["submit", onSubmit], ["pointerdown", onPointerDown], ["keydown", onKeyDown], ["focusout", onFocusOut], ["input", onEdInput]];
  evs.forEach(([n, f]) => main.addEventListener(n, f));
  cleanup = () => { evs.forEach(([n, f]) => main.removeEventListener(n, f)); onPointerUp(); };
  render();
}

/* Лог хүснэгт: эхлээд 20 мөр, доош гүйлгэхэд дараагийн 20-ыг нэмнэ (ачаалсан жагсаалтаас). */
// Хаагдсан (эсвэл оролдлого нь дууссан) шалгалтыг багш гараар дахин нээнэ: +1 оролдлого, хориг цуцлагдана.
async function reopenExam(b, after) {
  if (b.disabled) return; b.disabled = true;
  try {
    const r = await api(`/api/me/students/${b.dataset.reopenUser}/unblock`, { method: "POST", body: { lesson_id: b.dataset.reopenExam } });
    b.textContent = r.unblocked ? "✓ Нээгдлээ · +1 оролдлого" : "Нээх оролдлого алга"; b.classList.add("done");
    if (r.unblocked) { toast("🔓 Шалгалт дахин нээгдлээ — суралцагчид мэдэгдэл очлоо"); after?.(); }
  } catch (e) { b.disabled = false; toast(e.message, true); }
}

function scrollRows(box, items, rowFn, step = 20) {
  const tb = box.querySelector("tbody"); if (!tb) return;
  let i = 0;
  const more = document.createElement("div"); more.className = "ev-more muted small"; box.appendChild(more);
  const next = () => {
    tb.insertAdjacentHTML("beforeend", items.slice(i, i + step).map(rowFn).join("")); i += step;
    more.textContent = i < items.length ? `${Math.min(i, items.length)} / ${items.length} · доош гүйлгэвэл цааш` : items.length > step ? `Бүгд · ${items.length}` : "";
    if (i < items.length && box.isConnected && box.scrollHeight <= box.clientHeight + 60) requestAnimationFrame(next);
  };
  new IntersectionObserver((es) => { if (es.some((e) => e.isIntersecting) && i < items.length) next(); }, { root: box, rootMargin: "160px" }).observe(more);
  more.addEventListener("click", () => i < items.length && next());
  next();
}

/* ---------- Ном, өгүүлэл: зарах, хамгаалах, лог ----------
   Файлыг багшийн хөтөч дээр хуудас бүрээр зураг болгож илгээнэ — эх PDF уншигчид огт очихгүй. */
const BOOK_EVENTS = { view: ["👀", "Хуудсыг үзсэн"], preview: ["📖", "Үнэгүй хэсгийг уншсан"], read: ["📚", "Бүтнээр уншсан"], interest: ["♥", "Сонирхсон"],
  paywall: ["🔒", "Төлбөрийн хананд хүрсэн"], purchase: ["💰", "Худалдаж авсан"], limit: ["⚠️", "Хэт хурдан татах гэсэн"] };
async function books() {
  let [list, evs] = await Promise.all([api("/api/me/books"), api("/api/me/book-events?limit=1000")]);
  const sum = (k) => list.reduce((a, b) => a + (b[k] || 0), 0);
  const cover = (b) => (b.cover_url || (b.pages ? `/api/books/${b.id}/cover?v=${encodeURIComponent(b.updated_at)}` : ""));
  const render = () => {
    main.innerHTML = `<div class="gp-list-head"><div><h1>Ном, өгүүлэл</h1><p class="muted">${list.length} ном · ${list.filter((b) => b.published).length} нийтлэгдсэн</p></div>
        <button class="btn btn-gold" id="bkNew">${ico("plus", 18)}Ном нэмэх</button></div>
      <div class="bk-stats">${[["Үзсэн", sum("views")], ["Үнэгүй хэсэг уншсан", sum("previews")], ["Бүтнээр уншсан", sum("reads")], ["Сонирхсон", sum("interest")], ["Зарагдсан", sum("sales")], ["Орлого", money(sum("revenue"))]]
        .map(([k, v]) => `<div class="card bk-stat"><small class="muted">${k}</small><b>${v}</b></div>`).join("")}</div>
      <div class="bk-manage">${list.map((b) => `<article class="card bk-row" data-bid="${esc(b.id)}">
          <span class="bc-cover">${cover(b) ? `<img src="${esc(cover(b))}" alt="" loading="lazy">` : ""}</span>
          <div class="grow"><strong>${esc(b.title)}</strong><small class="muted">${b.kind === "article" ? "Өгүүлэл" : "Ном"}${b.author ? " · " + esc(b.author) : ""} · ${b.pages ? b.pages + " хуудас" : "<b style='color:var(--coral)'>файл оруулаагүй</b>"}</small>
            <span class="find-tagrow"><span class="ftag ${b.price ? "ftag-paid" : "ftag-free"}">${b.price ? money(b.price) : "Үнэгүй"}</span><span class="ftag ${b.published ? "ftag-free" : ""}">${b.published ? "Нийтлэгдсэн" : "Ноорог"}</span>${b.price ? `<span class="ftag">Эхний ${b.preview_pages} хуудас үнэгүй</span>` : ""}</span>
            <small class="muted">👀 ${b.views} · 📖 ${b.previews} · 📚 ${b.reads} · ♥ ${b.interest} · 💰 ${b.sales} (${money(b.revenue)})</small></div>
          <div class="bk-acts"><button class="btn btn-glass btn-sm" data-edit>${ico("edit", 15)}Засах</button>${b.published ? `<a class="btn btn-glass btn-sm" href="/b/${esc(b.id)}" target="_blank" rel="noopener">${ico("eye", 15)}Харах</a>` : ""}<button class="btn btn-ghost btn-sm" data-log>Лог</button></div></article>`).join("") || `<div class="empty">Ном хараахан алга.<br>PDF, Word (DOC, DOCX) файлаа оруулж зарж эхлээрэй.</div>`}</div>
      <section class="card"><div class="card-head"><h2>Үйлдлийн лог</h2><select id="bkLogFilter" aria-label="Номоор шүүх"><option value="">Бүх ном</option>${list.map((b) => `<option value="${esc(b.id)}">${esc(b.title)}</option>`).join("")}</select></div>
        <div class="bk-log" id="bkLog"></div></section>`;
    mountLog();
  };
  const title = (id) => list.find((b) => b.id === id)?.title || "";
  const logRow = (e) => `<tr class="ev-${esc(e.type)}">
      <td>${fmtDate(e.at)}</td><td>${esc(e.user_name || "Зочин")}</td><td>${(BOOK_EVENTS[e.type] || ["•", e.type]).join(" ")}${e.detail ? ` <small class="muted">${esc(e.detail)}</small>` : ""}</td><td>${esc(title(e.book_id))}</td></tr>`;
  const logHTML = (evs) => evs.length ? `<table class="tbl"><thead><tr><th>Цаг</th><th>Хэн</th><th>Үйлдэл</th><th>Ном</th></tr></thead><tbody></tbody></table>` : `<p class="muted small" style="margin:0">Одоогоор лог алга.</p>`;
  const mountLog = () => { const box = $("#bkLog"); if (box) { box.innerHTML = logHTML(evs); scrollRows(box, evs, logRow); } };
  const reload = async () => { list = await api("/api/me/books"); evs = await api("/api/me/book-events?limit=1000" + ($("#bkLogFilter")?.value ? "&book=" + $("#bkLogFilter").value : "")); render(); };

  const formHTML = (b = {}) => `<form class="form" id="bkForm">
      <div class="seg" role="radiogroup" aria-label="Төрөл"><label><input type="radio" name="kind" value="book" ${b.kind !== "article" ? "checked" : ""}><span>📕 Ном</span></label><label><input type="radio" name="kind" value="article" ${b.kind === "article" ? "checked" : ""}><span>📄 Өгүүлэл</span></label></div>
      <label>Нэр<input name="title" required maxlength="200" value="${esc(b.title || "")}"></label>
      <label>Зохиогч<input name="author" maxlength="120" value="${esc(b.author || me.display_name || "")}"></label>
      <label>Тайлбар<textarea name="description" rows="4" placeholder="Энэ номонд юу байгаа вэ?">${esc(b.description || "")}</textarea></label>
      <div class="kind-row"><label>Үнэ (₮, 0 = үнэгүй)<input name="price" type="number" min="0" step="500" value="${b.price ?? 0}"></label>
        <label>Үнэгүй үзүүлэх хуудас<input name="preview_pages" type="number" min="1" max="20" value="${b.preview_pages || 3}"></label></div>
      <fieldset><legend>Номын файл</legend>
        <p class="muted small" style="margin:0 0 8px">PDF, Word (DOC, DOCX) эсвэл PowerPoint. Хуудас бүр зураг болж хамгаалагдана — эх файл уншигчид очихгүй.${b.pages ? ` Одоо: <b>${b.pages} хуудас</b>.` : ""}</p>
        <input type="file" name="file" accept=".pdf,.doc,.docx,.odt,.rtf,.ppt,.pptx">
        <div class="progress" hidden><i></i></div><p class="muted small bk-prog" aria-live="polite"></p></fieldset>
      <fieldset><legend>Нүүр зураг (заавал биш)</legend><input type="file" name="cover" accept="image/*"><small class="muted">Оруулахгүй бол эхний хуудас нүүр болно.</small></fieldset>
      <label class="check"><input type="checkbox" name="published" ${b.published ? "checked" : ""}> Нийтлэх — профайл дээр харагдаж, зарагдана</label>
      <p class="form-error" role="alert"></p>
      <div class="hero-cta" style="margin:0;justify-content:space-between">${b.id ? `<button type="button" class="btn btn-ghost" data-del>Устгах</button>` : "<span></span>"}
        <span style="display:flex;gap:8px"><button type="button" class="btn btn-ghost" data-close>Болих</button><button class="btn btn-gold">Хадгалах</button></span></div></form>`;
  const openForm = (b) => {
    document.body.insertAdjacentHTML("beforeend", `<div class="modal" id="bkModal"><div class="modal-card" style="width:min(640px,100%)"><button class="icon-btn modal-x" data-close aria-label="Хаах">${ico("x", 18)}</button><h3 class="h3">${b ? "Ном засах" : "Ном нэмэх"}</h3>${formHTML(b || {})}</div></div>`);
    const modal = $("#bkModal"), f = $("#bkForm", modal);
    SG.openModal(modal);
    const closeIt = () => { SG.closeModal(modal); setTimeout(() => modal.remove(), 300); };
    modal.addEventListener("click", (e) => { if (e.target === modal || e.target.closest("[data-close]")) { e.preventDefault(); closeIt(); } });
    $("[data-del]", f)?.addEventListener("click", async () => {
      const msg = b.sales ? `Энэ номыг ${b.sales} хүн худалдаж авсан байна. Устгавал тэд хандах эрхээ алдана!\nХуудасны зургууд, нүүр зураг хамт устна. Үнэхээр устгах уу?` : "Энэ номыг хуудасны зургуудтай нь хамт устгах уу?";
      if (!confirm(msg)) return;
      try { await api(`/api/me/books/${b.id}${b.sales ? "?force=1" : ""}`, { method: "DELETE" }); toast("Ном файлуудтайгаа хамт устгагдлаа"); closeIt(); reload(); } catch (e) { toast(e.message, true); }
    });
    f.onsubmit = async (e) => {
      e.preventDefault();
      const err = $(".form-error", f), btn = $("button.btn-gold", f); err.textContent = ""; btn.disabled = true;
      try {
        let coverURL = b?.cover_url || "";
        if (f.cover.files[0]) { const fd = new FormData(); fd.append("file", f.cover.files[0]); coverURL = (await api("/api/me/files?visibility=public", { method: "POST", body: fd })).path; }
        const body = { kind: f.kind.value, title: f.title.value, author: f.author.value, description: f.description.value, price: +f.price.value || 0, preview_pages: +f.preview_pages.value || 3, cover_url: coverURL, published: false };
        const saved = b ? await api(`/api/me/books/${b.id}`, { method: "PUT", body: { ...body, published: b.pages ? f.published.checked : false } }) : await api("/api/me/books", { method: "POST", body });
        let pages = saved.pages || b?.pages || 0;
        if (f.file.files[0]) pages = await preparePages(saved.id, f.file.files[0], f);
        if (f.published.checked && pages) await api(`/api/me/books/${saved.id}`, { method: "PUT", body: { ...body, published: true } });
        else if (f.published.checked) throw new Error("Нийтлэхийн тулд номын файлаа оруулна уу");
        toast("Хадгалагдлаа ✓"); closeIt(); reload();
      } catch (ex) { err.textContent = ex.message; } finally { btn.disabled = false; }
    };
  };
  // Файл → PDF → хуудас бүрийг JPEG болгож илгээнэ.
  const preparePages = async (bookId, file, f) => {
    const bar = $(".progress", f), fill = $("i", bar), msg = $(".bk-prog", f);
    const say = (t, pct) => { msg.textContent = t; bar.hidden = pct == null; if (pct != null) fill.style.width = pct + "%"; };
    let src;
    if (/\.pdf$/i.test(file.name)) src = { data: new Uint8Array(await file.arrayBuffer()) };
    else { // Word/PowerPoint: сервер PDF болгоно (LibreOffice)
      say("Word файлыг PDF болгож байна…", 10);
      const fd = new FormData(); fd.append("file", file);
      let info = await api("/api/me/files?visibility=private&convert=pdf", { method: "POST", body: fd }); // ном: заавал PDF
      if (!/\.pdf$/i.test(info.path)) throw new Error("Сервер дээр LibreOffice суугаагүй тул Word файлыг хөрвүүлэх боломжгүй — PDF болгож оруулна уу");
      for (let i = 0; info.status === "processing" && i < 120; i++) {
        await new Promise((r) => setTimeout(r, 2000));
        const lib = await api("/api/me/files"); info = lib.files.find((x) => x.path === info.path) || info;
      }
      if (info.status !== "ready") throw new Error("PDF болгож чадсангүй");
      src = { url: info.url };
    }
    const pdf = await (await SG.pdfLib()).getDocument(src).promise, n = pdf.numPages;
    if (n > 3000) throw new Error("3000-аас олон хуудастай");
    let done = 0;
    const one = async (p) => {
      const page = await pdf.getPage(p), base = page.getViewport({ scale: 1 }), vp = page.getViewport({ scale: Math.min(3, 1400 / base.width) });
      const cv = document.createElement("canvas"); cv.width = Math.round(vp.width); cv.height = Math.round(vp.height);
      const ctx = cv.getContext("2d"); ctx.fillStyle = "#fff"; ctx.fillRect(0, 0, cv.width, cv.height);
      await page.render({ canvasContext: ctx, viewport: vp }).promise;
      const blob = await new Promise((r) => cv.toBlob(r, "image/jpeg", 0.86));
      for (let t = 0; ; t++) {
        const res = await fetch(`/api/me/books/${bookId}/pages/${p}`, { method: "POST", headers: { Authorization: "Bearer " + Auth.token, "Content-Type": "image/jpeg" }, body: blob });
        if (res.ok) break;
        if (t >= 2) throw new Error((await res.json().catch(() => null))?.error || `${p}-р хуудсыг илгээж чадсангүй`);
      }
      page.cleanup(); done++; say(`Хуудас бэлтгэж байна: ${done}/${n}`, Math.round(done / n * 100));
    };
    let next = 1;
    await Promise.all([0, 1, 2].map(async () => { while (next <= n) await one(next++); }));
    await api(`/api/me/books/${bookId}/pages-done`, { method: "POST", body: { total: n } });
    say(`✓ ${n} хуудас хамгаалагдан бэлэн боллоо`, null);
    return n;
  };
  const onClick = (e) => {
    if (e.target.closest("#bkNew")) return openForm(null);
    const row = e.target.closest(".bk-row"), b = row && list.find((x) => x.id === row.dataset.bid);
    if (e.target.closest("[data-edit]") && b) return openForm(b);
    if (e.target.closest("[data-log]") && b) { $("#bkLogFilter").value = b.id; $("#bkLogFilter").dispatchEvent(new Event("change", { bubbles: true })); $("#bkLog").scrollIntoView({ behavior: "smooth" }); }
  };
  const onChange = async (e) => { if (e.target.id === "bkLogFilter") { evs = await api("/api/me/book-events?limit=1000" + (e.target.value ? "&book=" + e.target.value : "")); mountLog(); } };
  main.addEventListener("click", onClick); main.addEventListener("change", onChange);
  cleanup = () => { main.removeEventListener("click", onClick); main.removeEventListener("change", onChange); };
  render();
}

/* ---------- Файлын сан ---------- */
async function files() {
  const [lib, st, plans] = await Promise.all([loadLibrary(), api("/api/me/storage"), api("/api/storage/plans", { token: null })]);
  const perMB = Math.min(...plans.plans.map((p) => p.price_month / p.mb));
  const pct = Math.min(100, (st.used / st.quota) * 100);
  main.innerHTML = panel(`<div class="panel-head"><h2>Файлын сан</h2><span class="chip ${pct > 85 ? "chip-coral" : "chip-gold"}">${fmtSize(st.used)} / ${fmtSize(st.quota)}</span></div>
      <div class="meter"><i id="fMeter"></i></div>
      <p class="muted small">Үнэгүй ${plans.free_label}${st.plan_active ? ` + ${esc(String(st.plan_mb >= 1024 ? st.plan_mb / 1024 + "GB" : st.plan_mb + "MB"))} багц (${fmtDate(st.expires_at)} хүртэл)` : ""}.
      Зураг автоматаар <b>WebP</b> болж ${plans.max_image_side}px хүртэл багасна, видео <b>WebM</b>, Word/PowerPoint <b>PDF (3D ном)</b> болно.</p>
      <div class="dropzone" id="drop" tabindex="0"><strong>📤 Файлаа энд чирж оруулах эсвэл дарж сонгох</strong><span class="muted small">Видео, PDF, Word, PowerPoint, зураг, аудио</span>
        <select id="vis" style="width:auto" onclick="event.stopPropagation()"><option value="private">🔒 Хаалттай (төлбөртэй хичээлд)</option><option value="public">🌐 Нээлттэй (профайл зураг г.м)</option></select>
        <input type="file" id="fileIn" multiple hidden></div>
      <div id="uploads" class="list" style="margin-top:12px"></div>`) +
    panel(`<div class="panel-head"><h2>Миний файлууд</h2><span class="muted small" id="fCount">${lib.files.length} файл</span></div>
      <div class="fl-bar">
        <div class="find-tags" id="fType" style="justify-content:flex-start">${FILE_TYPES.map(([k, t]) => `<button type="button" data-ft="${k}" aria-pressed="${k === fstate.type}">${t} <i>${lib.files.filter((f) => k === "all" || fileType(f) === k).length}</i></button>`).join("")}</div>
        <div class="fl-tools"><label class="rail-search">${ico("search", 16)}<input type="search" id="fSearch" placeholder="Файлын нэрээр хайх…" aria-label="Файл хайх" value="${esc(fstate.q)}"></label>
          <select id="fSort" aria-label="Эрэмбэлэх"><option value="new">Шинэ нь эхэнд</option><option value="old">Хуучин нь эхэнд</option><option value="big">Том нь эхэнд</option><option value="name">Нэрээр</option></select>
          <span class="fl-view" role="tablist"><button type="button" data-fv="tree" aria-pressed="${fstate.view === "tree"}" title="Модон бүтэц">🌳 Мод</button><button type="button" data-fv="grid" aria-pressed="${fstate.view === "grid"}" title="Карт">▦ Карт</button></span>
          <select id="fUse" aria-label="Ашиглалт"><option value="">Бүгд</option><option value="used">Хичээлд ашигласан</option><option value="unused">Ашиглаагүй</option></select></div></div>
      <div class="file-grid" id="fGrid"></div>`, 1) +
    panel(`<div class="panel-head"><h2>Багтаамж нэмэх</h2><span class="muted small">сарын төлбөр</span></div>
      <div class="plans">${plans.plans.map((p, i) => `<div class="plan ${i === 1 ? "sel" : ""}" data-mb="${p.mb}"><b>${esc(p.label)}</b><span>${money(p.price_month)}</span><small class="muted">/ сар</small></div>`).join("")}</div>
      <div class="form-row" style="margin-top:14px"><label>Хугацаа<select id="months">${plans.months.map((m) => `<option value="${m}">${m} сар</option>`).join("")}</select></label>
      <button class="btn btn-gold" id="buyPlan" style="align-self:end">Худалдаж авах · <span id="planTotal"></span></button></div>`, 2);
  requestAnimationFrame(() => ($("#fMeter").style.width = pct + "%"));
  $("#fSort").value = fstate.sort; $("#fUse").value = fstate.use;
  // Цэгцлэх: төрөл, хайлт, эрэмбэ, хичээлд ашигласан эсэх.
  const [usage, myBooks] = await Promise.all([api("/api/me/files/usage").catch(() => ({})), api("/api/me/books").catch(() => [])]);
  const drawFiles = () => {
    const q = fstate.q.toLowerCase();
    let fl = lib.files.filter((f) => (fstate.type === "all" || fileType(f) === fstate.type) && (!q || f.original_name.toLowerCase().includes(q)) &&
      (!fstate.use || (fstate.use === "used") === !!usage[f.path]?.length));
    const by = { new: (a, b) => new Date(b.mod_time) - new Date(a.mod_time), old: (a, b) => new Date(a.mod_time) - new Date(b.mod_time), big: (a, b) => b.size - a.size, name: (a, b) => a.original_name.localeCompare(b.original_name) };
    fl.sort(by[fstate.sort]);
    $("#fCount").textContent = `${fl.length} / ${lib.files.length} файл`;
    const grid = $("#fGrid"); grid.classList.toggle("as-tree", fstate.view === "tree");
    if (fstate.view === "tree") { grid.innerHTML = filesTree(fl, usage, q || fstate.type !== "all" ? [] : myBooks); return; }
    grid.innerHTML = fl.map((f) => fileCard(f, perMB, usage[f.path])).join("") || `<div class="empty" style="grid-column:1/-1">Тохирох файл алга</div>`;
    hydrateBooks($("#fGrid"));
  };
  drawFiles();
  $("#fType").onclick = (e) => { const b = e.target.closest("[data-ft]"); if (!b) return; fstate.type = b.dataset.ft; $$("#fType [data-ft]").forEach((x) => x.setAttribute("aria-pressed", x === b)); drawFiles(); };
  $("#fSearch").oninput = (e) => { fstate.q = e.target.value; drawFiles(); };
  $("#fSort").onchange = (e) => { fstate.sort = e.target.value; drawFiles(); };
  $("#fUse").onchange = (e) => { fstate.use = e.target.value; drawFiles(); };
  $$("[data-fv]").forEach((b) => (b.onclick = () => { fstate.view = b.dataset.fv; try { localStorage.setItem("sg_files_view", fstate.view); } catch {} $$("[data-fv]").forEach((x) => x.setAttribute("aria-pressed", x === b)); drawFiles(); }));

  // Багц сонгох
  const total = () => { const p = plans.plans.find((x) => x.mb == $(".plan.sel").dataset.mb); $("#planTotal").textContent = money(p.price_month * +$("#months").value); };
  $$(".plan").forEach((p) => (p.onclick = () => { $$(".plan").forEach((x) => x.classList.toggle("sel", x === p)); total(); }));
  $("#months").onchange = total; total();
  $("#buyPlan").onclick = async () => {
    try {
      const d = await api("/api/me/storage/purchase", { method: "POST", body: { mb: +$(".plan.sel").dataset.mb, months: +$("#months").value } });
      const p = plans.plans.find((x) => x.mb == $(".plan.sel").dataset.mb);
      window.SG.pay(d.order, d.payment, `Файлын сан · ${p?.label || ""} · ${$("#months").value} сар`, () => { toast("💾 Багтаамж нэмэгдлээ"); celebrate(); files(); }, { name: "Багтаамж" });
    } catch (e) { toast(e.message, true); }
  };

  // Upload
  const drop = $("#drop"), input = $("#fileIn");
  drop.onclick = () => input.click();
  drop.ondragover = (e) => { e.preventDefault(); drop.classList.add("over"); };
  drop.ondragleave = () => drop.classList.remove("over");
  drop.ondrop = (e) => { e.preventDefault(); drop.classList.remove("over"); upload([...e.dataTransfer.files]); };
  input.onchange = () => upload([...input.files]);
  async function upload(list) {
    for (const file of list) {
      const row = document.createElement("div");
      row.className = "row"; row.innerHTML = `<div class="grow"><strong>${esc(file.name)}</strong><div class="progress"><i></i></div></div><span class="muted small">${fmtSize(file.size)}</span>`;
      $("#uploads").append(row);
      try {
        await new Promise((res, rej) => {
          const xhr = new XMLHttpRequest(), fd = new FormData();
          fd.append("file", file);
          xhr.open("POST", "/api/me/files?visibility=" + $("#vis").value);
          xhr.setRequestHeader("Authorization", "Bearer " + Auth.token);
          xhr.upload.onprogress = (e) => e.lengthComputable && ($("i", row).style.width = (e.loaded / e.total) * 100 + "%");
          xhr.onload = () => (xhr.status < 300 ? res() : rej(new Error(JSON.parse(xhr.responseText || "{}").error || "Алдаа")));
          xhr.onerror = () => rej(new Error("Сүлжээний алдаа"));
          xhr.send(fd);
        });
        $("i", row).style.width = "100%"; row.style.opacity = ".6";
      } catch (e) { row.querySelector(".grow").insertAdjacentHTML("beforeend", `<span class="form-error small">${esc(e.message)}</span>`); }
    }
    setTimeout(files, 600);
  }
  main.onclick = async (e) => {
    const del = e.target.closest("[data-del]"), cp = e.target.closest("[data-copy]");
    if (cp) { navigator.clipboard.writeText(cp.dataset.copy); toast("Зам хуулагдлаа — хичээлд холбоно уу"); }
    if (del && confirm("Устгах уу?")) { try { await api(`/api/me/files/${del.dataset.del}`, { method: "DELETE" }); toast("Устгагдлаа"); files(); } catch (x) { toast(x.message, true); } }
  };
  // Боловсруулж буй файл байвал үе үе шинэчилнэ.
  const t = lib.files.some((f) => f.status === "processing") ? setInterval(async () => {
    const l = await api("/api/me/files").catch(() => null);
    if (l && !l.files.some((f) => f.status === "processing")) { clearInterval(t); files(); }
  }, 5000) : null;
  cleanup = () => { clearInterval(t); main.onclick = null; };
}

const FILE_TYPES = [["all", "Бүгд"], ["video", "🎬 Видео"], ["image", "🖼 Зураг"], ["audio", "🎧 Дуу"], ["doc", "📄 Баримт"], ["other", "📦 Бусад"]];
const fstate = { type: "all", q: "", sort: "new", use: "", view: (() => { try { return localStorage.getItem("sg_files_view") || "tree"; } catch { return "tree"; } })() };
const FICON = { video: "🎬", image: "🖼", audio: "🎧", doc: "📄", other: "📦" };
const FGROUP = { video: "Видео", image: "Зураг", audio: "Дуу", doc: "Баримт", other: "Бусад" };
// Модон бүтцийн нэг файл: картын үйлдлүүдтэй ижил data-* тул main.onclick ажиллана.
const treeFile = (f) => `<li class="tf"><span class="tf-ico">${FICON[fileType(f)]}</span>
  <span class="tf-name" title="${esc(f.original_name)}">${esc(f.original_name)}</span>
  <span class="tf-meta">${fmtSize(f.size)} · ${f.visibility === "public" ? "🌐" : "🔒"}</span>
  <span class="tf-acts"><button class="icon-btn" data-copy="${esc(f.path)}" title="Зам хуулах">🔗</button>${f.status === "ready" ? `<a class="icon-btn" href="${esc(f.url)}" target="_blank" rel="noopener" title="Нээх">↗</a>` : ""}<button class="icon-btn" data-del="${esc(f.visibility)}/${encodeURIComponent(f.name)}" title="Устгах">🗑</button></span></li>`;
const treeNode = (icon, label, count, inner, open) => `<li class="tn"><details ${open ? "open" : ""}><summary><span class="tn-ico">${icon}</span><span class="tn-name">${esc(label)}</span><span class="tn-count">${count}</span></summary><ul>${inner}</ul></details></li>`;
// Файлын сангийн модон бүтэц: Сургалт → Бүлэг → Хичээл → Файл, Номууд, Ашиглаагүй файлууд (төрлөөр).
function filesTree(files, usage, books) {
  const courses = new Map();
  const used = new Set(), few = files.length <= 30; // цөөн бол бүгдийг дэлгэнэ
  for (const f of files) {
    for (const u of usage[f.path] || []) {
      if (!u.course_id) continue;
      used.add(f.path);
      let c = courses.get(u.course_id);
      if (!c) courses.set(u.course_id, (c = { title: u.course, sections: new Map() }));
      const sk = u.section || "";
      let sec = c.sections.get(sk);
      if (!sec) c.sections.set(sk, (sec = new Map()));
      let l = sec.get(u.lesson_id);
      if (!l) sec.set(u.lesson_id, (l = { title: u.lesson, pos: u.position || 0, files: [] }));
      if (!l.files.includes(f)) l.files.push(f);
    }
  }
  const lessonsHTML = (sec) => [...sec.values()].sort((a, b) => a.pos - b.pos)
    .map((l) => treeNode("📝", l.title, l.files.length, l.files.map(treeFile).join(""), few)).join("");
  const courseHTML = [...courses.values()].map((c) => {
    const n = [...c.sections.values()].reduce((k, s) => k + [...s.values()].reduce((m, l) => m + l.files.length, 0), 0);
    const inner = [...c.sections.entries()].map(([name, sec]) => name
      ? treeNode("📂", name, [...sec.values()].reduce((m, l) => m + l.files.length, 0), lessonsHTML(sec), true)
      : lessonsHTML(sec)).join("");
    return treeNode("📘", c.title, n, inner, true);
  }).join("");
  const unused = files.filter((f) => !used.has(f.path));
  const byType = Object.keys(FGROUP).map((t) => [t, unused.filter((f) => fileType(f) === t)]).filter(([, fs]) => fs.length);
  const unusedHTML = byType.map(([t, fs]) => treeNode(FICON[t], FGROUP[t], fs.length, fs.map(treeFile).join(""), few)).join("");
  const bookHTML = books.map((b) => `<li class="tf"><span class="tf-ico">📗</span><span class="tf-name">${esc(b.title)}</span><span class="tf-meta">${b.pages} хуудас · 🔒 хамгаалагдсан</span><span class="tf-acts">${b.published ? `<a class="icon-btn" href="/b/${esc(b.id)}" target="_blank" rel="noopener" title="Харах">↗</a>` : ""}</span></li>`).join("");
  return `<ul class="ftree">${treeNode("🗂", "Бүх файл", files.length, [
    courseHTML ? treeNode("📚", "Сургалтууд", used.size, courseHTML, true) : "",
    books.length ? treeNode("📕", "Номууд", books.length, bookHTML, false) : "",
    unused.length ? treeNode("🗃", "Ашиглаагүй файлууд", unused.length, unusedHTML, true) : "",
  ].join("") || `<li class="muted small">Файл алга</li>`, true)}</ul>`;
}
const fileType = (f) => { const t = f.content_type || "", e = f.name.slice(f.name.lastIndexOf(".")).toLowerCase();
  return t.startsWith("video/") ? "video" : t.startsWith("image/") ? "image" : t.startsWith("audio/") ? "audio" : [".pdf", ".doc", ".docx", ".ppt", ".pptx", ".xls", ".xlsx", ".odt", ".odp", ".rtf", ".txt", ".epub"].includes(e) ? "doc" : "other"; };
function fileCard(f, perMB, used) {
  const ext = f.name.slice(f.name.lastIndexOf(".")).toLowerCase();
  let thumb;
  if (f.status === "processing") thumb = `<span class="status-processing small">${ext === ".pdf" ? "PDF болгож" : "WebM болгож"} байна…</span>`;
  else if (f.status === "failed") thumb = `<span class="form-error small">Хөрвүүлэлт амжилтгүй</span>`;
  else if (f.content_type.startsWith("image/")) thumb = `<img src="${esc(f.url)}" alt="" loading="lazy">`;
  else if (f.content_type.startsWith("video/")) thumb = `<video src="${esc(f.url)}#t=1" preload="metadata" muted></video>`;
  else if (ext === ".pdf") thumb = book3dHTML(f.url, f.original_name);
  else thumb = `<span style="font-size:2.4rem">📄</span>`;
  const cost = Math.max(1, Math.round((f.size / (1 << 20)) * perMB));
  return `<div class="file"><div class="file-thumb">${thumb}</div><div class="file-name">${esc(f.original_name)}</div>
    <span class="muted small">${fmtSize(f.size)} · ≈${money(cost)}/сар · ${f.visibility === "public" ? "🌐" : "🔒"} · ${fmtDate(f.mod_time)}</span>
    <span class="file-use ${used?.length ? "on" : ""}" title="${esc((used || []).map((u) => u.course + " › " + u.lesson).join("\n"))}">${used?.length ? `📌 ${esc(used[0].lesson)}${used.length > 1 ? ` +${used.length - 1}` : ""}` : "Ашиглаагүй"}</span>
    <div class="file-actions"><button class="btn btn-ghost btn-sm" data-copy="${esc(f.path)}" title="Хичээлд холбох зам">🔗</button>
    ${f.status === "ready" ? `<a class="btn btn-ghost btn-sm" href="${esc(f.url)}" target="_blank" rel="noopener">↗</a>` : ""}
    <button class="btn btn-danger btn-sm" data-del="${esc(f.visibility)}/${encodeURIComponent(f.name)}">🗑</button></div></div>`;
}

/* ---------- Шууд хичээл (Google Meet) ---------- */
async function live() {
  const [meetings, list] = await Promise.all([api("/api/me/meetings"), api("/api/me/courses")]);
  const fresh = await api("/api/me"); me = fresh;
  const dt = new Date(Date.now() + 3600e3); dt.setMinutes(0, 0, 0);
  const local = new Date(dt - dt.getTimezoneOffset() * 60e3).toISOString().slice(0, 16);
  main.innerHTML = panel(`<div class="panel-head"><h2>Google Meet</h2>${me.meet_connected ? `<span class="chip chip-teal">✓ Холбогдсон</span>` : `<span class="chip">Холбогдоогүй</span>`}</div>
      <p class="muted">Холбосноор шууд хичээл товлоход болон чатаас нэг товчоор Google Meet холбоос автоматаар үүснэ. Холбоос таны Google Calendar-т хадгалагдана.</p>
      ${me.meet_connected ? `<button class="btn btn-ghost btn-sm" id="meetOff">Салгах</button>` : `<button class="btn btn-gold" id="meetOn">📹 Google Meet холбох</button>`}`) +
    panel(`<h2>Шууд хичээл товлох</h2><form class="form" id="meetForm">
      <label>Сэдэв<input name="title" required maxlength="200" placeholder="ЭЕШ давтлага — Логарифм"></label>
      <div class="form-row"><label>Эхлэх цаг<input name="start" type="datetime-local" value="${local}" required></label>
      <label>Үргэлжлэх (мин)<input name="dur" type="number" min="10" max="480" value="60"></label></div>
      <label>Сургалт (элссэн суралцагчид харна)<select name="course"><option value="">— Ерөнхий —</option>${list.map((c) => `<option value="${c.id}">${esc(c.title)}</option>`).join("")}</select></label>
      <div class="form-row"><label>Үнэ (₮, 0 = үнэгүй)<input name="price" type="number" min="0" step="500" value="0"><small class="muted">Төлбөртэй бол зөвхөн худалдаж авсан хүн Meet холбоосыг харна. Сургалттай холбоно.</small></label>
        <label class="check" style="align-self:center"><input type="checkbox" name="members_free"> Сургалтад элссэн суралцагчдад үнэгүй</label></div>
      <button class="btn btn-gold" ${me.meet_connected ? "" : "disabled"}>Товлох</button></form>`, 1) +
    panel(`<h2>Удахгүй болох</h2><div class="meet-list" id="meetMine">${meetings.map((m) => `<div class="meet-item"><time>${fmtDate(m.starts_at)}</time><span style="flex:1">${esc(m.title)} · ${m.duration_min} мин
        ${m.price ? `<span class="chip chip-amber">${money(m.price)}${m.members_free ? " · элссэнд үнэгүй" : ""}</span> <span class="muted small">${m.buyers || 0} худалдаж авсан</span>` : `<span class="chip">Үнэгүй</span>`}</span>
        ${m.course_id ? `<button class="btn btn-ghost btn-sm" data-meet-price="${esc(m.id)}">${ico("money", 15)}Үнэ</button>` : ""}<a class="btn btn-glass btn-sm" href="${esc(m.meet_url)}" target="_blank" rel="noopener">${ico("live", 15)}Нээх</a></div>`).join("") || `<div class="empty">Товлосон хичээл алга</div>`}</div>`, 2);
  $("#meetOn")?.addEventListener("click", async () => { try { const d = await api("/api/me/meet/connect", { method: "POST" }); location.href = d.url; } catch (e) { toast(e.message, true); } });
  $("#meetOff")?.addEventListener("click", async () => { await api("/api/me/meet", { method: "DELETE" }); live(); });
  const f = $("#meetForm");
  f.onsubmit = async (e) => {
    e.preventDefault();
    try {
      await api("/api/me/meetings", { method: "POST", body: { title: f.title.value, starts_at: new Date(f.start.value).toISOString(), duration_min: +f.dur.value, course_id: f.course.value,
        price: +f.price.value || 0, members_free: f.members_free.checked } });
      toast("📹 Meet үүслээ"); live();
    } catch (err) { toast(err.message, true); }
  };
  // Үнэ засах: аль хэдийн худалдаж авсан хүмүүсийн эрх хэвээр.
  $("#meetMine")?.addEventListener("click", (e) => {
    const b = e.target.closest("[data-meet-price]"); if (!b) return;
    const m = meetings.find((x) => x.id === b.dataset.meetPrice); if (!m) return;
    document.body.insertAdjacentHTML("beforeend", `<div class="modal" id="mpModal"><div class="modal-card"><button class="icon-btn modal-x" data-close aria-label="Хаах">${ico("x", 18)}</button>
      <h3 class="h3">Шууд хичээлийн үнэ</h3><p class="muted small">${esc(m.title)} · ${fmtDate(m.starts_at)}</p>
      <form class="form" id="mpForm"><label>Үнэ (₮, 0 = үнэгүй)<input name="price" type="number" min="0" step="500" value="${m.price || 0}"></label>
        <label class="check"><input type="checkbox" name="members_free" ${m.members_free ? "checked" : ""}> Сургалтад элссэн суралцагчдад үнэгүй</label>
        <p class="muted small" style="margin:0">Аль хэдийн худалдаж авсан ${m.buyers || 0} хүний эрх хэвээр үлдэнэ.</p><p class="form-error" role="alert"></p>
        <div class="hero-cta" style="margin:0;justify-content:flex-end"><button type="button" class="btn btn-ghost" data-close>Болих</button><button class="btn btn-gold">Хадгалах</button></div></form></div></div>`);
    const md = $("#mpModal"), f2 = $("#mpForm", md);
    SG.openModal(md);
    const close = () => { SG.closeModal(md); setTimeout(() => md.remove(), 300); };
    md.addEventListener("click", (ev) => { if (ev.target === md || ev.target.closest("[data-close]")) { ev.preventDefault(); close(); } });
    f2.onsubmit = async (ev) => {
      ev.preventDefault();
      try { await api(`/api/me/meetings/${m.id}`, { method: "PUT", body: { price: +f2.price.value || 0, members_free: f2.members_free.checked } }); toast("Үнэ хадгалагдлаа ✓"); close(); live(); }
      catch (err) { $(".form-error", f2).textContent = err.message; }
    };
  });
}

/* ---------- Чат (inbox) ---------- */
async function chat(openId) {
  const convs = await api("/api/me/conversations");
  main.innerHTML = `<div class="inbox panel" style="padding:0">
    <div class="inbox-list" id="ibList">${convs.map(convItem).join("") || `<div class="empty" style="margin:14px">Одоогоор чат алга. Профайлаа түгээгээрэй!</div>`}</div>
    <div class="inbox-thread"><div class="chat-head" id="ibHead"><span class="muted">Яриа сонгоно уу</span></div>
      <div class="chat-body"><ol class="chat-msgs" id="ibMsgs"></ol></div>
      <form class="chat-input" id="ibForm" hidden><button type="button" id="ibMeet" data-plus>📹 Google Meet үүсгээд илгээх</button></form></div></div>`;
  let cur = null, curGroup = false;
  const list = $("#ibList"), msgs = $("#ibMsgs"), form = $("#ibForm"), body = msgs.parentElement;
  const thread = new SG.ChatThread({ ol: msgs, body, form, token: () => SG.Auth.token, convId: () => cur, role: () => "teacher", group: () => curGroup });
  const open = async (id) => {
    cur = id;
    $$(".inbox-item", list).forEach((x) => x.classList.toggle("active", x.dataset.id === id));
    $(`.inbox-item[data-id="${id}"]`)?.classList.remove("unread");
    const d = await api(`/api/chat/${id}/messages`);
    curGroup = d.conversation.kind === "group";
    $("#ibHead").innerHTML = `<strong>${esc(d.conversation.visitor_name)}</strong><span class="muted small">${curGroup ? "Бүлэг чат" : d.conversation.user_id ? "Бүртгэлтэй" : "Зочин"}</span>`;
    form.hidden = false; thread.set(d); form.body.focus();
    history.replaceState(null, "", "#chat=" + id);
  };
  list.onclick = (e) => { const it = e.target.closest(".inbox-item"); if (it) open(it.dataset.id); };
  form.addEventListener("click", async (e) => {
    if (!e.target.closest("#ibMeet") || !cur) return;
    try { const d = await api(`/api/chat/${cur}/meet`, { method: "POST" }); append(d.message); toast("📹 Meet холбоос илгээгдлээ"); }
    catch (x) { toast(x.message, true); if (x.status === 428) location.hash = "live"; }
  });
  const append = (m) => thread.append(m);
  const handler = async (d) => {
    if (["reaction", "read", "typing"].includes(d.type)) { thread.event(d); return; }
    if (d.type !== "message") return;
    const m = d.message;
    if (m.conversation_id === cur) append(m);
    let it = $(`.inbox-item[data-id="${m.conversation_id}"]`);
    if (!it) { const fresh = await api("/api/me/conversations"); list.innerHTML = fresh.map(convItem).join(""); it = $(`.inbox-item[data-id="${m.conversation_id}"]`); }
    if (it) { $("small", it).textContent = m.body; list.prepend(it); if (m.conversation_id !== cur) it.classList.add("unread"); it.animate([{ background: "rgba(31,60,143,.18)" }, { background: "transparent" }], { duration: 1200 }); }
  };
  Live.on(handler);
  cleanup = () => Live.handlers.delete(handler);
  if (openId) open(openId);
}

/* ---------- Профайл ---------- */
/* ---------- Суралцагчид: хэн элссэн, юу авсан, шууд чат ---------- */
const studentRow = (r) => {
  const total = r.courses.length, lessons = r.courses.reduce((n, c) => n + c.lessons.length, 0);
  const since = r.courses.map((c) => c.enrolled_at).filter(Boolean).sort()[0];
  return `<div class="item student" data-uid="${esc(r.user.id)}" data-conv="${esc(r.conv_id || "")}">${avatar(r.user, "avatar-sm")}
    <span class="grow"><strong>${esc(r.user.display_name)}</strong><small>${r.courses.map((c) => `${esc(c.course_title)}: ${c.enrolled ? "элссэн" : ""}${c.enrolled && c.lessons.length ? ", " : ""}${c.lessons.length ? c.lessons.length + " хичээл авсан" : ""}`).join(" · ")}${since ? " · " + fmtDay(since) : ""}</small></span>
    ${total > 1 ? `<span class="chip">${total} сургалт</span>` : ""}${lessons ? `<span class="chip chip-gold">${lessons} хичээл</span>` : ""}
    <button class="btn btn-gold btn-sm" data-chat-student>${ico("chat", 16)}Чат</button></div>`;
};
// Хяналтын самбарын нарийн баганад: зураг, нэр, товч мэдээлэл, чатын товч (дарахад дэлгэрэнгүй).
const studentMini = (r) => {
  const lessons = r.courses.reduce((n, c) => n + c.lessons.length, 0);
  return `<div class="anx-person student" data-uid="${esc(r.user.id)}" data-conv="${esc(r.conv_id || "")}" data-an-detail="${esc(r.user.id)}" role="button" tabindex="0">${avatar(r.user, "avatar-sm")}
    <span class="grow"><strong>${esc(r.user.display_name)}</strong><small>${r.courses.length} сургалт${lessons ? ` · ${lessons} хичээл авсан` : ""}</small></span>
    <button class="icon-btn" data-chat-student title="Чатлах" aria-label="${esc(r.user.display_name)}-тэй чатлах">${ico("chat", 17)}</button></div>`;
};
// Суралцагчтай чат: өмнө нь яриа байвал нээнэ, үгүй бол шинээр эхлүүлнэ.
async function chatWithStudent(el) {
  const b = el.querySelector("[data-chat-student]"); if (b) b.disabled = true;
  try {
    let id = el.dataset.conv;
    if (!id) { const d = await api(`/api/me/students/${el.dataset.uid}/chat`, { method: "POST" }); id = d.conversation.id; el.dataset.conv = id; Live.reconnect?.(); }
    window.openRailChat?.(id);
  } catch (e) { toast(e.message, true); } finally { if (b) b.disabled = false; }
}
document.addEventListener("click", (e) => { const b = e.target.closest("[data-chat-student]"); if (b) chatWithStudent(b.closest(".student")); });

async function students() {
  const [rows, courseList] = await Promise.all([api("/api/me/students"), api("/api/me/courses")]);
  const st = { course: "", days: 30 };
  let data = null, labels = {}, timer = null;
  const dur = (sec) => sec < 60 ? `${sec} сек` : sec < 3600 ? `${Math.round(sec / 60)} мин` : `${Math.floor(sec / 3600)} ц ${String(Math.round(sec % 3600 / 60)).padStart(2, "0")} мин`;
  const RISK = { ok: ["Хэвийн", "ok"], watch: ["Анхаарах", "watch"], risk: ["Эрсдэлтэй", "risk"] };
  const bar = (v) => `<span class="an-bar"><i style="width:${Math.max(0, Math.min(100, v))}%;background:${v >= 70 ? "var(--teal)" : v >= 40 ? "var(--accent)" : "var(--coral)"}"></i></span><b>${v}%</b>`;
  const chart = (daily) => {
    const max = Math.max(...daily.map((d) => d.active_sec + d.inactive_sec)), w = 100 / daily.length;
    if (!max) return `<p class="muted small an-empty">Энэ хугацаанд хичээл үзсэн идэвх бүртгэгдээгүй байна.</p>`;
    return `<svg class="an-chart" viewBox="0 0 100 40" preserveAspectRatio="none" role="img" aria-label="Өдөр бүрийн идэвхтэй ба идэвхгүй минут">${daily.map((d, i) => {
      const ha = d.active_sec / max * 38, hi = d.inactive_sec / max * 38;
      return `<rect x="${i * w + w * 0.15}" y="${40 - ha}" width="${w * 0.7}" height="${ha}" fill="#1f3c8f"><title>${d.day}: идэвхтэй ${dur(d.active_sec)}</title></rect><rect x="${i * w + w * 0.15}" y="${40 - ha - hi}" width="${w * 0.7}" height="${hi}" fill="#eaa02e" opacity=".75"><title>${d.day}: идэвхгүй ${dur(d.inactive_sec)}</title></rect>`;
    }).join("")}</svg><div class="an-legend"><span><i style="background:#1f3c8f"></i>Идэвхтэй</span><span><i style="background:#eaa02e"></i>Идэвхгүй, өөр цонхонд</span><span class="muted">${daily[0]?.day || ""} — ${daily[daily.length - 1]?.day || ""}</span></div>`;
  };
  const evRow = (e, who = true) => `<tr><td>${fmtDate(e.at)}</td>${who ? `<td>${esc(e.user_name || "")}</td>` : ""}<td>${esc(labels[e.type] || e.type)}</td><td class="muted">${esc(e.detail || "")}</td></tr>`;
  // Үйл явдлын лог: 20-оор ачаалж, доош гүйлгэхэд дараагийн 20-ыг (at, id курсороор) нэмнэ.
  // 30 сек тутмын шинэчлэлд DOM-оо хадгалж, зөвхөн шинэ үйл явдлыг дээр нь нэмнэ.
  const PAGE = 20;
  const makeFeed = (extra, who = true) => {
    const box = document.createElement("div"); box.className = "bk-log ev-feed";
    box.innerHTML = `<table class="tbl"><thead><tr><th>Цаг</th>${who ? "<th>Хэн</th>" : ""}<th>Үйл явдал</th><th>Дэлгэрэнгүй</th></tr></thead><tbody></tbody></table><div class="ev-more muted small" role="status"></div>`;
    const tb = $("tbody", box), more = $(".ev-more", box), seen = new Set(), url = (c = "") => `/api/me/analytics/events?${qs()}${extra}&limit=${PAGE}${c}`;
    const row = (e) => evRow(e, who);
    let cur = "", busy = false, done = false;
    const status = () => { more.textContent = done ? (seen.size > PAGE ? "Бүгдийг харууллаа" : "") : "Доош гүйлгэвэл цааш ачаална…"; if (done && !seen.size) tb.innerHTML = `<tr><td colspan="4" class="muted">Одоогоор алга.</td></tr>`; };
    const add = (evs, where) => { const fresh = evs.filter((e) => !seen.has(e.id)); fresh.forEach((e) => seen.add(e.id)); if (fresh.length && tb.querySelector("td[colspan]")) tb.innerHTML = ""; tb.insertAdjacentHTML(where, fresh.map(row).join("")); };
    const next = async () => {
      if (busy || done) return; busy = true; more.textContent = "Ачаалж байна…";
      try {
        const d = await api(url(cur)); Object.assign(labels, d.labels || {});
        add(d.events, "beforeend"); done = !d.more;
        cur = d.more ? `&before_at=${encodeURIComponent(d.next_at)}&before_id=${encodeURIComponent(d.next_id)}` : "";
      } catch { busy = false; more.textContent = "Ачаалж чадсангүй — дарж дахин оролдоно уу"; return; }
      busy = false; status();
      if (!done && box.isConnected && box.scrollHeight <= box.clientHeight + 60) next(); // гүйлгэх зай үүсэх хүртэл
    };
    const refresh = async () => { // шинэ үйл явдлыг дээр нь нэмнэ
      try { const d = await api(url()); add(d.events, "afterbegin"); status(); } catch {}
    };
    new IntersectionObserver((es) => { if (es.some((e) => e.isIntersecting)) next(); }, { root: box, rootMargin: "160px" }).observe(more);
    more.addEventListener("click", next);
    next();
    return { box, refresh, key: qs() + extra };
  };
  let feed = null;
  const mountFeed = () => {
    if (!feed || feed.key !== qs()) feed = makeFeed(""); else feed.refresh();
    const y = feed.box.scrollTop; $("#anFeed")?.replaceWith(feed.box); feed.box.scrollTop = y;
  };
  // Хүснэгтэд: суралцсан оноог юунаас бүрдсэнийг нэг мөрөөр.
  const learnHint = (x) => {
    const p = [];
    if (x.quiz_total) p.push(`✅ ${x.quiz_accuracy}% (${x.quiz_total})`);
    if (x.quiz_lessons) p.push(`🎯 ${x.quiz_mastered}/${x.quiz_lessons} хичээл`);
    if (x.video_clips) p.push(`▶ ${x.video_coverage}%`);
    if (x.reflections) p.push(`✍️ ${x.reflections}`);
    if (x.streak_days > 1) p.push(`🔥 ${x.streak_days} өдөр`);
    return p.join(" · ") || "Мэдээлэл алга";
  };
  // Видеоны үзэлтийн heatmap: хэсэг бүрийг хэдэн удаа үзсэнээр өнгөөр ялгана (саарал = огт үзээгүй).
  const heatmap = (w) => {
    if (!w.buckets?.length) return "";
    const max = Math.max(1, ...w.buckets);
    return `<div class="vh-row"><small class="vh-title">${esc(w.lesson || "")}</small><div class="vh-bar">${w.buckets.map((n, i) => {
      const pct = n / max, bg = n === 0 ? "#e1e6f0" : `rgba(31,60,143,${0.25 + pct * 0.65})`;
      const mm = Math.floor(i * 10 / 60), ss = String(i * 10 % 60).padStart(2, "0");
      return `<i style="background:${bg}" title="${mm}:${ss} — ${n ? n + " удаа үзсэн" : "алгассан"}"></i>`;
    }).join("")}</div></div>`;
  };
  const qs = () => `course=${encodeURIComponent(st.course)}&days=${st.days}`;
  // Өдөр бүрийн идэвх: тэнхлэгтэй давхар багана (идэвхтэй — хар хөх, идэвхгүй — улбар шар).
  const chart2 = (daily) => {
    const tot = (d) => d.active_sec + d.inactive_sec, max = Math.max(0, ...daily.map(tot));
    if (!max) return `<div class="anx-empty">${ico("chart", 28)}<p>Энэ хугацаанд хичээл үзсэн идэвх бүртгэгдээгүй байна.</p></div>`;
    const top = [30, 60, 120, 300, 600, 1200, 1800, 3600, 7200, 10800, 18000, 36000, 86400].find((x) => x >= max) || max;
    const sumA = daily.reduce((a, d) => a + d.active_sec, 0), best = daily.reduce((m, d) => (tot(d) > tot(m) ? d : m), daily[0]);
    const lbl = (day) => (day || "").slice(5).replace("-", "/"), mid = daily[Math.floor((daily.length - 1) / 2)];
    return `<div class="anx-chart-sum"><span><b>${dur(sumA)}</b>идэвхтэй</span><span><b>${dur(Math.round(sumA / daily.length))}</b>өдрийн дундаж</span><span><b>${lbl(best.day)}</b>хамгийн идэвхтэй өдөр</span></div>
      <div class="anx-chart" role="img" aria-label="Өдөр бүрийн идэвхтэй ба идэвхгүй хугацаа">
        <div class="anx-y" aria-hidden="true"><span>${dur(top)}</span><span>${dur(Math.round(top / 2))}</span><span>0</span></div>
        <div class="anx-plot"><div class="anx-cols">${daily.map((d) => `<div class="anx-col" title="${esc(d.day)}: идэвхтэй ${dur(d.active_sec)}, идэвхгүй ${dur(d.inactive_sec)}"><i class="ia" style="height:${d.inactive_sec / top * 100}%"></i><i class="ac" style="height:${d.active_sec / top * 100}%"></i></div>`).join("")}</div></div>
        <div class="anx-x" aria-hidden="true"><span>${lbl(daily[0]?.day)}</span><span>${lbl(mid?.day)}</span><span>${lbl(daily[daily.length - 1]?.day)}</span></div></div>`;
  };
  const render = () => {
    const t = data.totals, list = data.students, course = courseList.find((c) => c.id === st.course);
    const pct = (v) => Math.max(0, Math.min(100, +v || 0));
    const meter = (v) => `<span class="anx-meter"><i style="width:${pct(v)}%;background:${v >= 70 ? "var(--teal)" : v >= 40 ? "var(--accent)" : "var(--coral)"}"></i></span>`;
    const kpi = (ic, label, value, sub) => `<div class="anx-kpi"><span class="anx-ic">${ico(ic, 18)}</span><small>${label}</small><b>${value}</b><div class="anx-sub">${sub || ""}</div></div>`;
    const stat = (label, value, cls = "") => `<div class="anx-stat ${cls}"><small>${label}</small><b>${value}</b></div>`;
    const risk = { ok: 0, watch: 0, risk: 0 }; list.forEach((x) => (risk[x.risk] = (risk[x.risk] || 0) + 1));
    const ranks = Object.entries(t.rank_dist || {}).sort((a, b) => b[1] - a[1]).slice(0, 4), rankMax = Math.max(1, ...ranks.map(([, v]) => v));
    const initials = (n) => (n || "?").trim().split(/\s+/).map((w) => w[0]).join("").slice(0, 2).toUpperCase();
    main.innerHTML = `<div class="anx">
      <header class="anx-head">
        <div class="anx-title"><h2>${ico("chart", 22)}Хяналт ба статистик</h2><p>${course ? esc(course.title) : "Бүх сургалт"} · сүүлийн ${st.days} хоног${t.live ? ` · <span class="an-live">● ${t.live} одоо үзэж байна</span>` : ""}</p></div>
        <div class="anx-tools">
          <select id="anCourse" aria-label="Сургалт"><option value="">Бүх сургалт</option>${courseList.map((c) => `<option value="${esc(c.id)}" ${c.id === st.course ? "selected" : ""}>${esc(c.title)}</option>`).join("")}</select>
          <div class="anx-seg" role="group" aria-label="Хугацаа">${[7, 30, 90].map((d) => `<button type="button" data-an-days="${d}" aria-pressed="${d === st.days}">${d} хоног</button>`).join("")}</div>
          <a class="btn btn-glass btn-sm" id="anCsv" href="#" title="Excel-д нээгдэх тайлан">${ico("files", 15)}Excel</a><button class="btn btn-glass btn-sm" id="anPdf" title="Хэвлэх / PDF">${ico("book", 15)}PDF</button>
        </div>
      </header>
      <div class="anx-kpis">
        ${kpi("users", "Суралцагч", t.students, t.live ? `<span class="an-live">● ${t.live} одоо үзэж байна</span>` : `${rows.length} хүн элссэн / хичээл авсан`)}
        ${kpi("clock", "Идэвхтэй хугацаа", dur(t.active_sec), `нийт ${dur(t.total_sec)}-аас · ${t.active_pct}%`)}
        ${kpi("eye", "Анхаарлын индекс", t.attention + "%", meter(t.attention))}
        ${kpi("target", "Суралцсан оноо", t.learn_score + "%", meter(t.learn_score))}
      </div>
      <div class="anx-stats">
        ${stat("Идэвхтэй хувь", t.active_pct + "%")}${stat("Зөрчил", t.violations, t.violations ? "bad" : "")}${stat("Шалгалт", t.exams)}
        ${stat("Асуулга дуусгасан", `${t.quiz_mastered}<em>/${t.quiz_lessons}</em>`)}${stat("Бичсэн дүгнэлт", t.reflections)}${stat("Дээд цол", t.top_rank?.lessons ? esc(t.top_rank.name) : "—")}
      </div>
      <div class="anx-grid">
        <section class="anx-card anx-wide"><div class="anx-card-head"><h3>${ico("pulse", 18)}Өдөр бүрийн идэвх</h3><div class="an-legend"><span><i style="background:var(--brand)"></i>Идэвхтэй</span><span><i style="background:var(--accent)"></i>Идэвхгүй, өөр цонхонд</span></div></div>${chart2(data.daily || [])}</section>
        <section class="anx-card"><div class="anx-card-head"><h3>${ico("alert", 18)}Эрсдэлийн тойм</h3></div>
          ${list.length ? `<div class="anx-riskbar">${["ok", "watch", "risk"].map((k) => (risk[k] ? `<i class="${k}" style="flex:${risk[k]}" title="${RISK[k][0]}: ${risk[k]}"></i>` : "")).join("")}</div>` : ""}
          <ul class="anx-risks">${[["ok", "Идэвхтэй, анхааралтай"], ["watch", "Идэвх, анхаарал буурсан"], ["risk", "Зөрчил ихтэй эсвэл идэвхгүй"]].map(([k, hint]) => `<li><span class="an-risk ${k}">${RISK[k][0]}</span><small>${hint}</small><b>${risk[k] || 0}</b></li>`).join("")}</ul>
          <h4 class="anx-sub-h">Цолын тархалт</h4>
          ${ranks.length ? `<ul class="anx-ranks">${ranks.map(([k, v]) => `<li><span>${esc(k)}</span><i style="width:${v / rankMax * 100}%"></i><b>${v}</b></li>`).join("")}</ul>` : `<p class="muted small">Цол олгогдоогүй байна.</p>`}
          ${t.cheated_lessons ? `<p class="anx-warn">⛔ ${t.cheated_lessons} хичээлд хуулах оролдлого — оноо тооцогдоогүй</p>` : ""}</section>
      </div>
      <section class="anx-card"><div class="anx-card-head"><h3>${ico("users", 18)}Суралцагч бүрээр <span class="chip">${list.length}</span></h3><label class="rail-search anx-search">${ico("search", 16)}<input type="search" id="anSearch" placeholder="Нэрээр хайх…" aria-label="Хайх" value="${esc(st.q || "")}"></label></div>
        <p class="muted small anx-hint">Мөр дээр дарж дэлгэрэнгүйг (бичсэн дүгнэлт, асуултын хариулт, видео үзэлтийн зураглал) харна.</p>
        <div class="an-table-wrap"><table class="tbl an-table anx-table"><thead><tr><th>Суралцагч</th><th>Идэвхтэй</th><th>Анхаарал</th><th>Суралцсан</th><th>Зөрчил</th><th>Шалгалт</th><th>Төлөв</th><th><span class="sr-only">Үйлдэл</span></th></tr></thead><tbody>
        ${list.map((x) => `<tr class="an-row" data-uid="${esc(x.user_id)}"><td><div class="anx-who"><span class="anx-av" style="--h:${hueOfName(x.name)}">${esc(initials(x.name))}</span><div><b>${esc(x.name)}</b>${x.live ? ` <span class="an-live">●</span>` : ""}<small class="muted">${x.lessons} хичээл · ${fmtDate(x.last_at)}</small><span class="an-rank ${x.rank?.cheated ? "bad" : ""}" title="${x.rank?.points || 0} оноо">🎖 ${esc(x.rank?.name || "Шинэ цэрэг")}${x.rank?.cheated ? ` · ⛔${x.rank.cheated}` : ""}</span></div></div></td>
          <td data-label="Идэвхтэй"><b>${dur(x.active_sec)}</b><small class="muted">нийт ${dur(x.total_sec)}</small></td><td data-label="Анхаарал">${bar(x.attention)}</td>
          <td data-label="Суралцсан">${bar(x.learn_score)}<small class="muted">${learnHint(x)}</small></td>
          <td data-label="Зөрчил">${x.violations ? `<b class="an-bad">${x.violations}</b><small class="muted">${["tab_switch", "copy", "auto_block"].filter((k) => x.counts[k]).map((k) => `${esc(labels[k] || k)}: ${x.counts[k]}`).join(", ")}</small>` : `<span class="muted">0</span>`}</td>
          <td data-label="Шалгалт">${x.exam_best >= 0 ? `<b>${x.exam_best}%</b>${x.terminated ? `<small class="an-bad">${x.terminated} хаагдсан</small>` : ""}` : `<span class="muted">—</span>`}</td>
          <td data-label="Төлөв"><span class="an-risk ${RISK[x.risk][1]}">${RISK[x.risk][0]}</span></td>
          <td class="anx-acts"><div><button class="icon-btn" data-detail title="Дэлгэрэнгүй" aria-label="Дэлгэрэнгүй">${ico("eye", 17)}</button><button class="icon-btn" data-remind title="Сануулга илгээх" aria-label="Сануулга илгээх">${ico("chat", 17)}</button></div></td></tr>`).join("") || `<tr><td colspan="8"><div class="anx-empty">${ico("users", 26)}<p>Энэ хугацаанд хичээл үзсэн суралцагч алга.</p></div></td></tr>`}
        </tbody></table></div></section>
      <div class="anx-grid">
        <section class="anx-card anx-wide"><div class="anx-card-head"><h3>${ico("clock", 18)}Сүүлийн үйл явдал</h3><span class="muted small">зөрчил, сануулга, шалгалт</span></div><div id="anFeed"></div></section>
        <section class="anx-card"><div class="anx-card-head"><h3>${ico("users", 18)}Бүх суралцагчид <span class="chip">${rows.length}</span></h3></div>
          <p class="muted small anx-hint">Элссэн эсвэл хичээл худалдаж авсан хүмүүс. «Чат» дарахад баруун талд яриа нээгдэнэ.</p>
          <div class="anx-roster" id="stuList">${rows.map(studentMini).join("") || `<div class="anx-empty">${ico("users", 26)}<p>Одоогоор суралцагч алга. Профайлаа түгээж, үнэгүй хичээл нийтлээрэй.</p></div>`}</div></section>
      </div></div>`;
    if (st.q) onInput({ target: $("#anSearch") });
    $("#anCsv").href = "#";
    mountFeed();
  };
  const load = async () => { const d = await api(`/api/me/analytics?${qs()}`); data = d.data; labels = d.labels; render(); };
  // Тайлан: CSV-г токентой татна; PDF-г хэвлэх цонхоор.
  const download = async (url, name) => {
    const res = await fetch(url, { headers: { Authorization: "Bearer " + Auth.token } });
    if (!res.ok) return toast("Тайлан татаж чадсангүй", true);
    const a = document.createElement("a"); a.href = URL.createObjectURL(await res.blob()); a.download = name; a.click(); setTimeout(() => URL.revokeObjectURL(a.href), 2000);
  };
  const printReport = (title, html) => {
    const w = open("", "_blank"); if (!w) return toast("Шинэ цонх нээгдсэнгүй — хөтчийн popup зөвшөөрнө үү", true);
    w.document.write(`<!doctype html><html lang="mn"><head><meta charset="utf-8"><title>${esc(title)}</title><style>body{font:13px system-ui,sans-serif;color:#121829;margin:28px}h1{font-size:20px;margin:0 0 4px}h2{font-size:15px;margin:22px 0 8px}table{width:100%;border-collapse:collapse}th,td{border:1px solid #d6dbe6;padding:6px 8px;text-align:left;vertical-align:top}th{background:#eef2fb}.m{color:#59637a}</style></head><body>${html}<p class="m">surgalt.mn · ${new Date().toLocaleString()}</p></body></html>`);
    w.document.close(); setTimeout(() => w.print(), 300);
  };
  const classReport = () => {
    const c = courseList.find((x) => x.id === st.course);
    printReport("Ангийн тайлан", `<h1>Ангийн тайлан${c ? " — " + esc(c.title) : ""}</h1><p class="m">Сүүлийн ${st.days} хоног · ${data.totals.students} суралцагч · идэвхтэй ${data.totals.active_pct}% · анхаарал ${data.totals.attention}%</p>
      <table><tr><th>Суралцагч</th><th>Идэвхтэй</th><th>Нийт</th><th>Идэвхтэй %</th><th>Анхаарал %</th><th>Зөрчил</th><th>Шалгалт</th><th>Төлөв</th></tr>
      ${data.students.map((x) => `<tr><td>${esc(x.name)}</td><td>${dur(x.active_sec)}</td><td>${dur(x.total_sec)}</td><td>${x.active_pct}</td><td>${x.attention}</td><td>${x.violations}</td><td>${x.exam_best >= 0 ? x.exam_best + "%" : "—"}</td><td>${RISK[x.risk][0]}</td></tr>`).join("")}</table>`);
  };
  // Суралцагчийн дэлгэрэнгүй
  const detail = async (uid) => {
    const d = await api(`/api/me/analytics/students/${uid}?${qs()}`), x = d.student;
    if (!x) return toast("Мэдээлэл алга");
    // Шалгалт бүрийн хамгийн сүүлийн (дууссан) оролдлого — "Дахин нээх" товч зөвхөн тэр мөрөнд.
    const lastTry = new Set(Object.values((d.exams || []).reduce((o, a) => (a.status !== "active" && (!o[a.lesson_id] || a.started_at > o[a.lesson_id].started_at) && (o[a.lesson_id] = a), o), {})).map((a) => a.id));
    const html = `<h3 class="h3">${esc(x.name)}</h3>
      <div class="an-tiles"><div class="an-tile"><small>Идэвхтэй</small><b>${dur(x.active_sec)}</b></div><div class="an-tile"><small>Идэвхтэй хувь</small><b>${x.active_pct}%</b></div>
        <div class="an-tile"><small>Анхаарал</small><b>${x.attention}%</b></div><div class="an-tile ${x.violations ? "warn" : ""}"><small>Зөрчил</small><b>${x.violations}</b></div>
        <div class="an-tile learn"><small>Суралцсан оноо</small><b>${x.learn_score}%</b></div>
        <div class="an-tile"><small>Тогтмол байдал</small><b>${x.streak_days}</b><span class="muted small">дараалсан өдөр · ${x.active_days} нийт өдөр</span></div>
        <div class="an-tile ${x.rank?.cheated ? "warn" : "learn"}"><small>Нэгдсэн цол</small><b>${esc(x.rank?.name || "Шинэ цэрэг")} ${esc(x.rank?.insignia || "")}</b><span class="muted small">${x.rank?.points || 0} оноо${x.rank?.integration?.bonus ? ` (интеграц +${x.rank.integration.bonus})` : ""}${x.rank?.next ? ` · дараагийнх ${x.rank.next}` : ""}${x.rank?.cheated ? ` · ⛔ ${x.rank.cheated} хичээл тооцогдоогүй` : ""}</span></div></div>
      ${x.lesson_ranks?.length ? `<h4>Нэгдсэн цолд нэмсэн оноо — хичээл бүрээр</h4><p class="muted small" style="margin:0 0 8px">Цолыг хичээл бүрт биш, бүх хичээлийг нэгтгэж олгоно. Хичээл бүр 0-100 оноо нэмнэ (дууссан, идэвхтэй хугацаа, асуулга, дүгнэлт, видео; таб солилт хасна). Бүлэг, сургалтын бүх хичээлийг дуусгавал интеграцын нэмэлт оноо${x.rank?.integration ? ` — одоо +${x.rank.integration.bonus || 0} (${x.rank.integration.sections || 0}/${x.rank.integration.sections_all || 0} бүлэг, ${x.rank.integration.courses || 0}/${x.rank.integration.courses_all || 0} сургалт бүтэн)` : ""}. Хуулах оролдлоготой хичээлийн оноо тооцогдохгүй.</p>
        ${x.rank?.tips?.length ? `<ul class="muted small" style="margin:0 0 8px 18px">${x.rank.tips.map((t) => `<li>${esc(t)}</li>`).join("")}</ul>` : ""}
        <table class="tbl"><thead><tr><th>Хичээл</th><th>Нэмсэн оноо</th><th>Үндэслэл</th></tr></thead><tbody>${x.lesson_ranks.map((r) => `<tr class="${r.disqualified ? "an-wrongq" : ""}"><td>${esc(r.title)}</td><td>${r.disqualified ? "⛔ тооцогдоогүй" : "+" + r.points}</td><td class="muted small">${(r.reasons || []).map(esc).join(", ")}</td></tr>`).join("")}</tbody></table>` : ""}
      ${chart(d.daily)}
      <h4>Идэвхтэй суралцсаны нотолгоо</h4>
      <div class="an-tiles">
        <div class="an-tile"><small>Асуултад зөв хариулсан</small><b>${x.quiz_total ? x.quiz_accuracy + "%" : "—"}</b><span class="muted small">${x.quiz_total} асуулт${x.quiz_total ? `, дундаж ${Math.round(x.quiz_avg_ms / 1000)} сек` : ""}</span></div>
        <div class="an-tile ${x.quiz_guesses > 2 ? "warn" : ""}"><small>Хэт хурдан хариулсан</small><b>${x.quiz_guesses}</b><span class="muted small">1.5 сек-ээс богино (таамаг)</span></div>
        <div class="an-tile"><small>Асуулга дуусгасан</small><b>${x.quiz_lessons ? `${x.quiz_mastered}/${x.quiz_lessons}` : "—"}</b><span class="muted small">хичээл · анх удаад зөв ${x.quiz_questions ? x.quiz_first_try + "%" : "—"} · ${x.quiz_questions ? (x.quiz_attempts / 10).toFixed(1) + " оролдлого/асуулт" : ""}</span></div>
        <div class="an-tile"><small>▶ Видео үзэлт</small><b>${x.video_clips ? x.video_coverage + "%" : "—"}</b><span class="muted small">${x.video_clips} видео</span></div>
        <div class="an-tile"><small>Дүгнэлт бичсэн</small><b>${x.reflections}</b><span class="muted small">${x.reflections ? `дундаж ${x.reflection_words} үг` : "одоогоор алга"}</span></div></div>
      ${d.watches?.length ? `<h4>Видеоны үзэлтийн зураглал</h4><p class="muted small" style="margin:0 0 8px">Бараан хэсэг = давтаж үзсэн, цайвар саарал = алгассан хэсэг.</p><div class="vh">${d.watches.map(heatmap).join("")}</div>` : ""}
      ${d.reflections?.length ? `<h4>Бичсэн дүгнэлтүүд</h4><div class="refl-list">${d.reflections.map((r) => `<div class="refl-item"><b>${esc(r.lesson)}</b><small class="muted">${fmtDate(r.at)}</small><p>${esc(r.text)}</p></div>`).join("")}</div>` : ""}
      ${d.quiz_lessons?.length ? `<h4>Асуулгын үнэлгээ — хичээл бүрээр</h4><p class="muted small" style="margin:0 0 8px">Хичээл доторх өөрийгөө сорих асуултууд. Бүгдэд нь зөв хариулсан хичээлийн дараагийнх нээгдэнэ.</p>
        <table class="tbl"><thead><tr><th>Хичээл</th><th>Зөв / асуулт</th><th>Төлөв</th><th>Оролдлого</th><th>Анх удаад зөв</th><th>Дундаж</th></tr></thead><tbody>${d.quiz_lessons.map((q) => `<tr class="${q.done ? "" : "an-wrongq"}"><td>${esc(q.title)}</td><td>${q.correct}/${q.total}</td><td>${q.done ? `✓ Дууссан${q.done_at ? " · " + fmtDate(q.done_at) : ""}` : `${q.total - q.correct} үлдсэн`}</td><td>${q.attempts}${q.questions ? ` (${(q.attempts / q.questions).toFixed(1)}/асуулт)` : ""}</td><td>${q.questions ? q.first_try + "/" + q.questions : "—"}</td><td>${q.avg_ms ? (q.avg_ms / 1000).toFixed(1) + " сек" : "—"}${q.guesses ? ` · ⚡${q.guesses}` : ""}</td></tr>`).join("")}</tbody></table>` : ""}
      ${d.quiz_logs?.length ? `<h4>Асуултын хариултын лог</h4><div class="bk-log" id="anQuizLog"><table class="tbl"><thead><tr><th>Огноо</th><th>Хичээл</th><th>Асуулт</th><th>Хариулт</th><th>Хугацаа</th></tr></thead><tbody></tbody></table></div>` : ""}
      <h4>Хичээл тус бүрээр</h4><table class="tbl"><thead><tr><th>Хичээл</th><th>Идэвхтэй</th><th>Нийт</th><th>Удаа</th><th>Зөрчил</th></tr></thead><tbody>${d.lessons.map((l) => `<tr><td>${esc(l.title)}</td><td>${dur(l.active_sec)}</td><td>${dur(l.total_sec)}</td><td>${l.sessions}</td><td>${l.violations}</td></tr>`).join("") || `<tr><td colspan="5" class="muted">Алга</td></tr>`}</tbody></table>
      <h4>Шалгалтууд</h4><table class="tbl"><thead><tr><th>Огноо</th><th>Шалгалт</th><th>Оноо</th><th>Төлөв</th><th>Зөрчил</th><th></th></tr></thead><tbody>${d.exams.map((a) => `<tr><td>${fmtDate(a.started_at)}</td><td>${esc(d.lesson_titles?.[a.lesson_id] || "")}</td><td>${a.pct}%${a.passed ? " ✓" : ""}</td><td>${a.status === "terminated" ? `<b class="an-bad">Хаагдсан</b> · ${esc(a.reason || "")}` : a.status === "submitted" ? "Өгсөн" : a.status === "expired" ? "Хугацаа хэтэрсэн" : "Явагдаж байна"}</td><td>${a.violations}</td><td>${lastTry.has(a.id) ? `<button class="btn btn-glass btn-sm an-reopen" data-reopen-exam="${esc(a.lesson_id)}" data-reopen-user="${esc(uid)}" title="Дахин өгөх боломж олгоно (+1 оролдлого), суралцагчид мэдэгдэл очно">🔓 Дахин нээх</button>` : ""}</td></tr>`).join("") || `<tr><td colspan="6" class="muted">Алга</td></tr>`}</tbody></table>
      <h4>Лог</h4><div id="anStuFeed"></div>`;
    document.body.insertAdjacentHTML("beforeend", `<div class="modal" id="anModal"><div class="modal-card" style="width:min(900px,100%)"><button class="icon-btn modal-x" data-close aria-label="Хаах">${ico("x", 18)}</button>${html}
      <div class="hero-cta" style="margin:16px 0 0;justify-content:flex-end"><button class="btn btn-glass" data-x="csv">${ico("files", 16)}Excel</button><button class="btn btn-glass" data-x="pdf">${ico("book", 16)}PDF</button><button class="btn btn-gold" data-x="remind">${ico("chat", 16)}Сануулга илгээх</button></div></div></div>`);
    const m = $("#anModal"), sf = makeFeed(`&student=${encodeURIComponent(uid)}`, false); $("#anStuFeed", m).replaceWith(sf.box);
    const qRow = (q) => `<tr class="${q.correct ? "" : "an-wrongq"}"><td>${fmtDate(q.at)}</td><td>${esc(d.lesson_titles?.[q.lesson_id] || "")}</td><td>${esc(q.question)}</td><td>${q.correct ? "✓ Зөв" : "✗ Буруу"}</td><td>${q.ms ? (q.ms / 1000).toFixed(1) + " сек" + (q.ms < 1500 ? " ⚡" : "") : "—"}</td></tr>`;
    if ($("#anQuizLog", m)) scrollRows($("#anQuizLog", m), d.quiz_logs, qRow);
    SG.openModal(m);
    m.addEventListener("click", (e) => {
      if (e.target === m || e.target.closest("[data-close]")) { SG.closeModal(m); setTimeout(() => m.remove(), 300); return; }
      const ro = e.target.closest("[data-reopen-exam]"); if (ro) return reopenExam(ro);
      const b = e.target.closest("[data-x]"); if (!b) return;
      if (b.dataset.x === "csv") download(`/api/me/analytics/export?${qs()}&student=${uid}`, `suragch-${x.name}.csv`);
      if (b.dataset.x === "pdf") printReport("Суралцагчийн тайлан", `<h1>${esc(x.name)}</h1>${html.replace(/<svg[\s\S]*?<\/svg>/, "").replace(/<button[^>]*data-reopen-exam[\s\S]*?<\/button>/g, "").replace('<div id="anStuFeed"></div>', sf.box.querySelector("table").outerHTML).replace(/(id="anQuizLog"[\s\S]*?<tbody>)(<\/tbody>)/, (_, a, b) => a + (d.quiz_logs || []).map(qRow).join("") + b)}`);
      if (b.dataset.x === "remind") remind(uid, x.name);
    });
  };
  // Сануулга: суралцагчид мэдэгдэл очно.
  const remind = (uid, name) => {
    const r = rows.find((x) => x.user.id === uid), cs = r ? r.courses : [];
    const opts = (cs.length ? cs.map((c) => [c.course_id, c.course_title]) : courseList.map((c) => [c.id, c.title]));
    document.body.insertAdjacentHTML("beforeend", `<div class="modal" id="rmModal"><div class="modal-card"><button class="icon-btn modal-x" data-close aria-label="Хаах">${ico("x", 18)}</button>
      <h3 class="h3">${esc(name)}-д сануулга</h3><form class="form" id="rmForm"><label>Сургалт<select name="course">${opts.map(([id, t]) => `<option value="${esc(id)}" ${id === st.course ? "selected" : ""}>${esc(t)}</option>`).join("")}</select></label>
      <div class="find-chips" style="justify-content:flex-start">${["Хичээлээ анхааралтай, бусад цонхоо хаагаад үзээрэй.", "Даалгавраа хугацаанд нь хийгээрэй.", "Шалгалтдаа сайн бэлдээрэй, амжилт!"].map((t) => `<button type="button" data-t="${esc(t)}">${esc(t)}</button>`).join("")}</div>
      <label>Мессеж<textarea name="message" rows="3" maxlength="500" required></textarea></label><p class="form-error" role="alert"></p>
      <div class="hero-cta" style="margin:0;justify-content:flex-end"><button type="button" class="btn btn-ghost" data-close>Болих</button><button class="btn btn-gold">Илгээх</button></div></form></div></div>`);
    const m = $("#rmModal"), f = $("#rmForm", m); SG.openModal(m); f.message.focus();
    const close = () => { SG.closeModal(m); setTimeout(() => m.remove(), 300); };
    m.addEventListener("click", (e) => { if (e.target === m || e.target.closest("[data-close]")) { e.preventDefault(); close(); } const t = e.target.closest("[data-t]"); if (t) f.message.value = t.dataset.t; });
    f.onsubmit = async (e) => { e.preventDefault();
      try { await api(`/api/me/students/${uid}/remind`, { method: "POST", body: { course_id: f.course.value, message: f.message.value } }); toast("Сануулга илгээгдлээ ✓"); close(); load(); }
      catch (err) { $(".form-error", f).textContent = err.message; } };
  };
  const onClick = (e) => {
    const tr = e.target.closest("tr[data-uid]");
    if (e.target.closest("#anCsv")) { e.preventDefault(); return download(`/api/me/analytics/export?${qs()}`, "angi-tailan.csv"); }
    if (e.target.closest("#anPdf")) return classReport();
    const dd = e.target.closest("[data-an-days]"); if (dd) { st.days = +dd.dataset.anDays; load(); return; }
    const pd = e.target.closest("[data-an-detail]"); if (pd && !e.target.closest("[data-chat-student]")) return detail(pd.dataset.anDetail);
    if (tr && e.target.closest("[data-remind]")) return remind(tr.dataset.uid, $("b", tr).textContent);
    if (tr) return detail(tr.dataset.uid); // мөр хаана ч дарсан дэлгэрэнгүй нээнэ
  };
  const onChange = (e) => { if (e.target.id === "anCourse") { st.course = e.target.value; load(); } if (e.target.id === "anDays") { st.days = +e.target.value; load(); } };
  function onInput(e) { if (e.target?.id !== "anSearch") return; st.q = e.target.value; const q = st.q.trim().toLowerCase(); $$(".an-table tbody tr[data-uid]").forEach((tr) => (tr.hidden = !!q && !tr.textContent.toLowerCase().includes(q))); }
  main.addEventListener("click", onClick); main.addEventListener("change", onChange); main.addEventListener("input", onInput);
  timer = setInterval(() => { if (!document.hidden && !$(".modal.open")) load().catch(() => {}); }, 30000); // бодит хугацаанд ойрхон
  cleanup = () => { clearInterval(timer); main.removeEventListener("click", onClick); main.removeEventListener("change", onChange); main.removeEventListener("input", onInput); };
  await load();
}

/* ---------- Суралцагч ---------- */
async function learning() {
  const h = await api("/api/me/home");
  main.innerHTML = panel(`<div class="panel-head"><h2>Миний сургалтууд</h2><a class="btn btn-ghost btn-sm" href="/">Нүүр хуудас</a></div><div class="course-grid">${h.courses.map((c, i) => `
    <a class="course-card tilt" href="/c/${esc(c.course.id)}" style="--h:${hueOfName(c.course.title)}"><div class="course-art"><span class="course-num">${String(i + 1).padStart(2, "0")}</span><span class="glare"></span></div>
    <div class="course-body"><h3>${esc(c.course.title)}</h3><div class="course-meta"><span>${c.course.lesson_count} хичээл</span><span class="price free">▶ Үргэлжлүүлэх</span></div></div></a>`).join("") || `<div class="empty">Та одоогоор сургалтад элсээгүй байна</div>`}</div>`);
}

dispatchEvent(new Event("studio-ready"));
})();
