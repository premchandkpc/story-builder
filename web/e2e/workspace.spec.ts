import { test, expect, type Page } from "@playwright/test";

async function createStory(page: Page, name: string) {
  await page.goto("/");
  await page.getByLabel("API bearer token").fill("playwright-token");
  await page.getByRole("button", { name: "Connect", exact: true }).click();
  await page.getByLabel("New story title").fill(name);
  await page.getByRole("button", { name: "Create map" }).click();
  await expect(page.getByRole("heading", { name })).toBeVisible();
}

test("character versions and scene-owned drafts survive the full authoring loop", async ({
  page,
}, testInfo) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await createStory(page, "The Locked Door");
  await page.getByRole("button", { name: "Characters", exact: true }).click();
  await page.getByLabel("Name", { exact: true }).fill("Mira");
  await page.getByLabel("motivation", { exact: true }).fill("Find her brother");
  await page.getByLabel(/Initial knowledge/i).fill("The door is locked");
  await page.getByLabel(/Author secrets/i).fill("Her brother has the key");
  await page
    .getByRole("button", { name: "Create character", exact: true })
    .click();
  await expect(page.getByText(/Saved version 1\./)).toBeVisible();
  await page
    .getByLabel("motivation", { exact: true })
    .fill("Protect her brother");
  await page.getByRole("button", { name: "Save new version" }).click();
  await expect(page.getByText(/Saved version 2\./)).toBeVisible();
  await page.getByRole("button", { name: "+ Add scene" }).click();
  await page.getByText("Untitled scene 1", { exact: true }).click();
  await page
    .getByRole("button", { name: "Generate draft", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "Reject", exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "+ Add scene" }).click();
  await page.getByText("Untitled scene 2", { exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Accept into story", exact: true }),
  ).toHaveCount(0);
  await page.getByText("Untitled scene 1", { exact: true }).click();
  await page
    .getByRole("button", { name: "Generate draft", exact: true })
    .click();
  await page.getByRole("button", { name: "Reject", exact: true }).click();
  await expect(
    page.getByText("Draft rejected. The story is unchanged."),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Generate draft", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Accept into story", exact: true })
    .click();
  await expect(page.getByText("Accepted prose", { exact: true })).toBeVisible();
  await expect(page.getByLabel("Title", { exact: true })).toBeDisabled();
  await expect(
    page.getByRole("button", { name: "Generate draft", exact: true }),
  ).toBeDisabled();
  expect(errors).toEqual([]);
  await page.screenshot({
    path: testInfo.outputPath("desktop.png"),
    fullPage: true,
  });
});

test("character editor remains accessible on a phone", async ({
  page,
}, testInfo) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await createStory(page, "Pocket story");
  await page.getByRole("button", { name: "Characters", exact: true }).click();
  await page.getByLabel("Name", { exact: true }).fill("Arun");
  await page
    .getByRole("button", { name: "Create character", exact: true })
    .click();
  await expect(page.getByText(/Saved version 1\./)).toBeVisible();
  await page.getByRole("button", { name: "+ Add scene" }).click();
  await expect(
    page.getByText("Untitled scene 1", { exact: true }),
  ).toBeVisible();
  await page.getByText("Untitled scene 1", { exact: true }).click();
  await expect(page.getByLabel("Title", { exact: true })).toHaveValue(
    "Untitled scene 1",
  );
  await page.screenshot({
    path: testInfo.outputPath("mobile.png"),
    fullPage: true,
  });
});
