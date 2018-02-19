#!/usr/bin/php
<?php
// Copyright (c) 2018 Ivinco LTD

$options = getopt('hb::c::', array('plugin:', 'data:', 'limit::', 'csv', 'from::', 'port::', 'index::', 'maxmatches::'));
if ((count($options) == 1 and array_keys($options)[0] == 'h') or !isset($options['plugin']) or !isset($options['data'])) die("Usage: ".basename(__FILE__)." [-h] --plugin=path/to/plugin --data=path/to/data [--limit=N] [--from=N] [-b=N] [-c=N] [--csv]
-b batch size (1 by default)
-c concurrency (1 by default)
--limit max number of documents to process
--from starts with defined document
--csv will output only final result in csv compatible format\n");

$plugin = $options['plugin'];
if (!is_file($plugin)) die("ERROR: plugin can't be open\n");
require_once($plugin);
$plugin = new plugin();

$data = $options['data'];
if (is_dir($data)) {
	$handle = opendir($data);
	if (!$handle) die("ERROR: $dir can't be read\n");
	$els = array();
	$c = 0;
	while (false !== ($entry = readdir($handle))) if (!in_array($entry, array('.', '..'))) {
		$c++;
		if (isset($options['from']) and $c < $options['from']) continue;
		$els[] = $entry;
		if (isset($options['limit']) and count($els) == $options['limit']) break;
	}
} else {
	$els = file($data, FILE_IGNORE_NEW_LINES);
	if (isset($options['limit'])) $els = array_slice($els, 0, $options['limit']);
}
$elsCount = count($els);

file_put_contents('/tmp/stress_test_lock.tmp', "0\n");

$batchSize = isset($options['b'])?$options['b']:1;
$concurrency = isset($options['c'])?$options['c']:1;
$batch = array();
$children = array();
$latencies = array();
$readBuffer = array();
$checkTime = microtime(true);
$startTime = $checkTime;
$prevCount = 0;
$pluginOutput = array();
$semaphoreId = sem_get(1, 1);
do {
	if (sem_acquire($semaphoreId)) {
		$elsLeft = $elsCount - intval(trim(file_get_contents('/tmp/stress_test_lock.tmp')));
		sem_release($semaphoreId);
	}

	if (!isset($options['csv']) and (!count($children) or microtime(true) - $checkTime > 1)) { // dump stats in the beginning and each second
		clearstatcache();
		$throughputCurrent = floor((count($latencies) - $prevCount) / (microtime(true) - $checkTime));
		$throughputOverall = floor(count($latencies) / (microtime(true) - $startTime));
		echo date('H:i:s')."Time elapsed: ".round(microtime(true) - $startTime, 3)." sec, throughput (curr / from start): $throughputCurrent / $throughputOverall rps, ".count($children)." children running, $elsLeft elements left\n";
		$prevCount = count($latencies);
		$checkTime = microtime(true);
	}

	$sockets = array();
	if (count($children) < $concurrency and $elsLeft > 0) {
		$socketNumber = count($sockets);
		socket_create_pair(AF_UNIX, SOCK_STREAM, 0, $sockets[$socketNumber]);
		socket_set_nonblock($sockets[$socketNumber][0]);
		socket_set_nonblock($sockets[$socketNumber][1]);
		$pid = pcntl_fork();
		if (!$pid) { // child
			$plugin->init($options); // asking the plugin to initialize what it needs
			$socket = &$sockets[$socketNumber][1];
			socket_close($sockets[$socketNumber][0]);
			$stop = false;
			while (!$stop) {
			        if (sem_acquire($semaphoreId)) {
					$curPos = intval(trim(file_get_contents('/tmp/stress_test_lock.tmp')));

					$batch = array();
					while (count($batch) < $batchSize) {
						if ($curPos == $elsCount) {
							$stop = true;
							break;
						} else {
							$batch[$curPos] = $els[$curPos];
							$curPos++;
						}
					}
					file_put_contents('/tmp/stress_test_lock.tmp', $curPos);
					sem_release($semaphoreId);
				} else die("ERROR: couldn't get lock via flock\n");

				$docs = array();
				foreach ($batch as $id=>$doc) {
					if (isset($handle)) { // means $data is a dir containing files, not a single file where each line a doc
						$text = file_get_contents($data."/".$doc);
						if (preg_match('/\.gz$/', $doc)) $text = gzdecode($text); // if the filename looks encoded decode the contents
					} else $text = $doc;
					
					$docs[$id] = $text; 
				}

				if ($docs) {
					$queryInfo = $plugin->query($docs); //calling the plugin to make the work it needs to do
					socket_write($socket, serialize($queryInfo)."|");
				}
			}
			exit;
				} else if ($pid !== -1) { // parent
			socket_close($sockets[$socketNumber][1]);
			$children[$pid] = $sockets[$socketNumber][0];
		}
		}
		foreach($children as $pid => $socket) {
		$read = socket_read($socket, 1024*1024*1024);
		if ($read) {
			if (!isset($readBuffer[$pid])) $readBuffer[$pid] = '';
			$readBuffer[$pid] .= $read;
			if (substr($readBuffer[$pid], -1) == '|') { // if after the last read we can't read from the socket fully just skip to wait until another chance
				$tmp = explode('|', $readBuffer[$pid]);
				foreach ($tmp as $k=>$v) {
					if(!$v) unset($tmp[$k]);
					else {
						$queryInfo = unserialize($v);
						foreach ($queryInfo as $elementId => $info) {
							$latencies[] = $info['latency'];
							$pluginOutput[$elementId] = $info;
						}
					}
				}
				$readBuffer[$pid] = '';
			}
		}

		$res = pcntl_waitpid($pid, $status, WNOHANG);
		// If the process has already exited
		if($res == -1 || $res > 0) {
				unset($children[$pid]);
		}
	}
	usleep(1000);
} while (count($children) > 0);

$totalTime = microtime(true) - $startTime;

$pluginOutput = plugin::report($pluginOutput);

sort($latencies);

$result = array(
	'concurrency' => $concurrency,
	'batch size' => $batchSize,
	'total time' => round($totalTime, 3),
	'throughput' => floor(count($latencies) / $totalTime),
	'elements count' => $elsCount,
	'latencies count' => count($latencies),
	'avg latency, ms' => round(array_sum($latencies) / count($latencies) * 1000, 3),
	'median latency, ms' => round($latencies[floor(count($latencies) * 0.5)] * 1000, 3),
	'95p latency, ms' => round($latencies[floor(count($latencies) * 0.95)] * 1000, 3),
	'99p latency, ms' => round($latencies[floor(count($latencies) * 0.99)] * 1000, 3)
);
if (isset($options['csv'])) {
	echo implode(";", array_keys($result))."\n";
	echo implode(";", array_values($result))."\n";
} else {
	echo "\nFINISHED. Total time: {$result['total time']} sec, throughput: {$result['throughput']} rps\n";
	echo "Latency stats:\n";
	echo "\tcount: {$result['latencies count']} latencies analyzed\n";
	echo "\tavg: {$result['avg latency, ms']} ms\n";
	echo "\tmedian: {$result['median latency, ms']} ms\n";
	echo "\t95p: {$result['95p latency, ms']} ms\n";
	echo "\t99p: {$result['99p latency, ms']} ms\n";
	echo "\nPlugin's output:\n";
	foreach ($pluginOutput as $k => $v) echo "\t$k: $v\n";
}

