<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('v2_risk_audit_events', function (Blueprint $table): void {
            $table->bigIncrements('id');
            $table->string('event_id', 64)->unique();
            $table->unsignedBigInteger('user_id')->nullable();
            $table->string('node_id', 128)->nullable();
            $table->timestamp('occurred_at');
            $table->string('action', 32);
            $table->string('rule_tag', 128)->nullable();
            $table->string('destination', 255)->nullable();
            $table->string('destination_ip', 45)->nullable();
            $table->string('network', 16)->nullable();
            $table->string('protocol', 32)->nullable();
            $table->string('source_ip', 45)->nullable();
            $table->json('metadata')->nullable();
            $table->timestamps();

            $table->index(['user_id', 'occurred_at']);
            $table->index(['rule_tag', 'occurred_at']);
            $table->index(['node_id', 'occurred_at']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('v2_risk_audit_events');
    }
};
