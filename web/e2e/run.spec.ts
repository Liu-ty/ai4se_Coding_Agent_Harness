import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";

async function expectNoSeriousAxeFindings(page: Page) {
  const results = await new AxeBuilder({ page }).analyze();
  const severe = results.violations.filter(({ impact }) =>
    impact === "serious" || impact === "critical");
  expect(severe, severe.map(({ id, help }) => `${id}: ${help}`).join("\n")).toEqual([]);
}

test("SIMULATED feedback-loop exposes interception, feedback, changed action, success, and diff", async ({ page }) => {
  await page.goto("/?demo=1");
  await expect(page.getByRole("heading", { name: "Demo Gallery" })).toBeVisible();
  await expectNoSeriousAxeFindings(page);
  await page.getByRole("button", { name: "Open SIMULATED demo" }).click();
  await expect(page.getByText("POLICY_DENIED")).toBeVisible();
  await expect(page.getByText("TEST_FAILURE")).toBeVisible();
  await expect(page.getByText("ACTION_CHANGED")).toBeVisible();
  await expect(page.getByText("SUCCEEDED", { exact: true }).first()).toBeVisible();
  await expect(page.getByRole("region", { name: "Read-only diff" })).toContainText("return a + b");
  await expectNoSeriousAxeFindings(page);
});

test("keyboard-only supervised draft opens and closes exact one-time approval", async ({ page }) => {
  await page.goto("/");
  await page.keyboard.press("Tab");
  await page.keyboard.press("Tab");
  await page.keyboard.press("Tab");
  await page.keyboard.press("Enter");
  await expect(page.getByRole("heading", { name: "New Run" })).toBeVisible();
  await page.keyboard.press("Tab");
  await page.keyboard.type("C:\\workspace\\fixture");
  await page.keyboard.press("Tab");
  await page.keyboard.type("Repair the failing deterministic check");
  await page.keyboard.press("Tab");
  await page.keyboard.type("mock");
  await page.keyboard.press("Tab");
  await page.keyboard.type("mock-v1");
  await page.keyboard.press("Tab");
  await page.keyboard.press("Tab");
  await page.keyboard.press("Enter");
  await expect(page.getByRole("heading", { name: "Approval required" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Approve once" })).toBeVisible();
  await expect(page.getByText(/permanent rule/i)).toBeVisible();
  await page.keyboard.press("Tab");
  await page.keyboard.press("Enter");
  await expect(page.getByRole("heading", { name: "Approval required" })).toHaveCount(0);
  await expectNoSeriousAxeFindings(page);
});

test("all primary pages have zero serious or critical axe findings", async ({ page }) => {
  await page.goto("/");
  await expectNoSeriousAxeFindings(page);
  for (const name of ["New Run", "Credentials", "Demo Gallery"]) {
    await page.getByRole("button", { name: new RegExp(name, "i") }).click();
    await expect(page.getByRole("heading", { name })).toBeVisible();
    await expectNoSeriousAxeFindings(page);
  }
});
