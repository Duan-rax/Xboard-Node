<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('v2_risk_audit_events', function (Blueprint $table): void {
            if (!Schema::hasColumn('v2_risk_audit_events', 'client_source')) {
                $table->string('client_source', 255)->nullable()->after('node_id');
            }
            if (!Schema::hasColumn('v2_risk_audit_events', 'upload_bytes')) {
                $table->unsignedBigInteger('upload_bytes')->default(0)->after('network');
            }
            if (!Schema::hasColumn('v2_risk_audit_events', 'download_bytes')) {
                $table->unsignedBigInteger('download_bytes')->default(0)->after('upload_bytes');
            }
        });
    }

    public function down(): void
    {
        Schema::table('v2_risk_audit_events', function (Blueprint $table): void {
            $columns = array_filter(['client_source', 'upload_bytes', 'download_bytes'], fn (string $column) => Schema::hasColumn('v2_risk_audit_events', $column));
            if ($columns) {
                $table->dropColumn($columns);
            }
        });
    }
};
