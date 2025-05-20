# Laminate

A CLI tool for layering structured data over structured data.

Laminate allows you to take a base configuration file and apply one or more patch files to it, producing a final merged configuration. It supports various data formats like YAML, JSON, and TOML.

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
