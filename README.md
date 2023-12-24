# go_chat_server

Very simple chat, server side.

Client is [here](https://github.com/SCP002/go_chat_client).

## Comand line flags

| Command argument     | Description                                                                         |
| -------------------- | ----------------------------------------------------------------------------------- |
| -v, --version        | Print the program version                                                           |
| -h, --help           | Print help message                                                                  |
| -l, --logLevel       | Logging level. Can be from `0` (least verbose) to `6` (most verbose) [default: `4`] |

## Config fields

* `listen_address`  
  Address to listen to in format of `host:port` or `:port`.

## Tips

* On first run, it will ask for listen address and store it in config.
  Config file will be created automatically in the folder of executable.

## TLS mode

On startup, server will look for `chat.crt` and `chat.key` files in the folder of executable.
If they present, it will run in TLS (HTTPS) mode.

To create self-signed certificate on linux (OpenSSL 1.1.1+) you can run, for example:

```sh
openssl genrsa -out rootCA.key 4096
openssl req -x509 -new -nodes -key rootCA.key -sha256 -days 3650 -out rootCA.crt
openssl genrsa -out chat.key 2048
openssl req -new -key chat.key -addext "subjectAltName=DNS:localhost,IP:127.0.0.1,IP:192.168.0.100" -out chat.csr
openssl x509 -req -extfile <(printf "subjectAltName=DNS:localhost,IP:127.0.0.1,IP:192.168.0.100") -days 3650 -in chat.csr -CA rootCA.crt -CAkey rootCA.key -CAcreateserial -out chat.crt -sha256
```

Then, add `rootCA.crt` to the client-side certificate storage (trusted root certification authorities).

## Downloads

See [releases page](https://github.com/SCP002/go_chat_server/releases).

## Build from source code [Go / Golang]

1. Install [Golang](https://golang.org/) 1.21.4 or newer.

2. Download the source code:  

    ```sh
    git clone https://github.com/SCP002/go_chat_server.git
    ```

3. Install dependencies:

    ```sh
    cd src
    go mod tidy
    ```

    Or

    ```sh
    cd src
    go get ./...
    ```

4. Update dependencies (optional):

    ```sh
    go get -u ./...
    ```

5. To build a binary for current OS / architecture into `../build/` folder:

    ```sh
    go build -o ../build/ main.go
    ```

    Or use convenient cross-compile tool to build binaries for every OS / architecture pair:

    ```sh
    cd src
    go get github.com/mitchellh/gox
    go install github.com/mitchellh/gox
    gox -output "../build/{{.Dir}}_{{.OS}}_{{.Arch}}" ./
    ```
