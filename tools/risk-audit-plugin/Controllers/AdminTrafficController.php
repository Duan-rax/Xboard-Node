<?php

namespace Plugin\RiskAudit\Controllers;

use App\Http\Controllers\Controller;
use App\Models\User;
use Carbon\CarbonImmutable;
use Illuminate\Database\Query\Builder;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Cookie;
use Illuminate\Support\Facades\Crypt;
use Illuminate\Support\Facades\Hash;
use Illuminate\Support\Facades\Validator;

class AdminTrafficController extends Controller
{
    public function loginPage(Request $request)
    {
        if ($this->isAuditAdmin($request)) {
            return redirect($this->auditURL());
        }

        return view('RiskAudit::login', [
            'loginAction' => $this->auditURL('/login'),
            'error' => $request->query('error'),
            'email' => $request->query('email', ''),
        ]);
    }

    public function login(Request $request)
    {
        $origin = $request->header('Origin');
        if ($origin && parse_url($origin, PHP_URL_HOST) !== $request->getHost()) {
            abort(403, 'Invalid request origin');
        }

        $validator = Validator::make($request->all(), [
            'email' => ['required', 'email', 'max:255'],
            'password' => ['required', 'string', 'max:255'],
        ]);
        if ($validator->fails()) {
            return redirect($this->auditURL('/login') . '?error=invalid&email=' . rawurlencode((string) $request->input('email', '')));
        }
        $credentials = $validator->validated();
        $user = User::query()->where('email', strtolower(trim($credentials['email'])))->first();

        if (!$user || !$user->is_admin || !Hash::check($credentials['password'], $user->password)) {
            return redirect($this->auditURL('/login') . '?error=invalid&email=' . rawurlencode($credentials['email']));
        }

        $payload = Crypt::encryptString(json_encode([
            'user_id' => $user->id,
            'expires_at' => now()->addHours(8)->timestamp,
        ], JSON_THROW_ON_ERROR));

        return redirect($this->auditURL())->withCookie(
            cookie('risk_audit_admin', $payload, 480, '/', null, true, true, false, 'lax')
        );
    }

    public function logout(Request $request)
    {
        return redirect($this->auditURL('/login'))->withCookie(Cookie::forget('risk_audit_admin'));
    }

    public function page()
    {
        $securePath = admin_setting('secure_path', admin_setting('frontend_admin_path', hash('crc32b', config('app.key'))));

        return view('RiskAudit::admin', [
            'apiBase' => '/api/v2/' . $securePath . '/risk-audit',
            'adminPath' => '/' . $securePath,
        ]);
    }

