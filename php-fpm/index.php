<?php
$queries = [];
parse_str($_SERVER["QUERY_STRING"], $queries);
if (array_key_exists("status_code", $queries)) {
	http_response_code(intval($queries["status_code"]));
	header("Status-Code:" . $queries["status_code"]);
	header("X-Status-Code: " . $queries["status_code"]);
}

header("X-Request-Uri: " . $_SERVER["REQUEST_URI"]);

// Display the requested URL
echo "<h1>Requested URL:</h1>\n";
echo "<p>" . $_SERVER["REQUEST_URI"] . "</p>\n";

// Display the request method
echo "<h1>Request Method:</h1>\n";
echo "<p>" . $_SERVER["REQUEST_METHOD"] . "</p>\n";

// Display the headers
echo "<h1>Headers:</h1>\n";
echo "<pre>\n";
if (array_key_exists("REMOTE_ADDR", $_SERVER)) {
	echo "REMOTE_ADDR: " . $_SERVER["REMOTE_ADDR"] . "\n";
} else {
	echo "REMOTE_ADDR:\n";
}
$h = getallheaders();
$hKeys = array_keys($h);
asort($hKeys);
foreach ($hKeys as $key) {
	echo "$key: $h[$key]\n";
}
echo "</pre>\n";

// Display the body of the request
echo "<h1>Body:</h1>\n";
echo "<pre>\n";
echo file_get_contents("php://input");
echo "</pre>";
?>
