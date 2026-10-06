<?php
defined('BASEPATH') OR exit('No direct script access allowed');

class Gallery extends CI_Controller {

	/** @var string */
	private $upload_path;

	public function __construct()
	{
		parent::__construct();
		$this->load->library('session');
		$this->upload_path = rtrim($this->config->item('gallery_upload_path'), '/').'/';

		// Count visits in the session: proves session reads/writes across pods via Redis.
		if ($this->session->userdata('user'))
		{
			$this->session->set_userdata('visits', (int) $this->session->userdata('visits') + 1);
		}
	}

	public function index()
	{
		$this->require_login();

		$this->load->view('layout/header', array('title' => 'Galeri'));
		$this->load->view('gallery', array(
			'photos'  => $this->list_photos(),
			'flash'   => $this->session->flashdata('flash'),
			'runtime' => $this->runtime_info(),
		));
		$this->load->view('layout/footer');
	}

	public function upload()
	{
		$this->require_login();

		if ($this->input->method() !== 'post')
		{
			redirect('/');
		}

		$this->load->library('upload', array(
			'upload_path'   => $this->upload_path,
			'allowed_types' => 'jpg|jpeg|png|gif',
			'max_size'      => (int) $this->config->item('gallery_max_kb'),
			// Random names: Mountpoint for S3 does not support overwrite/rename,
			// so every upload must write a new object.
			'encrypt_name'  => TRUE,
			'overwrite'     => FALSE,
			'file_ext_tolower' => TRUE,
		));

		if ( ! $this->upload->do_upload('photo'))
		{
			$this->session->set_flashdata('flash', array(
				'type' => 'error',
				'text' => strip_tags($this->upload->display_errors('', ' ')),
			));
		}
		else
		{
			$file = $this->upload->data();
			$this->session->set_flashdata('flash', array(
				'type' => 'ok',
				'text' => sprintf('Saved as %s by pod %s.', $file['file_name'], gethostname()),
			));
		}

		redirect('/');
	}

	/**
	 * JSON for the demo script: which pod served the request, and whether the session held.
	 */
	public function whoami()
	{
		$this->output
			->set_content_type('application/json')
			->set_header('Cache-Control: no-store')
			->set_output(json_encode(array_merge($this->runtime_info(), array(
				'user'         => $this->session->userdata('user'),
				'login_pod'    => $this->session->userdata('login_pod'),
				'visits'       => (int) $this->session->userdata('visits'),
				'photo_count'  => count($this->list_photos()),
			)), JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES));
	}

	private function require_login()
	{
		if ( ! $this->session->userdata('user'))
		{
			redirect('login');
		}
	}

	private function runtime_info()
	{
		return array(
			'pod'  => gethostname(),
			'node' => getenv('NODE_NAME') ?: 'unknown',
		);
	}

	/**
	 * @return array<int, array{name: string, size: int, mtime: int}>
	 */
	private function list_photos()
	{
		$photos = array();

		// scandir, not glob(GLOB_BRACE): musl (Alpine) does not support GLOB_BRACE.
		foreach (@scandir($this->upload_path) ?: array() as $name)
		{
			if ( ! preg_match('/\.(jpe?g|png|gif)$/', $name))
			{
				continue;
			}

			$path     = $this->upload_path.$name;
			$photos[] = array(
				'name'  => $name,
				'size'  => (int) filesize($path),
				'mtime' => (int) filemtime($path),
			);
		}

		usort($photos, function ($a, $b) {
			return $b['mtime'] - $a['mtime'];
		});

		return $photos;
	}
}
