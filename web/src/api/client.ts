import type {
  Artifact,
  ConnectionState,
  CreateRunRequest,
  CredentialStatus,
  ErrorEnvelope,
  PreflightReport,
  Run,
  RunEvent,
  RunPage,
  StreamFailure,
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

  listRuns(offset = 0, limit = 50) {
    const query = new URLSearchParams({ offset: String(offset), limit: String(limit) });
    return this.json<RunPage>(`/api/v1/runs?${query}`);
  }
  createRun(input: CreateRunRequest) {
    return this.json<Run>("/api/v1/runs", { method: "POST", body: JSON.stringify(input) });
  }
  getRun(runId: string) {
    return this.json<Run>(`/api/v1/runs/${encodeURIComponent(runId)}`);
  }
  preflight(input: CreateRunRequest, signal?: AbortSignal) {
    return this.json<PreflightReport>("/api/v1/config/validate", {
      method: "POST", body: JSON.stringify(input), signal,
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
  async artifact(runId: string, artifactId: string) {
    const artifact = await this.json<Artifact>(
      `/api/v1/runs/${encodeURIComponent(runId)}/artifacts/${encodeURIComponent(artifactId)}`,
    );
    try {
      const binary = atob(artifact.content);
      const bytes = Uint8Array.from(binary, (character) => character.charCodeAt(0));
      return { ...artifact, content: new TextDecoder().decode(bytes) };
    } catch {
      return artifact;
    }
  }
  credentialStatus(provider: string, host: string, signal?: AbortSignal) {
    return this.json<CredentialStatus>(
      `/api/v1/credentials/${encodeURIComponent(provider)}/${encodeURIComponent(host)}`,
      { signal },
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
  connect?: (url: string, lastSequence?: number, signal?: AbortSignal) => Promise<Response>;
  onEvent: (event: RunEvent) => void;
  onState: (state: ConnectionState) => void;
  onError?: (failure: StreamFailure) => void;
  retryDelay?: number;
}

export class RunEventStream {
  private latest?: number;
  private stopped = false;
  private abortController?: AbortController;
  private reader?: ReadableStreamDefaultReader<Uint8Array>;

  constructor(private readonly options: StreamOptions) {}

  stop() {
    this.stopped = true;
    this.abortController?.abort();
    void this.reader?.cancel();
  }

  async run(url: string, maxConnections = 5) {
    const connect = this.options.connect ?? defaultConnect;
    let failure: Omit<StreamFailure, "attempts" | "lastSequence"> = {
      kind: "connect", message: "Event stream unavailable",
    };
    for (let attempt = 0; !this.stopped && attempt < maxConnections; attempt += 1) {
      let phase: StreamFailure["kind"] = "connect";
      const resuming = this.latest !== undefined;
      try {
        this.abortController = new AbortController();
        const response = await connect(url, this.latest, this.abortController.signal);
        if (!response.ok) {
          phase = "http";
          throw new ApiError(`HTTP_${response.status}`, "Event stream failed");
        }
        phase = "read";
        await this.consumeResponse(response, () => {
          this.options.onState(resuming || attempt > 0 ? "reconnected" : "connected");
        });
        if (this.stopped) return;
        failure = { kind: "read", message: "Event stream closed unexpectedly" };
      } catch (error) {
        if (this.stopped || (error instanceof DOMException && error.name === "AbortError")) return;
        failure = {
          kind: phase,
          message: error instanceof Error ? error.message : "Event stream unavailable",
        };
      } finally {
        this.reader = undefined;
        this.abortController = undefined;
      }
      if (this.stopped) return;
      if (attempt + 1 >= maxConnections) {
        const terminal = {
          ...failure, attempts: attempt + 1, lastSequence: this.latest,
        };
        this.options.onState("disconnected");
        this.options.onState("failed");
        this.options.onError?.(terminal);
        return;
      }
      this.options.onState("disconnected");
      this.options.onState("reconnecting");
      await new Promise((resolve) => setTimeout(resolve, this.options.retryDelay ?? 1000));
    }
  }

  private async consumeResponse(response: Response, onConnected: () => void) {
    if (!response.body) return;
    const reader = response.body.getReader();
    this.reader = reader;
    const decoder = new TextDecoder();
    let buffered = "";
    let announced = false;
    while (!this.stopped) {
      const { done, value } = await reader.read();
      if (!announced) {
        announced = true;
        onConnected();
      }
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
    let decoded: Record<string, unknown>;
    try {
      decoded = JSON.parse(data.join("\n")) as Record<string, unknown>;
    } catch {
      return;
    }
    const enveloped = decoded.payload && typeof decoded.payload === "object" &&
      !Array.isArray(decoded.payload);
    const payload = enveloped ? decoded.payload as Record<string, unknown> : decoded;
    const at = enveloped && typeof decoded.at === "string" ? decoded.at : undefined;
    this.latest = Math.max(this.latest ?? 0, id!);
    this.options.onEvent({ sequence: id!, type, at, payload });
  }
}

async function defaultConnect(url: string, latest?: number, signal?: AbortSignal) {
  return fetch(url, {
    credentials: "same-origin",
    signal,
    headers: latest === undefined ? {} : { "Last-Event-ID": String(latest) },
  });
}
