import { env } from '$env/dynamic/public';

export type ApiError = {
  error: {
    code: string;
    message: string;
    details?: unknown;
    requestId?: string;
  };
};

export type Format = {
  formatId: string;
  kind: 'audio' | 'video' | 'muxed';
  height?: number;
  fps?: number;
  container?: string;
  codec?: string;
  estimatedBytes?: number;
};

export type Analysis = {
  videoId: string;
  canonicalUrl: string;
  title: string;
  durationSeconds: number;
  thumbnailUrl?: string;
  formats: Format[];
};

export type JobCreated = {
  jobId: string;
  state: string;
  eventsUrl: string;
  downloadUrl: string | null;
  expiresAt: string | null;
};

export type JobEvent = {
  jobId: string;
  state: 'queued' | 'processing' | 'completed' | 'failed' | 'expired';
  progress: number;
  message: string;
  downloadUrl?: string;
  expiresAt?: string;
  error?: ApiError['error'] | null;
};

export async function analyze(url: string): Promise<Analysis> {
  return postJSON(apiURL('/api/analyze'), { url });
}

export async function createJob(videoId: string, videoFormatId: string, audioFormatId?: string): Promise<JobCreated> {
  return postJSON(apiURL('/api/jobs'), { videoId, format: { videoFormatId, audioFormatId } });
}

export function apiURL(path: string): string {
  const base = (env.PUBLIC_API_BASE_URL ?? '').trim().replace(/\/$/, '');
  return base ? `${base}${path}` : path;
}

async function postJSON<T>(url: string, body: unknown): Promise<T> {
  const res = await fetch(url, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  });
  if (!res.ok) {
    throw await toApiError(res);
  }
  return res.json();
}

async function toApiError(res: Response): Promise<ApiError> {
  try {
    return (await res.json()) as ApiError;
  } catch {
    return { error: { code: 'request_failed', message: `Request failed with status ${res.status}.` } };
  }
}

export function readableBytes(value?: number): string {
  if (!value) return 'Unknown size';
  const units = ['B', 'KB', 'MB', 'GB'];
  let n = value;
  let i = 0;
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024;
    i += 1;
  }
  return `${n.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}
