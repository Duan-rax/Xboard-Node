<?php

namespace Plugin\RiskAudit;

use App\Services\Plugin\AbstractPlugin;
use Illuminate\Console\Scheduling\Schedule;
use Illuminate\Support\Facades\DB;

class Plugin extends AbstractPlugin
{
    public function schedule(Schedule $schedule): void
    {
        $schedule->call(function (): void {
            $days = max(1, (int) $this->getConfig('retention_days', 30));

            DB::table('v2_risk_audit_events')
                ->where('occurred_at', '<', now()->subDays($days))
                ->delete();
        })->dailyAt('03:17')->withoutOverlapping();
    }
}
