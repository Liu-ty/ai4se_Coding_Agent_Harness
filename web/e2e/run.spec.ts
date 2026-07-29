import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";

const bootstrapToken = "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE";
const localURL = "http://127.0.0.1:4174";
const demoURL = "http://127.0.0.1:4175";

async function expectNoSeriousAxeFindings(page: Page) {
  const results = await new AxeBuilder({ page }).analyze();
  const severe = results.violations.filter(({ impact }) =>
    impact === "serious" || impact === "critical");
  expect(severe, severe.map(({ id, help }) => `${id}: ${help}`).join("\n")).toEqual([]);
}

async function fillRun(page: Page, task: string) {
  await page.getByLabel("Repository path").fill("C:\\workspace\\fixture");
  await page.getByLabel("Task description").fill(task);
  await page.getByLabel("Provider").fill("mock");
  await page.getByLabel("Model").fill("mock-v1");
  await expect(page.getByRole("button", { name: "Create run" })).toBeDisabled();
  await page.getByRole("button", { name: "Validate preflight" }).click();
  await expect(page.getByText("Preflight passed. Run creation is available.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Create run" })).toBeEnabled();
  await page.getByRole("button", { name: "Create run" }).click();
}

test("authenticated Go httpapi composition supports the complete supervised workflow", async ({ page, request }) => {
  const unauthenticated = await request.get(`${localURL}/api/v1/runs?offset=0&limit=50`);
  expect(unauthenticated.status()).toBe(403);

  await page.goto(`${localURL}/?bootstrap=${bootstrapToken}`);
  await expect(page).toHaveURL(`${localURL}/`);
  await expect(page.getByRole("heading", { name: "Dashboard" })).toBeVisible();
  await expect(page.getByRole("button", { name: "run-seeded" })).toBeVisible();
  await expect(page.getByTestId("kpi-card")).toHaveCount(4);
  await expect(page.getByRole("img", { name: "Run state distribution" })).toBeVisible();
  await expect(page.getByRole("region", { name: "Live activity" })).toBeVisible();
  await expectNoSeriousAxeFindings(page);

  const invalidCSRF = await page.evaluate(async () => {
    const response = await fetch("/api/v1/config/validate", {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json", "X-CSRF-Token": "wrong" },
      body: JSON.stringify({
        repo_root: "C:\\workspace\\fixture", task: "Blocked request",
        provider: "mock", model: "mock-v1", endpoint: "",
        confirm_custom_endpoint: false, profile: "supervised",
      }),
    });
    return response.status;
  });
  expect(invalidCSRF).toBe(403);

  await page.getByRole("button", { name: /New Run/ }).click();
  await expect(page.getByRole("heading", { name: "New Run" })).toBeVisible();
  await fillRun(page, "Repair the failing deterministic check");
  await expect(page.getByRole("heading", { name: "run-created-1" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Approval required" })).toBeVisible();
  await expect(page.getByRole("region", { name: "Read-only diff" })).toContainText("return a + b");
  await expect(page.getByText("Timestamp unavailable")).toHaveCount(0);
  await expect(page.getByText("Decisions 2 / 30")).toBeVisible();
  await expect(page.getByText("Mutations 1 / 5").first()).toBeVisible();
  await page.getByRole("button", { name: "Approve once" }).click();
  await expect(page.getByText("VALIDATING").first()).toBeVisible();
  await expect(page.getByRole("heading", { name: "Approval required" })).toHaveCount(0);

  await page.getByRole("button", { name: /New Run/ }).click();
  await fillRun(page, "Reject and stop the second bounded run");
  await expect(page.getByRole("heading", { name: "run-created-2" })).toBeVisible();
  await page.getByRole("button", { name: "Reject" }).click();
  await expect(page.getByText("DECIDING").first()).toBeVisible();
  await expect(page.getByText("APPROVAL_REJECTED")).toBeVisible();
  await page.getByRole("button", { name: "Cancel run" }).click();
  await expect(page.getByText("STOPPED").first()).toBeVisible();
  await expect(page.getByText("USER_CANCELLED")).toBeVisible();

  await page.getByRole("button", { name: /Credentials/ }).click();
  await expect(page.getByRole("heading", { name: "Credentials" })).toBeVisible();
  await expect(page.getByText("Not configured").first()).toBeVisible();
  await page.getByLabel("Secret").fill("one-time-secret");
  await page.getByRole("button", { name: "Add credential" }).click();
  await expect(page.getByLabel("Secret")).toHaveValue("");
  await expect(page.getByText("Configured").first()).toBeVisible();
  await expect(page.getByRole("button", { name: "Update credential" })).toBeVisible();
  await page.getByRole("button", { name: "Clear credential" }).click();
  await expect(page.getByText("Not configured").first()).toBeVisible();
  await expect(page.getByRole("button", { name: "Add credential" })).toBeVisible();
  await expectNoSeriousAxeFindings(page);
});

test("Go demo router exposes only fixed SIMULATED data and prunes mutations", async ({ page, request }) => {
  await page.goto(`${demoURL}/`);
  await expect(page.getByRole("heading", { name: "Demo Gallery" })).toBeVisible();
  await page.getByRole("button", { name: "Open SIMULATED demo" }).click();
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
