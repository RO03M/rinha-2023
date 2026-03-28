import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { SharedArray } from 'k6/data';
import { Counter, Rate, Trend } from 'k6/metrics';

// ─── Config ──────────────────────────────────────────────────────────────────

const BASE_URL = __ENV.BASE_URL || 'http://localhost:9999';
const SCENARIO = __ENV.SCENARIO || 'full';

// ─── Custom metrics ───────────────────────────────────────────────────────────

const createdIds = new SharedArray('ids', () => []);   // filled at runtime via state
const insertErrors = new Counter('insert_errors');
const searchErrors = new Counter('search_errors');
const getErrors = new Counter('get_errors');
const insertRate = new Rate('insert_success_rate');
const searchRate = new Rate('search_success_rate');
const insertLatency = new Trend('insert_latency', true);
const searchLatency = new Trend('search_latency', true);
const getLatency = new Trend('get_latency', true);

// ─── Test data ────────────────────────────────────────────────────────────────

const NOMES = [
	'Ana Silva', 'Bruno Souza', 'Carlos Oliveira', 'Daniela Santos', 'Eduardo Lima',
	'Fernanda Costa', 'Gabriel Alves', 'Helena Rocha', 'Igor Ferreira', 'Julia Martins',
	'Kevin Barbosa', 'Larissa Pereira', 'Marcos Ribeiro', 'Natalia Gomes', 'Otavio Carvalho',
	'Patricia Mendes', 'Rafael Nunes', 'Sabrina Teixeira', 'Thiago Cardoso', 'Vanessa Araujo',
];

const APELIDOS_BASE = [
	'ana', 'bru', 'carol', 'dani', 'edu', 'fer', 'gab', 'hel', 'igor', 'ju',
	'kev', 'lari', 'marc', 'nati', 'ota', 'pati', 'rafa', 'sabi', 'thi', 'van',
];

const STACKS = [
	['Go', 'Postgres'], ['Node', 'Redis'], ['Python', 'MongoDB'], ['Java', 'MySQL'],
	['Rust', 'SQLite'], ['TypeScript', 'Kafka'], ['Elixir', 'RabbitMQ'], ['Kotlin', 'Cassandra'],
	['C#', 'Redis', 'Docker'], ['PHP', 'MySQL', 'Nginx'],
	null, null, null,  // ~23% chance of null stack
];

const SEARCH_TERMS = [
	'Go', 'Node', 'Python', 'Java', 'Rust', 'TypeScript', 'Elixir',
	'Silva', 'Souza', 'ana', 'bru', 'redis', 'postgres',
];

// ─── Scenarios ────────────────────────────────────────────────────────────────

export const options = SCENARIO === 'stress'
	? {
		scenarios: {
			criacao: {
				executor: 'ramping-vus',
				startVUs: 0,
				stages: [
					{ duration: '10s', target: 50 },
					{ duration: '40s', target: 200 },
					{ duration: '10s', target: 0 },
				],
				exec: 'criacaoPessoas',
				tags: { scenario: 'criacao' },
			},
			consulta: {
				executor: 'ramping-vus',
				startVUs: 0,
				stages: [
					{ duration: '20s', target: 0 },  // wait for some inserts first
					{ duration: '20s', target: 50 },
					{ duration: '20s', target: 0 },
				],
				exec: 'consultaPessoas',
				tags: { scenario: 'consulta' },
			},
		},
		thresholds: {
			http_req_failed: ['rate<0.01'],
			insert_success_rate: ['rate>0.95'],
			search_success_rate: ['rate>0.99'],
			insert_latency: ['p(95)<500'],
			search_latency: ['p(95)<1000'],
		},
	}
	: SCENARIO === 'consulta'
		? {
			scenarios: {
				consulta: {
					executor: 'constant-vus',
					vus: 50,
					duration: '30s',
					exec: 'consultaPessoas',
				},
			},
			thresholds: {
				search_success_rate: ['rate>0.99'],
				search_latency: ['p(95)<500'],
			},
		}
		: /* full / default */ {
			scenarios: {
				criacao: {
					executor: 'constant-vus',
					vus: 50,
					duration: '30s',
					exec: 'criacaoPessoas',
					tags: { scenario: 'criacao' },
				},
				consulta: {
					executor: 'constant-vus',
					vus: 20,
					duration: '30s',
					exec: 'consultaPessoas',
					startTime: '5s',
					tags: { scenario: 'consulta' },
				},
				get_por_id: {
					executor: 'constant-vus',
					vus: 10,
					duration: '30s',
					exec: 'getPorId',
					startTime: '5s',
					tags: { scenario: 'get' },
				},
			},
			thresholds: {
				http_req_failed: ['rate<0.01'],
				insert_success_rate: ['rate>0.90'],
				search_success_rate: ['rate>0.99'],
				insert_latency: ['p(95)<500'],
				search_latency: ['p(95)<500'],
				get_latency: ['p(95)<500'],
			},
		};

