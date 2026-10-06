<?php
defined('BASEPATH') OR exit('No direct script access allowed');

class Auth extends CI_Controller {

	public function __construct()
	{
		parent::__construct();
		$this->load->library('session');
	}

	public function login()
	{
		if ($this->session->userdata('user'))
		{
			redirect('/');
		}

		$data = array('error' => NULL);

		if ($this->input->method() === 'post')
		{
			$username = (string) $this->input->post('username');
			$password = (string) $this->input->post('password');
			$users    = $this->config->item('gallery_users');

			if (isset($users[$username]) && hash_equals($users[$username], $password))
			{
				// Rotate the session ID after login (prevents session fixation).
				$this->session->sess_regenerate(TRUE);
				$this->session->set_userdata(array(
					'user'         => $username,
					'login_pod'    => gethostname(),
					'logged_in_at' => date(DATE_ATOM),
					'visits'       => 0,
				));
				redirect('/');
			}

			$data['error'] = 'Invalid username or password.';
		}

		$this->load->view('layout/header', array('title' => 'Login'));
		$this->load->view('login', $data);
		$this->load->view('layout/footer');
	}

	public function logout()
	{
		$this->session->sess_destroy();
		redirect('login');
	}
}
