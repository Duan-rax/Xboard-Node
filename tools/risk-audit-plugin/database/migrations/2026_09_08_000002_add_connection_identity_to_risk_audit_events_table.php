<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('v2_risk_audit_events', function (Blueprint $table): void {
            if (!Schema::hasColumn('v2_risk_audit_events', 'connection_id')) {
                $table->string('connection_id', 128)->nullable()->after('event_id');
                $table->index(['connection_id', 'occurred_at']);
            }
            if (!Schema::hasColumn('v2_risk_audit_events', 'event_type')) {
                $table->string('event_type', 16)->default('rule')->after('connection_id');
            }
        });
    }

    public function down(): void
    {
        Schema::table('v2_risk_audit_events', function (Blueprint $table): void {
            if (Schema::hasColumn('v2_risk_audit_events', 'connection_id')) {
                $table->dropIndex(['connection_id', 'occurred_at']);
                $table->dropColumn('connection_id');
            }
            if (Schema::hasColumn('v2_risk_audit_events', 'event_type')) {
                $table->dropColumn('event_type');
            }
        });
    }
};
