// Pure TypeScript RFC 6238 TOTP, RFC 4226 HOTP, Steam Guard & Migration Engine
// 100% offline, zero external dependencies, WebCrypto hardware accelerated.

export interface ParsedOTP {
  type: 'totp' | 'hotp' | 'steam';
  issuer: string;
  account: string;
  secret: string;
  algorithm: 'SHA1' | 'SHA256' | 'SHA512';
  digits: number;
  period: number;
  counter: number;
}

const BASE32_ALPHABET = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';
const STEAM_ALPHABET = '23456789BCDFGHJKMNPQRTVWXY';

/**
 * Clean and decode RFC 4648 Base32 string into Uint8Array
 */
export function base32Decode(str: string): Uint8Array {
  const clean = str.toUpperCase().replace(/[\s\-_=]/g, '');
  if (!clean) return new Uint8Array(0);

  let bits = 0;
  let value = 0;
  const output: number[] = [];

  for (let i = 0; i < clean.length; i++) {
    const val = BASE32_ALPHABET.indexOf(clean[i]);
    if (val === -1) continue; // skip unrecognized chars

    value = (value << 5) | val;
    bits += 5;

    if (bits >= 8) {
      output.push((value >>> (bits - 8)) & 0xff);
      bits -= 8;
    }
  }

  return new Uint8Array(output);
}

/**
 * Encode Uint8Array to RFC 4648 Base32 string
 */
export function base32Encode(bytes: Uint8Array): string {
  let result = '';
  let bits = 0;
  let value = 0;

  for (let i = 0; i < bytes.length; i++) {
    value = (value << 8) | bytes[i];
    bits += 8;

    while (bits >= 5) {
      result += BASE32_ALPHABET[(value >>> (bits - 5)) & 31];
      bits -= 5;
    }
  }

  if (bits > 0) {
    result += BASE32_ALPHABET[(value << (5 - bits)) & 31];
  }

  return result;
}

function getWebCryptoHashName(algo: string): string {
  const upper = (algo || 'SHA1').toUpperCase();
  if (upper === 'SHA256') return 'SHA-256';
  if (upper === 'SHA512') return 'SHA-512';
  return 'SHA-1';
}

const getSubtleCrypto = () => {
  return (typeof window !== 'undefined' ? window.crypto : globalThis.crypto).subtle;
};

/**
 * Compute HMAC digest using Web Crypto API
 */
async function computeHMAC(secretBytes: Uint8Array, counter: bigint, algorithm: string): Promise<Uint8Array> {
  const hashName = getWebCryptoHashName(algorithm);
  const subtle = getSubtleCrypto();

  const key = await subtle.importKey(
    'raw',
    secretBytes,
    { name: 'HMAC', hash: { name: hashName } },
    false,
    ['sign']
  );

  // 8-byte big-endian uint64 buffer
  const counterBuffer = new ArrayBuffer(8);
  const dataView = new DataView(counterBuffer);
  dataView.setBigUint64(0, counter, false);

  const signature = await subtle.sign('HMAC', key, counterBuffer);
  return new Uint8Array(signature);
}

/**
 * Generate standard RFC 6238 TOTP code
 */
export async function generateTOTP(
  secret: string,
  period = 30,
  digits = 6,
  algorithm = 'SHA1',
  timestamp = Date.now()
): Promise<string> {
  const secretBytes = base32Decode(secret);
  if (secretBytes.length === 0) return '------';

  const step = Math.floor(timestamp / 1000 / (period || 30));
  const counter = BigInt(step);

  const hmac = await computeHMAC(secretBytes, counter, algorithm);

  // Dynamic truncation
  const offset = hmac[hmac.length - 1] & 0x0f;
  const binary =
    ((hmac[offset] & 0x7f) << 24) |
    ((hmac[offset + 1] & 0xff) << 16) |
    ((hmac[offset + 2] & 0xff) << 8) |
    (hmac[offset + 3] & 0xff);

  const mod = Math.pow(10, digits || 6);
  const otp = binary % mod;
  return otp.toString().padStart(digits || 6, '0');
}

/**
 * Generate Next Cycle TOTP code (timestamp + period)
 */
export async function generateNextTOTP(
  secret: string,
  period = 30,
  digits = 6,
  algorithm = 'SHA1',
  timestamp = Date.now()
): Promise<string> {
  return generateTOTP(secret, period, digits, algorithm, timestamp + (period || 30) * 1000);
}

/**
 * Generate Steam Guard 5-character alphanumeric OTP code
 */
export async function generateSteamOTP(secret: string, timestamp = Date.now()): Promise<string> {
  const secretBytes = base32Decode(secret);
  if (secretBytes.length === 0) return '-----';

  const step = Math.floor(timestamp / 1000 / 30);
  const counter = BigInt(step);

  const hmac = await computeHMAC(secretBytes, counter, 'SHA1');

  const offset = hmac[hmac.length - 1] & 0x0f;
  let fullCode =
    ((hmac[offset] & 0x7f) << 24) |
    ((hmac[offset + 1] & 0xff) << 16) |
    ((hmac[offset + 2] & 0xff) << 8) |
    (hmac[offset + 3] & 0xff);

  let code = '';
  for (let i = 0; i < 5; i++) {
    code += STEAM_ALPHABET[fullCode % STEAM_ALPHABET.length];
    fullCode = Math.floor(fullCode / STEAM_ALPHABET.length);
  }

  return code;
}

