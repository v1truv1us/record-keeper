import http from 'node:http';
import https from 'node:https';
import { spawn } from 'node:child_process';

const port = Number(process.env.PORT || 3000);
const webhookUrl = process.env.NOTIFICATION_WEBHOOK || '';
const coolifyToken = process.env.COOLIFY_API_TOKEN || '';
const coolifyAppUuid = process.env.COOLIFY_APP_UUID || '';
const coolifyBase = process.env.COOLIFY_BASE_URL || 'https://coolify.fergify.work';

async function notify(result) {
	if (!webhookUrl) return;
	const payload = JSON.stringify({
		content: `🚨 **AudioFile smoke test FAILED** at ${result.finishedAt}\n\`\`\`${(result.output || '').slice(-500)}\`\`\``,
	});
	try {
		const url = new URL(webhookUrl);
		const mod = url.protocol === 'https:' ? https : http;
		await new Promise((resolve, reject) => {
			const req = mod.request(url, { method: 'POST', headers: { 'Content-Type': 'application/json', 'Content-Length': Buffer.byteLength(payload) } }, (res) => { res.resume(); res.on('end', resolve); });
			req.on('error', reject);
			req.write(payload);
			req.end();
		});
		console.log('Notification sent');
	} catch (e) {
		console.error('Notification failed:', e.message);
	}
}

async function rollback() {
	if (!coolifyToken || !coolifyAppUuid) return;
	try {
		const url = new URL(`${coolifyBase}/api/v1/deployments?uuid=${coolifyAppUuid}`);
		await new Promise((resolve, reject) => {
			const req = https.request(url, { headers: { Authorization: `Bearer ${coolifyToken}`, Accept: 'application/json' } }, (res) => {
				let body = '';
				res.on('data', (c) => body += c);
				res.on('end', () => {
					console.log('Rollback check:', body.slice(0, 200));
					resolve();
				});
			});
			req.on('error', reject);
			req.end();
		});
	} catch (e) {
		console.error('Rollback failed:', e.message);
	}
}
let lastRun = { status: 'never-run', code: null, output: '', finishedAt: null };
let running = false;

function runSmoke() {
	if (running) return Promise.resolve({ status: 'already-running' });
	running = true;
	lastRun = { status: 'running', code: null, output: '', finishedAt: null };
	return new Promise((resolve) => {
		const child = spawn('node', ['scripts/production-smoke.mjs'], {
			cwd: process.cwd(),
			env: process.env,
		});
		child.stdout.on('data', (chunk) => { lastRun.output += chunk; });
		child.stderr.on('data', (chunk) => { lastRun.output += chunk; });
		child.on('close', async (code) => {
			lastRun = {
				...lastRun,
				status: code === 0 ? 'passed' : 'failed',
				code,
				finishedAt: new Date().toISOString(),
			};
			running = false;
			if (code !== 0) {
				await notify(lastRun);
				await rollback();
			}
			resolve(lastRun);
		});
	});
}

const server = http.createServer(async (req, res) => {
	if (req.url === '/health') {
		res.writeHead(200, { 'Content-Type': 'application/json' });
		res.end(JSON.stringify({ ok: true, running, lastRun }));
		return;
	}
	if (req.url === '/run' && req.method === 'POST') {
		const result = await runSmoke();
		res.writeHead(result.status === 'failed' ? 500 : 200, { 'Content-Type': 'application/json' });
		res.end(JSON.stringify(result));
		return;
	}
	res.writeHead(404);
	res.end('not found');
});

const intervalMs = Number(process.env.SMOKE_INTERVAL_MS || 30 * 60 * 1000); // default 30min

server.listen(port, '0.0.0.0', () => {
	console.log(`Smoke runner listening on ${port} (interval: ${intervalMs}ms)`);
	runSmoke();
	setInterval(() => runSmoke(), intervalMs);
});
