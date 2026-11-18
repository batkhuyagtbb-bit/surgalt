/* surgalt.mn — «Цагаан самбар: багаар ажиллах» (Fabric.js 5.5).
   Зурах, бичих, арилгах; дүрс, тэмдэгт, эможи; зураг (тайрах, лассогоор хайчлах, дэвсгэр арилгах);
   «Багш заана» / «Хувиараа» / «Багаар» горим; бүх өөрчлөлт бодит цагаар (HTTP → сервер → WebSocket).
   app.js-ийн SG.openBoard() энэ файл, Fabric, board.css-ийг хэрэгтэй үед л ачаална. */
(() => {
"use strict";
const SG = window.SG, F = window.fabric;
const { esc, api, toast, Auth, Live } = SG;
const BI = /* Lucide (ISC) */ {"select":"<path d=\"M4.037 4.688a.495.495 0 0 1 .651-.651l16 6.5a.5.5 0 0 1-.063.947l-6.124 1.58a2 2 0 0 0-1.438 1.435l-1.579 6.126a.5.5 0 0 1-.947.063z\"/>","hand":"<path d=\"M18 11V6a2 2 0 0 0-2-2a2 2 0 0 0-2 2\"/><path d=\"M14 10V4a2 2 0 0 0-2-2a2 2 0 0 0-2 2v2\"/><path d=\"M10 10.5V6a2 2 0 0 0-2-2a2 2 0 0 0-2 2v8\"/><path d=\"M18 8a2 2 0 1 1 4 0v6a8 8 0 0 1-8 8h-2c-2.8 0-4.5-.86-5.99-2.34l-3.6-3.6a2 2 0 0 1 2.83-2.82L7 15\"/>","pen":"<path d=\"M13 21h8\"/><path d=\"M21.174 6.812a1 1 0 0 0-3.986-3.987L3.842 16.174a2 2 0 0 0-.5.83l-1.321 4.352a.5.5 0 0 0 .623.622l4.353-1.32a2 2 0 0 0 .83-.497z\"/>","marker":"<path d=\"m9 11-6 6v3h9l3-3\"/><path d=\"m22 12-4.6 4.6a2 2 0 0 1-2.8 0l-5.2-5.2a2 2 0 0 1 0-2.8L14 4\"/>","eraser":"<path d=\"M21 21H8a2 2 0 0 1-1.42-.587l-3.994-3.999a2 2 0 0 1 0-2.828l10-10a2 2 0 0 1 2.829 0l5.999 6a2 2 0 0 1 0 2.828L12.834 21\"/><path d=\"m5.082 11.09 8.828 8.828\"/>","shapes":"<path d=\"M8.3 10a.7.7 0 0 1-.626-1.079L11.4 3a.7.7 0 0 1 1.198-.043L16.3 8.9a.7.7 0 0 1-.572 1.1Z\"/><rect x=\"3\" y=\"14\" width=\"7\" height=\"7\" rx=\"1\"/><circle cx=\"17.5\" cy=\"17.5\" r=\"3.5\"/>","arrow":"<path d=\"M13 5H19V11\"/><path d=\"M19 5L5 19\"/>","text":"<path d=\"M12 4v16\"/><path d=\"M4 7V5a1 1 0 0 1 1-1h14a1 1 0 0 1 1 1v2\"/><path d=\"M9 20h6\"/>","sticky":"<path d=\"M21 9a2.4 2.4 0 0 0-.706-1.706l-3.588-3.588A2.4 2.4 0 0 0 15 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2z\"/><path d=\"M15 3v5a1 1 0 0 0 1 1h5\"/>","image":"<path d=\"M16 5h6\"/><path d=\"M19 2v6\"/><path d=\"M21 11.5V19a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h7.5\"/><path d=\"m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21\"/><circle cx=\"9\" cy=\"9\" r=\"2\"/>","icons":"<path d=\"M21 9a2.4 2.4 0 0 0-.706-1.706l-3.588-3.588A2.4 2.4 0 0 0 15 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2z\"/><path d=\"M15 3v5a1 1 0 0 0 1 1h5\"/><path d=\"M8 13h.01\"/><path d=\"M16 13h.01\"/><path d=\"M10 16s.8 1 2 1c1.3 0 2-1 2-1\"/>","emoji":"<path d=\"M15 10V9\"/><path d=\"M7.084 14.302a5.12 5.12 0 009.833 0 .24.24 0 00-.235-.302H7.32a.24.24 0 00-.235.302\"/><path d=\"M9 10V9\"/><circle cx=\"12\" cy=\"12\" r=\"10\"/>","laser":"<path d=\"m21.64 3.64-1.28-1.28a1.21 1.21 0 0 0-1.72 0L2.36 18.64a1.21 1.21 0 0 0 0 1.72l1.28 1.28a1.2 1.2 0 0 0 1.72 0L21.64 5.36a1.2 1.2 0 0 0 0-1.72\"/><path d=\"m14 7 3 3\"/><path d=\"M5 6v4\"/><path d=\"M19 14v4\"/><path d=\"M10 2v2\"/><path d=\"M7 8H3\"/><path d=\"M21 16h-4\"/><path d=\"M11 3H9\"/>","undo":"<path d=\"M9 14 4 9l5-5\"/><path d=\"M4 9h10.5a5.5 5.5 0 0 1 5.5 5.5a5.5 5.5 0 0 1-5.5 5.5H11\"/>","redo":"<path d=\"m15 14 5-5-5-5\"/><path d=\"M20 9H9.5A5.5 5.5 0 0 0 4 14.5A5.5 5.5 0 0 0 9.5 20H13\"/>","zin":"<circle cx=\"11\" cy=\"11\" r=\"8\"/><line x1=\"21\" x2=\"16.65\" y1=\"21\" y2=\"16.65\"/><line x1=\"11\" x2=\"11\" y1=\"8\" y2=\"14\"/><line x1=\"8\" x2=\"14\" y1=\"11\" y2=\"11\"/>","zout":"<circle cx=\"11\" cy=\"11\" r=\"8\"/><line x1=\"21\" x2=\"16.65\" y1=\"21\" y2=\"16.65\"/><line x1=\"8\" x2=\"14\" y1=\"11\" y2=\"11\"/>","fit":"<path d=\"M3 7V5a2 2 0 0 1 2-2h2\"/><path d=\"M17 3h2a2 2 0 0 1 2 2v2\"/><path d=\"M21 17v2a2 2 0 0 1-2 2h-2\"/><path d=\"M7 21H5a2 2 0 0 1-2-2v-2\"/>","download":"<path d=\"M12 15V3\"/><path d=\"M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4\"/><path d=\"m7 10 5 5 5-5\"/>","lock":"<rect width=\"18\" height=\"11\" x=\"3\" y=\"11\" rx=\"2\" ry=\"2\"/><path d=\"M7 11V7a5 5 0 0 1 10 0v4\"/>","unlock":"<rect width=\"18\" height=\"11\" x=\"3\" y=\"11\" rx=\"2\" ry=\"2\"/><path d=\"M7 11V7a5 5 0 0 1 9.9-1\"/>","copy":"<rect width=\"14\" height=\"14\" x=\"8\" y=\"8\" rx=\"2\" ry=\"2\"/><path d=\"M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2\"/>","front":"<rect x=\"8\" y=\"8\" width=\"8\" height=\"8\" rx=\"2\"/><path d=\"M4 10a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2\"/><path d=\"M14 20a2 2 0 0 0 2 2h4a2 2 0 0 0 2-2v-4a2 2 0 0 0-2-2\"/>","back":"<rect x=\"14\" y=\"14\" width=\"8\" height=\"8\" rx=\"2\"/><rect x=\"2\" y=\"2\" width=\"8\" height=\"8\" rx=\"2\"/><path d=\"M7 14v1a2 2 0 0 0 2 2h1\"/><path d=\"M14 7h1a2 2 0 0 1 2 2v1\"/>","users":"<path d=\"M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2\"/><path d=\"M16 3.128a4 4 0 0 1 0 7.744\"/><path d=\"M22 21v-2a4 4 0 0 0-3-3.87\"/><circle cx=\"9\" cy=\"7\" r=\"4\"/>","grid":"<rect width=\"7\" height=\"7\" x=\"3\" y=\"3\" rx=\"1\"/><rect width=\"7\" height=\"7\" x=\"14\" y=\"3\" rx=\"1\"/><rect width=\"7\" height=\"7\" x=\"14\" y=\"14\" rx=\"1\"/><rect width=\"7\" height=\"7\" x=\"3\" y=\"14\" rx=\"1\"/>","x":"<path d=\"M18 6 6 18\"/><path d=\"m6 6 12 12\"/>","more":"<circle cx=\"12\" cy=\"12\" r=\"1\"/><circle cx=\"19\" cy=\"12\" r=\"1\"/><circle cx=\"5\" cy=\"12\" r=\"1\"/>","crop":"<path d=\"M6 2v14a2 2 0 0 0 2 2h14\"/><path d=\"M18 22V8a2 2 0 0 0-2-2H2\"/>","wand":"<path d=\"M15 4V2\"/><path d=\"M15 16v-2\"/><path d=\"M8 9h2\"/><path d=\"M20 9h2\"/><path d=\"M17.8 11.8 19 13\"/><path d=\"M15 9h.01\"/><path d=\"M17.8 6.2 19 5\"/><path d=\"m3 21 9-9\"/><path d=\"M12.2 6.2 11 5\"/>","lasso":"<path d=\"M3.704 14.467a10 8 0 1 1 3.115 2.375\"/><path d=\"M7 22a5 5 0 0 1-2-3.994\"/><circle cx=\"5\" cy=\"16\" r=\"2\"/>","check":"<path d=\"M20 6 9 17l-5-5\"/>","pause":"<rect x=\"14\" y=\"3\" width=\"5\" height=\"18\" rx=\"1\"/><rect x=\"5\" y=\"3\" width=\"5\" height=\"18\" rx=\"1\"/>","play":"<path d=\"M5 5a2 2 0 0 1 3.008-1.728l11.997 6.998a2 2 0 0 1 .003 3.458l-12 7A2 2 0 0 1 5 19z\"/>","eye":"<path d=\"M2.062 12.348a1 1 0 0 1 0-.696 10.75 10.75 0 0 1 19.876 0 1 1 0 0 1 0 .696 10.75 10.75 0 0 1-19.876 0\"/><circle cx=\"12\" cy=\"12\" r=\"3\"/>","spot":"<path d=\"M11 6a13 13 0 0 0 8.4-2.8A1 1 0 0 1 21 4v12a1 1 0 0 1-1.6.8A13 13 0 0 0 11 14H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2z\"/><path d=\"M6 14a12 12 0 0 0 2.4 7.2 2 2 0 0 0 3.2-2.4A8 8 0 0 1 10 14\"/><path d=\"M8 6v8\"/>","shuffle":"<path d=\"m18 14 4 4-4 4\"/><path d=\"m18 2 4 4-4 4\"/><path d=\"M2 18h1.973a4 4 0 0 0 3.3-1.7l5.454-8.6a4 4 0 0 1 3.3-1.7H22\"/><path d=\"M2 6h1.972a4 4 0 0 1 3.6 2.2\"/><path d=\"M22 18h-6.041a4 4 0 0 1-3.3-1.8l-.359-.45\"/>","distribute":"<line x1=\"15\" x2=\"15\" y1=\"12\" y2=\"18\"/><line x1=\"12\" x2=\"18\" y1=\"15\" y2=\"15\"/><rect width=\"14\" height=\"14\" x=\"8\" y=\"8\" rx=\"2\" ry=\"2\"/><path d=\"M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2\"/>","plus":"<path d=\"M5 12h14\"/><path d=\"M12 5v14\"/>","minus":"<path d=\"M5 12h14\"/>","upload":"<path d=\"M12 3v12\"/><path d=\"m17 8-5-5-5 5\"/><path d=\"M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4\"/>","link":"<path d=\"M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71\"/><path d=\"M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71\"/>","search":"<path d=\"m21 21-4.34-4.34\"/><circle cx=\"11\" cy=\"11\" r=\"8\"/>","bold":"<path d=\"M6 12h9a4 4 0 0 1 0 8H7a1 1 0 0 1-1-1V5a1 1 0 0 1 1-1h7a4 4 0 0 1 0 8\"/>","fill":"<path d=\"M11 7 6 2\"/><path d=\"M18.992 12H2.041\"/><path d=\"M21.145 18.38A3.34 3.34 0 0 1 20 16.5a3.3 3.3 0 0 1-1.145 1.88c-.575.46-.855 1.02-.855 1.595A2 2 0 0 0 20 22a2 2 0 0 0 2-2.025c0-.58-.285-1.13-.855-1.595\"/><path d=\"m8.5 4.5 2.148-2.148a1.205 1.205 0 0 1 1.704 0l7.296 7.296a1.205 1.205 0 0 1 0 1.704l-7.592 7.592a3.615 3.615 0 0 1-5.112 0l-3.888-3.888a3.615 3.615 0 0 1 0-5.112L5.67 7.33\"/>","line":"<path d=\"M5 12h14\"/>","out":"<path d=\"m16 17 5-5-5-5\"/><path d=\"M21 12H9\"/><path d=\"M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4\"/>","board":"<path d=\"M2 3h20\"/><path d=\"M21 3v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V3\"/><path d=\"m7 21 5-5 5 5\"/>","restore":"<path d=\"m11 10 3 3\"/><path d=\"M6.5 21A3.5 3.5 0 1 0 3 17.5a2.62 2.62 0 0 1-.708 1.792A1 1 0 0 0 3 21z\"/><path d=\"M9.969 17.031 21.378 5.624a1 1 0 0 0-3.002-3.002L6.967 14.031\"/>","target":"<line x1=\"2\" x2=\"5\" y1=\"12\" y2=\"12\"/><line x1=\"19\" x2=\"22\" y1=\"12\" y2=\"12\"/><line x1=\"12\" x2=\"12\" y1=\"2\" y2=\"5\"/><line x1=\"12\" x2=\"12\" y1=\"19\" y2=\"22\"/><circle cx=\"12\" cy=\"12\" r=\"7\"/><circle cx=\"12\" cy=\"12\" r=\"3\"/>","dash":"<circle cx=\"12\" cy=\"12\" r=\"1\"/><circle cx=\"19\" cy=\"12\" r=\"1\"/><circle cx=\"5\" cy=\"12\" r=\"1\"/>","user":"<path d=\"M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2\"/><circle cx=\"12\" cy=\"7\" r=\"4\"/>","globe":"<circle cx=\"12\" cy=\"12\" r=\"10\"/><path d=\"M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20\"/><path d=\"M2 12h20\"/>","chev":"<path d=\"m6 9 6 6 6-6\"/>","send":"<path d=\"M14.536 21.686a.5.5 0 0 0 .937-.024l6.5-19a.496.496 0 0 0-.635-.635l-19 6.5a.5.5 0 0 0-.024.937l7.93 3.18a2 2 0 0 1 1.112 1.11z\"/><path d=\"m21.854 2.147-10.94 10.939\"/>","trash":"<path d=\"M10 11v6\"/><path d=\"M14 11v6\"/><path d=\"M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6\"/><path d=\"M3 6h18\"/><path d=\"M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2\"/>"};
const bi = (n, s = 20) => `<svg viewBox="0 0 24 24" width="${s}" height="${s}" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${BI[n] || ""}</svg>`;
const V = document.querySelector('script[src*="/static/app.js"]')?.src.split("v=")[1] || "";
Object.assign(BI, /* Lucide (ISC) */ {
  mic: '<path d="M12 19v3"/><path d="M19 10v2a7 7 0 0 1-14 0v-2"/><rect x="9" y="2" width="6" height="13" rx="3"/>',
  micoff: '<path d="M12 19v3"/><path d="M15 9.34V5a3 3 0 0 0-5.68-1.33"/><path d="M16.95 16.95A7 7 0 0 1 5 12v-2"/><path d="M18.89 13.23A7 7 0 0 0 19 12v-2"/><path d="m2 2 20 20"/><path d="M9 9v3a3 3 0 0 0 5.12 2.12"/>',
  chat: '<path d="M2.992 16.342a2 2 0 0 1 .094 1.167l-1.065 3.29a1 1 0 0 0 1.236 1.168l3.413-.998a2 2 0 0 1 1.099.092 10 10 0 1 0-4.777-4.719"/>',
  vol: '<path d="M11 4.702a.705.705 0 0 0-1.203-.498L6.413 7.587A1.4 1.4 0 0 1 5.416 8H3a1 1 0 0 0-1 1v6a1 1 0 0 0 1 1h2.416a1.4 1.4 0 0 1 .997.413l3.383 3.384A.705.705 0 0 0 11 19.298z"/><path d="M16 9a5 5 0 0 1 0 6"/><path d="M19.364 18.364a9 9 0 0 0 0-12.728"/>',
  volx: '<path d="M11 4.702a.705.705 0 0 0-1.203-.498L6.413 7.587A1.4 1.4 0 0 1 5.416 8H3a1 1 0 0 0-1 1v6a1 1 0 0 0 1 1h2.416a1.4 1.4 0 0 1 .997.413l3.383 3.384A.705.705 0 0 0 11 19.298z"/><line x1="22" x2="16" y1="9" y2="15"/><line x1="16" x2="22" y1="9" y2="15"/>',
});

/* ---------- дуу: 16кГц IMA ADPCM (зөвхөн WebRTC холбогдож чадаагүй сонсогчид серверээр) ---------- */
const VSR = 16000, VCH = 3200; // 200 мс-ийн хэсэг
const AD_STEP = [7, 8, 9, 10, 11, 12, 13, 14, 16, 17, 19, 21, 23, 25, 28, 31, 34, 37, 41, 45, 50, 55, 60, 66, 73, 80, 88, 97, 107, 118, 130, 143, 157, 173, 190, 209, 230, 253, 279, 307, 337, 371, 408, 449, 494, 544, 598, 658, 724, 796, 876, 963, 1060, 1166, 1282, 1411, 1552, 1707, 1878, 2066, 2272, 2499, 2749, 3024, 3327, 3660, 4026, 4428, 4871, 5358, 5894, 6484, 7132, 7845, 8630, 9493, 10442, 11487, 12635, 13899, 15289, 16818, 18500, 20350, 22385, 24623, 27086, 29794, 32767];
const AD_IDX = [-1, -1, -1, -1, 2, 4, 6, 8, -1, -1, -1, -1, 2, 4, 6, 8];
// Хэсэг бүр бие даан тайлагдана: [таамаг lo, hi, алхмын индекс] + 4 бит/дээж (алдагдсан хэсэг дараагийнхыг эвдэхгүй).
function adEnc(pcm, idx0) {
  const out = new Uint8Array(3 + (pcm.length >> 1));
  let pred = pcm[0], idx = idx0;
  out[0] = pred & 255; out[1] = (pred >> 8) & 255; out[2] = idx;
  for (let i = 0; i < pcm.length; i++) {
    const step = AD_STEP[idx]; let diff = pcm[i] - pred, code = 0, delta = step >> 3;
    if (diff < 0) { code = 8; diff = -diff; }
    if (diff >= step) { code |= 4; diff -= step; delta += step; }
    if (diff >= step >> 1) { code |= 2; diff -= step >> 1; delta += step >> 1; }
    if (diff >= step >> 2) { code |= 1; delta += step >> 2; }
    pred = Math.max(-32768, Math.min(32767, pred + (code & 8 ? -delta : delta)));
    idx = Math.max(0, Math.min(88, idx + AD_IDX[code]));
    if (i & 1) out[3 + (i >> 1)] |= code << 4; else out[3 + (i >> 1)] = code;
  }
  return { data: out, idx };
}
function adDec(b) {
  let pred = ((b[0] | (b[1] << 8)) << 16) >> 16, idx = Math.min(88, b[2]);
  const n = (b.length - 3) * 2, out = new Float32Array(n);
  for (let i = 0; i < n; i++) {
    const byte = b[3 + (i >> 1)], code = i & 1 ? byte >> 4 : byte & 15, step = AD_STEP[idx];
    let delta = step >> 3;
    if (code & 4) delta += step;
    if (code & 2) delta += step >> 1;
    if (code & 1) delta += step >> 2;
    pred = Math.max(-32768, Math.min(32767, pred + (code & 8 ? -delta : delta)));
    idx = Math.max(0, Math.min(88, idx + AD_IDX[code]));
    out[i] = pred / 32768;
  }
  return out;
}
const b64 = (u8) => { let s = ""; for (let i = 0; i < u8.length; i += 0x8000) s += String.fromCharCode.apply(null, u8.subarray(i, i + 0x8000)); return btoa(s); };
const unb64 = (s) => { const b = atob(s), u = new Uint8Array(b.length); for (let i = 0; i < b.length; i++) u[i] = b.charCodeAt(i); return u; };
const f32cat = (a, b) => { if (!a.length) return b; const o = new Float32Array(a.length + b.length); o.set(a); o.set(b, a.length); return o; };
// resampler — хөтчийн давтамжаас (48/44.1кГц) 16кГц: цонхны дунджаар (энгийн доод шүүлтүүр).
function resampler(rate) {
  const step = rate / VSR; let pos = 0, carry = new Float32Array(0);
  return (input) => {
    const data = f32cat(carry, input), out = [];
    while (pos + step <= data.length) {
      const a = Math.floor(pos), b = Math.max(a + 1, Math.floor(pos + step)); let s = 0;
      for (let i = a; i < b; i++) s += data[i];
      out.push(s / (b - a)); pos += step;
    }
    const keep = Math.floor(pos); carry = data.slice(keep); pos -= keep;
    return Float32Array.from(out);
  };
}
const ICE = [{ urls: ["stun:stun.l.google.com:19302", "stun:stun1.l.google.com:19302"] }, { urls: "stun:stun.cloudflare.com:3478" }];
const iceDone = (pc) => pc.iceGatheringState === "complete" ? Promise.resolve() : new Promise((res) => {
  const t = setTimeout(res, 2500);
  pc.addEventListener("icegatheringstatechange", () => { if (pc.iceGatheringState === "complete") { clearTimeout(t); res(); } });
});

/* ---------- тогтмол ---------- */
const COLORS = ["#121829", "#e11d48", "#f97316", "#eab308", "#16a34a", "#06b6d4", "#1f3c8f", "#7c3aed", "#db2777", "#92400e", "#64748b", "#ffffff"];
const NOTES = ["#fff3a3", "#ffd6e0", "#d4f5e0", "#d6e8ff", "#ffe2c2", "#eadcff"];
const WIDTHS = [2, 4, 8, 14];
const FONTS = [["Manrope", "Энгийн"], ["Caveat", "Гар бичмэл"], ["Georgia", "Сонгодог"], ["Consolas", "Код"]];
const SIZES = [18, 28, 44, 72];
const EMOJI_FONT = '"Segoe UI Emoji","Apple Color Emoji","Noto Color Emoji",sans-serif';
const TEAMS = [["🦊", "Үнэгнүүд", "#ea580c"], ["🐼", "Пандагууд", "#334155"], ["🦁", "Арслангууд", "#d97706"], ["🐸", "Мэлхийнүүд", "#16a34a"],
  ["🐳", "Халимууд", "#0284c7"], ["🦋", "Эрвээхэйнүүд", "#7c3aed"], ["🐯", "Барууд", "#c2410c"], ["🐧", "Оцон шувууд", "#0f172a"],
  ["🐰", "Туулайнууд", "#db2777"], ["🦅", "Бүргэдүүд", "#92400e"], ["🐬", "Далайн гахайнууд", "#0891b2"], ["🐝", "Зөгийнүүд", "#ca8a04"]];
const EMOJI = [
  ["Түгээмэл", "⭐ 👍 ✅ 💯 🏆 ❤️ 👏 🎉 🌟 🥇 😀 🤔 ❓ ❗ ❌ 🔥 💡 🎯"],
  ["Сэтгэл", "😀 😃 😄 😁 😆 😅 😂 🤣 😊 😇 🙂 😉 😍 🥰 😘 😋 😛 🤪 🤓 😎 🥳 🤩 😏 😞 😔 😟 😕 🙁 😣 😫 😩 🥺 😢 😭 😤 😠 😡 🤯 😳 😱 😨 🤗 🤔 🤭 🤫 😶 😐 😬 🙄 😮 😲 🥱 😴 😷 🤒 🤠"],
  ["Гар", "👍 👎 👏 🙌 👐 🤲 🤝 🙏 ✌️ 🤞 🤟 🤘 👌 🤏 👈 👉 👆 👇 ☝️ ✋ 🤚 🖐️ 👋 💪 ✍️"],
  ["Амьтан", "🐶 🐱 🐭 🐹 🐰 🦊 🐻 🐼 🐨 🐯 🦁 🐮 🐷 🐸 🐵 🐔 🐧 🐦 🐤 🦆 🦅 🦉 🐴 🦄 🐝 🐛 🦋 🐌 🐞 🐜 🐢 🐍 🦖 🦕 🐙 🦀 🐠 🐟 🐬 🐳 🦈 🐊 🐘 🦒 🦓 🐪 🦘 🐑 🐐 🦌 🐕 🐈 🐓 🦚 🦜"],
  ["Байгаль", "🌸 🌼 🌻 🌺 🌹 🌷 🌱 🌿 ☘️ 🍀 🍁 🍂 🍃 🌳 🌲 🌴 🌵 🌾 🍄 🌍 🌎 🌏 🌙 ⭐ 🌟 ✨ ⚡ 🔥 🌈 ☀️ ⛅ ☁️ 🌧️ ⛈️ ❄️ ☃️ 💧 🌊 🌋 ⛰️ 🏔️"],
  ["Хоол", "🍎 🍐 🍊 🍋 🍌 🍉 🍇 🍓 🫐 🍒 🍑 🥭 🍍 🥥 🥝 🍅 🥕 🌽 🥒 🥦 🧄 🧅 🥔 🍞 🧀 🥚 🍳 🥞 🍗 🍖 🍔 🍟 🍕 🌭 🥪 🌮 🍜 🍝 🍣 🍦 🍩 🍪 🎂 🍰 🧁 🍫 🍬 🍭 🥛 ☕ 🧃"],
  ["Сургууль", "📚 📖 📝 ✏️ 🖍️ 🖊️ 📏 📐 ✂️ 📌 📎 🗂️ 📒 📓 📕 📗 📘 📙 🎒 🏫 🧑‍🏫 👩‍🏫 👨‍🏫 🧑‍🎓 🎓 🔬 🔭 🧪 🧫 🧬 🧮 💡 🖥️ 💻 ⌨️ 🗺️ 🧭 ⏰ ⌛"],
  ["Тоглоом", "⚽ 🏀 🏈 ⚾ 🎾 🏐 🏉 🎱 🏓 🏸 ⛳ 🏹 🥊 🥋 ⛸️ 🎿 🏆 🥇 🥈 🥉 🏅 🎖️ 🎯 🎲 🧩 🎮 🪁 🎨 🎭 🎤 🎧 🎹 🥁 🎸 🎺 🎻"],
  ["Тээвэр", "🚗 🚕 🚙 🚌 🚎 🏎️ 🚓 🚑 🚒 🚐 🛻 🚚 🚜 🚲 🛴 🏍️ 🚂 🚆 🚇 ✈️ 🚀 🛸 🚁 ⛵ 🚤 🛳️ 🚢 ⚓ 🏠 🏡 🏢 🏥 🏦 🏪 🏰 🗼 ⛺"],
  ["Тэмдэг", "✅ ❌ ❓ ❗ ⭕ 🔴 🟠 🟡 🟢 🔵 🟣 ⚫ ⚪ 🟥 🟧 🟨 🟩 🟦 🟪 ⬛ ⬜ ❤️ 🧡 💛 💚 💙 💜 🖤 🤍 💯 💢 💥 💫 💬 💭 🔔 🎵 🎶 ➕ ➖ ✖️ ➗ 🟰 ♾️ 🔟 1️⃣ 2️⃣ 3️⃣ 4️⃣ 5️⃣ 🇲🇳"],
];
const BGS = [["dots", "Цэгэн"], ["grid", "Нүдэн тор"], ["lines", "Шугамтай"], ["none", "Хоосон"]];
const ERASABLE = new Set(["pen", "marker", "line", "shape", "icon", "emoji", "text"]);
const PROPS = ["id", "k", "lk", "by", "fm", "note", "cr"];
const IMG_KINDS = [["", "Бүгд"], ["clip", "Зурмал"], ["photo", "Гэрэл зураг"]];
const IMG_IDEAS = ["нарны аймаг", "наран цэцэг", "хүний бие", "амьтан", "монгол гэр", "галт уул", "үлэг гүрвэл", "дэлхий"];
const U = 100; // дүрсийн суурь хэмжээ (scaleX/Y-ээр томруулна, хүрээ strokeUniform)

const regular = (n, rot = -Math.PI / 2) => Array.from({ length: n }, (_, i) => [0.5 + 0.5 * Math.cos(rot + i * 2 * Math.PI / n), 0.5 + 0.5 * Math.sin(rot + i * 2 * Math.PI / n)]);
const starPts = (n = 5) => Array.from({ length: n * 2 }, (_, i) => { const a = -Math.PI / 2 + i * Math.PI / n, k = i % 2 ? 0.2 : 0.5; return [0.5 + k * Math.cos(a), 0.5 + k * Math.sin(a)]; });
const SH = {
  rect: ["Тэгш өнцөгт", { r: 0 }],
  round: ["Бөөрөнхий өнцөгт", { r: 18 }],
  ellipse: ["Тойрог", { e: 1 }],
  triangle: ["Гурвалжин", { p: [[0.5, 0], [1, 1], [0, 1]] }],
  rtri: ["Тэгш өнцөгт гурвалжин", { p: [[0, 0], [1, 1], [0, 1]] }],
  diamond: ["Ромб", { p: [[0.5, 0], [1, 0.5], [0.5, 1], [0, 0.5]] }],
  para: ["Параллелограмм", { p: [[0.25, 0], [1, 0], [0.75, 1], [0, 1]] }],
  trap: ["Трапец", { p: [[0.22, 0], [0.78, 0], [1, 1], [0, 1]] }],
  pentagon: ["Таван өнцөгт", { p: regular(5) }],
  hexagon: ["Зургаан өнцөгт", { p: regular(6, 0) }],
  star: ["Од", { p: starPts(5) }],
  heart: ["Зүрх", { d: "M50 90C22 70 4 54 4 32 4 16 16 6 30 6c9 0 16 5 20 12 4-7 11-12 20-12 14 0 26 10 26 26 0 22-18 38-46 58z" }],
  cloud: ["Үүл", { d: "M27 82C13 82 3 72 3 59c0-12 9-21 21-22 2-14 14-25 28-25 12 0 22 7 26 18 11 1 19 10 19 22 0 16-11 30-27 30z" }],
  bubble: ["Ярианы бөмбөлөг", { d: "M14 6h72c7 0 12 5 12 12v40c0 7-5 12-12 12H46L26 92l3-22H14C7 70 2 65 2 58V18C2 11 7 6 14 6z" }],
  arrowR: ["Бүдүүн сум", { p: [[0, 0.3], [0.6, 0.3], [0.6, 0.06], [1, 0.5], [0.6, 0.94], [0.6, 0.7], [0, 0.7]] }],
  plus: ["Загалмай", { p: [[0.36, 0], [0.64, 0], [0.64, 0.36], [1, 0.36], [1, 0.64], [0.64, 0.64], [0.64, 1], [0.36, 1], [0.36, 0.64], [0, 0.64], [0, 0.36], [0.36, 0.36]] }],
  bolt: ["Аянга", { p: [[0.58, 0], [0.12, 0.56], [0.46, 0.56], [0.36, 1], [0.88, 0.4], [0.54, 0.4], [0.66, 0]] }],
  moon: ["Сар", { d: "M64 6C42 12 28 30 28 52s14 38 36 44c-6 2-11 3-17 3C23 99 4 78 4 52S23 4 47 4c6 0 12 1 17 2z" }],
};
const LINES = { line: ["Шугам", '<path d="M5 19 19 5"/>'], arrow: ["Сум", '<path d="M5 19 19 5M10 5h9v9"/>'], darrow: ["Хоёр талт сум", '<path d="M5 19 19 5M10 5h9v9M14 19H5v-9"/>'] };
const shSVG = (k) => {
  const s = SH[k][1];
  const inner = "r" in s ? `<rect x="6" y="18" width="88" height="64" rx="${s.r * 0.8}"/>` : s.e ? '<ellipse cx="50" cy="50" rx="44" ry="36"/>'
    : s.p ? `<polygon points="${s.p.map(([x, y]) => `${6 + x * 88},${6 + y * 88}`).join(" ")}"/>` : `<path d="${s.d}"/>`;
  return `<svg viewBox="-4 -4 108 108" width="26" height="26" fill="none" stroke="currentColor" stroke-width="7" stroke-linejoin="round" aria-hidden="true">${inner}</svg>`;
};
function makeShape(k) {
  const s = SH[k][1];
  if ("r" in s) return new F.Rect({ width: U, height: U, rx: s.r, ry: s.r });
  if (s.e) return new F.Ellipse({ rx: U / 2, ry: U / 2 });
  if (s.p) return new F.Polygon(s.p.map(([x, y]) => ({ x: x * U, y: y * U })));
  return new F.Path(s.d);
}

/* ---------- жижиг туслахууд ---------- */
const rid = (n = 12) => { const a = crypto.getRandomValues(new Uint8Array(n)); let s = ""; for (const x of a) s += "abcdefghijklmnopqrstuvwxyz0123456789"[x % 36]; return s; };
const clamp = (v, a, b) => Math.max(a, Math.min(b, v));
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const hexA = (hex, a) => { const n = parseInt(String(hex).slice(1), 16); return `rgba(${(n >> 16) & 255},${(n >> 8) & 255},${n & 255},${a})`; };
const throttle = (fn, ms) => {
  let last = 0, timer = null, args = null;
  return (...a) => {
    args = a; const now = Date.now();
    if (now - last >= ms) { last = now; fn(...args); }
    else if (!timer) timer = setTimeout(() => { timer = null; last = Date.now(); fn(...args); }, ms - (now - last));
  };
};
const initials = (n) => String(n || "?").trim().split(/\s+/).map((w) => w[0]).join("").slice(0, 2).toUpperCase();
const hue = (s) => { let h = 0; for (const c of String(s)) h = (h * 31 + c.codePointAt(0)) % 360; return h; };
const avatar = (p) => `<span class="bd-av" style="--h:${hue(p.uid || p.name)}" title="${esc(p.name)}">${esc(initials(p.name))}</span>`;
const enliven = (o) => new Promise((res) => { try { F.util.enlivenObjects([o], (arr) => res(arr[0] || null)); } catch { res(null); } });
const geoKey = (j) => j.type === "path" ? JSON.stringify(j.path) : j.type === "polygon" || j.type === "polyline" ? JSON.stringify(j.points) : j.type === "group" ? JSON.stringify(j.objects) : "";
const teamOf = (n) => TEAMS[n % TEAMS.length];
const ERASER_CURSOR = `url("data:image/svg+xml,${encodeURIComponent('<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24"><circle cx="12" cy="12" r="9" fill="rgba(255,255,255,.7)" stroke="#121829" stroke-width="1.5"/></svg>')}") 12 12, crosshair`;
function busyToast(msg) {
  const t = document.createElement("div"); t.className = "toast bd-busy"; t.innerHTML = `<span class="bd-spin"></span>${esc(msg)}`;
  document.getElementById("toasts")?.append(t);
  return () => t.remove();
}
let fontsP = null;
function ensureFonts() {
  if (fontsP) return fontsP;
  if (!document.querySelector('link[data-bd-font]')) {
    const l = document.createElement("link"); l.rel = "stylesheet"; l.dataset.bdFont = 1;
    l.href = "https://fonts.googleapis.com/css2?family=Caveat:wght@500;700&display=swap"; document.head.append(l);
  }
  fontsP = Promise.race([Promise.all(["500 22px Caveat", "700 22px Caveat", "500 22px Manrope"].map((f) => document.fonts?.load(f).catch(() => {}))), sleep(2500)]);
  return fontsP;
}

/* ---------- Fabric-ийн анхдагч тохиргоо ---------- */
F.Object.prototype.set({ originX: "center", originY: "center", transparentCorners: false, cornerColor: "#ffffff", cornerStrokeColor: "#1f3c8f",
  borderColor: "#1f3c8f", cornerStyle: "circle", cornerSize: 12, touchCornerSize: 30, padding: 4, borderScaleFactor: 1.6 });
F.Image.prototype.srcFromAttribute = true; // src-ийг "/board-img/..." харьцангуй замаар хадгална
F.Object.NUM_FRACTION_DIGITS = 2;
if (!F.Sticky) {
  // Наалт тэмдэглэл: өнгөт цаас + гар бичмэл текст (Textbox дээр).
  F.Sticky = F.util.createClass(F.Textbox, {
    type: "sticky", note: NOTES[0], objectCaching: false,
    initialize(text, o) {
      this.callSuper("initialize", text, Object.assign({ width: 200, fontSize: 24, fontFamily: "Caveat", fill: "#2b2400", textAlign: "center", originY: "top", padding: 16, k: "note" }, o));
    },
    calcTextHeight() { return Math.max(this.callSuper("calcTextHeight"), this.width * 0.72); },
    _renderBackground(ctx) {
      const p = this.padding, w = this.width + p * 2, h = this.height + p * 2;
      ctx.save(); ctx.shadowColor = "rgba(18,24,41,.22)"; ctx.shadowBlur = 14; ctx.shadowOffsetY = 5;
      ctx.fillStyle = this.note; ctx.fillRect(-w / 2, -h / 2, w, h); ctx.restore();
      ctx.fillStyle = "rgba(0,0,0,.06)"; ctx.fillRect(-w / 2, -h / 2, w, 12);
    },
    toObject(p) { return this.callSuper("toObject", ["note"].concat(p || [])); },
  });
  F.Sticky.fromObject = (o, cb) => F.Object._fromObject("Sticky", Object.assign({}, o, { styles: F.util.stylesFromArray(o.styles, o.text) }), cb, "text");
}

let cur = null;
function open(o) { try { cur?.destroy(); } catch {} cur = board(o); }
window.SGBoard = { open, get current() { return cur; } };

function shellHTML(title) {
  return `<header class="bd-top">
    <button type="button" class="bd-ib bd-close" data-a="close" title="Хаах">${bi("x")}</button>
    <div class="bd-ttl"><b title="${esc(title)}">${esc(title)}</b><span class="bd-room"></span></div>
    <div class="bd-rooms" hidden></div>
    <div class="bd-modes" role="radiogroup" aria-label="Ажиллах горим" hidden>
      <button type="button" data-mode="teach" role="radio">${bi("board", 17)}<span>Багш заана</span></button>
      <button type="button" data-mode="solo" role="radio">${bi("user", 17)}<span>Хувиараа</span></button>
      <button type="button" data-mode="team" role="radio">${bi("users", 17)}<span>Багаар</span></button>
    </div>
    <div class="bd-sp"></div>
    <div class="bd-peers"></div>
    <div class="bd-grp bd-hist"><button type="button" class="bd-ib" data-a="undo" title="Буцаах (Ctrl+Z)" disabled>${bi("undo")}</button><button type="button" class="bd-ib" data-a="redo" title="Дахин хийх (Ctrl+Y)" disabled>${bi("redo")}</button></div>
    <div class="bd-grp bd-zoom"><button type="button" class="bd-ib" data-a="zout" title="Жижигрүүлэх (−)">${bi("zout")}</button><button type="button" class="bd-zv" data-a="fit" title="Бүгдийг харах (0)">100%</button><button type="button" class="bd-ib" data-a="zin" title="Томруулах (+)">${bi("zin")}</button></div>
    <button type="button" class="bd-pill bd-lockbtn" data-a="lock" hidden>${bi("pause", 17)}<span>Түр зогсоох</span></button>
    <select class="bd-vto" aria-label="Хэнд ярих" title="Микрофоноор хэнд ярих" hidden></select>
    <button type="button" class="bd-pill bd-micbtn" data-a="mic" hidden></button>
    <button type="button" class="bd-pill bd-handbtn" data-a="hand" hidden></button>
    <button type="button" class="bd-pill bd-chatbtn" data-a="chat" hidden>${bi("chat", 17)}<span>Чат</span><i class="bd-badge" hidden></i></button>
    <button type="button" class="bd-pill bd-panelbtn" data-a="panel" hidden>${bi("grid", 17)}<span>Анги</span><i class="bd-badge" hidden></i></button>
    <button type="button" class="bd-ib" data-a="more" data-pop="more" title="Бусад">${bi("more")}</button>
  </header>
  <div class="bd-stage">
    <div class="bd-wrap bg-dots"><canvas></canvas></div>
    <nav class="bd-tools" aria-label="Хэрэгслүүд">
      <button type="button" data-t="select" title="Сонгох, зөөх (V)">${bi("select")}</button>
      <button type="button" data-t="hand" title="Гүйлгэх (H · Space)">${bi("hand")}</button>
      <i class="bd-sep"></i>
      <button type="button" data-t="pen" title="Үзэг (P)">${bi("pen")}</button>
      <button type="button" data-t="marker" title="Тодруулагч (M)">${bi("marker")}</button>
      <button type="button" data-t="eraser" title="Баллуур (E)">${bi("eraser")}</button>
      <i class="bd-sep"></i>
      <button type="button" data-t="shape" data-pop="shapes" title="Дүрс, шугам, сум (S)">${bi("shapes")}</button>
      <button type="button" data-t="text" title="Текст (T)">${bi("text")}</button>
      <button type="button" data-t="note" title="Наалт тэмдэглэл (N)">${bi("sticky")}</button>
      <button type="button" data-pop="image" title="Зураг (I)">${bi("image")}</button>
      <button type="button" data-pop="icons" title="Дүрс тэмдэгт">${bi("icons")}</button>
      <button type="button" data-pop="emoji" title="Эможи">${bi("emoji")}</button>
      <i class="bd-sep bd-tonly"></i>
      <button type="button" data-t="laser" class="bd-tonly" title="Лазер заагч (K)">${bi("laser")}</button>
    </nav>
    <div class="bd-pop" data-p="shapes" hidden></div>
    <div class="bd-pop bd-pop-img" data-p="image" hidden></div>
    <div class="bd-pop bd-pop-lg" data-p="icons" hidden></div>
    <div class="bd-pop bd-pop-lg" data-p="emoji" hidden></div>
    <div class="bd-pop bd-menu" data-p="more" hidden></div>
    <div class="bd-style" hidden></div>
    <div class="bd-banner" hidden></div>
    <div class="bd-lockov" hidden><div><b>⏸</b><h3>Багш самбарыг түр зогсоолоо</h3><p>Багшдаа анхаарлаа хандуулаарай 👀</p></div></div>
    <div class="bd-loading"><span class="bd-spin"></span>Самбарыг ачаалж байна…</div>
    <aside class="bd-panel" hidden aria-label="Анги"></aside>
    <div class="bd-voice" hidden aria-live="polite"></div>
    <aside class="bd-chat" hidden aria-label="Багийн чат">
      <header class="bd-ph"><div><b>${bi("chat", 18)} Багийн чат</b><small class="bd-ch-sub"></small></div><button type="button" class="bd-ib" data-a="chat" title="Хаах">${bi("x", 18)}</button></header>
      <div class="bd-ch-teams" hidden></div>
      <div class="bd-ch-tabs" role="tablist"></div>
      <div class="bd-ch-members"></div>
      <div class="bd-ch-list" role="log" aria-live="polite"></div>
      <form class="bd-ch-f"><label class="bd-ch-all" hidden><input type="checkbox"> 📢 Бүх багт нэг дор илгээх</label>
        <textarea rows="1" maxlength="1000" placeholder="Бичих…" aria-label="Мессеж"></textarea><button type="submit" class="bd-ch-send" title="Илгээх (Enter)">${bi("send", 18)}</button></form>
    </aside>
    <input type="file" class="bd-file" accept="image/*" multiple hidden>
  </div>`;
}

/* ================= самбар ================= */
function board({ base, lid, bid, title = "Цагаан самбар", teacher: owner = false }) {
  ensureFonts();
  const root = document.createElement("div");
  root.className = "bd"; root.setAttribute("role", "dialog"); root.setAttribute("aria-modal", "true"); root.setAttribute("aria-label", title);
  root.innerHTML = shellHTML(title);
  document.body.append(root);
  document.documentElement.classList.add("bd-on");
  const q = (s) => root.querySelector(s), qa = (s) => [...root.querySelectorAll(s)];
  const stage = q(".bd-stage"), wrap = q(".bd-wrap"), styleBar = q(".bd-style"), banner = q(".bd-banner"), panel = q(".bd-panel"), fileIn = q(".bd-file");
  const cv = new F.Canvas(q(".bd-wrap canvas"), {
    preserveObjectStacking: true, stopContextMenu: true, fireMiddleClick: true, targetFindTolerance: 6, enableRetinaScaling: true,
    selectionColor: "rgba(31,60,143,.07)", selectionBorderColor: "#1f3c8f", selectionLineWidth: 1.5,
  });

  let me = { uid: "", name: "", teacher: !!owner }, st = { mode: "teach", teams: 4, assign: {}, names: {}, copy: true, lock: false, spot: "", v: 0 };
  let room = "main", ver = 0, canEdit = false, peers = [], dead = false;
  const cid = rid(10);
  const objs = new Map(), gen = new Map();
  let order = [];
  let tool = "pen", shapeKind = "rect";
  const style = { color: "#121829", width: 4, fill: 0, dash: false, font: "Manrope", size: 28, note: NOTES[0], bold: false, bg: "dots" };
  try { Object.assign(style, JSON.parse(localStorage.getItem("sg_board_style") || "{}")); } catch {}
  const saveStyle = () => { try { localStorage.setItem("sg_board_style", JSON.stringify(style)); } catch {} };
  let follow = true, lastView = null;

  /* ---------- эрх (сервертэй ижил дүрэм) ---------- */
  const ownRoomOf = (s) => s.mode === "solo" ? "u:" + me.uid : s.mode === "team" && s.assign?.[me.uid] != null && s.assign[me.uid] < s.teams ? "t:" + s.assign[me.uid] : "";
  const ownRoom = () => ownRoomOf(st);
  const canReadRoom = (r) => me.teacher || r === "main" || r === "u:" + me.uid || (!!st.spot && r === st.spot) || r === ownRoom();
  const canWriteRoom = (r) => me.teacher || (!st.lock && r !== "main" && r === ownRoom());
  const defaultRoom = () => me.teacher ? "main" : st.spot || ownRoom() || "main";
  const editable = (o) => canEdit && (me.teacher || !o.lk);
  function roomLabel(r) {
    if (r === "main") return me.teacher ? "Миний самбар (багш)" : "Багшийн самбар";
    if (r.startsWith("t:")) { const t = teamOf(+r.slice(2)); return `${t[0]} ${t[1]}`; }
    const uid = r.slice(2);
    return uid === me.uid ? "Миний самбар" : (st.names?.[uid] || "Суралцагч");
  }

  /* ---------- синк: өөрчлөлтийг багцалж илгээх ---------- */
  let queue = [], inflight = false, flushT = null, syncing = false, again = false, buffered = [];
  const post = (p, body) => api(base + p, { method: "POST", body });
  function send(op) {
    if (dead || (!canEdit && op.t !== "view")) return;
    queue.push(op);
    if (!flushT && !inflight) flushT = setTimeout(flush, 45);
  }
  function coalesce(ops) {
    const out = [], at = new Map();
    for (const op of ops) {
      if (op.t === "ptr" || op.t === "view") { const i = out.findIndex((x) => x && x.t === op.t); if (i >= 0) out[i] = null; out.push(op); continue; }
      if (op.t === "upd" && at.has(op.id)) { const i = at.get(op.id); out[i] = { ...out[i], o: op.o }; continue; }
      if (op.t === "del" && at.has(op.id)) { const i = at.get(op.id), wasAdd = out[i].t === "add"; out[i] = null; at.delete(op.id); if (wasAdd) continue; }
      out.push(op);
      if (op.t === "add" || op.t === "upd") at.set(op.id, out.length - 1);
    }
    return out.filter(Boolean);
  }
  async function flush() {
    flushT = null;
    if (inflight || !queue.length || dead && !queue.length) return;
    let batch = coalesce(queue); queue = [];
    if (batch.length > 280) { queue = batch.slice(280); batch = batch.slice(0, 280); }
    inflight = true;
    const r0 = room;
    try {
      const r = await post("/ops", { room: r0, cid, ops: batch });
      if (r?.full) toast("Самбар дүүрсэн — заримыг нь устгаад дахин оролдоно уу", true);
    } catch (e) {
      if (e.status === 429 || !e.status) { // сүлжээ тасарсан: дараа нь дахин (түр дохиог хаяна)
        queue = batch.filter((x) => x.t !== "ptr" && x.t !== "view").concat(queue);
        inflight = false; flushT = setTimeout(flush, e.status === 429 ? 900 : 2500); return;
      }
      toast(e.message, true);
      if (e.status === 403 || e.status === 400) resync();
    }
    inflight = false;
    if (queue.length) flushT = setTimeout(flush, 25);
  }
  async function flushAll() {
    clearTimeout(flushT); flushT = null;
    for (let i = 0; i < 200 && inflight; i++) await sleep(25);
    if (queue.length) await flush();
  }

  /* ---------- серверээс ирсэн өөрчлөлт ---------- */
  function onOps(d) {
    if (me.teacher) markThumb(d.room);
    if (d.room !== room || d.v <= ver) return;
    if (d.v !== ver + 1) { resync(); return; }
    ver = d.v;
    if (d.cid === cid) return; // өөрийн илгээснийх
    for (const op of d.ops || []) {
      if (op.t === "add" || op.t === "upd") upsert(op.o);
      else if (op.t === "del") removeId(op.id);
      else if (op.t === "front" || op.t === "back") reorder(op.id, op.t === "front");
      else if (op.t === "clear") { resync(); return; }
    }
    cv.requestRenderAll();
  }
  const onLive = (d) => {
    if (dead) return;
    if (d.type === "reconnected") { resync(); loadChat(); return; }
    if (d.type !== "board" || d.lid !== lid || d.bid !== bid) return;
    if (d.ev === "ops") { if (syncing) buffered.push(d); else onOps(d); }
    else if (d.ev === "fx") { if (d.room === room && d.cid !== cid) onFx(d.ops || []); }
    else if (d.ev === "state") onState(d.state);
    else if (d.ev === "chat") onChat(d);
    else if (d.ev === "voice") onVoice(d.voice);
    else if (d.ev === "rtc") onRtc(d.rtc);
    else if (d.ev === "peers") { peers = d.peers || []; renderPeers(); renderPanel(); peersVoice(); renderChat(); }
    else if (d.ev === "reload") { if (d.rooms?.includes(room)) resync(); (d.rooms || []).forEach(markThumb); }
    else if (d.ev === "touch") markThumb(d.room);
  };

  function busyObj(o) {
    const t = cv._currentTransform?.target;
    return o.isEditing || (t && (t === o || (t.type === "activeSelection" && o.group === t)));
  }
  const SKIP = new Set(["type", "version", "path", "points", "objects", "src", "filters", "resizeFilter", "crossOrigin", "styles"]);
  function upsert(o) {
    if (!o?.id) return;
    if (!order.includes(o.id)) order.push(o.id);
    const curObj = objs.get(o.id);
    if (curObj && busyObj(curObj)) return; // хэрэглэгч яг одоо барьж байна — өөрийнх нь ялна
    const g = (gen.get(o.id) || 0) + 1; gen.set(o.id, g);
    if (curObj && curObj.type === o.type && (o.type !== "image" || curObj.getSrc() === o.src) && curObj.__geo === geoKey(o)) {
      if (curObj.group) cv.discardActiveObject();
      const p = {}; for (const k in o) if (!SKIP.has(k)) p[k] = o[k];
      curObj.set(p); curObj.setCoords(); prepare(curObj);
      return;
    }
    enliven(o).then((obj) => {
      if (!obj || gen.get(o.id) !== g || dead) return;
      obj.__geo = geoKey(o);
      prepare(obj);
      const old = objs.get(o.id);
      if (old) {
        const i = cv.getObjects().indexOf(old), act = cv.getActiveObject() === old;
        if (old.group) cv.discardActiveObject();
        cv.remove(old); cv.insertAt(obj, Math.max(0, i), false);
        if (act) cv.setActiveObject(obj);
      } else place(obj);
      objs.set(o.id, obj);
      cv.requestRenderAll();
    });
  }
  function place(obj) {
    const i = order.indexOf(obj.id), list = cv.getObjects();
    for (let j = i + 1; i >= 0 && j < order.length; j++) {
      const nx = objs.get(order[j]), k = nx ? list.indexOf(nx) : -1;
      if (k >= 0) { cv.insertAt(obj, k, false); return; }
    }
    cv.add(obj);
  }
  function removeId(id) {
    gen.set(id, (gen.get(id) || 0) + 1);
    const o = objs.get(id);
    if (o) { if (o.group) cv.discardActiveObject(); if (o.isEditing) o.exitEditing(); cv.remove(o); objs.delete(id); }
    const i = order.indexOf(id); if (i >= 0) order.splice(i, 1);
  }
  function reorder(id, front) {
    const i = order.indexOf(id); if (i < 0) return;
    order.splice(i, 1); front ? order.push(id) : order.unshift(id);
    const o = objs.get(id); if (o) front ? cv.bringToFront(o) : cv.sendToBack(o);
  }
  function prepare(o) {
    const ed = editable(o);
    o.set({ selectable: ed, evented: ed, hasControls: ed, lockMovementX: !ed, lockMovementY: !ed, hoverCursor: ed ? "move" : "default" });
    o.borderDashArray = o.lk ? [6, 4] : null;
    if (o.type === "path" && o.k !== "shape") o.perPixelTargetFind = true;
    if (o.type === "i-text" || o.type === "sticky") o.set({ editable: ed, cursorColor: "#1f3c8f", editingBorderColor: "#1f3c8f", selectionColor: "rgba(31,60,143,.2)" });
  }
  function ser(o) {
    let j;
    if (o.group) { // олноор сонгосон үед байрлал бүлэгтэй харьцангуй — абсолют болгоно
      const t = F.util.saveObjectTransform(o);
      F.util.addTransformToObject(o, o.group.calcTransformMatrix());
      j = o.toObject(PROPS);
      o.set(t); o.setCoords();
    } else j = o.toObject(PROPS);
    o.__geo = geoKey(j);
    return j;
  }
  async function loadObjects(list) {
    cv.discardActiveObject();
    cv.getObjects().slice().forEach((o) => cv.remove(o));
    objs.clear(); order = list.map((o) => o.id);
    for (const id of order) gen.set(id, (gen.get(id) || 0) + 1);
    const made = await Promise.all(list.map(enliven));
    if (dead) return;
    cv.renderOnAddRemove = false;
    made.forEach((obj, i) => { if (!obj) return; obj.__geo = geoKey(list[i]); prepare(obj); cv.add(obj); objs.set(obj.id, obj); });
    cv.renderOnAddRemove = true;
    cv.requestRenderAll();
  }
  async function resync(fit) {
    if (syncing) { again = true; return; }
    syncing = true; buffered = [];
    try {
      const d = await api(base + "?room=" + encodeURIComponent(room));
      if (dead) return;
      applyMeta(d);
      await loadObjects(d.objects || []);
      ver = d.v;
      for (const op of queue) { if (op.t === "add" || op.t === "upd") upsert(op.o); else if (op.t === "del") removeId(op.id); } // илгээгдээгүй өөрчлөлт
      const buf = buffered; buffered = [];
      syncing = false;
      for (const b of buf) if (b.room === room && b.v > ver) onOps(b);
      if (fit) fitView();
    } catch (e) { if (!dead) toast(e.message, true); }
    finally {
      syncing = false;
      if (again) { again = false; resync(); }
    }
  }
  function applyMeta(d) {
    if (d.me) me = d.me;
    if (d.peers) peers = d.peers;
    if (d.room && d.room !== room) room = d.room;
    if (d.state) onState(d.state, true);
  }

  /* ---------- горим, анги ---------- */
  function onState(s, quiet) {
    if (!s) return;
    const prev = st; st = s; st.assign ||= {}; st.names ||= {};
    if (!me.teacher && !quiet) {
      const want = defaultRoom();
      const moved = ownRoomOf(prev) !== ownRoom() || (prev.spot || "") !== (st.spot || "") || !canReadRoom(room);
      if (moved && want !== room) { switchRoom(want); modeToast(prev); }
    }
    const was = canEdit; canEdit = canWriteRoom(room);
    if (was !== canEdit) { cv.getObjects().forEach(prepare); setTool(canEdit ? (["hand", "laser"].includes(tool) ? "pen" : tool) : "hand"); }
    renderChrome();
    chatState(prev, quiet);
    voiceState(prev, quiet);
  }
  function modeToast(prev) {
    if (st.spot && st.spot !== prev.spot) return toast(`📢 Багш «${roomLabel(st.spot)}» самбарыг бүх ангид харуулж байна`);
    if (st.mode === prev.mode && ownRoomOf(prev) === ownRoom()) return;
    if (st.mode === "solo") toast("✏️ Хувиараа ажиллах цаг — өөрийн самбар дээрээ зурна уу");
    else if (st.mode === "team") { const r = ownRoom(); toast(r ? `👥 Багаар ажиллана — та ${roomLabel(r)} багт байна` : "👥 Багаар ажиллана"); }
    else toast("👩‍🏫 Багш тайлбарлаж байна — багшийн самбарыг харна уу");
  }
  async function switchRoom(r) {
    await flushAll();
    if (dead) return;
    room = r; undoS.length = 0; redoS.length = 0; syncUndo();
    if (me.teacher && r.startsWith("t:")) chatTeam = +r.slice(2); // багш багийн самбарыг нээвэл чат нь ч тэр баг
    canEdit = canWriteRoom(room);
    lasers.forEach((l) => l.el.remove()); lasers.clear();
    await resync(true);
    if (room === "main" && follow && lastView) applyView(lastView);
    cv.getObjects().forEach(prepare);
    setTool(canEdit ? (["hand", "laser"].includes(tool) && !me.teacher ? "pen" : tool) : "hand");
    renderChrome();
  }
  const setState = async (body) => {
    try { const d = await post("/state", body); onState(d.state); return d; }
    catch (e) { toast(e.message, true); }
  };

  /* ---------- түүх (буцаах / дахих) ---------- */
  const undoS = [], redoS = [];
  function record(ch) { ch = ch.filter(Boolean); if (!ch.length) return; undoS.push(ch); if (undoS.length > 100) undoS.shift(); redoS.length = 0; syncUndo(); }
  function syncUndo() { q('[data-a="undo"]').disabled = !undoS.length || !canEdit; q('[data-a="redo"]').disabled = !redoS.length || !canEdit; }
  function travel(from, to, useBefore) {
    const ch = from.pop(); if (!ch || !canEdit) return;
    cv.discardActiveObject();
    for (const [id, b, a] of ch) {
      const j = useBefore ? b : a;
      if (j == null) { if (objs.has(id)) { removeId(id); send({ t: "del", id }); } continue; }
      const t = objs.has(id) ? "upd" : "add";
      upsert(j); send({ t, id, o: j });
    }
    to.push(ch); syncUndo(); cv.requestRenderAll(); renderStyle();
  }
  const undo = () => travel(undoS, redoS, true), redo = () => travel(redoS, undoS, false);

  /* ---------- локал өөрчлөлт ---------- */
  function addLocal(o, { onCanvas = false, select = false, hist = true } = {}) {
    o.id = o.id || rid(); o.by = me.uid;
    prepare(o);
    if (!onCanvas) cv.add(o);
    objs.set(o.id, o); if (!order.includes(o.id)) order.push(o.id);
    const j = ser(o);
    send({ t: "add", id: o.id, o: j });
    if (hist) record([[o.id, null, j]]);
    if (select) { if (tool !== "select") setTool("select"); cv.setActiveObject(o); renderStyle(); }
    cv.requestRenderAll();
    return j;
  }
  function restyle(list, fn) {
    const ch = [];
    for (const o of list) {
      if (!editable(o)) continue;
      const b = ser(o); fn(o); o.setCoords(); o.dirty = true;
      const a = ser(o); send({ t: "upd", id: o.id, o: a }); ch.push([o.id, b, a]);
    }
    record(ch); cv.requestRenderAll();
  }
  function delSel() {
    const list = cv.getActiveObjects().filter(editable); if (!list.length) return;
    cv.discardActiveObject();
    record(list.map((o) => { const j = ser(o); removeId(o.id); send({ t: "del", id: o.id }); return [o.id, j, null]; }));
    cv.requestRenderAll(); renderStyle();
  }
  let clip = null;
  async function pasteJSON(list, dx = 28) {
    if (!canEdit || !list?.length) return;
    cv.discardActiveObject();
    const made = [], ch = [];
    for (const j of list) {
      const o = await enliven({ ...j, id: rid(), lk: me.teacher ? j.lk : false, left: j.left + dx, top: j.top + dx });
      if (!o) continue;
      ch.push([o.id, null, addLocal(o, { hist: false })]); made.push(o);
    }
    record(ch);
    if (tool !== "select") setTool("select");
    if (made.length > 1) cv.setActiveObject(new F.ActiveSelection(made, { canvas: cv })); else if (made[0]) cv.setActiveObject(made[0]);
    cv.requestRenderAll(); renderStyle();
  }
  const duplicate = () => pasteJSON(cv.getActiveObjects().filter(editable).map(ser));
  function zorder(front) {
    const list = cv.getActiveObjects().filter(editable);
    for (const o of front ? list : list.slice().reverse()) { reorder(o.id, front); send({ t: front ? "front" : "back", id: o.id }); }
    cv.requestRenderAll();
  }

  /* ---------- Fabric үйл явдал → өөрчлөлт ---------- */
  let before = null;
  const targets = (t) => !t ? [] : t.type === "activeSelection" ? t.getObjects() : [t];
  cv.on("before:transform", (e) => { before = targets(e.transform?.target).map((o) => [o, ser(o)]); });
  const liveMove = throttle(() => {
    const list = targets(cv._currentTransform?.target);
    if (list.length && list.length <= 25) for (const o of list) if (objs.has(o.id)) send({ t: "upd", id: o.id, o: ser(o) });
  }, 90);
  ["object:moving", "object:scaling", "object:rotating", "object:skewing"].forEach((ev) => cv.on(ev, liveMove));
  cv.on("object:modified", (e) => {
    if (!e.transform) return; // текст засварын дуусгавар — доор тусад нь
    const ch = [];
    for (const o of targets(e.target)) {
      if (!objs.has(o.id)) continue;
      const a = ser(o); send({ t: "upd", id: o.id, o: a });
      ch.push([o.id, before?.find(([x]) => x === o)?.[1] || null, a]);
    }
    before = null; record(ch);
  });
  cv.on("path:created", ({ path }) => {
    if (pinched) { cv.remove(path); return; }
    path.set({ k: tool === "marker" ? "marker" : "pen", fill: null });
    addLocal(path, { onCanvas: true });
  });
  let textBefore = null;
  cv.on("text:editing:entered", ({ target }) => { textBefore = [target.id, ser(target)]; });
  const liveText = throttle((o) => { if (objs.has(o.id)) send({ t: "upd", id: o.id, o: ser(o) }); }, 180);
  cv.on("text:changed", ({ target }) => liveText(target));
  cv.on("text:editing:exited", ({ target: o }) => {
    if (!objs.has(o.id)) return;
    const b = textBefore?.[0] === o.id ? textBefore[1] : null; textBefore = null;
    if (!o.text.trim() && o.k !== "note") { // хоосон текст үлдээхгүй
      removeId(o.id); send({ t: "del", id: o.id });
      const last = undoS[undoS.length - 1];
      if (last?.length === 1 && last[0][0] === o.id && last[0][1] == null) { undoS.pop(); syncUndo(); } else if (b?.text) record([[o.id, b, null]]);
    } else {
      const a = ser(o); send({ t: "upd", id: o.id, o: a });
      if (b && b.text !== a.text) record([[o.id, b, a]]);
    }
    cv.requestRenderAll();
  });
  cv.on("selection:created", () => renderStyle());
  cv.on("selection:updated", () => renderStyle());
  cv.on("selection:cleared", () => renderStyle());
  cv.on("mouse:dblclick", (e) => { const o = e.target; if (o?.type === "image" && editable(o)) editImage(o); });

  /* ---------- хэрэгслүүд ---------- */
  function setTool(t, sub) {
    if (!canEdit && t !== "laser") t = "hand";
    if (t === "laser" && !me.teacher) t = "hand";
    tool = t; if (sub) shapeKind = sub;
    cv.isDrawingMode = t === "pen" || t === "marker";
    if (cv.isDrawingMode) {
      const b = new F.PencilBrush(cv);
      b.decimate = 1.2; b.strokeLineCap = "round"; b.strokeLineJoin = "round";
      b.color = t === "marker" ? hexA(style.color === "#ffffff" ? "#eab308" : style.color, 0.38) : style.color;
      b.width = t === "marker" ? Math.max(14, style.width * 4) : style.width;
      cv.freeDrawingBrush = b; cv.freeDrawingCursor = "crosshair";
    }
    const sel = t === "select" && canEdit;
    cv.selection = sel; cv.skipTargetFind = !sel;
    if (!sel && cv.getActiveObject() && !cv.getActiveObject().isEditing) cv.discardActiveObject();
    cv.defaultCursor = t === "hand" ? "grab" : t === "select" ? "default" : t === "eraser" ? ERASER_CURSOR : "crosshair";
    cv.setCursor?.(cv.defaultCursor);
    qa(".bd-tools [data-t]").forEach((b) => { const on = b.dataset.t === t; b.classList.toggle("on", on); b.setAttribute("aria-pressed", on); });
    renderStyle(); cv.requestRenderAll();
  }

  /* ---------- харагдац: томруулах, гүйлгэх ---------- */
  const viewRect = () => { const v = cv.viewportTransform, z = v[0]; return { x: -v[4] / z, y: -v[5] / z, w: cv.width / z, h: cv.height / z }; };
  const viewCenter = () => { const r = viewRect(); return { x: r.x + r.w / 2, y: r.y + r.h / 2 }; };
  function contentRect(list = cv.getObjects()) {
    if (!list.length) return null;
    let x1 = Infinity, y1 = Infinity, x2 = -Infinity, y2 = -Infinity;
    for (const o of list) { const b = o.getBoundingRect(true, true); x1 = Math.min(x1, b.left); y1 = Math.min(y1, b.top); x2 = Math.max(x2, b.left + b.width); y2 = Math.max(y2, b.top + b.height); }
    return { x: x1, y: y1, w: x2 - x1, h: y2 - y1 };
  }
  function fitRect(r, maxZ = 1, pad = 60) {
    const z = clamp(Math.min((cv.width - pad * 2) / Math.max(r.w, 1), (cv.height - pad * 2) / Math.max(r.h, 1), maxZ), 0.1, 6);
    cv.setViewportTransform([z, 0, 0, z, cv.width / 2 - (r.x + r.w / 2) * z, cv.height / 2 - (r.y + r.h / 2) * z]);
    afterView();
  }
  function fitView() {
    const r = contentRect();
    if (r) fitRect(r, 1); else { cv.setViewportTransform([1, 0, 0, 1, cv.width / 2, cv.height / 2]); afterView(); }
  }
  function zoomTo(z, pt) { cv.zoomToPoint(pt || new F.Point(cv.width / 2, cv.height / 2), clamp(z, 0.1, 6)); afterView(true); }
  const sendView = throttle(() => {
    if (!me.teacher || room !== "main" || (st.mode !== "teach" && st.spot !== "main")) return;
    const r = viewRect(); send({ t: "view", o: { x: Math.round(r.x), y: Math.round(r.y), w: Math.round(r.w), h: Math.round(r.h) } });
  }, 280);
  function afterView(user) {
    const v = cv.viewportTransform, z = v[0], g = 24 * z;
    wrap.style.backgroundSize = `${g}px ${g}px`; wrap.style.backgroundPosition = `${v[4]}px ${v[5]}px`;
    q(".bd-zv").textContent = Math.round(z * 100) + "%";
    placeLasers();
    if (user && !me.teacher && room === "main" && follow && lastView) { follow = false; renderBanner(); }
    if (user) sendView();
    cv.requestRenderAll();
  }
  function applyView(r) {
    lastView = r;
    if (!follow || me.teacher || room !== "main") return;
    const z = clamp(Math.min(cv.width / r.w, cv.height / r.h), 0.1, 6);
    cv.setViewportTransform([z, 0, 0, z, cv.width / 2 - (r.x + r.w / 2) * z, cv.height / 2 - (r.y + r.h / 2) * z]);
    afterView();
  }

  /* ---------- лазер заагч ---------- */
  const lasers = new Map();
  function showLaser(key, o) {
    let l = lasers.get(key);
    if (o.h) { if (l) { l.el.remove(); lasers.delete(key); } return; }
    if (!l) { const el = document.createElement("div"); el.className = "bd-laser"; el.innerHTML = `<i></i><span>${esc(o.n || "Багш")}</span>`; stage.append(el); l = { el }; lasers.set(key, l); }
    l.x = o.x; l.y = o.y; clearTimeout(l.t); l.t = setTimeout(() => { l.el.remove(); lasers.delete(key); }, 2500);
    placeLasers();
  }
  function placeLasers() { const v = cv.viewportTransform; for (const l of lasers.values()) l.el.style.transform = `translate(${l.x * v[0] + v[4]}px,${l.y * v[3] + v[5]}px)`; }
  function onFx(ops) { for (const op of ops) { if (op.t === "ptr" && op.o) showLaser("t", op.o); else if (op.t === "view" && op.o) applyView(op.o); } }
  const sendPtr = throttle((p) => send({ t: "ptr", o: { x: Math.round(p.x), y: Math.round(p.y), n: me.name } }), 45);

  /* ---------- хулгана, мэдрэгч ---------- */
  let drag = null, spaceDown = false, pinched = false, pinch = null;
  const clientXY = (e) => { const t = e.touches?.[0] || e.changedTouches?.[0] || e; return { x: t.clientX, y: t.clientY }; };
  cv.on("mouse:down", (opt) => {
    const e = opt.e, pt = cv.getPointer(e), c = clientXY(e);
    if (pinch) return;
    if (spaceDown || tool === "hand" || e.button === 1) { drag = { kind: "pan", x: c.x, y: c.y }; cv.setCursor("grabbing"); return; }
    if (!canEdit && tool !== "laser") return;
    if (tool === "eraser") { drag = { kind: "erase", ch: [] }; eraseAt(pt); return; }
    if (tool === "shape") { drag = { kind: "shape", start: pt, obj: null }; return; }
    if (tool === "text") { addText(pt); return; }
    if (tool === "note") { addNote(pt); return; }
    if (tool === "laser") { drag = { kind: "laser" }; showLaser("me", { x: pt.x, y: pt.y, n: me.name }); sendPtr(pt); }
  });
  cv.on("mouse:move", (opt) => {
    const e = opt.e;
    if (tool === "laser" && me.teacher && !pinch) { const pt = cv.getPointer(e); showLaser("me", { x: pt.x, y: pt.y, n: me.name }); sendPtr(pt); return; }
    if (!drag || pinch) return;
    const pt = cv.getPointer(e), c = clientXY(e);
    if (drag.kind === "pan") { cv.relativePan(new F.Point(c.x - drag.x, c.y - drag.y)); drag.x = c.x; drag.y = c.y; afterView(true); return; }
    if (drag.kind === "erase") { eraseAt(pt); return; }
    if (drag.kind === "shape") shapeDrag(pt, e.shiftKey);
  });
  cv.on("mouse:up", (opt) => {
    if (!drag) return;
    const d = drag; drag = null;
    if (d.kind === "pan") { cv.setCursor(cv.defaultCursor); return; }
    if (d.kind === "erase") { record(d.ch); return; }
    if (d.kind === "shape") shapeEnd(d, cv.getPointer(opt.e));
  });
  cv.on("mouse:out", () => { if (tool === "laser" && me.teacher) { showLaser("me", { h: 1 }); send({ t: "ptr", o: { h: 1 } }); } });
  cv.on("mouse:wheel", (opt) => {
    const e = opt.e; e.preventDefault(); e.stopPropagation();
    if (e.ctrlKey || e.metaKey) zoomTo(cv.getZoom() * 0.999 ** e.deltaY, new F.Point(e.offsetX, e.offsetY));
    else { cv.relativePan(new F.Point(-e.deltaX, -e.deltaY)); afterView(true); }
  });
  // Хоёр хуруугаар томруулах/гүйлгэх (утас, таблет).
  const pinchInfo = (ts) => { const r = wrap.getBoundingClientRect(), a = ts[0], b = ts[1]; return { x: (a.clientX + b.clientX) / 2 - r.left, y: (a.clientY + b.clientY) / 2 - r.top, d: Math.hypot(a.clientX - b.clientX, a.clientY - b.clientY) }; };
  wrap.addEventListener("touchstart", (e) => {
    if (e.touches.length !== 2) return;
    e.preventDefault(); e.stopPropagation();
    if (cv._isCurrentlyDrawing) { cv._isCurrentlyDrawing = false; cv.clearContext(cv.contextTop); pinched = true; }
    if (drag?.obj) { cv.remove(drag.obj); } drag = null;
    cv.isDrawingMode = false; cv.skipTargetFind = true; cv.discardActiveObject();
    pinch = { ...pinchInfo(e.touches), z: cv.getZoom() };
  }, { passive: false, capture: true });
  wrap.addEventListener("touchmove", (e) => {
    if (!pinch || e.touches.length < 2) return;
    e.preventDefault(); e.stopPropagation();
    const p = pinchInfo(e.touches);
    cv.relativePan(new F.Point(p.x - pinch.x, p.y - pinch.y));
    cv.zoomToPoint(new F.Point(p.x, p.y), clamp(pinch.z * p.d / pinch.d, 0.1, 6));
    pinch.x = p.x; pinch.y = p.y; afterView(true);
  }, { passive: false, capture: true });
  const pinchEnd = (e) => { if (pinch && e.touches.length === 0) { pinch = null; setTimeout(() => { pinched = false; }, 50); setTool(tool); } };
  wrap.addEventListener("touchend", pinchEnd, true);
  wrap.addEventListener("touchcancel", pinchEnd, true);

  /* ---------- баллуур ---------- */
  function segDist(px, py, ax, ay, bx, by) {
    const dx = bx - ax, dy = by - ay, l2 = dx * dx + dy * dy;
    const t = l2 ? clamp(((px - ax) * dx + (py - ay) * dy) / l2, 0, 1) : 0;
    return Math.hypot(px - ax - t * dx, py - ay - t * dy);
  }
  function hits(o, pt, r) {
    const b = o.getBoundingRect(true, true);
    if (pt.x < b.left - r || pt.x > b.left + b.width + r || pt.y < b.top - r || pt.y > b.top + b.height + r) return false;
    if (o.type !== "path" || o.k === "shape") return o.containsPoint(pt, null, true, true);
    const lp = F.util.transformPoint(pt, F.util.invertTransform(o.calcTransformMatrix()));
    const x = lp.x + o.pathOffset.x, y = lp.y + o.pathOffset.y, s = Math.max(Math.abs(o.scaleX), Math.abs(o.scaleY)) || 1, tol = (o.strokeWidth || 1) / 2 + r / s;
    let px = null, py = null;
    for (const c of o.path) {
      const n = c.length; if (n < 3) continue;
      const cx = c[n - 2], cy = c[n - 1];
      if ((px === null ? Math.hypot(x - cx, y - cy) : segDist(x, y, px, py, cx, cy)) <= tol) return true;
      px = cx; py = cy;
    }
    return false;
  }
  function eraseAt(pt) {
    const r = 9 / cv.getZoom(), list = cv.getObjects();
    for (let i = list.length - 1; i >= 0; i--) {
      const o = list[i];
      if (!editable(o) || !ERASABLE.has(o.k) || !hits(o, pt, r)) continue;
      drag?.ch?.push([o.id, ser(o), null]); removeId(o.id); send({ t: "del", id: o.id });
    }
    cv.requestRenderAll();
  }

  /* ---------- дүрс, шугам ---------- */
  const fillFor = (c, fm) => fm === 2 ? c : fm === 1 ? hexA(c, 0.18) : "transparent";
  const dashFor = (w) => style.dash ? [w * 2.5, w * 2] : null;
  function lineObj(a, b, kind) {
    const dx = b.x - a.x, dy = b.y - a.y, len = Math.max(1, Math.hypot(dx, dy)), w = style.width, h = Math.min(len * 0.45, 10 + w * 2.6);
    let d = `M 0 0 L ${len} 0`;
    if (kind !== "line") d += ` M ${len - h} ${-h * 0.6} L ${len} 0 L ${len - h} ${h * 0.6}`;
    if (kind === "darrow") d += ` M ${h} ${-h * 0.6} L 0 0 L ${h} ${h * 0.6}`;
    const p = new F.Path(d, { stroke: style.color, strokeWidth: w, fill: null, strokeLineCap: "round", strokeLineJoin: "round", strokeDashArray: dashFor(w), k: "line" });
    p.set({ left: (a.x + b.x) / 2, top: (a.y + b.y) / 2, angle: Math.atan2(dy, dx) * 180 / Math.PI });
    return p;
  }
  function styleShape(o) {
    o.set({ stroke: style.color, strokeWidth: style.width, fill: fillFor(style.color, style.fill), strokeUniform: true, strokeLineJoin: "round", strokeDashArray: dashFor(style.width), k: "shape", fm: style.fill });
    return o;
  }
  function shapeDrag(pt, shift) {
    const d = drag, a = d.start;
    if (!d.obj && Math.hypot(pt.x - a.x, pt.y - a.y) * cv.getZoom() < 4) return;
    if (LINES[shapeKind]) {
      let b = pt;
      if (shift) { const ang = Math.round(Math.atan2(pt.y - a.y, pt.x - a.x) / (Math.PI / 4)) * (Math.PI / 4), l = Math.hypot(pt.x - a.x, pt.y - a.y); b = { x: a.x + Math.cos(ang) * l, y: a.y + Math.sin(ang) * l }; }
      if (d.obj) cv.remove(d.obj);
      d.obj = lineObj(a, b, shapeKind); d.obj.set({ selectable: false, evented: false }); cv.add(d.obj);
    } else {
      if (!d.obj) { d.obj = styleShape(makeShape(shapeKind)); d.obj.set({ selectable: false, evented: false }); cv.add(d.obj); }
      let w = Math.abs(pt.x - a.x), h = Math.abs(pt.y - a.y);
      if (shift) w = h = Math.max(w, h);
      const sx = Math.sign(pt.x - a.x) || 1, sy = Math.sign(pt.y - a.y) || 1;
      d.obj.set({ left: a.x + (sx * w) / 2, top: a.y + (sy * h) / 2, scaleX: Math.max(w, 4) / d.obj.width, scaleY: Math.max(h, 4) / d.obj.height });
    }
    cv.requestRenderAll();
  }
  function shapeEnd(d, pt) {
    let o = d.obj;
    if (o) cv.remove(o);
    if (!o) { // товшилт: анхдагч хэмжээгээр
      if (LINES[shapeKind]) o = lineObj({ x: pt.x - 80, y: pt.y }, { x: pt.x + 80, y: pt.y }, shapeKind);
      else { o = styleShape(makeShape(shapeKind)); const regularish = ["ellipse", "star", "pentagon", "hexagon", "heart", "plus", "diamond"].includes(shapeKind); o.set({ left: pt.x, top: pt.y, scaleX: (regularish ? 140 : 180) / o.width, scaleY: 140 / o.height }); }
    }
    o.set({ selectable: true, evented: true });
    addLocal(o, { select: true });
  }
  function addText(pt) {
    const t = new F.IText("", { left: pt.x, top: pt.y - style.size * 0.6, originX: "left", originY: "top", fontFamily: style.font, fontSize: style.size, fontWeight: style.bold ? 700 : 500, fill: style.color === "#ffffff" ? "#121829" : style.color, lineHeight: 1.2, k: "text" });
    addLocal(t, { select: true });
    t.enterEditing(); t.hiddenTextarea?.focus();
  }
  function addNote(pt) {
    const n = new F.Sticky("", { left: pt.x, top: pt.y - 90, note: style.note });
    addLocal(n, { select: true });
    n.enterEditing(); n.hiddenTextarea?.focus();
  }
  // Шинэ зүйлсийг дэлгэцийн голд шатлан байрлуулна (дараалан нэмэхэд давхцахгүй).
  let dropN = 0, dropAt = 0;
  function dropPoint() {
    const c = viewCenter(); if (Date.now() - dropAt > 8000) dropN = 0;
    dropAt = Date.now(); const k = dropN++ % 8, s = 36 / cv.getZoom();
    return { x: c.x + k * s, y: c.y + k * s };
  }
  function addEmoji(e, at) {
    const c = at || dropPoint();
    const t = new F.Text(e, { left: c.x, top: c.y, fontSize: 96, fontFamily: EMOJI_FONT, k: "emoji" });
    addLocal(t, { select: true });
  }
  function addIcon(inner, at) {
    const col = style.color === "#ffffff" ? "#121829" : style.color;
    const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="24" height="24"><g fill="none" stroke="${col}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">${inner}</g></svg>`;
    F.loadSVGFromString(svg, (list, opts) => {
      if (!list?.length) return;
      const g = F.util.groupSVGElements(list, opts), c = at || dropPoint();
      g.set({ left: c.x, top: c.y, scaleX: 4, scaleY: 4, k: "icon" }); // 24px viewBox → 96px
      addLocal(g, { select: true });
    });
  }
  function colorize(o, c) {
    if (o.k === "marker") o.set("stroke", hexA(c === "#ffffff" ? "#eab308" : c, 0.38));
    else if (o.k === "note" || o.k === "emoji" || o.k === "image") return;
    else if (o.k === "text" || o.type === "i-text") o.set("fill", c);
    else if (o.k === "icon") { (o.type === "group" ? o.getObjects() : [o]).forEach((x) => x.set("stroke", c)); o.dirty = true; }
    else { o.set("stroke", c); if (o.k === "shape") o.set("fill", fillFor(c, o.fm || 0)); }
  }

  /* ---------- зураг ---------- */
  const toBlob = (c) => new Promise((res) => c.toBlob((b) => (b && b.type === "image/webp" ? res(b) : c.toBlob(res, "image/png")), "image/webp", 0.9));
  async function shrink(blob) {
    if (blob.type === "image/gif") return blob;
    let bmp; try { bmp = await createImageBitmap(blob); } catch { return blob; }
    const s = Math.min(1, 2000 / Math.max(bmp.width, bmp.height));
    if (s === 1 && blob.size < 4e6 && /png|jpe?g|webp/.test(blob.type)) { bmp.close?.(); return blob; }
    const c = document.createElement("canvas"); c.width = Math.round(bmp.width * s); c.height = Math.round(bmp.height * s);
    c.getContext("2d").drawImage(bmp, 0, 0, c.width, c.height); bmp.close?.();
    return toBlob(c);
  }
  async function uploadImage(blob) {
    const body = await shrink(blob);
    const r = await fetch(base + "/image", { method: "POST", headers: { Authorization: "Bearer " + Auth.token, "Content-Type": body.type || "application/octet-stream" }, body });
    const d = await r.json().catch(() => null);
    if (!r.ok) throw new Error(d?.error || "Зураг оруулж чадсангүй");
    return d;
  }
  function addImage(d, at, cr) {
    return new Promise((res) => F.Image.fromURL(d.url, (img, err) => {
      if (!img || err || !img.width) { toast("Зургийг ачаалж чадсангүй", true); return res(null); }
      const vp = viewRect(), s = Math.min(1, (vp.w * 0.5) / img.width, (vp.h * 0.55) / img.height), c = at || dropPoint();
      img.set({ left: c.x, top: c.y, scaleX: s, scaleY: s, k: "image" });
      if (cr) img.cr = String(cr).slice(0, 200);
      addLocal(img, { select: true }); res(img);
    }));
  }
  async function insertBlobs(blobs, at) {
    if (!canEdit) return toast("Энэ самбарт одоо зураг оруулах боломжгүй", true);
    for (const [i, b] of blobs.entries()) {
      const done = busyToast("Зураг оруулж байна…");
      try { await addImage(await uploadImage(b), at && { x: at.x + i * 30, y: at.y + i * 30 }); }
      catch (e) { toast(e.message, true); } finally { done(); }
    }
  }
  async function insertURL(u, at) {
    if (!canEdit) return toast("Энэ самбарт одоо зураг оруулах боломжгүй", true);
    u = String(u || "").trim();
    if (/^data:image\//i.test(u)) { try { return insertBlobs([await (await fetch(u)).blob()], at); } catch { return toast("Зургийг уншиж чадсангүй", true); } }
    if (!/^https?:\/\//i.test(u)) return toast("Зургийн холбоос https://-ээр эхлэх ёстой", true);
    const done = busyToast("Зураг татаж байна…");
    try { await addImage(await post("/fetch", { url: u }), at); } catch (e) { toast(e.message, true); } finally { done(); }
  }
  // Хайлтаас: том хувилбарыг сервер татна; болохгүй бол жижиг хувилбараар.
  async function insertHit(h, at) {
    if (!canEdit) return toast("Энэ самбарт одоо зураг оруулах боломжгүй", true);
    const done = busyToast("Зураг оруулж байна…");
    try {
      let d;
      try { d = await post("/fetch", { url: h.url }); } catch (e) { if (!h.thumb || h.thumb === h.url) throw e; d = await post("/fetch", { url: h.thumb }); }
      await addImage(d, at, h.credit);
    } catch (e) { toast(e.message, true); } finally { done(); }
  }
  const imgFromHTML = (html) => { const m = /<img[^>]+src\s*=\s*["']([^"']+)["']/i.exec(html || ""); return m ? m[1].replace(/&amp;/g, "&") : ""; };
  async function editImage(o) {
    const r = await imageEditor(o.getSrc());
    if (!r || dead) return;
    const done = busyToast("Хадгалж байна…");
    try {
      const d = await uploadImage(r.blob);
      const b = ser(o), c = o.getCenterPoint(), W0 = o.width, H0 = o.height;
      const dx = (r.x + r.w / 2 - W0 / 2) * o.scaleX * (o.flipX ? -1 : 1), dy = (r.y + r.h / 2 - H0 / 2) * o.scaleY * (o.flipY ? -1 : 1);
      const ang = F.util.degreesToRadians(o.angle || 0), cx = c.x + dx * Math.cos(ang) - dy * Math.sin(ang), cy = c.y + dx * Math.sin(ang) + dy * Math.cos(ang);
      const j = { ...b, src: d.url, width: d.w, height: d.h, cropX: 0, cropY: 0, left: cx, top: cy, originX: "center", originY: "center", scaleX: (r.w * o.scaleX) / d.w, scaleY: (r.h * o.scaleY) / d.h };
      cv.discardActiveObject();
      upsert(j); send({ t: "upd", id: o.id, o: j }); record([[o.id, b, j]]);
    } catch (e) { toast(e.message, true); } finally { done(); }
  }

  /* ---------- товчлуурууд ---------- */
  const typing = () => { const a = document.activeElement; return (a && (/^(INPUT|TEXTAREA|SELECT)$/.test(a.tagName) || a.isContentEditable) && !a.closest?.(".canvas-container")) || !!cv.getActiveObject()?.isEditing; };
  const KEYS = { v: "select", h: "hand", p: "pen", m: "marker", e: "eraser", s: "shape", t: "text", n: "note", k: "laser" };
  function onKey(e) {
    if (dead || document.querySelector(".bd-ie")) return;
    const k = e.key.toLowerCase(), mod = e.ctrlKey || e.metaKey;
    if (e.key === "Escape") {
      if (cv.getActiveObject()?.isEditing) return;
      if (chatOn && chatBox.contains(document.activeElement)) { if (innerWidth < 820) toggleChat(false); else document.activeElement.blur(); return; }
      if (closePops()) { e.preventDefault(); return; }
      if (!panel.hidden && me.teacher && innerWidth < 900) { togglePanel(false); return; }
      if (cv.getActiveObject()) { cv.discardActiveObject(); cv.requestRenderAll(); renderStyle(); return; }
      if (tool !== "select" && canEdit) setTool("select");
      return;
    }
    if (typing()) return;
    if (e.code === "Space") { e.preventDefault(); if (!e.repeat) { spaceDown = true; cv.isDrawingMode = false; cv.setCursor("grab"); } return; }
    if (mod && k === "z") { e.preventDefault(); e.shiftKey ? redo() : undo(); return; }
    if (mod && k === "y") { e.preventDefault(); redo(); return; }
    if (mod && k === "d") { e.preventDefault(); duplicate(); return; }
    if (mod && k === "a") { e.preventDefault(); if (tool !== "select") setTool("select"); const list = cv.getObjects().filter(editable); if (list.length) { cv.setActiveObject(list.length > 1 ? new F.ActiveSelection(list, { canvas: cv }) : list[0]); cv.requestRenderAll(); } return; }
    if (mod) return;
    if (e.key === "Delete" || e.key === "Backspace") { e.preventDefault(); delSel(); return; }
    if (k === "+" || k === "=") { zoomTo(cv.getZoom() * 1.2); return; }
    if (k === "-") { zoomTo(cv.getZoom() / 1.2); return; }
    if (k === "0") { fitView(); return; }
    if (k === "i") { openPop("image"); return; }
    if (KEYS[k] && (canEdit || k === "h")) { if (KEYS[k] === "shape") { setTool("shape"); openPop("shapes"); } else setTool(KEYS[k]); }
  }
  const onKeyUp = (e) => { if (e.code === "Space" && spaceDown) { spaceDown = false; setTool(tool); } };
  function onCopy(e) {
    if (typing() || dead) return;
    const list = cv.getActiveObjects(); if (!list.length) return;
    clip = list.map(ser); e.clipboardData?.setData("text/plain", "⧉surgalt-board"); e.preventDefault();
  }
  async function onPaste(e) {
    if (typing() || dead || document.querySelector(".bd-ie")) return;
    const dt = e.clipboardData; if (!dt) return;
    const files = [...dt.items].filter((i) => i.kind === "file" && i.type.startsWith("image/")).map((i) => i.getAsFile()).filter(Boolean);
    if (files.length) { e.preventDefault(); return insertBlobs(files); }
    const text = dt.getData("text/plain").trim();
    if (text === "⧉surgalt-board" && clip) { e.preventDefault(); return pasteJSON(clip, 28); }
    const src = imgFromHTML(dt.getData("text/html"));
    if (src) { e.preventDefault(); return insertURL(src); }
    if (/^https?:\/\/\S+$/i.test(text) || /^data:image\//i.test(text)) { e.preventDefault(); return insertURL(text); }
    if (text && canEdit) { // энгийн текст: самбарт бичвэр болгоно
      e.preventDefault(); const c = dropPoint();
      const t = new F.IText(text.slice(0, 2000), { left: c.x, top: c.y, originX: "left", originY: "top", fontFamily: style.font, fontSize: style.size, fill: style.color === "#ffffff" ? "#121829" : style.color, k: "text" });
      addLocal(t, { select: true });
    }
  }
  // Чирж оруулах: файл, өөр табаас зураг, дүрс тэмдэгт/эможи.
  wrap.addEventListener("dragover", (e) => { e.preventDefault(); e.dataTransfer.dropEffect = "copy"; wrap.classList.add("bd-drop"); });
  wrap.addEventListener("dragleave", () => wrap.classList.remove("bd-drop"));
  wrap.addEventListener("drop", (e) => {
    e.preventDefault(); wrap.classList.remove("bd-drop");
    const dt = e.dataTransfer, pt = cv.restorePointerVpt(new F.Point(e.offsetX, e.offsetY));
    const em = dt.getData("text/x-sg-emoji"), ic = dt.getData("text/x-sg-icon"), hit = dt.getData("text/x-sg-img");
    if (hit) { try { return insertHit(JSON.parse(hit), pt); } catch { return; } }
    if (em) return canEdit && addEmoji(em, pt);
    if (ic) return canEdit && addIcon(ic, pt);
    const files = [...dt.files].filter((f) => f.type.startsWith("image/"));
    if (files.length) return insertBlobs(files, pt);
    const src = imgFromHTML(dt.getData("text/html")) || dt.getData("text/uri-list").split("\n").find((l) => l && !l.startsWith("#")) || "";
    if (src) insertURL(src.trim(), pt);
  });
  fileIn.onchange = () => { const fl = [...fileIn.files]; fileIn.value = ""; if (fl.length) insertBlobs(fl); };

  /* ---------- попап цонхнууд ---------- */
  function closePops() { const open = qa(".bd-pop:not([hidden])"); open.forEach((p) => (p.hidden = true)); qa("[data-pop].pop-on").forEach((b) => b.classList.remove("pop-on")); return open.length > 0; }
  function openPop(name, btn) {
    const p = q(`.bd-pop[data-p="${name}"]`); if (!p) return;
    const wasOpen = !p.hidden; closePops(); if (wasOpen) return;
    btn ||= q(`[data-pop="${name}"]`);
    ({ shapes: renderShapes, image: renderImagePop, icons: renderIcons, emoji: renderEmoji, more: renderMore })[name](p);
    p.hidden = false; btn?.classList.add("pop-on");
    if (btn && innerWidth > 720) {
      const r = btn.getBoundingClientRect(), s = stage.getBoundingClientRect();
      if (name === "more") { p.style.top = "8px"; p.style.right = "12px"; p.style.left = "auto"; }
      else { // том цонх (зураг хайлт, дүрс, эможи) дээрээсээ — үр дүн ачаалагдаж өндөр нь өсөхөд тайрагдахгүй
        const big = name === "image" || name === "icons" || name === "emoji";
        p.style.left = r.right - s.left + 10 + "px"; p.style.top = (big ? 8 : clamp(r.top - s.top - 8, 8, s.height - p.offsetHeight - 8)) + "px"; p.style.right = "auto";
      }
    }
  }
  function renderShapes(p) {
    p.innerHTML = `<div class="bd-pop-h"><b>Дүрс</b><small>Чирж хэмжээг нь тохируулна · Shift — тэнцүү харьцаатай</small></div>
      <div class="bd-shgrid">${Object.keys(SH).map((k) => `<button type="button" data-sh="${k}" class="${shapeKind === k ? "on" : ""}" title="${esc(SH[k][0])}">${shSVG(k)}</button>`).join("")}</div>
      <div class="bd-pop-h"><b>Шугам, сум</b></div>
      <div class="bd-shgrid">${Object.entries(LINES).map(([k, [t, svg]]) => `<button type="button" data-sh="${k}" class="${shapeKind === k ? "on" : ""}" title="${esc(t)}"><svg viewBox="0 0 24 24" width="26" height="26" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">${svg}</svg></button>`).join("")}</div>`;
  }
  /* Зураг хайлт: самбар дотроо хайж, дармагц (эсвэл чирээд) оруулна. Хайлтын төлөв попап хаагдсан ч хадгалагдана. */
  const isr = { q: "", kind: "", page: 1, items: [], more: false, busy: false, en: "" };
  function renderImagePop(p) {
    p.innerHTML = `<div class="bd-pop-h"><b>Зураг хайх, оруулах</b><button type="button" class="bd-mini" data-img="file">${bi("upload", 15)}Компьютерээс</button></div>
      <form class="bd-f bd-isf" data-img="search"><div class="bd-row"><input name="q" type="search" placeholder="Юу хайх вэ? — жишээ нь: наран цэцэг" autocomplete="off" value="${esc(isr.q)}" aria-label="Зураг хайх"><button class="btn btn-gold btn-sm" title="Хайх">${bi("search", 16)}</button></div></form>
      <div class="bd-tabs bd-ikind">${IMG_KINDS.map(([k, n]) => `<button type="button" data-ik="${k}" class="${isr.kind === k ? "on" : ""}">${n}</button>`).join("")}</div>
      <div class="bd-ires"></div>
      <details class="bd-imore"><summary>Бусад арга: Google, холбоос, хуулж буулгах</summary>
        <form class="bd-f" data-img="google"><div class="bd-row"><input name="q" placeholder="Google зургаас хайх" required autocomplete="off" value="${esc(isr.q)}"><button class="btn btn-glass btn-sm">Google ↗</button></div>
          <small class="muted">Google шинэ цонхонд нээгдэнэ. Зураг дээр <b>баруун товч → «Зургийг хуулах»</b> дараад энд <kbd>Ctrl</kbd>+<kbd>V</kbd> дарна уу (эсвэл зургийг чирж оруулна уу).</small></form>
        <form class="bd-f" data-img="url"><div class="bd-row"><input name="u" placeholder="Зургийн холбоос https://…" required autocomplete="off" inputmode="url"><button class="btn btn-glass btn-sm">Оруулах</button></div></form>
      </details>
      <p class="bd-tip">${bi("crop", 15)}<span>Оруулсан зураг дээр <b>давхар дарж</b> тайрах, <b>дэвсгэрийг арилгах</b> боломжтой.</span></p>`;
    paintHits(p);
    if (innerWidth > 720) setTimeout(() => p.querySelector(".bd-isf input")?.focus(), 30);
  }
  function paintHits(p) {
    const box = p.querySelector(".bd-ires"); if (!box) return;
    if (isr.busy && !isr.items.length) { box.innerHTML = `<p class="bd-pop-wait"><span class="bd-spin"></span>Хайж байна…</p>`; return; }
    if (!isr.q) { box.innerHTML = `<p class="bd-ihint">Санаа:</p><div class="bd-ideas">${IMG_IDEAS.map((t) => `<button type="button" data-idea="${esc(t)}">${esc(t)}</button>`).join("")}</div>`; return; }
    if (!isr.items.length) { box.innerHTML = `<p class="bd-ihint">«${esc(isr.q)}» гэсэн зураг олдсонгүй — өөр үгээр эсвэл англиар хайж үзээрэй.</p>`; return; }
    box.innerHTML = `<div class="bd-igrid2">${isr.items.map((h, i) => `<button type="button" class="bd-ihit" data-hit="${i}" draggable="true" title="${esc((h.title || "") + (h.credit ? " — " + h.credit : ""))}"><img src="${esc(h.thumb)}" alt="${esc(h.title || "")}" loading="lazy" referrerpolicy="no-referrer" style="aspect-ratio:${h.w && h.h ? h.w + "/" + h.h : "4/3"}"></button>`).join("")}</div>
      ${isr.more ? `<button type="button" class="btn btn-glass btn-sm bd-imorebtn" data-img="more" ${isr.busy ? "disabled" : ""}>${isr.busy ? "Ачаалж байна…" : "Цааш үзэх"}</button>` : ""}
      <p class="bd-credit muted">${isr.en && isr.en !== isr.q ? `«${esc(isr.en)}» гэж хайлаа · ` : ""}Нээлттэй лицензтэй зургууд (Wikimedia Commons, Pixabay). Дармагц самбарт орно, чирж оруулж ч болно.</p>`;
    box.querySelectorAll("img").forEach((im) => { im.onerror = () => im.closest(".bd-ihit")?.remove(); });
  }
  async function searchImages(p, more) {
    if (isr.busy || !isr.q) return;
    isr.busy = true; if (!more) { isr.items = []; isr.page = 1; } paintHits(p);
    try {
      const d = await api(`/api/board/images?q=${encodeURIComponent(isr.q)}&kind=${isr.kind}&page=${more ? isr.page + 1 : 1}`);
      if (more) isr.page++;
      const seen = new Set(isr.items.map((h) => h.url));
      isr.items = isr.items.concat((d.items || []).filter((h) => !seen.has(h.url)));
      isr.more = !!d.more; isr.en = d.en || "";
    } catch (e) { toast(e.message, true); }
    finally { isr.busy = false; paintHits(p); }
  }
  let iconData = null;
  async function renderIcons(p) {
    if (!iconData) {
      p.innerHTML = `<p class="bd-pop-wait"><span class="bd-spin"></span>Ачаалж байна…</p>`;
      try { iconData = await (await fetch("/static/board-icons.json?v=" + V)).json(); } catch { p.innerHTML = `<p class="bd-pop-wait">Ачаалж чадсангүй</p>`; return; }
    }
    const svg = (inner) => `<svg viewBox="0 0 24 24" width="26" height="26" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${inner}</svg>`;
    p.innerHTML = `<div class="bd-pop-h"><input type="search" class="bd-search" placeholder="Хайх — ном, нар, сум, амьтан…" aria-label="Дүрс хайх"></div>
      <div class="bd-tabs">${iconData.cats.map((c, i) => `<button type="button" data-cat="${i}" class="${i ? "" : "on"}">${esc(c.n)}</button>`).join("")}</div>
      <div class="bd-igrid"></div><p class="bd-credit muted">Lucide (ISC)</p>`;
    const grid = p.querySelector(".bd-igrid");
    const show = (items) => { grid.innerHTML = items.length ? items.map(([n, t, inner]) => `<button type="button" draggable="true" data-ic="${esc(n)}" title="${esc(t)}">${svg(inner)}<small>${esc(t)}</small></button>`).join("") : `<p class="muted small">Олдсонгүй</p>`; };
    const all = iconData.cats.flatMap((c) => c.i);
    p._inner = new Map(all.map(([n, , inner]) => [n, inner]));
    show(iconData.cats[0].i);
    p.querySelector(".bd-search").oninput = (e) => {
      const s = e.target.value.trim().toLowerCase();
      p.querySelectorAll(".bd-tabs button").forEach((b) => b.classList.toggle("on", !s && b.dataset.cat === "0"));
      show(s ? all.filter(([n, t]) => t.toLowerCase().includes(s) || n.includes(s)) : iconData.cats[0].i);
    };
  }
  function renderEmoji(p) {
    p.innerHTML = `<div class="bd-tabs">${EMOJI.map(([n], i) => `<button type="button" data-ecat="${i}" class="${i ? "" : "on"}">${esc(n)}</button>`).join("")}</div><div class="bd-egrid"></div>`;
    showEmoji(p, 0);
  }
  function showEmoji(p, i) { p.querySelector(".bd-egrid").innerHTML = EMOJI[i][1].split(" ").map((e) => `<button type="button" draggable="true" data-em="${e}">${e}</button>`).join(""); }
  function renderMore(p) {
    const t = me.teacher;
    p.innerHTML = `<div class="bd-pop-h"><b>Дэвсгэр</b></div>
      <div class="bd-bgs">${BGS.map(([k, n]) => `<button type="button" data-bg="${k}" class="bg-${k} ${style.bg === k ? "on" : ""}"><i></i><small>${n}</small></button>`).join("")}</div>
      <button type="button" class="bd-mi" data-a="export">${bi("download", 18)}Зураг болгон татах (PNG)</button>
      ${canEdit ? `<button type="button" class="bd-mi bd-danger" data-a="clear">${bi("trash", 18)}Энэ самбарыг цэвэрлэх</button>` : ""}
      ${t ? `<hr><button type="button" class="bd-mi" data-a="distribute">${bi("distribute", 18)}Миний самбарыг бүх самбарт тараах</button>
        <label class="bd-mi bd-chk"><input type="checkbox" data-a="copy" ${st.copy ? "checked" : ""}>Шинэ самбар миний самбараас эхэлнэ</label>` : ""}
      <hr><p class="bd-keys"><b>Товчлол:</b> V сонгох · P үзэг · M тодруулагч · E баллуур · S дүрс · T текст · N тэмдэглэл · Space гүйлгэх · Ctrl+Z буцаах · Ctrl+D хувилах · Del устгах</p>`;
  }

  /* ---------- хэв маягийн самбар (доод) ---------- */
  function renderStyle() {
    if (!canEdit) { styleBar.hidden = true; return; }
    const sel = cv.getActiveObjects().filter(editable), ks = new Set(sel.map((o) => o.k));
    const m = sel.length ? "sel" : tool, has = (...a) => a.some((k) => ks.has(k));
    const showColor = m === "pen" || m === "marker" || m === "shape" || m === "text" || has("pen", "marker", "shape", "line", "text", "icon");
    const showNote = m === "note" || has("note");
    const showWidth = m === "pen" || m === "marker" || m === "shape" || has("pen", "marker", "shape", "line");
    const showFill = (m === "shape" && !LINES[shapeKind]) || has("shape");
    const showDash = m === "shape" || has("shape", "line");
    const showFont = m === "text" || has("text", "note");
    let h = "";
    if (showColor) h += `<div class="bd-sg bd-colors">${COLORS.map((c) => `<button type="button" data-color="${c}" class="${style.color === c ? "on" : ""}" style="--c:${c}" title="${c}"></button>`).join("")}</div>`;
    if (showNote) h += `<div class="bd-sg bd-colors bd-notes">${NOTES.map((c) => `<button type="button" data-note="${c}" class="${style.note === c ? "on" : ""}" style="--c:${c}"></button>`).join("")}</div>`;
    if (showWidth) h += `<div class="bd-sg">${WIDTHS.map((w) => `<button type="button" data-w="${w}" class="bd-w ${style.width === w ? "on" : ""}" title="${w}px"><i style="--w:${Math.min(w, 12) + 2}px"></i></button>`).join("")}</div>`;
    if (showFill) h += `<div class="bd-sg"><button type="button" data-fill class="bd-fill f${style.fill}" title="Дүүргэлт: ${["хоосон", "цайвар", "бүтэн"][style.fill]}">${bi("fill", 18)}</button>${showDash ? `<button type="button" data-dash class="${style.dash ? "on" : ""}" title="Тасархай зураас"><svg viewBox="0 0 24 24" width="20" height="20" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"><path d="M3 12h3M10 12h4M18 12h3"/></svg></button>` : ""}</div>`;
    else if (showDash) h += `<div class="bd-sg"><button type="button" data-dash class="${style.dash ? "on" : ""}" title="Тасархай зураас"><svg viewBox="0 0 24 24" width="20" height="20" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"><path d="M3 12h3M10 12h4M18 12h3"/></svg></button></div>`;
    if (showFont) h += `<div class="bd-sg"><select data-font aria-label="Фонт">${FONTS.map(([f, n]) => `<option value="${f}" ${style.font === f ? "selected" : ""}>${n}</option>`).join("")}</select>
      ${SIZES.map((s, i) => `<button type="button" data-size="${s}" class="bd-size ${style.size === s ? "on" : ""}" title="${s}px" style="font-size:${11 + i * 3}px">A</button>`).join("")}
      <button type="button" data-bold class="${style.bold ? "on" : ""}" title="Тод">${bi("bold", 17)}</button></div>`;
    if (sel.length) {
      const one = sel.length === 1 ? sel[0] : null, locked = sel.every((o) => o.lk);
      h += `<div class="bd-sg bd-acts">${one?.type === "image" ? `<button type="button" data-act="edit" class="bd-act-wide">${bi("crop", 17)}<span>Тайрах · дэвсгэр</span></button>${one.cr ? `<button type="button" data-act="credit" title="${esc(one.cr)}">©</button>` : ""}` : ""}
        <button type="button" data-act="dup" title="Хувилах (Ctrl+D)">${bi("copy", 18)}</button>
        <button type="button" data-act="front" title="Хамгийн урд гаргах">${bi("front", 18)}</button>
        <button type="button" data-act="back" title="Хамгийн ард оруулах">${bi("back", 18)}</button>
        ${me.teacher ? `<button type="button" data-act="lock" class="${locked ? "on" : ""}" title="${locked ? "Түгжээг тайлах" : "Түгжих — суралцагч хөдөлгөж, устгаж чадахгүй"}">${bi(locked ? "lock" : "unlock", 18)}</button>` : ""}
        <button type="button" data-act="del" class="bd-del" title="Устгах (Del)">${bi("trash", 18)}</button></div>`;
    } else if (m === "eraser") h += `<p class="bd-hint">Арилгах зураас, дүрсийн дээгүүр чирнэ үү</p>`;
    else if (m === "laser") h += `<p class="bd-hint">Хулганаа хөдөлгөхөд суралцагчид улаан цэгийг харна</p>`;
    styleBar.innerHTML = h; styleBar.hidden = !h;
  }
  styleBar.addEventListener("click", (e) => {
    const b = e.target.closest("button"); if (!b) return;
    const sel = cv.getActiveObjects().filter(editable);
    if (b.dataset.color) { style.color = b.dataset.color; if (sel.length) restyle(sel, (o) => colorize(o, style.color)); }
    else if (b.dataset.note) { style.note = b.dataset.note; restyle(sel.filter((o) => o.k === "note"), (o) => o.set("note", style.note)); }
    else if (b.dataset.w) { style.width = +b.dataset.w; restyle(sel.filter((o) => ["pen", "shape", "line"].includes(o.k) || o.k === "marker"), (o) => { o.set("strokeWidth", o.k === "marker" ? Math.max(14, style.width * 4) : style.width); if (o.strokeDashArray) o.set("strokeDashArray", [style.width * 2.5, style.width * 2]); }); }
    else if ("fill" in b.dataset) { style.fill = (style.fill + 1) % 3; restyle(sel.filter((o) => o.k === "shape"), (o) => { o.fm = style.fill; o.set("fill", fillFor(o.stroke, style.fill)); }); }
    else if ("dash" in b.dataset) { style.dash = !style.dash; restyle(sel.filter((o) => o.k === "shape" || o.k === "line"), (o) => o.set("strokeDashArray", style.dash ? [o.strokeWidth * 2.5, o.strokeWidth * 2] : null)); }
    else if (b.dataset.size) { style.size = +b.dataset.size; restyle(sel.filter((o) => o.k === "text" || o.k === "note"), (o) => o.set("fontSize", o.k === "note" ? Math.round(style.size * 0.85) : style.size)); }
    else if ("bold" in b.dataset) { style.bold = !style.bold; restyle(sel.filter((o) => o.k === "text" || o.k === "note"), (o) => o.set("fontWeight", style.bold ? 700 : 500)); }
    else if (b.dataset.act) {
      const a = b.dataset.act;
      if (a === "del") delSel(); else if (a === "dup") duplicate(); else if (a === "front") zorder(true); else if (a === "back") zorder(false);
      else if (a === "edit" && sel[0]?.type === "image") editImage(sel[0]);
      else if (a === "credit") { toast("Зургийн эх сурвалж: " + (sel[0]?.cr || "")); return; }
      else if (a === "lock") { const lk = !sel.every((o) => o.lk); restyle(sel, (o) => { o.lk = lk; }); sel.forEach(prepare); }
      renderStyle(); return;
    }
    saveStyle(); if (cv.isDrawingMode) setTool(tool); else renderStyle();
  });
  styleBar.addEventListener("change", (e) => {
    if (!e.target.matches("[data-font]")) return;
    style.font = e.target.value; saveStyle();
    restyle(cv.getActiveObjects().filter((o) => editable(o) && (o.k === "text" || o.k === "note")), (o) => o.set("fontFamily", style.font));
  });

  /* ---------- дээд мөр, мэдэгдэл, анги ---------- */
  function renderChrome() { renderTop(); renderPeers(); renderBanner(); renderStyle(); renderPanel(); syncUndo(); renderChat(); renderVoice(); }
  function renderTop() {
    q(".bd-room").textContent = roomLabel(room);
    q(".bd-tools").hidden = !canEdit && !me.teacher;
    q(".bd-tools").classList.toggle("bd-ro", !canEdit);
    root.classList.toggle("is-teacher", me.teacher);
    const modes = q(".bd-modes"); modes.hidden = !me.teacher;
    qa(".bd-modes [data-mode]").forEach((b) => { const on = b.dataset.mode === st.mode; b.classList.toggle("on", on); b.setAttribute("aria-checked", on); });
    q(".bd-lockbtn").hidden = !me.teacher || st.mode === "teach";
    q(".bd-lockbtn").classList.toggle("on", !!st.lock);
    q(".bd-lockbtn").innerHTML = `${bi(st.lock ? "play" : "pause", 17)}<span>${st.lock ? "Үргэлжлүүлэх" : "Түр зогсоох"}</span>`;
    q(".bd-panelbtn").hidden = !me.teacher;
    // Дуут өрөө: микрофон (ярих эрхтэй үед), гар өргөх (суралцагч), багш хэнд ярихаа сонгоно.
    const tgt = micTarget(), mb = q(".bd-micbtn"), up = (st.hands || []).includes(me.uid), hb = q(".bd-handbtn"), vs = q(".bd-vto");
    mb.hidden = !mic && !tgt; mb.classList.toggle("on", !!mic);
    mb.innerHTML = `${bi(mic ? "mic" : "micoff", 17)}<span>${mic ? (tgt === "all" ? "Ярьж байна" : "Багтаа ярьж байна") : tgt === "all" ? "Ярих" : "Багтаа ярих"}</span><i class="bd-lv" aria-hidden="true"></i>`;
    mb.title = mic ? "Микрофон унтраах" : tgt === "all" ? "Микрофон асааж бүх ангид ярих" : "Микрофон асааж багийнхантайгаа ярих";
    mb.setAttribute("aria-pressed", String(!!mic));
    hb.hidden = me.teacher || onStage(); hb.classList.toggle("on", up);
    hb.innerHTML = `<span class="bd-hand" aria-hidden="true">✋</span><span>${up ? "Гараа буулгах" : "Гар өргөх"}</span>`;
    hb.title = up ? "Гараа буулгах" : "Гар өргөж бүх ангид ярих эрх хүсэх";
    vs.hidden = !me.teacher || st.mode !== "team";
    if (!vs.hidden) { vs.innerHTML = `<option value="all">📢 Бүх анги</option>` + Array.from({ length: st.teams }, (_, i) => `<option value="t:${i}">${teamOf(i)[0]} ${esc(teamOf(i)[1])}</option>`).join(""); vs.value = tgt || "all"; }
    const rooms = q(".bd-rooms"), own = ownRoom();
    if (!me.teacher && own && !st.spot) {
      rooms.hidden = false;
      rooms.innerHTML = [own, "main"].map((r) => `<button type="button" data-room="${esc(r)}" class="${r === room ? "on" : ""}">${esc(roomLabel(r))}</button>`).join("");
    } else if (me.teacher && room !== "main") {
      rooms.hidden = false; rooms.innerHTML = `<button type="button" data-room="main" class="bd-back">${bi("board", 16)}Миний самбар руу</button>`;
    } else rooms.hidden = true;
    root.classList.toggle("has-rooms", !rooms.hidden);
    q(".bd-lockov").hidden = !(st.lock && !me.teacher && room !== "main");
    wrap.className = "bd-wrap bg-" + (style.bg || "dots");
    afterView();
  }
  function renderPeers() {
    const here = peers.filter((p) => p.uid !== me.uid && p.room === room);
    q(".bd-peers").innerHTML = here.slice(0, 5).map(avatar).join("") + (here.length > 5 ? `<span class="bd-av bd-more">+${here.length - 5}</span>` : "");
    q(".bd-peers").title = here.map((p) => p.name).join(", ");
    const b = q(".bd-panelbtn .bd-badge"), n = peers.filter((p) => !p.teacher).length;
    if (b) { b.hidden = !n; b.textContent = n; }
  }
  function renderBanner() {
    const msgs = [];
    if (me.teacher) {
      if (st.spot) msgs.push(`${bi("spot", 17)}<span>Бүх ангид <b>${esc(roomLabel(st.spot))}</b> самбарыг харуулж байна</span><button type="button" data-a="unspot">Зогсоох</button>`);
      if (st.lock && st.mode !== "teach") msgs.push(`${bi("pause", 17)}<span>Суралцагчдын самбар түр зогссон</span><button type="button" data-a="lock">Үргэлжлүүлэх</button>`);
      if (room !== "main") msgs.push(`${bi("eye", 17)}<span><b>${esc(roomLabel(room))}</b> — шууд харж, засаж, ⭐ тавьж болно</span><button type="button" data-a="spot-here">Ангид харуулах</button>`);
    } else if (st.spot && room === st.spot) msgs.push(`${bi("spot", 17)}<span>Багш <b>${esc(roomLabel(st.spot))}</b> самбарыг ангид харуулж байна</span>`);
    else if (!canEdit && room === "main") msgs.push(`${bi("eye", 17)}<span>Багшийн самбарыг харж байна</span>${!follow && lastView ? `<button type="button" data-a="follow">${bi("target", 15)}Багшийг дагах</button>` : ""}`);
    else if (!canEdit && st.lock) msgs.push(`${bi("pause", 17)}<span>Багш самбарыг түр зогсоосон</span>`);
    banner.innerHTML = msgs.map((m) => `<div>${m}</div>`).join("");
    banner.hidden = !msgs.length;
  }
  function togglePanel(on) {
    panel.hidden = on === undefined ? !panel.hidden : !on;
    if (!panel.hidden && chatOn) { chatOn = false; chatBox.hidden = true; root.classList.remove("chat-on"); renderChat(); }
    q(".bd-panelbtn").classList.toggle("on", !panel.hidden);
    root.classList.toggle("panel-on", !panel.hidden);
    renderPanel();
  }

  /* ---------- багшийн «Анги» самбар: шууд жижиг зургууд, багууд ---------- */
  const thumbs = new Map();
  let sc = null, thumbBusy = false;
  function markThumb(r) { if (!me.teacher || !r) return; const t = thumbs.get(r) || {}; t.need = true; thumbs.set(r, t); kickThumbs(); }
  async function renderThumb(list) {
    sc ||= new F.StaticCanvas(null, { width: 320, height: 200, renderOnAddRemove: false, enableRetinaScaling: false });
    sc.clear(); sc.backgroundColor = "#ffffff";
    if (!list.length) return "";
    const made = (await Promise.all(list.map(enliven))).filter(Boolean);
    made.forEach((o) => sc.add(o));
    let x1 = Infinity, y1 = Infinity, x2 = -Infinity, y2 = -Infinity;
    for (const o of made) { o.setCoords(); const b = o.getBoundingRect(true, true); x1 = Math.min(x1, b.left); y1 = Math.min(y1, b.top); x2 = Math.max(x2, b.left + b.width); y2 = Math.max(y2, b.top + b.height); }
    const z = Math.min(296 / Math.max(x2 - x1, 1), 176 / Math.max(y2 - y1, 1), 1.2);
    sc.setViewportTransform([z, 0, 0, z, 160 - ((x1 + x2) / 2) * z, 100 - ((y1 + y2) / 2) * z]);
    sc.renderAll();
    return sc.toDataURL({ format: "jpeg", quality: 0.78 });
  }
  async function kickThumbs() {
    if (thumbBusy || panel.hidden || dead) return;
    thumbBusy = true;
    try {
      for (;;) {
        const vis = [...panel.querySelectorAll(".bd-card[data-room]")].map((c) => c.dataset.room);
        const r = vis.find((x) => { const t = thumbs.get(x); return !t || !("url" in t) || (t.need && Date.now() - (t.at || 0) > 1500); });
        if (!r || panel.hidden || dead) break;
        const t = thumbs.get(r) || {}; t.need = false; t.at = Date.now(); thumbs.set(r, t);
        try { const d = await api(`${base}?peek=1&room=${encodeURIComponent(r)}`); t.url = await renderThumb(d.objects || []); t.n = (d.objects || []).length; }
        catch { t.url = t.url || ""; }
        const card = panel.querySelector(`.bd-card[data-room="${CSS.escape(r)}"]`);
        if (card) paintThumb(card, t);
        await sleep(120);
      }
    } finally { thumbBusy = false; }
    if (!panel.hidden && [...thumbs.values()].some((t) => t.need)) setTimeout(kickThumbs, 900);
  }
  function paintThumb(card, t) {
    const box = card.querySelector(".bd-thumb");
    box.innerHTML = t.url ? `<img src="${t.url}" alt="">` : `<span>${"url" in t ? "Хоосон" : "…"}</span>`;
  }
  function cardHTML(r, name, extra = "", online = false) {
    return `<article class="bd-card ${room === r ? "on" : ""} ${st.spot === r ? "spot" : ""}" data-room="${esc(r)}">
      <button type="button" class="bd-thumb" data-open="${esc(r)}" title="Нээх"><span>…</span></button>
      <div class="bd-card-h"><i class="bd-dot ${online ? "on" : ""}"></i><b>${name}</b>
        <button type="button" class="bd-ib bd-sm ${st.spot === r ? "on" : ""}" data-spot="${esc(r)}" title="${st.spot === r ? "Харуулахаа болих" : "Бүх ангид харуулах"}">${bi("spot", 16)}</button></div>${extra}</article>`;
  }
  function renderPanel() {
    if (!me.teacher || panel.hidden) return;
    const online = new Map(peers.filter((p) => !p.teacher).map((p) => [p.uid, p]));
    const names = st.names || {}, ids = Object.keys(names).sort((a, b) => names[a].localeCompare(names[b]));
    let h = `<header class="bd-ph"><div><b>Анги</b><small>${online.size} онлайн · ${ids.length} суралцагч самбар нээсэн</small></div><button type="button" class="bd-ib" data-a="panel" title="Хаах">${bi("x", 18)}</button></header><div class="bd-pb">`;
    if (st.mode === "teach") {
      h += `<p class="bd-pnote">${bi("board", 18)}<span>Суралцагчид <b>таны самбарыг</b> шууд харж байна. <b>Хувиараа</b> эсвэл <b>Багаар</b> горимд шилжүүлбэл хүн бүр (баг бүр) өөрийн самбартай болж, тэдгээр нь энд шууд харагдана.</span></p>`;
      h += `<div class="bd-people">${online.size ? [...online.values()].map((p) => `<span class="bd-person">${avatar(p)}${esc(p.name)}</span>`).join("") : `<p class="muted small">Одоогоор суралцагч нэгдээгүй байна. Суралцагчид хичээлээ нээж «Самбар нээх» дармагц энд харагдана.</p>`}</div>`;
    } else if (st.mode === "solo") {
      h += ids.length ? `<div class="bd-cards">${ids.map((u) => cardHTML("u:" + u, esc(names[u]), "", online.has(u))).join("")}</div>` : `<p class="muted small bd-pnote">Суралцагчид самбар нээмэгц тус бүрийн самбар энд шууд харагдана.</p>`;
    } else {
      const teams = Array.from({ length: st.teams }, () => []), free = [];
      for (const u of ids) { const n = st.assign?.[u]; (n != null && n < st.teams ? teams[n] : free).push(u); }
      h += `<div class="bd-teamctl"><span>Баг</span><button type="button" class="bd-ib bd-sm" data-teams="-1" ${st.teams <= 2 ? "disabled" : ""}>${bi("minus", 16)}</button><b>${st.teams}</b><button type="button" class="bd-ib bd-sm" data-teams="1" ${st.teams >= 12 ? "disabled" : ""}>${bi("plus", 16)}</button>
        <button type="button" class="btn btn-glass btn-sm" data-a="shuffle">${bi("shuffle", 16)}Санамсаргүй хуваах</button></div>
        <p class="muted small">Суралцагчийн нэрийг чирж өөр багт шилжүүлнэ.</p>`;
      const chip = (u) => `<span class="bd-chip" draggable="true" data-uid="${esc(u)}"><i class="bd-dot ${online.has(u) ? "on" : ""}"></i>${esc(names[u])}${online.get(u)?.mic ? `<i class="bd-mic-i" title="Микрофон асаалттай">${bi("mic", 11)}</i>` : ""}</span>`;
      const chatBtn = (i) => { const n = unreadOf("b:" + i) + unreadOf("t:" + i); return `<button type="button" class="bd-mini bd-chatmini" data-chteam="${i}" title="Багийн чат, самбар">${bi("chat", 13)}Чат<b ${n ? "" : "hidden"}>${n || ""}</b></button>`; };
      h += `<div class="bd-cards">${teams.map((m, i) => { const t = teamOf(i); return cardHTML("t:" + i, `<span style="color:${t[2]}">${t[0]} ${esc(t[1])}</span>`, `<div class="bd-members" data-team="${i}">${m.map(chip).join("") || `<small class="muted">Суралцагчийг чирж оруулна</small>`}</div><div class="bd-card-f">${chatBtn(i)}</div>`, m.some((u) => online.has(u))); }).join("")}</div>`;
      if (free.length) h += `<div class="bd-free"><b>Багт ороогүй</b><div class="bd-members" data-team="-1">${free.map(chip).join("")}</div></div>`;
    }
    h += `</div><footer class="bd-pf">${st.mode !== "teach" ? `<button type="button" class="btn btn-glass btn-sm" data-a="distribute">${bi("distribute", 16)}Миний самбарыг бүгдэд тараах</button>` : ""}
      <label class="bd-chk"><input type="checkbox" data-a="copy" ${st.copy ? "checked" : ""}>Шинэ самбар миний самбараас (ажлын хуудас) эхэлнэ</label></footer>`;
    panel.innerHTML = h;
    panel.querySelectorAll(".bd-card[data-room]").forEach((c) => { const t = thumbs.get(c.dataset.room); if (t && "url" in t) paintThumb(c, t); });
    kickThumbs();
  }
  panel.addEventListener("dragstart", (e) => { const c = e.target.closest(".bd-chip"); if (c) { e.dataTransfer.setData("text/x-sg-uid", c.dataset.uid); e.dataTransfer.effectAllowed = "move"; } });
  panel.addEventListener("dragover", (e) => { const t = e.target.closest("[data-team], .bd-card[data-room^='t:']"); if (t && [...e.dataTransfer.types].includes("text/x-sg-uid")) { e.preventDefault(); t.classList.add("bd-over"); } });
  panel.addEventListener("dragleave", (e) => e.target.closest?.(".bd-over")?.classList.remove("bd-over"));
  panel.addEventListener("drop", (e) => {
    const uid = e.dataTransfer.getData("text/x-sg-uid"); if (!uid) return;
    const el = e.target.closest("[data-team], .bd-card[data-room^='t:']"); if (!el) return;
    e.preventDefault(); el.classList.remove("bd-over");
    const n = el.dataset.team != null ? +el.dataset.team : +el.dataset.room.slice(2);
    setState({ assign: { [uid]: n } });
  });

  /* ---------- багийн чат (баруун талд): «t:N» — баг хоорондоо, «b:N» — баг ↔ багш ---------- */
  const chats = new Map(), chatBox = q(".bd-chat"), chatList = q(".bd-ch-list"), chatTa = q(".bd-ch-f textarea");
  let chatOn = false, chatTab = "t", chatTeam = 0, chatShut = false;
  const SEEN = `sg_bdchat:${lid}:${bid}`;
  let seen = {};
  try { seen = JSON.parse(localStorage.getItem(SEEN) || "{}") || {}; } catch {}
  const myTeamOf = (s) => { const n = s.assign?.[me.uid]; return n != null && n < s.teams ? n : -1; };
  const myTeam = () => myTeamOf(st);
  const chatAvail = () => st.mode === "team" && (me.teacher || myTeam() >= 0);
  const chatTeamNow = () => (me.teacher ? chatTeam : myTeam());
  const chatCh = (k = chatTab) => `${k}:${chatTeamNow()}`;
  const unreadOf = (ch) => { const s = seen[ch] || ""; let n = 0; for (const m of chats.get(ch) || []) if (m.uid !== me.uid && m.id > s) n++; return n; };
  const hhmm = (ms) => { const d = new Date(ms); return `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`; };
  const linkify = (s) => s.replace(/https?:\/\/[^\s<]+/g, (u) => `<a href="${u}" target="_blank" rel="noopener noreferrer">${u}</a>`);
  async function loadChat() {
    if (!chatAvail()) return;
    try {
      const d = await api(base + "/chat"); if (dead) return;
      chats.clear(); for (const [k, v] of Object.entries(d.chats || {})) chats.set(k, v);
      renderChat();
    } catch {}
  }
  function onChat(d) {
    const list = chats.get(d.room) || []; chats.set(d.room, list);
    if (d.del) { const i = list.findIndex((m) => m.id === d.del); if (i >= 0) list.splice(i, 1); renderChat(); return; }
    const m = d.msg; if (!m || list.some((x) => x.id === m.id)) return;
    list.push(m); if (list.length > 300) list.shift();
    const viewing = chatOn && d.room === chatCh() && !document.hidden;
    if (m.uid !== me.uid && !viewing && (me.teacher ? d.room.startsWith("b:") : m.t)) toast(`💬 ${m.name}: ${m.body.length > 90 ? m.body.slice(0, 90) + "…" : m.body}`);
    renderChat();
  }
  function toggleChat(on) {
    chatOn = on === undefined ? !chatOn : !!on;
    if (chatOn && !panel.hidden) togglePanel(false);
    chatBox.hidden = !chatOn; root.classList.toggle("chat-on", chatOn);
    renderChat();
    if (chatOn && innerWidth > 720) setTimeout(() => !chatTa.disabled && chatTa.focus({ preventScroll: true }), 60);
  }
  // Нэр дээр дарахад тэр хүний самбар шууд харагдана: багш → багшийн самбар, суралцагч → багийнх нь самбар.
  function boardOf(uid, isTeacher) {
    if (isTeacher) return "main";
    const n = st.assign?.[uid];
    if (n == null || n >= st.teams) return "";
    return me.teacher || n === myTeam() ? "t:" + n : "";
  }
  function showBoard(r, keepChat) {
    if (!r) return;
    if (r !== room) switchRoom(r);
    if (!keepChat && innerWidth < 820 && chatOn) toggleChat(false);
  }
  function openTeam(n) { chatTeam = n; if (!chatOn) toggleChat(true); else renderChat(); showBoard("t:" + n, true); }
  function msgHTML(m, prev) {
    const mine = m.uid === me.uid, cont = prev && prev.uid === m.uid && m.at - prev.at < 180000;
    const head = cont ? "" : `<div class="bd-msg-h">${mine ? "" : `<button type="button" class="bd-ch-who" data-who="${esc(m.uid)}" data-wt="${m.t ? 1 : ""}" title="${m.t ? "Багшийн самбарыг харах" : "Самбарыг нь харах"}">${m.t ? "👩‍🏫 " : ""}${esc(m.name)}</button>`}<time>${hhmm(m.at)}</time></div>`;
    return `<div class="bd-msg${mine ? " me" : ""}${m.t ? " tch" : ""}${cont ? " cont" : ""}">${head}<p>${linkify(esc(m.body))}</p>${me.teacher ? `<button type="button" class="bd-msg-del" data-del="${esc(m.id)}" title="Устгах" aria-label="Устгах">${bi("x", 13)}</button>` : ""}</div>`;
  }
  function renderChat() {
    const avail = chatAvail(), team = chatTeamNow();
    document.documentElement.classList.toggle("bd-chatting", chatOn && avail);
    if (!avail && chatOn) { chatOn = false; chatBox.hidden = true; root.classList.remove("chat-on"); }
    if (me.teacher && chatTeam >= st.teams) chatTeam = 0;
    const ch = chatCh(), list = chats.get(ch) || [];
    if (chatOn && !document.hidden && list.length && seen[ch] !== list[list.length - 1].id) { // харж буй сувгийг уншсан гэж тэмдэглэнэ
      seen[ch] = list[list.length - 1].id;
      try { localStorage.setItem(SEEN, JSON.stringify(seen)); } catch {}
    }
    let total = 0;
    if (me.teacher) for (let i = 0; i < st.teams; i++) total += unreadOf("b:" + i);
    else if (team >= 0) total = unreadOf("t:" + team) + unreadOf("b:" + team);
    const btn = q(".bd-chatbtn"), bg = btn.querySelector(".bd-badge");
    btn.hidden = !avail; btn.classList.toggle("on", chatOn); bg.hidden = !total; bg.textContent = total > 99 ? "99+" : total;
    panel.querySelectorAll(".bd-chatmini[data-chteam]").forEach((b) => { const i = b.dataset.chteam, n = unreadOf("b:" + i) + unreadOf("t:" + i), x = b.querySelector("b"); x.textContent = n || ""; x.hidden = !n; });
    if (!chatOn) return;
    const t = teamOf(Math.max(team, 0));
    q(".bd-ch-sub").textContent = me.teacher ? "Нэр дээр дарж самбарыг нь шууд харна" : `${t[0]} ${t[1]} · нэр дээр дарж самбарыг харна`;
    const chips = q(".bd-ch-teams"); chips.hidden = !me.teacher;
    // Өөрчлөгдсөн хэсгийг л дахин зурна (хүн нэгдэх г.м. үйл явдлаар товч солигдож даралт алдагдахгүй).
    const setHTML = (el, h) => { if (el._h !== h) { el._h = h; el.innerHTML = h; return true; } return false; };
    if (me.teacher) setHTML(chips, Array.from({ length: st.teams }, (_, i) => {
      const tt = teamOf(i), b = unreadOf("b:" + i), u = unreadOf("t:" + i);
      return `<button type="button" class="bd-ch-team ${i === chatTeam ? "on" : ""}" data-chteam="${i}" style="--tc:${tt[2]}" title="${esc(tt[1])} — чат ба самбарыг нь харах">${tt[0]} ${esc(tt[1])}${b ? `<i class="bd-tb">${b}</i>` : u ? `<i class="bd-tdot"></i>` : ""}</button>`;
    }).join(""));
    setHTML(q(".bd-ch-tabs"), [["t", me.teacher ? "👥 Багийн доторх" : `👥 ${t[1]}`], ["b", me.teacher ? "🙋 Багштай" : "👩‍🏫 Багштай"]].map(([k, label]) => {
      const n = unreadOf(`${k}:${team}`);
      return `<button type="button" role="tab" data-chtab="${k}" class="${k === chatTab ? "on" : ""}" aria-selected="${k === chatTab}">${label}${n ? `<i class="bd-tb">${n}</i>` : ""}</button>`;
    }).join(""));
    const on = new Map(peers.map((p) => [p.uid, p])), tp = peers.find((p) => p.teacher);
    const mem = (uid, name, isT) => {
      const p = on.get(uid);
      return `<button type="button" class="bd-ch-mem ${p ? "on" : ""}" data-who="${esc(uid)}" data-wt="${isT ? 1 : ""}" title="${isT ? "Багшийн самбарыг харах" : "Багийн самбарыг харах"}"><span class="bd-vav" data-vuid="${esc(uid)}">${avatar({ uid, name })}<i class="bd-vring"></i></span><span>${isT ? "👩‍🏫 " : ""}${esc(name)}</span>${p?.mic ? `<i class="bd-mic-i" title="Микрофон асаалттай">${bi("mic", 12)}</i>` : ""}</button>`;
    };
    const ids = Object.keys(st.names || {}).filter((u) => st.assign?.[u] === team).sort((a, b) => (st.names[a] || "").localeCompare(st.names[b] || ""));
    setHTML(q(".bd-ch-members"), (tp ? mem(tp.uid, tp.name, true) : "") + ids.map((u) => mem(u, st.names[u], false)).join(""));
    const atBottom = chatList.dataset.ch !== ch || chatList.scrollHeight - chatList.scrollTop - chatList.clientHeight < 90;
    chatList.dataset.ch = ch;
    const changed = setHTML(chatList, list.length ? list.map((m, i) => msgHTML(m, list[i - 1])).join("")
      : `<p class="bd-ch-empty">${chatTab === "t" ? (me.teacher ? "Энэ багийнхан одоогоор хоорондоо бичээгүй байна." : "Багийнхантайгаа энд ярилцаарай 👋<br><small>Багш харах боломжтой.</small>")
        : (me.teacher ? "Энэ багаас асуулт ирээгүй байна.<br><small>Та бичвэл багийн бүх гишүүнд очно.</small>" : "Багаараа багшдаа асуулт, санал бичээрэй 🙋<br><small>Багийн бүх гишүүн харна.</small>")}</p>`);
    if (changed && atBottom) chatList.scrollTop = chatList.scrollHeight;
    const lockMe = !me.teacher && (st.lock || st.mode !== "team");
    chatTa.disabled = lockMe; q(".bd-ch-send").disabled = lockMe;
    chatTa.placeholder = lockMe ? "Багш самбарыг түр зогсоосон" : chatTab === "t" ? (me.teacher ? "Багт бичих…" : "Багийнхандаа бичих…") : me.teacher ? "Багт хариулах…" : "Багшдаа бичих…";
    q(".bd-ch-all").hidden = !(me.teacher && chatTab === "b");
  }
  const autoGrow = () => { chatTa.style.height = "auto"; chatTa.style.height = Math.min(120, chatTa.scrollHeight) + "px"; };
  async function sendChat() {
    const body = chatTa.value.trim(); if (!body || chatTa.disabled) return;
    const ch = me.teacher && chatTab === "b" && q(".bd-ch-all input").checked ? "all" : chatCh();
    chatTa.disabled = true;
    try {
      const r = await post("/chat", { ch, body });
      chatTa.value = ""; autoGrow();
      if (ch !== "all") for (const m of r?.msgs || []) onChat({ room: ch, msg: m });
    } catch (e) { toast(e.message, true); }
    finally { chatTa.disabled = false; renderChat(); if (!chatTa.disabled) chatTa.focus({ preventScroll: true }); }
  }
  async function delMsg(id) {
    if (!confirm("Энэ мессежийг устгах уу?")) return;
    try { await post("/chat", { ch: chatCh(), del: id }); } catch (e) { toast(e.message, true); }
  }
  q(".bd-ch-f").addEventListener("submit", (e) => { e.preventDefault(); sendChat(); });
  chatTa.addEventListener("keydown", (e) => { if (e.key === "Enter" && !e.shiftKey && !e.isComposing) { e.preventDefault(); sendChat(); } });
  chatTa.addEventListener("input", autoGrow);
  function chatState(prev, quiet) {
    if (myTeamOf(prev) !== myTeam() || (prev.mode !== "team" && st.mode === "team")) loadChat();
    if (!quiet && !me.teacher && st.mode === "team" && prev.mode !== "team" && myTeam() >= 0 && innerWidth >= 1100 && !chatShut) toggleChat(true);
  }

  /* ---------- дуут өрөө (Clubhouse маягийн): багш ба суралцагчид дуугаар ----------
     Дуу WebRTC-ээр шууд (P2P) явна — сервер зөвхөн холболтын дохиог дамжуулна. Шууд холбогдож чадаагүй
     (симметр NAT) сонсогчид л 16кГц ADPCM хэсгийг серверээр илгээнэ. Ярьж буй хүнийг цагираг, эквалайзераар харуулна. */
  const vbox = q(".bd-voice");
  const forceRelay = (() => { try { return localStorage.getItem("sg_voice_relay") === "1"; } catch { return false; } })();
  let mic = null, vTo = "all", vMuted = false, needTap = false, octx = null, gAll = null, gTeam = null, lastHeard = 0, vTicks = 0;
  const outLinks = new Map(), inLinks = new Map(), relayPlay = new Map(), talking = new Map();
  const onStage = (uid = me.uid) => (st.stage || []).includes(uid);
  const prio = (uid) => peers.some((p) => p.uid === uid && p.teacher) || onStage(uid); // багш, тайзан дээрх хүн ярихад багийн яриа намсна
  function micTarget() {
    if (me.teacher) { const n = vTo.startsWith("t:") ? +vTo.slice(2) : -1; return n >= 0 && (st.mode !== "team" || n >= st.teams) ? "all" : vTo; }
    if (onStage()) return "all";
    const n = myTeam();
    return st.mode === "team" && !st.lock && n >= 0 ? "t:" + n : "";
  }
  function listenersFor(to) {
    if (!to) return [];
    const n = to.startsWith("t:") ? +to.slice(2) : -1;
    return peers.filter((p) => p.uid !== me.uid && (to === "all" || (p.teacher ? p.room === to : st.assign?.[p.uid] === n))).map((p) => p.uid);
  }
  function audioOut() {
    if (!octx) {
      const AC = window.AudioContext || window.webkitAudioContext; octx = new AC();
      gAll = octx.createGain(); gTeam = octx.createGain(); gAll.connect(octx.destination); gTeam.connect(octx.destination);
    }
    return octx;
  }
  function unlockAudio() {
    audioOut().resume().catch(() => {});
    for (const L of inLinks.values()) L.audio.play().catch(() => {});
    if (needTap) { needTap = false; renderVoice(); }
  }
  root.addEventListener("pointerdown", () => { if (needTap || octx?.state === "suspended") unlockAudio(); }, true);
  async function micOn() {
    if (mic) return;
    if (!navigator.mediaDevices?.getUserMedia || !window.RTCPeerConnection) return toast("Энэ хөтөч микрофон дэмжихгүй байна — Chrome эсвэл Edge ашиглана уу", true);
    if (!micTarget()) return toast("Одоо ярих боломжгүй — ✋ гараа өргөөд багшийн зөвшөөрлийг хүлээнэ үү", true);
    let stream;
    try { stream = await navigator.mediaDevices.getUserMedia({ audio: { echoCancellation: true, noiseSuppression: true, autoGainControl: true, channelCount: 1 } }); }
    catch { return toast("Микрофон ашиглах зөвшөөрөл өгнө үү (хаягийн мөрний 🔒 дүрс → Микрофон)", true); }
    if (dead || mic) { stream.getTracks().forEach((t) => t.stop()); return; }
    const ctx = audioOut(); ctx.resume().catch(() => {});
    const src = ctx.createMediaStreamSource(stream), an = ctx.createAnalyser(); an.fftSize = 1024; src.connect(an);
    mic = { stream, track: stream.getAudioTracks()[0], src, an, buf: new Float32Array(an.fftSize), level: 0, talkMs: 0, lastRep: Date.now(),
      to: "", proc: null, sink: null, rs: null, pend: new Float32Array(0), idx: 0, hang: 0, seq: 0, q: [], busy: false };
    mic.track.onended = () => micOff();
    post("/voice", { mic: true }).catch(() => {});
    syncLinks(); renderTop(); renderVoice();
    toast(micTarget() === "all" ? "🎙 Микрофон асаалттай — бүх анги таныг сонсож байна" : "🎙 Микрофон асаалттай — багийнхан таныг сонсож байна");
  }
  function micOff(quiet) {
    if (!mic) return;
    const m = mic; mic = null;
    if (!quiet) post("/voice", { mic: false, talk: Math.round(m.talkMs) }).catch(() => {});
    m.stream.getTracks().forEach((t) => t.stop());
    try { m.src.disconnect(); m.proc?.disconnect(); m.sink?.disconnect(); } catch {}
    for (const uid of [...outLinks.keys()]) dropOut(uid, true);
    talking.delete(me.uid);
    q(".bd-micbtn")?.style.setProperty("--lv", "0");
    renderTop(); renderVoice();
  }
  // syncLinks — микрофон асаалттай үед сонсогч бүртэй нэг WebRTC холболт (зөвхөн илгээх).
  function syncLinks() {
    const to = mic ? micTarget() : "";
    if (mic && !to) { micOff(); toast("Микрофон унтарлаа — одоо ярих боломжгүй", true); return; }
    if (mic) mic.to = to;
    const want = new Set(listenersFor(to));
    for (const uid of [...outLinks.keys()]) if (!want.has(uid)) dropOut(uid);
    for (const uid of want) if (!outLinks.has(uid)) connectOut(uid);
    relayCheck();
  }
  async function connectOut(uid) {
    const L = { pc: null, id: rid(10), ok: false, relay: forceRelay, t: 0 }; outLinks.set(uid, L);
    if (forceRelay) { relayCheck(); return; }
    const pc = L.pc = new RTCPeerConnection({ iceServers: ICE });
    pc.addTransceiver(mic.track, { direction: "sendonly", streams: [mic.stream] });
    const fail = () => { if (outLinks.get(uid) === L && !L.ok) { L.relay = true; relayCheck(); renderVoice(); } };
    pc.onconnectionstatechange = () => {
      const s = pc.connectionState;
      if (s === "connected") { L.ok = true; L.relay = false; clearTimeout(L.t); relayCheck(); renderVoice(); }
      else if (s === "failed") { L.ok = false; fail(); }
      else if (s === "disconnected") { clearTimeout(L.t); L.t = setTimeout(() => { if (pc.connectionState !== "connected") { L.ok = false; fail(); } }, 4000); }
    };
    L.t = setTimeout(fail, 9000);
    try {
      await pc.setLocalDescription(await pc.createOffer());
      await iceDone(pc);
      if (outLinks.get(uid) !== L) return;
      await post("/voice", { sig: { to: uid, t: "offer", id: L.id, sdp: pc.localDescription.sdp } });
    } catch { fail(); }
  }
  function dropOut(uid, silent) {
    const L = outLinks.get(uid); if (!L) return;
    outLinks.delete(uid); clearTimeout(L.t);
    try { L.pc?.close(); } catch {}
    if (!silent && L.pc) post("/voice", { sig: { to: uid, t: "bye", id: L.id } }).catch(() => {});
    relayCheck();
  }
  async function onRtc(r) {
    if (!r) return;
    if (r.t === "answer") { const L = outLinks.get(r.from); if (L?.pc && L.id === r.id) L.pc.setRemoteDescription({ type: "answer", sdp: r.sdp }).catch(() => { L.relay = true; relayCheck(); }); return; }
    if (r.t === "bye") { if (inLinks.get(r.from)?.id === r.id) dropIn(r.from); return; }
    dropIn(r.from); // offer: ярьж буй хүнээс шинэ холболт
    const pc = new RTCPeerConnection({ iceServers: ICE }), audio = new Audio();
    audio.autoplay = true; audio.muted = vMuted;
    const L = { pc, id: r.id, audio }; inLinks.set(r.from, L);
    pc.ontrack = (e) => { audio.srcObject = e.streams[0] || new MediaStream([e.track]); audio.play().catch(() => { needTap = true; renderVoice(); }); };
    try {
      await pc.setRemoteDescription({ type: "offer", sdp: r.sdp });
      await pc.setLocalDescription(await pc.createAnswer());
      await iceDone(pc);
      if (inLinks.get(r.from) !== L) return;
      await post("/voice", { sig: { to: r.from, t: "answer", id: r.id, sdp: pc.localDescription.sdp } });
    } catch {}
  }
  function dropIn(uid) { const L = inLinks.get(uid); if (!L) return; inLinks.delete(uid); try { L.pc.close(); } catch {} L.audio.srcObject = null; }
  // Нөөц зам: WebRTC холбогдоогүй сонсогч байвал л микрофоныг 16кГц ADPCM болгож серверээр (зөвхөн тэдэнд).
  function relayCheck() {
    const need = !!mic && [...outLinks.values()].some((L) => L.relay);
    if (need && !mic.proc) {
      const ctx = audioOut(), proc = ctx.createScriptProcessor(4096, 1, 1), sink = ctx.createGain(); sink.gain.value = 0;
      mic.rs = resampler(ctx.sampleRate); mic.proc = proc; mic.sink = sink;
      proc.onaudioprocess = (e) => relayData(e.inputBuffer.getChannelData(0));
      mic.src.connect(proc); proc.connect(sink); sink.connect(ctx.destination);
    } else if (!need && mic?.proc) {
      try { mic.src.disconnect(mic.proc); mic.proc.disconnect(); mic.sink.disconnect(); } catch {}
      mic.proc = mic.sink = null; mic.pend = new Float32Array(0);
    }
  }
  function relayData(input) {
    if (!mic || !mic.rs || !mic.track.enabled) return;
    mic.pend = f32cat(mic.pend, mic.rs(input));
    while (mic && mic.pend.length >= VCH) {
      const c = mic.pend.subarray(0, VCH); mic.pend = mic.pend.slice(VCH);
      let s = 0; for (const v of c) s += v * v;
      const rms = Math.sqrt(s / VCH), echo = performance.now() - lastHeard < 300; // бусдын дуу гарч байхад цуурайг шүүнэ
      if (rms > (echo ? 0.05 : 0.01)) mic.hang = 4; else if (mic.hang > 0) mic.hang--; else continue; // чимээгүй үед илгээхгүй
      const only = [...outLinks].filter(([, L]) => L.relay).map(([u]) => u); if (!only.length) return;
      const pcm = new Int16Array(VCH); for (let i = 0; i < VCH; i++) pcm[i] = Math.max(-1, Math.min(1, c[i])) * 32767;
      const enc = adEnc(pcm, mic.idx); mic.idx = enc.idx;
      mic.q.push({ to: mic.to, seq: ++mic.seq, d: b64(enc.data), only }); if (mic.q.length > 5) mic.q.shift();
      pump();
    }
  }
  async function pump() {
    if (!mic || mic.busy) return;
    const m = mic; m.busy = true;
    try {
      while (mic === m && m.q.length) {
        try { await post("/voice", m.q.shift()); }
        catch (e) { if (e.status === 403) { micOff(); toast(e.message, true); return; } }
      }
    } finally { m.busy = false; }
  }
  function onVoice(v) {
    if (!v?.d || v.uid === me.uid) return;
    talking.set(v.uid, performance.now()); lastHeard = performance.now();
    if (vMuted) return;
    const ctx = audioOut();
    if (ctx.state !== "running") { ctx.resume().catch(() => {}); if (!needTap) { needTap = true; renderVoice(); } return; }
    const p = relayPlay.get(v.uid) || { next: 0, seq: 0 }; relayPlay.set(v.uid, p);
    if (v.seq <= p.seq && p.seq - v.seq < 50) return; // хоцорсон хэсэг
    p.seq = v.seq;
    const pcm = adDec(unb64(v.d)), buf = ctx.createBuffer(1, pcm.length, VSR); buf.getChannelData(0).set(pcm);
    const now = ctx.currentTime; if (p.next < now || p.next > now + 1.2) p.next = now + 0.22; // життер буфер
    const s = ctx.createBufferSource(); s.buffer = buf; s.connect(prio(v.uid) ? gAll : gTeam); s.start(p.next); p.next += buf.duration;
  }
  async function toggleHand() {
    const up = !(st.hands || []).includes(me.uid);
    try { await post("/voice", { hand: up }); toast(up ? "✋ Гараа өргөлөө — багш ярих эрх өгөхийг хүлээнэ үү" : "Гараа буулгалаа"); } catch (e) { toast(e.message, true); }
  }
  // 100мс тутам: өөрийн микрофоны түвшин, бусдын ярьж буй эсэх (RTP audioLevel), ярихад намсгах.
  function vTick() {
    const now = performance.now();
    if (mic) {
      mic.an.getFloatTimeDomainData(mic.buf); let s = 0; for (const v of mic.buf) s += v * v;
      const rms = Math.sqrt(s / mic.buf.length);
      mic.level = mic.level * 0.55 + Math.min(1, rms * 7) * 0.45;
      if (rms > 0.02) { mic.talkMs += 100; talking.set(me.uid, now); }
      if (Date.now() - mic.lastRep > 10000) { const ms = Math.round(mic.talkMs); mic.talkMs = 0; mic.lastRep = Date.now(); if (ms) post("/voice", { talk: ms }).catch(() => {}); }
      q(".bd-micbtn")?.style.setProperty("--lv", mic.level.toFixed(2));
    }
    for (const [uid, L] of inLinks) for (const r of L.pc.getReceivers()) { const x = r.getSynchronizationSources?.()[0]; if (x && x.audioLevel > 0.02) { talking.set(uid, now); lastHeard = now; } }
    if (++vTicks % 3 === 0) for (const [uid, L] of inLinks) { // 300мс-ийн дундаж энерги: богино авиа ч алдагдахгүй
      L.pc.getStats().then((rep) => rep.forEach((x) => {
        if (x.type !== "inbound-rtp" || x.kind !== "audio" || x.totalSamplesDuration == null) return;
        if (L.d != null && x.totalSamplesDuration > L.d && Math.sqrt((x.totalAudioEnergy - L.e) / (x.totalSamplesDuration - L.d)) > 0.015) { talking.set(uid, performance.now()); lastHeard = performance.now(); }
        L.e = x.totalAudioEnergy; L.d = x.totalSamplesDuration;
      })).catch(() => {});
    }
    let prioTalk = false; const who = [];
    for (const [uid, t] of talking) { if (now - t > 450) { talking.delete(uid); continue; } if (uid !== me.uid) { who.push(uid); if (prio(uid)) prioTalk = true; } }
    root.querySelectorAll("[data-vuid]").forEach((el) => el.classList.toggle("talk", talking.has(el.dataset.vuid)));
    const vol = prioTalk ? 0.3 : 1;
    for (const [uid, L] of inLinks) L.audio.volume = prio(uid) ? 1 : vol;
    if (gTeam) gTeam.gain.value = vol;
    vbox.classList.toggle("talking", who.length > 0 || talking.has(me.uid));
    const now2 = q(".bd-vnow");
    if (now2) {
      const names = who.map((u) => { const p = peers.find((x) => x.uid === u); return p ? (p.teacher ? "👩‍🏫 Багш" : p.name) : st.names?.[u] || "Хэн нэгэн"; });
      const txt = names.length ? `${names.slice(0, 2).join(", ")}${names.length > 2 ? ` +${names.length - 2}` : ""} ярьж байна` : talking.has(me.uid) ? "Та ярьж байна" : "";
      if (now2.textContent !== txt) now2.textContent = txt;
      now2.hidden = !txt;
    }
  }
  const vTimer = setInterval(vTick, 100);
  function renderVoice() {
    const nameOf = (uid) => peers.find((p) => p.uid === uid)?.name || st.names?.[uid] || "Суралцагч";
    const n = myTeam();
    const rel = peers.filter((p) => p.mic && p.uid !== me.uid && (me.teacher || p.teacher || onStage(p.uid) || (st.mode === "team" && n >= 0 && st.assign?.[p.uid] === n)));
    const queue = me.teacher && ((st.hands || []).length || (st.stage || []).length);
    const show = !!mic || rel.length > 0 || queue || needTap;
    vbox.hidden = !show;
    if (!show) return;
    const av = (p) => `<span class="bd-vav" data-vuid="${esc(p.uid)}" title="${esc(p.teacher ? "Багш · " + p.name : p.name)}">${avatar(p)}<i class="bd-vring"></i>${p.teacher ? `<i class="bd-vbadge">👩‍🏫</i>` : ""}</span>`;
    const relayN = [...outLinks.values()].filter((L) => L.relay).length;
    let h = `<div class="bd-vroom"><span class="bd-vlive"><i></i>ШУУД</span><b>Дуут өрөө</b>
      <span class="bd-vavs">${mic ? av({ uid: me.uid, name: me.name, teacher: me.teacher }) : ""}${rel.slice(0, 7).map(av).join("")}${rel.length > 7 ? `<span class="bd-av bd-more">+${rel.length - 7}</span>` : ""}</span>
      <span class="bd-eq" aria-hidden="true"><i></i><i></i><i></i><i></i><i></i></span>
      <button type="button" class="bd-ib bd-sm" data-a="vmute" title="${vMuted ? "Дууг нээх" : "Дууг хаах (бусдыг сонсохгүй)"}">${bi(vMuted ? "volx" : "vol", 16)}</button></div>
      <div class="bd-vnow" hidden></div>`;
    if (mic && relayN) h += `<div class="bd-vinfo" title="Эдгээр сонсогчийн сүлжээ шууд холболт зөвшөөрөөгүй тул дуу серверээр дамжиж байна">↪ ${relayN} сонсогч серверээр</div>`;
    if (needTap) h += `<div class="bd-vtap"><button type="button" data-a="vtap">🔊 Дуу сонсохын тулд энд дарна уу</button></div>`;
    if (queue) h += `<div class="bd-vq">${(st.hands || []).map((u) => `<span class="bd-vh">✋ ${esc(nameOf(u))}<button type="button" data-grant="${esc(u)}">🎤 Ярих эрх</button><button type="button" class="bd-vx" data-revoke="${esc(u)}" title="Гарыг буулгах">${bi("x", 12)}</button></span>`).join("")}${(st.stage || []).map((u) => `<span class="bd-vh on">🎤 ${esc(nameOf(u))} · бүх ангид<button type="button" class="bd-vx" data-revoke="${esc(u)}" title="Ярих эрхийг авах">${bi("x", 12)}</button></span>`).join("")}</div>`;
    if (vbox._h !== h) { vbox._h = h; vbox.innerHTML = h; }
    vTick();
  }
  function voiceState(prev, quiet) {
    if (!quiet) {
      if (me.teacher) { const nw = (st.hands || []).filter((u) => !(prev.hands || []).includes(u)); if (nw.length) toast(`✋ ${nw.map((u) => st.names?.[u] || "Суралцагч").join(", ")} гар өргөлөө — «Дуут өрөө» хэсгээс ярих эрх өгнө үү`); }
      else if (onStage() && !(prev.stage || []).includes(me.uid)) toast("🎤 Багш танд бүх ангид ярих эрх өглөө — «Ярих» дарж микрофоноо асаагаарай");
      else if (!onStage() && (prev.stage || []).includes(me.uid)) toast("Ярих эрх дууслаа — баярлалаа 🙏");
    }
    if (mic) syncLinks();
  }
  function peersVoice() {
    for (const uid of [...inLinks.keys()]) if (!peers.some((p) => p.uid === uid && p.mic)) dropIn(uid);
    if (mic) syncLinks();
    renderVoice();
  }
  q(".bd-vto").addEventListener("change", (e) => { vTo = e.target.value; if (mic) syncLinks(); renderTop(); renderVoice(); });

  /* ---------- товшилтууд ---------- */
  root.addEventListener("click", async (e) => {
    const b = e.target.closest("button, [data-a]"); if (!b || !root.contains(b)) return;
    if (b.closest(".bd-pop")) return popClick(e, b);
    if (b.dataset.t) {
      if (b.dataset.pop) { setTool(b.dataset.t); openPop(b.dataset.pop, b); return; }
      closePops(); setTool(b.dataset.t); return;
    }
    if (b.dataset.pop) { openPop(b.dataset.pop, b); return; }
    if (b.dataset.mode) { if (b.dataset.mode !== st.mode) setState({ mode: b.dataset.mode, spot: "" }); return; }
    if (b.dataset.room) { if (b.dataset.room !== room) switchRoom(b.dataset.room); return; }
    if (b.dataset.open) { if (b.dataset.open !== room) switchRoom(b.dataset.open); if (innerWidth < 900) togglePanel(false); return; }
    if (b.dataset.spot) { setState({ spot: st.spot === b.dataset.spot ? "" : b.dataset.spot }); return; }
    if (b.dataset.teams) { setState({ teams: clamp(st.teams + +b.dataset.teams, 2, 12) }); return; }
    if (b.dataset.chteam != null) { openTeam(+b.dataset.chteam); return; }
    if (b.dataset.chtab) { chatTab = b.dataset.chtab; renderChat(); if (!chatTa.disabled && innerWidth > 720) chatTa.focus({ preventScroll: true }); return; }
    if (b.dataset.who) { showBoard(boardOf(b.dataset.who, !!b.dataset.wt)); return; }
    if (b.dataset.del) { delMsg(b.dataset.del); return; }
    if (b.dataset.grant) { setState({ stage: { [b.dataset.grant]: true } }); return; }
    if (b.dataset.revoke) { setState({ stage: { [b.dataset.revoke]: false } }); return; }
    const a = b.dataset.a;
    if (a === "chat") { if (chatOn) chatShut = true; return toggleChat(); }
    if (a === "mic") return mic ? micOff() : micOn();
    if (a === "hand") return toggleHand();
    if (a === "vmute") { vMuted = !vMuted; for (const L of inLinks.values()) L.audio.muted = vMuted; toast(vMuted ? "🔇 Дуут өрөөний дууг хаалаа" : "🔊 Дуут өрөөний дуу нээгдлээ"); renderVoice(); return; }
    if (a === "vtap") return unlockAudio();
    if (a === "close") return close();
    if (a === "undo") return undo();
    if (a === "redo") return redo();
    if (a === "zin") return zoomTo(cv.getZoom() * 1.25);
    if (a === "zout") return zoomTo(cv.getZoom() / 1.25);
    if (a === "fit") return fitView();
    if (a === "panel") return togglePanel();
    if (a === "lock") return setState({ lock: !st.lock });
    if (a === "unspot") return setState({ spot: "" });
    if (a === "spot-here") return setState({ spot: room });
    if (a === "shuffle") return setState({ shuffle: true });
    if (a === "follow") { follow = true; if (lastView) applyView(lastView); renderBanner(); return; }
    if (a === "distribute") return distribute();
  });
  root.addEventListener("change", (e) => { if (e.target.matches('input[data-a="copy"]')) setState({ copy: e.target.checked }); });
  async function distribute() {
    closePops();
    if (st.mode === "teach") return toast("Эхлээд «Хувиараа» эсвэл «Багаар» горимд шилжүүлнэ үү");
    if (!confirm("Таны самбарыг бүх суралцагчийн (багийн) самбарт хуулна. Тэдний самбар дээрх одоогийн зураг солигдоно. Үргэлжлүүлэх үү?")) return;
    await flushAll();
    const d = await setState({ distribute: true });
    if (d) toast("Ажлын хуудас тарааллаа ✓");
  }
  function popClick(e, b) {
    const p = b.closest(".bd-pop");
    if (b.dataset.sh) { shapeKind = b.dataset.sh; setTool("shape"); closePops(); return; }
    if (b.dataset.img === "file") { closePops(); fileIn.click(); return; }
    if (b.dataset.hit) { const h = isr.items[+b.dataset.hit]; if (h) { insertHit(h); b.classList.add("bd-picked"); if (innerWidth <= 720) closePops(); } return; }
    if (b.dataset.img === "more") { searchImages(p, true); return; }
    if (b.dataset.idea) { isr.q = b.dataset.idea; p.querySelector(".bd-isf input").value = isr.q; searchImages(p); return; }
    if ("ik" in b.dataset) { isr.kind = b.dataset.ik; p.querySelectorAll("[data-ik]").forEach((x) => x.classList.toggle("on", x === b)); if (isr.q) searchImages(p); return; }
    if (b.dataset.cat) { p.querySelectorAll(".bd-tabs button").forEach((x) => x.classList.toggle("on", x === b)); p.querySelector(".bd-search").value = ""; const grid = p.querySelector(".bd-igrid"); const items = iconData.cats[+b.dataset.cat].i; grid.innerHTML = items.map(([n, t, inner]) => `<button type="button" draggable="true" data-ic="${esc(n)}" title="${esc(t)}"><svg viewBox="0 0 24 24" width="26" height="26" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">${inner}</svg><small>${esc(t)}</small></button>`).join(""); return; }
    if (b.dataset.ic) { addIcon(p._inner.get(b.dataset.ic)); return; }
    if (b.dataset.ecat) { p.querySelectorAll(".bd-tabs button").forEach((x) => x.classList.toggle("on", x === b)); showEmoji(p, +b.dataset.ecat); return; }
    if (b.dataset.em) { addEmoji(b.dataset.em); return; }
    if (b.dataset.bg) { style.bg = b.dataset.bg; saveStyle(); p.querySelectorAll("[data-bg]").forEach((x) => x.classList.toggle("on", x === b)); renderTop(); return; }
    const a = b.dataset.a;
    if (a === "export") { closePops(); exportPNG(); return; }
    if (a === "clear") { closePops(); clearBoard(); return; }
    if (a === "distribute") { distribute(); return; }
  }
  root.addEventListener("submit", (e) => {
    const f = e.target.closest("form.bd-f"); if (!f) return;
    e.preventDefault();
    if (f.dataset.img === "search") { const p = f.closest(".bd-pop"), v = f.q.value.trim(); if (v) { isr.q = v; searchImages(p); } return; }
    if (f.dataset.img === "google") window.open("https://www.google.com/search?tbm=isch&safe=active&q=" + encodeURIComponent(f.q.value.trim()), "_blank", "noopener");
    else if (f.dataset.img === "url") { closePops(); insertURL(f.u.value); }
  });
  root.addEventListener("dragstart", (e) => {
    const hit = e.target.closest?.("[data-hit]");
    if (hit && isr.items[+hit.dataset.hit]) { e.dataTransfer.setData("text/x-sg-img", JSON.stringify(isr.items[+hit.dataset.hit])); e.dataTransfer.effectAllowed = "copy"; }
    const em = e.target.closest?.("[data-em]"), ic = e.target.closest?.("[data-ic]");
    if (em) e.dataTransfer.setData("text/x-sg-emoji", em.dataset.em);
    if (ic) e.dataTransfer.setData("text/x-sg-icon", ic.closest(".bd-pop")._inner.get(ic.dataset.ic));
  });
  root.addEventListener("pointerdown", (e) => { if (!e.target.closest(".bd-pop, [data-pop]")) closePops(); }, true);

  function clearBoard() {
    if (!canEdit) return;
    const list = cv.getObjects().filter(editable); if (!list.length) return toast("Самбар аль хэдийн хоосон байна");
    if (!confirm(me.teacher ? "Энэ самбарын бүх зүйлийг арилгах уу? (Ctrl+Z-ээр буцааж болно)" : "Өөрийн зурсан бүх зүйлийг арилгах уу? (Ctrl+Z-ээр буцааж болно)")) return;
    cv.discardActiveObject();
    record(list.map((o) => { const j = ser(o); removeId(o.id); return [o.id, j, null]; }));
    send({ t: "clear" }); cv.requestRenderAll();
  }
  function exportPNG() {
    const r = contentRect(); if (!r) return toast("Самбар хоосон байна");
    const vpt = cv.viewportTransform.slice(), pad = 40, mult = Math.min(2, 8000 / Math.max(r.w + pad * 2, r.h + pad * 2));
    cv.discardActiveObject(); cv.setViewportTransform([1, 0, 0, 1, 0, 0]); cv.backgroundColor = "#ffffff";
    const url = cv.toDataURL({ format: "png", left: r.x - pad, top: r.y - pad, width: r.w + pad * 2, height: r.h + pad * 2, multiplier: mult });
    cv.backgroundColor = ""; cv.setViewportTransform(vpt); afterView();
    const a = document.createElement("a"); a.href = url; a.download = `${title.replace(/[\\/:*?"<>|]+/g, " ").trim() || "самбар"}.png`; a.click();
  }

  /* ---------- амьд байх, хаах ---------- */
  let lastBeat = Date.now();
  async function beat() {
    if (dead || document.hidden && Date.now() - lastBeat < 30000) return;
    lastBeat = Date.now();
    try {
      const r = await post("/presence", { room });
      if (dead) return;
      if (r.room !== room) switchRoom(r.room);
      else if (r.sv !== st.v || (r.v !== ver && !inflight && !queue.length && !syncing)) resync();
    } catch {}
  }
  const hb = setInterval(() => { const wsOK = Live.ws?.readyState === 1; if (!wsOK || Date.now() - lastBeat > 14000) beat(); }, 3000);
  const ro = new ResizeObserver(() => { cv.setDimensions({ width: wrap.clientWidth, height: wrap.clientHeight }); afterView(); });
  ro.observe(wrap);
  document.addEventListener("keydown", onKey);
  document.addEventListener("keyup", onKeyUp);
  document.addEventListener("copy", onCopy);
  document.addEventListener("paste", onPaste);
  Live.handlers.add(onLive);
  history.pushState({ sgBoard: 1 }, "", location.href);
  const onPop = () => destroy();
  addEventListener("popstate", onPop);
  function close() { destroy(); if (history.state?.sgBoard) history.back(); }
  function destroy() {
    if (dead) return; dead = true;
    if (queue.length) flush();
    micOff(true); for (const uid of [...inLinks.keys()]) dropIn(uid);
    clearInterval(vTimer); try { octx?.close(); } catch {}
    fetch(base + "/presence", { method: "POST", keepalive: true, headers: { Authorization: "Bearer " + Auth.token, "Content-Type": "application/json" }, body: JSON.stringify({ leave: true }) }).catch(() => {});
    clearInterval(hb); ro.disconnect(); Live.handlers.delete(onLive);
    document.removeEventListener("keydown", onKey); document.removeEventListener("keyup", onKeyUp);
    document.removeEventListener("copy", onCopy); document.removeEventListener("paste", onPaste);
    removeEventListener("popstate", onPop);
    try { cv.dispose(); } catch {}
    root.remove(); document.documentElement.classList.remove("bd-on", "bd-chatting");
    if (cur?.destroy === destroy) cur = null;
  }

  /* ---------- эхлэл ---------- */
  (async () => {
    try {
      Live.connect(Auth.token);
      cv.setDimensions({ width: wrap.clientWidth, height: wrap.clientHeight });
      const [d] = await Promise.all([api(base), ensureFonts()]);
      if (dead) return;
      me = d.me; st = d.state || st; st.assign ||= {}; st.names ||= {}; peers = d.peers || []; room = d.room; canEdit = !!d.write;
      await loadObjects(d.objects || []);
      ver = d.v;
      F.util.clearFabricFontCache?.();
      fitView();
      setTool(canEdit ? "pen" : "hand");
      if (me.teacher) (d.rooms || []).forEach((r) => thumbs.set(r.room, { need: true }));
      renderChrome();
      q(".bd-loading")?.remove();
      if (me.teacher && st.mode !== "teach" && innerWidth >= 1100) togglePanel(true);
      if (chatAvail()) { loadChat(); if (!me.teacher && innerWidth >= 1100) toggleChat(true); }
    } catch (e) {
      const l = q(".bd-loading");
      if (l) l.innerHTML = `<p>${esc(e.message || "Самбар нээгдсэнгүй")}</p><button type="button" class="btn btn-glass btn-sm" data-a="close">Хаах</button>`;
    }
  })();

  return { destroy, cv, get room() { return room; }, get state() { return st; },
    // Дуут өрөөний оношилгоо (SGBoard.current.voice): холболт бүрийн төлөв, нөөц зам.
    get voice() {
      const lvl = (pc) => pc?.getReceivers().map((r) => r.getSynchronizationSources?.()[0]?.audioLevel ?? null)[0] ?? null;
      return { mic: !!mic, to: mic?.to || "", out: [...outLinks].map(([u, L]) => ({ u, state: L.pc?.connectionState || "relay", relay: L.relay })),
        in: [...inLinks].map(([u, L]) => ({ u, state: L.pc.connectionState, level: lvl(L.pc), playing: !L.audio.paused })) };
    } };
}

/* ================= зураг засагч: тайрах, лассо, дэвсгэр арилгах ================= */
function imageEditor(src) {
  return new Promise((resolve) => {
    const m = document.createElement("div");
    m.className = "bd-ie";
    m.innerHTML = `<div class="bd-ie-card" role="dialog" aria-modal="true" aria-label="Зураг засах">
      <header><b>Зураг засах</b><span class="muted small">Тайрах · хайчлах · дэвсгэр арилгах</span><button type="button" class="bd-ib" data-ie="cancel" title="Болих">${bi("x")}</button></header>
      <div class="bd-ie-body">
        <aside class="bd-ie-tools">
          <button type="button" data-m="auto" class="bd-ie-auto">${bi("wand", 18)}<span>Дэвсгэр арилгах</span></button>
          <button type="button" data-m="wand">${bi("target", 18)}<span>Дарж арилгах</span></button>
          <button type="button" data-m="crop">${bi("crop", 18)}<span>Тайрах</span></button>
          <button type="button" data-m="lasso">${bi("lasso", 18)}<span>Лассогоор хайчлах</span></button>
          <button type="button" data-m="erase">${bi("eraser", 18)}<span>Баллуур</span></button>
          <button type="button" data-m="restore">${bi("restore", 18)}<span>Сэргээх бийр</span></button>
          <label class="bd-ie-rng">Мэдрэмж <output>28</output><input type="range" data-r="tol" min="4" max="90" value="28"></label>
          <label class="bd-ie-rng">Бийрний хэмжээ <output>36</output><input type="range" data-r="brush" min="6" max="160" value="36"></label>
          <div class="bd-ie-row"><button type="button" data-ie="undo" disabled>${bi("undo", 16)}Буцаах</button><button type="button" data-ie="reset" disabled>Анхны төлөв</button></div>
        </aside>
        <div class="bd-ie-stage"><canvas></canvas><p class="bd-ie-hint"></p><div class="bd-ie-apply" hidden></div></div>
      </div>
      <footer><button type="button" class="btn btn-ghost" data-ie="cancel">Болих</button><button type="button" class="btn btn-gold" data-ie="save">${bi("check", 17)}Самбарт хадгалах</button></footer>
    </div>`;
    document.body.append(m);
    const view = m.querySelector("canvas"), vx = view.getContext("2d"), stageEl = m.querySelector(".bd-ie-stage"), hint = m.querySelector(".bd-ie-hint"), applyBar = m.querySelector(".bd-ie-apply");
    const mk = (w, h) => { const c = document.createElement("canvas"); c.width = Math.max(1, Math.round(w)); c.height = Math.max(1, Math.round(h)); c.ctx = c.getContext("2d", { willReadFrequently: true }); return c; };
    const cloneC = (c) => { const n = mk(c.width, c.height); n.ctx.drawImage(c, 0, 0); return n; };
    let cur = null, orig = null, s0 = 1, offX = 0, offY = 0, mode = "", tol = 28, brush = 36, hist = [], crop = null, lasso = null, dragging = null, hover = null;
    let sx = 1, ox = 0, oy = 0, dirty = false;
    const HINTS = {
      auto: "Ирмэгт залгаа нэг өнгийн дэвсгэрийг арилгалаа. Үлдсэн хэсгийг «Дарж арилгах» хэрэгслээр арилгана уу.",
      wand: "Арилгах өнгөтэй хэсэг дээр дарна (мэдрэмжийг тохируулж болно)",
      crop: "Хүрээний булан, ирмэгийг чирж тайрах хэсгээ сонгоно",
      lasso: "Үлдээх (эсвэл хасах) хэсгийнхээ эргэн тойронд зурна",
      erase: "Арилгах газраа будна", restore: "Андуурч арилгасан хэсгийг будаж сэргээнэ",
    };
    const im = new Image();
    im.onload = () => {
      s0 = Math.min(1, 2000 / Math.max(im.naturalWidth, im.naturalHeight));
      cur = mk(im.naturalWidth * s0, im.naturalHeight * s0); cur.ctx.drawImage(im, 0, 0, cur.width, cur.height);
      orig = cloneC(cur); size(); setMode("wand");
    };
    im.onerror = () => { toast("Зураг ачаалагдсангүй", true); done(null); };
    im.src = src;

    function size() {
      const W = stageEl.clientWidth, H = stageEl.clientHeight, dpr = devicePixelRatio || 1;
      view.width = W * dpr; view.height = H * dpr; view.style.width = W + "px"; view.style.height = H + "px";
      draw();
    }
    function draw() {
      if (!cur) return;
      const W = stageEl.clientWidth, H = stageEl.clientHeight, dpr = devicePixelRatio || 1;
      sx = Math.min((W - 48) / cur.width, (H - 48) / cur.height, 2); ox = (W - cur.width * sx) / 2; oy = (H - cur.height * sx) / 2;
      vx.setTransform(dpr, 0, 0, dpr, 0, 0); vx.clearRect(0, 0, W, H);
      const cs = 10; vx.save(); vx.beginPath(); vx.rect(ox, oy, cur.width * sx, cur.height * sx); vx.clip();
      for (let y = 0; y * cs < cur.height * sx; y++) for (let x = 0; x * cs < cur.width * sx; x++) { vx.fillStyle = (x + y) % 2 ? "#e6e9f0" : "#ffffff"; vx.fillRect(ox + x * cs, oy + y * cs, cs, cs); }
      vx.restore();
      vx.imageSmoothingQuality = "high"; vx.drawImage(cur, ox, oy, cur.width * sx, cur.height * sx);
      if (mode === "crop" && crop) {
        const r = toV(crop);
        vx.fillStyle = "rgba(18,24,41,.5)"; vx.beginPath(); vx.rect(ox, oy, cur.width * sx, cur.height * sx); vx.rect(r.x, r.y, r.w, r.h); vx.fill("evenodd");
        vx.strokeStyle = "#fff"; vx.lineWidth = 2; vx.strokeRect(r.x, r.y, r.w, r.h);
        vx.strokeStyle = "rgba(255,255,255,.5)"; vx.lineWidth = 1; vx.beginPath();
        for (const f of [1 / 3, 2 / 3]) { vx.moveTo(r.x + r.w * f, r.y); vx.lineTo(r.x + r.w * f, r.y + r.h); vx.moveTo(r.x, r.y + r.h * f); vx.lineTo(r.x + r.w, r.y + r.h * f); }
        vx.stroke();
        vx.fillStyle = "#fff"; for (const [hx, hy] of handles(r)) { vx.beginPath(); vx.arc(hx, hy, 6, 0, 7); vx.fill(); vx.strokeStyle = "#1f3c8f"; vx.lineWidth = 2; vx.stroke(); }
      }
      if (lasso?.length > 1) {
        vx.save(); vx.lineWidth = 2; vx.setLineDash([6, 4]); vx.strokeStyle = "#1f3c8f"; vx.fillStyle = "rgba(31,60,143,.12)";
        vx.beginPath(); lasso.forEach((p, i) => { const x = ox + p.x * sx, y = oy + p.y * sx; i ? vx.lineTo(x, y) : vx.moveTo(x, y); });
        if (!dragging) vx.closePath(); vx.fill(); vx.stroke(); vx.restore();
      }
      if (hover && (mode === "erase" || mode === "restore")) { vx.beginPath(); vx.arc(hover.vx, hover.vy, brush / 2, 0, 7); vx.strokeStyle = "#1f3c8f"; vx.lineWidth = 1.5; vx.stroke(); }
    }
    const toV = (r) => ({ x: ox + r.x * sx, y: oy + r.y * sx, w: r.w * sx, h: r.h * sx });
    const handles = (r) => [[r.x, r.y], [r.x + r.w / 2, r.y], [r.x + r.w, r.y], [r.x + r.w, r.y + r.h / 2], [r.x + r.w, r.y + r.h], [r.x + r.w / 2, r.y + r.h], [r.x, r.y + r.h], [r.x, r.y + r.h / 2]];
    const HN = ["nw", "n", "ne", "e", "se", "s", "sw", "w"];
    const toImg = (e) => { const b = view.getBoundingClientRect(), x = e.clientX - b.left, y = e.clientY - b.top; return { x: (x - ox) / sx, y: (y - oy) / sx, vx: x, vy: y }; };
    function push() { hist.push({ cur: cloneC(cur), orig, offX, offY }); if (hist.length > 14) hist.shift(); dirty = true; syncBtns(); }
    function syncBtns() { m.querySelector('[data-ie="undo"]').disabled = !hist.length; m.querySelector('[data-ie="reset"]').disabled = !dirty; }
    function setMode(k) {
      mode = k; crop = k === "crop" ? { x: cur.width * 0.08, y: cur.height * 0.08, w: cur.width * 0.84, h: cur.height * 0.84 } : null; lasso = null;
      m.querySelectorAll("[data-m]").forEach((b) => b.classList.toggle("on", b.dataset.m === k));
      hint.textContent = HINTS[k] || "";
      view.style.cursor = k === "crop" ? "move" : k === "erase" || k === "restore" ? "none" : "crosshair";
      showApply(); draw();
    }
    function showApply() {
      if (mode === "crop") applyBar.innerHTML = `<button type="button" class="btn btn-gold btn-sm" data-ie="crop">${bi("check", 16)}Тайрах</button>`;
      else if (mode === "lasso" && lasso?.length > 2 && !dragging) applyBar.innerHTML = `<button type="button" class="btn btn-gold btn-sm" data-ie="keep">Доторхыг үлдээх</button><button type="button" class="btn btn-glass btn-sm" data-ie="cut">Доторхыг хасах</button><button type="button" class="btn btn-ghost btn-sm" data-ie="relasso">Дахин зурах</button>`;
      else { applyBar.hidden = true; return; }
      applyBar.hidden = false;
    }
    /* дэвсгэр арилгах: өнгөний ойролцоо байдлаар үерлэх (flood fill) */
    function flood(seeds, single) {
      const W = cur.width, H = cur.height, id = cur.ctx.getImageData(0, 0, W, H), d = id.data, n = W * H;
      const T = (tol * 4.4) ** 2, rm = new Uint8Array(n), qq = new Int32Array(n);
      let removed = 0;
      for (const s0i of seeds) {
        if (rm[s0i]) continue;
        const j0 = s0i * 4, r0 = d[j0], g0 = d[j0 + 1], b0 = d[j0 + 2], clear0 = d[j0 + 3] < 16;
        const near = (p) => { const j = p * 4; if (d[j + 3] < 16) return true; if (clear0) return false; const dr = d[j] - r0, dg = d[j + 1] - g0, db = d[j + 2] - b0; return dr * dr + dg * dg + db * db <= T; };
        if (!near(s0i)) continue;
        let h = 0, t = 0; qq[t++] = s0i; rm[s0i] = 1;
        while (h < t) {
          const p = qq[h++], x = p % W;
          if (x > 0 && !rm[p - 1] && near(p - 1)) { rm[p - 1] = 1; qq[t++] = p - 1; }
          if (x < W - 1 && !rm[p + 1] && near(p + 1)) { rm[p + 1] = 1; qq[t++] = p + 1; }
          if (p >= W && !rm[p - W] && near(p - W)) { rm[p - W] = 1; qq[t++] = p - W; }
          if (p < n - W && !rm[p + W] && near(p + W)) { rm[p + W] = 1; qq[t++] = p + W; }
        }
        removed += t;
        if (single) break;
      }
      if (!removed) return 0;
      for (let p = 0; p < n; p++) if (rm[p]) d[p * 4 + 3] = 0;
      for (let p = 0; p < n; p++) { // зөөлөн ирмэг
        if (rm[p] || !d[p * 4 + 3]) continue;
        const x = p % W;
        if ((x > 0 && rm[p - 1]) || (x < W - 1 && rm[p + 1]) || (p >= W && rm[p - W]) || (p < n - W && rm[p + W])) d[p * 4 + 3] = Math.round(d[p * 4 + 3] * 0.5);
      }
      cur.ctx.putImageData(id, 0, 0);
      return removed / n;
    }
    function autoBg() {
      push();
      const W = cur.width, H = cur.height, d = cur.ctx.getImageData(0, 0, W, H).data, border = [];
      for (let x = 0; x < W; x++) border.push(x, (H - 1) * W + x);
      for (let y = 1; y < H - 1; y++) border.push(y * W, y * W + W - 1);
      // Ирмэгийн давамгай өнгө л дэвсгэр: ирмэгт хүрсэн биетийг (модны иш г.м) хамт арилгахгүй.
      const step = Math.max(1, Math.floor(border.length / 600)), samp = [], T = (tol * 4.4) ** 2;
      for (let i = 0; i < border.length; i += step) samp.push(border[i]);
      const need = Math.max(3, samp.length * 0.06);
      const like = (p) => {
        const j = p * 4; if (d[j + 3] < 16) return true;
        let n = 0;
        for (const s of samp) { const k = s * 4; if (d[k + 3] < 16) continue; const dr = d[j] - d[k], dg = d[j + 1] - d[k + 1], db = d[j + 2] - d[k + 2]; if (dr * dr + dg * dg + db * db <= T && ++n >= need) return true; }
        return false;
      };
      const f = flood(border.filter(like), false);
      if (f > 0.97 || f === 0) { undoOne(); toast(f ? "Дэвсгэрийг ялгаж чадсангүй — мэдрэмжийг багасгаж үзэх эсвэл «Дарж арилгах» хэрэгслийг ашиглана уу" : "Ирмэгт нэг өнгийн дэвсгэр олдсонгүй — «Дарж арилгах» эсвэл лассо ашиглана уу", true); return; }
      setMode("wand"); hint.textContent = HINTS.auto; draw();
    }
    function undoOne() { const h = hist.pop(); if (!h) return; cur = h.cur; orig = h.orig; offX = h.offX; offY = h.offY; dirty = hist.length > 0; syncBtns(); setMode(mode === "crop" ? "wand" : mode); }
    function cropTo(r) {
      const x = Math.round(clamp(r.x, 0, cur.width - 1)), y = Math.round(clamp(r.y, 0, cur.height - 1));
      const w = Math.round(clamp(r.w, 4, cur.width - x)), h = Math.round(clamp(r.h, 4, cur.height - y));
      push();
      const c2 = mk(w, h); c2.ctx.drawImage(cur, -x, -y); const o2 = mk(w, h); o2.ctx.drawImage(orig, -x, -y);
      cur = c2; orig = o2; offX += x; offY += y; setMode("wand");
    }
    function lassoApply(keep) {
      if (!lasso || lasso.length < 3) return;
      push();
      const p = new Path2D(); lasso.forEach((pt, i) => (i ? p.lineTo(pt.x, pt.y) : p.moveTo(pt.x, pt.y))); p.closePath();
      cur.ctx.save(); cur.ctx.globalCompositeOperation = keep ? "destination-in" : "destination-out"; cur.ctx.fill(p); cur.ctx.restore();
      lasso = null; showApply(); draw();
    }
    function paint(a, b) {
      const w = brush / sx;
      if (mode === "erase") {
        const c = cur.ctx; c.save(); c.globalCompositeOperation = "destination-out"; c.lineCap = "round"; c.lineWidth = w; c.beginPath(); c.moveTo(a.x, a.y); c.lineTo(b.x, b.y); c.stroke(); c.restore();
      } else {
        const x = Math.floor(Math.min(a.x, b.x) - w), y = Math.floor(Math.min(a.y, b.y) - w), W = Math.ceil(Math.abs(a.x - b.x) + w * 2), H = Math.ceil(Math.abs(a.y - b.y) + w * 2);
        const t = mk(W, H); t.ctx.drawImage(orig, -x, -y); t.ctx.globalCompositeOperation = "destination-in"; t.ctx.lineCap = "round"; t.ctx.lineWidth = w;
        t.ctx.beginPath(); t.ctx.moveTo(a.x - x, a.y - y); t.ctx.lineTo(b.x - x, b.y - y); t.ctx.stroke();
        cur.ctx.drawImage(t, x, y);
      }
      draw();
    }
    view.addEventListener("pointerdown", (e) => {
      if (!cur) return; const p = toImg(e); view.setPointerCapture(e.pointerId);
      if (mode === "crop") {
        const r = toV(crop), hi = handles(r).findIndex(([hx, hy]) => Math.hypot(hx - p.vx, hy - p.vy) < 14);
        dragging = { kind: hi >= 0 ? HN[hi] : (p.vx > r.x && p.vx < r.x + r.w && p.vy > r.y && p.vy < r.y + r.h ? "move" : ""), p, start: { ...crop } };
      } else if (mode === "lasso") { dragging = { kind: "lasso" }; lasso = [p]; applyBar.hidden = true; }
      else if (mode === "wand") {
        const x = Math.floor(p.x), y = Math.floor(p.y); if (x < 0 || y < 0 || x >= cur.width || y >= cur.height) return;
        push(); if (!flood([y * cur.width + x], true)) { hist.pop(); dirty = hist.length > 0; syncBtns(); } draw();
      } else if (mode === "erase" || mode === "restore") { push(); dragging = { kind: "paint", last: p }; paint(p, p); }
    });
    view.addEventListener("pointermove", (e) => {
      if (!cur) return; const p = toImg(e); hover = p;
      if (!dragging) { if (mode === "erase" || mode === "restore") draw(); return; }
      if (dragging.kind === "lasso") { const l = lasso[lasso.length - 1]; if (Math.hypot(l.x - p.x, l.y - p.y) * sx > 3) { lasso.push(p); draw(); } return; }
      if (dragging.kind === "paint") { paint(dragging.last, p); dragging.last = p; return; }
      if (mode === "crop" && dragging.kind) {
        const s = dragging.start, dx = p.x - dragging.p.x, dy = p.y - dragging.p.y, k = dragging.kind, W = cur.width, H = cur.height;
        let { x, y, w, h } = s;
        if (k === "move") { x = clamp(s.x + dx, 0, W - s.w); y = clamp(s.y + dy, 0, H - s.h); }
        else {
          if (k.includes("w")) { x = clamp(s.x + dx, 0, s.x + s.w - 8); w = s.x + s.w - x; }
          if (k.includes("e")) w = clamp(s.w + dx, 8, W - s.x);
          if (k.includes("n")) { y = clamp(s.y + dy, 0, s.y + s.h - 8); h = s.y + s.h - y; }
          if (k.includes("s")) h = clamp(s.h + dy, 8, H - s.y);
        }
        crop = { x, y, w, h }; draw();
      }
    });
    const up = () => { if (dragging?.kind === "lasso") { dragging = null; if (lasso.length < 3) lasso = null; showApply(); draw(); return; } dragging = null; };
    view.addEventListener("pointerup", up); view.addEventListener("pointercancel", up);
    view.addEventListener("pointerleave", () => { hover = null; draw(); });
    m.querySelectorAll("input[type=range]").forEach((r) => r.addEventListener("input", () => {
      r.previousElementSibling.textContent = r.value;
      if (r.dataset.r === "tol") tol = +r.value; else { brush = +r.value; draw(); }
    }));
    m.addEventListener("click", (e) => {
      const b = e.target.closest("button"); if (!b) return;
      if (b.dataset.m) { if (b.dataset.m === "auto") autoBg(); else setMode(b.dataset.m); return; }
      const a = b.dataset.ie;
      if (a === "cancel") return done(null);
      if (a === "undo") return undoOne();
      if (a === "reset") { if (!hist.length) return; const h0 = hist[0]; hist = []; cur = h0.cur; orig = h0.orig; offX = h0.offX; offY = h0.offY; dirty = false; syncBtns(); setMode("wand"); return; }
      if (a === "crop") return cropTo(crop);
      if (a === "keep") return lassoApply(true);
      if (a === "cut") return lassoApply(false);
      if (a === "relasso") { lasso = null; showApply(); draw(); return; }
      if (a === "save") return save();
    });
    const onKey = (e) => { if (e.key === "Escape") { e.stopPropagation(); done(null); } else if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "z") { e.preventDefault(); undoOne(); } };
    document.addEventListener("keydown", onKey, true);
    const ro = new ResizeObserver(size); ro.observe(stageEl);
    async function save() {
      if (!dirty) return done(null);
      const W = cur.width, H = cur.height, d = cur.ctx.getImageData(0, 0, W, H).data;
      let x1 = W, y1 = H, x2 = -1, y2 = -1;
      for (let y = 0; y < H; y++) for (let x = 0; x < W; x++) if (d[(y * W + x) * 4 + 3] > 8) { if (x < x1) x1 = x; if (x > x2) x2 = x; if (y < y1) y1 = y; if (y > y2) y2 = y; }
      if (x2 < 0) return toast("Зураг бүхэлдээ арилсан байна — «Буцаах» дарна уу", true);
      const out = mk(x2 - x1 + 1, y2 - y1 + 1); out.ctx.drawImage(cur, -x1, -y1);
      const blob = await new Promise((res) => out.toBlob((b) => (b && b.type === "image/webp" ? res(b) : out.toBlob(res, "image/png")), "image/webp", 0.92));
      done({ blob, x: (offX + x1) / s0, y: (offY + y1) / s0, w: out.width / s0, h: out.height / s0 });
    }
    function done(v) { document.removeEventListener("keydown", onKey, true); ro.disconnect(); m.remove(); resolve(v); }
  });
}
})();
