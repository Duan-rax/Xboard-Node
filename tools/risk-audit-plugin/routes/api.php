<?php

use Illuminate\Support\Facades\Route;
use Plugin\RiskAudit\Controllers\AuditEventController;

Route::prefix('api/v1/risk-audit')->group(function (): void {
    Route::post('/events', [AuditEventController::class, 'store']);
});
