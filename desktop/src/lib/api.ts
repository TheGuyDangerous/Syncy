import { invoke } from "@tauri-apps/api/core";

export interface Status {
  device_id: string;
  folders: number;
  devices: number;
}

export type Direction = "sendreceive" | "sendonly" | "receiveonly";

export interface Folder {
  id: string;
  label: string;
  path: string;
  direction: Direction;
  paused: boolean;
  added_at: string;
}

export interface Device {
  id: string;
  name: string;
  trusted: boolean;
  last_seen: string;
  added_at: string;
}

export interface Conflict {
  folder_id: string;
  path: string;
}

export interface FileVersion {
  stamp: string;
  path: string;
  mod_time: string;
  size: number;
}

export interface NewFolder {
  id: string;
  path: string;
  label?: string;
  direction?: Direction;
}

export class ApiError extends Error {}

interface DaemonInfo {
  base_url: string;
  token: string;
}

let daemon: Promise<DaemonInfo> | null = null;

function daemonInfo(): Promise<DaemonInfo> {
  if (!daemon) {
    daemon = invoke<DaemonInfo>("daemon_info").catch((e: unknown) => {
      daemon = null;
      throw e;
    });
  }
  return daemon;
}

const OFFLINE =
  "Can't reach the sync engine. It may still be starting — try again in a moment.";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let info: DaemonInfo;
  try {
    info = await daemonInfo();
  } catch {
    throw new ApiError(OFFLINE);
  }
  let res: Response;
  try {
    res = await fetch(info.base_url + path, {
      ...init,
      headers: {
        Authorization: `Bearer ${info.token}`,
        ...(init?.body != null ? { "Content-Type": "application/json" } : {}),
      },
    });
  } catch {
    throw new ApiError(OFFLINE);
  }
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new ApiError(body.trim() || `The engine returned an error (${res.status}).`);
  }
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

export const api = {
  status: () => request<Status>("/status"),
  folders: () => request<Folder[]>("/folders"),
  addFolder: (folder: NewFolder) =>
    request<void>("/folders", { method: "POST", body: JSON.stringify(folder) }),
  removeFolder: (id: string) =>
    request<void>(`/folders/${encodeURIComponent(id)}`, { method: "DELETE" }),
  devices: () => request<Device[]>("/devices"),
  conflicts: () => request<Conflict[]>("/conflicts"),
  versions: (folderId: string, relPath: string) =>
    request<FileVersion[]>(
      `/folders/${encodeURIComponent(folderId)}/versions?path=${encodeURIComponent(relPath)}`,
    ),
};

export function errorMessage(e: unknown): string {
  if (e instanceof Error) return e.message;
  return String(e);
}
