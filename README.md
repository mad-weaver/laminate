# Laminate

A CLI tool for layering structured data over structured data.

Laminate allows you to take a base configuration file and apply one or more patch files to it, producing a final merged configuration. It supports various data formats like YAML, JSON, and TOML.

## Installation and Building

### Via `go install`

If you have Go installed, you can install the `laminate` tool directly using `go install`:

```bash
go install github.com/mad-weaver/laminate
```

This will download the source code, build the executable, and place it in your Go binary directory (`$GOPATH/bin` or `$HOME/go/bin`). Make sure this directory is in your system's PATH.

### Building from Source

Alternatively, you can clone the repository and build the executable manually:

1. Clone the repository:
   ```bash
   git clone https://github.com/mad-weaver/laminate.git
   ```
2. Change into the source directory:
   ```bash
   cd laminate/
   ```
3. Build the executable:
   ```bash
   go build -o laminate main.go
   ```

This will create a `laminate` executable. You can then run it using `./laminate [options]`.

## Usage

```bash
laminate [global options]
```

### Global Options

| Option                | Alias | Description                                                                                                | Default     | Environment Variable |
|-----------------------|-------|------------------------------------------------------------------------------------------------------------|-------------|----------------------|
| `--source value`      | `-s`  | Specify source data to patch over. Use '-' for stdin.                                                      |             |                      |
| `--patch value`       | `-p`  | Apply patch file over source. Can be specified multiple times. Use '-' for stdin.                            |             |                      |
| `--debug`             |       | Enable debug logging.                                                                                      | `false`     | `LAMINATE_DEBUG`     |
| `--loglevel value`    | `-l`  | Specify log level (debug, info, warn, error).                                                              | `"info"`    |                      |
| `--logformat value`   | `-f`  | Specify log format (json, text, rich).                                                                     | `"text"`    |                      |
| `--output-format value` |       | Specify output format (json, yaml, toml). If not specified, it defaults to the format of the source file. |             |                      |
| `--merge-strategy value`|       | Specify list merge strategy (preserve, overwrite).                                                         | `"overwrite"` |                      |
| `--help`              | `-h`  | Show help.                                                                                                 |             |                      |

## Examples

Let's say you have a base configuration file `base.yaml`:

```yaml
server:
  host: localhost
  port: 8080
database:
  name: myapp
```

And you want to apply two patches.

`patch1.json`:
```json
{
  "server": {
    "port": 9090
  },
  "database": {
    "user": "admin"
  }
}
```

`patch2.yaml`:
```yaml
database:
  password: secret
  name: myapp_prod
```

You can merge them using the following command:

```bash
laminate --source base.yaml --patch patch1.json --patch patch2.yaml
```

By default, the output format will match the source format (YAML in this case), and the merge strategy for lists is `overwrite`.

The output will be:

```yaml
database:
  name: myapp_prod
  password: secret
  user: admin
server:
  host: localhost
  port: 9090
```

### Specifying Output Format

You can specify a different output format using the `--output-format` flag:

```bash
laminate --source base.yaml --patch patch1.json --patch patch2.yaml --output-format json
```

Output:

```json
{
  "database": {
    "name": "myapp_prod",
    "password": "secret",
    "user": "admin"
  },
  "server": {
    "host": "localhost",
    "port": 9090
  }
}
```

### Merge Strategies

Laminate supports two merge strategies for lists/arrays when patching:

*   **`overwrite` (default):** The list in the patch file completely replaces the list in the source.
*   **`preserve`:** Elements from the patch list are appended to the source list. For lists of complex objects, all items from the patch are appended as new items, even if they appear to be an update to an existing item (e.g., based on a shared key like `name`). It does not perform a deep merge or update of existing items within the list based on a key.

**Example with `preserve` (Illustrative - requires data designed for this strategy):**

If `base_list.yaml` contains:
```yaml
server:
  plugins:
    - name: auth
      enabled: true
      config:
        timeout: 30
    - name: logger
      enabled: true
      config:
        level: "info"
    - name: metrics
      enabled: false
```

