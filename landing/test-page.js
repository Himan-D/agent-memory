import { chromium } from 'playwright';

const browser = await chromium.launch({ headless: true });
const page = await browser.newPage();

const errors = [];
page.on('console', msg => {
  if (msg.type() === 'error') errors.push(msg.text());
});
page.on('pageerror', err => errors.push(err.message));

try {
  await page.goto('http://localhost:8080', { waitUntil: 'networkidle', timeout: 10000 });
  await page.waitForTimeout(2000);
  
  console.log('Title:', await page.title());
  console.log('Body length:', (await page.evaluate(() => document.body.innerHTML)).length);
  console.log('Errors:', errors.length > 0 ? errors : 'None!');
} catch (e) {
  console.log('Error:', e.message);
}

await browser.close();