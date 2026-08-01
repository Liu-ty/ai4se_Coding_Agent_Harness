import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "vitest";
import { App } from "./App";
import type { RuntimeConfig } from "./api/types";

const localRuntime: RuntimeConfig = {
  csrfToken: "csrf-test",
  capabilities: {
    createRuns: true, cancelRuns: true, approvals: true, artifacts: true,
    configValidation: true, credentials: true, demo: false, fixedRuns: [],
  },
};
const run = {
  id: "run-1", state: "AWAITING_APPROVAL", profile: "supervised", task: "Repair",
  repo_root: "C:\\repo", current_stage: "unit", created_at: "2026-07-29T00:00:00Z",
  updated_at: "2026-07-29T00:01:00Z",
};

afterEach(() => {
  vi.unstubAllGlobals();
  window.history.replaceState({}, "", "/");
});

it("loads real run pages, selected snapshot, SSE event, and referenced diff artifact", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = String(input);
    if (path.includes("?offset=")) return Response.json({
      runs: [run], page: { offset: 0, limit: 50, returned: 1, has_more: false },
    });
    if (path.endsWith("/events")) return new Response(
      "id: 1\nevent: ApprovalRequired\ndata: " + JSON.stringify({
        at: "2026-07-29T00:02:00Z",
        payload: {
          summary: "Exact patch requires approval", digest: "sha256:abc",
          action: { kind: "apply_patch", args: { patch: "bounded" } },
          affected_files: ["src/sum.ts"], risk: "GUARDED",
          risk_reason: "Tracked source change",
          artifact_id: "diff-1", budgets: { decisions: { used: 1, limit: 30 } },
        },
      }) + "\n\n",
    );
    if (path.endsWith("/artifacts/diff-1")) return Response.json({
      id: "diff-1", run_id: "run-1", kind: "diff", sha256: "abc",
      content: btoa("- return 1\n+ return a + b"), truncated: false,
    });
    if (path.endsWith("/run-1")) return Response.json(run);
    throw new Error(`unexpected request ${path}`);
  });
  vi.stubGlobal("fetch", fetchMock);
  render(<App runtimeConfig={localRuntime} />);
  await userEvent.click(await screen.findByRole("button", { name: "run-1" }));
  expect(await screen.findByRole("heading", { name: "Approval required" })).toBeVisible();
  expect(await screen.findByRole("region", { name: "Read-only diff" })).toHaveTextContent("return a + b");
  expect(screen.getByText(/2026-07-29 00:02:00 UTC/)).toBeVisible();
  expect(fetchMock).toHaveBeenCalledWith(
    "/api/v1/runs/run-1/artifacts/diff-1", expect.any(Object),
  );
});

it("shows list failure recovery and retries the real endpoint", async () => {
  const fetchMock = vi.fn()
    .mockRejectedValueOnce(new Error("server unavailable"))
    .mockResolvedValueOnce(Response.json({
      runs: [], page: { offset: 0, limit: 50, returned: 0, has_more: false },
    }));
  vi.stubGlobal("fetch", fetchMock);
  render(<App runtimeConfig={localRuntime} />);
  expect(await screen.findByRole("alert")).toHaveTextContent("server unavailable");
  await userEvent.click(screen.getByRole("button", { name: "Retry loading runs" }));
  expect(await screen.findByText(/No runs were returned/i)).toBeVisible();
  expect(fetchMock).toHaveBeenCalledTimes(2);
});

it("keeps local runs out of the SIMULATED demo gallery and shows an explicit empty state", async () => {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(Response.json({
    runs: [run], page: { offset: 0, limit: 50, returned: 1, has_more: false },
  })));
  render(<App runtimeConfig={localRuntime} />);
  await userEvent.click(await screen.findByRole("button", { name: /Demo Gallery/ }));
  expect(await screen.findByRole("heading", { name: "Demo Gallery" })).toBeVisible();
  expect(screen.getByText(/no fixed demo scenarios/i)).toBeVisible();
  expect(screen.queryByRole("button", { name: "Open SIMULATED demo" })).toBeNull();
  expect(screen.queryByText("run-1")).toBeNull();
});

