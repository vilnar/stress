<?php
class plugin {

	private $sphinxql = false;

	public function init($options) {
        $port = isset($options['port']) ? $options['port'] : 9312;
        $this->idx = isset($options['index']) ? $options['index'] : 'idx';
        $this->maxmatches = '';
        if (isset($options['maxmatches']))
            $this->maxmatches = " limit ".$options['maxmatches']." option max_matches=".$options['maxmatches'];
        
		$this->sphinxql = new mysqli('127.0.0.1', '', '', '', $port);
	}

	public function query($queries) {
                $out = array();
		foreach ($queries as $id=>$query) {
			$t = microtime(true);
			$res = $this->sphinxql->query("select * from ".$this->idx." where match('".$this->sphinxql->escape_string($query)."')".$this->maxmatches);
			$out[$id] = array('latency' => microtime(true) - $t, 'num_rows' => $res->num_rows);
			/*$ids = array();
			while($row = $res->fetch_array()) $ids[] = $row['id'];
			sort($ids);
			if ($ids) file_put_contents('/tmp/compare/ms_'.$id, implode("\n", $ids));*/
		}
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
