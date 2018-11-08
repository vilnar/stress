<?php
class plugin {

        private $sphinxql = false;

        public function init() {
                $this->sphinxql = new mysqli('manticore', '', '', '', 9306);
                $this->sphinxql->query("alter table testrt add column str string");
                $this->sphinxql->query("alter rtindex testrt reconfigure");
                $this->sphinxql->query("set global query_log_format=sphinxql");
                $this->sphinxql->query("truncate rtindex testrt");
        }

        public function query($queries) {
                $out = array();
                foreach ($queries as $id=>$query) {
                        $t = microtime(true);
                        $query = "insert into testrt values($id, '$query', '$query', ".rand(1,100).", '$query')";
                        $res = $this->sphinxql->query($query);
                        $out[$id] = array('latency' => microtime(true) - $t);
                }
                return $out;
        }

        public static function report($queriesInfo) {
                return array('Count' => count($queriesInfo));
        }
}
