// Интроны дуу — файлгүй, WebAudio-оор нийлэгжүүлнэ (хүч ба итгэл): сансрын гүн дуу → хуягт машины нүргээн → хувирах
// үеийн хурцдах хэмнэл, бөмбөр, төмөр залгаас → робот бүрэлдэх мөчийн «БРААМ» → цэргийн бөмбөрийн өнхрөлт → ялалтын
// гуулин хөг (B♭ – C – D мажор). Хөгжим интроны цагтай уялдана: start(интроны хугацаа, аудио хугацаа) — дундаас нь дуу
// асаавал тэр мөчөөс үргэлжилнэ. OfflineAudioContext-д ч ажиллана (туршилтад урьдчилан бичиж шалгана).
const hz = (m) => 440 * 2 ** ((m - 69) / 12); // MIDI нот → Гц

export function createScore(ctx, dest, { T0, formed, total }) {
  const sr = ctx.sampleRate, BEAT = (formed - T0) / 6; // хувиралт 6 цохилт (≈160 bpm): «БРААМ» яг дараагийн цохилтод
  const out = ctx.createGain(), bus = ctx.createGain(), comp = ctx.createDynamicsCompressor(), lim = ctx.createDynamicsCompressor();
  comp.threshold.value = -8; comp.knee.value = 6; comp.ratio.value = 6; comp.attack.value = 0.003; comp.release.value = 0.2; // зөвхөн оргилыг хязгаарлана
  lim.threshold.value = -2; lim.knee.value = 0; lim.ratio.value = 20; lim.attack.value = 0.001; lim.release.value = 0.1; // тасрахаас сэргийлнэ
  bus.gain.value = 0.85; bus.connect(comp).connect(lim).connect(out).connect(dest);
  const noiseBuf = ctx.createBuffer(1, sr * 2, sr), nd = noiseBuf.getChannelData(0);
  for (let i = 0; i < nd.length; i++) nd[i] = Math.random() * 2 - 1;
  const verb = ctx.createConvolver(), ir = ctx.createBuffer(2, Math.floor(sr * 2.6), sr); // цуурай: 2.6 сек
  for (let c = 0; c < 2; c++) { const d = ir.getChannelData(c); for (let i = 0; i < d.length; i++) d[i] = (Math.random() * 2 - 1) * (1 - i / d.length) ** 3; }
  verb.buffer = ir;
  const wet = ctx.createGain(); wet.gain.value = 0.5; wet.connect(verb).connect(bus);

  const gain = (v, to) => { const g = ctx.createGain(); g.gain.value = v; if (to) g.connect(to); return g; };
  const voice = (rev) => { const g = gain(1, bus); if (rev) g.connect(gain(rev, wet)); return g; }; // хуурай + цуурай
  const filt = (type, f, q, to) => { const b = ctx.createBiquadFilter(); b.type = type; b.frequency.value = f; b.Q.value = q; b.connect(to); return b; };
  const osc = (type, f, t, end, to, det = 0) => { const o = ctx.createOscillator(); o.type = type; o.frequency.value = f; o.detune.value = det; o.connect(to); o.start(t); o.stop(end); o.t = t; return o; };
  const noise = (t, end, to) => { const s = ctx.createBufferSource(); s.buffer = noiseBuf; s.loop = true; s.connect(to); s.start(t, Math.random() * 1.5); s.stop(end); return s; };
  // Цохилтын дугтуй: a сек-д оргилд хүрээд d сек-д унтарна; swell: зөөлөн орж, зөөлөн гарна
  const glide = (o, f1, t1) => { o.frequency.setValueAtTime(o.frequency.value, o.t); o.frequency.exponentialRampToValueAtTime(f1, t1); return o; }; // давтамж гулсах (эхлэх цэгтэй)
  const hit = (g, t, peak, a, d) => { g.gain.setValueAtTime(0.0001, t); g.gain.exponentialRampToValueAtTime(peak, t + a); g.gain.exponentialRampToValueAtTime(0.0001, t + a + d); return g; };
  const swell = (g, t, end, peak, a, r) => { g.gain.setValueAtTime(0, t); g.gain.linearRampToValueAtTime(peak, t + a); g.gain.setValueAtTime(peak, Math.max(t + a, end - r)); g.gain.linearRampToValueAtTime(0, end); return g; };

  // ---- Хэрэгслүүд
  const drone = (t, end) => { // сансрын гүн дуу (D1) ба агаарын шуугиан
    const g = swell(gain(0, voice(0.35)), t, end, 0.07, 1.4, 1), lp = filt("lowpass", 150, 1.2, g);
    osc("sawtooth", hz(26), t, end, lp, -7); osc("sawtooth", hz(26), t, end, lp, 6); osc("sine", hz(38), t, end, gain(0.4, g));
    osc("sine", 0.11, t, end, gain(60, lp.frequency));
    noise(t, end, filt("bandpass", 650, 0.7, gain(0.4, g)));
  };
  const space = (t, end) => { // нимгэн хөг (D–A–D): эхэнд орон зай мэдрүүлнэ
    const lp = filt("lowpass", 1400, 0.5, swell(gain(0, voice(0.8)), t, end, 0.03, 1.2, 1.2));
    for (const m of [62, 69, 74]) for (const d of [-9, 9]) osc("triangle", hz(m), t, end, lp, d);
  };
  const engine = (t, stop) => { // хуягт машин: дизель нүргээн ойртож чангарна, дугуйн шуугиан, агаарын тоормосны «пшш»
    const end = stop + 0.45, peak = t + (stop - t) * 0.7, g = gain(0, voice(0.15)), am = gain(1, g);
    g.gain.setValueAtTime(0.0001, t); g.gain.exponentialRampToValueAtTime(0.2, peak); g.gain.exponentialRampToValueAtTime(0.0001, end);
    const lp = filt("lowpass", 170, 3, am);
    lp.frequency.setValueAtTime(170, t); lp.frequency.linearRampToValueAtTime(700, peak); lp.frequency.linearRampToValueAtTime(140, end);
    for (const [type, f, det] of [["sawtooth", 36, 0], ["sawtooth", 36, 16], ["square", 18, 0]]) {
      const o = osc(type, f, t, end, lp, det);
      o.frequency.setValueAtTime(f * 0.85, t); o.frequency.linearRampToValueAtTime(f * 1.45, peak); o.frequency.linearRampToValueAtTime(f * 0.75, end);
    }
    glide(osc("square", 10, t, end, gain(0.3, am.gain)), 18, peak); // цилиндрийн цохилт
    noise(t, end, filt("bandpass", 320, 0.9, gain(0.5, g)));
    noise(stop - 0.06, stop + 0.6, filt("highpass", 3200, 0.7, hit(gain(0, voice(0.35)), stop - 0.06, 0.1, 0.012, 0.5)));
  };
  const whoosh = (t, dur, f0, f1, peak) => {
    const g = gain(0, voice(0.4)); g.gain.setValueAtTime(0.0001, t); g.gain.exponentialRampToValueAtTime(peak, t + dur * 0.55); g.gain.exponentialRampToValueAtTime(0.0001, t + dur);
    const bp = filt("bandpass", f0, 1.2, g); bp.frequency.setValueAtTime(f0, t); bp.frequency.exponentialRampToValueAtTime(f1, t + dur);
    noise(t, t + dur + 0.05, bp);
  };
  const servos = (t, end) => { // серво моторуудын шүгэл
    const bp = filt("bandpass", 1300, 3, swell(gain(0, voice(0.2)), t, end, 0.025, 0.15, 0.2));
    for (const det of [0, 11]) {
      const o = osc("sawtooth", 300, t, end, bp, det);
      for (let x = t; x < end; x += 0.09 + Math.random() * 0.12) o.frequency.setTargetAtTime(240 + Math.random() * 420, x, 0.025);
    }
  };
  const pluck = (t, m, v, cut, len) => { // хурцдах хэмнэлийн нот: октавтай хөрөө долгион, резонанстай шүүлтүүр
    const lp = filt("lowpass", cut, 4, hit(gain(0, voice(0.2)), t, 0.11 * v, 0.004, len));
    osc("sawtooth", hz(m), t, t + len + 0.05, lp, -6); osc("sawtooth", hz(m + 12), t, t + len + 0.05, lp, 6);
  };
  const drum = (t, v) => { // том бөмбөр
    glide(osc("sine", 130, t, t + 0.7, hit(gain(0, voice(0.35)), t, 0.6 * v, 0.004, 0.55)), 46, t + 0.22);
    noise(t, t + 0.1, filt("lowpass", 1600, 0.8, hit(gain(0, voice(0.3)), t, 0.2 * v, 0.002, 0.07)));
  };
  const riser = (t, end) => { // «БРААМ» руу өсөх шуугиан
    const g = gain(0, voice(0.5)); g.gain.setValueAtTime(0.0001, t); g.gain.exponentialRampToValueAtTime(0.2, end); g.gain.linearRampToValueAtTime(0, end + 0.02);
    const bp = filt("bandpass", 300, 1.8, g); bp.frequency.setValueAtTime(300, t); bp.frequency.exponentialRampToValueAtTime(7000, end);
    noise(t, end + 0.03, bp);
    glide(osc("sawtooth", hz(38), t, end + 0.03, filt("lowpass", 2000, 1, gain(0.3, g))), hz(62), end);
  };
  const cymbal = (t, peak, d) => noise(t, t + d + 0.1, filt("highpass", 5500, 0.6, hit(gain(0, voice(0.6)), t, peak, 0.004, d)));
  const curve = new Float32Array(1024).map((_, i) => Math.tanh(2.4 * (i / 511.5 - 1)) / Math.tanh(2.4));
  const braam = (t) => { // робот бүрэлдэх мөч: D-ийн квинт аккорд, барзгар хөрөө долгион, шүүлтүүр огцом нээгдэнэ + дэд цохилт + цан
    const end = t + 3.4, g = gain(0, voice(0.5));
    g.gain.setValueAtTime(0.0001, t); g.gain.exponentialRampToValueAtTime(0.55, t + 0.05); g.gain.setTargetAtTime(0.0001, t + 0.5, 0.7);
    const lp = filt("lowpass", 110, 3, g); lp.frequency.setValueAtTime(110, t); lp.frequency.exponentialRampToValueAtTime(1600, t + 0.15); lp.frequency.setTargetAtTime(240, t + 0.25, 0.5);
    const sh = ctx.createWaveShaper(); sh.curve = curve; sh.connect(lp);
    for (const m of [26, 33, 38, 45, 50]) for (const d of [-10, 0, 9]) osc("sawtooth", hz(m), t, end, gain(0.2, sh), d);
    glide(osc("sine", 92, t, t + 2, hit(gain(0, voice(0.1)), t, 0.45, 0.005, 1.7)), 31, t + 1.2);
    noise(t, t + 0.6, filt("lowpass", 1000, 0.7, hit(gain(0, voice(0.45)), t, 0.32, 0.003, 0.5)));
    cymbal(t, 0.14, 2.6);
  };
  const powerOn = (t) => { // нүд асах
    const g = hit(gain(0, voice(0.4)), t, 0.06, 0.03, 0.45);
    glide(osc("sine", 200, t, t + 0.55, g), 1600, t + 0.3);
    glide(osc("triangle", 400, t, t + 0.55, gain(0.4, g)), 3200, t + 0.3);
  };
  const snare = (t, v) => { // цэргийн бөмбөр
    noise(t, t + 0.16, filt("bandpass", 2100, 0.8, hit(gain(0, voice(0.3)), t, v, 0.002, 0.12)));
    osc("triangle", 185, t, t + 0.09, hit(gain(0, voice(0.15)), t, v * 0.55, 0.002, 0.06));
  };
  const brass = (notes, t, end, v) => { // гуулин хөг: 3 хөвөөтэй хөрөө долгион, шүүлтүүрийн «ваа», зөөлөн доргио
    const g = gain(0, voice(0.45));
    g.gain.setValueAtTime(0.0001, t); g.gain.exponentialRampToValueAtTime(0.24 * v, t + 0.06); g.gain.exponentialRampToValueAtTime(0.15 * v, t + 0.45);
    g.gain.setValueAtTime(0.15 * v, Math.max(t + 0.45, end - 0.4)); g.gain.exponentialRampToValueAtTime(0.0001, end);
    const lp = filt("lowpass", 380, 1.2, g); lp.frequency.setValueAtTime(380, t); lp.frequency.exponentialRampToValueAtTime(3400, t + 0.08); lp.frequency.setTargetAtTime(1500, t + 0.1, 0.3);
    const vib = gain(0); vib.gain.setValueAtTime(0, t); vib.gain.linearRampToValueAtTime(7, t + 0.7); osc("sine", 5.3, t, end, vib);
    for (const m of notes) for (const d of [-7, 0, 7]) vib.connect(osc("sawtooth", hz(m), t, end, gain(0.3, lp), d).detune);
  };
  const strings = (notes, t, end) => {
    const lp = filt("lowpass", 2200, 0.6, swell(gain(0, voice(0.7)), t, end, 0.08, 0.5, 0.6));
    for (const m of notes) for (const d of [-11, 11]) osc("sawtooth", hz(m), t, end, lp, d);
  };
  const timpani = (m, t, v) => {
    const f = hz(m), g = hit(gain(0, voice(0.35)), t, 0.55 * v, 0.004, 1.3);
    glide(osc("sine", f * 1.2, t, t + 1.5, g), f, t + 0.07);
    glide(osc("sine", f * 1.8, t, t + 0.9, gain(0.3, g)), f * 1.5, t + 0.07);
    noise(t, t + 0.09, filt("lowpass", 1000, 0.9, hit(gain(0, voice(0.25)), t, 0.14 * v, 0.002, 0.06)));
  };
  const bell = (m, t) => { const g = hit(gain(0, voice(0.7)), t, 0.05, 0.003, 1.4); osc("sine", hz(m), t, t + 1.5, g); osc("sine", hz(m) * 2.01, t, t + 0.8, gain(0.25, g)); };
  const clank = (t, k) => { // төмөр хэсэг залгагдах «чанг»: тэгш бус харьцаатай металл давтамжууд + бүдүүн цохилт
    const base = (230 + Math.random() * 240) / Math.sqrt(k), g = hit(gain(0, voice(0.45)), t, 0.09 * k, 0.003, 0.32);
    [1, 2.76, 5.4, 8.93].forEach((r, i) => osc(i ? "sine" : "triangle", base * r, t, t + 0.4, gain(1 / (i + 1), g)));
    glide(osc("sine", 150, t, t + 0.2, hit(gain(0, voice(0.1)), t, 0.4 * k, 0.003, 0.12)), 55, t + 0.1);
    noise(t, t + 0.05, filt("highpass", 2500, 0.8, hit(gain(0, voice(0.2)), t, 0.12 * k, 0.001, 0.03)));
  };
  const tick = (t) => osc("square", 1800 + Math.random() * 600, t, t + 0.05, filt("bandpass", 2600, 2.5, hit(gain(0, voice(0.12)), t, 0.03, 0.002, 0.035)));

  let started = false, lastClank = -1;
  return {
    start(tIntro, tCtx) {
      if (started) return;
      started = true;
      const now = ctx.currentTime, at = (τ) => tCtx + (τ - tIntro), F = formed, END = total + 0.7;
      const once = (τ, fn) => { const t = at(τ); if (t >= now - 0.01) fn(Math.max(t, now)); };          // нэг удаагийн: өнгөрсөн бол алгасна
      const span = (τ0, τ1, fn) => { const a = Math.max(at(τ0), now), b = at(τ1); if (b - a > 0.3) fn(a, b); }; // үргэлжлэх: дундаас бол одооноос
      span(0, END, drone);
      span(0, T0 + 1.2, space);
      if (tIntro < 0.4) engine(Math.max(at(0.05), now), at(T0 - 0.15));
      once(T0 - 0.05, (t) => whoosh(t, 0.7, 200, 2800, 0.12));
      span(T0, F - 0.05, servos);
      for (let i = 0; i < 12; i++) once(T0 + i * BEAT / 2, (t) => pluck(t, i % 8 === 7 ? 41 : 38, i % 4 === 0 ? 1 : i % 2 ? 0.55 : 0.75, 320 * 10 ** (i / 11), BEAT * 0.45));
      for (const [b, v] of [[0, 1], [2, 0.7], [3, 0.8], [4, 0.85], [4.5, 0.7], [5, 0.9], [5.5, 0.8], [5.75, 0.9]]) once(T0 + b * BEAT, (t) => drum(t, v));
      span(F - 1.6, F, riser);
      once(F, braam);
      once(F + 0.1, powerOn);
      const bb = F + 3 * BEAT, c = F + 5 * BEAT, d = F + 7 * BEAT; // ялалтын хөг: B♭ (брэнд гарна) → C → D
      for (let x = F + 0.45; x < bb - 0.02;) { const k = (x - F - 0.45) / (bb - F - 0.45); once(x, (t) => snare(t, 0.05 + 0.22 * k ** 1.5)); x += 0.075 - 0.035 * k; }
      once(bb, (t) => { brass([34, 46, 50, 53, 58, 62], t, at(c) + 0.05, 1); timpani(46, t, 1); cymbal(t, 0.1, 1.8); });
      once(c, (t) => { brass([36, 48, 52, 55, 60, 64], t, at(d) + 0.05, 1.05); timpani(48, t, 0.9); });
      for (let x = d - 0.36; x < d - 0.02; x += 0.045) once(x, (t) => timpani(38, t, 0.18 + 0.5 * (1 - (d - x) / 0.36)));
      span(d, END, (a, b) => { brass([38, 50, 54, 57, 62, 66, 69], a, b, 1.3); strings([62, 66, 69, 74], a, b); });
      once(d, (t) => { timpani(38, t, 1.1); cymbal(t, 0.12, 2.2); [74, 78, 81, 86].forEach((m, i) => bell(m, t + 0.12 + i * 0.07)); });
    },
    clank(k = 1, t = ctx.currentTime) { if (started && t - lastClank > 0.035) { lastClank = t; clank(t, k); } },
    tick(t = ctx.currentTime) { if (started) tick(t); },
    mute(m) { out.gain.setTargetAtTime(m ? 0 : 1, ctx.currentTime, 0.05); },
    fadeOut(sec = 0.6) { out.gain.setTargetAtTime(0, ctx.currentTime, sec / 4); },
  };
}