it("refreshes the selected snapshot when an SSE event is the only state change", async () => {
  let snapshots = 0;
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = String(input);
    if (path.includes("?offset=")) return Response.json({
      runs: [run], page: { offset: 0, limit: 50, returned: 1, has_more: false },
    });
    if (path.endsWith("/events")) return new Response(
      "id: 2\nevent: RunSucceeded\ndata: " + JSON.stringify({
        at: "2026-07-29T00:03:00Z", payload: { reason: "ALL_REQUIRED_CHECKS_PASSED" },
      }) + "\n\n",
    );
    if (path.endsWith("/run-1")) {
      snapshots += 1;
      return Response.json({
        ...run, state: snapshots === 1 ? "AWAITING_APPROVAL" : "SUCCEEDED",
      });
    }
    throw new Error(`unexpected request ${path}`);
  });
  vi.stubGlobal("fetch", fetchMock);
  render(<App runtimeConfig={localRuntime} />);
  await userEvent.click(await screen.findByRole("button", { name: "run-1" }));
  expect(await screen.findByText("SUCCEEDED")).toBeVisible();
  expect(snapshots).toBeGreaterThanOrEqual(2);
});

it("does not let an older selected-run request overwrite a newer selection", async () => {
  let resolveOld!: (response: Response) => void;
  const oldSnapshot = new Promise<Response>((resolve) => { resolveOld = resolve; });
  const run2 = { ...run, id: "run-2", task: "Newer selection" };
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = String(input);
    if (path.includes("?offset=")) return Response.json({
      runs: [run, run2], page: { offset: 0, limit: 50, returned: 2, has_more: false },
    });
    if (path.endsWith("/run-1")) return oldSnapshot;
    if (path.endsWith("/run-2")) return Response.json(run2);
    if (path.endsWith("/events")) return new Response("");
    throw new Error(`unexpected request ${path}`);
  });
  vi.stubGlobal("fetch", fetchMock);
  render(<App runtimeConfig={localRuntime} />);
  await userEvent.click(await screen.findByRole("button", { name: "run-1" }));
  await userEvent.click(screen.getByRole("button", { name: /Dashboard/ }));
  await userEvent.click(screen.getByRole("button", { name: "run-2" }));
  expect(await screen.findByRole("heading", { name: "run-2" })).toBeVisible();
  resolveOld(Response.json(run));
  await vi.waitFor(() =>
    expect(screen.getByRole("heading", { name: "run-2" })).toBeVisible());
  expect(screen.queryByRole("heading", { name: "run-1" })).toBeNull();
});

it.each([
  ["ApprovalGranted", "AWAITING_APPROVAL"],
  ["ApprovalRejected", "AWAITING_APPROVAL"],
  ["RunSucceeded", "AWAITING_APPROVAL"],
  ["RunStarted", "EXECUTING"],
])("does not reopen a closed approval after %s in state %s", async (closingType, state) => {
  const currentRun = { ...run, state };
  const approvalPayload = {
    digest: "sha256:closed",
    action: { kind: "apply_patch", args: { patch: "bounded" } },
    affected_files: ["src/sum.ts"],
    risk: "GUARDED",
    risk_reason: "Tracked source change",
  };
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = String(input);
    if (path.includes("?offset=")) return Response.json({
      runs: [currentRun], page: { offset: 0, limit: 50, returned: 1, has_more: false },
    });
    if (path.endsWith("/events")) return new Response(
      `id: 1\nevent: ApprovalRequired\ndata: ${JSON.stringify({
        at: "2026-07-29T00:01:00Z", payload: approvalPayload,
      })}\n\nid: 2\nevent: ${closingType}\ndata: ${JSON.stringify({
        at: "2026-07-29T00:02:00Z", payload: { digest: "sha256:closed" },
      })}\n\n`,
    );
    if (path.endsWith("/run-1")) return Response.json(currentRun);
    throw new Error(`unexpected request ${path}`);
  });
  vi.stubGlobal("fetch", fetchMock);
  render(<App runtimeConfig={localRuntime} />);
  await userEvent.click(await screen.findByRole("button", { name: "run-1" }));
  expect(await screen.findByText("Latest sequence: 2. Reconnect resumes from this cursor.")).toBeVisible();
  expect(screen.queryByRole("heading", { name: "Approval required" })).toBeNull();
});

