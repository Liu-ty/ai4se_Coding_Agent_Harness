import { afterEach, expect, it, vi } from "vitest";
import { ApiClient, RunEventStream } from "./client";

afterEach(() => vi.unstubAllGlobals());

it("sends typed JSON mutations with the injected CSRF token", async () => {
  const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
  vi.stubGlobal("fetch", fetchMock);
  const client = new ApiClient({ csrfToken: "csrf-test" });
  await client.approve("run-1", "digest-a");
  expect(fetchMock).toHaveBeenCalledWith(
    "/api/v1/runs/run-1/approvals/digest-a/approve",
    expect.objectContaining({
      method: "POST",
      credentials: "same-origin",
      headers: expect.objectContaining({ "X-CSRF-Token": "csrf-test" }),
    }),
  );
});

it("lists the bounded run page using the frozen pagination schema", async () => {
  const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
    runs: [{ id: "run-1", state: "DECIDING", profile: "supervised", task: "Repair",
      repo_root: "C:\\repo", current_stage: "unit", created_at: "2026-07-29T00:00:00Z",
      updated_at: "2026-07-29T00:01:00Z" }],
    page: { offset: 0, limit: 25, returned: 1, has_more: false },
  }), { status: 200, headers: { "Content-Type": "application/json" } }));
  vi.stubGlobal("fetch", fetchMock);
  const page = await new ApiClient().listRuns(0, 25);
  expect(page.runs[0].id).toBe("run-1");
  expect(page.page).toEqual({ offset: 0, limit: 25, returned: 1, has_more: false });
  expect(fetchMock).toHaveBeenCalledWith("/api/v1/runs?offset=0&limit=25", expect.any(Object));
});

it("reconnects SSE from the latest observed sequence", async () => {
  const requests: string[] = [];
  const states: string[] = [];
  const stream = new RunEventStream({
    connect: async (_url, lastSequence) => {
      requests.push(lastSequence === undefined ? "initial" : String(lastSequence));
      if (requests.length === 1) {
        return new Response("id: 41\nevent: FeedbackProduced\ndata: {\"summary\":\"failed\"}\n\n");
      }
      return new Response("id: 42\nevent: RunSucceeded\ndata: {\"reason\":\"SUCCEEDED\"}\n\n");
    },
    onEvent: () => undefined,
    onState: (state) => states.push(state),
    retryDelay: 0,
  });
  await stream.run("/api/v1/runs/run-1/events", 2);
  expect(requests).toEqual(["initial", "41"]);
  expect(states).toEqual(["connected", "disconnected", "reconnecting", "reconnected"]);
});

it("delivers an SSE event before the long-lived response closes", async () => {
  let controller!: ReadableStreamDefaultController<Uint8Array>;
  const response = new Response(new ReadableStream({
    start(value) { controller = value; },
  }));
  const observed = new Promise<number>((resolve) => {
    const stream = new RunEventStream({
      connect: async () => response,
      onEvent: (event) => { resolve(event.sequence); stream.stop(); controller.close(); },
      onState: () => undefined,
    });
    void stream.run("/api/v1/runs/run-1/events", 1);
  });
  controller.enqueue(new TextEncoder().encode(
    "id: 9\nevent: FeedbackProduced\ndata: {\"summary\":\"bounded\"}\n\n",
  ));
  await expect(Promise.race([
    observed,
    new Promise((_, reject) => setTimeout(() => reject(new Error("event delivery stalled")), 100)),
  ])).resolves.toBe(9);
});

it("stop aborts the active request and cancels its reader before another chunk", async () => {
  let cancelled = false;
  let signal: AbortSignal | undefined;
  let controller!: ReadableStreamDefaultController<Uint8Array>;
  const response = new Response(new ReadableStream({
    start(value) { controller = value; },
    cancel() { cancelled = true; },
  }));
  const stream = new RunEventStream({
    connect: async (_url, _latest, currentSignal) => {
      signal = currentSignal;
      return response;
    },
    onEvent: () => undefined,
    onState: () => undefined,
  });
  const running = stream.run("/events");
  await vi.waitFor(() => expect(signal).toBeDefined());
  stream.stop();
  await running;
  expect(signal?.aborted).toBe(true);
  expect(cancelled).toBe(true);
});

it("retries rejected connects and read errors, then announces reconnected after success", async () => {
  const states: string[] = [];
  const observed: number[] = [];
  let attempt = 0;
  const stream = new RunEventStream({
    connect: async () => {
      attempt += 1;
      if (attempt === 1) throw new Error("connect failed");
      if (attempt === 2) return new Response(new ReadableStream({
        start(controller) { controller.error(new Error("read failed")); },
      }));
      return new Response("id: 7\nevent: RunEvent\ndata: {\"ok\":true}\n\n");
    },
    onEvent: (event) => observed.push(event.sequence),
    onState: (state) => states.push(state),
    retryDelay: 0,
  });
  await stream.run("/events", 3);
  expect(observed).toEqual([7]);
  expect(states).toEqual([
    "disconnected", "reconnecting", "disconnected", "reconnecting", "reconnected",
  ]);
});

it("bounds default reconnect attempts", async () => {
  const connect = vi.fn().mockRejectedValue(new Error("offline"));
  const stream = new RunEventStream({
    connect,
    onEvent: () => undefined,
    onState: () => undefined,
    retryDelay: 0,
  });
  await stream.run("/events");
  expect(connect).toHaveBeenCalledTimes(5);
});

it("skips a malformed frame and continues with the next valid event", async () => {
  const observed: number[] = [];
  const stream = new RunEventStream({
    connect: async () => new Response(
      "id: 2\ndata: {bad json}\n\nid: 3\nevent: FeedbackProduced\ndata: {\"ok\":true}\n\n",
    ),
    onEvent: (event) => observed.push(event.sequence),
    onState: () => undefined,
  });
  await stream.run("/events", 1);
  expect(observed).toEqual([3]);
});

it("unwraps the server timestamp and payload SSE envelope", async () => {
  let observed: { at?: string; summary?: unknown } = {};
  const stream = new RunEventStream({
    connect: async () => new Response(
      "id: 11\nevent: FeedbackProduced\ndata: {\"at\":\"2026-07-29T01:02:03Z\",\"payload\":{\"summary\":\"failed\"}}\n\n",
    ),
    onEvent: (event) => { observed = { at: event.at, summary: event.payload.summary }; },
    onState: () => undefined,
  });
  await stream.run("/events", 1);
  expect(observed).toEqual({ at: "2026-07-29T01:02:03Z", summary: "failed" });
});
