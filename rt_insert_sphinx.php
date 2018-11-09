<?php
class plugin {

        private $sphinxql = false;

        public static function init_global() {
                $sphinxql = new mysqli('sphinx', '', '', '', 9306);
                $sphinxql->query("set global query_log_format=sphinxql");
                $sphinxql->query("truncate rtindex rt");
                $sphinxql->close();
        }

        public function init() {
                $this->sphinxql = new mysqli('sphinx', '', '', '', 9306);
        }

        public function query($queries) {
                $out = array();
                foreach ($queries as $id=>$query) {
                        $t = microtime(true);
                        $query = "insert into rt values(".($id+1).", '$query', '$query', '$query', '$query', 1, 2, 3, 4, 5, 6, '$query')";
                        $res = $this->sphinxql->query($query);
                        if ($this->sphinxql->error) echo "ERROR: {$this->sphinxql->error}\n";

                        $res = $this->sphinxql->query("select filecontent from rt where id = ".($id+1));
                        $row = $res->fetch_row();
                        if ($row[0] != $queries[$id]) print_r($row);

                        $out[$id] = array('latency' => microtime(true) - $t);
                }
                return $out;
        }

        public static function report($queriesInfo) {
                return array('Count' => count($queriesInfo));
        }
}
