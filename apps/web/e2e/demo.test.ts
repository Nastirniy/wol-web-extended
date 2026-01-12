import { expect, test } from '@playwright/test';

test('home page has expected main content', async ({ page }) => {
	await page.goto('/home');
	await expect(page.locator('main')).toBeVisible();
});
