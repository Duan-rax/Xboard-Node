<?php

namespace Plugin\RiskAudit\Controllers;

use App\Http\Controllers\PluginController;
use App\Models\User;
use Carbon\CarbonImmutable;
use Illuminate\Database\Query\Builder;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Cache;

class AuditEventController extends PluginController
{
    public function store(Request $request): JsonResponse
    {
        if ($error = $this->beforePluginAction()) {
            return response()->json(['message' => $error[1]], $error[0]);
        }

        $secret = (string) $this->getConfig('webhook_secret', '');
        if ($secret === '') {
            return response()->json(['message' => 'Webhook secret is not configured'], 503);
        }

        $timestamp = $request->header('X-Audit-Timestamp');
        $signature = $request->header('X-Audit-Signature');
        if (is_string($timestamp) || is_string($signature)) {
            $skew = max(30, (int) $this->getConfig('max_clock_skew_seconds', 300));
            if (!is_string($timestamp) || !ctype_digit($timestamp) || abs(time() - (int) $timestamp) > $skew) {
                return response()->json(['message' => 'Invalid or expired audit timestamp'], 401);
            }
            if (!is_string($signature) || !preg_match('/^[a-f0-9]{64}$/i', $signature)) {
                return response()->json(['message' => 'Invalid audit signature'], 401);
            }
            $expected = hash_hmac('sha256', $timestamp . '.' . $request->getContent(), $secret);
            if (!hash_equals($expected, $signature)) {
                return response()->json(['message' => 'Invalid audit signature'], 401);
            }
        } elseif (!hash_equals($secret, (string) $request->header('X-Audit-Token', ''))) {
            // Xray route webhooks can set fixed headers but cannot calculate an HMAC.
            return response()->json(['message' => 'Invalid audit token'], 401);
        }

        $this->normaliseXrayWebhook($request);

        $data = $request->validate([
            'event_id' => ['required', 'string', 'max:64', 'regex:/^[A-Za-z0-9_-]+$/'],
            'connection_id' => ['nullable', 'string', 'max:128'],
            'event_type' => ['nullable', 'in:start,delta,close,rule'],
            'user_id' => ['nullable', 'integer', 'min:1'],
            'node_id' => ['nullable', 'string', 'max:128'],
            'client_source' => ['nullable', 'string', 'max:255'],
            'occurred_at' => ['nullable', 'date'],
            'action' => ['required', 'string', 'max:32'],
            'rule_tag' => ['nullable', 'string', 'max:128'],
            'destination' => ['nullable', 'string', 'max:255'],
            'destination_ip' => ['nullable', 'ip'],
            'network' => ['nullable', 'string', 'max:16'],
            'upload_bytes' => ['nullable', 'integer', 'min:0'],
            'download_bytes' => ['nullable', 'integer', 'min:0'],
            'protocol' => ['nullable', 'string', 'max:32'],
            'metadata' => ['nullable', 'array'],
        ]);

        $occurredAt = isset($data['occurred_at'])
            ? CarbonImmutable::parse($data['occurred_at'])->utc()
            : now()->utc();

        $userId = isset($data['user_id']) ? (int) $data['user_id'] : null;
        if ($userId === null && !empty($data['metadata']['user']) && is_string($data['metadata']['user'])) {
            $userReference = trim($data['metadata']['user']);
            if (preg_match('/(?:^|@)([1-9][0-9]*)$/', $userReference, $matches)) {
                $userId = (int) $matches[1];
            } else {
                $cacheKey = 'risk_audit_uuid_' . hash('sha256', $userReference);
                $resolved = Cache::remember($cacheKey, 600, fn () => User::query()->where('uuid', $userReference)->value('id'));
                $userId = $resolved ? (int) $resolved : null;
            }
        }

        $event = [
            'event_id' => $data['event_id'],
            'connection_id' => $data['connection_id'] ?? $data['event_id'],
            'event_type' => $data['event_type'] ?? 'rule',
            'user_id' => $userId,
            'node_id' => $data['node_id'] ?? null,
            'client_source' => $data['client_source'] ?? null,
            'occurred_at' => $occurredAt,
            'action' => $data['action'],
            'rule_tag' => $data['rule_tag'] ?? null,
            'destination' => $data['destination'] ?? null,
            'destination_ip' => $data['destination_ip'] ?? null,
            'network' => $data['network'] ?? null,
            'upload_bytes' => $data['upload_bytes'] ?? 0,
            'download_bytes' => $data['download_bytes'] ?? 0,
            'protocol' => $data['protocol'] ?? null,
            'source_ip' => $request->ip(),
            'metadata' => isset($data['metadata']) ? json_encode($data['metadata'], JSON_THROW_ON_ERROR) : null,
            'created_at' => now(),
            'updated_at' => now(),
        ];

        $inserted = DB::table('v2_risk_audit_events')->insertOrIgnore($event) === 1;

        return response()->json([
            'data' => [
                'event_id' => $data['event_id'],
                'accepted' => true,
                'duplicate' => !$inserted,
            ],
        ], $inserted ? 201 : 200);
    }