it("preserves approval events and their cursor after a local decision", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = String(input);
    if (path.includes("?offset=")) return Response.json({
      runs: [run], page: { offset: 0, limit: 50, returned: 1, has_more: false },
    });
    if (path.endsWith("/events")) return new Response(
      "id: 7\nevent: ApprovalRequired\ndata: " + JSON.stringify({
        at: "2026-07-29T00:02:00Z",
        payload: {
          digest: "sha256:preserve", action: { kind: "apply_patch", args: {} },
          affected_files: ["src/sum.ts"], risk: "GUARDED", risk_reason: "Tracked change",
        },
      }) + "\n\n",
    );
    if (path.includes("/approve")) return new Response(null, { status: 204 });
    if (path.endsWith("/run-1")) return Response.json(run);
    throw new Error(`unexpected request ${path}`);
  });
  vi.stubGlobal("fetch", fetchMock);
  render(<App runtimeConfig={localRuntime} />);
  await userEvent.click(await screen.findByRole("button", { name: "run-1" }));
  await userEvent.click(await screen.findByRole("button", { name: "Approve once" }));
  expect(await screen.findByText("#7 ApprovalRequired")).toBeVisible();
  expect(screen.getByText("Latest sequence: 7. Reconnect resumes from this cursor.")).toBeVisible();
  expect(screen.queryByRole("heading", { name: "Approval required" })).toBeNull();
});

it("does not enable demo capabilities for unrelated query parameters", async () => {
  window.history.replaceState({}, "", "/?nofixture=demo");
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(Response.json({
    runs: [], page: { offset: 0, limit: 50, returned: 0, has_more: false },
  })));
  render(<App />);
  expect(await screen.findByRole("button", { name: /New Run/ })).toBeVisible();
  expect(screen.getByRole("button", { name: /Dashboard/ })).toHaveAttribute("aria-current", "page");
  await userEvent.click(screen.getByRole("button", { name: /New Run/ }));
  expect(screen.getByRole("button", { name: /New Run/ })).toHaveAttribute("aria-current", "page");
  expect(screen.getByRole("button", { name: /Dashboard/ })).not.toHaveAttribute("aria-current");
});

it("keeps run B cancellation enabled while run A has a delayed cancellation", async () => {
  let resolveCancel!: (response: Response) => void;
  const cancelResponse = new Promise<Response>((resolve) => { resolveCancel = resolve; });
  const run2 = { ...run, id: "run-2", state: "EXECUTING", task: "Newer selection" };
  const cancelledRun = { ...run, state: "STOPPED" };
  const cancelledRun2 = { ...run2, state: "STOPPED" };
  let run1Snapshots = 0;
  let run2Snapshots = 0;
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = String(input);
    if (path.includes("?offset=")) return Response.json({
      runs: [run, run2], page: { offset: 0, limit: 50, returned: 2, has_more: false },
    });
    if (path.endsWith("/run-1/cancel")) return cancelResponse;
    if (path.endsWith("/run-2/cancel")) return new Response(null, { status: 204 });
    if (path.endsWith("/run-1")) {
      run1Snapshots += 1;
      return Response.json(run1Snapshots === 1 ? run : cancelledRun);
    }
    if (path.endsWith("/run-2")) {
      run2Snapshots += 1;
      return Response.json(run2Snapshots === 1 ? run2 : cancelledRun2);
    }
    if (path.endsWith("/events")) return new Response("");
    throw new Error(`unexpected request ${path}`);
  });
  vi.stubGlobal("fetch", fetchMock);
  render(<App runtimeConfig={localRuntime} />);
  await userEvent.click(await screen.findByRole("button", { name: "run-1" }));
  await userEvent.click(await screen.findByRole("button", { name: "Cancel run" }));
  await userEvent.click(screen.getByRole("button", { name: /Dashboard/ }));
  await userEvent.click(screen.getByRole("button", { name: "run-2" }));
  expect(await screen.findByRole("heading", { name: "run-2" })).toBeVisible();
  const cancelRun2 = screen.getByRole("button", { name: "Cancel run" });
  expect(cancelRun2).toBeEnabled();
  await userEvent.click(cancelRun2);
  expect(await screen.findByText("STOPPED")).toBeVisible();
  await act(async () => resolveCancel(new Response(null, { status: 204 })));
  expect(screen.getByRole("heading", { name: "run-2" })).toBeVisible();
  expect(screen.queryByRole("heading", { name: "run-1" })).toBeNull();
});
