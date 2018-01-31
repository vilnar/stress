<?php
class plugin {

	private $ci = false;

	public function init() {
		$this->ci = curl_init();
		curl_setopt($this->ci, CURLOPT_CUSTOMREQUEST, 'GET');
		curl_setopt($this->ci, CURLOPT_HTTPHEADER, array('Content-Type: application/json'));
		curl_setopt($this->ci, CURLOPT_RETURNTRANSFER, 1);
		curl_setopt($this->ci, CURLOPT_HTTPHEADER, array('Content-Type: application/json'));
                curl_setopt($this->ci, CURLOPT_URL, 'http://localhost:9200/wikipedia/_search');
	}

	public function query($queries) {
                $out = array();
		foreach ($queries as $id=>$query) {
			$query = ltrim($query, '+-');
			$query = str_replace(array('+', '-'), array(' AND ', ' NOT '), $query);
                        $json = '{
  "size": 100000,
  "query": {
    "query_string": {
      "default_field": "body",
      "query": "'.$query.'"
    }
  }
}';

			$t = microtime(true);
			curl_setopt($this->ci, CURLOPT_POSTFIELDS, $json);
			$res = curl_exec($this->ci);
			$res = json_decode($res);
			$out[$id] = array('latency' => microtime(true) - $t, 'hits' => $res->hits->total);
		}
		return $out;
	}

	public static function report($queriesInfo) {
		$totalMatches = 0;
		foreach($queriesInfo as $id => $info) {
			$totalMatches += $info['hits'];
		}
		return array(
		'Total matches' => $totalMatches,
		'Count' => count($queriesInfo));
	}
}
