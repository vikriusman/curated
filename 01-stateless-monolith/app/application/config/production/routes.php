<?php
defined('BASEPATH') OR exit('No direct script access allowed');

$route['default_controller']   = 'gallery';
$route['404_override']         = '';
$route['translate_uri_dashes'] = FALSE;

$route['login']   = 'auth/login';
$route['logout']  = 'auth/logout';
$route['upload']  = 'gallery/upload';
$route['whoami']  = 'gallery/whoami';
$route['healthz'] = 'health/index';
