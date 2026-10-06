<?php defined('BASEPATH') OR exit('No direct script access allowed'); ?>
<header>
  <h1>Stateless Gallery</h1>
  <span><?= html_escape($this->session->userdata('user')) ?> · <a href="<?= site_url('logout') ?>">sign out</a></span>
</header>

<div class="pod">
  this request was served by pod <b><?= html_escape($runtime['pod']) ?></b> on node <b><?= html_escape($runtime['node']) ?></b>
  · logged in on pod <?= html_escape($this->session->userdata('login_pod')) ?>
  · session visits: <?= (int) $this->session->userdata('visits') ?>
</div>

<?php if ($flash): ?>
  <div class="flash <?= $flash['type'] === 'ok' ? 'ok' : 'error' ?>"><?= html_escape($flash['text']) ?></div>
<?php endif; ?>

<div class="card">
  <?= form_open_multipart('upload') ?>
    <input type="file" name="photo" accept="image/*" required>
    <button type="submit">Upload</button>
  <?= form_close() ?>
</div>

<?php if (empty($photos)): ?>
  <p class="muted">No photos yet. Upload one, then refresh: the pod changes, the photo stays.</p>
<?php else: ?>
  <div class="grid">
    <?php foreach ($photos as $p): ?>
      <figure>
        <img src="<?= base_url('uploads/'.rawurlencode($p['name'])) ?>" alt="" loading="lazy">
        <figcaption><?= html_escape($p['name']) ?> · <?= round($p['size'] / 1024) ?> KB</figcaption>
      </figure>
    <?php endforeach; ?>
  </div>
<?php endif; ?>
