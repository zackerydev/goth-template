import { test, expect } from '@playwright/test';

// One worker and a separate server keep the shared demo isolated from local development.
test.beforeEach(async ({ page, request }) => {
  await request.post('/lab/reset');
  page.on('pageerror', error => { throw error; });
});

async function snapshot(page, name) {
  // Only the server's wall-clock prefix is variable; preserve the event text and layout.
  await page.locator('.event p').evaluateAll(nodes => {
    for (const node of nodes) node.textContent = node.textContent.replace(/^\d{2}:\d{2}:\d{2}/, '12:00:00');
  });
  await expect(page).toHaveScreenshot(`${name}.png`, {
    fullPage: false,
    mask: [page.locator('.request-time')],
  });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
}

for (const [name, path] of Object.entries({
  board: '/lab',
  list: '/lab?view=list',
  editor: '/lab?id=1',
  patterns: '/lab?view=patterns&lesson=validation',
  activity: '/lab?view=activity',
})) {
  test(`${name} layout`, async ({ page }) => {
    await page.goto(path);
    await expect(page.locator('#workspace')).toBeVisible();
    if (name === 'editor') await expect(page.locator('#editor')).toHaveJSProperty('open', true);
    await snapshot(page, name);
    if (name === 'board') {
      const card = await page.locator('.task-card').first().boundingBox();
      expect(card.y).toBeLessThan(page.viewportSize().width > 760 ? 360 : 600);
    }
  });
}

test('editing keeps the board stable, validates inline, and restores keyboard focus', async ({ page }) => {
  await page.goto('/lab');
  const before = await page.locator('#task-1').boundingBox();
  await page.locator('#task-1').click();
  await expect(page).toHaveURL(/id=1/);
  await expect(page.locator('#task-title')).toBeFocused();
  expect(await page.locator('#editor').evaluate(node => node.matches(':modal'))).toBe(true);
  for (let step = 0; step < 12; step++) {
    await page.keyboard.press('Tab');
    // Native dialogs may yield Tab focus to browser chrome (BODY), never to background controls.
    const focusRegion = await page.evaluate(() => document.activeElement.closest('#editor')?.id ?? document.activeElement.tagName);
    expect(['editor', 'BODY']).toContain(focusRegion);
  }
  const after = await page.locator('#task-1').boundingBox();
  expect(after.width).toBe(before.width);
  expect(after.x).toBe(before.x);
  await page.locator('#task-title').fill('x');
  const invalid = page.waitForResponse(response => response.url().includes('/lab/task?') && response.status() === 422);
  await page.getByRole('button', { name: 'Save changes' }).click();
  await invalid;
  await expect(page.getByRole('alert')).toContainText('between 3 and 100');
  await expect(page.locator('#task-title')).toHaveValue('x');
  await expect(page.locator('#task-title')).toBeFocused();
  await snapshot(page, 'validation');
  await page.locator('#task-title').fill('Ship the accessible workspace');
  await page.locator('#task-status').selectOption('Done');
  await page.getByRole('button', { name: 'Save changes' }).click();
  await expect(page.locator('#editor')).toHaveCount(0);
  await expect(page.getByRole('region', { name: 'Done tasks' }).locator('#task-1')).toContainText('Ship the accessible workspace');
  await expect(page.locator('#notice')).toContainText('Saved: Ship the accessible workspace');
  await expect(page.locator('#task-1')).toBeFocused();
  await page.locator('#task-1').click();
  await expect(page.locator('#task-title')).toBeFocused();
  await page.keyboard.press('Escape');
  await expect(page.locator('#editor')).toHaveCount(0);
  await expect(page.locator('#task-1')).toBeFocused();
});

test('search preserves focus, combines filters, clears, and shows an empty state', async ({ page }) => {
  await page.goto('/lab');
  // Detect accidental full document navigation during htmx interactions.
  await page.evaluate(() => { window.testDocumentMarker = 'same-document'; });
  await page.getByRole('searchbox', { name: 'Search tasks' }).fill('keyboard');
  await expect(page.locator('.task-card')).toHaveCount(1);
  await expect(page.getByRole('searchbox')).toBeFocused();
  await page.getByLabel('Filter by status').selectOption('Done');
  await expect(page.locator('.task-card')).toHaveCount(0);
  await expect(page.locator('.filter-summary')).toContainText('0 matching tasks');
  await snapshot(page, 'empty-search');
  await page.getByRole('link', { name: 'Clear filters' }).click();
  await expect(page.locator('.task-card')).toHaveCount(9);
  await expect(page.getByRole('searchbox')).toHaveValue('');
  expect(await page.evaluate(() => window.testDocumentMarker)).toBe('same-document');
});

test('bulk selection has feedback and the inspector records actual responses', async ({ page }) => {
  await page.goto('/lab?view=list');
  await page.locator('input[name="task"][value="1"]').check();
  await page.locator('input[name="task"][value="3"]').check();
  await expect(page.locator('#selection-count')).toHaveText('2 selected');
  await page.getByRole('button', { name: 'Complete selected' }).click();
  await expect(page.locator('#task-1').locator('..').locator('..')).toContainText('Done');
  await expect(page.locator('#task-3').locator('..').locator('..')).toContainText('Done');
  await page.locator('#wire > summary').click();
  await expect(page.locator('.request-entry').first()).toContainText('POST');
  await expect(page.locator('.request-entry').first()).toContainText('200');
  await snapshot(page, 'inspector');
  await page.locator('.request-entry > summary').first().click();
  await expect(page.locator('.request-entry pre').first()).toContainText('<hx-partial');
  await page.getByRole('button', { name: 'Clear log' }).click();
  await expect(page.locator('.request-entry')).toHaveCount(0);
});

