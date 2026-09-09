<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>流量审计 · 管理员登录</title>
  <style>
    *{box-sizing:border-box}body{margin:0;min-height:100vh;display:grid;place-items:center;background:#f2f5f9;color:#10213e;font-family:Inter,"Microsoft YaHei",system-ui,sans-serif}.card{width:min(420px,calc(100vw - 32px));padding:34px;background:#fff;border:1px solid #e3e9f0;border-radius:16px;box-shadow:0 10px 30px #19376212}.mark{display:grid;place-items:center;width:40px;height:40px;border-radius:12px;background:linear-gradient(135deg,#1677ff,#58a5ff);color:#fff;font-size:22px;font-weight:800;margin-bottom:19px}h1{font-size:24px;margin:0 0 8px;letter-spacing:-.04em}.sub{color:#7788a0;line-height:1.6;margin:0 0 24px;font-size:14px}label{display:block;font-size:13px;font-weight:700;margin:15px 0 7px;color:#41526a}input{width:100%;height:42px;border:1px solid #dbe3ec;border-radius:8px;padding:0 11px;font:14px inherit}input:focus{outline:2px solid #b9d7ff;border-color:#1677ff}.error{margin:0 0 14px;padding:10px 12px;border-radius:8px;background:#fff2f0;color:#d94735;font-size:13px}.submit{margin-top:22px;width:100%;height:43px;border:0;border-radius:8px;background:#1677ff;color:#fff;font:700 14px inherit;cursor:pointer}.hint{margin-top:16px;color:#8795a9;font-size:12px;line-height:1.6}
  </style>
</head>
<body>
  <main class="card">
    <div class="mark">A</div>
    <h1>流量审计</h1>
    <p class="sub">使用现有 Xboard 管理员账号验证身份。不会要求或保存 API Token。</p>
    @if ($error === 'invalid')
      <div class="error">管理员账号或密码无效。</div>
    @endif
    <form method="post" action="{{ $loginAction }}">
      <label for="email">管理员邮箱</label>
      <input id="email" name="email" type="email" autocomplete="username" value="{{ $email }}" required autofocus>
      <label for="password">密码</label>
      <input id="password" name="password" type="password" autocomplete="current-password" required>
      <button class="submit" type="submit">进入流量审计</button>
    </form>
    <p class="hint">此会话仅用于 RiskAudit 管理页；关闭浏览器或清除该站点 Cookie 后需重新验证。</p>
  </main>
</body>
</html>
