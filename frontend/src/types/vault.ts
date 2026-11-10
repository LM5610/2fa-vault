export type OTPType = 'totp' | 'hotp' | 'steam';
export type OTPAlgorithm = 'SHA1' | 'SHA256' | 'SHA512';

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
  code?: string;
  remaining?: number;
  progress?: number;
  created_at?: string;
  updated_at?: string;
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
