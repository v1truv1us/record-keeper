// Shared API configuration — reads from window at runtime so builds work anywhere.
const API_BASE: string =
	typeof window !== "undefined"
		? ((window as any).__API_BASE__ ?? "http://localhost:8080")
		: "http://localhost:8080";

export function apiUrl(path: string): string {
	return `${API_BASE}${path}`;
}

interface ApiFetchOptions extends RequestInit {
	/** Skip auth token and 401 redirect for public endpoints */
	public?: boolean;
}

/**
 * Fetch wrapper with optional auth. Set `public: true` for endpoints
 * that don't require authentication (e.g. release search, barcode scan).
 */
export async function apiFetch(path: string, init: ApiFetchOptions = {}): Promise<Response> {
	const isPublic = init.public ?? false;
	delete (init as any).public;

	const headers = new Headers(init.headers || {});

	if (!isPublic) {
		const { supabase } = await import('./supabase');
		const { data: { session } } = await supabase.auth.getSession();

		if (session?.access_token) {
			headers.set('Authorization', `Bearer ${session.access_token}`);
		}

		const res = await fetch(apiUrl(path), { ...init, headers });

		if (res.status === 401) {
			window.location.replace('/login');
			throw new Error('Session expired');
		}

		return res;
	}

	if (!headers.has('Content-Type') && init.body && typeof init.body === 'string') {
		headers.set('Content-Type', 'application/json');
	}

	return fetch(apiUrl(path), { ...init, headers });
}
