<?php

namespace Plugin\RiskAudit\Middleware;

use App\Models\User;
use Closure;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Crypt;

class RequireAuditAdmin
{
    public function handle(Request $request, Closure $next)
    {
        $userId = null;
        try {
            $payload = json_decode(Crypt::decryptString((string) $request->cookie('risk_audit_admin', '')), true, 512, JSON_THROW_ON_ERROR);
            if (is_array($payload) && is_numeric($payload['user_id'] ?? null) && is_numeric($payload['expires_at'] ?? null) && (int) $payload['expires_at'] >= time()) {
                $userId = (int) $payload['user_id'];
            }
        } catch (\Throwable) {
            $userId = null;
        }
        $allowed = $userId !== null && User::query()
            ->whereKey($userId)
            ->where('is_admin', true)
            ->exists();

        if ($allowed) {
            return $next($request);
        }

        if ($request->expectsJson() || str_starts_with($request->path(), 'api/')) {
            return response()->json(['message' => 'Administrator session required'], 401);
        }

        $securePath = admin_setting('secure_path', admin_setting('frontend_admin_path', hash('crc32b', config('app.key'))));

        return redirect(url('/' . trim($securePath, '/') . '/risk-audit/login'));
    }
}
