import http from 'node:http';
import { spawn } from 'node:child_process';

const port = Number(process.env.PORT || 3000);
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
		child.on('close', (code) => {
			lastRun = {
				...lastRun,
				status: code === 0 ? 'passed' : 'failed',
				code,
				finishedAt: new Date().toISOString(),
			};
			running = false;
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
