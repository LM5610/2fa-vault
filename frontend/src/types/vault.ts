export type OTPType = 'totp' | 'hotp' | 'steam';
export type OTPAlgorithm = 'SHA1' | 'SHA256' | 'SHA512';

export interface RecoveryCode {
  code: string;
  used: boolean;
}

export interface Account {
  id: string;
  issuer: string;
  account_name: string;
  secret: string;
  type: OTPType;
  digits: number;
  period: number;
  algorithm: OTPAlgorithm;
  counter?: number;
  tags?: string[];
  notes?: string;
  pinned?: boolean;
  order?: number;
  recovery_codes?: RecoveryCode[];
  code?: string;
  next_code?: string;
  remaining?: number;
  progress?: number;
  created_at?: string;
  updated_at?: string;
}

export interface TimeDriftResponse {
  system_time: string;
  network_time: string;
  drift_seconds: number;
  is_drifted: boolean;
  source?: string;
  error?: string;
}

export interface SyncInfoResponse {
  lan_ips: string[];
}

export interface SyncPairInitResponse {
  token: string;
  expires_in: number;
  pairing_url: string;
  lan_ips: string[];
}

export interface VaultStatus {
  initialized: boolean;
  locked: boolean;
  auto_lock_minutes: number;
  account_count?: number;
}

export interface OTPPreviewRequest {
  secret: string;
  digits?: number;
  period?: number;
  algorithm?: string;
  type?: string;
  counter?: number;
}

export interface OTPPreviewResponse {
  valid: boolean;
  code?: string;
  formatted_code?: string;
  remaining?: number;
  progress?: number;
  period?: number;
  error?: string;
}

export interface QRDecodeResponse {
  text: string;
  parsed?: {
    issuer?: string;
    account_name?: string;
    secret?: string;
    type?: OTPType;
    digits?: number;
    period?: number;
    algorithm?: OTPAlgorithm;
    counter?: number;
  };
}

export interface SnapshotInfo {
  filename: string;
  size_bytes: number;
  created_at: string;
}

export interface WebDAVConfig {
  url: string;
  username: string;
  password?: string;
  has_password?: boolean;
  auto_sync: boolean;
  remote_dir: string;
  last_sync?: string;
  last_status?: string;
  last_error?: string;
}
