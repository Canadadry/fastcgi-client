# FastCGI Utility

This utility facilitates communication with PHP via FastCGI and allows inspection of FastCGI protocol frames. It includes three main commands: `server`, `client`, and `sniff`.

## Installation

To install the FastCGI utility, you need to have Go installed. Clone the repository and build the project:

```bash
git clone git@github.com:Canadadry/fastcgi-client.git
cd fastcgi-client
go build -o fcgi
```

## Usage

```bash
fcgi <action> [options]
```

## Actions

### server

Starts a web server that serves static files and forwards requests to a FastCGI server.

**Options:**

 - `-document-root`: The document root to serve files from (default: current working directory).
 - `-listen`: The web server bind address to listen to (default: localhost:8080).
 - `-server`: The FastCGI server address to forward requests to (default: 127.0.0.1:9000).
 - `-index`: The default script to call when the path cannot be served by an existing file (default: index.php).

**Example:**

```bash
fcgi server -document-root /var/www -listen localhost:8080 -server 127.0.0.1:9000 -index index.php
```

### client

Sends a request to a FastCGI server.

**Options:**

 - `-host`: The FastCGI server address (default: 127.0.0.1:9000).
 - `-method`: The HTTP request method (default: GET).
 - `-url`: The request URL (default: /).
 - `-index`: The request index (default: index.php).
 - `-document-root`: The request document root (default: current working directory).
 - `-body`: The request body.
 - `-env`: The request environment as JSON.
 - `-header`: The request header as JSON.
 - `-help`: Print command help.

```bash
fcgi client -host 127.0.0.1:9000 -method POST -url /test -index index.php -document-root /var/www -body "test body" -env env.json -header "{}"
```

### sniff

Starts a proxy to inspect FastCGI frames.

**Options:**

 - `-forward-to`: The address of the FastCGI server to forward to (default: 127.0.0.1:9000).
 - `-listen`: The proxy FastCGI listen address (default: 127.0.0.1:9001).
 - `-help`: Print command help.


**example:**

 ```bash
 fcgi sniff -forward-to 127.0.0.1:9000 -listen 127.0.0.1:9001
 ```

## Examples

### Start a Web Server

Start a web server with the document root /var/www, listening on localhost:8080, and forwarding to the FastCGI server at 127.0.0.1:9000:

```bash
fcgi server -document-root /var/www -listen localhost:8080 -server 127.0.0.1:9000 -index index.php
```

### Send a Request to FastCGI Server

Send a POST request to the FastCGI server at 127.0.0.1:9000, with the URL /test, using the document root /var/www, and including a request body and environment variables from env.json:

```bash
fcgi client -host 127.0.0.1:9000 -method POST -url /test -index index.php -document-root /var/www -body "test body" -env env.json -header "{}"
```

### Start a Proxy to Inspect FastCGI Frames

Start a proxy that listens on 127.0.0.1:9001 and forwards requests to the FastCGI server at 127.0.0.1:9000:

```bash
fcgi sniff -forward-to 127.0.0.1:9000 -listen 127.0.0.1:9001
```
