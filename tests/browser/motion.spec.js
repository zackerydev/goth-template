import { test, expect } from '@playwright/test';

test.use({ reducedMotion: 'no-preference' });

test.beforeEach(async ({ page, request }) => {
  await request.post('/lab/reset');
  page.on('pageerror', error => { throw error; });
  await page.addInitScript(() => {
    const start = document.startViewTransition.bind(document);
    window.motion = { started: 0, finished: 0, errors: [], moves: [], holdEditor: false, paused: false, scrolls: [] };
    for (const event of ['htmx:before:request', 'htmx:after:settle', 'htmx:after:swap']) {
      document.addEventListener(event, () => window.motion.scrolls.push([event, scrollY]));
    }
    document.startViewTransition = (...args) => {
      window.motion.started++;
      const transition = start(...args);
      transition.ready.then(() => {
        window.motion.moves.push(document.getAnimations()
          .filter(animation => animation.animationName?.includes('task-'))
          .map(animation => ({ name: animation.animationName, transforms: animation.effect.getKeyframes().map(frame => frame.transform) })));
      }).catch(error => window.motion.errors.push(error.message));
      transition.finished.then(() => window.motion.finished++);
      return transition;
    };
    // Pause the actual live-element animation, not a document snapshot.
    document.addEventListener('htmx:after:swap', () => {
      const editor = document.getElementById('editor');
      if (!window.motion.holdEditor || !editor) return;
      for (const animation of editor.getAnimations()) {
        animation.pause();
        animation.currentTime = 60;
      }
      window.motion.holdEditor = false;
      window.motion.paused = true;
    });
  });
});

async function finished(page, count) {
  await expect.poll(() => page.evaluate(() => window.motion.finished)).toBe(count);
  expect(await page.evaluate(() => window.motion.errors)).toEqual([]);
}

async function pauseEditor(page, id) {
  await page.evaluate(() => { window.motion.holdEditor = true; window.motion.paused = false; });
  await page.locator(`#task-${id}`).click();
  await expect.poll(() => page.evaluate(() => window.motion.paused)).toBe(true);
}

test('views animate shared tasks but editing never captures the page', async ({ page }, testInfo) => {
  await page.goto('/lab');
  await page.getByRole('navigation', { name: 'Task view' }).getByRole('link', { name: 'List' }).click();
  await finished(page, 1);
  const move = await page.evaluate(() => window.motion.moves[0].find(animation => animation.name.includes('task-1')));
  expect(move.transforms[0]).not.toBe(move.transforms.at(-1));
  await pauseEditor(page, 1);
  const animations = await page.locator('#editor').evaluate(editor => editor.getAnimations().map(animation => animation.animationName));
  expect(animations).toEqual([testInfo.project.name === 'mobile' ? 'mobile-editor-in' : 'editor-in']);
  expect(await page.locator('#editor').evaluate(editor => editor.matches(':modal'))).toBe(true);
  expect(await page.evaluate(() => window.motion.started)).toBe(1);
  await expect(page).toHaveScreenshot('editor-enter-motion.png', { animations: 'allow' });
  await page.locator('#editor').evaluate(editor => editor.getAnimations().forEach(animation => animation.finish()));
  await expect(page.locator('#task-title')).toBeFocused();
  await page.keyboard.press('Escape');
  await expect(page.locator('#editor')).toHaveCount(0);
  await expect(page.locator('#task-1')).toBeFocused();
  await page.getByRole('navigation', { name: 'Task view' }).getByRole('link', { name: 'Board' }).click();
  await finished(page, 2);
  await page.locator('#task-3').click();
  await page.locator('#task-status').selectOption('Done');
  await page.getByRole('button', { name: 'Save changes' }).click();
  await expect(page.getByRole('region', { name: 'Done tasks' }).locator('#task-3')).toBeVisible();
  expect(await page.evaluate(() => window.motion.started)).toBe(2);
});

test('search, validation, and polling remain quiet', async ({ page }) => {
  await page.goto('/lab');
  await page.getByRole('searchbox').fill('keyboard');
  await expect(page.locator('.task-card')).toHaveCount(1);
  await page.locator('#task-4').click();
  await page.locator('#task-title').fill('x');
  await page.getByRole('button', { name: 'Save changes' }).click();
  await expect(page.getByRole('alert')).toContainText('between 3 and 100');
  await expect(page.locator('#task-title')).toBeFocused();
  // Validation must not replay the sheet entrance or capture the page.
  expect(await page.locator('#editor').evaluate(editor => editor.getAnimations().length)).toBe(0);
  expect(await page.evaluate(() => window.motion.started)).toBe(0);
  await page.keyboard.press('Escape');
  await expect(page.locator('#editor')).toHaveCount(0);
  await page.getByRole('navigation', { name: 'Main navigation' }).getByRole('link', { name: 'Activity' }).click();
  await finished(page, 1);
  const poll = page.waitForResponse(response => response.url().endsWith('/lab/pulse'));
  await page.getByRole('button', { name: 'Run preview' }).click();
  await finished(page, 2);
  await poll;
  await expect(page.getByRole('progressbar')).not.toHaveAttribute('value', '0');
  expect(await page.evaluate(() => window.motion.started)).toBe(2);
});

