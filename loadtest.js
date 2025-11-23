import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
    // Сценарий нагрузки (Ramp-up pattern)
    stages: [
        { duration: '10s', target: 50 }, // Разгон: за 10 сек поднимаем до 50 пользователей
        { duration: '30s', target: 50 }, // Плато: держим 50 пользователей полминуты
        { duration: '10s', target: 0 },  // Завершение: плавно снижаем до 0
    ],

    // Критерии успеха (SLI из задания)
    thresholds: {
        // 95% запросов должны быть быстрее 300мс (SLI времени ответа)
        http_req_duration: ['p(95)<300'],
        // Ошибок должно быть меньше 0.1% (SLI успешности 99.9%)
        http_req_failed: ['rate<0.001'],
    },
};

// Адрес приложения
const BASE_URL = 'http://172.18.0.5:8080';

export function setup() {
    const teamName = `load_team_${Date.now()}`;
    const url = `${BASE_URL}/team/add`;

    // Генерируем 100 пользователей для команды
    const members = [];
    for (let i = 1; i <= 100; i++) {
        members.push({
            user_id: `lu_${i}`,
            username: `LoadUser_${i}`,
            is_active: true
        });
    }

    const payload = JSON.stringify({
        team_name: teamName,
        members: members
    });

    const params = { headers: { 'Content-Type': 'application/json' } };

    console.log(`Setting up team: ${teamName}...`);
    const res = http.post(url, payload, params);

    // Проверяем, что команда создалась
    check(res, {
        'Setup: Team created (200/201)': (r) => r.status === 200 || r.status === 201,
    });

    // Передаем список ID пользователей в основную фазу теста
    return { user_ids: members.map(m => m.user_id) };
}

// --- 2. ОСНОВНОЙ ТЕСТ (VU LOOP) ---
export default function (data) {
    const url = `${BASE_URL}/pullRequest/create`;

    // Генерируем уникальный ID для каждого PR, чтобы не ловить ошибку дубликата
    // __VU - номер виртуального пользователя, __ITER - номер итерации
    const prId = `pr-load-${__VU}-${__ITER}-${Date.now()}`;

    // Выбираем случайного автора из списка, который мы вернули из setup()
    const randomAuthor = data.user_ids[Math.floor(Math.random() * data.user_ids.length)];

    const payload = JSON.stringify({
        pull_request_id: prId,
        pull_request_name: "Load Test Feature",
        author_id: randomAuthor,
    });

    const params = { headers: { 'Content-Type': 'application/json' } };

    const res = http.post(url, payload, params);

    // Проверяем успех
    check(res, {
        'status is 201': (r) => r.status === 201,
    });

    // Имитируем небольшую паузу между действиями пользователя (100мс)
    sleep(0.1);
}