// ─── State shared across VUs (via module-level array) ─────────────────────────

// k6 doesn't have true shared mutable state across VUs, so we store IDs
// in a module-level array appended during the test. Each VU reads its own slice.
const _ids = [];

// ─── Helpers ─────────────────────────────────────────────────────────────────

function randomInt(min, max) {
	return Math.floor(Math.random() * (max - min + 1)) + min;
}

function uniqueNickname() {
	const base = APELIDOS_BASE[randomInt(0, APELIDOS_BASE.length - 1)];
	return `${base}_${Date.now()}_${randomInt(1000, 9999)}`;
}

function randomPessoa() {
	const nome = NOMES[randomInt(0, NOMES.length - 1)];
	const stack = STACKS[randomInt(0, STACKS.length - 1)];
	return {
		nome,
		apelido: uniqueNickname(),
		nascimento: `${randomInt(1970, 2000)}-${String(randomInt(1, 12)).padStart(2, '0')}-${String(randomInt(1, 28)).padStart(2, '0')}`,
		stack,
	};
}

// ─── Exported scenario functions ─────────────────────────────────────────────

export function criacaoPessoas() {
	const pessoa = randomPessoa();
	const payload = JSON.stringify(pessoa);

	const start = Date.now();
	const res = http.post(`${BASE_URL}/pessoas`, payload, {
		headers: { 'Content-Type': 'application/json' },
	});
	insertLatency.add(Date.now() - start);

	const ok = check(res, {
		'POST /pessoas → 201': (r) => r.status === 201,
	});

	insertRate.add(ok);

	if (!ok) {
		insertErrors.add(1);
		// 422 for duplicate nickname is expected; log others
		if (res.status !== 422) {
			console.warn(`POST /pessoas failed: ${res.status} body=${res.body?.substring(0, 200)}`);
		}
		return;
	}

	// Capture the Location header so other VUs can fetch this person
	const location = res.headers['Location'];
	if (location) {
		const id = location.split('/').pop();
		_ids.push(id);
	}
}

export function consultaPessoas() {
	const term = SEARCH_TERMS[randomInt(0, SEARCH_TERMS.length - 1)];

	const start = Date.now();
	const res = http.get(`${BASE_URL}/pessoas?t=${encodeURIComponent(term)}`);
	searchLatency.add(Date.now() - start);

	const ok = check(res, {
		'GET /pessoas?t → 200': (r) => r.status === 200,
		'response is JSON array': (r) => {
			try {
				const body = JSON.parse(r.body);
				return Array.isArray(body);
			} catch {
				return false;
			}
		},
	});

	searchRate.add(ok);
	if (!ok) {
		searchErrors.add(1);
		console.warn(`GET /pessoas?t=${term} failed: ${res.status} body=${res.body?.substring(0, 200)}`);
	}
}

export function getPorId() {
	if (_ids.length === 0) {
		sleep(0.2);
		return;
	}

	const id = _ids[randomInt(0, _ids.length - 1)];

	const start = Date.now();
	const res = http.get(`${BASE_URL}/pessoas/${id}`);
	getLatency.add(Date.now() - start);

	const ok = check(res, {
		'GET /pessoas/:id → 200': (r) => r.status === 200,
		'body has id field': (r) => {
			try {
				return JSON.parse(r.body).id === id;
			} catch {
				return false;
			}
		},
	});

	if (!ok) {
		getErrors.add(1);
		console.warn(`GET /pessoas/${id} failed: ${res.status}`);
	}
}

// ─── Teardown: validate final count ──────────────────────────────────────────

export function teardown() {
	const res = http.get(`${BASE_URL}/contagem-pessoas`);
	const total = parseInt(res.body, 10);

	check(res, {
		'GET /contagem-pessoas → 200': (r) => r.status === 200,
		'count is a number': () => !isNaN(total),
		'count > 0': () => total > 0,
	});

	console.log(`\n✓ Final person count: ${total}`);
	console.log(`  IDs captured by this VU run: ${_ids.length}`);
}
