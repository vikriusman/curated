<?php defined('BASEPATH') OR exit('No direct script access allowed'); ?>
<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title><?= html_escape($title) ?> · Stateless Gallery</title>
<style>
  :root { --bg:#f6f5f2; --fg:#1d1d1b; --muted:#6b6b66; --card:#fff; --line:#e3e1db; --accent:#b45309; --ok:#166534; --err:#b91c1c; }
  @media (prefers-color-scheme: dark) { :root { --bg:#151514; --fg:#ecebe6; --muted:#9b9a94; --card:#1f1f1d; --line:#33322f; --accent:#f59e0b; --ok:#4ade80; --err:#f87171; } }
  * { box-sizing:border-box; }
  body { margin:0; font:15px/1.5 system-ui, sans-serif; background:var(--bg); color:var(--fg); }
  main { max-width:960px; margin:0 auto; padding:24px 16px; }
  header { display:flex; justify-content:space-between; align-items:center; gap:12px; flex-wrap:wrap; }
  h1 { font-size:20px; margin:0; }
  a { color:var(--accent); }
  .pod { font:12px ui-monospace, monospace; background:var(--card); border:1px solid var(--line); border-radius:6px; padding:8px 12px; margin:16px 0; }
  .pod b { color:var(--accent); }
  .card { background:var(--card); border:1px solid var(--line); border-radius:8px; padding:16px; }
  .grid { display:grid; grid-template-columns:repeat(auto-fill, minmax(180px, 1fr)); gap:12px; margin-top:16px; }
  .grid figure { margin:0; background:var(--card); border:1px solid var(--line); border-radius:8px; overflow:hidden; }
  .grid img { width:100%; aspect-ratio:1; object-fit:cover; display:block; }
  .grid figcaption { font:11px ui-monospace, monospace; color:var(--muted); padding:6px 8px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
  .flash { padding:8px 12px; border-radius:6px; margin:12px 0; border:1px solid currentColor; }
  .flash.ok { color:var(--ok); } .flash.error { color:var(--err); }
  input, button { font:inherit; padding:8px 10px; border-radius:6px; border:1px solid var(--line); background:var(--bg); color:var(--fg); }
  button { background:var(--accent); color:#fff; border:0; cursor:pointer; }
  form.login { max-width:320px; margin:10vh auto; display:grid; gap:10px; }
  .muted { color:var(--muted); }
</style>
</head>
<body>
<main>
