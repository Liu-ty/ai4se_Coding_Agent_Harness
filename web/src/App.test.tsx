import { render, screen } from "@testing-library/react";
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

afterEach(() => vi.unstubAllGlobals());

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
          action: "apply_patch", files: ["src/sum.ts"], risk: "Tracked source change",
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
