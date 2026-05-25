import { chromium, request } from '@playwright/test';

const baseURL = process.env.BASE_URL || 'https://audiofile.app';
const browserlessEndpoint = process.env.BROWSERLESS_WS_ENDPOINT;

function assert(condition, message) {
	if (!condition) throw new Error(message);
}

async function checkAPI() {
	const api = await request.newContext({ baseURL });
	try {
		const health = await api.get('/api/health');
		assert(health.status() === 200, `health returned ${health.status()}`);
		const healthBody = await health.json();
		assert(healthBody.status === 'ok', `health status was ${JSON.stringify(healthBody)}`);

		const search = await api.get('/api/releases/search?q=kind%20of%20blue%20miles%20davis');
		assert(search.status() === 200, `release search returned ${search.status()}`);
		const releases = await search.json();
		assert(Array.isArray(releases), 'release search did not return an array');
		assert(releases.length > 0, 'release search returned no results');
		assert(
			releases.some((release) =>
				/kind of blue/i.test(release.title ?? '') || /miles davis/i.test(release.artist ?? ''),
			),
			'release search returned no recognizable Kind of Blue / Miles Davis result',
		);

		const barcode = await api.post('/api/releases/scan', { data: { barcode: '018771210510' } });
		assert(barcode.status() === 200, `barcode scan returned ${barcode.status()}`);
		const barcodeBody = await barcode.json();
		assert(barcodeBody.barcode === '018771210510', `barcode response was ${JSON.stringify(barcodeBody)}`);
		assert(Array.isArray(barcodeBody.results), 'barcode scan results was not an array');
	} finally {
		await api.dispose();
	}
}

async function checkBrowser() {
	const browser = browserlessEndpoint
		? await chromium.connectOverCDP(browserlessEndpoint)
		: await chromium.launch();
	try {
		const page = await browser.newPage({ viewport: { width: 1280, height: 720 } });
		await page.goto(`${baseURL}/login/`);
		await page.getByRole('heading', { name: 'Welcome back' }).waitFor();
		await page.goto(`${baseURL}/signup/`);
		await page.getByRole('heading', { name: 'Start your collection' }).waitFor();
		await page.goto(`${baseURL}/collection/`);
		await page.waitForURL(/\/login\/?$/);
	} finally {
		await browser.close();
	}
}

await checkAPI();
await checkBrowser();
console.log(`Production smoke passed for ${baseURL}`);
