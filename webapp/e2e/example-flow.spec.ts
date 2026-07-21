import { expect, test } from "@playwright/test";
import { readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

test("bundled example stays local and walks Browse to Read", async ({ page }) => {
  const requests: Array<{ url: string; method: string; body: string | null }> = [];
  const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..", "dist");
  const contentTypes: Record<string, string> = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".wasm": "application/wasm" };
  await page.route("**/*", async (route) => {
    const request = route.request();
    requests.push({ url: request.url(), method: request.method(), body: request.postData() });
    const url = new URL(request.url());
    if (url.origin !== "http://127.0.0.1:4174") return route.abort();
    const relative = url.pathname === "/" ? "index.html" : url.pathname.slice(1);
    try {
      await route.fulfill({ body: await readFile(path.join(root, relative)), contentType: contentTypes[path.extname(relative)] ?? "application/octet-stream" });
    } catch {
      await route.fulfill({ status: 404, body: "not found" });
    }
  });

  await page.goto("/");
  await expect(page.getByText("Your trace is parsed in this tab and never uploaded.")).toBeVisible();
  await page.getByRole("button", { name: "300-event coding trace" }).click();

  await expect(page.getByRole("main", { name: "Browse trajectories" })).toBeVisible({ timeout: 15_000 });
  await expect(page.getByRole("option").first()).toContainText("coding-bugfix-rollout-01");
  await page.keyboard.press("Enter");
  await expect(page.getByRole("main", { name: "Read trajectory" })).toBeVisible();
  await expect(page.locator(".selection-address")).toContainText("#");

  expect(requests.some((request) => new URL(request.url).origin !== "http://127.0.0.1:4174")).toBe(false);
  expect(requests.some((request) => request.method !== "GET")).toBe(false);
  expect(requests.some((request) => request.url.includes("/api/") || request.body?.includes("coding-event-0000"))).toBe(false);
  expect(requests.map((request) => new URL(request.url).pathname)).toContain("/rlviz.wasm");
});