/**
 * Generate Next Cycle Steam Guard code
 */
export async function generateNextSteamOTP(secret: string, timestamp = Date.now()): Promise<string> {
  return generateSteamOTP(secret, timestamp + 30000);
}

/**
 * Generate RFC 4226 HOTP code
 */
export async function generateHOTP(secret: string, counter: number, digits = 6, algorithm = 'SHA1'): Promise<string> {
  const secretBytes = base32Decode(secret);
  if (secretBytes.length === 0) return '------';

  const hmac = await computeHMAC(secretBytes, BigInt(counter), algorithm);

  const offset = hmac[hmac.length - 1] & 0x0f;
  const binary =
    ((hmac[offset] & 0x7f) << 24) |
    ((hmac[offset + 1] & 0xff) << 16) |
    ((hmac[offset + 2] & 0xff) << 8) |
    (hmac[offset + 3] & 0xff);

  const mod = Math.pow(10, digits || 6);
  const otp = binary % mod;
  return otp.toString().padStart(digits || 6, '0');
}

/**
 * Parse standard otpauth:// URI
 */
export function parseOTPAuthURI(uriString: string): ParsedOTP {
  const uri = new URL(uriString.trim());
  if (uri.protocol !== 'otpauth:') {
    throw new Error('无效的 2FA 协议头，必须为 otpauth://');
  }

  const typeStr = uri.host.toLowerCase();
  let type: 'totp' | 'hotp' | 'steam' = 'totp';
  if (typeStr === 'hotp') type = 'hotp';
  else if (typeStr === 'steam') type = 'steam';

  let rawPath = decodeURIComponent(uri.pathname.replace(/^\//, ''));
  let issuer = uri.searchParams.get('issuer') || '';
  let account = rawPath;

  if (rawPath.includes(':')) {
    const parts = rawPath.split(':');
    if (!issuer) issuer = parts[0].trim();
    account = parts.slice(1).join(':').trim();
  }

  const secret = (uri.searchParams.get('secret') || '').toUpperCase().replace(/\s+/g, '');
  if (!secret) throw new Error('缺少有效的 2FA Secret 密钥');

  let algorithm: 'SHA1' | 'SHA256' | 'SHA512' = 'SHA1';
  const algoParam = (uri.searchParams.get('algorithm') || 'SHA1').toUpperCase();
  if (algoParam === 'SHA256') algorithm = 'SHA256';
  else if (algoParam === 'SHA512') algorithm = 'SHA512';

  const digits = parseInt(uri.searchParams.get('digits') || '6', 10);
  const period = parseInt(uri.searchParams.get('period') || '30', 10);
  const counter = parseInt(uri.searchParams.get('counter') || '0', 10);

  return {
    type,
    issuer: issuer || account || '未知服务商',
    account: account || '默认账号',
    secret,
    algorithm,
    digits: isNaN(digits) ? 6 : digits,
    period: isNaN(period) ? 30 : period,
    counter: isNaN(counter) ? 0 : counter,
  };
}

/**
 * Build standard otpauth:// URI string
 */
export function buildOTPAuthURI(account: {
  type?: string;
  issuer?: string;
  account_name?: string;
  secret: string;
  algorithm?: string;
  digits?: number;
  period?: number;
  counter?: number;
}): string {
  const type = account.type || 'totp';
  const label = account.issuer
    ? `${encodeURIComponent(account.issuer)}:${encodeURIComponent(account.account_name || '')}`
    : encodeURIComponent(account.account_name || 'Account');

  const params = new URLSearchParams();
  params.set('secret', account.secret);
  if (account.issuer) params.set('issuer', account.issuer);
  if (account.algorithm && account.algorithm !== 'SHA1') params.set('algorithm', account.algorithm);
  if (account.digits && account.digits !== 6) params.set('digits', account.digits.toString());
  if (account.period && account.period !== 30) params.set('period', account.period.toString());
  if (type === 'hotp') params.set('counter', (account.counter || 0).toString());

  return `otpauth://${type}/${label}?${params.toString()}`;
}

/**
 * Pure TypeScript Google Authenticator Migration Payload Parser
 * Parses `otpauth-migration://offline?data=...` without external protobuf library
 */
export function parseGoogleMigrationURI(uriString: string): ParsedOTP[] {
  const uri = new URL(uriString.trim());
  if (uri.protocol !== 'otpauth-migration:') {
    throw new Error('无效的 Google 迁移协议头，必须为 otpauth-migration://');
  }

  let dataParam = uri.searchParams.get('data') || '';
  if (!dataParam) throw new Error('缺少 data 参数');

  // URL-safe base64 normalization
  dataParam = dataParam.replace(/-/g, '+').replace(/_/g, '/');
  while (dataParam.length % 4 !== 0) dataParam += '=';

  const binaryString = atob(dataParam);
  const bytes = new Uint8Array(binaryString.length);
  for (let i = 0; i < binaryString.length; i++) {
    bytes[i] = binaryString.charCodeAt(i);
  }

  // Parse protobuf MigrationPayload
  let offset = 0;
  const accounts: ParsedOTP[] = [];

  while (offset < bytes.length) {
    const [tag, wireType, newOffset] = readVarint(bytes, offset);
    offset = newOffset;
    const fieldNum = tag >>> 3;

    if (fieldNum === 1 && wireType === 2) {
      // repeated OtpParameters otp_parameters = 1
      const [len, _, lenOffset] = readVarint(bytes, offset);
      offset = lenOffset;
      const paramBytes = bytes.subarray(offset, offset + Number(len));
      offset += Number(len);

      const parsed = parseOtpParameters(paramBytes);
      if (parsed) accounts.push(parsed);
    } else {
      offset = skipField(bytes, offset, wireType);
    }
  }

  return accounts;
}

function parseOtpParameters(bytes: Uint8Array): ParsedOTP | null {
  let offset = 0;
  let secretBytes: Uint8Array = new Uint8Array(0);
  let name = '';
  let issuer = '';
  let algorithm: 'SHA1' | 'SHA256' | 'SHA512' = 'SHA1';
  let digits = 6;
  let type: 'totp' | 'hotp' | 'steam' = 'totp';
  let counter = 0;

  while (offset < bytes.length) {
    const [tag, wireType, newOffset] = readVarint(bytes, offset);
    offset = newOffset;
    const fieldNum = tag >>> 3;

    if (fieldNum === 1 && wireType === 2) {
      // secret bytes
      const [len, _, lOff] = readVarint(bytes, offset);
      offset = lOff;
      secretBytes = bytes.subarray(offset, offset + Number(len));
      offset += Number(len);
    } else if (fieldNum === 2 && wireType === 2) {
      // name string
      const [len, _, lOff] = readVarint(bytes, offset);
      offset = lOff;
      name = new TextDecoder().decode(bytes.subarray(offset, offset + Number(len)));
      offset += Number(len);
    } else if (fieldNum === 3 && wireType === 2) {
      // issuer string
      const [len, _, lOff] = readVarint(bytes, offset);
      offset = lOff;
      issuer = new TextDecoder().decode(bytes.subarray(offset, offset + Number(len)));
      offset += Number(len);
    } else if (fieldNum === 4 && wireType === 0) {
      // algorithm enum: 1=SHA1, 2=SHA256, 3=SHA512
      const [val, _, vOff] = readVarint(bytes, offset);
      offset = vOff;
      if (val === 2n) algorithm = 'SHA256';
      else if (val === 3n) algorithm = 'SHA512';
      else algorithm = 'SHA1';
    } else if (fieldNum === 5 && wireType === 0) {
      // digits enum: 1=6, 2=8
      const [val, _, vOff] = readVarint(bytes, offset);
      offset = vOff;
      digits = val === 2n ? 8 : 6;
    } else if (fieldNum === 6 && wireType === 0) {
      // type enum: 1=HOTP, 2=TOTP
      const [val, _, vOff] = readVarint(bytes, offset);
      offset = vOff;
      type = val === 1n ? 'hotp' : 'totp';
    } else if (fieldNum === 7 && wireType === 0) {
      // counter
      const [val, _, vOff] = readVarint(bytes, offset);
      offset = vOff;
      counter = Number(val);
    } else {
      offset = skipField(bytes, offset, wireType);
    }
  }

  if (secretBytes.length === 0) return null;

  const base32Secret = base32Encode(secretBytes);
  let accountName = name;
  if (name.includes(':') && !issuer) {
    const parts = name.split(':');
    issuer = parts[0].trim();
    accountName = parts.slice(1).join(':').trim();
  }

  return {
    type,
    issuer: issuer || accountName || 'Google Authenticator',
    account: accountName || '默认账号',
    secret: base32Secret,
    algorithm,
    digits,
    period: 30,
    counter,
  };
}

function readVarint(bytes: Uint8Array, offset: number): [bigint, number, number] {
  let res = 0n;
  let shift = 0n;
  let pos = offset;

  while (pos < bytes.length) {
    const b = bytes[pos++];
    res |= BigInt(b & 0x7f) << shift;
    if ((b & 0x80) === 0) {
      const tag = Number(res);
      const wireType = tag & 0x07;
      return [res, wireType, pos];
    }
    shift += 7n;
  }
  return [res, 0, pos];
}

function skipField(bytes: Uint8Array, offset: number, wireType: number): number {
  if (wireType === 0) {
    // Varint
    let pos = offset;
    while (pos < bytes.length && (bytes[pos++] & 0x80) !== 0) {}
    return pos;
  }
  if (wireType === 1) return offset + 8; // 64-bit
  if (wireType === 2) {
    // Length-delimited
    const [len, _, pos] = readVarint(bytes, offset);
    return pos + Number(len);
  }
  if (wireType === 5) return offset + 4; // 32-bit
  return offset;
}
