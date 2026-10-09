// Интро: робот хэсэг хэсгээрээ нисэж ирж угсрагдаад, нүд нь асаж гараа даллан, «surgalt.mn» гарч ирнэ (Three.js / WebGL).
// app.js зөвхөн анх орох үед (эсвэл ?intro=1) ачаална; Three.js-ийг энд л татна.
const THREE_URL = "https://cdn.jsdelivr.net/npm/three@0.170.0/build/three.module.min.js";

export async function runIntro({ onDone } = {}) {
  const T = await import(THREE_URL);
  const el = document.createElement("div");
  el.className = "intro";
  el.setAttribute("role", "img");
  el.setAttribute("aria-label", "surgalt.mn — робот угсрагдаж буй интро");
  const brand = "surgalt", tld = ".mn", slogan = "Сур, сур, бас дахин сур";
  const chars = (s, off = 0) => [...s].map((c, i) => `<span class="ch" style="--i:${i + off}">${c === " " ? "&nbsp;" : c}</span>`).join("");
  el.innerHTML = `<canvas></canvas>
    <div class="intro-brand"><b>${chars(brand)}<i>${chars(tld, brand.length)}</i></b><span>${chars(slogan, brand.length + 3)}</span></div>
    <button type="button" class="intro-skip">Алгасах <span aria-hidden="true">›</span></button>`;
  document.body.append(el);
  document.documentElement.classList.add("intro-on");

  const canvas = el.querySelector("canvas");
  const renderer = new T.WebGLRenderer({ canvas, antialias: true, powerPreference: "high-performance" });
  renderer.setPixelRatio(Math.min(devicePixelRatio || 1, 1.75));
  renderer.setSize(innerWidth, innerHeight, false);
  renderer.outputColorSpace = T.SRGBColorSpace;
  renderer.toneMapping = T.ACESFilmicToneMapping;
  renderer.toneMappingExposure = 1.15;

  const BG = 0x0a1430;
  const scene = new T.Scene();
  scene.background = new T.Color(BG);
  scene.fog = new T.Fog(BG, 10, 26);
  const camera = new T.PerspectiveCamera(36, innerWidth / innerHeight, 0.1, 100);

  scene.add(new T.HemisphereLight(0xa9bdff, BG, 1.1));
  const key = new T.DirectionalLight(0xffffff, 2.4); key.position.set(4, 7, 6); scene.add(key);
  const rim = new T.DirectionalLight(0xeaa02e, 2.6); rim.position.set(-6, 3, -5); scene.add(rim);
  const blue = new T.PointLight(0x4b7bff, 9, 14); blue.position.set(-3, 2, 3); scene.add(blue);

  // Шал: гэрэлтсэн тор ба улбар шар цагираг (угсралт дуусахад тэлнэ)
  const grid = new T.GridHelper(28, 56, 0x3355c0, 0x15244d);
  grid.material.transparent = true; grid.material.opacity = 0.55; scene.add(grid);
  const disc = new T.Mesh(new T.CircleGeometry(1.35, 64), new T.MeshBasicMaterial({ color: 0x1f3c8f, transparent: true, opacity: 0.35 }));
  disc.rotation.x = -Math.PI / 2; disc.position.y = 0.005; scene.add(disc);
  const ring = new T.Mesh(new T.RingGeometry(1.32, 1.4, 96), new T.MeshBasicMaterial({ color: 0xeaa02e, transparent: true, opacity: 0, side: T.DoubleSide }));
  ring.rotation.x = -Math.PI / 2; ring.position.y = 0.01; scene.add(ring);

  const M = (color, metalness, roughness, extra = {}) => new T.MeshStandardMaterial({ color, metalness, roughness, ...extra });
  const navy = M(0x2447a8, 0.38, 0.34), white = M(0xf2f5fc, 0.18, 0.3), dark = M(0x1b2754, 0.5, 0.3), visor = M(0x050a1c, 0.6, 0.1);
  const core = M(0xeaa02e, 0.1, 0.4, { emissive: 0xeaa02e, emissiveIntensity: 0 });
  const eyes = M(0xffc163, 0.1, 0.3, { emissive: 0xffb340, emissiveIntensity: 0 });

  const box = (w, h, d) => new T.BoxGeometry(w, h, d);
  const cyl = (r, h, r2 = r) => new T.CylinderGeometry(r2, r, h, 28);
  const ball = (r) => new T.SphereGeometry(r, 28, 18);
  const parts = [];
  const robot = new T.Group(); scene.add(robot);
  // part(геометр, материал, байрлал, хэзээ ирэх (сек), [эргэлт], [эцэг group])
  const part = (geo, mat, pos, at, rot = [0, 0, 0], parent = robot) => {
    const m = geo.isObject3D ? geo : new T.Mesh(geo, mat);
    m.position.set(...pos); m.rotation.set(...rot);
    parent.add(m);
    if (parent === robot) {
      const dir = new T.Vector3(Math.random() - 0.5, Math.random() * 0.9 + 0.2, Math.random() - 0.5).normalize();
      parts.push({ m, at, to: m.position.clone(), rot: m.rotation.clone(), from: m.position.clone().addScaledVector(dir, 6 + Math.random() * 4),
        spin: new T.Euler((Math.random() - 0.5) * 7, (Math.random() - 0.5) * 7, (Math.random() - 0.5) * 7), done: false });
    }
    return m;
  };
  for (const s of [-1, 1]) { // хөл: тавхай → шилбэ → өвдөг → гуя
    const d = s < 0 ? 0 : 0.05;
    part(box(0.44, 0.18, 0.64), navy, [s * 0.33, 0.09, 0.06], 0.15 + d);
    part(cyl(0.13, 0.56), white, [s * 0.33, 0.46, 0], 0.32 + d);
    part(ball(0.155), dark, [s * 0.33, 0.77, 0], 0.48 + d);
    part(cyl(0.155, 0.46), navy, [s * 0.33, 1.02, 0], 0.62 + d);
  }
  part(box(0.95, 0.26, 0.52), dark, [0, 1.3, 0], 0.82);
  part(box(1.14, 0.98, 0.68), navy, [0, 1.88, 0], 0.98);
  part(box(0.72, 0.56, 0.06), white, [0, 1.93, 0.345], 1.14);
  part(cyl(0.15, 0.05), core, [0, 1.93, 0.39], 1.28, [Math.PI / 2, 0, 0]);
  for (const s of [-1, 1]) part(ball(0.18), dark, [s * 0.7, 2.24, 0], 1.38 + (s > 0 ? 0.06 : 0));
  const arms = {};
  for (const s of [-1, 1]) { // гар: мөрнөөс эргэдэг group (баруун гараараа даллана)
    const g = new T.Group();
    part(cyl(0.11, 0.52), white, [0, -0.3, 0], 0, [0, 0, 0], g);
    part(ball(0.12), dark, [0, -0.6, 0], 0, [0, 0, 0], g);
    part(cyl(0.1, 0.44), navy, [0, -0.86, 0.02], 0, [0, 0, 0], g);
    part(ball(0.135), dark, [0, -1.13, 0.04], 0, [0, 0, 0], g);
    arms[s] = part(g, null, [s * 0.74, 2.2, 0], 1.5 + (s > 0 ? 0.1 : 0));
  }
  part(cyl(0.1, 0.16), dark, [0, 2.44, 0], 1.72);
  part(box(0.84, 0.62, 0.62), white, [0, 2.82, 0], 1.84);
  part(box(0.66, 0.25, 0.05), visor, [0, 2.85, 0.32], 2.0);
  for (const s of [-1, 1]) {
    part(cyl(0.085, 0.07), navy, [s * 0.45, 2.82, 0], 2.08 + (s > 0 ? 0.04 : 0), [0, 0, Math.PI / 2]);
    part(ball(0.06), eyes, [s * 0.16, 2.86, 0.35], 2.16 + (s > 0 ? 0.04 : 0));
  }
  part(cyl(0.02, 0.34), dark, [0, 3.3, 0], 2.28);
  part(ball(0.055), eyes, [0, 3.5, 0], 2.36);
  for (const p of parts) { p.m.position.copy(p.from); p.m.visible = false; }

  // Оч: хэсэг залгагдах бүрт цацарна (нэмэлт гэрэлтэлттэй цэгүүд)
  const N = 520, sp = new Float32Array(N * 3), sv = new Float32Array(N * 3), life = new Float32Array(N);
  const sgeo = new T.BufferGeometry(); sgeo.setAttribute("position", new T.BufferAttribute(sp, 3));
  const sparks = new T.Points(sgeo, new T.PointsMaterial({ color: 0xffbf5a, size: 0.09, transparent: true, opacity: 0.95, blending: T.AdditiveBlending, depthWrite: false }));
  scene.add(sparks);
  const flash = new T.PointLight(0xffb340, 0, 4); scene.add(flash); // залгагдах бүрт гялбана
  let si = 0;
  const burst = (at, n = 26) => {
    flash.position.copy(at); flash.intensity = 14;
    for (let k = 0; k < n; k++) {
      const i = si++ % N;
      sp.set([at.x, at.y, at.z], i * 3);
      sv.set([(Math.random() - 0.5) * 3.2, Math.random() * 2.6 + 0.4, (Math.random() - 0.5) * 3.2], i * 3);
      life[i] = 0.5 + Math.random() * 0.45;
    }
  };
  for (let i = 0; i < N; i++) sp[i * 3 + 1] = -99;

  const outBack = (t) => { const c = 1.9; return 1 + (c + 1) * (t - 1) ** 3 + c * (t - 1) ** 2; };
  const outCubic = (t) => 1 - (1 - t) ** 3;
  const clamp01 = (t) => Math.min(1, Math.max(0, t));
  const fly = 0.62, assembled = 2.36 + fly, total = assembled + 2.7;
  const portrait = innerWidth / innerHeight < 0.85;
  let start = performance.now(), last = start, raf = 0, ended = false, brandShown = false;
  const look = new T.Vector3(0, portrait ? 1.55 : 1.35, 0); // робот дээд хэсэгт, бичиг хөлийн доор

  const resize = () => { camera.aspect = innerWidth / innerHeight; camera.updateProjectionMatrix(); renderer.setSize(innerWidth, innerHeight, false); };
  resize(); addEventListener("resize", resize);

  const finish = () => {
    if (ended) return; ended = true;
    el.classList.add("out");
    setTimeout(() => {
      cancelAnimationFrame(raf); removeEventListener("resize", resize);
      renderer.dispose(); scene.traverse((o) => { o.geometry?.dispose(); });
      el.remove(); document.documentElement.classList.remove("intro-on");
      onDone?.();
    }, 650);
  };
  el.querySelector(".intro-skip").onclick = finish;
  addEventListener("keydown", function esc(e) { if (e.key === "Escape") { removeEventListener("keydown", esc); finish(); } });

  const frame = (now) => {
    raf = requestAnimationFrame(frame);
    const t = (now - start) / 1000, dt = Math.min(0.05, (now - last) / 1000); last = now;
    for (const p of parts) { // нисэж ирээд газар бүртээ «тачиг» хийнэ
      const k = clamp01((t - p.at) / fly);
      p.m.visible = t >= p.at - 0.02;
      if (!p.m.visible) continue;
      const e = outBack(k), r = 1 - outCubic(k);
      p.m.position.lerpVectors(p.from, p.to, e);
      p.m.rotation.set(p.rot.x + p.spin.x * r, p.rot.y + p.spin.y * r, p.rot.z + p.spin.z * r);
      if (k >= 1 && !p.done) { p.done = true; burst(p.m.getWorldPosition(new T.Vector3())); }
    }
    core.emissiveIntensity = t > 1.28 + fly ? Math.min(2.6, (t - 1.28 - fly) * 6) * (0.85 + 0.15 * Math.sin(t * 6)) : 0;
    if (t > assembled) { // нүд анивчиж асна → цагираг тэлнэ → гараа даллана
      const a = t - assembled;
      eyes.emissiveIntensity = a < 0.12 || (a > 0.22 && a < 0.3) ? 0 : 3.2;
      const rk = clamp01(a / 1.1);
      ring.scale.setScalar(1 + rk * 2.4); ring.material.opacity = rk < 1 ? 0.9 * (1 - rk) : 0;
      const wave = clamp01((a - 0.35) / 0.35), down = clamp01((a - 2.05) / 0.35);
      arms[1].rotation.z = (2.55 * outCubic(wave) - 2.55 * outCubic(down)) + (wave >= 1 && down <= 0 ? Math.sin((a - 0.7) * 11) * 0.32 : 0);
      robot.position.y = Math.sin(a * 2.4) * 0.03;
      if (!brandShown && a > 0.3) { brandShown = true; el.classList.add("show-brand"); }
    }
    // Камер: доороос хажуу талаас эхэлж урд тал руу эргэлдэнэ, эцэст нь бага зэрэг ойртоно
    const c = outCubic(clamp01(t / (assembled + 0.4)));
    const ang = (1 - c) * 1.35 + 0.18 * Math.sin(t * 0.5) * c, rad = (portrait ? 13.4 : 11) - c * 1.7 - clamp01((t - assembled) / 3) * 0.45;
    camera.position.set(Math.sin(ang) * rad, 0.9 + c * 1.25, Math.cos(ang) * rad);
    camera.lookAt(look);
    for (let i = 0; i < N; i++) { // оч: таталцлаар унаж бүдгэрнэ
      if (life[i] <= 0) continue;
      life[i] -= dt;
      sv[i * 3 + 1] -= 6.5 * dt;
      sp[i * 3] += sv[i * 3] * dt; sp[i * 3 + 1] += sv[i * 3 + 1] * dt; sp[i * 3 + 2] += sv[i * 3 + 2] * dt;
      if (life[i] <= 0 || sp[i * 3 + 1] < 0) { life[i] = 0; sp[i * 3 + 1] = -99; }
    }
    sgeo.attributes.position.needsUpdate = true;
    flash.intensity *= Math.pow(0.0008, dt); // хурдан унтарна
    grid.material.opacity = 0.35 + 0.2 * Math.sin(t * 1.6);
    renderer.render(scene, camera);
    if (t > total) finish();
  };
  raf = requestAnimationFrame(frame);
  return { skip: finish };
}