test('navigation, back, and direct task links keep a functioning shell', async ({ page }) => {
  await page.goto('/lab?view=list');
  await page.locator('#task-2').click();
  await expect(page.locator('#editor')).toBeVisible();
  await page.goBack();
  await expect(page.locator('#editor')).toHaveCount(0);
  await expect(page.getByRole('navigation', { name: 'Task view' }).getByRole('link', { name: 'List' })).toHaveAttribute('aria-current', 'page');
  await page.goForward();
  await expect(page.locator('#task-title')).toBeFocused();
  await page.reload();
  await expect(page.locator('#task-title')).toBeFocused();
  await page.keyboard.press('Escape');
  await expect(page.locator('#editor')).toHaveCount(0);
  await page.getByRole('navigation', { name: 'Main navigation' }).getByRole('link', { name: 'Patterns' }).click();
  await expect(page.locator('#view-heading')).toHaveText('Hypermedia patterns');
  await expect(page.getByRole('navigation', { name: 'Main navigation' }).getByRole('link', { name: 'Patterns' })).toHaveAttribute('aria-current', 'page');
  if (page.viewportSize().width <= 760) {
    await page.getByLabel('Choose a pattern').selectOption('bulk');
  } else {
    await page.getByRole('navigation', { name: 'Patterns', exact: true }).getByRole('link', { name: 'Bulk actions + native forms' }).click();
  }
  await expect(page).toHaveURL(/view=patterns.*lesson=bulk|lesson=bulk.*view=patterns/);
  await expect(page.locator('.lesson h2')).toHaveText('Bulk actions + native forms');
});

test('the open inspector leaves the final tasks reachable', async ({ page }) => {
  await page.goto('/lab');
  await page.locator('#wire > summary').click();
  await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
  const lowestCard = await page.locator('.task-card').evaluateAll(cards => Math.max(...cards.map(card => card.getBoundingClientRect().bottom)));
  const dock = await page.locator('#wire').boundingBox();
  expect(lowestCard).toBeLessThan(dock.y);
  await snapshot(page, 'board-scrolled');
});

test('inherited loading feedback works on navigation', async ({ page }) => {
  await page.goto('/lab');
  let release;
  const gate = new Promise(resolve => { release = resolve; });
  await page.route(/view=patterns/, async route => { await gate; await route.continue(); });
  await page.getByRole('navigation', { name: 'Main navigation' }).getByRole('link', { name: 'Patterns' }).click();
  await expect(page.locator('#request-loading')).toHaveClass(/htmx-request/);
  release();
  await expect(page.locator('#view-heading')).toHaveText('Hypermedia patterns');
  await expect(page.locator('#request-loading')).not.toHaveClass(/htmx-request/);
});

test('replacement searches cancel stale requests without reporting an error', async ({ page }) => {
  await page.goto('/lab');
  let release;
  const gate = new Promise(resolve => { release = resolve; });
  await page.route(/\/lab\?/, async route => {
    if (new URL(route.request().url()).searchParams.get('q') === 'key') await gate;
    await route.continue();
  });
  await page.getByRole('searchbox').fill('key');
  await expect(page.locator('#loading')).toHaveClass(/htmx-request/);
  await page.getByRole('searchbox').fill('keyboard');
  await expect(page.locator('.task-card')).toHaveCount(1);
  release();
  await expect(page.locator('#task-4')).toBeVisible();
  await expect(page.locator('#notice')).toBeEmpty();
  await expect(page.getByRole('searchbox')).toBeFocused();
});

test('a failed save preserves the draft and restores the submit button', async ({ page }) => {
  await page.goto('/lab?id=1');
  let release;
  const gate = new Promise(resolve => { release = resolve; });
  await page.route(/\/lab\/task\?/, async route => {
    await gate;
    await route.fulfill({ status: 503, contentType: 'text/plain', body: 'Temporarily unavailable' });
  });
  await page.locator('#task-title').fill('Keep this unsaved draft');
  await page.getByRole('button', { name: 'Save changes' }).click();
  await expect(page.getByRole('button', { name: 'Save changes' })).toBeDisabled();
  await expect(page.locator('#editor-loading')).toHaveClass(/htmx-request/);
  release();
  await expect(page.getByRole('alert')).toContainText('current view is unchanged');
  await expect(page.locator('#task-title')).toHaveValue('Keep this unsaved draft');
  await expect(page.getByRole('button', { name: 'Save changes' })).toBeEnabled();
  await snapshot(page, 'save-failure');
});

test('a task removed by another visitor does not destroy the editor', async ({ page, request }) => {
  await page.goto('/lab?id=1');
  await request.post('/lab/remove?id=1');
  await page.getByRole('button', { name: 'Save changes' }).click();
  await expect(page.getByRole('alert')).toContainText('no longer exists');
  await expect(page.locator('#task-title')).toHaveValue('Make the first five seconds unforgettable');
  await page.getByRole('link', { name: 'Close task' }).click();
  await expect(page.locator('#editor')).toHaveCount(0);
  await expect(page.locator('#task-1')).toHaveCount(0);
});

test('native forms still edit with JavaScript disabled', async ({ browser }) => {
  const context = await browser.newContext({ javaScriptEnabled: false });
  const page = await context.newPage();
  await page.goto('http://127.0.0.1:19169/lab?id=1');
  await page.locator('#task-title').fill('Saved without JavaScript');
  await page.getByRole('button', { name: 'Save changes' }).click();
  await expect(page.locator('#editor')).toHaveCount(0);
  await expect(page.locator('#task-1')).toContainText('Saved without JavaScript');
  await context.close();
});
