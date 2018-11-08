<?php
class plugin {

        private $sphinxql = false;

        public static function init_global() {
                $sphinxql = new mysqli('manticore', '', '', '', 9306);
                $sphinxql->query("alter table testrt add column str string");
                $sphinxql->query("alter rtindex testrt reconfigure");
                $sphinxql->query("set global query_log_format=sphinxql");
                $sphinxql->query("truncate rtindex testrt");
                $sphinxql->close();
        }

        public function init() {
                $this->sphinxql = new mysqli('manticore', '', '', '', 9306);
        }

        public function query($queries) {
                $out = array();
                foreach ($queries as $id=>$query) {
                        $t = microtime(true);
                        $query = "insert into testrt values(".($id+1).", '$query', '$query', ".rand(1,100).", '$query')";
                        $res = $this->sphinxql->query($query);
                        if ($this->sphinxql->error) echo "ERROR: {$this->sphinxql->error}\n";
                        $out[$id] = array('latency' => microtime(true) - $t);
                }
                return $out;
        }

        public static function report($queriesInfo) {
                return array('Count' => count($queriesInfo));
        }
}