And `patch_list.yaml` contains:
```yaml
server:
  plugins:
    - name: auth # Matches existing plugin
      config:
        retries: 3 # New field for auth
    - name: cache # New plugin
      enabled: true
      config:
        size: 1024
```

Command:
```bash
laminate --source base_list.yaml --patch patch_list.yaml --merge-strategy preserve
```

Expected conceptual output (actual output might vary slightly in structure if `preserve` does deep merging on list items by a key):
```yaml
server:
  plugins:
    - name: auth
      enabled: true
      config:
        timeout: 30
    - name: logger
      enabled: true
      config:
        level: "info"
    - name: metrics
      enabled: false
    - name: auth # This 'auth' entry is from patch_list.yaml, appended as a new item
      config:
        retries: 3
    - name: cache # New plugin added
      enabled: true
      config:
        size: 1024
```

## Deleting Keys

To delete a key from the source data, set its value in a patch file to the special string `__TOMBSTONE__`.

**Example:**

If `base_delete.yaml` contains:

```yaml
settings:
  database:
    host: localhost
    port: 5432
    username: admin
    password: mysecretpassword
  logging:
    level: info
```

And `patch_delete.yaml` contains:

```yaml
settings:
  database:
    password: __TOMBSTONE__
```

Command:

```bash
laminate --source base_delete.yaml --patch patch_delete.yaml
```

Output:

```yaml
settings:
  database:
    host: localhost
    port: 5432
    username: admin
  logging:
    level: info
```

Note that the `password` key under `settings.database` has been removed.

## Using Standard Input

Both the `--source` and `--patch` arguments can accept `-` as a value. This indicates that Laminate should read the structured data from standard input (`stdin`) instead of a file or URL.

## Using URLs for Source and Patch

In addition to local file paths and standard input (`-`), both the `--source` and `--patch` arguments can accept URLs from various schemes. This allows Laminate to load configuration data directly from remote services.

The following URL schemes are supported:

*   **`s3://`**: Amazon S3 buckets. The format is `s3://<bucket-name>/<object-key>`. Credentials are typically picked up from standard AWS environment variables or configuration.
    Example: `s3://my-config-bucket/configs/app_settings.yaml`
*   **`appconfig://`**: AWS AppConfig. The format is `appconfig://<application-name>/<environment-name>/<configuration-profile-name>`. Authentication uses standard AWS methods.
    Example: `appconfig://my-application/prod/service-config`
*   **`vault://`**: HashiCorp Vault. The format is `vault://<vault-server>/<path/to/secret>`. Authentication is typically via a `VAULT_TOKEN` environment variable. Note that the loader adds `https://` to the server address.
    Example: `vault://vault.example.com/secret/data/my-app/config`
*   **`consul://`**: Consul KV store. The format is `consul://<consul-server>/<path/to/key>`. Authentication relies on standard Consul environment variables or configuration. The server can include a port (`<consul-server>:<port>`).
    Example: `consul://consul.example.com:8500/configs/service/myapp`
*   **`http://`** and **`https://`**: Standard web URLs.
    Example: `https://my-web-server.com/config.json` or `http://another-server.org/settings.yaml`

### Example: Fetching from URL and Patching with Stdin

This example demonstrates fetching data from a public HTTP endpoint (icanhazdadjoke.com) using a URL as the source and patching it with data provided directly via standard input (`-`). It also shows how Laminate can convert the output to a specified format (YAML in this case), even if the source data is in a different format (JSON).

```bash
echo "response_type: LAMINATE" | laminate --source https://icanhazdadjoke.com/slack --patch - --output-format yaml
```

Expected Output:

```yaml
attachments:
    - fallback: What did one plate say to the other plate? Dinner is on me!
      footer: <https://icanhazdadjoke.com/j/EBAsPfiNuzd|permalink> - <https://icanhazdadjoke.com|icanhazdadjoke.com>
      text: What did one plate say to the other plate? Dinner is on me!
response_type: LAMINATE
username: icanhazdadjoke
```
