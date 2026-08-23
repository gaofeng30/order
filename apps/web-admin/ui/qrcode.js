/* Minimal ISO/IEC 18004 QR encoder for the PC login challenge.
   The login payload is ASCII and bounded to QR version 8-L (192 bytes).
   No payload leaves the browser and no remote QR service is used. */
(function () {
  const VERSION = 8;
  const MODULES = VERSION * 4 + 17;
  const DATA_CODEWORDS = 194;
  const BLOCK_DATA = 97;
  const ECC_CODEWORDS = 24;
  const ALIGNMENT = [6, 24, 42];
  const G15 = 0x537, G18 = 0x1f25, G15_MASK = 0x5412;

  const EXP = new Uint8Array(512);
  const LOG = new Uint8Array(256);
  let value = 1;
  for (let i = 0; i < 255; i++) {
    EXP[i] = value;
    LOG[value] = i;
    value <<= 1;
    if (value & 0x100) value ^= 0x11d;
  }
  for (let i = 255; i < EXP.length; i++) EXP[i] = EXP[i - 255];

  function multiply(a, b) { return a && b ? EXP[LOG[a] + LOG[b]] : 0; }
  function generator(degree) {
    let out = [1];
    for (let i = 0; i < degree; i++) {
      const next = new Array(out.length + 1).fill(0);
      for (let j = 0; j < out.length; j++) {
        next[j] ^= out[j];
        next[j + 1] ^= multiply(out[j], EXP[i]);
      }
      out = next;
    }
    return out;
  }
  const GENERATOR = generator(ECC_CODEWORDS);

  function ecc(data) {
    const result = new Array(ECC_CODEWORDS).fill(0);
    for (const byte of data) {
      const factor = byte ^ result[0];
      result.shift();
      result.push(0);
      for (let i = 0; i < ECC_CODEWORDS; i++) result[i] ^= multiply(GENERATOR[i + 1], factor);
    }
    return result;
  }

  function appendBits(target, data, length) {
    for (let i = length - 1; i >= 0; i--) target.push((data >>> i) & 1);
  }
  function codewords(text) {
    const bytes = Array.from(new TextEncoder().encode(text));
    if (!bytes.length || bytes.length > 192) throw new Error('PC 登录二维码载荷长度无效');
    const bits = [];
    appendBits(bits, 0x4, 4); // byte mode
    appendBits(bits, bytes.length, 8);
    bytes.forEach(byte => appendBits(bits, byte, 8));
    appendBits(bits, 0, Math.min(4, DATA_CODEWORDS * 8 - bits.length));
    while (bits.length % 8) bits.push(0);
    const data = [];
    for (let i = 0; i < bits.length; i += 8) {
      let byte = 0;
      for (let j = 0; j < 8; j++) byte = (byte << 1) | bits[i + j];
      data.push(byte);
    }
    for (let pad = 0; data.length < DATA_CODEWORDS; pad++) data.push(pad % 2 ? 0x11 : 0xec);
    const blocks = [data.slice(0, BLOCK_DATA), data.slice(BLOCK_DATA)];
    const checks = blocks.map(ecc);
    const out = [];
    for (let i = 0; i < BLOCK_DATA; i++) blocks.forEach(block => out.push(block[i]));
    for (let i = 0; i < ECC_CODEWORDS; i++) checks.forEach(block => out.push(block[i]));
    return out;
  }

  function bchDigit(data) {
    let digit = 0;
    while (data) { digit++; data >>>= 1; }
    return digit;
  }
  function bchTypeInfo(data) {
    let value = data << 10;
    while (bchDigit(value) - bchDigit(G15) >= 0) value ^= G15 << (bchDigit(value) - bchDigit(G15));
    return ((data << 10) | value) ^ G15_MASK;
  }
  function bchVersion(data) {
    let value = data << 12;
    while (bchDigit(value) - bchDigit(G18) >= 0) value ^= G18 << (bchDigit(value) - bchDigit(G18));
    return (data << 12) | value;
  }

  function probe(matrix, row, col) {
    for (let r = -1; r <= 7; r++) for (let c = -1; c <= 7; c++) {
      const y = row + r, x = col + c;
      if (y < 0 || y >= MODULES || x < 0 || x >= MODULES) continue;
      matrix[y][x] = r >= 0 && r <= 6 && c >= 0 && c <= 6 &&
        (r === 0 || r === 6 || c === 0 || c === 6 || (r >= 2 && r <= 4 && c >= 2 && c <= 4));
    }
  }
  function setup(matrix) {
    probe(matrix, 0, 0);
    probe(matrix, MODULES - 7, 0);
    probe(matrix, 0, MODULES - 7);
    for (const row of ALIGNMENT) for (const col of ALIGNMENT) {
      if (matrix[row][col] !== null) continue;
      for (let r = -2; r <= 2; r++) for (let c = -2; c <= 2; c++)
        matrix[row + r][col + c] = Math.max(Math.abs(r), Math.abs(c)) !== 1;
    }
    for (let i = 8; i < MODULES - 8; i++) {
      if (matrix[i][6] === null) matrix[i][6] = i % 2 === 0;
      if (matrix[6][i] === null) matrix[6][i] = i % 2 === 0;
    }
    const versionBits = bchVersion(VERSION);
    for (let i = 0; i < 18; i++) {
      const dark = ((versionBits >>> i) & 1) === 1;
      matrix[Math.floor(i / 3)][i % 3 + MODULES - 11] = dark;
      matrix[i % 3 + MODULES - 11][Math.floor(i / 3)] = dark;
    }
    const formatBits = bchTypeInfo(0x8); // error correction L, mask 0
    for (let i = 0; i < 15; i++) {
      const dark = ((formatBits >>> i) & 1) === 1;
      if (i < 6) matrix[i][8] = dark;
      else if (i < 8) matrix[i + 1][8] = dark;
      else matrix[MODULES - 15 + i][8] = dark;
      if (i < 8) matrix[8][MODULES - i - 1] = dark;
      else if (i < 9) matrix[8][15 - i] = dark;
      else matrix[8][15 - i - 1] = dark;
    }
    matrix[MODULES - 8][8] = true;
  }

  function matrix(text) {
    const out = Array.from({ length: MODULES }, () => new Array(MODULES).fill(null));
    setup(out);
    const data = codewords(text);
    let row = MODULES - 1, direction = -1, byteIndex = 0, bitIndex = 7;
    for (let col = MODULES - 1; col > 0; col -= 2) {
      if (col === 6) col--;
      for (;;) {
        for (let offset = 0; offset < 2; offset++) {
          const x = col - offset;
          if (out[row][x] !== null) continue;
          let dark = byteIndex < data.length && ((data[byteIndex] >>> bitIndex) & 1) === 1;
          if ((row + x) % 2 === 0) dark = !dark;
          out[row][x] = dark;
          if (--bitIndex < 0) { byteIndex++; bitIndex = 7; }
        }
        row += direction;
        if (row >= 0 && row < MODULES) continue;
        row -= direction;
        direction = -direction;
        break;
      }
    }
    if (byteIndex !== data.length) throw new Error('PC 登录二维码编码失败');
    return out;
  }

  function render(canvas, text, size) {
    if (!canvas || typeof canvas.getContext !== 'function') throw new Error('PC 登录二维码画布不可用');
    const modules = matrix(String(text || ''));
    const quiet = 4;
    const pixels = Number(size) || 228;
    const scale = Math.max(1, Math.floor(pixels / (MODULES + quiet * 2)));
    const width = (MODULES + quiet * 2) * scale;
    canvas.width = width;
    canvas.height = width;
    canvas.style.width = width + 'px';
    canvas.style.height = width + 'px';
    const ctx = canvas.getContext('2d');
    ctx.imageSmoothingEnabled = false;
    ctx.fillStyle = '#fff';
    ctx.fillRect(0, 0, width, width);
    ctx.fillStyle = '#111';
    for (let row = 0; row < MODULES; row++) for (let col = 0; col < MODULES; col++)
      if (modules[row][col]) ctx.fillRect((col + quiet) * scale, (row + quiet) * scale, scale, scale);
    return { modules, width };
  }

  window.PCQRCode = { render, matrix };
})();
