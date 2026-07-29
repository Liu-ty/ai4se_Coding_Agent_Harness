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
