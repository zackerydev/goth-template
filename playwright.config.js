import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests/browser',
  outputDir: 'tmp/browser-results',
  snapshotPathTemplate: '{testDir}/snapshots/{platform}/{projectName}/{arg}{ext}',
  workers: 1,
  retries: 0,
  timeout: 30000,
  expect: { toHaveScreenshot: { animations: 'disabled', maxDiffPixelRatio: 0.001 } },
  updateSnapshots: process.env.UPDATE_VISUALS === '1' ? 'all' : 'none',
  reporter: [['list'], ['html', { outputFolder: 'tmp/browser-report', open: 'never' }]],
  use: {
    baseURL: 'http://127.0.0.1:19169',
    browserName: 'chromium',
    reducedMotion: 'reduce',
    locale: 'en-US',
    timezoneId: 'UTC',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [
    { name: 'desktop', use: { viewport: { width: 1440, height: 1000 } } },
    { name: 'mobile', use: { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true } },
  ],
  webServer: {
    command: 'mise exec -- go build -o tmp/visual-app ./cmd/app && APP_PORT=19169 ./tmp/visual-app',
    url: 'http://127.0.0.1:19169/lab',
    reuseExistingServer: false,
  },
});
