/* Нүүрний hero дэвсгэр: Three.js-ийн 3D цэгийн сүлжээ (particles.js маягийн, гүнтэй). Камер урагш нисэх мэт —
   цэгүүд ойртож ирж, ард гармагц дахин алсад гарч ирнэ; ойрхон цэгүүд зураасаар холбогдоно; алс хэсэг цагаан
   манан руу бүдгэрнэ; хулганаар бага зэрэг эргэнэ. Hero харагдахгүй / таб нуугдвал зогсоно; хөдөлгөөн багасгах
   тохиргоонд нэг л кадр зурна. Зөвхөн нэвтрээгүй зочинд (app.js-ээс ачаална). */
(function () {
  const host = document.querySelector(".home-top");
  if (!host || document.documentElement.classList.contains("authed")) return;
  const reduced = matchMedia("(prefers-reduced-motion: reduce)").matches;
  const v = document.querySelector('script[src*="/static/app.js"]')?.src.split("v=")[1] || "";
  const s = document.createElement("script");
  s.src = "/static/vendor/three-r128.min.js?v=" + v; s.async = true; s.onload = init;
  document.head.append(s);

  function init() {
    const T = window.THREE; if (!T) return;
    const canvas = document.createElement("canvas");
    canvas.className = "hero-fx"; canvas.setAttribute("aria-hidden", "true");
    host.prepend(canvas);
    let renderer;
    try { renderer = new T.WebGLRenderer({ canvas, alpha: true, antialias: true, powerPreference: "low-power" }); }
    catch { canvas.remove(); return; } // WebGL байхгүй — энгийн дэвсгэр хэвээр
    renderer.setPixelRatio(Math.min(1.5, window.devicePixelRatio || 1));
    renderer.setClearColor(0xffffff, 0);

    const scene = new T.Scene();
    scene.fog = new T.Fog(0xffffff, 160, 1450);
    const cam = new T.PerspectiveCamera(60, 1, 1, 2200);
    cam.position.set(0, 0, 400);

    const mobile = innerWidth < 700;
    const N = mobile ? 120 : 260, W = mobile ? 420 : 720, H = mobile ? 560 : 420, DEPTH = 1500, LINK = mobile ? 175 : 170, SPEED = 55;
    const pos = new Float32Array(N * 3), col = new Float32Array(N * 3);
    const blue = new T.Color("#2f56d6"), sky = new T.Color("#7b95ff"), amber = new T.Color("#f0a83a");
    const place = (i, z) => { pos[i * 3] = (Math.random() - 0.5) * W * 2; pos[i * 3 + 1] = (Math.random() - 0.5) * H * 2; pos[i * 3 + 2] = z; };
    for (let i = 0; i < N; i++) {
      place(i, 400 - Math.random() * DEPTH);
      const c = Math.random() < 0.12 ? amber : Math.random() < 0.4 ? sky : blue;
      col[i * 3] = c.r; col[i * 3 + 1] = c.g; col[i * 3 + 2] = c.b;
    }
    const g = new T.BufferGeometry();
    g.setAttribute("position", new T.BufferAttribute(pos, 3));
    g.setAttribute("color", new T.BufferAttribute(col, 3));
    // Дугуй, зөөлөн ирмэгтэй цэг
    const sp = document.createElement("canvas"); sp.width = sp.height = 64;
    const cx = sp.getContext("2d"), gr = cx.createRadialGradient(32, 32, 0, 32, 32, 32);
    gr.addColorStop(0, "rgba(255,255,255,1)"); gr.addColorStop(0.45, "rgba(255,255,255,.9)"); gr.addColorStop(1, "rgba(255,255,255,0)");
    cx.fillStyle = gr; cx.beginPath(); cx.arc(32, 32, 32, 0, 7); cx.fill();
    const pm = new T.PointsMaterial({ size: mobile ? 11 : 9, map: new T.CanvasTexture(sp), vertexColors: true, transparent: true, depthWrite: false, sizeAttenuation: true, alphaTest: 0.01 });
    const world = new T.Group(); scene.add(world);
    const pts = new T.Points(g, pm); pts.frustumCulled = false; world.add(pts);

    // Холбоос зураас: ойр байх тусам тод (цагаан дэвсгэр рүү өнгө бүдгэрнэ)
    const MAXL = N * 6, lp = new Float32Array(MAXL * 6), lc = new Float32Array(MAXL * 6);
    const lg = new T.BufferGeometry();
    lg.setAttribute("position", new T.BufferAttribute(lp, 3).setUsage(T.DynamicDrawUsage));
    lg.setAttribute("color", new T.BufferAttribute(lc, 3).setUsage(T.DynamicDrawUsage));
    const lines = new T.LineSegments(lg, new T.LineBasicMaterial({ vertexColors: true, transparent: true, opacity: 0.8 }));
    lines.frustumCulled = false; world.add(lines);
    const L = new T.Color("#2f56d6");

    let mx = 0, my = 0, tx = 0, ty = 0;
    addEventListener("pointermove", (e) => { tx = e.clientX / innerWidth - 0.5; ty = e.clientY / innerHeight - 0.5; }, { passive: true });

    const step = (dt) => {
      const adv = SPEED * dt;
      for (let i = 0; i < N; i++) { const z = pos[i * 3 + 2] + adv; if (z > cam.position.z - 40) place(i, z - DEPTH); else pos[i * 3 + 2] = z; }
      g.attributes.position.needsUpdate = true;
      let k = 0;
      for (let i = 0; i < N && k < MAXL; i++) {
        const ax = pos[i * 3], ay = pos[i * 3 + 1], az = pos[i * 3 + 2];
        for (let j = i + 1; j < N && k < MAXL; j++) {
          const dx = ax - pos[j * 3], dy = ay - pos[j * 3 + 1], dz = az - pos[j * 3 + 2], d2 = dx * dx + dy * dy + dz * dz;
          if (d2 >= LINK * LINK) continue;
          const t = Math.min(1, (1 - Math.sqrt(d2) / LINK) * 1.7), r = 1 - (1 - L.r) * t, gg = 1 - (1 - L.g) * t, b = 1 - (1 - L.b) * t, o = k * 6;
          lp[o] = ax; lp[o + 1] = ay; lp[o + 2] = az; lp[o + 3] = pos[j * 3]; lp[o + 4] = pos[j * 3 + 1]; lp[o + 5] = pos[j * 3 + 2];
          lc[o] = lc[o + 3] = r; lc[o + 1] = lc[o + 4] = gg; lc[o + 2] = lc[o + 5] = b;
          k++;
        }
      }
      lg.setDrawRange(0, k * 2);
      lg.attributes.position.needsUpdate = true; lg.attributes.color.needsUpdate = true;
      world.rotation.z += 0.025 * dt;
      mx += (tx - mx) * 0.04; my += (ty - my) * 0.04;
      cam.position.x = mx * 70; cam.position.y = -my * 50; cam.lookAt(mx * 20, -my * 14, 0);
    };

    const resize = () => { const w = host.clientWidth, h = host.clientHeight; renderer.setSize(w, h, false); cam.aspect = w / Math.max(1, h); cam.updateProjectionMatrix(); renderer.render(scene, cam); };
    let run = false, visible = true, last = 0, raf = 0;
    const frame = (now) => { raf = 0; const dt = Math.min(0.05, (now - last) / 1000); last = now; step(dt); renderer.render(scene, cam); if (run) raf = requestAnimationFrame(frame); };
    const sync = () => {
      const want = visible && !document.hidden && !reduced;
      if (want && !run) { run = true; last = performance.now(); raf = requestAnimationFrame(frame); }
      else if (!want && run) { run = false; if (raf) cancelAnimationFrame(raf); raf = 0; }
    };
    new IntersectionObserver(([e]) => { visible = e.isIntersecting; sync(); }).observe(host);
    document.addEventListener("visibilitychange", sync);
    if (window.ResizeObserver) new ResizeObserver(resize).observe(host); else addEventListener("resize", resize);
    step(0); resize();
    requestAnimationFrame(() => canvas.classList.add("on"));
    sync();
  }
})();
