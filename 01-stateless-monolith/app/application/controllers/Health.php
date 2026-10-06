<?php
defined('BASEPATH') OR exit('No direct script access allowed');

/**
 * Kubernetes probes. Does not load the session library.
 *
 *   /healthz          liveness: the PHP process is alive
 *   /healthz?ready=1  readiness: the upload mount is readable
 *
 * Redis is deliberately not checked in readiness: if Redis goes down, every pod
 * would turn unready at once and take the whole app down, not just login.
 */
class Health extends CI_Controller {

	public function index()
	{
		$status = 200;
		$body   = array('status' => 'ok', 'pod' => gethostname());

		if ($this->input->get('ready'))
		{
			$path = $this->config->item('gallery_upload_path');
			if ( ! is_dir($path) OR ! is_readable($path))
			{
				$status         = 503;
				$body['status'] = 'upload mount unavailable';
			}
		}

		$this->output
			->set_status_header($status)
			->set_content_type('application/json')
			->set_output(json_encode($body));
	}
}
