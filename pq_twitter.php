<?php
class plugin {

	private $sphinxql = false;

	public function init() {
		$this->sphinxql = new mysqli('127.0.0.1', '', '', '', 9314);
	}

	public function query($docs) {
		foreach ($docs as $k=>$doc) $docs[$k] = "'".$this->sphinxql->escape_string(json_decode($doc))."'";
		if (count($docs) > 1) $query = "(".implode(",", $docs).")";
		else $query = $docs[array_keys($docs)[0]];
		$t = microtime(true);
		$res = $this->sphinxql->query("call pq('pq', $query, 0 as docs_json, 1 as docs)");
		$t = microtime(true) - $t;
		$counts = array();
		while ($row = $res->fetch_assoc()) {
			$matchedDocs = explode(',', $row['Documents']);
			foreach ($matchedDocs as $v) @$counts[array_keys($docs)[$v - 1]]++;
		}
		$out = array();
		foreach ($docs as $id => $v) $out[$id] = array('latency' => $t, 'num_rows' => @$counts[$id]);
		return $out;
	}

	public static function report($queriesInfo) {
		$totalMatches = 0;
		foreach($queriesInfo as $id => $info) {
			$totalMatches += $info['num_rows'];
		}
		return array(
		'Total matches' => $totalMatches,
		'Count' => count($queriesInfo));
	}
}
