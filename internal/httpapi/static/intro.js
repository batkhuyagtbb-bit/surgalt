// Интро: «Оюуны сансар огторгуйн хамгаалагч» — сансарт хөвөх голограмм тавцан дээр цэргийн хуягт машин орж ирээд
// хамгаалагч-робот болж хувирч, ёслол хийнэ (Three.js / WebGL). Дүр нь surgalt.mn-ийн өөрийн загвар (брэндийн өнгө,
// «s» од тэмдэг, цолны тэмдэг) — ямар нэг кино, тоглоомын дүрийн хуулбар биш.
// app.js зөвхөн анх орох үед (эсвэл ?intro=1) ачаална; Three.js-ийг энд л татна. Дуу: intro-sound.js (WebAudio, файлгүй).
const THREE_URL = "https://cdn.jsdelivr.net/npm/three@0.170.0/build/three.module.min.js";

export async function runIntro({ onDone } = {}) {
  const [T, S] = await Promise.all([import(THREE_URL), import(`./intro-sound.js${new URL(import.meta.url).search}`).catch(() => null)]);
  const el = document.createElement("div");
  el.className = "intro";
  el.setAttribute("role", "img");
  el.setAttribute("aria-label", "surgalt.mn — Оюуны сансар огторгуйн хамгаалагч интро");
  const brand = "surgalt", tld = ".mn", slogan = "Сур, сур, бас дахин сур", title = "Оюуны сансар огторгуйн хамгаалагч";
  const chars = (s, off = 0) => [...s].map((c, i) => `<span class="ch" style="--i:${i + off}">${c === " " ? "&nbsp;" : c}</span>`).join("");
  el.innerHTML = `<canvas></canvas>
    <div class="intro-brand"><em class="intro-title">${[...title].map((c, i) => `<span class="tc" style="--i:${i}">${c === " " ? "&nbsp;" : c}</span>`).join("")}</em><b>${chars(brand)}<i>${chars(tld, brand.length)}</i></b><span>${chars(slogan, brand.length + 3)}</span></div>
    <div class="intro-ctl">
      <button type="button" class="intro-sound" aria-pressed="false" hidden><svg class="snd-off" viewBox="0 0 24 24" aria-hidden="true"><path d="M4 9h3l5-4v14l-5-4H4z"/><path d="M16 9.5l5 5M21 9.5l-5 5"/></svg><svg class="snd-on" viewBox="0 0 24 24" aria-hidden="true"><path d="M4 9h3l5-4v14l-5-4H4z"/><path d="M16 9a4 4 0 0 1 0 6M18.5 6.5a7.5 7.5 0 0 1 0 11"/></svg><span>Дуу асаах</span></button>
      <button type="button" class="intro-skip">Алгасах <span aria-hidden="true">›</span></button>
    </div>`;
  document.body.append(el);
  document.documentElement.classList.add("intro-on");

  const canvas = el.querySelector("canvas"), tcs = [...el.querySelectorAll(".tc")];
  const renderer = new T.WebGLRenderer({ canvas, antialias: true, powerPreference: "high-performance" });
  renderer.setPixelRatio(Math.min(devicePixelRatio || 1, 1.75));
  renderer.setSize(innerWidth, innerHeight, false);
  renderer.outputColorSpace = T.SRGBColorSpace;
  renderer.toneMapping = T.ACESFilmicToneMapping;
  renderer.toneMappingExposure = 1.2;

  const BG = 0x050b1f;
  const scene = new T.Scene();
  scene.background = new T.Color(BG);
  const camera = new T.PerspectiveCamera(36, innerWidth / innerHeight, 0.1, 400);
  const texs = [];
  const canvasTex = (w, h, draw) => { const c = document.createElement("canvas"); c.width = w; c.height = h; draw(c.getContext("2d"), w, h); const t = new T.CanvasTexture(c); t.colorSpace = T.SRGBColorSpace; t.anisotropy = 4; texs.push(t); return t; };

  scene.add(new T.HemisphereLight(0xb3c4ff, BG, 1.1));
  const key = new T.DirectionalLight(0xffffff, 2.6); key.position.set(5, 9, 7); scene.add(key);
  const rim = new T.DirectionalLight(0xeaa02e, 3.2); rim.position.set(-7, 4, -6); scene.add(rim);
  const blue = new T.PointLight(0x4b7bff, 14, 20); blue.position.set(-4, 3, 4); scene.add(blue);

  // ---- Сансар: од, мананцар, цагирагтай гариг
  const SN = 2600, spos = new Float32Array(SN * 3), scol = new Float32Array(SN * 3);
  for (let i = 0; i < SN; i++) {
    const v = new T.Vector3(Math.random() * 2 - 1, Math.random() * 1.6 - 0.35, Math.random() * 2 - 1).normalize().multiplyScalar(80 + Math.random() * 60);
    spos.set([v.x, v.y, v.z], i * 3);
    const warm = Math.random() < 0.1, b = 0.65 + Math.random() * 0.35;
    scol.set(warm ? [1, 0.75, 0.38] : [b * 0.85, b * 0.9, b], i * 3);
  }
  const sgeo0 = new T.BufferGeometry(); sgeo0.setAttribute("position", new T.BufferAttribute(spos, 3)); sgeo0.setAttribute("color", new T.BufferAttribute(scol, 3));
  const stars = new T.Points(sgeo0, new T.PointsMaterial({ size: 1.7, sizeAttenuation: false, vertexColors: true, transparent: true, opacity: 0.95, depthWrite: false }));
  scene.add(stars);
  const glowTex = (rgb) => canvasTex(256, 256, (g) => { const gr = g.createRadialGradient(128, 128, 0, 128, 128, 128); gr.addColorStop(0, `rgba(${rgb},0.9)`); gr.addColorStop(0.4, `rgba(${rgb},0.35)`); gr.addColorStop(1, `rgba(${rgb},0)`); g.fillStyle = gr; g.fillRect(0, 0, 256, 256); });
  const cloud = (rgb, x, y, z, s, op) => { const sp = new T.Sprite(new T.SpriteMaterial({ map: glowTex(rgb), transparent: true, opacity: op, blending: T.AdditiveBlending, depthWrite: false })); sp.position.set(x, y, z); sp.scale.set(s, s, 1); scene.add(sp); };
  cloud("44,76,166", -45, 22, -95, 110, 0.6); cloud("31,60,143", 50, 8, -110, 120, 0.55); cloud("234,160,46", 30, 34, -115, 55, 0.16); cloud("90,70,190", -10, 40, -120, 90, 0.22);
  const planetTex = canvasTex(256, 128, (g, w, h) => { for (let y = 0; y < h; y++) { const k = 0.5 + 0.5 * Math.sin(y * 0.18) * Math.sin(y * 0.05); g.fillStyle = `rgb(${18 + 30 * k},${34 + 45 * k},${92 + 70 * k})`; g.fillRect(0, y, w, 1); } });
  const planet = new T.Mesh(new T.SphereGeometry(13, 48, 32), new T.MeshStandardMaterial({ map: planetTex, roughness: 0.9, metalness: 0, emissive: 0x0a1640, emissiveIntensity: 0.6 }));
  planet.rotation.z = 0.35; scene.add(planet);
  const pring = new T.Mesh(new T.RingGeometry(17, 24, 96), new T.MeshBasicMaterial({ color: 0xeaa02e, transparent: true, opacity: 0.22, side: T.DoubleSide, depthWrite: false }));
  pring.rotation.set(1.25, 0.25, 0.35); scene.add(pring);

  // ---- Голограмм тавцан (сансарт хөвнө) ба тэлэх цагираг
  const holo = canvasTex(1024, 1024, (g, w) => {
    const c = w / 2;
    g.strokeStyle = "rgba(90,130,255,0.55)"; g.lineWidth = 2;
    for (let r = 60; r < c; r += 60) { g.beginPath(); g.arc(c, c, r, 0, Math.PI * 2); g.stroke(); }
    for (let a = 0; a < 24; a++) { g.beginPath(); g.moveTo(c, c); g.lineTo(c + Math.cos(a * Math.PI / 12) * c, c + Math.sin(a * Math.PI / 12) * c); g.stroke(); }
    g.strokeStyle = "rgba(234,160,46,0.8)"; g.lineWidth = 6; g.beginPath(); g.arc(c, c, c - 8, 0, Math.PI * 2); g.stroke();
    const fade = g.createRadialGradient(c, c, c * 0.2, c, c, c); fade.addColorStop(0, "rgba(5,11,31,0)"); fade.addColorStop(1, "rgba(5,11,31,0.85)");
    g.globalCompositeOperation = "destination-out"; g.fillStyle = fade; g.fillRect(0, 0, w, w);
  });
  const deck = new T.Mesh(new T.CircleGeometry(7, 96), new T.MeshBasicMaterial({ map: holo, transparent: true, opacity: 0.85, blending: T.AdditiveBlending, depthWrite: false }));
  deck.rotation.x = -Math.PI / 2; deck.position.y = 0.002; scene.add(deck);
  const disc = new T.Mesh(new T.CircleGeometry(2.0, 72), new T.MeshBasicMaterial({ color: 0x1f3c8f, transparent: true, opacity: 0.35 }));
  disc.rotation.x = -Math.PI / 2; disc.position.y = 0.004; scene.add(disc);
  const ring = new T.Mesh(new T.RingGeometry(1.96, 2.06, 120), new T.MeshBasicMaterial({ color: 0xeaa02e, transparent: true, opacity: 0, side: T.DoubleSide, depthWrite: false }));
  ring.rotation.x = -Math.PI / 2; ring.position.y = 0.01; scene.add(ring);

  // ---- Материал, тэмдэгүүд
  const M = (color, metalness, roughness, extra = {}) => new T.MeshStandardMaterial({ color, metalness, roughness, ...extra });
  const navy = M(0x223f96, 0.45, 0.42), navy2 = M(0x182f72, 0.5, 0.4), silver = M(0xd3dbea, 0.6, 0.25), gun = M(0x242c47, 0.7, 0.35);
  const glass = M(0x0a1230, 0.9, 0.06), tire = M(0x10141f, 0.15, 0.85), rim2 = M(0x9aa6bd, 0.8, 0.25);
  const amber = M(0xeaa02e, 0.2, 0.35, { emissive: 0xeaa02e, emissiveIntensity: 0.3 });
  const eyes = M(0xffc163, 0.1, 0.3, { emissive: 0xffb340, emissiveIntensity: 0 });
  const lamp = M(0xffffff, 0.1, 0.3, { emissive: 0xfff3d6, emissiveIntensity: 1.8 });
  const decalMat = (tex) => new T.MeshStandardMaterial({ map: tex, transparent: true, metalness: 0.3, roughness: 0.55, depthWrite: false });
  const emblemTex = canvasTex(256, 256, (g) => { // улбар шар од, дотор нь «s» (surgalt)
    g.fillStyle = "#0e1c4a"; g.beginPath(); g.arc(128, 128, 122, 0, Math.PI * 2); g.fill();
    g.lineWidth = 10; g.strokeStyle = "#eaa02e"; g.beginPath(); g.arc(128, 128, 110, 0, Math.PI * 2); g.stroke();
    g.fillStyle = "#eaa02e"; g.beginPath();
    for (let i = 0; i < 10; i++) { const r = i % 2 ? 40 : 96, a = -Math.PI / 2 + i * Math.PI / 5; g.lineTo(128 + Math.cos(a) * r, 132 + Math.sin(a) * r); }
    g.closePath(); g.fill();
    g.fillStyle = "#0e1c4a"; g.font = "800 78px Manrope, Arial, sans-serif"; g.textAlign = "center"; g.textBaseline = "middle"; g.fillText("s", 128, 132);
  });
  const emblemMat = new T.MeshStandardMaterial({ map: emblemTex, emissiveMap: emblemTex, emissive: 0xffffff, emissiveIntensity: 0, metalness: 0.2, roughness: 0.4 });
  const stencil = canvasTex(256, 96, (g) => { g.fillStyle = "#eaa02e"; g.font = "800 58px Manrope, Arial, sans-serif"; g.textAlign = "center"; g.textBaseline = "middle"; g.fillText("SRG-01", 128, 50); });
  const chevron = canvasTex(128, 160, (g) => { // цолны тэмдэг: од ба гурван шеврон
    g.fillStyle = "#eaa02e";
    g.beginPath(); for (let i = 0; i < 10; i++) { const r = i % 2 ? 9 : 22, a = -Math.PI / 2 + i * Math.PI / 5; g.lineTo(64 + Math.cos(a) * r, 28 + Math.sin(a) * r); } g.closePath(); g.fill();
    for (let k = 0; k < 3; k++) { const y = 62 + k * 30; g.beginPath(); g.moveTo(16, y); g.lineTo(64, y + 22); g.lineTo(112, y); g.lineTo(112, y + 12); g.lineTo(64, y + 34); g.lineTo(16, y + 12); g.closePath(); g.fill(); }
  });

  const box = (w, h, d) => new T.BoxGeometry(w, h, d);
  const cyl = (r, h, seg = 28) => new T.CylinderGeometry(r, r, h, seg);
  const mesh = (geo, mat, pos = [0, 0, 0], rot = [0, 0, 0], parent) => { const m = new T.Mesh(geo, mat); m.position.set(...pos); m.rotation.set(...rot); parent?.add(m); return m; };
  const wheel = () => { // бартаат дугуй: дугуйн хээтэй
    const w = new T.Group(), spin = new T.Group(); w.add(spin);
    mesh(cyl(0.3, 0.24, 32), tire, [0, 0, 0], [0, 0, 0], spin);
    for (let i = 0; i < 12; i++) { const a = i * Math.PI / 6; mesh(box(0.08, 0.26, 0.1), tire, [Math.cos(a) * 0.3, 0, Math.sin(a) * 0.3], [0, -a, 0], spin); }
    mesh(cyl(0.16, 0.26, 20), rim2, [0, 0, 0], [0, 0, 0], spin);
    w.userData.spin = spin; return w;
  };

  const bot = new T.Group(); scene.add(bot);
  const parts = [];
  const Q = (r) => new T.Quaternion().setFromEuler(new T.Euler(...r));
  // part(обьект, машин үеийн [байрлал, эргэлт], робот үеийн [байрлал, эргэлт], хувирах эхлэл (сек), үргэлжлэх)
  const part = (o, veh, rob, at, dur = 0.7, opts = {}) => {
    bot.add(o);
    const p = { o, at, dur, vp: new T.Vector3(...veh[0]), vq: Q(veh[1]), rp: new T.Vector3(...rob[0]), rq: Q(rob[1]), done: false, ...opts };
    o.position.copy(p.vp); o.quaternion.copy(p.vq); parts.push(p); return o;
  };
  const H = Math.PI / 2, T0 = 1.45; // хувиралт эхлэх хугацаа
  // ---- Хөл: хуягт машины арын их бие (хоёр хагас) → шилбэ; арын бампер → тавхай; доод тал → гуя
  for (const s of [-1, 1]) {
    const leg = new T.Group();
    mesh(box(0.58, 1.24, 0.7), navy, [0, 0, 0], [0, 0, 0], leg);
    mesh(box(0.6, 0.09, 0.72), amber, [0, 0.34, 0], [0, 0, 0], leg);
    mesh(box(0.62, 0.32, 0.32), silver, [0, 0.62, 0.32], [0, 0, 0], leg);
    mesh(new T.PlaneGeometry(0.5, 0.19), decalMat(stencil), [0, -0.12, 0.356], [0, 0, 0], leg);
    part(leg, [[-1.25, 0.76, s * 0.37], [0, 0, H]], [[s * 0.45, 0.88, 0], [0, 0, 0]], T0 + 0.1, 0.85);
    part(mesh(box(0.62, 0.28, 0.96), gun), [[-2.2, 0.52, s * 0.37], [0, 0, H]], [[s * 0.45, 0.14, 0.12], [0, 0, 0]], T0 + 0.15, 0.8);
    part(mesh(box(0.46, 0.74, 0.5), gun), [[-0.3, 0.64, s * 0.34], [0, 0, H]], [[s * 0.45, 1.84, 0], [0, 0, 0]], T0 + 0.25, 0.8);
  }
  // ---- Дугуйнууд: арын 4 → шилбэний гадна тал; урд 2 → мөрний гадна тал
  const wheels = [];
  for (const s of [-1, 1]) for (const [vx, ry] of [[-0.8, 0.62], [-1.7, 1.14]]) {
    const w = wheel(); wheels.push(w);
    part(w, [[vx, 0.3, s * 0.84], [H, 0, 0]], [[s * 0.82, ry, 0], [0, 0, H]], T0, 0.75);
  }
  for (const s of [-1, 1]) { const w = wheel(); wheels.push(w); part(w, [[1.55, 0.3, s * 0.84], [H, 0, 0]], [[s * 1.48, 3.7, 0], [0, 0, H]], T0 + 0.9, 0.75); }
  // ---- Их бие: бүсэлхий, хэвлий, урд бүхээг → цээж, урд хуяг (гэрэл, од тэмдэг) → цээжний хуяг
  part(mesh(box(1.0, 0.38, 0.62), gun), [[0.35, 0.58, 0], [0, H, 0]], [[0, 2.37, 0], [0, 0, 0]], T0 + 0.45, 0.7);
  part(mesh(box(0.86, 0.44, 0.54), silver), [[0.78, 0.66, 0], [0, H, 0]], [[0, 2.77, 0], [0, 0, 0]], T0 + 0.55, 0.7);
  const chest = new T.Group();
  mesh(box(1.74, 0.98, 0.9), navy2, [0, 0, 0], [0, 0, 0], chest);
  mesh(box(1.52, 0.3, 0.06), glass, [0, 0.38, -0.46], [0, 0, 0], chest);           // харах ангархай (машинд) → нуруу
  for (const s of [-1, 1]) mesh(box(0.12, 0.6, 0.94), gun, [s * 0.8, -0.05, 0], [0, 0, 0], chest); // хажуугийн хуяг
  part(chest, [[1.3, 0.96, 0], [0, H, 0]], [[0, 3.44, 0], [0, 0, 0]], T0 + 0.7, 0.85);
  const plate = new T.Group();
  mesh(box(1.2, 0.66, 0.09), silver, [0, 0, 0], [0, 0, 0], plate);
  for (const s of [-1, 1]) mesh(box(0.2, 0.1, 0.06), lamp, [s * 0.45, -0.22, 0.05], [0, 0, 0], plate);
  mesh(new T.CircleGeometry(0.25, 48), emblemMat, [0, 0.05, 0.051], [0, 0, 0], plate);
  part(plate, [[1.82, 0.82, 0], [0, H, 0]], [[0, 3.42, 0.49], [0, 0, 0]], T0 + 0.8, 0.8);
  // ---- Мөрний хуяг: цолны тэмдэгтэй
  for (const s of [-1, 1]) {
    const sh = new T.Group();
    mesh(box(0.64, 0.5, 0.7), navy, [0, 0, 0], [0, 0, 0], sh);
    mesh(new T.PlaneGeometry(0.24, 0.3), decalMat(chevron), [s * 0.321, 0, 0], [0, s * H, 0], sh);
    part(sh, [[1.0, 0.66, s * 0.74], [0, 0, 0]], [[s * 1.12, 3.72, 0], [0, 0, 0]], T0 + 1.0, 0.7);
  }
  // ---- Гар: мөр ба тохойгоор нугардаг (ёслол хийнэ)
  const arms = {}, elbows = {};
  for (const s of [-1, 1]) {
    const a = new T.Group(), el2 = new T.Group();
    mesh(box(0.34, 0.64, 0.36), silver, [0, -0.33, 0], [0, 0, 0], a);
    el2.position.set(0, -0.66, 0); a.add(el2);
    mesh(new T.SphereGeometry(0.17, 20, 14), gun, [0, 0, 0], [0, 0, 0], el2);
    mesh(box(0.42, 0.66, 0.44), navy, [0, -0.36, 0.03], [0, 0, 0], el2);
    mesh(box(0.44, 0.08, 0.46), amber, [0, -0.16, 0.03], [0, 0, 0], el2);
    mesh(box(0.33, 0.33, 0.33), gun, [0, -0.84, 0.05], [0, 0, 0], el2);
    elbows[s] = el2;
    arms[s] = part(a, [[0.95, 0.5, s * 0.88], [0, 0, -H * 0.98]], [[s * 1.14, 3.48, 0], [0, 0, 0]], T0 + 1.2, 0.8);
  }
  // ---- Толгой: цэргийн дуулга, нүүрэвч, улбар шар визор, радио антен — бүхээг дотроос дээш гарна
  const hd = new T.Group();
  mesh(cyl(0.13, 0.2), gun, [0, -0.38, 0], [0, 0, 0], hd);
  mesh(box(0.5, 0.5, 0.5), navy2, [0, -0.02, 0], [0, 0, 0], hd);
  mesh(new T.SphereGeometry(0.36, 32, 16, 0, Math.PI * 2, 0, Math.PI / 2), navy, [0, 0.16, -0.02], [0, 0, 0], hd); // дуулганы бөмбөгөр
  mesh(cyl(0.42, 0.05, 40), navy, [0, 0.16, 0.03], [0, 0, 0], hd);               // дуулганы хүрээ
  mesh(box(0.38, 0.26, 0.06), silver, [0, -0.12, 0.27], [0, 0, 0], hd);          // нүүрэвч
  mesh(box(0.38, 0.07, 0.04), eyes, [0, 0.03, 0.28], [0, 0, 0], hd);             // визор
  mesh(cyl(0.012, 0.5, 8), gun, [0.28, 0.42, -0.12], [0, 0, -0.12], hd);         // радио антен
  const tip = mesh(new T.SphereGeometry(0.03, 12, 8), eyes, [0.31, 0.67, -0.12], [0, 0, 0], hd);
  part(hd, [[1.3, 0.78, 0], [0, H, 0]], [[0, 4.3, 0], [0, 0, 0]], T0 + 1.6, 0.6, { hidden: true });

  // ---- Оч ба залгагдах гялбаа
  const N = 700, sp = new Float32Array(N * 3), sv = new Float32Array(N * 3), life = new Float32Array(N);
  const sgeo = new T.BufferGeometry(); sgeo.setAttribute("position", new T.BufferAttribute(sp, 3));
  scene.add(new T.Points(sgeo, new T.PointsMaterial({ color: 0xffbf5a, size: 0.1, transparent: true, opacity: 0.95, blending: T.AdditiveBlending, depthWrite: false })));
  for (let i = 0; i < N; i++) sp[i * 3 + 1] = -99;
  const flash = new T.PointLight(0xffb340, 0, 5); scene.add(flash);
  let si = 0;
  const burst = (at, n = 24) => {
    flash.position.copy(at); flash.intensity = 18;
    for (let k = 0; k < n; k++) {
      const i = si++ % N;
      sp.set([at.x, at.y, at.z], i * 3);
      sv.set([(Math.random() - 0.5) * 3.6, Math.random() * 3 + 0.5, (Math.random() - 0.5) * 3.6], i * 3);
      life[i] = 0.45 + Math.random() * 0.5;
    }
  };

  const ease = (t) => (t < 0.5 ? 4 * t * t * t : 1 - (-2 * t + 2) ** 3 / 2);
  const outCubic = (t) => 1 - (1 - t) ** 3;
  const outBack = (t) => { const c = 1.6; return 1 + (c + 1) * (t - 1) ** 3 + c * (t - 1) ** 2; };
  const clamp01 = (t) => Math.min(1, Math.max(0, t));
  const DRIVE = 1.35, formed = T0 + 2.25, total = formed + 4, BEAT = (formed - T0) / 6; // хөгжмийн цохилт (intro-sound.js-тэй ижил)
  const portrait = innerWidth / innerHeight < 0.85;
  let start = performance.now(), last = start, raf = 0, ended = false, typed = 0, brandShown = false;
  // ---- Дуу: хөтөч хэрэглэгч товшихоос өмнө дуу гаргахыг хориглодог тул «Дуу асаах» товч эсвэл дэлгэц дээр товшиход
  // нэгдэнэ (зөвшөөрөлтэй бол шууд). Хөгжим интроны цагтай уялдаж, нэгдсэн мөчөөс үргэлжилнэ.
  const AC = window.AudioContext || window.webkitAudioContext;
  let ac = null, score = null, joined = false, muted = false;
  if (S && AC) try { ac = new AC(); score = S.createScore(ac, ac.destination, { T0, formed, total }); } catch { ac = score = null; }
  const sndBtn = el.querySelector(".intro-sound");
  const paintSnd = () => { const on = joined && !muted; sndBtn.classList.toggle("on", on); sndBtn.setAttribute("aria-pressed", String(on)); sndBtn.querySelector("span").textContent = on ? "Дуу хаах" : "Дуу асаах"; };
  const join = () => { if (joined || ended || ac?.state !== "running") return; joined = true; score.start((performance.now() - start) / 1000, ac.currentTime); paintSnd(); };
  const unlock = () => { if (score && !joined && !ended) ac.resume().then(join, () => {}); };
  if (score) {
    sndBtn.hidden = false; ac.onstatechange = join; unlock();
    el.addEventListener("click", (e) => { if (!e.target.closest(".intro-skip")) unlock(); });
    sndBtn.addEventListener("click", () => { if (joined) { muted = !muted; score.mute(muted); paintSnd(); } });
  }
  const tmpQ = new T.Quaternion(), look = new T.Vector3(), IDQ = new T.Quaternion();
  // Ёслол: мөрөөс тохой урагш-дээш (u), шуу дуулганы зах руу (f) — эргэлтийг векторуудаас яг тооцоолно.
  const DOWN = new T.Vector3(0, -1, 0), u = new T.Vector3(0.2, 0.75, 0.6).normalize(), f = new T.Vector3(-0.82, 0.37, -0.1).normalize();
  const qS = new T.Quaternion().setFromUnitVectors(DOWN, u);
  const qE = qS.clone().invert().multiply(new T.Quaternion().setFromUnitVectors(u, f)).multiply(qS);
  const wide = !portrait; // өргөн дэлгэцэнд робот баруун талд, гарчиг зүүн талд

  // Камер: машиныг хажуугаас дагаж, хувирах үед урд доод талаас дээш харсан баатарлаг өнцөг рүү
  const aim = (t) => {
    const c = ease(clamp01((t - 0.6) / 3.4));
    const ang = 0.95 * (1 - c) + (portrait ? 0.04 : 0.12) * Math.sin(t * 0.45) * c; // босоо дэлгэцэнд нарийн өнцөг тул бага найгана
    const rad = (portrait ? 17 : 12.6) - c * 1.4 - clamp01((t - formed) / 3) * 0.5;
    camera.position.set(Math.sin(ang) * rad + bot.position.x * 0.25, 1.6 - c * 0.55, Math.cos(ang) * rad);
    const side = wide ? ease(clamp01((t - formed + 0.6) / 1.4)) : 0;
    look.set(bot.position.x * 0.6 - side * 2.1, 0.9 + c * (portrait ? 1.25 : 1.1), 0);
    camera.lookAt(look);
  };
  // Гариг: эцсийн кадрт өргөн дэлгэцэнд зүүн дээд буланд (гарчгийн дээр), босоо дэлгэцэнд баруун дээд буланд.
  const placePlanet = () => {
    aim(formed + 1.6); camera.updateMatrixWorld();
    const dir = new T.Vector3(...(portrait ? [0.48, 0.72] : [-0.6, 0.66]), 0.5).unproject(camera).sub(camera.position).normalize();
    planet.position.copy(camera.position).addScaledVector(dir, 128); pring.position.copy(planet.position);
    const k = portrait ? 0.45 : 0.62; planet.scale.setScalar(k); pring.scale.setScalar(k);
  };
  const resize = () => { camera.aspect = innerWidth / innerHeight; camera.updateProjectionMatrix(); renderer.setSize(innerWidth, innerHeight, false); placePlanet(); };
  resize(); addEventListener("resize", resize);
  const finish = () => {
    if (ended) return; ended = true;
    el.classList.add("out"); score?.fadeOut(0.6); removeEventListener("keydown", onKey);
    setTimeout(() => {
      cancelAnimationFrame(raf); removeEventListener("resize", resize); Promise.resolve(ac?.close?.()).catch(() => {});
      renderer.dispose(); texs.forEach((t) => t.dispose()); scene.traverse((o) => o.geometry?.dispose());
      el.remove(); document.documentElement.classList.remove("intro-on");
      onDone?.();
    }, 650);
  };
  el.querySelector(".intro-skip").onclick = finish;
  const onKey = (e) => { if (e.key === "Escape") finish(); else unlock(); };
  addEventListener("keydown", onKey);

  const frame = (now) => {
    raf = requestAnimationFrame(frame);
    const t = (now - start) / 1000, dt = Math.min(0.05, (now - last) / 1000); last = now;
    stars.rotation.y += dt * 0.008; planet.rotation.y += dt * 0.03;
    // 1) Хуягт машин тавцан дээр орж ирнэ, 2) хувирна, 3) ёслол хийнэ
    const drive = clamp01(t / DRIVE);
    bot.position.x = -12 * (1 - outCubic(drive));
    if (t < DRIVE + 0.25) for (const w of wheels) w.userData.spin.rotation.y -= dt * 16 * (1 - drive * 0.9);
    for (const p of parts) {
      const k = clamp01((t - p.at) / p.dur), e = p.hidden ? outBack(k) : ease(k);
      p.o.visible = !p.hidden || t >= p.at;
      p.o.position.lerpVectors(p.vp, p.rp, e);
      p.o.quaternion.copy(tmpQ.slerpQuaternions(p.vq, p.rq, Math.min(1, e)));
      if (k >= 1 && !p.done) { p.done = true; burst(p.o.getWorldPosition(new T.Vector3())); score?.clank(p.hidden ? 1.6 : 0.75 + Math.random() * 0.5); }
    }
    emblemMat.emissiveIntensity = t > T0 + 1.3 ? Math.min(1.3, (t - T0 - 1.3) * 2) * (0.85 + 0.15 * Math.sin(t * 5)) : 0;
    if (t > formed) {
      const a = t - formed;
      eyes.emissiveIntensity = a < 0.1 || (a > 0.18 && a < 0.26) ? 0 : 3.4;
      tip.scale.setScalar(1 + 0.4 * Math.max(0, Math.sin(a * 9)));
      const rk = clamp01(a / 1.2); ring.scale.setScalar(1 + rk * 2.1); ring.material.opacity = rk < 1 ? 0.95 * (1 - rk) : 0;
      // Цэргийн ёслол: баруун гараа дуулганы хүрээнд хүргэнэ
      const up = outCubic(clamp01((a - 0.45) / 0.55));
      arms[1].quaternion.slerpQuaternions(IDQ, qS, up);
      elbows[1].quaternion.slerpQuaternions(IDQ, qE, up);
      arms[-1].rotation.z = -0.08 * up;
      bot.position.y = Math.sin(a * 2.2) * 0.02;
      // Гарчиг: дохио хүлээн авч буй мэт тэмдэгт бүрээр бичигдэнэ (CSS анимацийн төгсгөлд найдахгүй, хугацаагаар)
      const n = Math.min(tcs.length + 2, Math.floor(a / 0.034));
      if (n !== typed) { if (n > typed && n <= tcs.length) score?.tick(); typed = n; tcs.forEach((ch, i) => { ch.classList.toggle("on", i < n); ch.classList.toggle("cur", i < n && i >= n - 2); }); }
      if (!brandShown && a > 3 * BEAT) { brandShown = true; el.classList.add("show-brand"); } // ялалтын хөгийн эхний (B♭) цохилттой зэрэг
    }
    aim(t);
    for (let i = 0; i < N; i++) {
      if (life[i] <= 0) continue;
      life[i] -= dt; sv[i * 3 + 1] -= 7 * dt;
      sp[i * 3] += sv[i * 3] * dt; sp[i * 3 + 1] += sv[i * 3 + 1] * dt; sp[i * 3 + 2] += sv[i * 3 + 2] * dt;
      if (life[i] <= 0 || sp[i * 3 + 1] < 0) { life[i] = 0; sp[i * 3 + 1] = -99; }
    }
    sgeo.attributes.position.needsUpdate = true;
    flash.intensity *= Math.pow(0.0006, dt);
    deck.rotation.z += dt * 0.05;
    renderer.render(scene, camera);
    if (t > total) finish();
  };
  raf = requestAnimationFrame(frame);
  return { skip: finish };
}