test('reduced motion disables both forms of animation and responds to changes', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.goto('/lab');
  await pauseEditor(page, 1);
  expect(await page.locator('#editor').evaluate(editor => editor.getAnimations().length)).toBe(0);
  expect(await page.evaluate(() => window.motion.started)).toBe(0);
  await page.keyboard.press('Escape');
  await expect(page.locator('#editor')).toHaveCount(0);
  await page.emulateMedia({ reducedMotion: 'no-preference' });
  await expect.poll(() => page.evaluate(() => htmx.config.transitions)).toBe(true);
  await page.getByRole('navigation', { name: 'Task view' }).getByRole('link', { name: 'List' }).click();
  await finished(page, 1);
  await pauseEditor(page, 1);
  expect(await page.locator('#editor').evaluate(editor => editor.getAnimations().length)).toBe(1);
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await expect.poll(() => page.locator('#editor').evaluate(editor => editor.getAnimations().length)).toBe(0);
  await expect.poll(() => page.evaluate(() => htmx.config.transitions)).toBe(false);
  await page.keyboard.press('Escape');
  await page.getByRole('navigation', { name: 'Task view' }).getByRole('link', { name: 'Board' }).click();
  await expect(page.locator('.board')).toBeVisible();
  expect(await page.evaluate(() => window.motion.started)).toBe(1);
});

async function backgroundBrightness(page) {
  const image = await page.screenshot({ clip: { x: 700, y: 16, width: 32, height: 32 }, animations: 'allow' });
  return page.evaluate(async bytes => {
    const bitmap = await createImageBitmap(new Blob([new Uint8Array(bytes)], { type: 'image/png' }));
    const context = new OffscreenCanvas(bitmap.width, bitmap.height).getContext('2d');
    context.drawImage(bitmap, 0, 0);
    const pixels = context.getImageData(0, 0, bitmap.width, bitmap.height).data;
    let sum = 0;
    for (let i = 0; i < pixels.length; i += 4) sum += (pixels[i] + pixels[i + 1] + pixels[i + 2]) / 3;
    bitmap.close();
    return sum / (pixels.length / 4);
  }, [...image]);
}

test('the live backdrop stays constant throughout the sheet entrance', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name === 'mobile', 'The full-width mobile sheet covers the background sample.');
  await page.goto('/lab');
  const before = await backgroundBrightness(page);
  await pauseEditor(page, 1);
  const samples = [];
  for (const time of [0, 30, 70, 130]) {
    await page.locator('#editor').evaluate((editor, time) => editor.getAnimations().forEach(animation => { animation.currentTime = time; }), time);
    samples.push(await backgroundBrightness(page));
  }
  await page.locator('#editor').evaluate(editor => editor.getAnimations().forEach(animation => animation.finish()));
  const settled = await backgroundBrightness(page);
  expect(settled).toBeLessThan(before - 40);
  for (const sample of samples) expect(Math.abs(sample - settled)).toBeLessThan(2);
  await page.keyboard.press('Escape');
  await expect(page.locator('#editor')).toHaveCount(0);
  expect(Math.abs(await backgroundBrightness(page) - before)).toBeLessThan(2);
  expect(await page.evaluate(() => window.motion.started)).toBe(0);
});

test('rapid opening and closing preserves the scrolled workspace', async ({ page }) => {
  await page.setViewportSize({ ...page.viewportSize(), height: 600 });
  await page.goto('/lab');
  await page.locator('#task-8').scrollIntoViewIfNeeded();
  const scroll = await page.evaluate(() => scrollY);
  const box = await page.locator('#task-8').boundingBox();
  for (let cycle = 0; cycle < 3; cycle++) {
    await pauseEditor(page, 8);
    await expect(page.locator('#task-title')).toBeFocused();
    expect(await page.evaluate(() => scrollY), JSON.stringify(await page.evaluate(() => window.motion.scrolls))).toBe(scroll);
    await page.keyboard.press('Escape');
    await expect(page.locator('#editor')).toHaveCount(0);
    await expect(page.locator('#task-8')).toBeFocused();
    expect(await page.evaluate(() => scrollY)).toBe(scroll);
    expect(await page.locator('#task-8').boundingBox()).toEqual(box);
  }
  expect(await page.evaluate(() => window.motion.started)).toBe(0);
});

test('browsers without the API still navigate and edit normally', async ({ page }) => {
  await page.goto('/lab');
  await page.evaluate(() => { document.startViewTransition = undefined; });
  await page.getByRole('navigation', { name: 'Task view' }).getByRole('link', { name: 'List' }).click();
  await expect(page.locator('table')).toBeVisible();
  await page.locator('#task-1').click();
  await expect(page.locator('#task-title')).toBeFocused();
  await page.locator('#task-title').fill('A working fallback');
  await page.getByRole('button', { name: 'Save changes' }).click();
  await expect(page.locator('#editor')).toHaveCount(0);
  await expect(page.locator('#task-1')).toContainText('A working fallback');
  expect(await page.evaluate(() => window.motion.started)).toBe(0);
});
