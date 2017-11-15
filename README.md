dropsite is a simple, HTTP-based file server. It allows you to upload and download files over http or https and provides a basic interactive interface for use with a web browser.  

This was originally an excersize in learning Go, but I quickly found it to be pretty useful. There are lots of ways to spin up a webserver on the cli, but I wanted one that had a specific set of skills, didn't require root privileges.  That said, 1) my Go is pretty bad so beware and b) dropsite doesn't do much other than share files, please temper your expectations. 

=Just Run It=

If you don't want to mess with the Go source, you can just run the binary. It should work on most Debian-based Linux distros.   

dropsite does require a few files to run. For convenience they are included with the source, but feel free to customize:
 
* drop_form.html - go html template for serving the upload form
* cert.pem - TLS server cerficate for dropsite's HTTPS file server
* key.pem - TLS server cerficate key for dropsite's HTTPS file server

The default local directory used for storing and serving files is /var/dropsite. Please ensure this directory exists and is writable by the user running dropsite.  Alternatively, you can use the "--dir" flag and specify a different directoy name.  

The following command line flag options are supported:

* --dir - Directory to use for storing and serving files.
* --cert - PEM formatted TLS server certificate file.
* --key - PEM formatted TLS server certificate key file.
* --http_port - Emphemoral port for serving dropsite via HTTP (use sudo if non-emphemoral)
* --https_port - Emphemoral port for serving dropsite via HTTPS (use sudo if non-emphemoral)