    public function summary(Request $request): JsonResponse
    {
        [$from, $to, $range] = $this->resolveRange($request);
        $filters = $request->validate([
            'user_id' => ['nullable', 'integer', 'min:1'],
            'node_id' => ['nullable', 'string', 'max:128'],
        ]);
        $query = $this->rangeQuery($from, $to)
            ->when(isset($filters['user_id']), fn (Builder $q) => $q->where('user_id', $filters['user_id']))
            ->when(isset($filters['node_id']), fn (Builder $q) => $q->where('node_id', $filters['node_id']));

        $totals = (clone $query)->selectRaw(
            'COALESCE(SUM(upload_bytes), 0) AS upload_bytes, ' .
            'COALESCE(SUM(download_bytes), 0) AS download_bytes, ' .
            'COUNT(DISTINCT COALESCE(connection_id, event_id)) AS connections'
        )->first();

        $groupBy = $request->query('group_by', 'service');
        $rows = (clone $query)->get([
            'event_id', 'connection_id', 'destination', 'upload_bytes', 'download_bytes', 'action', 'protocol', 'network',
        ]);
        $groups = [];
        foreach ($rows as $row) {
            $label = $groupBy === 'destination'
                ? $this->destinationHost($row->destination)
                : $this->serviceName($row->destination);
            $label = $label !== '' ? $label : 'Direct IP / Other';
            $groups[$label] ??= [
                'label' => $label,
                'upload_bytes' => 0,
                'download_bytes' => 0,
                'connections' => 0,
                '_connection_ids' => [],
                '_destinations' => [],
            ];
            $groups[$label]['upload_bytes'] += (int) $row->upload_bytes;
            $groups[$label]['download_bytes'] += (int) $row->download_bytes;
            $groups[$label]['_connection_ids'][(string) ($row->connection_id ?: $row->event_id)] = true;

            $host = $this->destinationHost($row->destination) ?: 'Direct IP';
            $groups[$label]['_destinations'][$host] ??= [
                'label' => $host,
                'upload_bytes' => 0,
                'download_bytes' => 0,
                '_connection_ids' => [],
            ];
            $groups[$label]['_destinations'][$host]['upload_bytes'] += (int) $row->upload_bytes;
            $groups[$label]['_destinations'][$host]['download_bytes'] += (int) $row->download_bytes;
            $groups[$label]['_destinations'][$host]['_connection_ids'][(string) ($row->connection_id ?: $row->event_id)] = true;
        }

        $distribution = array_values($groups);
        foreach ($distribution as &$item) {
            $item['connections'] = count($item['_connection_ids']);
            unset($item['_connection_ids']);
            $destinations = array_values($item['_destinations']);
            foreach ($destinations as &$destination) {
                $destination['connections'] = count($destination['_connection_ids']);
                unset($destination['_connection_ids']);
                $destination['traffic_bytes'] = $destination['upload_bytes'] + $destination['download_bytes'];
            }
            unset($destination);
            usort($destinations, fn (array $a, array $b) => $b['traffic_bytes'] <=> $a['traffic_bytes']);
            $item['destinations'] = array_slice($destinations, 0, 20);
            unset($item['_destinations']);
            $item['traffic_bytes'] = $item['upload_bytes'] + $item['download_bytes'];
        }
        unset($item);

        $metric = $request->query('metric', 'traffic');
        usort($distribution, fn (array $a, array $b) => $b[$metric === 'connections' ? 'connections' : 'traffic_bytes'] <=> $a[$metric === 'connections' ? 'connections' : 'traffic_bytes']);

        return response()->json([
            'data' => [
                'range' => $range,
                'user_id' => $filters['user_id'] ?? null,
                'from' => $from->toIso8601String(),
                'to' => $to->toIso8601String(),
                'cards' => [
                    'upload_bytes' => (int) $totals->upload_bytes,
                    'download_bytes' => (int) $totals->download_bytes,
                    'total_bytes' => (int) $totals->upload_bytes + (int) $totals->download_bytes,
                    'connections' => (int) $totals->connections,
                ],
                'distribution' => array_slice($distribution, 0, 50),
            ],
        ]);
    }

    public function events(Request $request): JsonResponse
    {
        [$from, $to] = $this->resolveRange($request);
        $filters = $request->validate([
            'user_id' => ['nullable', 'integer', 'min:1'],
            'node_id' => ['nullable', 'string', 'max:128'],
            'action' => ['nullable', 'string', 'max:32'],
            'rule_tag' => ['nullable', 'string', 'max:128'],
            'destination' => ['nullable', 'string', 'max:255'],
            'per_page' => ['nullable', 'integer', 'min:1', 'max:100'],
        ]);

        $query = $this->rangeQuery($from, $to)
            ->when(isset($filters['user_id']), fn (Builder $q) => $q->where('user_id', $filters['user_id']))
            ->when(isset($filters['node_id']), fn (Builder $q) => $q->where('node_id', $filters['node_id']))
            ->when(isset($filters['action']), fn (Builder $q) => $q->where('action', $filters['action']))
            ->when(isset($filters['rule_tag']), fn (Builder $q) => $q->where('rule_tag', $filters['rule_tag']))
            ->when(isset($filters['destination']), fn (Builder $q) => $q->where('destination', 'like', '%' . addcslashes($filters['destination'], '%_\\') . '%'))
            ->orderByDesc('occurred_at')
            ->orderByDesc('id');

        return response()->json($query->paginate($filters['per_page'] ?? 50));
    }

