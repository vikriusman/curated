<?php defined('BASEPATH') OR exit('No direct script access allowed'); ?>
<?= form_open('login', array('class' => 'login card')) ?>
  <h1>Stateless Gallery</h1>
  <p class="muted">Served by pod <code><?= html_escape(gethostname()) ?></code></p>
  <?php if ($error): ?><div class="flash error"><?= html_escape($error) ?></div><?php endif; ?>
  <input name="username" placeholder="username" autocomplete="username" required>
  <input name="password" type="password" placeholder="password" autocomplete="current-password" required>
  <button type="submit">Sign in</button>
<?= form_close() ?>
