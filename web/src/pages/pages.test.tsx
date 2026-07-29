import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "vitest";
import { Credentials } from "./Credentials";
import { DemoGallery } from "./DemoGallery";
import { NewRun } from "./NewRun";

it("clears credential secret state immediately after submit starts", async () => {
  let resolveSubmit!: () => void;
  const submit = vi.fn(() => new Promise<void>((resolve) => { resolveSubmit = resolve; }));
  render(<Credentials onSave={submit} />);
  const input = screen.getByLabelText("Secret");
  await userEvent.type(input, "canary-value");
  await userEvent.click(screen.getByRole("button", { name: "Save credential" }));
  expect(input).toHaveValue("");
  resolveSubmit();
});

it("demo gallery labels every scenario and hides unavailable controls", () => {
  render(<DemoGallery fixedRuns={["feedback-loop"]} onOpen={vi.fn()} />);
  expect(screen.getAllByText("SIMULATED").length).toBeGreaterThan(1);
  expect(screen.queryByRole("button", { name: /new run/i })).toBeNull();
  expect(screen.queryByRole("link", { name: /credentials/i })).toBeNull();
});

it("supports a keyboard-only supervised run draft", async () => {
  const onCreate = vi.fn();
  const user = userEvent.setup();
  render(<NewRun onCreate={onCreate} />);
  await user.tab();
  await user.keyboard("C:\\repo");
  await user.tab();
  await user.keyboard("Repair the failing check");
  await user.tab();
  await user.keyboard("mock");
  await user.tab();
  await user.keyboard("mock-v1");
  await user.tab();
  await user.tab();
  expect(document.activeElement).toBe(screen.getByRole("button", { name: "Create supervised draft" }));
  await user.keyboard("{Enter}");
  expect(onCreate).toHaveBeenCalledWith(expect.objectContaining({ profile: "supervised" }));
});
