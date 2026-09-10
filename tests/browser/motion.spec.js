import { test, expect } from '@playwright/test';

test.use({ reducedMotion: 'no-preference' });

test.beforeEach(async ({ page, request }) => {
  await request.post('/lab/reset');
  page.on('pageerror', error => { throw error; });
  // Observe the real browser API, including rejected captures (e.g. duplicate names).
  await page.addInitScript(() => {
    const start = document.startViewTransition.bind(document);
    window.motion = { started: 0, finished: 0, errors: [], frames: [], moves: [], hold: false, paused: false };
    document.startViewTransition = (...args) => {
      window.motion.started++;
      const transition = start(...args);
      transition.ready.then(() => {
        const animations = document.getAnimations().filter(animation => animation.animationName);
        window.motion.frames.push(animations.map(animation => animation.animationName));
        window.motion.moves.push(animations.filter(animation => animation.animationName.includes('task-')).map(animation => ({
          name: animation.animationName,
          transforms: animation.effect.getKeyframes().map(frame => frame.transform),
        })));
        if (window.motion.hold) {
          for (const animation of animations) {
            animation.pause();
            animation.currentTime = 100;
          }
          window.motion.hold = false;
          window.motion.paused = true;
        }
      }).catch(error => window.motion.errors.push(error.message));
      transition.finished.then(() => window.motion.finished++);
      return transition;
    };
  });
});

async function finished(page, count) {
  await expect.poll(() => page.evaluate(() => window.motion.finished)).toBe(count);
  expect(await page.evaluate(() => window.motion.errors)).toEqual([]);
}

test('native shared elements and editor entry/exit animate without breaking focus', async ({ page }, testInfo) => {
  await page.goto('/lab');
  await page.getByRole('navigation', { name: 'Task view' }).getByRole('link', { name: 'List' }).click();
  await finished(page, 1);
  expect(await page.evaluate(() => window.motion.frames.flat().some(name => name.includes('task-1')))).toBe(true);
  await page.evaluate(() => { window.motion.hold = true; });
  await page.locator('#task-1').click();
  await expect.poll(() => page.evaluate(() => window.motion.paused)).toBe(true);
  const sheetAnimation = testInfo.project.name === 'mobile' ? 'mobile-sheet-in' : 'sheet-in';
  expect(await page.evaluate(() => window.motion.frames.flat())).toContain(sheetAnimation);
  expect(await page.locator('#editor').evaluate(node => node.matches(':modal'))).toBe(true);
  await expect(page).toHaveScreenshot('editor-enter-motion.png', { animations: 'allow' });
  await page.evaluate(() => document.getAnimations().forEach(animation => animation.finish()));
  await finished(page, 2);
  await expect(page.locator('#task-title')).toBeFocused();
  await page.keyboard.press('Escape');
  await finished(page, 3);
  expect(await page.evaluate(() => window.motion.frames.flat())).toContain(testInfo.project.name === 'mobile' ? 'mobile-sheet-out' : 'sheet-out');
  await expect(page.locator('#task-1')).toBeFocused();
  await page.getByRole('navigation', { name: 'Task view' }).getByRole('link', { name: 'Board' }).click();
  await finished(page, 4);
  await page.locator('#task-3').click();
  await finished(page, 5);
  await page.locator('#task-status').selectOption('Done');
  await page.getByRole('button', { name: 'Save changes' }).click();
  await finished(page, 6);
  await expect(page.getByRole('region', { name: 'Done tasks' }).locator('#task-3')).toBeVisible();
  const move = await page.evaluate(() => window.motion.moves.at(-1).find(animation => animation.name.includes('task-3')));
  expect(move.transforms[0]).not.toBe(move.transforms.at(-1));
});

test('live search, validation, and polling do not start transitions', async ({ page }) => {
  await page.goto('/lab');
  await page.getByRole('searchbox').fill('keyboard');
  await expect(page.locator('.task-card')).toHaveCount(1);
  expect(await page.evaluate(() => window.motion.started)).toBe(0);
  await page.locator('#task-4').click();
  await finished(page, 1);
  await page.locator('#task-title').fill('x');
  await page.getByRole('button', { name: 'Save changes' }).click();
  await expect(page.getByRole('alert')).toContainText('between 3 and 100');
  await expect(page.locator('#task-title')).toBeFocused();
  expect(await page.evaluate(() => window.motion.started)).toBe(1);
  await page.keyboard.press('Escape');
  await finished(page, 2);
  await page.getByRole('navigation', { name: 'Main navigation' }).getByRole('link', { name: 'Activity' }).click();
  await finished(page, 3);
  const poll = page.waitForResponse(response => response.url().endsWith('/lab/pulse'));
  await page.getByRole('button', { name: 'Run preview' }).click();
  await finished(page, 4);
  await poll;
  await expect(page.getByRole('progressbar')).not.toHaveAttribute('value', '0');
  expect(await page.evaluate(() => window.motion.started)).toBe(4);
});

test('reduced motion disables native transitions, including preference changes', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.goto('/lab');
  await page.locator('#task-1').click();
  await expect(page.locator('#task-title')).toBeFocused();
  expect(await page.evaluate(() => window.motion.started)).toBe(0);
  await page.emulateMedia({ reducedMotion: 'no-preference' });
  await expect.poll(() => page.evaluate(() => htmx.config.transitions)).toBe(true);
  await page.keyboard.press('Escape');
  await finished(page, 1);
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await expect.poll(() => page.evaluate(() => htmx.config.transitions)).toBe(false);
  await page.locator('#task-1').click();
  await expect(page.locator('#task-title')).toBeFocused();
  expect(await page.evaluate(() => window.motion.started)).toBe(1);
});

test('browsers without the API still navigate and edit normally', async ({ page }) => {
  await page.goto('/lab');
  await page.evaluate(() => { document.startViewTransition = undefined; });
  await page.locator('#task-1').click();
  await expect(page.locator('#task-title')).toBeFocused();
  await page.locator('#task-title').fill('A working fallback');
  await page.getByRole('button', { name: 'Save changes' }).click();
  await expect(page.locator('#editor')).toHaveCount(0);
  await expect(page.locator('#task-1')).toContainText('A working fallback');
  expect(await page.evaluate(() => window.motion.started)).toBe(0);
});
