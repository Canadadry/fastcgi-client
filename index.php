<?php
// Display the requested URL
echo "<h1>Requested URL:</h1>";
echo "<p>" . $_SERVER["REQUEST_URI"] . "</p>";

// Display the request method
echo "<h1>Request Method:</h1>";
echo "<p>" . $_SERVER["REQUEST_METHOD"] . "</p>";

// Display the headers
echo "<h1>Headers:</h1>";
echo "<pre>";
foreach (getallheaders() as $name => $value) {
	echo "$name: $value\n";
}
echo "</pre>";

// Display the body of the request
echo "<h1>Body:</h1>";
echo "<pre>";
echo file_get_contents("php://input");
echo "</pre>";
?>
