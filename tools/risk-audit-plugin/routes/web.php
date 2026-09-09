<?php

use Illuminate\Support\Facades\Route;
use Plugin\RiskAudit\Controllers\AdminTrafficController;
use Plugin\RiskAudit\Middleware\RequireAuditAdmin;

$securePath = admin_setting('secure_path', admin_setting('frontend_admin_path', hash('crc32b', config('app.key'))));

Route::get('/' . $securePath . '/risk-audit/login', [AdminTrafficController::class, 'loginPage'])
    ->name('risk-audit.login');
Route::post('/' . $securePath . '/risk-audit/login', [AdminTrafficController::class, 'login'])
    ->middleware('throttle:5,1')
    ->withoutMiddleware(\Illuminate\Foundation\Http\Middleware\VerifyCsrfToken::class)
    ->name('risk-audit.login.submit');

Route::get('/' . $securePath . '/risk-audit', [AdminTrafficController::class, 'page'])
    ->middleware(RequireAuditAdmin::class)
    ->name('risk-audit.admin-page');
Route::post('/' . $securePath . '/risk-audit/logout', [AdminTrafficController::class, 'logout'])
    ->middleware(RequireAuditAdmin::class)
    ->name('risk-audit.logout');

Route::prefix('api/v2/' . $securePath . '/risk-audit')
    ->middleware(RequireAuditAdmin::class)
    ->group(function (): void {
        Route::get('/summary', [AdminTrafficController::class, 'summary']);
        Route::get('/events', [AdminTrafficController::class, 'events']);
    });
