import http from 'k6/http'
import { check, sleep } from 'k6'
import { SharedArray } from 'k6/data'

export const options = {
	vus: 20, 
	duration: '30s',
	thresholds: {
		http_req_duration: ['p(95)<300'], 
		http_req_failed: ['rate<0.001'], 
	},
}

const BASE_URL = 'http://localhost:8080'

const users = new SharedArray('users', function () {
	return [
		{ id: 'u1', username: 'user1' },
		{ id: 'u2', username: 'user2' },
		{ id: 'u3', username: 'user3' },
		{ id: 'u4', username: 'user4' },
		{ id: 'u5', username: 'user5' },
	]
})

export function setup() {
	const payload = JSON.stringify({
		team_name: 'backend',
		members: users.map(u => ({
			user_id: u.id,
			username: u.username,
			is_active: true,
		})),
	})

	const res = http.post(`${BASE_URL}/team/add`, payload, {
		headers: { 'Content-Type': 'application/json' },
	})

	check(res, {
		'team created or already exists': r => r.status === 201 || r.status === 400,
	})

	return { teamName: 'backend' }
}

let prCounter = 0

export default function (data) {
	const author = users[0] 
	const prId = `pr-${__VU}-${prCounter++}`

	const createPayload = JSON.stringify({
		pull_request_id: prId,
		pull_request_name: `Test PR ${prId}`,
		author_id: author.id,
	})

	const resCreate = http.post(`${BASE_URL}/pullRequest/create`, createPayload, {
		headers: { 'Content-Type': 'application/json' },
	})

	check(resCreate, {
		'create PR success': r => r.status === 201,
	})

	const reviewer = users[1]
	const resReview = http.get(
		`${BASE_URL}/users/getReview?user_id=${reviewer.id}`
	)

	check(resReview, {
		'getReview ok': r => r.status === 200,
	})

	sleep(0.5)
}