    public function index(Request $request): JsonResponse
    {
        if ($response = $this->authorizeRead($request)) {
            return $response;
        }

        $filters = $request->validate([
            'user_id' => ['nullable', 'integer', 'min:1'],
            'node_id' => ['nullable', 'string', 'max:128'],
            'action' => ['nullable', 'string', 'max:32'],
            'rule_tag' => ['nullable', 'string', 'max:128'],
            'destination' => ['nullable', 'string', 'max:255'],
            'from' => ['nullable', 'date'],
            'to' => ['nullable', 'date'],
            'per_page' => ['nullable', 'integer', 'min:1', 'max:100'],
        ]);

        $query = $this->filteredEvents($filters)->orderByDesc('occurred_at')->orderByDesc('id');

        return response()->json($query->paginate($filters['per_page'] ?? 50));
    }

    public function summary(Request $request): JsonResponse
    {
        if ($response = $this->authorizeRead($request)) {
            return $response;
        }

        $filters = $request->validate([
            'from' => ['nullable', 'date'],
            'to' => ['nullable', 'date'],
        ]);
        $query = $this->filteredEvents($filters);

        return response()->json([
            'data' => [
                'total' => (clone $query)->count(),
                'by_rule' => (clone $query)
                    ->select('rule_tag', DB::raw('COUNT(*) as total'))
                    ->groupBy('rule_tag')
                    ->orderByDesc('total')
                    ->limit(20)
                    ->get(),
                'top_destinations' => (clone $query)
                    ->whereNotNull('destination')
                    ->select('destination', DB::raw('COUNT(*) as total'))
                    ->groupBy('destination')
                    ->orderByDesc('total')
                    ->limit(20)
                    ->get(),
            ],
        ]);
    }

    private function authorizeRead(Request $request): ?JsonResponse
    {
        if ($error = $this->beforePluginAction()) {
            return response()->json(['message' => $error[1]], $error[0]);
        }

        $token = (string) $this->getConfig('read_token', '');
        $provided = (string) $request->header('X-Audit-Read-Token', '');

        if ($token === '' || $provided === '' || !hash_equals($token, $provided)) {
            return response()->json(['message' => 'Unauthorized'], 401);
        }

        return null;
    }

    private function normaliseXrayWebhook(Request $request): void
    {
        $payload = $request->all();
        if (isset($payload['event_id']) || !isset($payload['destination'])) {
            return;
        }

        $userId = null;
        $email = (string) ($payload['email'] ?? '');
        if (preg_match('/(?:^|@)([1-9][0-9]*)$/', $email, $matches)) {
            $userId = (int) $matches[1];
        }

        $timestamp = isset($payload['ts']) && is_numeric($payload['ts'])
            ? CarbonImmutable::createFromTimestampUTC((int) $payload['ts'])->toIso8601String()
            : now()->utc()->toIso8601String();

        $request->replace([
            'event_id' => 'xray-' . hash('sha256', $request->getContent()),
            'connection_id' => 'xray-' . hash('sha256', $request->getContent()),
            'event_type' => 'rule',
            'user_id' => $userId,
            'node_id' => $request->header('X-Audit-Node'),
            'occurred_at' => $timestamp,
            'action' => (string) ($payload['outboundTag'] ?? 'observe'),
            'rule_tag' => $request->header('X-Audit-Rule'),
            'destination' => (string) $payload['destination'],
            'client_source' => $payload['source'] ?? null,
            'network' => $payload['network'] ?? null,
            'protocol' => $payload['protocol'] ?? null,
            'metadata' => [
                'kernel' => 'xray',
                'inbound_tag' => $payload['inboundTag'] ?? null,
                'inbound_name' => $payload['inboundName'] ?? null,
                'original_target' => $payload['originalTarget'] ?? null,
            ],
        ]);
    }

    /** @param array<string, mixed> $filters */
    private function filteredEvents(array $filters): Builder
    {
        return DB::table('v2_risk_audit_events')
            ->when(isset($filters['user_id']), fn (Builder $q) => $q->where('user_id', $filters['user_id']))
            ->when(isset($filters['node_id']), fn (Builder $q) => $q->where('node_id', $filters['node_id']))
            ->when(isset($filters['action']), fn (Builder $q) => $q->where('action', $filters['action']))
            ->when(isset($filters['rule_tag']), fn (Builder $q) => $q->where('rule_tag', $filters['rule_tag']))
            ->when(isset($filters['destination']), fn (Builder $q) => $q->where('destination', 'like', '%' . addcslashes($filters['destination'], '%_\\') . '%'))
            ->when(isset($filters['from']), fn (Builder $q) => $q->where('occurred_at', '>=', CarbonImmutable::parse($filters['from'])->utc()))
            ->when(isset($filters['to']), fn (Builder $q) => $q->where('occurred_at', '<=', CarbonImmutable::parse($filters['to'])->utc()));
    }
}
