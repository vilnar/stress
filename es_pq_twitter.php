<?php

class plugin {

	private $sphinxql = false;

	public function init() {
		$this->ci = curl_init();
		curl_setopt($this->ci, CURLOPT_CUSTOMREQUEST, 'GET');
		curl_setopt($this->ci, CURLOPT_HTTPHEADER, array('Content-Type: application/json'));
		curl_setopt($this->ci, CURLOPT_RETURNTRANSFER, 1);
		curl_setopt($this->ci, CURLOPT_HTTPHEADER, array('Content-Type: application/json'));
                curl_setopt($this->ci, CURLOPT_URL, 'http://localhost:9200/pq/_search');
	}

	public function query($docs) {
                $out = array();
		foreach ($docs as $id=>$doc) $docs[$id] = '{"message" : "'.preg_replace('/\n/', '\n', htmlspecialchars(json_decode($doc))).'"}';
                $json = '{
    "size": 100000,
    "query" : {
        "percolate" : {
            "field" : "query",
            "documents" : ['.implode(',', $docs).']
        }
    }
}';

		$t = microtime(true);
		curl_setopt($this->ci, CURLOPT_POSTFIELDS, $json);
		$res = curl_exec($this->ci);
		$res = json_decode($res);

                $counts = array();
                if (isset($res->hits->hits)) foreach ($res->hits->hits as $hit) {
                        $matchedDocs = $hit->fields->_percolator_document_slot;
                        foreach ($matchedDocs as $v) @$counts[array_keys($docs)[$v]]++;
                }

		foreach ($docs as $id=>$doc) $out[$id] = array('latency' => microtime(true) - $t, 'hits' => @$counts[$id]);
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
