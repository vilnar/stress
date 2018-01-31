<?php
// This is a plugin example which points out the functions that have to be implemented and 
// demonstrated the expected behavior
class plugin {

	private $sphinxql = false;

	/* this function will be called when each parallel worker starts, 
	it's expected that you initialize a conneciton here etc.*/
	public function init() {
		// Let's make a SphinxQL connection which will be reused later
		$this->sphinxql = new mysqli('127.0.0.1', '', '', '', 9314);
	}

	/* this function will be called when a parallel worker is done with preparing a batch (of 1 or multiple documents)
	Accepts: array(document_id => document)
	Returns: array(
		document_id (as appeared in the input) => array('latency' => value_in_seconds[, optionally smth else about the query])
	)
	*/
	public function query($queries) {
                $out = array();
		foreach ($queries as $id=>$query) {
			$t = microtime(true);
			$res = $this->sphinxql->query("select * from idx where match('".$this->sphinxql->escape_string($query)."') limit 100000 option max_matches=100000");
			$out[$id] = array('latency' => microtime(true) - $t, 'num_rows' => $res->num_rows);
		}
		return $out;
	}

	/* this static function is called after the test.
	Here the plugin is expected to do some additional analysis based on the primary query results.
	The output will be merged into the output of the main script.
        Accepts: array(document_id => array returned by query())
	Returns: array('name' => 'value'[, 'name2' => 'value2', ...])
	*/
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
