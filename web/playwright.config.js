import {defineConfig} from '@playwright/test';
export default defineConfig({
 testDir: '.', testMatch: 'browser.spec.js', workers: 1,
 use: {baseURL: 'http://127.0.0.1:18080', headless: true},
 webServer: {
   command: 'go run ./cmd/box-view -addr 127.0.0.1:18080 -db :memory: -demo',
   cwd: '..', url: 'http://127.0.0.1:18080', reuseExistingServer: false, timeout: 120000,
 },
});
