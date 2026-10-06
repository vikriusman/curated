<?php
defined('BASEPATH') OR exit('No direct script access allowed');

/*
 * Overrides for ENVIRONMENT=production.
 * CI3 loads config/config.php first, then this file, so only keys that
 * differ from the defaults live here.
 *
 * Everything that differs between environments comes from env vars (12-factor),
 * so the same image runs locally and on AWS.
 */

// Must be explicit: otherwise CI3 guesses from SERVER_ADDR (the pod IP),
// so redirects behind the ingress point at an internal address.
$config['base_url']   = getenv('BASE_URL') ?: '';
$config['index_page'] = '';

// Logs go to pod-local disk (never the S3 mount). Level 1 = errors only.
$config['log_threshold'] = 1;

$config['encryption_key'] = getenv('APP_KEY') ?: '';

/*
 * Core of the stateless pattern: sessions are NOT stored on pod disk.
 * CI3's built-in redis driver uses the phpredis extension.
 * save_path format: tcp://host:port?auth=...&database=...&prefix=...
 */
$config['sess_driver']             = 'redis';
$config['sess_save_path']          = getenv('SESSION_REDIS_URL') ?: 'tcp://redis:6379?prefix=gallery:sess:';
$config['sess_cookie_name']        = 'gallery_session';
$config['sess_expiration']         = 7200;
$config['sess_match_ip']           = FALSE; // client IP can change behind a load balancer
$config['sess_time_to_update']     = 300;
$config['sess_regenerate_destroy'] = TRUE;

$config['cookie_httponly'] = TRUE;
$config['cookie_samesite'] = 'Lax';

// CSRF tokens live in a cookie, so they are stateless too.
$config['csrf_protection']  = TRUE;
$config['csrf_token_name']  = 'csrf_token';
$config['csrf_cookie_name'] = 'csrf_cookie';
$config['csrf_regenerate']  = FALSE; // multiple gallery tabs do not invalidate each other's token

// Trust X-Forwarded-For from the in-cluster ingress.
$config['proxy_ips'] = getenv('TRUSTED_PROXIES') ?: '10.0.0.0/8';

/*
 * Gallery app settings.
 */
// Upload directory: FUSE mount (Mountpoint for Amazon S3) shared by every pod.
$config['gallery_upload_path'] = getenv('UPLOAD_PATH') ?: '/data/uploads';
$config['gallery_max_kb']      = 4096;

// Dummy user. The password comes from env, not from code.
$config['gallery_users'] = array(
	'demo' => getenv('DEMO_PASSWORD') ?: 'demo',
);
