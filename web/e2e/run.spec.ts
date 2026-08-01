import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";

const bootstrapToken = "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE";
const localURL = "http://127.0.0.1:4174";
const demoURL = "http://127.0.0.1:4175";
const approvalCanary = "quartz-orchid-7429";

async function expectNoSeriousAxeFindings(page: Page) {
  const results = await new AxeBuilder({ page }).analyze();
  const severe = results.violations.filter(({ impact }) =>
    impact === "serious" || impact === "critical");
  expect(severe, severe.map(({ id, help }) => `${id}: ${help}`).join("\n")).toEqual([]);
}

async function tabToButton(page: Page, name: string, maximum = 30) {
  const button = page.getByRole("button", { name });
  for (let count = 0; count < maximum; count += 1) {
    await page.keyboard.press("Tab");
    if (await button.evaluate((element) => element === document.activeElement)) return;
  }
  throw new Error(`keyboard focus did not reach ${name}`);
}

async function fillRunWithKeyboard(page: Page, repoRoot: string, task: string) {
  await page.keyboard.press("Tab");
  await expect(page.getByLabel("Repository path")).toBeFocused();
  await page.keyboard.type(repoRoot);
  await page.keyboard.press("Tab");
  await expect(page.getByLabel("Task description")).toBeFocused();
  await page.keyboard.type(task);
  await page.keyboard.press("Tab");
  await expect(page.getByLabel("Provider")).toBeFocused();
  await page.keyboard.type("openai");
  await page.keyboard.press("Tab");
  await expect(page.getByLabel("Model")).toBeFocused();
  await page.keyboard.type("mock-v1");
  await page.keyboard.press("Tab");
  await expect(page.getByLabel("Permission profile")).toBeFocused();
  await page.keyboard.press("Tab");
  await expect(page.getByRole("button", { name: "Validate preflight" })).toBeFocused();
  await page.keyboard.press("Tab");
  await page.keyboard.press("Enter");
  await page.keyboard.press("Tab");
  await expect(page.getByLabel("Endpoint", { exact: true })).toBeFocused();
  await page.keyboard.type("https://api.openai.com");
  await page.keyboard.press("Tab");
  await expect(page.getByLabel("Confirm this custom endpoint")).toBeFocused();
  await page.keyboard.press("Space");
  await page.keyboard.press("Shift+Tab");
  await page.keyboard.press("Shift+Tab");
  await page.keyboard.press("Shift+Tab");
  await expect(page.getByRole("button", { name: "Validate preflight" })).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(page.getByText("Preflight passed. Run creation is available.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Create run" })).toBeEnabled();
  await expectNoSeriousAxeFindings(page);
  await page.keyboard.press("Tab");
  await expect(page.getByRole("button", { name: "Create run" })).toBeFocused();
  await page.keyboard.press("Enter");
}

