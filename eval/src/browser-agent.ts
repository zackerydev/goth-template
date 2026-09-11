import type { Task } from "./types.ts";

export type BrowserOutcome = {
  answer: number | null;
  turns: number;
  usedSearch: boolean;
  actions: number;
  trace: string[];
  error?: string;
};

export async function runScriptedBrowser(origin: string, task: Task): Promise<BrowserOutcome> {
  let playwright: typeof import("playwright");
  try {
    playwright = await import("playwright");
  } catch {
    return { answer: null, turns: 0, usedSearch: false, actions: 0, trace: [], error: "playwright not installed" };
  }

  const browser = await playwright.chromium.launch({ headless: true });
  const page = await browser.newPage();
  const trace: string[] = [];
  let actions = 0;
  let usedSearch = false;
  let answer: number | null = null;
  page.on("dialog", (dialog) => {
    void dialog.accept();
  });

  try {
    await page.goto(`${origin}/app/`, { waitUntil: "networkidle" });
    await page.getByRole("heading", { name: "Contacts" }).waitFor({ timeout: 10_000 });
    actions += 1;
    trace.push("opened /app/");

    switch (task.id) {
      case "update-dan":
        usedSearch = true;
        actions += await search(page, "Dan", trace);
        await page.getByRole("button", { name: "Edit" }).first().click();
        actions += 1;
        await page.getByLabel("Last Name").fill("Hibiki");
        await page.getByRole("button", { name: "Save" }).click();
        actions += 2;
        trace.push("updated Dan to Hibiki");
        break;
      case "create-ken":
        await page.getByRole("button", { name: "Add Contact" }).click();
        await page.getByLabel("First Name").fill("Ken");
        await page.getByLabel("Last Name").fill("Masters");
        await page.getByLabel("Email").fill("ken.masters@eval.test");
        await page.getByLabel("Phone").fill("555-0199");
        await page.getByRole("button", { name: "Save" }).click();
        actions += 6;
        trace.push("created Ken Masters");
        break;
      case "delete-chun-li":
        usedSearch = true;
        actions += await search(page, "Chun-Li", trace);
        await page.getByRole("button", { name: "Edit" }).first().click();
        await page.getByRole("button", { name: "Delete Contact" }).click();
        actions += 2;
        trace.push("deleted Chun-Li");
        break;
      case "count-ryu":
        usedSearch = true;
        actions += await search(page, "Ryu", trace);
        answer = await page.locator("tbody tr").count();
        trace.push(`counted ${answer} table rows`);
        break;
    }
  } catch (error) {
    await browser.close();
    return {
      answer,
      turns: actions,
      usedSearch,
      actions,
      trace,
      error: error instanceof Error ? error.message : String(error),
    };
  }

  await browser.close();
  return { answer, turns: actions, usedSearch, actions, trace };
}

async function search(page: import("playwright").Page, query: string, trace: string[]): Promise<number> {
  await page.getByLabel("Search Term").fill(query);
  await page.getByRole("button", { name: "Search" }).click();
  await page.waitForTimeout(200);
  trace.push(`searched ${query}`);
  return 2;
}
