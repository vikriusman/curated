<?php
defined('BASEPATH') OR exit('No direct script access allowed');

// The session library is deliberately NOT autoloaded: /healthz must not create
// a new session key in Redis every time a Kubernetes probe runs.
$autoload['packages']  = array();
$autoload['libraries'] = array();
$autoload['drivers']   = array();
$autoload['helper']    = array('url', 'form', 'html');
$autoload['config']    = array();
$autoload['language']  = array();
$autoload['model']     = array();