test("authenticated Go httpapi composition supports the complete supervised workflow", async ({ page, request }) => {
  const unauthenticated = await request.get(`${localURL}/api/v1/runs?offset=0&limit=50`);
  expect(unauthenticated.status()).toBe(403);

  await page.goto(`${localURL}/?bootstrap=${bootstrapToken}`);
  await expect(page).toHaveURL(`${localURL}/`);
  const repositoryResponse = await request.get(`${localURL}/e2e/repository`);
  const { repo_root: repoRoot } = await repositoryResponse.json() as { repo_root: string };
  await expect(page.getByRole("heading", { name: "Dashboard" })).toBeVisible();
  await expect(page.getByText(/No runs were returned/i)).toBeVisible();
  await expect(page.getByTestId("kpi-card")).toHaveCount(4);
  await expect(page.getByRole("img", { name: "Run state distribution" })).toBeVisible();
  await expect(page.getByRole("region", { name: "Live activity" })).toBeVisible();
  await expectNoSeriousAxeFindings(page);
  const credentialPutStatus = await page.evaluate(async (secret) => {
    const runtime = window.__AI4SE_RUNTIME__;
    const response = await fetch("/api/v1/credentials/openai/api.openai.com", {
      method: "PUT",
      credentials: "same-origin",
      headers: {
        "Content-Type": "application/json",
        "X-CSRF-Token": runtime?.csrfToken ?? "",
      },
      body: JSON.stringify({ secret }),
    });
    return response.status;
  }, approvalCanary);
  expect(credentialPutStatus).toBe(204);

  const invalidCSRF = await page.evaluate(async () => {
    const response = await fetch("/api/v1/config/validate", {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json", "X-CSRF-Token": "wrong" },
      body: JSON.stringify({
        repo_root: "C:\\workspace\\fixture", task: "Blocked request",
        provider: "openai", model: "mock-v1", endpoint: "https://api.openai.com",
        confirm_custom_endpoint: false, profile: "supervised",
      }),
    });
    return response.status;
  });
  expect(invalidCSRF).toBe(403);

  await page.keyboard.press("Tab");
  await page.keyboard.press("Tab");
  await page.keyboard.press("Tab");
  await expect(page.getByRole("button", { name: /New Run/ })).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("heading", { name: "New Run" })).toBeVisible();
  await expectNoSeriousAxeFindings(page);
  await fillRunWithKeyboard(page, repoRoot, "Repair the failing deterministic check");
  await expect(page.getByRole("heading", { name: "run-created-1" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Approval required" })).toBeVisible();
  await expectNoSeriousAxeFindings(page);
  await expect(page.getByText("Timestamp unavailable")).toHaveCount(0);
  await expect(page.getByText("apply_patch", { exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Approval required" }).locator(".."))
    .toContainText("Affected files: [REDACTED].txt");
  await expect(page.locator("body")).not.toContainText(approvalCanary);
  const storedEvents = await request.get(`${localURL}/e2e/events/run-created-1`);
  const storedEventBody = await storedEvents.text();
  expect(storedEventBody).toContain("ApprovalRequired");
  expect(storedEventBody).toContain("[REDACTED]");
  expect(storedEventBody).not.toContain(approvalCanary);
  const streamedEvents = await page.evaluate(async () => {
    const response = await fetch("/api/v1/runs/run-created-1/events", {
      credentials: "same-origin",
    });
    const reader = response.body?.getReader();
    if (!reader) throw new Error("event stream body is unavailable");
    const decoder = new TextDecoder();
    let output = "";
    while (!output.includes("ApprovalRequired")) {
      const { done, value } = await reader.read();
      if (done) break;
      output += decoder.decode(value, { stream: true });
    }
    await reader.cancel();
    return output;
  });
  expect(streamedEvents).toContain("ApprovalRequired");
  expect(streamedEvents).toContain("[REDACTED]");
  expect(streamedEvents).not.toContain(approvalCanary);
  await tabToButton(page, "Approve once");
  await expect(page.getByRole("button", { name: "Approve once" })).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("heading", { name: "Approval required" })).toHaveCount(0);
  await expect(page.getByText("SUCCEEDED").first()).toBeVisible();
  await expectNoSeriousAxeFindings(page);

  await page.getByRole("button", { name: /Credentials/ }).click();
  await expect(page.getByRole("heading", { name: "Credentials" })).toBeVisible();
  await expect(page.getByText("Configured").first()).toBeVisible();
  await page.getByLabel("Secret").fill("one-time-secret");
  await page.getByRole("button", { name: "Update credential" }).click();
  await expect(page.getByLabel("Secret")).toHaveValue("");
  await expect(page.getByRole("button", { name: "Update credential" })).toBeVisible();
  await page.getByRole("button", { name: "Clear credential" }).click();
  await expect(page.getByText("Not configured").first()).toBeVisible();
  await expect(page.getByRole("button", { name: "Add credential" })).toBeVisible();
  await expectNoSeriousAxeFindings(page);

  const armDashboardFailure = await request.post(`${localURL}/e2e/fail-next`, {
    data: { target: "runs" },
  });
  expect(armDashboardFailure.status()).toBe(204);
  await page.reload();
  await expect(page.getByRole("alert")).toContainText("Injected run-list failure");
  await expect(page.getByRole("button", { name: "Retry loading runs" })).toBeVisible();
  await expectNoSeriousAxeFindings(page);
  await page.getByRole("button", { name: "Retry loading runs" }).click();
  await expect(page.getByRole("heading", { name: "Dashboard" })).toBeVisible();
  await expect(page.getByRole("button", { name: "run-created-1" })).toBeVisible();

  const armCredentialFailure = await request.post(`${localURL}/e2e/fail-next`, {
    data: { target: "credential" },
  });
  expect(armCredentialFailure.status()).toBe(204);
  await page.getByRole("button", { name: /Credentials/ }).click();
  await expect(page.getByRole("alert")).toContainText("Injected credential-status failure");
  await expect(page.getByRole("button", { name: "Retry status" })).toBeVisible();
  await expectNoSeriousAxeFindings(page);
  await page.getByRole("button", { name: "Retry status" }).click();
  await expect(page.getByText("Not configured").first()).toBeVisible();
});

test("Go demo router exposes only fixed SIMULATED data and prunes mutations", async ({ page, request }) => {
  await page.goto(`${demoURL}/`);
  await expect(page.getByRole("heading", { name: "Demo Gallery" })).toBeVisible();
  await expectNoSeriousAxeFindings(page);
  await page.getByRole("button", { name: "Open SIMULATED demo-feedback demo" }).click();
  await expect(page.getByText("POLICY_DENIED")).toBeVisible();
  await expect(page.getByText("TEST_FAILURE")).toBeVisible();
  await expect(page.getByText("ACTION_CHANGED")).toBeVisible();
  await expect(page.getByText("SUCCEEDED").first()).toBeVisible();
  await expect(page.locator(".timeline .sim-label")).toHaveCount(4);
  await expect(page.getByRole("region", { name: "Read-only diff" })).toContainText("return a + b");
  await expect(page.getByText("ALL_REQUIRED_CHECKS_PASSED")).toBeVisible();
  await expectNoSeriousAxeFindings(page);

  const create = await request.post(`${demoURL}/api/v1/runs`, { data: {} });
  const credential = await request.get(
    `${demoURL}/api/v1/credentials/openai/api.openai.com`,
  );
  const artifact = await request.get(
    `${demoURL}/api/v1/runs/demo-feedback/artifacts/not-fixed`,
  );
  expect(create.status()).toBe(404);
  expect(credential.status()).toBe(404);
  expect(artifact.status()).toBe(404);
});