    /** @return array{CarbonImmutable, CarbonImmutable, string} */
    private function resolveRange(Request $request): array
    {
        $range = (string) $request->query('range', '24h');
        $to = now()->utc()->toImmutable();
        $from = match ($range) {
            '1h' => $to->subHour(),
            '6h' => $to->subHours(6),
            '7d' => $to->subDays(7),
            '14d' => $to->subDays(14),
            default => $to->subDay(),
        };

        return [$from, $to, in_array($range, ['1h', '6h', '24h', '7d', '14d'], true) ? $range : '24h'];
    }

    private function rangeQuery(CarbonImmutable $from, CarbonImmutable $to): Builder
    {
        return DB::table('v2_risk_audit_events')->whereBetween('occurred_at', [$from, $to]);
    }

    private function isAuditAdmin(Request $request): bool
    {
        $payload = $this->auditCookiePayload($request);

        return $payload !== null && User::query()
            ->whereKey($payload['user_id'])
            ->where('is_admin', true)
            ->exists();
    }

    private function auditURL(string $suffix = ''): string
    {
        $securePath = admin_setting('secure_path', admin_setting('frontend_admin_path', hash('crc32b', config('app.key'))));

        return url('/' . trim($securePath, '/') . '/risk-audit' . $suffix);
    }

    /** @return array{user_id: int, expires_at: int}|null */
    private function auditCookiePayload(Request $request): ?array
    {
        try {
            $value = (string) $request->cookie('risk_audit_admin', '');
            $payload = json_decode(Crypt::decryptString($value), true, 512, JSON_THROW_ON_ERROR);
            if (!is_array($payload) || !is_numeric($payload['user_id'] ?? null) || !is_numeric($payload['expires_at'] ?? null)) {
                return null;
            }
            if ((int) $payload['expires_at'] < time()) {
                return null;
            }

            return ['user_id' => (int) $payload['user_id'], 'expires_at' => (int) $payload['expires_at']];
        } catch (\Throwable) {
            return null;
        }
    }

    private function destinationHost(?string $destination): string
    {
        if (!$destination) {
            return '';
        }
        $value = preg_replace('/^[a-z]+:/i', '', $destination);
        $value = trim((string) $value, '[]');
        if (preg_match('/^\[([^]]+)](?::\d+)?$/', $value, $matches)) {
            return $matches[1];
        }

        return preg_replace('/:\d+$/', '', $value) ?: '';
    }

    private function serviceName(?string $destination): string
    {
        $host = strtolower($this->destinationHost($destination));
        foreach ([
            'Connectivity Check' => ['gstatic.com', 'msftconnecttest.com', 'msftncsi.com', 'captive.apple.com'],
            'Google' => ['google.com', 'googleapis.com', 'googleusercontent.com'],
            'Javdb' => ['javdb.com', 'jdbstatic.com'],
            'Yandex' => ['yandex.com', 'yandex.ru', 'yandex.net'],
            'Cloudflare' => ['cloudflare.com', 'cloudflare-dns.com'],
            'Microsoft' => ['microsoft.com', 'windows.com', 'live.com', 'office.com'],
            'Apple' => ['apple.com', 'icloud.com'],
            'Tencent' => ['qq.com', 'qcloud.com', 'weixin.com'],
            'Alibaba' => ['aliyun.com', 'taobao.com', 'youku.com'],
            'ByteDance' => ['bytedance.com', 'douyin.com', 'tiktok.com'],
        ] as $service => $domains) {
            foreach ($domains as $domain) {
                if ($host === $domain || str_ends_with($host, '.' . $domain)) {
                    return $service;
                }
            }
        }

        return filter_var($host, FILTER_VALIDATE_IP) ? 'Direct IP' : ($host ?: 'Other');
    }
}
