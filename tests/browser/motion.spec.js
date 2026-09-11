import { test, expect } from '@playwright/test';

test.use({ reducedMotion: 'no-preference' });

test.beforeEach(async ({ page, request }) => {
  await request.post('/lab/reset');
  page.on('pageerror', error => { throw error; });
  await page.addInitScript(() => {
    window.motion = { captures: 0, effects: [] };
    const start = document.startViewTransition.bind(document);
    document.startViewTransition = (...args) => {
      window.motion.captures++;
      return start(...args);
    };
    for (const type of ['animationstart', 'transitionrun']) {
      document.addEventListener(type, event => window.motion.effects.push(event.animationName ?? event.propertyName));
    }
  });
});

async function expectNoMotion(page) {
  expect(await page.evaluate(() => window.motion)).toEqual({ captures: 0, effects: [] });
  expect(await page.evaluate(() => document.getAnimations().length)).toBe(0);
}

test('navigation, editing, and feedback never animate', async ({ page }) => {
  await page.goto('/lab');
  await page.locator('#task-1').hover();
  await page.getByRole('navigation', { name: 'Task view' }).getByRole('link', { name: 'List' }).click();
  await expect(page.locator('table')).toBeVisible();
  await page.locator('#task-1').click();
  await expect(page.locator('#task-title')).toBeFocused();
  await expectNoMotion(page);
  await page.locator('#task-title').fill('x');
  await page.getByRole('button', { name: 'Save changes' }).click();
  await expect(page.getByRole('alert')).toContainText('between 3 and 100');
  await page.locator('#task-title').fill('Immediate updates');
  await page.getByRole('button', { name: 'Save changes' }).click();
  await expect(page.locator('#editor')).toHaveCount(0);
  await expect(page.locator('#notice')).toContainText('Saved: Immediate updates');
  await expect(page.locator('#task-1')).toBeFocused();
  // Neither OS preference enables motion anymore.
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.getByRole('navigation', { name: 'Task view' }).getByRole('link', { name: 'Board' }).click();
  await expect(page.locator('.board')).toBeVisible();
  await page.emulateMedia({ reducedMotion: 'no-preference' });
  await page.locator('#task-1').click();
  await expect(page.locator('#task-title')).toBeFocused();
  await page.keyboard.press('Escape');
  await expect(page.locator('#editor')).toHaveCount(0);
  await expectNoMotion(page);
});

test('loading indicators appear immediately without htmx fades', async ({ page }) => {
  await page.goto('/lab');
  let release;
  const gate = new Promise(resolve => { release = resolve; });
  await page.route(/view=patterns/, async route => { await gate; await route.continue(); });
  await page.getByRole('navigation', { name: 'Main navigation' }).getByRole('link', { name: 'Patterns' }).click();
  await expect(page.locator('#request-loading')).toHaveClass(/htmx-request/);
  await expect(page.locator('#request-loading')).toHaveCSS('opacity', '1');
  await expectNoMotion(page);
  release();
  await expect(page.locator('#view-heading')).toHaveText('Hypermedia patterns');
  await expect(page.locator('#request-loading')).toHaveCSS('opacity', '0');
  await expectNoMotion(page);
});

test('rapid opening and closing preserves scroll and focus without animation', async ({ page }) => {
  await page.setViewportSize({ ...page.viewportSize(), height: 600 });
  await page.goto('/lab');
  await page.locator('#task-8').scrollIntoViewIfNeeded();
  const scroll = await page.evaluate(() => scrollY);
  const box = await page.locator('#task-8').boundingBox();
  for (let cycle = 0; cycle < 3; cycle++) {
    await page.locator('#task-8').click();
    await expect(page.locator('#task-title')).toBeFocused();
    expect(await page.evaluate(() => scrollY)).toBe(scroll);
    await page.keyboard.press('Escape');
    await expect(page.locator('#editor')).toHaveCount(0);
    await expect(page.locator('#task-8')).toBeFocused();
    expect(await page.evaluate(() => scrollY)).toBe(scroll);
    expect(await page.locator('#task-8').boundingBox()).toEqual(box);
  }
  await expectNoMotion(page);
});
