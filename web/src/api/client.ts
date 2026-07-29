import type {
  Artifact,
  ConnectionState,
  CreateRunRequest,
  CredentialStatus,
  ErrorEnvelope,
  PreflightReport,
  Run,
  RunEvent,
} from "./types";

export class ApiError extends Error {
  constructor(
    readonly code: string,
    message: string,
    readonly requestId?: string,
  ) {
    super(message);
  }
}

export class ApiClient {
  constructor(private readonly options: { csrfToken?: string } = {}) {}

  createRun(input: CreateRunRequest) {
    return this.json<Run>("/api/v1/runs", { method: "POST", body: JSON.stringify(input) });
  }
  getRun(runId: string) {
    return this.json<Run>(`/api/v1/runs/${encodeURIComponent(runId)}`);
  }
  preflight(input: CreateRunRequest) {
    return this.json<PreflightReport>("/api/v1/config/validate", {
      method: "POST", body: JSON.stringify(input),
    });
  }
  cancel(runId: string) {
    return this.empty(`/api/v1/runs/${encodeURIComponent(runId)}/cancel`, "POST", {});
  }
  approve(runId: string, digest: string) {
    return this.empty(
      `/api/v1/runs/${encodeURIComponent(runId)}/approvals/${encodeURIComponent(digest)}/approve`,
      "POST", {},
    );
  }
  reject(runId: string, digest: string, terminate: boolean) {
    return this.empty(
      `/api/v1/runs/${encodeURIComponent(runId)}/approvals/${encodeURIComponent(digest)}/reject`,
      "POST", { terminate },
    );
  }
  artifact(runId: string, artifactId: string) {
    return this.json<Artifact>(
      `/api/v1/runs/${encodeURIComponent(runId)}/artifacts/${encodeURIComponent(artifactId)}`,
    );
  }
  credentialStatus(provider: string, host: string) {
    return this.json<CredentialStatus>(
      `/api/v1/credentials/${encodeURIComponent(provider)}/${encodeURIComponent(host)}`,
    );
  }
  saveCredential(provider: string, host: string, secret: string) {
    return this.empty(
      `/api/v1/credentials/${encodeURIComponent(provider)}/${encodeURIComponent(host)}`,
      "PUT", { secret },
    );
  }
  clearCredential(provider: string, host: string) {
    return this.empty(
      `/api/v1/credentials/${encodeURIComponent(provider)}/${encodeURIComponent(host)}`,
      "DELETE",
    );
  }

  private async empty(path: string, method: string, body?: unknown) {
    await this.request(path, {
      method,
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  }

  private async json<T>(path: string, init: RequestInit = {}): Promise<T> {
    const response = await this.request(path, init);
    return response.json() as Promise<T>;
  }

  private async request(path: string, init: RequestInit) {
    const mutation = init.method && init.method !== "GET";
    const response = await fetch(path, {
      ...init,
      credentials: "same-origin",
      headers: {
        "Content-Type": "application/json",
        ...(mutation && this.options.csrfToken
          ? { "X-CSRF-Token": this.options.csrfToken }
          : {}),
        ...init.headers,
      },
    });
    if (!response.ok) {
      let envelope: ErrorEnvelope | undefined;
      try { envelope = await response.json() as ErrorEnvelope; } catch { /* bounded fallback */ }
      throw new ApiError(
        envelope?.error.code ?? `HTTP_${response.status}`,
        envelope?.error.message ?? "Request could not be completed",
        envelope?.error.request_id,
      );
    }
    return response;
  }
}

interface StreamOptions {
  connect?: (url: string, lastSequence?: number) => Promise<Response>;
  onEvent: (event: RunEvent) => void;
  onState: (state: ConnectionState) => void;
  retryDelay?: number;
}

export class RunEventStream {
  private latest?: number;
  private stopped = false;

  constructor(private readonly options: StreamOptions) {}

  stop() { this.stopped = true; }

  async run(url: string, maxConnections = Number.POSITIVE_INFINITY) {
    const connect = this.options.connect ?? defaultConnect;
    for (let attempt = 0; !this.stopped && attempt < maxConnections; attempt += 1) {
      if (attempt === 0) this.options.onState("connected");
      else this.options.onState("reconnected");
      const response = await connect(url, this.latest);
      if (!response.ok) throw new ApiError(`HTTP_${response.status}`, "Event stream failed");
      await this.consumeResponse(response);
      if (attempt + 1 >= maxConnections || this.stopped) break;
      this.options.onState("disconnected");
      this.options.onState("reconnecting");
      await new Promise((resolve) => setTimeout(resolve, this.options.retryDelay ?? 1000));
    }
  }

  private async consumeResponse(response: Response) {
    if (!response.body) return;
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffered = "";
    while (!this.stopped) {
      const { done, value } = await reader.read();
      buffered += decoder.decode(value, { stream: !done });
      const frames = buffered.split(/\r?\n\r?\n/);
      buffered = frames.pop() ?? "";
      frames.forEach((frame) => this.consumeFrame(frame));
      if (done) {
        if (buffered.trim()) this.consumeFrame(buffered);
        break;
      }
    }
  }

  private consumeFrame(frame: string) {
    if (!frame.trim() || frame.startsWith(":")) return;
    let id: number | undefined;
    let type = "RunEvent";
    const data: string[] = [];
    for (const line of frame.split(/\r?\n/)) {
      if (line.startsWith("id:")) id = Number(line.slice(3).trim());
      else if (line.startsWith("event:")) type = line.slice(6).trim();
      else if (line.startsWith("data:")) data.push(line.slice(5).trimStart());
    }
    if (!Number.isSafeInteger(id) || id! <= 0 || data.length === 0) return;
    const payload = JSON.parse(data.join("\n")) as Record<string, unknown>;
    this.latest = Math.max(this.latest ?? 0, id!);
    this.options.onEvent({ sequence: id!, type, payload });
  }
}

async function defaultConnect(url: string, latest?: number) {
  return fetch(url, {
    credentials: "same-origin",
    headers: latest === undefined ? {} : { "Last-Event-ID": String(latest) },
  });
}
